package proxmox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// unwrapAgentResult handles the Proxmox agent response quirk where agent
// endpoints wrap their payload in an extra {"result": ...} layer on top of
// the standard {"data": ...} envelope (already stripped by doRequest).
func unwrapAgentResult(data json.RawMessage) json.RawMessage {
	if !bytes.Contains(data, []byte(`"result"`)) {
		return data
	}
	var wrapper struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(data, &wrapper); err == nil && len(wrapper.Result) > 0 {
		return wrapper.Result
	}
	return data
}

type ExecResult struct {
	ExitCode int    `json:"exitcode"`
	OutData  string `json:"out-data"`
	ErrData  string `json:"err-data"`
}

// ExecCommand runs a command inside a VM/container via Proxmox API.
// For LXC containers, this uses the pct exec API endpoint.
// For QEMU VMs, this requires the qemu-guest-agent to be installed and running inside the VM.
func (c *Client) ExecCommand(ctx context.Context, node string, vmid int, vmType string, command []string) (*ExecResult, error) {
	if err := validateVMType(vmType); err != nil {
		return nil, err
	}
	if len(command) == 0 {
		return nil, fmt.Errorf("command darf nicht leer sein")
	}
	if vmType == "lxc" {
		return c.execLXC(ctx, node, vmid, command)
	}
	return c.execQEMU(ctx, node, vmid, command)
}

func (c *Client) execLXC(ctx context.Context, node string, vmid int, command []string) (*ExecResult, error) {
	path := fmt.Sprintf("/nodes/%s/lxc/%d/status/exec", node, vmid)
	params := url.Values{}
	params.Set("command", strings.Join(command, " "))
	data, err := c.doRequestWithBody(ctx, http.MethodPost, path, params)
	if err != nil {
		return nil, fmt.Errorf("exec lxc command: %w", err)
	}
	var result ExecResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse exec result: %w", err)
	}
	return &result, nil
}

func (c *Client) execQEMU(ctx context.Context, node string, vmid int, command []string) (*ExecResult, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/agent/exec", node, vmid)
	params := url.Values{}
	// Proxmox API expects the full command as a single string
	params.Set("command", strings.Join(command, " "))
	data, err := c.doRequestWithBody(ctx, http.MethodPost, path, params)
	if err != nil {
		// The client layer already detects "guest agent not running" in doRequestWithBody.
		// Add VM context for better error messages upstream.
		return nil, fmt.Errorf("exec auf QEMU VM %d: %w", vmid, err)
	}
	inner := unwrapAgentResult(data)
	var pidResp struct {
		PID int `json:"pid"`
	}
	if err := json.Unmarshal(inner, &pidResp); err != nil {
		return nil, fmt.Errorf("parse exec pid: %w (raw: %s)", err, string(data))
	}
	if pidResp.PID == 0 {
		return nil, fmt.Errorf("exec returned pid 0 (raw: %s)", string(data))
	}
	return c.waitExecResult(ctx, node, vmid, pidResp.PID)
}

func (c *Client) waitExecResult(ctx context.Context, node string, vmid int, pid int) (*ExecResult, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/agent/exec-status?pid=%d", node, vmid, pid)
	// Use context deadline if available, otherwise default to 30s
	deadline, ok := ctx.Deadline()
	timeoutDur := 30 * time.Second
	if ok {
		remaining := time.Until(deadline)
		if remaining < timeoutDur {
			timeoutDur = remaining
		}
	}
	timer := time.NewTimer(timeoutDur)
	defer timer.Stop()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	type execStatus struct {
		Exited   int    `json:"exited"`
		ExitCode int    `json:"exitcode"`
		OutData  string `json:"out-data"`
		ErrData  string `json:"err-data"`
	}

	consecutiveErrors := 0
	const maxConsecutiveErrors = 5 // Tolerate transient polling failures

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("exec auf VM %d abgebrochen (PID %d): %w", vmid, pid, ctx.Err())
		case <-timer.C:
			return nil, fmt.Errorf("exec timeout nach %s für VM %d (PID %d). Der Befehl läuft möglicherweise noch im Hintergrund", timeoutDur, vmid, pid)
		case <-ticker.C:
			data, err := c.doRequest(ctx, http.MethodGet, path)
			if err != nil {
				consecutiveErrors++
				if consecutiveErrors >= maxConsecutiveErrors {
					return nil, fmt.Errorf("exec status nach %d Versuchen nicht abrufbar (VM %d, PID %d): %w", maxConsecutiveErrors, vmid, pid, err)
				}
				continue
			}
			consecutiveErrors = 0
			inner := unwrapAgentResult(data)
			var status execStatus
			if err := json.Unmarshal(inner, &status); err != nil {
				return nil, fmt.Errorf("parse exec status: %w", err)
			}
			if status.Exited == 1 {
				return &ExecResult{
					ExitCode: status.ExitCode,
					OutData:  status.OutData,
					ErrData:  status.ErrData,
				}, nil
			}
		}
	}
}
