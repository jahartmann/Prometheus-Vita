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
func (c *Client) ExecCommand(ctx context.Context, node string, vmid int, vmType string, command []string) (*ExecResult, error) {
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
		return nil, fmt.Errorf("exec qemu command: %w", err)
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

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			return nil, fmt.Errorf("exec timeout after %s", timeoutDur)
		case <-ticker.C:
			data, err := c.doRequest(ctx, http.MethodGet, path)
			if err != nil {
				return nil, fmt.Errorf("poll exec status: %w", err)
			}
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
