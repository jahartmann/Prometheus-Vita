package backup

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"strings"

	"github.com/antigravity/prometheus/internal/ssh"
)

// CollectedFile represents a single file gathered from a remote node via SSH.
type CollectedFile struct {
	Path        string
	Content     []byte
	Hash        string
	Size        int64
	Permissions string
	Owner       string
}

// DefaultPaths defines the default set of configuration file paths to back up
// from a Proxmox node. Paths ending with "/" are treated as directories and
// will be traversed recursively.
var DefaultPaths = []string{
	"/etc/pve/",
	"/etc/network/interfaces",
	"/etc/hostname",
	"/etc/hosts",
	"/etc/resolv.conf",
	"/etc/fstab",
	"/etc/apt/sources.list",
	"/etc/ssh/sshd_config",
}

// FileCollector gathers configuration files from remote nodes over SSH.
type FileCollector struct{}

// NewFileCollector creates a new FileCollector instance.
func NewFileCollector() *FileCollector {
	return &FileCollector{}
}

// CollectFiles connects to a remote node via the provided SSH client and
// collects all files at the given paths. Directory paths (ending with "/")
// are expanded recursively. Files that cannot be read are skipped with a
// warning logged.
func (fc *FileCollector) CollectFiles(ctx context.Context, client *ssh.Client, paths []string) ([]CollectedFile, error) {
	var filePaths []string

	for _, path := range paths {
		if strings.HasSuffix(path, "/") {
			// Directory: list files recursively
			discovered, err := fc.listDirectory(ctx, client, path)
			if err != nil {
				slog.Warn("failed to list directory",
					slog.String("path", path),
					slog.Any("error", err),
				)
				continue
			}
			filePaths = append(filePaths, discovered...)
		} else {
			filePaths = append(filePaths, path)
		}
	}

	var collected []CollectedFile
	for _, fp := range filePaths {
		file, err := fc.collectSingleFile(ctx, client, fp)
		if err != nil {
			slog.Warn("failed to collect file, skipping",
				slog.String("path", fp),
				slog.Any("error", err),
			)
			continue
		}
		collected = append(collected, *file)
	}

	return collected, nil
}

// listDirectory uses find to recursively list all regular files under the
// given directory path on the remote node.
func (fc *FileCollector) listDirectory(ctx context.Context, client *ssh.Client, dirPath string) ([]string, error) {
	result, err := client.RunCommand(ctx, fmt.Sprintf("find %s -type f 2>/dev/null", dirPath))
	if err != nil {
		return nil, fmt.Errorf("find files in %s: %w", dirPath, err)
	}

	var paths []string
	for _, line := range strings.Split(strings.TrimSpace(result.Stdout), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			paths = append(paths, line)
		}
	}

	return paths, nil
}

// collectSingleFile reads the content, permissions, ownership, and size of a
// single file on the remote node, then computes a SHA-256 hash of its content.
func (fc *FileCollector) collectSingleFile(ctx context.Context, client *ssh.Client, filePath string) (*CollectedFile, error) {
	// Read file content
	content, err := client.CopyFrom(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", filePath, err)
	}

	// Get file metadata: permissions, owner:group, size
	statResult, err := client.RunCommand(ctx, fmt.Sprintf("stat -c '%%a %%U:%%G %%s' %q", filePath))
	if err != nil {
		return nil, fmt.Errorf("stat file %s: %w", filePath, err)
	}

	permissions := ""
	owner := ""
	var size int64

	statOutput := strings.TrimSpace(statResult.Stdout)
	parts := strings.SplitN(statOutput, " ", 3)
	if len(parts) == 3 {
		permissions = parts[0]
		owner = parts[1]
		fmt.Sscanf(parts[2], "%d", &size)
	}

	hash := fmt.Sprintf("%x", sha256.Sum256(content))

	return &CollectedFile{
		Path:        filePath,
		Content:     content,
		Hash:        hash,
		Size:        size,
		Permissions: permissions,
		Owner:       owner,
	}, nil
}
