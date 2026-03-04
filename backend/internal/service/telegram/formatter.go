package telegram

import (
	"strings"
)

// FormatAgentResponse converts agent response text to Telegram-compatible Markdown.
func FormatAgentResponse(content string) string {
	if content == "" {
		return "Keine Antwort erhalten."
	}

	// Telegram Markdown v1 is limited. Escape problematic characters
	// but keep basic formatting like *bold* and `code`.
	content = escapeTelegramMarkdown(content)

	// Truncate very long messages (Telegram limit is 4096)
	if len(content) > 4000 {
		content = content[:3997] + "..."
	}

	return content
}

// escapeTelegramMarkdown escapes characters that could break Telegram Markdown v1.
func escapeTelegramMarkdown(s string) string {
	// Only escape characters that are problematic in Markdown v1
	// Keep * for bold, ` for code, _ for italic
	replacer := strings.NewReplacer(
		"[", "\\[",
		"]", "\\]",
	)
	return replacer.Replace(s)
}
