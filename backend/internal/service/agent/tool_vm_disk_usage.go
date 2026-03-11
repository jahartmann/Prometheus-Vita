package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	nodeService "github.com/antigravity/prometheus/internal/service/node"
)

type VMDiskUsageTool struct {
	nodeService *nodeService.Service
}

func NewVMDiskUsageTool(nodeSvc *nodeService.Service) *VMDiskUsageTool {
	return &VMDiskUsageTool{nodeService: nodeSvc}
}

func (t *VMDiskUsageTool) Name() string {
	return "vm_disk_usage"
}

func (t *VMDiskUsageTool) Description() string {
	return "Zeigt die Festplattenbelegung einer VM oder eines Containers (df -h)"
}

func (t *VMDiskUsageTool) ReadOnly() bool { return true }

func (t *VMDiskUsageTool) Parameters() json.RawMessage {
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

func (t *VMDiskUsageTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
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

	result, err := t.nodeService.ExecVMCommand(ctx, nodeID, params.VMID, params.VMType,
		[]string{"df", "-h"})
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler: %v", err)})
	}

	return json.Marshal(map[string]interface{}{
		"output":    result.OutData,
		"exit_code": result.ExitCode,
	})
}
