package agent

import (
	"context"
	"encoding/json"
	"fmt"

	brainService "github.com/antigravity/prometheus/internal/service/brain"
)

type RecallKnowledgeTool struct {
	brainService *brainService.Service
}

func NewRecallKnowledgeTool(brainSvc *brainService.Service) *RecallKnowledgeTool {
	return &RecallKnowledgeTool{brainService: brainSvc}
}

func (t *RecallKnowledgeTool) Name() string {
	return "recall_knowledge"
}

func (t *RecallKnowledgeTool) Description() string {
	return "Durchsucht die Wissensbasis nach relevanten Eintraegen"
}

func (t *RecallKnowledgeTool) ReadOnly() bool { return true }

func (t *RecallKnowledgeTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "Suchbegriff fuer die Wissensbasis"
			}
		},
		"required": ["query"]
	}`)
}

func (t *RecallKnowledgeTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("parse arguments: %w", err)
	}

	entries, err := t.brainService.Search(ctx, params.Query)
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler bei der Suche: %v", err)})
	}

	if len(entries) == 0 {
		return json.Marshal(map[string]interface{}{
			"results": []interface{}{},
			"message": "Keine relevanten Eintraege gefunden",
		})
	}

	results := make([]map[string]interface{}, 0, len(entries))
	for _, e := range entries {
		results = append(results, map[string]interface{}{
			"id":       e.ID.String(),
			"category": e.Category,
			"subject":  e.Subject,
			"content":  e.Content,
		})
	}

	return json.Marshal(map[string]interface{}{
		"results": results,
		"count":   len(results),
	})
}
