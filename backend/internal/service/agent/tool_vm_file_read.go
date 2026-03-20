package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	nodeService "github.com/antigravity/prometheus/internal/service/node"
)

type VMFileReadTool struct {
	nodeService *nodeService.Service
}

func NewVMFileReadTool(nodeSvc *nodeService.Service) *VMFileReadTool {
	return &VMFileReadTool{nodeService: nodeSvc}
}

func (t *VMFileReadTool) Name() string {
	return "vm_file_read"
}

func (t *VMFileReadTool) Description() string {
	return "Liest eine Datei aus einer VM oder einem Container"
}

func (t *VMFileReadTool) ReadOnly() bool { return true }

func (t *VMFileReadTool) Parameters() json.RawMessage {
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
			}
		},
		"required": ["node_id", "vmid", "vm_type", "path"]
	}`)
}

func (t *VMFileReadTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		NodeID string `json:"node_id"`
		VMID   int    `json:"vmid"`
		VMType string `json:"vm_type"`
		Path   string `json:"path"`
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

	if err := ValidateFilePath(params.Path); err != nil {
		return json.Marshal(map[string]string{"error": err.Error()})
	}

	content, err := t.nodeService.ReadVMFile(ctx, nodeID, params.VMID, params.VMType, params.Path)
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler beim Lesen: %v", err)})
	}

	return json.Marshal(map[string]interface{}{
		"path":    params.Path,
		"content": content,
	})
}
