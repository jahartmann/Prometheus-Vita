package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	nodeService "github.com/antigravity/prometheus/internal/service/node"
)

type VMServiceActionTool struct {
	nodeService *nodeService.Service
}

func NewVMServiceActionTool(nodeSvc *nodeService.Service) *VMServiceActionTool {
	return &VMServiceActionTool{nodeService: nodeSvc}
}

func (t *VMServiceActionTool) Name() string {
	return "vm_service_action"
}

func (t *VMServiceActionTool) Description() string {
	return "Startet, stoppt oder startet einen Systemd-Service in einer VM neu"
}

func (t *VMServiceActionTool) ReadOnly() bool { return false }

func (t *VMServiceActionTool) Parameters() json.RawMessage {
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
			"service": {
				"type": "string",
				"description": "Der Service-Name (z.B. nginx, mysql)"
			},
			"action": {
				"type": "string",
				"enum": ["start", "stop", "restart", "enable", "disable"],
				"description": "Die auszufuehrende Aktion"
			}
		},
		"required": ["node_id", "vmid", "vm_type", "service", "action"]
	}`)
}

func (t *VMServiceActionTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		NodeID  string `json:"node_id"`
		VMID    int    `json:"vmid"`
		VMType  string `json:"vm_type"`
		Service string `json:"service"`
		Action  string `json:"action"`
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

	validActions := map[string]bool{"start": true, "stop": true, "restart": true, "enable": true, "disable": true}
	if !validActions[params.Action] {
		return json.Marshal(map[string]string{"error": "Ungueltige Aktion. Erlaubt: start, stop, restart, enable, disable"})
	}

	result, err := t.nodeService.ExecVMCommand(ctx, nodeID, params.VMID, params.VMType,
		[]string{"systemctl", params.Action, params.Service})
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler: %v", err)})
	}

	return json.Marshal(map[string]interface{}{
		"service":   params.Service,
		"action":    params.Action,
		"exit_code": result.ExitCode,
		"stdout":    result.OutData,
		"stderr":    result.ErrData,
		"message":   fmt.Sprintf("Service %s: %s erfolgreich", params.Service, params.Action),
	})
}
