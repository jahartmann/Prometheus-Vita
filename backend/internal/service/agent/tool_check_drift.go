package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/antigravity/prometheus/internal/service/drift"
)

type CheckDriftTool struct {
	driftSvc *drift.Service
}

func NewCheckDriftTool(driftSvc *drift.Service) *CheckDriftTool {
	return &CheckDriftTool{driftSvc: driftSvc}
}

func (t *CheckDriftTool) Name() string {
	return "check_drift"
}

func (t *CheckDriftTool) Description() string {
	return "Prueft Konfigurationsdrift eines Nodes durch Vergleich mit dem letzten Backup"
}

func (t *CheckDriftTool) ReadOnly() bool {
	return true
}

func (t *CheckDriftTool) Parameters() json.RawMessage {
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

func (t *CheckDriftTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
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

	check, err := t.driftSvc.CheckDrift(ctx, nodeID)
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Drift-Check fehlgeschlagen: %v", err)})
	}

	return json.Marshal(check)
}
