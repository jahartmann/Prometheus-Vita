package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	nodeService "github.com/antigravity/prometheus/internal/service/node"
)

type VMExecTool struct {
	nodeService *nodeService.Service
}

func NewVMExecTool(nodeSvc *nodeService.Service) *VMExecTool {
	return &VMExecTool{nodeService: nodeSvc}
}

func (t *VMExecTool) Name() string {
	return "vm_exec"
}

func (t *VMExecTool) Description() string {
	return "Fuehrt einen Befehl innerhalb einer VM oder eines Containers aus und gibt stdout, stderr und Exit-Code zurueck"
}

func (t *VMExecTool) ReadOnly() bool { return false }

func (t *VMExecTool) Parameters() json.RawMessage {
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
			"command": {
				"type": "string",
				"description": "Der auszufuehrende Befehl"
			}
		},
		"required": ["node_id", "vmid", "vm_type", "command"]
	}`)
}

func (t *VMExecTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		NodeID  string `json:"node_id"`
		VMID    int    `json:"vmid"`
		VMType  string `json:"vm_type"`
		Command string `json:"command"`
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

	result, err := t.nodeService.ExecVMCommand(ctx, nodeID, params.VMID, params.VMType, []string{"sh", "-c", params.Command})
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler beim Ausfuehren: %v", err)})
	}

	return json.Marshal(map[string]interface{}{
		"stdout":    result.OutData,
		"stderr":    result.ErrData,
		"exit_code": result.ExitCode,
	})
}
