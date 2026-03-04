package agent

import (
	"context"
	"encoding/json"

	"github.com/antigravity/prometheus/internal/service/briefing"
)

type GetBriefingTool struct {
	briefingSvc *briefing.Service
}

func NewGetBriefingTool(briefingSvc *briefing.Service) *GetBriefingTool {
	return &GetBriefingTool{briefingSvc: briefingSvc}
}

func (t *GetBriefingTool) Name() string {
	return "get_briefing"
}

func (t *GetBriefingTool) Description() string {
	return "Ruft das aktuellste Morning Briefing mit Zusammenfassung der Infrastruktur ab"
}

func (t *GetBriefingTool) ReadOnly() bool { return true }

func (t *GetBriefingTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {},
		"required": []
	}`)
}

func (t *GetBriefingTool) Execute(ctx context.Context, _ json.RawMessage) (json.RawMessage, error) {
	b, err := t.briefingSvc.GetLatest(ctx)
	if err != nil {
		return json.Marshal(map[string]string{"message": "Noch kein Morning Briefing verfuegbar"})
	}

	return json.Marshal(map[string]interface{}{
		"id":           b.ID.String(),
		"summary":      b.Summary,
		"data":         json.RawMessage(b.Data),
		"generated_at": b.GeneratedAt.Format("2006-01-02 15:04:05"),
	})
}
