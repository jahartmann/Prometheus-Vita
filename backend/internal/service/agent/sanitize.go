package agent

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// blockedPathPrefixes are paths that should never be read or written by the AI agent.
var blockedPathPrefixes = []string{
	"/etc/shadow",
	"/etc/gshadow",
	"/etc/sudoers",
	"/root/.ssh",
	"/home/*/.ssh",
	"/proc",
	"/sys",
	"/dev",
	"/run/secrets",
}

// blockedExactPaths are exact paths that should never be accessed.
var blockedExactPaths = map[string]bool{
	"/etc/passwd": true,
}

// ValidateFilePath checks that a file path is not targeting security-sensitive locations.
func ValidateFilePath(path string) error {
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) {
		return fmt.Errorf("nur absolute Pfade sind erlaubt")
	}

	for _, blocked := range blockedPathPrefixes {
		if strings.Contains(blocked, "*") {
			// Handle glob-style patterns like /home/*/.ssh
			matched, _ := filepath.Match(blocked, clean)
			if matched {
				return fmt.Errorf("Zugriff auf %q ist nicht erlaubt", path)
			}
			// Also check prefix for deeper paths
			prefix := strings.Split(blocked, "*")[0]
			suffix := strings.Split(blocked, "*")[1]
			if strings.HasPrefix(clean, prefix) && strings.Contains(clean, suffix) {
				return fmt.Errorf("Zugriff auf %q ist nicht erlaubt", path)
			}
		} else if strings.HasPrefix(clean, blocked) {
			return fmt.Errorf("Zugriff auf %q ist nicht erlaubt", path)
		}
	}

	if blockedExactPaths[clean] {
		return fmt.Errorf("Zugriff auf %q ist nicht erlaubt", path)
	}

	return nil
}

// dangerousCommandPatterns matches shell metacharacters and dangerous command patterns.
var dangerousCommandPatterns = regexp.MustCompile(
	`(?i)(;\s*rm\s+-rf|;\s*dd\s+if=|>\s*/dev/sd|mkfs\.|` +
		`curl\s+.*\|\s*sh|wget\s+.*\|\s*sh|` +
		`chmod\s+[0-7]*777|` +
		`/etc/shadow|/etc/passwd|\.ssh/authorized_keys|` +
		`\bshutdown\b|\breboot\b|\binit\s+0\b|\bpoweroff\b)`,
)

// ValidateSSHCommand checks that a command does not contain dangerous patterns.
func ValidateSSHCommand(command string) error {
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("leerer Befehl ist nicht erlaubt")
	}

	if dangerousCommandPatterns.MatchString(command) {
		return fmt.Errorf("Befehl enthaelt potenziell gefaehrliche Ausdruecke und wurde blockiert")
	}

	return nil
}
