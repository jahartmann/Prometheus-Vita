package drift

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/backup"
	"github.com/antigravity/prometheus/internal/service/crypto"
	"github.com/antigravity/prometheus/internal/ssh"
	"github.com/google/uuid"
)

type Service struct {
	driftRepo repository.DriftRepository
	backupRepo repository.BackupRepository
	fileRepo  repository.BackupFileRepository
	nodeRepo  repository.NodeRepository
	encryptor *crypto.Encryptor
	sshPool   *ssh.Pool
	collector *backup.FileCollector
}

func NewService(
	driftRepo repository.DriftRepository,
	backupRepo repository.BackupRepository,
	fileRepo repository.BackupFileRepository,
	nodeRepo repository.NodeRepository,
	encryptor *crypto.Encryptor,
	sshPool *ssh.Pool,
) *Service {
	return &Service{
		driftRepo:  driftRepo,
		backupRepo: backupRepo,
		fileRepo:   fileRepo,
		nodeRepo:   nodeRepo,
		encryptor:  encryptor,
		sshPool:    sshPool,
		collector:  backup.NewFileCollector(),
	}
}

func (s *Service) CheckDrift(ctx context.Context, nodeID uuid.UUID) (*model.DriftCheck, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}

	check := &model.DriftCheck{
		NodeID:    nodeID,
		Status:    model.DriftStatusRunning,
		CheckedAt: time.Now().UTC(),
	}
	if err := s.driftRepo.Create(ctx, check); err != nil {
		return nil, fmt.Errorf("create drift check: %w", err)
	}

	// Get latest backup for comparison
	latestBackup, err := s.backupRepo.GetLatestByNode(ctx, nodeID)
	if err != nil {
		check.Status = model.DriftStatusFailed
		check.ErrorMessage = "no backup found for comparison"
		_ = s.driftRepo.Update(ctx, check)
		return check, nil
	}

	// Get backup files
	backupFiles, err := s.fileRepo.GetByBackupID(ctx, latestBackup.ID)
	if err != nil {
		check.Status = model.DriftStatusFailed
		check.ErrorMessage = fmt.Sprintf("failed to get backup files: %v", err)
		_ = s.driftRepo.Update(ctx, check)
		return check, nil
	}

	// Decrypt SSH credentials and connect
	privateKey, err := s.encryptor.Decrypt(node.SSHPrivateKey)
	if err != nil {
		check.Status = model.DriftStatusFailed
		check.ErrorMessage = "failed to decrypt SSH key"
		_ = s.driftRepo.Update(ctx, check)
		return check, nil
	}

	sshClient, err := s.sshPool.Get(nodeID.String(), ssh.SSHConfig{
		Host:       node.Hostname,
		Port:       node.SSHPort,
		User:       node.SSHUser,
		PrivateKey: privateKey,
	})
	if err != nil {
		check.Status = model.DriftStatusFailed
		check.ErrorMessage = fmt.Sprintf("SSH connection failed: %v", err)
		_ = s.driftRepo.Update(ctx, check)
		return check, nil
	}

	// Collect current files from node
	currentFiles, err := s.collector.CollectFiles(ctx, sshClient, backup.DefaultPaths)
	if err != nil {
		check.Status = model.DriftStatusFailed
		check.ErrorMessage = fmt.Sprintf("failed to collect files: %v", err)
		_ = s.driftRepo.Update(ctx, check)
		return check, nil
	}

	// Convert backup files to CollectedFile format for comparison
	var oldFiles []backup.CollectedFile
	for _, bf := range backupFiles {
		oldFiles = append(oldFiles, backup.CollectedFile{
			Path:    bf.FilePath,
			Hash:    bf.FileHash,
			Content: bf.Content,
			Size:    bf.FileSize,
		})
	}

	// Diff
	diffs := backup.DiffFiles(oldFiles, currentFiles)

	// Count changes
	var changed, added, removed int
	var details []model.DriftFileDetail
	for _, d := range diffs {
		switch d.Status {
		case "modified":
			changed++
			details = append(details, model.DriftFileDetail{FilePath: d.FilePath, Status: d.Status, Diff: d.Diff})
		case "added":
			added++
			details = append(details, model.DriftFileDetail{FilePath: d.FilePath, Status: d.Status})
		case "removed":
			removed++
			details = append(details, model.DriftFileDetail{FilePath: d.FilePath, Status: d.Status})
		}
	}

	detailsJSON, _ := json.Marshal(details)

	check.Status = model.DriftStatusCompleted
	check.TotalFiles = len(currentFiles)
	check.ChangedFiles = changed
	check.AddedFiles = added
	check.RemovedFiles = removed
	check.Details = detailsJSON
	check.CheckedAt = time.Now().UTC()

	if err := s.driftRepo.Update(ctx, check); err != nil {
		return nil, fmt.Errorf("update drift check: %w", err)
	}

	slog.Info("drift check completed",
		slog.String("node_id", nodeID.String()),
		slog.Int("changed", changed),
		slog.Int("added", added),
		slog.Int("removed", removed),
	)

	return check, nil
}

func (s *Service) GetLatestByNode(ctx context.Context, nodeID uuid.UUID) (*model.DriftCheck, error) {
	return s.driftRepo.GetLatestByNode(ctx, nodeID)
}

func (s *Service) ListByNode(ctx context.Context, nodeID uuid.UUID, limit int) ([]model.DriftCheck, error) {
	return s.driftRepo.ListByNode(ctx, nodeID, limit)
}

func (s *Service) ListAll(ctx context.Context, limit int) ([]model.DriftCheck, error) {
	return s.driftRepo.ListAll(ctx, limit)
}
