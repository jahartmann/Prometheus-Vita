package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	nodeService "github.com/antigravity/prometheus/internal/service/node"
)

type GetStorageTool struct {
	nodeService *nodeService.Service
}

func NewGetStorageTool(nodeSvc *nodeService.Service) *GetStorageTool {
	return &GetStorageTool{nodeService: nodeSvc}
}

func (t *GetStorageTool) Name() string {
	return "get_storage"
}

func (t *GetStorageTool) Description() string {
	return "Ruft Storage-Informationen fuer einen Node ab (Typ, Groesse, Auslastung)"
}

func (t *GetStorageTool) ReadOnly() bool { return true }

func (t *GetStorageTool) Parameters() json.RawMessage {
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

func (t *GetStorageTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
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

	storage, err := t.nodeService.GetStorage(ctx, nodeID)
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler beim Abrufen der Storage-Infos: %v", err)})
	}

	return json.Marshal(storage)
}
