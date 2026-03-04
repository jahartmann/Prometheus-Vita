package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/crypto"
	"github.com/antigravity/prometheus/internal/ssh"
	"github.com/google/uuid"
)

// RestoreService handles restoring backed-up configuration files to nodes and
// generating downloadable archives of backup contents.
type RestoreService struct {
	backupRepo repository.BackupRepository
	fileRepo   repository.BackupFileRepository
	nodeRepo   repository.NodeRepository
	encryptor  *crypto.Encryptor
	sshPool    *ssh.Pool
	collector  *FileCollector
}

// NewRestoreService creates a new RestoreService with the required
// dependencies.
func NewRestoreService(
	backupRepo repository.BackupRepository,
	fileRepo repository.BackupFileRepository,
	nodeRepo repository.NodeRepository,
	encryptor *crypto.Encryptor,
	sshPool *ssh.Pool,
) *RestoreService {
	return &RestoreService{
		backupRepo: backupRepo,
		fileRepo:   fileRepo,
		nodeRepo:   nodeRepo,
		encryptor:  encryptor,
		sshPool:    sshPool,
		collector:  NewFileCollector(),
	}
}

// RestoreFiles restores the specified files from a backup to the originating
// node. When req.DryRun is true, no files are written; instead a preview of
// what would change is returned. When req.DryRun is false, each file is
// uploaded to the node and its permissions/ownership are restored.
func (rs *RestoreService) RestoreFiles(ctx context.Context, backupID uuid.UUID, req model.RestoreRequest) (*model.RestorePreview, error) {
	// Get backup to find the node
	backup, err := rs.backupRepo.GetByID(ctx, backupID)
	if err != nil {
		return nil, fmt.Errorf("get backup: %w", err)
	}

	// Get node
	node, err := rs.nodeRepo.GetByID(ctx, backup.NodeID)
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}

	// Build SSH config
	sshCfg, err := rs.buildSSHConfig(node)
	if err != nil {
		return nil, fmt.Errorf("build ssh config: %w", err)
	}

	// Get SSH client
	client, err := rs.sshPool.Get(backup.NodeID.String(), sshCfg)
	if err != nil {
		return nil, fmt.Errorf("get ssh client: %w", err)
	}
	defer rs.sshPool.Return(backup.NodeID.String(), client)

	preview := &model.RestorePreview{
		Files: make([]model.RestoreFilePreview, 0, len(req.FilePaths)),
	}

	for _, filePath := range req.FilePaths {
		// Get the backed up file
		backupFile, err := rs.fileRepo.GetSingleFile(ctx, backupID, filePath)
		if err != nil {
			slog.Warn("backup file not found, skipping",
				slog.String("file_path", filePath),
				slog.Any("error", err),
			)
			preview.Files = append(preview.Files, model.RestoreFilePreview{
				FilePath:   filePath,
				Action:     "skip",
				BackupHash: "",
			})
			continue
		}

		filePreview := model.RestoreFilePreview{
			FilePath:   filePath,
			BackupHash: backupFile.FileHash,
		}

		// Read current file from node for comparison
		currentContent, err := client.CopyFrom(ctx, filePath)
		if err != nil {
			// File doesn't exist on node -- will be created
			filePreview.Action = "create"
			if req.DryRun {
				preview.Files = append(preview.Files, filePreview)
				continue
			}
		} else {
			currentHash := fmt.Sprintf("%x", sha256.Sum256(currentContent))
			filePreview.CurrentHash = currentHash

			if currentHash == backupFile.FileHash {
				filePreview.Action = "skip"
				preview.Files = append(preview.Files, filePreview)
				continue
			}

			filePreview.Action = "restore"
			if req.DryRun {
				diff := generateUnifiedDiff(string(currentContent), string(backupFile.Content))
				filePreview.Diff = diff
				preview.Files = append(preview.Files, filePreview)
				continue
			}
		}

		// Actually restore the file
		if err := client.CopyTo(ctx, backupFile.Content, filePath); err != nil {
			return nil, fmt.Errorf("restore file %s: %w", filePath, err)
		}

		// Restore permissions
		if backupFile.FilePermissions != "" {
			if _, err := client.RunCommand(ctx, fmt.Sprintf("chmod %s %q", backupFile.FilePermissions, filePath)); err != nil {
				slog.Warn("failed to restore file permissions",
					slog.String("file_path", filePath),
					slog.Any("error", err),
				)
			}
		}

		// Restore ownership
		if backupFile.FileOwner != "" {
			if _, err := client.RunCommand(ctx, fmt.Sprintf("chown %s %q", backupFile.FileOwner, filePath)); err != nil {
				slog.Warn("failed to restore file ownership",
					slog.String("file_path", filePath),
					slog.Any("error", err),
				)
			}
		}

		filePreview.Action = "restore"
		preview.Files = append(preview.Files, filePreview)
	}

	slog.Info("restore operation completed",
		slog.String("backup_id", backupID.String()),
		slog.Bool("dry_run", req.DryRun),
		slog.Int("file_count", len(preview.Files)),
	)

	return preview, nil
}

// GenerateArchive creates a gzip-compressed tar archive containing all files
// from the specified backup. Each file in the archive preserves its original
// path (with the leading "/" stripped) and permission mode.
func (rs *RestoreService) GenerateArchive(ctx context.Context, backupID uuid.UUID) (io.Reader, error) {
	// Get all backup files with content
	fileList, err := rs.fileRepo.GetByBackupID(ctx, backupID)
	if err != nil {
		return nil, fmt.Errorf("get backup files: %w", err)
	}

	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	for _, f := range fileList {
		// Fetch full file content
		fullFile, err := rs.fileRepo.GetSingleFile(ctx, backupID, f.FilePath)
		if err != nil {
			slog.Warn("failed to get file content for archive",
				slog.String("file_path", f.FilePath),
				slog.Any("error", err),
			)
			continue
		}

		// Parse permissions to file mode
		mode := int64(0644)
		if fullFile.FilePermissions != "" {
			if parsed, err := strconv.ParseInt(fullFile.FilePermissions, 8, 64); err == nil {
				mode = parsed
			}
		}

		// Strip leading "/" for tar path
		tarPath := strings.TrimPrefix(fullFile.FilePath, "/")

		header := &tar.Header{
			Name: tarPath,
			Size: int64(len(fullFile.Content)),
			Mode: mode,
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, fmt.Errorf("write tar header for %s: %w", fullFile.FilePath, err)
		}

		if _, err := tarWriter.Write(fullFile.Content); err != nil {
			return nil, fmt.Errorf("write tar content for %s: %w", fullFile.FilePath, err)
		}
	}

	if err := tarWriter.Close(); err != nil {
		return nil, fmt.Errorf("close tar writer: %w", err)
	}
	if err := gzWriter.Close(); err != nil {
		return nil, fmt.Errorf("close gzip writer: %w", err)
	}

	return &buf, nil
}

// buildSSHConfig constructs an ssh.SSHConfig from a node's stored (encrypted)
// credentials.
func (rs *RestoreService) buildSSHConfig(node *model.Node) (ssh.SSHConfig, error) {
	cfg := ssh.SSHConfig{
		Host: node.Hostname,
		Port: node.SSHPort,
		User: node.SSHUser,
	}

	if node.SSHPrivateKey != "" {
		decrypted, err := rs.encryptor.Decrypt(node.SSHPrivateKey)
		if err != nil {
			return ssh.SSHConfig{}, fmt.Errorf("decrypt ssh private key: %w", err)
		}
		cfg.PrivateKey = decrypted
	}

	return cfg, nil
}
