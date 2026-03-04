package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	nodeService "github.com/antigravity/prometheus/internal/service/node"
)

type StopVMTool struct {
	nodeService *nodeService.Service
}

func NewStopVMTool(nodeSvc *nodeService.Service) *StopVMTool {
	return &StopVMTool{nodeService: nodeSvc}
}

func (t *StopVMTool) Name() string {
	return "stop_vm"
}

func (t *StopVMTool) Description() string {
	return "Stoppt eine VM oder einen Container auf einem Proxmox-Node"
}

func (t *StopVMTool) ReadOnly() bool { return false }

func (t *StopVMTool) Parameters() json.RawMessage {
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

func (t *StopVMTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
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

	upid, err := t.nodeService.StopVM(ctx, nodeID, params.VMID, vmType)
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler beim Stoppen der VM: %v", err)})
	}

	return json.Marshal(map[string]interface{}{
		"upid":    upid,
		"vmid":    params.VMID,
		"message": fmt.Sprintf("VM %d wurde erfolgreich gestoppt.", params.VMID),
	})
}
