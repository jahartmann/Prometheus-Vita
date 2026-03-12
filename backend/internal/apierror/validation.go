package apierror

import (
	"path/filepath"
	"strings"
)

// ValidatePath checks a file path for traversal attacks and invalid characters.
// Returns a cleaned absolute path or an APIError.
func ValidatePath(path string) (string, error) {
	if path == "" {
		return "/", nil
	}

	// Reject null bytes
	if strings.ContainsRune(path, 0) {
		return "", VMPathInvalid(path)
	}

	// Clean the path (resolves . and ..)
	cleaned := filepath.Clean(path)

	// Convert to forward slashes
	cleaned = filepath.ToSlash(cleaned)

	// Must be absolute
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}

	// After cleaning, reject if it still contains ..
	if strings.Contains(cleaned, "..") {
		return "", VMPathInvalid(path)
	}

	return cleaned, nil
}
