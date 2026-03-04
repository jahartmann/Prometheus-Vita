package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/antigravity/prometheus/internal/service/updates"
)

type CheckUpdatesTool struct {
	updateSvc *updates.Service
}

func NewCheckUpdatesTool(updateSvc *updates.Service) *CheckUpdatesTool {
	return &CheckUpdatesTool{updateSvc: updateSvc}
}

func (t *CheckUpdatesTool) Name() string {
	return "check_updates"
}

func (t *CheckUpdatesTool) Description() string {
	return "Prueft verfuegbare Paket-Updates auf einem Node"
}

func (t *CheckUpdatesTool) ReadOnly() bool {
	return true
}

func (t *CheckUpdatesTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"node_id": {
				"type": "string",
				"description": "Die UUID des Nodes"
			}
		},
		"required": ["node_id"]
	}`)
}

func (t *CheckUpdatesTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		NodeID string `json:"node_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("parse arguments: %w", err)
	}

	nodeID, err := uuid.Parse(params.NodeID)
	if err != nil {
		return json.Marshal(map[string]string{"error": "Ungueltige Node-ID"})
	}

	check, err := t.updateSvc.CheckUpdates(ctx, nodeID)
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Update-Check fehlgeschlagen: %v", err)})
	}

	return json.Marshal(check)
}
