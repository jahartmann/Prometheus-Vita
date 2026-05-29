package agent

import "github.com/antigravity/prometheus/internal/model"

// pairAwareWindow returns the last `max` messages of a conversation, but drops
// any leading "tool" messages left orphaned by the cut. A tool-result message
// must immediately follow the assistant message that issued its tool_calls;
// replaying an orphaned tool result to OpenAI/Anthropic returns a hard 400 that
// would break the conversation permanently once it grows past the window.
func pairAwareWindow(msgs []model.ChatMessage, max int) []model.ChatMessage {
	if max > 0 && len(msgs) > max {
		msgs = msgs[len(msgs)-max:]
	}
	i := 0
	for i < len(msgs) && msgs[i].Role == model.RoleTool {
		i++
	}
	return msgs[i:]
}
