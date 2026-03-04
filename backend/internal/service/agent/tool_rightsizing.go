package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/antigravity/prometheus/internal/service/rightsizing"
)

type RightsizingTool struct {
	rightsizingSvc *rightsizing.Service
}

func NewRightsizingTool(rightsizingSvc *rightsizing.Service) *RightsizingTool {
	return &RightsizingTool{rightsizingSvc: rightsizingSvc}
}

func (t *RightsizingTool) Name() string {
	return "rightsizing"
}

func (t *RightsizingTool) Description() string {
	return "Analysiert VM-Ressourcen und gibt Empfehlungen fuer Right-Sizing (Ueber-/Unterprovisionierung)"
}

func (t *RightsizingTool) ReadOnly() bool {
	return true
}

func (t *RightsizingTool) Parameters() json.RawMessage {
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

func (t *RightsizingTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
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

	recommendations, err := t.rightsizingSvc.AnalyzeNode(ctx, nodeID)
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Right-Sizing-Analyse fehlgeschlagen: %v", err)})
	}

	if len(recommendations) == 0 {
		return json.Marshal(map[string]string{"message": "Alle VMs sind optimal provisioniert"})
	}

	return json.Marshal(recommendations)
}
