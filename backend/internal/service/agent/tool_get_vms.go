package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	nodeService "github.com/antigravity/prometheus/internal/service/node"
)

type GetVMsTool struct {
	nodeService *nodeService.Service
}

func NewGetVMsTool(nodeSvc *nodeService.Service) *GetVMsTool {
	return &GetVMsTool{nodeService: nodeSvc}
}

func (t *GetVMsTool) Name() string {
	return "get_vms"
}

func (t *GetVMsTool) Description() string {
	return "Ruft alle VMs und Container auf einem bestimmten Node ab"
}

func (t *GetVMsTool) ReadOnly() bool { return true }

func (t *GetVMsTool) Parameters() json.RawMessage {
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

func (t *GetVMsTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
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

	vms, err := t.nodeService.GetVMs(ctx, nodeID)
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler beim Abrufen der VMs: %v", err)})
	}

	return json.Marshal(vms)
}
