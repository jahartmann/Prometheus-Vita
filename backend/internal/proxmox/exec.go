package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

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
	params.Set("command", command[0])
	for i, arg := range command[1:] {
		params.Set(fmt.Sprintf("arg%d", i), arg)
	}
	data, err := c.doRequestWithBody(ctx, http.MethodPost, path, params)
	if err != nil {
		return nil, fmt.Errorf("exec qemu command: %w", err)
	}
	var pidResp struct {
		PID int `json:"pid"`
	}
	if err := json.Unmarshal(data, &pidResp); err != nil {
		return nil, fmt.Errorf("parse exec pid: %w", err)
	}
	return c.waitExecResult(ctx, node, vmid, pidResp.PID)
}

func (c *Client) waitExecResult(ctx context.Context, node string, vmid int, pid int) (*ExecResult, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/agent/exec-status?pid=%d", node, vmid, pid)
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, fmt.Errorf("exec timeout after 30s")
		case <-ticker.C:
			data, err := c.doRequest(ctx, http.MethodGet, path)
			if err != nil {
				return nil, fmt.Errorf("poll exec status: %w", err)
			}
			var status struct {
				Exited   int    `json:"exited"`
				ExitCode int    `json:"exitcode"`
				OutData  string `json:"out-data"`
				ErrData  string `json:"err-data"`
			}
			if err := json.Unmarshal(data, &status); err != nil {
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
