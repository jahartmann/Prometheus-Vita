package logstream

import (
	"bytes"
	"regexp"
	"strings"
	"unicode/utf8"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// Sanitize removes ANSI escape codes, null bytes, and validates UTF-8.
func Sanitize(raw string) string {
	// Remove ANSI escape sequences
	cleaned := ansiRegex.ReplaceAllString(raw, "")
	// Remove null bytes
	cleaned = strings.ReplaceAll(cleaned, "\x00", "")
	// Validate UTF-8
	if !utf8.ValidString(cleaned) {
		var buf bytes.Buffer
		for len(cleaned) > 0 {
			r, size := utf8.DecodeRuneInString(cleaned)
			if r == utf8.RuneError && size == 1 {
				buf.WriteRune('\uFFFD')
				cleaned = cleaned[1:]
			} else {
				buf.WriteRune(r)
				cleaned = cleaned[size:]
			}
		}
		cleaned = buf.String()
	}
	return cleaned
}
