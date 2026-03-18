package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	nodeService "github.com/antigravity/prometheus/internal/service/node"
)

type RunSSHCommandTool struct {
	nodeService *nodeService.Service
}

func NewRunSSHCommandTool(nodeSvc *nodeService.Service) *RunSSHCommandTool {
	return &RunSSHCommandTool{nodeService: nodeSvc}
}

func (t *RunSSHCommandTool) Name() string {
	return "run_ssh_command"
}

func (t *RunSSHCommandTool) Description() string {
	return "Fuehrt einen SSH-Befehl auf einem Node aus und gibt stdout, stderr und Exit-Code zurueck"
}

func (t *RunSSHCommandTool) ReadOnly() bool { return false }

func (t *RunSSHCommandTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"node_id": {
				"type": "string",
				"description": "Die UUID des Nodes"
			},
			"command": {
				"type": "string",
				"description": "Der auszufuehrende Befehl (z.B. 'uptime', 'df -h')"
			}
		},
		"required": ["node_id", "command"]
	}`)
}

func (t *RunSSHCommandTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		NodeID  string `json:"node_id"`
		Command string `json:"command"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("parse arguments: %w", err)
	}

	nodeID, err := uuid.Parse(params.NodeID)
	if err != nil {
		return json.Marshal(map[string]string{"error": "Ungueltige Node-ID"})
	}

	if err := ValidateSSHCommand(params.Command); err != nil {
		return json.Marshal(map[string]string{"error": err.Error()})
	}

	result, err := t.nodeService.RunSSHCommand(ctx, nodeID, params.Command)
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler beim Ausfuehren des Befehls: %v", err)})
	}

	return json.Marshal(map[string]interface{}{
		"stdout":    result.Stdout,
		"stderr":    result.Stderr,
		"exit_code": result.ExitCode,
	})
}
