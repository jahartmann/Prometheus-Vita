package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	nodeService "github.com/antigravity/prometheus/internal/service/node"
)

type VMFileWriteTool struct {
	nodeService *nodeService.Service
}

func NewVMFileWriteTool(nodeSvc *nodeService.Service) *VMFileWriteTool {
	return &VMFileWriteTool{nodeService: nodeSvc}
}

func (t *VMFileWriteTool) Name() string {
	return "vm_file_write"
}

func (t *VMFileWriteTool) Description() string {
	return "Schreibt eine Datei in eine VM oder einen Container"
}

func (t *VMFileWriteTool) ReadOnly() bool { return false }

func (t *VMFileWriteTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"node_id": {
				"type": "string",
				"description": "Die UUID des Nodes"
			},
			"vmid": {
				"type": "integer",
				"description": "Die VM-ID (z.B. 101)"
			},
			"vm_type": {
				"type": "string",
				"enum": ["qemu", "lxc"],
				"description": "VM-Typ: qemu oder lxc"
			},
			"path": {
				"type": "string",
				"description": "Der Dateipfad (z.B. /etc/nginx/nginx.conf)"
			},
			"content": {
				"type": "string",
				"description": "Der Dateiinhalt"
			}
		},
		"required": ["node_id", "vmid", "vm_type", "path", "content"]
	}`)
}

func (t *VMFileWriteTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		NodeID  string `json:"node_id"`
		VMID    int    `json:"vmid"`
		VMType  string `json:"vm_type"`
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("parse arguments: %w", err)
	}

	nodeID, err := uuid.Parse(params.NodeID)
	if err != nil {
		return json.Marshal(map[string]string{"error": "Ungueltige Node-ID"})
	}

	if params.VMType == "" {
		params.VMType = "lxc"
	}

	if err := t.nodeService.WriteVMFile(ctx, nodeID, params.VMID, params.VMType, params.Path, params.Content); err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler beim Schreiben: %v", err)})
	}

	return json.Marshal(map[string]interface{}{
		"path":    params.Path,
		"message": fmt.Sprintf("Datei %s erfolgreich geschrieben", params.Path),
	})
}
