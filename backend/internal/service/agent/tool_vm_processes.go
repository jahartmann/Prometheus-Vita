package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	nodeService "github.com/antigravity/prometheus/internal/service/node"
)

type VMProcessesTool struct {
	nodeService *nodeService.Service
}

func NewVMProcessesTool(nodeSvc *nodeService.Service) *VMProcessesTool {
	return &VMProcessesTool{nodeService: nodeSvc}
}

func (t *VMProcessesTool) Name() string {
	return "vm_processes"
}

func (t *VMProcessesTool) Description() string {
	return "Ruft die Prozessliste einer VM oder eines Containers ab (ps aux)"
}

func (t *VMProcessesTool) ReadOnly() bool { return true }

func (t *VMProcessesTool) Parameters() json.RawMessage {
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
			}
		},
		"required": ["node_id", "vmid", "vm_type"]
	}`)
}

func (t *VMProcessesTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		NodeID string `json:"node_id"`
		VMID   int    `json:"vmid"`
		VMType string `json:"vm_type"`
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

	result, err := t.nodeService.ExecVMCommand(ctx, nodeID, params.VMID, params.VMType, []string{"ps", "aux", "--sort=-pcpu"})
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler beim Abrufen der Prozesse: %v", err)})
	}

	return json.Marshal(map[string]interface{}{
		"output":    result.OutData,
		"exit_code": result.ExitCode,
	})
}
