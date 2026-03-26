package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/crypto"
	"github.com/antigravity/prometheus/internal/ssh"
	"github.com/google/uuid"
)

var (
	validPermissions = regexp.MustCompile(`^[0-7]{3,4}$`)
	validOwner       = regexp.MustCompile(`^[a-zA-Z0-9._-]+:[a-zA-Z0-9._-]+$`)
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

// preValidateRestore checks that all files can be restored before any actual
// changes are made. It verifies that backup files exist and can be decrypted,
// and that all target parent directories exist on the node.
func (rs *RestoreService) preValidateRestore(ctx context.Context, client *ssh.Client, backupID uuid.UUID, filePaths []string) error {
	for _, filePath := range filePaths {
		// Check backup file exists and can be decrypted
		backupFile, err := rs.fileRepo.GetSingleFile(ctx, backupID, filePath)
		if err != nil {
			// File not found in backup is not a validation failure — it will be
			// skipped during restore. Only fail on truly unexpected errors.
			continue
		}

		if backupFile.IsEncrypted && len(backupFile.Content) > 0 {
			if rs.encryptor == nil {
				return fmt.Errorf("Backup-Datei '%s' ist verschlüsselt, aber kein Entschlüsselungskey konfiguriert", filePath)
			}
			if _, err := rs.encryptor.Decrypt(string(backupFile.Content)); err != nil {
				return fmt.Errorf("Entschlüsselung fehlgeschlagen für '%s' — möglicherweise wurde der Encryption-Key rotiert: %w", filePath, err)
			}
		}

		// Check parent directory exists on target node
		parentDir := filepath.Dir(filePath)
		result, err := client.RunCommand(ctx, "test -d "+ssh.ShellQuote(parentDir)+" && echo ok")
		if err != nil || result == nil || !strings.Contains(result.Stdout, "ok") {
			return fmt.Errorf("Verzeichnis '%s' existiert nicht auf dem Ziel-Node", parentDir)
		}
	}

	return nil
}

// RestoreFiles restores the specified files from a backup to the originating
// node. When req.DryRun is true, no files are written; instead a preview of
// what would change is returned. When req.DryRun is false, a pre-validation
// pass runs first to ensure ALL files can be restored before touching anything
// (atomic semantics). Only after successful validation are files written.
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

	// Pre-validate ALL files before making any changes (even if not dry-run).
	// This ensures we don't end up in a partial-restore state.
	if err := rs.preValidateRestore(ctx, client, backupID, req.FilePaths); err != nil {
		return nil, fmt.Errorf("Vorab-Validierung fehlgeschlagen: %w", err)
	}

	preview := &model.RestorePreview{
		Files: make([]model.RestoreFilePreview, 0, len(req.FilePaths)),
	}

	// Track which files were already changed for recovery logging on failure
	var restoredFiles []string

	for _, filePath := range req.FilePaths {
		// Get the backed up file and decrypt if needed
		backupFile, err := rs.fileRepo.GetSingleFile(ctx, backupID, filePath)
		if err == nil && backupFile.IsEncrypted && len(backupFile.Content) > 0 {
			if rs.encryptor == nil {
				return nil, fmt.Errorf("Backup verschlüsselt, aber kein Entschlüsselungskey konfiguriert")
			}
			decrypted, decErr := rs.encryptor.Decrypt(string(backupFile.Content))
			if decErr != nil {
				return nil, fmt.Errorf("Entschlüsselung fehlgeschlagen für %s — möglicherweise wurde der Encryption-Key rotiert: %w", filePath, decErr)
			}
			backupFile.Content = []byte(decrypted)
		}
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
			// Log which files were already changed and which weren't for manual recovery
			slog.Error("Restore fehlgeschlagen — Teildaten wurden bereits geschrieben",
				slog.String("failed_file", filePath),
				slog.Any("already_restored", restoredFiles),
				slog.Any("error", err),
			)
			return nil, fmt.Errorf("restore file %s: %w (bereits wiederhergestellt: %v)", filePath, err, restoredFiles)
		}
		restoredFiles = append(restoredFiles, filePath)

		// Restore permissions (validate to prevent command injection)
		if backupFile.FilePermissions != "" {
			if validPermissions.MatchString(backupFile.FilePermissions) {
				if _, err := client.RunCommand(ctx, fmt.Sprintf("chmod %s %q", backupFile.FilePermissions, filePath)); err != nil {
					slog.Warn("failed to restore file permissions",
						slog.String("file_path", filePath),
						slog.Any("error", err),
					)
				}
			} else {
				slog.Warn("skipping invalid file permissions",
					slog.String("file_path", filePath),
					slog.String("permissions", backupFile.FilePermissions),
				)
			}
		}

		// Restore ownership (validate to prevent command injection)
		if backupFile.FileOwner != "" {
			if validOwner.MatchString(backupFile.FileOwner) {
				if _, err := client.RunCommand(ctx, fmt.Sprintf("chown %s %q", backupFile.FileOwner, filePath)); err != nil {
					slog.Warn("failed to restore file ownership",
						slog.String("file_path", filePath),
						slog.Any("error", err),
					)
				}
			} else {
				slog.Warn("skipping invalid file owner",
					slog.String("file_path", filePath),
					slog.String("owner", backupFile.FileOwner),
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
// from the specified backup and streams it directly to the provided writer.
// Each file in the archive preserves its original path (with the leading "/"
// stripped) and permission mode. Streaming avoids buffering the entire archive
// in memory, preventing OOM for large backups.
func (rs *RestoreService) GenerateArchive(ctx context.Context, backupID uuid.UUID, w io.Writer) error {
	// Get all backup files with content
	fileList, err := rs.fileRepo.GetByBackupID(ctx, backupID)
	if err != nil {
		return fmt.Errorf("get backup files: %w", err)
	}

	gzWriter := gzip.NewWriter(w)
	defer gzWriter.Close()
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

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
		// Decrypt content if encrypted
		if fullFile.IsEncrypted && len(fullFile.Content) > 0 {
			if rs.encryptor == nil {
				return fmt.Errorf("Backup-Datei %s ist verschlüsselt, aber kein Entschlüsselungskey konfiguriert", f.FilePath)
			}
			decrypted, decErr := rs.encryptor.Decrypt(string(fullFile.Content))
			if decErr != nil {
				return fmt.Errorf("Entschlüsselung fehlgeschlagen für %s — möglicherweise wurde der Encryption-Key rotiert: %w", f.FilePath, decErr)
			}
			fullFile.Content = []byte(decrypted)
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
			return fmt.Errorf("write tar header for %s: %w", fullFile.FilePath, err)
		}

		if _, err := tarWriter.Write(fullFile.Content); err != nil {
			return fmt.Errorf("write tar content for %s: %w", fullFile.FilePath, err)
		}
	}

	return nil
}

// buildSSHConfig constructs an ssh.SSHConfig from a node's stored (encrypted)
// credentials.
func (rs *RestoreService) buildSSHConfig(node *model.Node) (ssh.SSHConfig, error) {
	cfg := ssh.SSHConfig{
		Host:    node.Hostname,
		Port:    node.SSHPort,
		User:    node.SSHUser,
		HostKey: node.SSHHostKey,
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
