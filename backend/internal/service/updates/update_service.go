package updates

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/crypto"
	"github.com/antigravity/prometheus/internal/ssh"
	"github.com/google/uuid"
)

var securityKeywords = []string{
	"security", "CVE", "USN", "DSA",
}

type Service struct {
	updateRepo repository.UpdateRepository
	nodeRepo   repository.NodeRepository
	encryptor  *crypto.Encryptor
	sshPool    *ssh.Pool
}

func NewService(
	updateRepo repository.UpdateRepository,
	nodeRepo repository.NodeRepository,
	encryptor *crypto.Encryptor,
	sshPool *ssh.Pool,
) *Service {
	return &Service{
		updateRepo: updateRepo,
		nodeRepo:   nodeRepo,
		encryptor:  encryptor,
		sshPool:    sshPool,
	}
}

func (s *Service) CheckUpdates(ctx context.Context, nodeID uuid.UUID) (*model.UpdateCheck, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}

	check := &model.UpdateCheck{
		NodeID:    nodeID,
		Status:    model.UpdateCheckRunning,
		CheckedAt: time.Now().UTC(),
	}
	if err := s.updateRepo.Create(ctx, check); err != nil {
		return nil, fmt.Errorf("create update check: %w", err)
	}

	privateKey, err := s.encryptor.Decrypt(node.SSHPrivateKey)
	if err != nil {
		check.Status = model.UpdateCheckFailed
		check.ErrorMessage = "failed to decrypt SSH key"
		_ = s.updateRepo.Update(ctx, check)
		return check, nil
	}

	sshClient, err := s.sshPool.Get(nodeID.String(), ssh.SSHConfig{
		Host:       node.Hostname,
		Port:       node.SSHPort,
		User:       node.SSHUser,
		PrivateKey: privateKey,
	})
	if err != nil {
		check.Status = model.UpdateCheckFailed
		check.ErrorMessage = fmt.Sprintf("SSH connection failed: %v", err)
		_ = s.updateRepo.Update(ctx, check)
		return check, nil
	}
	defer s.sshPool.Return(nodeID.String(), sshClient)

	// Update package list first
	_, _ = sshClient.RunCommand(ctx, "apt-get update -qq 2>/dev/null")

	// Get upgradable packages
	result, err := sshClient.RunCommand(ctx, "apt list --upgradable 2>/dev/null")
	if err != nil {
		check.Status = model.UpdateCheckFailed
		check.ErrorMessage = fmt.Sprintf("failed to check updates: %v", err)
		_ = s.updateRepo.Update(ctx, check)
		return check, nil
	}

	packages := parseAptOutput(result.Stdout)

	var securityCount int
	for _, p := range packages {
		if p.IsSecurity {
			securityCount++
		}
	}

	packagesJSON, _ := json.Marshal(packages)

	check.Status = model.UpdateCheckCompleted
	check.TotalUpdates = len(packages)
	check.SecurityUpdates = securityCount
	check.Packages = packagesJSON
	check.CheckedAt = time.Now().UTC()

	if err := s.updateRepo.Update(ctx, check); err != nil {
		return nil, fmt.Errorf("update check: %w", err)
	}

	slog.Info("update check completed",
		slog.String("node_id", nodeID.String()),
		slog.Int("total", len(packages)),
		slog.Int("security", securityCount),
	)

	return check, nil
}

func parseAptOutput(output string) []model.PackageUpdate {
	var packages []model.PackageUpdate
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Listing") {
			continue
		}

		// Format: package/source version1 arch [upgradable from: version2]
		parts := strings.SplitN(line, "/", 2)
		if len(parts) < 2 {
			continue
		}
		pkgName := parts[0]

		rest := parts[1]
		sourceParts := strings.SplitN(rest, " ", 2)
		source := ""
		if len(sourceParts) > 0 {
			source = sourceParts[0]
		}

		newVersion := ""
		currentVersion := ""
		if len(sourceParts) > 1 {
			versionPart := sourceParts[1]
			vParts := strings.SplitN(versionPart, " ", 2)
			if len(vParts) > 0 {
				newVersion = vParts[0]
			}
			// Extract current version from [upgradable from: X.Y.Z]
			if idx := strings.Index(versionPart, "upgradable from: "); idx != -1 {
				cv := versionPart[idx+len("upgradable from: "):]
				cv = strings.TrimRight(cv, "]")
				currentVersion = strings.TrimSpace(cv)
			}
		}

		isSecurity := false
		for _, kw := range securityKeywords {
			if strings.Contains(strings.ToLower(source), strings.ToLower(kw)) {
				isSecurity = true
				break
			}
		}

		packages = append(packages, model.PackageUpdate{
			Name:           pkgName,
			CurrentVersion: currentVersion,
			NewVersion:     newVersion,
			IsSecurity:     isSecurity,
			Source:         source,
		})
	}

	return packages
}

func (s *Service) GetLatestByNode(ctx context.Context, nodeID uuid.UUID) (*model.UpdateCheck, error) {
	return s.updateRepo.GetLatestByNode(ctx, nodeID)
}

func (s *Service) ListByNode(ctx context.Context, nodeID uuid.UUID, limit int) ([]model.UpdateCheck, error) {
	return s.updateRepo.ListByNode(ctx, nodeID, limit)
}

func (s *Service) ListAll(ctx context.Context, limit int) ([]model.UpdateCheck, error) {
	return s.updateRepo.ListAll(ctx, limit)
}
