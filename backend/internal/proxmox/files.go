package proxmox

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// validateFilePath checks that the path is safe for use in VM file operations.
// Prevents path traversal and command injection via crafted paths.
// NOTE: This validates for the QEMU guest agent API (form-parameter context).
// For LXC shell contexts, use shellQuote() additionally.
func validateFilePath(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("Dateipfad darf nicht leer sein")
	}
	// Normalize the path to resolve /./foo, //foo, etc.
	filePath = path.Clean(filePath)
	if !strings.HasPrefix(filePath, "/") {
		return fmt.Errorf("Dateipfad muss absolut sein (mit / beginnen): %q", filePath)
	}
	// Block path traversal
	if strings.Contains(filePath, "..") {
		return fmt.Errorf("Dateipfad darf keine '..' Komponenten enthalten: %q", filePath)
	}
	// Block null bytes (can confuse C-based tools)
	if strings.ContainsRune(filePath, 0) {
		return fmt.Errorf("Dateipfad enthält ungültige Null-Bytes")
	}
	// Block newlines (can break command parsing)
	if strings.ContainsAny(filePath, "\n\r") {
		return fmt.Errorf("Dateipfad enthält ungültige Zeilenumbrüche")
	}
	// Block shell metacharacters that could be exploited in LXC exec context.
	// These are safe in QEMU agent API (form params) but dangerous in sh -c commands.
	if strings.ContainsAny(filePath, "$`|;&<>(){}!\\") {
		return fmt.Errorf("Dateipfad enthält ungültige Sonderzeichen: %q", filePath)
	}
	return nil
}

// shellQuote wraps a string in single quotes for safe use in POSIX shell commands.
// Single quotes prevent ALL shell interpretation including $(), ``, etc.
// Any embedded single quotes are escaped via the '\'' idiom.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

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
	if err := validateVMType(vmType); err != nil {
		return "", err
	}
	if err := validateFilePath(path); err != nil {
		return "", err
	}
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
	inner := unwrapAgentResult(data)
	var resp struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.Unmarshal(inner, &resp); err != nil {
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
	if err := validateVMType(vmType); err != nil {
		return err
	}
	if err := validateFilePath(path); err != nil {
		return err
	}
	if vmType == "lxc" {
		return c.writeFileLXC(ctx, node, vmid, path, content)
	}
	return c.writeFileQEMU(ctx, node, vmid, path, content)
}

func (c *Client) writeFileLXC(ctx context.Context, node string, vmid int, path string, content string) error {
	// For large content, base64 via echo exceeds ARG_MAX (~128KB on Linux).
	// Use chunked write via multiple commands.
	const maxEchoSize = 65536 // 64KB - safe limit considering base64 4/3 expansion = ~87KB arg

	if len(content) > maxEchoSize {
		return c.writeFileLXCChunked(ctx, node, vmid, path, content)
	}

	// Use base64 encoding to avoid shell escaping issues.
	// mkdir -p ensures the parent directory exists.
	// SECURITY: Use shellQuote (single quotes) for path to prevent shell injection.
	// Go's %q produces double-quoted strings where $(cmd) is still interpreted!
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	safePath := shellQuote(path)
	cmd := fmt.Sprintf("mkdir -p $(dirname %s) && echo %s | base64 -d > %s", safePath, encoded, safePath)
	result, err := c.ExecCommand(ctx, node, vmid, "lxc", []string{"sh", "-c", cmd})
	if err != nil {
		return fmt.Errorf("write file via lxc exec: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("write failed (exit %d): %s", result.ExitCode, result.ErrData)
	}
	return nil
}

// writeFileLXCChunked writes large files in chunks to avoid exceeding ARG_MAX.
func (c *Client) writeFileLXCChunked(ctx context.Context, node string, vmid int, path string, content string) error {
	safePath := shellQuote(path)
	// Create parent directory
	_, err := c.ExecCommand(ctx, node, vmid, "lxc", []string{"sh", "-c", "mkdir -p $(dirname " + safePath + ")"})
	if err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}

	// Write in chunks via base64
	chunkSize := 49152 // 48KB per chunk (base64 -> 64KB, well under ARG_MAX)
	first := true
	for i := 0; i < len(content); i += chunkSize {
		end := i + chunkSize
		if end > len(content) {
			end = len(content)
		}
		chunk := content[i:end]
		encoded := base64.StdEncoding.EncodeToString([]byte(chunk))

		op := ">>"
		if first {
			op = ">"
			first = false
		}
		cmd := fmt.Sprintf("echo %s | base64 -d %s %s", encoded, op, safePath)
		result, err := c.ExecCommand(ctx, node, vmid, "lxc", []string{"sh", "-c", cmd})
		if err != nil {
			return fmt.Errorf("write chunk to file: %w", err)
		}
		if result.ExitCode != 0 {
			return fmt.Errorf("write chunk failed (exit %d): %s", result.ExitCode, result.ErrData)
		}
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
	if err := validateVMType(vmType); err != nil {
		return nil, err
	}
	if err := validateFilePath(path); err != nil {
		return nil, err
	}
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
	if err := validateVMType(vmType); err != nil {
		return err
	}
	if err := validateFilePath(path); err != nil {
		return err
	}
	// Block dangerous root-level deletions (covers /, //, ///, etc.)
	if strings.TrimRight(path, "/") == "" || path == "/*" || path == "/." {
		return fmt.Errorf("Löschung von '%s' ist nicht erlaubt", path)
	}
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
	if err := validateVMType(vmType); err != nil {
		return err
	}
	if err := validateFilePath(path); err != nil {
		return err
	}
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
