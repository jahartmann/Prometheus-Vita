package backup

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/crypto"
	"github.com/antigravity/prometheus/internal/service/monitor"
	"github.com/antigravity/prometheus/internal/ssh"
	"github.com/google/uuid"
)

// NotificationSender is an optional interface for sending notifications.
type NotificationSender interface {
	Notify(ctx context.Context, eventType, subject, body string)
}

// Service provides backup creation, listing, and diff operations for node
// configuration files.
type Service struct {
	backupRepo repository.BackupRepository
	fileRepo   repository.BackupFileRepository
	nodeRepo   repository.NodeRepository
	encryptor  *crypto.Encryptor
	sshPool    *ssh.Pool
	collector  *FileCollector
	wsHub      *monitor.WSHub
	notifSvc   NotificationSender
}

// SetNotificationService sets an optional notification service for backup events.
func (s *Service) SetNotificationService(svc NotificationSender) {
	s.notifSvc = svc
}

// NewService creates a new backup Service with the required dependencies.
func NewService(
	backupRepo repository.BackupRepository,
	fileRepo repository.BackupFileRepository,
	nodeRepo repository.NodeRepository,
	encryptor *crypto.Encryptor,
	sshPool *ssh.Pool,
	wsHub *monitor.WSHub,
) *Service {
	return &Service{
		backupRepo: backupRepo,
		fileRepo:   fileRepo,
		nodeRepo:   nodeRepo,
		encryptor:  encryptor,
		sshPool:    sshPool,
		collector:  NewFileCollector(),
		wsHub:      wsHub,
	}
}

// CreateBackup initiates a new configuration backup for the given node. It
// connects to the node over SSH, collects the configured set of files,
// computes diffs against the previous backup, persists everything to the
// database, and broadcasts a WebSocket notification on completion.
func (s *Service) CreateBackup(ctx context.Context, nodeID uuid.UUID, req model.CreateBackupRequest) (*model.ConfigBackup, error) {
	// Get the node
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}

	// Determine backup type
	backupType := req.BackupType
	if backupType == "" {
		backupType = model.BackupTypeManual
	}
	if !backupType.IsValid() {
		return nil, fmt.Errorf("invalid backup type: %s", backupType)
	}

	// Get next version number
	version, err := s.backupRepo.GetNextVersion(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("get next version: %w", err)
	}

	// Create backup record
	backup := &model.ConfigBackup{
		NodeID:     nodeID,
		Version:    version,
		BackupType: backupType,
		Status:     model.BackupStatusPending,
		Notes:      req.Notes,
	}

	if err := s.backupRepo.Create(ctx, backup); err != nil {
		return nil, fmt.Errorf("create backup record: %w", err)
	}

	// Update status to running
	if err := s.backupRepo.UpdateStatus(ctx, backup.ID, model.BackupStatusRunning, ""); err != nil {
		return nil, fmt.Errorf("update backup status to running: %w", err)
	}
	backup.Status = model.BackupStatusRunning

	// Build SSH config from node
	sshCfg, err := s.buildSSHConfig(node)
	if err != nil {
		s.failBackup(ctx, backup.ID, fmt.Sprintf("build ssh config: %v", err))
		return nil, fmt.Errorf("build ssh config: %w", err)
	}

	// Get SSH client from pool
	client, err := s.sshPool.Get(nodeID.String(), sshCfg)
	if err != nil {
		s.failBackup(ctx, backup.ID, fmt.Sprintf("get ssh client: %v", err))
		return nil, fmt.Errorf("get ssh client: %w", err)
	}
	defer s.sshPool.Return(nodeID.String(), client)

	// Collect files
	collected, err := s.collector.CollectFiles(ctx, client, DefaultPaths)
	if err != nil {
		s.failBackup(ctx, backup.ID, fmt.Sprintf("collect files: %v", err))
		return nil, fmt.Errorf("collect files: %w", err)
	}

	if len(collected) == 0 {
		s.failBackup(ctx, backup.ID, "no files collected")
		return nil, fmt.Errorf("no files collected from node %s", nodeID)
	}

	// Get previous backup files for diff
	var previousFiles []CollectedFile
	prevBackup, err := s.getPreviousCompletedBackup(ctx, nodeID, backup.ID)
	if err == nil && prevBackup != nil {
		previousFiles, err = s.getCollectedFilesFromBackup(ctx, prevBackup.ID)
		if err != nil {
			slog.Warn("failed to load previous backup files for diff",
				slog.String("backup_id", prevBackup.ID.String()),
				slog.Any("error", err),
			)
		}
	}

	// Compute diffs
	diffs := DiffFiles(previousFiles, collected)
	diffMap := make(map[string]FileDiff, len(diffs))
	for _, d := range diffs {
		diffMap[d.FilePath] = d
	}

	// Create backup file records
	var totalSize int64
	backupFiles := make([]model.BackupFile, 0, len(collected))
	for _, cf := range collected {
		totalSize += cf.Size

		diffStr := ""
		if fd, ok := diffMap[cf.Path]; ok && fd.Diff != "" {
			diffStr = fd.Diff
		}

		content := cf.Content
		if s.encryptor != nil {
			encryptedContent, encErr := s.encryptor.Encrypt(string(content))
			if encErr != nil {
				slog.Warn("failed to encrypt backup file, storing plaintext",
					slog.String("path", cf.Path),
					slog.Any("error", encErr),
				)
			} else {
				content = []byte(encryptedContent)
			}
		}

		backupFiles = append(backupFiles, model.BackupFile{
			BackupID:         backup.ID,
			FilePath:         cf.Path,
			FileHash:         cf.Hash,
			FileSize:         cf.Size,
			FilePermissions:  cf.Permissions,
			FileOwner:        cf.Owner,
			Content:          content,
			DiffFromPrevious: diffStr,
		})
	}

	if err := s.fileRepo.CreateBatch(ctx, backupFiles); err != nil {
		s.failBackup(ctx, backup.ID, fmt.Sprintf("save backup files: %v", err))
		return nil, fmt.Errorf("create backup files: %w", err)
	}

	// Mark backup as completed
	if err := s.backupRepo.UpdateCompleted(ctx, backup.ID, len(backupFiles), totalSize); err != nil {
		s.failBackup(ctx, backup.ID, fmt.Sprintf("update backup completed: %v", err))
		return nil, fmt.Errorf("update backup completed: %w", err)
	}

	backup.Status = model.BackupStatusCompleted
	backup.FileCount = len(backupFiles)
	backup.TotalSize = totalSize

	// Generate recovery guide
	guide := s.generateRecoveryGuide(node.Name, node.Hostname, version, backupFiles)
	if err := s.backupRepo.UpdateRecoveryGuide(ctx, backup.ID, guide); err != nil {
		slog.Warn("failed to save recovery guide", slog.Any("error", err))
	}
	backup.RecoveryGuide = guide

	// Broadcast via WebSocket
	if s.wsHub != nil {
		s.wsHub.BroadcastMessage(monitor.WSMessage{
			Type: "backup_completed",
			Data: backup,
		})
	}

	// Send notification
	if s.notifSvc != nil {
		s.notifSvc.Notify(ctx, "backup_completed",
			fmt.Sprintf("Backup completed for %s", node.Name),
			fmt.Sprintf("Backup v%d for node %s completed successfully. %d files, %d bytes.",
				version, node.Name, len(backupFiles), totalSize),
		)
	}

	slog.Info("backup completed",
		slog.String("backup_id", backup.ID.String()),
		slog.String("node_id", nodeID.String()),
		slog.Int("file_count", len(backupFiles)),
		slog.Int64("total_size", totalSize),
	)

	return backup, nil
}

// ListAllBackups returns all backups across all nodes, ordered by creation
// time descending.
func (s *Service) ListAllBackups(ctx context.Context) ([]model.ConfigBackup, error) {
	return s.backupRepo.ListAll(ctx)
}

// ListBackups returns all backups for the given node, ordered by creation
// time descending.
func (s *Service) ListBackups(ctx context.Context, nodeID uuid.UUID) ([]model.ConfigBackup, error) {
	return s.backupRepo.ListByNode(ctx, nodeID)
}

// GetBackup retrieves a single backup by its ID.
func (s *Service) GetBackup(ctx context.Context, backupID uuid.UUID) (*model.ConfigBackup, error) {
	return s.backupRepo.GetByID(ctx, backupID)
}

// GetBackupFiles returns all file metadata (without content) for a backup.
func (s *Service) GetBackupFiles(ctx context.Context, backupID uuid.UUID) ([]model.BackupFile, error) {
	return s.fileRepo.GetByBackupID(ctx, backupID)
}

// GetBackupFile retrieves a single file (with content) from a backup by its
// file path. Encrypted content is transparently decrypted.
func (s *Service) GetBackupFile(ctx context.Context, backupID uuid.UUID, filePath string) (*model.BackupFile, error) {
	file, err := s.fileRepo.GetSingleFile(ctx, backupID, filePath)
	if err != nil {
		return nil, err
	}
	if s.encryptor != nil && len(file.Content) > 0 {
		decrypted, decErr := s.encryptor.Decrypt(string(file.Content))
		if decErr == nil {
			file.Content = []byte(decrypted)
		}
	}
	return file, nil
}

// DeleteBackup removes a backup and all its associated files from the
// database. Files are deleted first, then the backup record.
// Note: ideally these would share a DB transaction; a future migration should
// add ON DELETE CASCADE to config_backup_files.backup_id.
func (s *Service) DeleteBackup(ctx context.Context, backupID uuid.UUID) error {
	// Verify backup exists first
	if _, err := s.backupRepo.GetByID(ctx, backupID); err != nil {
		return err
	}
	if err := s.fileRepo.DeleteByBackupID(ctx, backupID); err != nil {
		return fmt.Errorf("delete backup files: %w", err)
	}
	if err := s.backupRepo.Delete(ctx, backupID); err != nil {
		return fmt.Errorf("delete backup record: %w", err)
	}
	return nil
}

// DiffBackup computes the file-level diff between the specified backup and
// its predecessor for the same node.
func (s *Service) DiffBackup(ctx context.Context, backupID uuid.UUID) ([]FileDiff, error) {
	backup, err := s.backupRepo.GetByID(ctx, backupID)
	if err != nil {
		return nil, fmt.Errorf("get backup: %w", err)
	}

	// Get current backup files as CollectedFile
	currentFiles, err := s.getCollectedFilesFromBackup(ctx, backupID)
	if err != nil {
		return nil, fmt.Errorf("get current backup files: %w", err)
	}

	// Get previous backup files
	var previousFiles []CollectedFile
	prevBackup, err := s.getPreviousCompletedBackup(ctx, backup.NodeID, backupID)
	if err == nil && prevBackup != nil {
		previousFiles, err = s.getCollectedFilesFromBackup(ctx, prevBackup.ID)
		if err != nil {
			return nil, fmt.Errorf("get previous backup files: %w", err)
		}
	}

	return DiffFiles(previousFiles, currentFiles), nil
}

// buildSSHConfig constructs an ssh.SSHConfig from a node's stored (encrypted)
// credentials.
func (s *Service) buildSSHConfig(node *model.Node) (ssh.SSHConfig, error) {
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

// failBackup marks a backup as failed with the given error message.
func (s *Service) failBackup(ctx context.Context, backupID uuid.UUID, errMsg string) {
	if err := s.backupRepo.UpdateStatus(ctx, backupID, model.BackupStatusFailed, errMsg); err != nil {
		slog.Error("failed to update backup status to failed",
			slog.String("backup_id", backupID.String()),
			slog.Any("error", err),
		)
	}

	if s.notifSvc != nil {
		s.notifSvc.Notify(ctx, "backup_failed",
			fmt.Sprintf("Backup failed: %s", backupID),
			fmt.Sprintf("Backup %s failed: %s", backupID, errMsg),
		)
	}
}

// getPreviousCompletedBackup finds the most recent completed backup for a
// node, excluding the specified backup ID.
func (s *Service) getPreviousCompletedBackup(ctx context.Context, nodeID, excludeID uuid.UUID) (*model.ConfigBackup, error) {
	backups, err := s.backupRepo.ListByNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	for _, b := range backups {
		if b.ID != excludeID && b.Status == model.BackupStatusCompleted {
			result := b
			return &result, nil
		}
	}

	return nil, errors.New("no previous completed backup found")
}

// getCollectedFilesFromBackup reconstructs CollectedFile values from the
// stored BackupFile records for a given backup, loading content via the
// single-file repository method.
func (s *Service) getCollectedFilesFromBackup(ctx context.Context, backupID uuid.UUID) ([]CollectedFile, error) {
	files, err := s.fileRepo.GetByBackupID(ctx, backupID)
	if err != nil {
		return nil, err
	}

	collected := make([]CollectedFile, 0, len(files))
	for _, f := range files {
		// GetByBackupID doesn't include content, so we need to get each file individually
		fullFile, err := s.fileRepo.GetSingleFile(ctx, backupID, f.FilePath)
		if err != nil {
			slog.Warn("failed to get file content for diff",
				slog.String("file_path", f.FilePath),
				slog.Any("error", err),
			)
			continue
		}
		content := fullFile.Content
		if s.encryptor != nil {
			decrypted, decErr := s.encryptor.Decrypt(string(content))
			if decErr == nil {
				content = []byte(decrypted)
			}
		}
		collected = append(collected, CollectedFile{
			Path:        fullFile.FilePath,
			Content:     content,
			Hash:        fullFile.FileHash,
			Size:        fullFile.FileSize,
			Permissions: fullFile.FilePermissions,
			Owner:       fullFile.FileOwner,
		})
	}

	return collected, nil
}

// GetRecoveryGuide retrieves the recovery guide for a backup.
func (s *Service) GetRecoveryGuide(ctx context.Context, backupID uuid.UUID) (string, error) {
	backup, err := s.backupRepo.GetByID(ctx, backupID)
	if err != nil {
		return "", fmt.Errorf("get backup: %w", err)
	}
	return backup.RecoveryGuide, nil
}

// generateRecoveryGuide creates a template-based recovery guide for a backup.
func (s *Service) generateRecoveryGuide(nodeName, hostname string, version int, files []model.BackupFile) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# Recovery Guide - %s - v%d\n\n", nodeName, version))
	b.WriteString(fmt.Sprintf("## Erstellt: %s\n\n", time.Now().Format("02.01.2006 15:04")))

	b.WriteString("## System-Information\n\n")
	b.WriteString(fmt.Sprintf("- Hostname: %s\n\n", hostname))

	b.WriteString(fmt.Sprintf("## Gesicherte Dateien (%d)\n\n", len(files)))
	for _, f := range files {
		b.WriteString(fmt.Sprintf("- `%s` (%s, %s)\n", f.FilePath, f.FilePermissions, f.FileOwner))
	}
	b.WriteString("\n")

	b.WriteString("## Wiederherstellungs-Schritte\n\n")
	b.WriteString("1. Neuen Proxmox-Node installieren\n")
	b.WriteString("2. Node in Prometheus-Vita registrieren\n")
	b.WriteString(fmt.Sprintf("3. Backup v%d auswaehlen -> Wiederherstellen\n", version))
	b.WriteString("4. Dateien pruefen\n")
	b.WriteString("5. Services neustarten\n")

	return b.String()
}
