package proxmox

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	tokenID    string
	tokenSecret string
	httpClient *http.Client
}

func NewClient(hostname string, port int, tokenID, tokenSecret string) *Client {
	return &Client{
		baseURL:    fmt.Sprintf("https://%s:%d/api2/json", hostname, port),
		tokenID:    tokenID,
		tokenSecret: tokenSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

func (c *Client) doRequest(ctx context.Context, method, path string) (json.RawMessage, error) {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s=%s", c.tokenID, c.tokenSecret))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return result.Data, nil
}

func (c *Client) GetVersion(ctx context.Context) (*VersionInfo, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/version")
	if err != nil {
		return nil, err
	}
	var v VersionInfo
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("unmarshal version: %w", err)
	}
	return &v, nil
}

func (c *Client) GetNodeStatus(ctx context.Context, node string) (*NodeStatus, error) {
	data, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/nodes/%s/status", node))
	if err != nil {
		return nil, err
	}

	var raw struct {
		CPU     float64 `json:"cpu"`
		Memory  struct {
			Total int64 `json:"total"`
			Used  int64 `json:"used"`
			Free  int64 `json:"free"`
		} `json:"memory"`
		RootFS struct {
			Total int64 `json:"total"`
			Used  int64 `json:"used"`
		} `json:"rootfs"`
		Uptime  int64  `json:"uptime"`
		KVersion string `json:"kversion"`
		PVEVersion string `json:"pveversion"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal node status: %w", err)
	}

	return &NodeStatus{
		Node:       node,
		Status:     "online",
		Uptime:     raw.Uptime,
		CPUUsage:   raw.CPU,
		MemTotal:   raw.Memory.Total,
		MemUsed:    raw.Memory.Used,
		MemFree:    raw.Memory.Free,
		DiskTotal:  raw.RootFS.Total,
		DiskUsed:   raw.RootFS.Used,
		KVersion:   raw.KVersion,
		PVEVersion: raw.PVEVersion,
	}, nil
}

func (c *Client) GetNodes(ctx context.Context) ([]string, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/nodes")
	if err != nil {
		return nil, err
	}

	var nodes []struct {
		Node   string `json:"node"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(data, &nodes); err != nil {
		return nil, fmt.Errorf("unmarshal nodes: %w", err)
	}

	result := make([]string, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, n.Node)
	}
	return result, nil
}

func (c *Client) GetVMs(ctx context.Context, node string) ([]VMInfo, error) {
	qemuData, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/nodes/%s/qemu", node))
	if err != nil {
		return nil, err
	}

	var qemuVMs []VMInfo
	if err := json.Unmarshal(qemuData, &qemuVMs); err != nil {
		return nil, fmt.Errorf("unmarshal qemu vms: %w", err)
	}
	for i := range qemuVMs {
		qemuVMs[i].Type = "qemu"
	}

	lxcData, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/nodes/%s/lxc", node))
	if err != nil {
		return nil, err
	}

	var lxcCTs []VMInfo
	if err := json.Unmarshal(lxcData, &lxcCTs); err != nil {
		return nil, fmt.Errorf("unmarshal lxc cts: %w", err)
	}
	for i := range lxcCTs {
		lxcCTs[i].Type = "lxc"
	}

	return append(qemuVMs, lxcCTs...), nil
}

func (c *Client) GetStorage(ctx context.Context, node string) ([]StorageInfo, error) {
	data, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/nodes/%s/storage", node))
	if err != nil {
		return nil, err
	}

	var storages []StorageInfo
	if err := json.Unmarshal(data, &storages); err != nil {
		return nil, fmt.Errorf("unmarshal storage: %w", err)
	}

	for i := range storages {
		if storages[i].Total > 0 {
			storages[i].UsagePercent = float64(storages[i].Used) / float64(storages[i].Total) * 100
		}
	}

	return storages, nil
}

func (c *Client) GetNetworkInterfaces(ctx context.Context, node string) ([]NetworkInterface, error) {
	data, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/nodes/%s/network", node))
	if err != nil {
		return nil, err
	}
	var ifaces []NetworkInterface
	if err := json.Unmarshal(data, &ifaces); err != nil {
		return nil, fmt.Errorf("unmarshal network interfaces: %w", err)
	}
	return ifaces, nil
}

func (c *Client) GetDisks(ctx context.Context, node string) ([]DiskInfo, error) {
	data, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/nodes/%s/disks/list", node))
	if err != nil {
		return nil, err
	}
	var disks []DiskInfo
	if err := json.Unmarshal(data, &disks); err != nil {
		return nil, fmt.Errorf("unmarshal disks: %w", err)
	}
	return disks, nil
}

// doRequestWithBody sends a request with a form-encoded body (for POST/PUT).
func (c *Client) doRequestWithBody(ctx context.Context, method, path string, params url.Values) (json.RawMessage, error) {
	reqURL := c.baseURL + path

	var body io.Reader
	if params != nil {
		body = strings.NewReader(params.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s=%s", c.tokenID, c.tokenSecret))
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return result.Data, nil
}

// StopVM stops a VM/CT on the given node. Returns the task UPID.
func (c *Client) StopVM(ctx context.Context, node string, vmid int, vmType string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/%s/%d/status/stop", node, vmType, vmid)
	data, err := c.doRequestWithBody(ctx, http.MethodPost, path, nil)
	if err != nil {
		return "", fmt.Errorf("stop vm %d: %w", vmid, err)
	}
	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", fmt.Errorf("unmarshal stop upid: %w", err)
	}
	return upid, nil
}

// StartVM starts a VM/CT on the given node. Returns the task UPID.
func (c *Client) StartVM(ctx context.Context, node string, vmid int, vmType string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/%s/%d/status/start", node, vmType, vmid)
	data, err := c.doRequestWithBody(ctx, http.MethodPost, path, nil)
	if err != nil {
		return "", fmt.Errorf("start vm %d: %w", vmid, err)
	}
	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", fmt.Errorf("unmarshal start upid: %w", err)
	}
	return upid, nil
}

// ShutdownVM gracefully shuts down a VM/CT. Returns the task UPID.
func (c *Client) ShutdownVM(ctx context.Context, node string, vmid int, vmType string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/%s/%d/status/shutdown", node, vmType, vmid)
	data, err := c.doRequestWithBody(ctx, http.MethodPost, path, nil)
	if err != nil {
		return "", fmt.Errorf("shutdown vm %d: %w", vmid, err)
	}
	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", fmt.Errorf("unmarshal shutdown upid: %w", err)
	}
	return upid, nil
}

// SuspendVM suspends (pauses) a VM. Returns the task UPID.
func (c *Client) SuspendVM(ctx context.Context, node string, vmid int) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/status/suspend", node, vmid)
	data, err := c.doRequestWithBody(ctx, http.MethodPost, path, nil)
	if err != nil {
		return "", fmt.Errorf("suspend vm %d: %w", vmid, err)
	}
	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", fmt.Errorf("unmarshal suspend upid: %w", err)
	}
	return upid, nil
}

// ResumeVM resumes a suspended VM. Returns the task UPID.
func (c *Client) ResumeVM(ctx context.Context, node string, vmid int) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/status/resume", node, vmid)
	data, err := c.doRequestWithBody(ctx, http.MethodPost, path, nil)
	if err != nil {
		return "", fmt.Errorf("resume vm %d: %w", vmid, err)
	}
	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", fmt.Errorf("unmarshal resume upid: %w", err)
	}
	return upid, nil
}

// GetTaskStatus returns the status of a Proxmox task by UPID.
func (c *Client) GetTaskStatus(ctx context.Context, node string, upid string) (*TaskStatus, error) {
	encodedUPID := url.PathEscape(upid)
	data, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/nodes/%s/tasks/%s/status", node, encodedUPID))
	if err != nil {
		return nil, fmt.Errorf("get task status: %w", err)
	}
	var status TaskStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("unmarshal task status: %w", err)
	}
	return &status, nil
}

// GetTaskLog returns log entries for a Proxmox task.
func (c *Client) GetTaskLog(ctx context.Context, node string, upid string, start int) ([]TaskLogEntry, error) {
	encodedUPID := url.PathEscape(upid)
	path := fmt.Sprintf("/nodes/%s/tasks/%s/log?start=%d&limit=500", node, encodedUPID, start)
	data, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, fmt.Errorf("get task log: %w", err)
	}
	var entries []TaskLogEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("unmarshal task log: %w", err)
	}
	return entries, nil
}

// CreateVzdump creates a vzdump backup of a VM/CT. Returns the task UPID.
func (c *Client) CreateVzdump(ctx context.Context, node string, vmid int, opts VzdumpOptions) (string, error) {
	params := url.Values{}
	params.Set("vmid", fmt.Sprintf("%d", vmid))
	if opts.Storage != "" {
		params.Set("storage", opts.Storage)
	}
	if opts.Mode != "" {
		params.Set("mode", opts.Mode)
	}
	if opts.Compress != "" {
		params.Set("compress", opts.Compress)
	}
	if opts.Remove > 0 {
		params.Set("remove", fmt.Sprintf("%d", opts.Remove))
	}

	data, err := c.doRequestWithBody(ctx, http.MethodPost, fmt.Sprintf("/nodes/%s/vzdump", node), params)
	if err != nil {
		return "", fmt.Errorf("create vzdump for vm %d: %w", vmid, err)
	}
	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", fmt.Errorf("unmarshal vzdump upid: %w", err)
	}
	return upid, nil
}

// GetVMRRDData returns RRD performance data for a VM/CT.
func (c *Client) GetVMRRDData(ctx context.Context, node string, vmid int, vmType string) ([]VMRRDDataPoint, error) {
	path := fmt.Sprintf("/nodes/%s/%s/%d/rrddata?timeframe=day", node, vmType, vmid)
	data, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, fmt.Errorf("get vm rrd data: %w", err)
	}
	var points []VMRRDDataPoint
	if err := json.Unmarshal(data, &points); err != nil {
		return nil, fmt.Errorf("unmarshal rrd data: %w", err)
	}
	return points, nil
}

// RestoreVM restores a VM from a vzdump archive. Returns the task UPID.
func (c *Client) RestoreVM(ctx context.Context, node string, archive string, storage string, vmid int) (string, error) {
	params := url.Values{}
	params.Set("archive", archive)
	params.Set("storage", storage)
	params.Set("vmid", fmt.Sprintf("%d", vmid))
	params.Set("force", "1")

	data, err := c.doRequestWithBody(ctx, http.MethodPost, fmt.Sprintf("/nodes/%s/qemu", node), params)
	if err != nil {
		return "", fmt.Errorf("restore vm %d: %w", vmid, err)
	}
	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		// Some Proxmox versions return UPID directly as string
		upid = strings.Trim(string(data), "\"")
		if upid == "" {
			return "", fmt.Errorf("unmarshal restore upid: %w", err)
		}
	}
	return upid, nil
}

