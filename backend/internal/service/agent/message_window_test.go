package agent

import (
	"testing"

	"github.com/antigravity/prometheus/internal/model"
)

// A "tool" result message must be preceded by its assistant tool_calls message,
// or OpenAI/Anthropic reject the request with a 400. After windowing to the last
// N messages, a leading orphan tool message must be dropped.
func TestPairAwareWindowDropsLeadingOrphanToolMessages(t *testing.T) {
	msgs := []model.ChatMessage{
		{Role: model.RoleUser, Content: "1"},
		{Role: model.RoleAssistant, Content: "2"}, // had tool_calls
		{Role: model.RoleTool, Content: "3"},      // tool result -> would be the window start
		{Role: model.RoleAssistant, Content: "4"},
	}
	got := pairAwareWindow(msgs, 2)
	if len(got) != 1 || got[0].Content != "4" {
		t.Fatalf("expected leading orphan tool message dropped, got %+v", got)
	}
}

func TestPairAwareWindowKeepsShortHistory(t *testing.T) {
	msgs := []model.ChatMessage{
		{Role: model.RoleUser, Content: "1"},
		{Role: model.RoleAssistant, Content: "2"},
	}
	if got := pairAwareWindow(msgs, 50); len(got) != 2 {
		t.Fatalf("short history should be unchanged, got %d", len(got))
	}
}

func TestPairAwareWindowKeepsAssistantWithResultsAtStart(t *testing.T) {
	msgs := []model.ChatMessage{
		{Role: model.RoleUser, Content: "1"},
		{Role: model.RoleAssistant, Content: "2"}, // tool_calls
		{Role: model.RoleTool, Content: "3"},      // its result
	}
	// last 2 = [assistant, tool]; assistant at start is valid, keep both.
	got := pairAwareWindow(msgs, 2)
	if len(got) != 2 || got[0].Content != "2" {
		t.Fatalf("assistant+results window should be kept, got %+v", got)
	}
}
