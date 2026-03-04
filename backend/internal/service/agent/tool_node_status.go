package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	nodeService "github.com/antigravity/prometheus/internal/service/node"
)

type NodeStatusTool struct {
	nodeService *nodeService.Service
}

func NewNodeStatusTool(nodeSvc *nodeService.Service) *NodeStatusTool {
	return &NodeStatusTool{nodeService: nodeSvc}
}

func (t *NodeStatusTool) Name() string {
	return "get_node_status"
}

func (t *NodeStatusTool) Description() string {
	return "Ruft den aktuellen Status eines Nodes ab (CPU, Memory, Disk, Uptime)"
}

func (t *NodeStatusTool) Parameters() json.RawMessage {
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

func (t *NodeStatusTool) ReadOnly() bool { return true }

func (t *NodeStatusTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
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

	status, err := t.nodeService.GetStatus(ctx, nodeID)
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler beim Abrufen des Status: %v", err)})
	}

	return json.Marshal(status)
}
