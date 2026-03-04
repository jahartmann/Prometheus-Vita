package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	brainService "github.com/antigravity/prometheus/internal/service/brain"
)

type SaveKnowledgeTool struct {
	brainService *brainService.Service
}

func NewSaveKnowledgeTool(brainSvc *brainService.Service) *SaveKnowledgeTool {
	return &SaveKnowledgeTool{brainService: brainSvc}
}

func (t *SaveKnowledgeTool) Name() string {
	return "save_knowledge"
}

func (t *SaveKnowledgeTool) Description() string {
	return "Speichert Wissen in der Wissensbasis fuer spaetere Nutzung"
}

func (t *SaveKnowledgeTool) ReadOnly() bool { return false }

func (t *SaveKnowledgeTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"category": {
				"type": "string",
				"description": "Kategorie des Wissens (z.B. node_config, troubleshooting, best_practice)"
			},
			"subject": {
				"type": "string",
				"description": "Kurze Beschreibung des Themas"
			},
			"content": {
				"type": "string",
				"description": "Der Wissensinhalt"
			}
		},
		"required": ["category", "subject", "content"]
	}`)
}

func (t *SaveKnowledgeTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		Category string `json:"category"`
		Subject  string `json:"subject"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("parse arguments: %w", err)
	}

	req := model.CreateBrainEntryRequest{
		Category: params.Category,
		Subject:  params.Subject,
		Content:  params.Content,
	}

	entry, err := t.brainService.Create(ctx, req, nil)
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler beim Speichern: %v", err)})
	}

	return json.Marshal(map[string]interface{}{
		"id":       entry.ID.String(),
		"category": entry.Category,
		"subject":  entry.Subject,
		"message":  "Wissen erfolgreich gespeichert",
	})
}
