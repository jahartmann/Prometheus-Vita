package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	nodeService "github.com/antigravity/prometheus/internal/service/node"
)

type StartVMTool struct {
	nodeService *nodeService.Service
}

func NewStartVMTool(nodeSvc *nodeService.Service) *StartVMTool {
	return &StartVMTool{nodeService: nodeSvc}
}

func (t *StartVMTool) Name() string {
	return "start_vm"
}

func (t *StartVMTool) Description() string {
	return "Startet eine VM oder einen Container auf einem Proxmox-Node"
}

func (t *StartVMTool) ReadOnly() bool { return false }

func (t *StartVMTool) Parameters() json.RawMessage {
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
				"description": "VM-Typ: qemu (KVM) oder lxc (Container). Standard: qemu"
			}
		},
		"required": ["node_id", "vmid"]
	}`)
}

func (t *StartVMTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
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

	vmType := params.VMType
	if vmType == "" {
		vmType = "qemu"
	}

	upid, err := t.nodeService.StartVM(ctx, nodeID, params.VMID, vmType)
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler beim Starten der VM: %v", err)})
	}

	return json.Marshal(map[string]interface{}{
		"upid":    upid,
		"vmid":    params.VMID,
		"message": fmt.Sprintf("VM %d wurde erfolgreich gestartet.", params.VMID),
	})
}
