package node

import (
	"context"
	"strings"
	"testing"

	"github.com/antigravity/prometheus/internal/ssh"
	"github.com/google/uuid"
)

type preflightRunner struct {
	commands []string
	output   string
	err      error
}

func (r *preflightRunner) RunSSHCommand(ctx context.Context, nodeID uuid.UUID, command string) (*ssh.CommandResult, error) {
	r.commands = append(r.commands, command)
	return &ssh.CommandResult{Stdout: r.output, ExitCode: 0}, r.err
}

func TestParseToolPreflightOutput(t *testing.T) {
	output := "nmap|/usr/bin/nmap\nss|/usr/sbin/ss\njournalctl|\npct|/usr/sbin/pct\nqm|/usr/sbin/qm\n"

	checks := parseToolPreflightOutput(output, []ToolDefinition{
		{Name: "nmap", Command: "command -v nmap"},
		{Name: "ss", Command: "command -v ss"},
		{Name: "journalctl", Command: "command -v journalctl"},
		{Name: "pct", Command: "command -v pct"},
		{Name: "qm", Command: "command -v qm"},
	})

	if len(checks) != 5 {
		t.Fatalf("expected 5 checks, got %d", len(checks))
	}
	if !checks[0].Available || checks[0].Path != "/usr/bin/nmap" {
		t.Fatalf("expected nmap to be available with path, got %+v", checks[0])
	}
	if checks[2].Available {
		t.Fatalf("expected journalctl to be unavailable, got %+v", checks[2])
	}
}

func TestRunToolPreflightUsesSingleShellCommand(t *testing.T) {
	// Provide stdout matching every entry in defaultToolDefinitions so the
	// test is robust against new tools being added to that list.
	var b strings.Builder
	for _, def := range defaultToolDefinitions {
		b.WriteString(def.Name)
		b.WriteString("|/usr/bin/")
		b.WriteString(def.Name)
		b.WriteString("\n")
	}
	runner := &preflightRunner{output: b.String()}
	nodeID := uuid.New()

	result, err := RunToolPreflight(context.Background(), runner, nodeID)
	if err != nil {
		t.Fatalf("RunToolPreflight returned error: %v", err)
	}
	if result.NodeID != nodeID {
		t.Fatalf("expected node id %s, got %s", nodeID, result.NodeID)
	}
	if len(runner.commands) != 1 {
		t.Fatalf("expected one SSH command, got %d", len(runner.commands))
	}
	if !strings.Contains(runner.commands[0], "command -v nmap") {
		t.Fatalf("expected nmap check in command, got %q", runner.commands[0])
	}
	if len(result.Tools) != len(defaultToolDefinitions) {
		t.Fatalf("expected %d tools, got %d", len(defaultToolDefinitions), len(result.Tools))
	}
}
