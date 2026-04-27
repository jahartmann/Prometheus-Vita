package node

import (
	"context"
	"fmt"
	"strings"

	"github.com/antigravity/prometheus/internal/ssh"
	"github.com/google/uuid"
)

type SSHCommandRunner interface {
	RunSSHCommand(ctx context.Context, nodeID uuid.UUID, command string) (*ssh.CommandResult, error)
}

type ToolDefinition struct {
	Name    string
	Command string
}

type ToolCheck struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Path      string `json:"path,omitempty"`
}

type ToolPreflightResult struct {
	NodeID uuid.UUID   `json:"node_id"`
	Tools  []ToolCheck `json:"tools"`
}

var defaultToolDefinitions = []ToolDefinition{
	{Name: "nmap", Command: "command -v nmap"},
	{Name: "ss", Command: "command -v ss"},
	{Name: "journalctl", Command: "command -v journalctl"},
	{Name: "pct", Command: "command -v pct"},
	{Name: "qm", Command: "command -v qm"},
}

func RunToolPreflight(ctx context.Context, runner SSHCommandRunner, nodeID uuid.UUID) (*ToolPreflightResult, error) {
	command := buildToolPreflightCommand(defaultToolDefinitions)
	result, err := runner.RunSSHCommand(ctx, nodeID, command)
	if err != nil {
		return nil, err
	}

	output := ""
	if result != nil {
		output = result.Stdout
	}

	return &ToolPreflightResult{
		NodeID: nodeID,
		Tools:  parseToolPreflightOutput(output, defaultToolDefinitions),
	}, nil
}

func buildToolPreflightCommand(tools []ToolDefinition) string {
	parts := make([]string, 0, len(tools))
	for _, tool := range tools {
		parts = append(parts, fmt.Sprintf("path=$(%s 2>/dev/null || true); printf '%s|%%s\\n' \"$path\"", tool.Command, tool.Name))
	}
	return strings.Join(parts, "; ")
}

func parseToolPreflightOutput(output string, tools []ToolDefinition) []ToolCheck {
	paths := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		name, path, ok := strings.Cut(line, "|")
		if !ok {
			continue
		}
		paths[strings.TrimSpace(name)] = strings.TrimSpace(path)
	}

	checks := make([]ToolCheck, 0, len(tools))
	for _, tool := range tools {
		path := paths[tool.Name]
		checks = append(checks, ToolCheck{
			Name:      tool.Name,
			Available: path != "",
			Path:      path,
		})
	}

	return checks
}
