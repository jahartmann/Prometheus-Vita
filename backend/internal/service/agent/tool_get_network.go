package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	nodeService "github.com/antigravity/prometheus/internal/service/node"
)

type GetNetworkTool struct {
	nodeService *nodeService.Service
}

func NewGetNetworkTool(nodeSvc *nodeService.Service) *GetNetworkTool {
	return &GetNetworkTool{nodeService: nodeSvc}
}

func (t *GetNetworkTool) Name() string {
	return "get_network_config"
}

func (t *GetNetworkTool) Description() string {
	return "Ruft die Netzwerk-Interfaces und deren Konfiguration fuer einen Node ab"
}

func (t *GetNetworkTool) ReadOnly() bool { return true }

func (t *GetNetworkTool) Parameters() json.RawMessage {
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

func (t *GetNetworkTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
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

	interfaces, err := t.nodeService.GetNetworkInterfaces(ctx, nodeID)
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler beim Abrufen der Netzwerk-Konfiguration: %v", err)})
	}

	return json.Marshal(interfaces)
}
