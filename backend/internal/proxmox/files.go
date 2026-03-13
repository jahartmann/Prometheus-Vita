package proxmox

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// FileEntry represents a parsed file/directory entry from ls -la output.
type FileEntry struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "file", "directory", "symlink", "other"
	Permissions string `json:"permissions"`
	Owner       string `json:"owner"`
	Group       string `json:"group"`
	Size        int64  `json:"size"`
	Modified    string `json:"modified"`
	LinkTarget  string `json:"link_target,omitempty"`
}

// ReadFile reads a file from inside a VM/container.
// For LXC: uses pct pull equivalent via exec + cat.
// For QEMU: uses qm guest file-read via the guest agent API.
func (c *Client) ReadFile(ctx context.Context, node string, vmid int, vmType string, path string) (string, error) {
	if vmType == "lxc" {
		return c.readFileLXC(ctx, node, vmid, path)
	}
	return c.readFileQEMU(ctx, node, vmid, path)
}

func (c *Client) readFileLXC(ctx context.Context, node string, vmid int, path string) (string, error) {
	result, err := c.ExecCommand(ctx, node, vmid, "lxc", []string{"cat", path})
	if err != nil {
		return "", fmt.Errorf("read file via lxc exec: %w", err)
	}
	if result.ExitCode != 0 {
		return "", fmt.Errorf("cat failed (exit %d): %s", result.ExitCode, result.ErrData)
	}
	return result.OutData, nil
}

func (c *Client) readFileQEMU(ctx context.Context, node string, vmid int, path string) (string, error) {
	apiPath := fmt.Sprintf("/nodes/%s/qemu/%d/agent/file-read", node, vmid)
	params := url.Values{}
	params.Set("file", path)
	// file-read uses GET with query parameters
	fullPath := apiPath + "?" + params.Encode()
	data, err := c.doRequest(ctx, http.MethodGet, fullPath)
	if err != nil {
		return "", fmt.Errorf("read file via qemu agent: %w", err)
	}
	var resp struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("parse file-read response: %w", err)
	}
	if resp.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(resp.Content)
		if err != nil {
			return "", fmt.Errorf("decode base64 content: %w", err)
		}
		return string(decoded), nil
	}
	return resp.Content, nil
}

// WriteFile writes content to a file inside a VM/container.
// For LXC: uses exec with tee.
// For QEMU: uses qm guest file-write via the guest agent API.
func (c *Client) WriteFile(ctx context.Context, node string, vmid int, vmType string, path string, content string) error {
	if vmType == "lxc" {
		return c.writeFileLXC(ctx, node, vmid, path, content)
	}
	return c.writeFileQEMU(ctx, node, vmid, path, content)
}

func (c *Client) writeFileLXC(ctx context.Context, node string, vmid int, path string, content string) error {
	// Use base64 encoding to avoid shell escaping issues.
	// mkdir -p ensures the parent directory exists.
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	cmd := fmt.Sprintf("mkdir -p $(dirname %s) && echo %s | base64 -d > %s", path, encoded, path)
	result, err := c.ExecCommand(ctx, node, vmid, "lxc", []string{"sh", "-c", cmd})
	if err != nil {
		return fmt.Errorf("write file via lxc exec: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("write failed (exit %d): %s", result.ExitCode, result.ErrData)
	}
	return nil
}

func (c *Client) writeFileQEMU(ctx context.Context, node string, vmid int, path string, content string) error {
	apiPath := fmt.Sprintf("/nodes/%s/qemu/%d/agent/file-write", node, vmid)
	params := url.Values{}
	params.Set("file", path)
	params.Set("content", content)
	_, err := c.doRequestWithBody(ctx, http.MethodPost, apiPath, params)
	if err != nil {
		return fmt.Errorf("write file via qemu agent: %w", err)
	}
	return nil
}

// ListDirectory lists the contents of a directory inside a VM/container.
// Executes ls -la --time-style=long-iso and parses the output.
func (c *Client) ListDirectory(ctx context.Context, node string, vmid int, vmType string, path string) ([]FileEntry, error) {
	result, err := c.ExecCommand(ctx, node, vmid, vmType, []string{"ls", "-la", "--time-style=long-iso", path})
	if err != nil {
		return nil, fmt.Errorf("list directory: %w", err)
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("ls failed (exit %d): %s", result.ExitCode, result.ErrData)
	}

	return ParseDirectoryListing(result.OutData), nil
}

// DeleteFile removes a file or directory inside a VM/container.
func (c *Client) DeleteFile(ctx context.Context, node string, vmid int, vmType string, path string) error {
	result, err := c.ExecCommand(ctx, node, vmid, vmType, []string{"rm", "-rf", path})
	if err != nil {
		return fmt.Errorf("delete file: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("delete failed (exit %d): %s", result.ExitCode, result.ErrData)
	}
	return nil
}

// MakeDirectory creates a directory inside a VM/container.
func (c *Client) MakeDirectory(ctx context.Context, node string, vmid int, vmType string, path string) error {
	result, err := c.ExecCommand(ctx, node, vmid, vmType, []string{"mkdir", "-p", path})
	if err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("mkdir failed (exit %d): %s", result.ExitCode, result.ErrData)
	}
	return nil
}

// parseDirectoryListing parses output from `ls -la --time-style=long-iso`.
func ParseDirectoryListing(output string) []FileEntry {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var entries []FileEntry

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "total ") {
			continue
		}

		entry := parseLsLine(line)
		if entry != nil && entry.Name != "." && entry.Name != ".." {
			entries = append(entries, *entry)
		}
	}

	return entries
}

// parseLsLine parses a single line from ls -la --time-style=long-iso output.
// Format: drwxr-xr-x 2 root root 4096 2024-01-15 10:30 dirname
func parseLsLine(line string) *FileEntry {
	fields := strings.Fields(line)
	if len(fields) < 8 {
		return nil
	}

	perms := fields[0]
	owner := fields[2]
	group := fields[3]

	var size int64
	fmt.Sscanf(fields[4], "%d", &size)

	modified := fields[5] + " " + fields[6]

	// Name is everything after the date/time fields
	nameStart := 0
	fieldCount := 0
	for i, ch := range line {
		if ch == ' ' || ch == '\t' {
			if i > 0 && line[i-1] != ' ' && line[i-1] != '\t' {
				fieldCount++
			}
			if fieldCount >= 7 {
				nameStart = i + 1
				break
			}
		}
	}

	if nameStart == 0 || nameStart >= len(line) {
		return nil
	}

	name := strings.TrimSpace(line[nameStart:])

	fileType := "file"
	linkTarget := ""

	switch {
	case perms[0] == 'd':
		fileType = "directory"
	case perms[0] == 'l':
		fileType = "symlink"
		parts := strings.SplitN(name, " -> ", 2)
		if len(parts) == 2 {
			name = parts[0]
			linkTarget = parts[1]
		}
	case perms[0] == '-':
		fileType = "file"
	default:
		fileType = "other"
	}

	return &FileEntry{
		Name:        name,
		Type:        fileType,
		Permissions: perms,
		Owner:       owner,
		Group:       group,
		Size:        size,
		Modified:    modified,
		LinkTarget:  linkTarget,
	}
}
