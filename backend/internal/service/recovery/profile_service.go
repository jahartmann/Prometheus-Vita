package recovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/crypto"
	"github.com/antigravity/prometheus/internal/ssh"
	"github.com/google/uuid"
)

type ProfileService struct {
	profileRepo repository.NodeProfileRepository
	nodeRepo    repository.NodeRepository
	encryptor   *crypto.Encryptor
	sshPool     *ssh.Pool
}

func NewProfileService(
	profileRepo repository.NodeProfileRepository,
	nodeRepo repository.NodeRepository,
	encryptor *crypto.Encryptor,
	sshPool *ssh.Pool,
) *ProfileService {
	return &ProfileService{
		profileRepo: profileRepo,
		nodeRepo:    nodeRepo,
		encryptor:   encryptor,
		sshPool:     sshPool,
	}
}

func (s *ProfileService) CollectProfile(ctx context.Context, nodeID uuid.UUID) (*model.NodeProfile, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}

	sshCfg, err := s.buildSSHConfig(node)
	if err != nil {
		return nil, fmt.Errorf("build ssh config: %w", err)
	}

	client, err := s.sshPool.Get(nodeID.String(), sshCfg)
	if err != nil {
		return nil, fmt.Errorf("get ssh client: %w", err)
	}
	defer s.sshPool.Return(nodeID.String(), client)

	profile := &model.NodeProfile{
		NodeID: nodeID,
	}

	// Collect CPU info
	if result, err := client.RunCommand(ctx, "lscpu"); err == nil && result.ExitCode == 0 {
		s.parseCPUInfo(result.Stdout, profile)
	} else {
		slog.Warn("failed to collect cpu info", slog.String("node_id", nodeID.String()), slog.Any("error", err))
	}

	// Collect memory info
	if result, err := client.RunCommand(ctx, "free -b"); err == nil && result.ExitCode == 0 {
		s.parseMemoryInfo(result.Stdout, profile)
	}

	// Collect disk info
	if result, err := client.RunCommand(ctx, "lsblk -J -b -o NAME,SIZE,TYPE,MOUNTPOINT,FSTYPE,MODEL"); err == nil && result.ExitCode == 0 {
		profile.Disks = json.RawMessage(result.Stdout)
	}

	// Collect network interfaces
	if result, err := client.RunCommand(ctx, "ip -j link show"); err == nil && result.ExitCode == 0 {
		profile.NetworkInterfaces = json.RawMessage(result.Stdout)
	}

	// Collect PVE version
	if result, err := client.RunCommand(ctx, "pveversion 2>/dev/null || echo unknown"); err == nil && result.ExitCode == 0 {
		profile.PVEVersion = strings.TrimSpace(result.Stdout)
	}

	// Collect kernel version
	if result, err := client.RunCommand(ctx, "uname -r"); err == nil && result.ExitCode == 0 {
		profile.KernelVersion = strings.TrimSpace(result.Stdout)
	}

	// Collect installed packages (top 500 by size)
	if result, err := client.RunCommand(ctx, "dpkg-query -W -f='${Package} ${Version}\\n' 2>/dev/null | head -500"); err == nil && result.ExitCode == 0 {
		packages := s.parsePackages(result.Stdout)
		if data, err := json.Marshal(packages); err == nil {
			profile.InstalledPackages = data
		}
	}

	// Collect storage layout
	if result, err := client.RunCommand(ctx, "pvesm status 2>/dev/null"); err == nil && result.ExitCode == 0 {
		storageEntries := s.parseStorageLayout(result.Stdout)
		if data, err := json.Marshal(storageEntries); err == nil {
			profile.StorageLayout = data
		}
	}

	if err := s.profileRepo.Create(ctx, profile); err != nil {
		return nil, fmt.Errorf("create profile: %w", err)
	}

	slog.Info("node profile collected",
		slog.String("node_id", nodeID.String()),
		slog.String("profile_id", profile.ID.String()),
	)

	return profile, nil
}

func (s *ProfileService) GetLatestProfile(ctx context.Context, nodeID uuid.UUID) (*model.NodeProfile, error) {
	return s.profileRepo.GetLatest(ctx, nodeID)
}

func (s *ProfileService) ListProfiles(ctx context.Context, nodeID uuid.UUID) ([]model.NodeProfile, error) {
	return s.profileRepo.ListByNode(ctx, nodeID)
}

func (s *ProfileService) buildSSHConfig(node *model.Node) (ssh.SSHConfig, error) {
	cfg := ssh.SSHConfig{
		Host: node.Hostname,
		Port: node.SSHPort,
		User: node.SSHUser,
	}
	if node.SSHPrivateKey != "" {
		decrypted, err := s.encryptor.Decrypt(node.SSHPrivateKey)
		if err != nil {
			return ssh.SSHConfig{}, fmt.Errorf("decrypt ssh private key: %w", err)
		}
		cfg.PrivateKey = decrypted
	}
	return cfg, nil
}

func (s *ProfileService) parseCPUInfo(output string, profile *model.NodeProfile) {
	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "Model name":
			profile.CPUModel = value
		case "CPU(s)":
			if v, err := strconv.Atoi(value); err == nil {
				profile.CPUThreads = v
			}
		case "Core(s) per socket":
			if v, err := strconv.Atoi(value); err == nil {
				profile.CPUCores = v
			}
		}
	}
}

func (s *ProfileService) parseMemoryInfo(output string, profile *model.NodeProfile) {
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "Mem:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if v, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
					profile.MemoryTotalBytes = v
				}
			}
			break
		}
	}
}

func (s *ProfileService) parsePackages(output string) []map[string]string {
	var packages []map[string]string
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			packages = append(packages, map[string]string{
				"name":    parts[0],
				"version": parts[1],
			})
		}
	}
	return packages
}

func (s *ProfileService) parseStorageLayout(output string) []map[string]string {
	var entries []map[string]string
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return entries
	}
	// Skip header line
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			entries = append(entries, map[string]string{
				"name":   fields[0],
				"type":   fields[1],
				"status": fields[2],
				"total":  fields[3],
			})
		}
	}
	return entries
}
