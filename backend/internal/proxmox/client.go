package proxmox

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// maxResponseSize limits the size of Proxmox API responses to prevent OOM
// on malformed or proxied responses. 10 MB is generous for any PVE endpoint.
const maxResponseSize = 10 * 1024 * 1024

// validVMTypes lists accepted VM type values for Proxmox API calls.
var validVMTypes = map[string]bool{"qemu": true, "lxc": true}

// validateVMType returns an error if vmType is not "qemu" or "lxc".
func validateVMType(vmType string) error {
	if !validVMTypes[vmType] {
		return fmt.Errorf("ungültiger VM-Typ %q (erlaubt: qemu, lxc)", vmType)
	}
	return nil
}

type Client struct {
	baseURL    string
	tokenID    string
	tokenSecret string
	httpClient *http.Client
}

func NewClient(hostname string, port int, tokenID, tokenSecret string, tlsCfg *tls.Config) *Client {
	if tlsCfg == nil {
		slog.Warn("Proxmox TLS-Verifikation deaktiviert – Verbindung ist anfällig für MITM-Angriffe", slog.String("host", hostname))
		tlsCfg = &tls.Config{InsecureSkipVerify: true}
	}
	return &Client{
		baseURL:    fmt.Sprintf("https://%s:%d/api2/json", hostname, port),
		tokenID:    tokenID,
		tokenSecret: tokenSecret,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig:       tlsCfg,
				DialContext:           (&net.Dialer{Timeout: 15 * time.Second}).DialContext,
				TLSHandshakeTimeout:   15 * time.Second,
				ResponseHeaderTimeout: 60 * time.Second,
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
		return nil, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, fmt.Errorf("authentication failed (HTTP %d) for %s: %s", resp.StatusCode, path, truncateBody(body))
	}
	if resp.StatusCode == 500 && strings.Contains(string(body), "QEMU guest agent is not running") {
		return nil, fmt.Errorf("QEMU Guest Agent ist nicht aktiv auf der VM (Pfad: %s). Bitte qemu-guest-agent installieren und starten", path)
	}
	if resp.StatusCode == 500 && strings.Contains(string(body), "not running") {
		return nil, fmt.Errorf("VM/CT ist nicht aktiv (HTTP 500 für %s): %s", path, truncateBody(body))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error %d on %s: %s", resp.StatusCode, path, truncateBody(body))
	}

	var result struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return result.Data, nil
}

// truncateBody limits error body output to 500 chars to prevent log flooding.
func truncateBody(body []byte) string {
	s := string(body)
	if len(s) > 500 {
		return s[:500] + "... (truncated)"
	}
	return s
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
		Swap struct {
			Total int64 `json:"total"`
			Used  int64 `json:"used"`
			Free  int64 `json:"free"`
		} `json:"swap"`
		RootFS struct {
			Total int64 `json:"total"`
			Used  int64 `json:"used"`
			Free  int64 `json:"free"`
			Avail int64 `json:"avail"`
		} `json:"rootfs"`
		CPUInfo struct {
			CPUs  int    `json:"cpus"`
			Model string `json:"model"`
		} `json:"cpuinfo"`
		NetIn      float64  `json:"netin"`
		NetOut     float64  `json:"netout"`
		LoadAvg    []string `json:"loadavg"`
		Uptime     int64    `json:"uptime"`
		KVersion   string   `json:"kversion"`
		PVEVersion string   `json:"pveversion"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal node status: %w", err)
	}

	loadAvg := make([]float64, 0, len(raw.LoadAvg))
	for _, s := range raw.LoadAvg {
		var f float64
		if _, err := fmt.Sscanf(s, "%f", &f); err == nil {
			loadAvg = append(loadAvg, f)
		}
	}

	// Ensure LoadAvg always has exactly 3 elements (1min, 5min, 15min)
	for len(loadAvg) < 3 {
		loadAvg = append(loadAvg, 0)
	}
	if len(loadAvg) > 3 {
		loadAvg = loadAvg[:3]
	}

	// Validate CPU is in valid range
	cpuUsage := raw.CPU * 100
	if math.IsNaN(cpuUsage) || math.IsInf(cpuUsage, 0) || cpuUsage < 0 {
		cpuUsage = 0
	}
	if cpuUsage > 100 {
		cpuUsage = 100 // Cap at 100% (can happen with overcommit)
	}

	// Validate memory values are non-negative and consistent
	if raw.Memory.Total < 0 {
		raw.Memory.Total = 0
	}
	if raw.Memory.Used < 0 {
		raw.Memory.Used = 0
	}
	if raw.Memory.Used > raw.Memory.Total {
		raw.Memory.Used = raw.Memory.Total
	}

	return &NodeStatus{
		Node:       node,
		Status:     "online",
		Uptime:     raw.Uptime,
		CPUUsage:   cpuUsage,
		CPUCores:   raw.CPUInfo.CPUs,
		CPUModel:   raw.CPUInfo.Model,
		MemTotal:   raw.Memory.Total,
		MemUsed:    raw.Memory.Used,
		MemFree:    raw.Memory.Free,
		SwapTotal:  raw.Swap.Total,
		SwapUsed:   raw.Swap.Used,
		DiskTotal:  raw.RootFS.Total,
		DiskUsed:   raw.RootFS.Used,
		NetIn:      int64(raw.NetIn),
		NetOut:     int64(raw.NetOut),
		LoadAvg:    loadAvg,
		KVersion:   raw.KVersion,
		PVEVersion: raw.PVEVersion,
	}, nil
}

// GetNodeSyslog retrieves syslog entries from the Proxmox API.
func (c *Client) GetNodeSyslog(ctx context.Context, node string, limit int) (string, error) {
	path := fmt.Sprintf("/nodes/%s/syslog?limit=%d", node, limit)
	data, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return "", fmt.Errorf("get node syslog: %w", err)
	}
	var entries []struct {
		N int    `json:"n"`
		T string `json:"t"`
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return "", fmt.Errorf("parse syslog: %w", err)
	}
	var lines string
	for _, e := range entries {
		lines += e.T + "\n"
	}
	return lines, nil
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
	var allVMs []VMInfo
	var qemuErr, lxcErr error

	// Fetch QEMU VMs - continue even if this fails (LXC might still work)
	qemuData, qemuErr := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/nodes/%s/qemu", node))
	if qemuErr == nil {
		var qemuVMs []VMInfo
		if err := json.Unmarshal(qemuData, &qemuVMs); err == nil {
			for i := range qemuVMs {
				qemuVMs[i].Type = "qemu"
			}
			allVMs = append(allVMs, qemuVMs...)
		}
	}

	// Fetch LXC containers - continue even if this fails (QEMU might have worked)
	lxcData, lxcErr := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/nodes/%s/lxc", node))
	if lxcErr == nil {
		var lxcCTs []VMInfo
		if err := json.Unmarshal(lxcData, &lxcCTs); err == nil {
			for i := range lxcCTs {
				lxcCTs[i].Type = "lxc"
			}
			allVMs = append(allVMs, lxcCTs...)
		}
	}

	// If BOTH endpoints failed, the node is likely unreachable or the PVE name is wrong.
	// Return an error so the caller can trigger fallback logic.
	if qemuErr != nil && lxcErr != nil {
		return nil, fmt.Errorf("beide VM-Endpunkte fehlgeschlagen für Node '%s': QEMU: %v, LXC: %v", node, qemuErr, lxcErr)
	}

	// Log partial failures so operators can see incomplete data
	if qemuErr != nil && lxcErr == nil {
		slog.Warn("VM-Abfrage teilweise fehlgeschlagen: QEMU-Endpunkt nicht erreichbar",
			slog.String("node", node), slog.Any("error", qemuErr))
	}
	if lxcErr != nil && qemuErr == nil {
		slog.Warn("VM-Abfrage teilweise fehlgeschlagen: LXC-Endpunkt nicht erreichbar",
			slog.String("node", node), slog.Any("error", lxcErr))
	}

	return allVMs, nil
}

func (c *Client) GetStorage(ctx context.Context, node string) ([]StorageInfo, error) {
	data, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/nodes/%s/storage", node))
	if err != nil {
		return nil, err
	}

	var raw []struct {
		Storage string `json:"storage"`
		Type    string `json:"type"`
		Content string `json:"content"`
		Total   int64  `json:"total"`
		Used    int64  `json:"used"`
		Avail   int64  `json:"avail"`
		Active  *int   `json:"active"`
		Enabled *int   `json:"enabled"`
		Shared  *int   `json:"shared"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal storage: %w", err)
	}

	storages := make([]StorageInfo, 0, len(raw))
	for _, r := range raw {
		isActive := r.Active == nil || *r.Active == 1
		isShared := r.Shared != nil && *r.Shared == 1

		s := StorageInfo{
			Storage:   r.Storage,
			Type:      r.Type,
			Content:   r.Content,
			Total:     r.Total,
			Used:      r.Used,
			Available: r.Avail,
			Active:    isActive,
			Shared:    isShared,
		}
		if s.Total > 0 {
			s.UsagePercent = float64(s.Used) / float64(s.Total) * 100
		}
		// Cap usage percent at valid range (thin-provisioned storage can report Available > Total)
		if s.UsagePercent < 0 {
			s.UsagePercent = 0
		}
		if s.UsagePercent > 100 {
			s.UsagePercent = 100
		}
		// Ensure Available is non-negative
		if s.Available < 0 {
			s.Available = 0
		}
		storages = append(storages, s)
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
		return nil, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, fmt.Errorf("authentication failed (HTTP %d) for %s: %s", resp.StatusCode, path, truncateBody(respBody))
	}
	if resp.StatusCode == 500 && strings.Contains(string(respBody), "QEMU guest agent is not running") {
		return nil, fmt.Errorf("QEMU Guest Agent ist nicht aktiv auf der VM (Pfad: %s). Bitte qemu-guest-agent installieren und starten", path)
	}
	if resp.StatusCode == 500 && strings.Contains(string(respBody), "not running") {
		return nil, fmt.Errorf("VM/CT ist nicht aktiv (HTTP 500 für %s): %s", path, truncateBody(respBody))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error %d on %s: %s", resp.StatusCode, path, truncateBody(respBody))
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
	if err := validateVMType(vmType); err != nil {
		return "", err
	}
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
	if err := validateVMType(vmType); err != nil {
		return "", err
	}
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
	if err := validateVMType(vmType); err != nil {
		return "", err
	}
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

// SuspendVM suspends (pauses) a QEMU VM. Returns the task UPID.
// NOTE: Proxmox does not support suspending LXC containers — use freeze/unfreeze
// via lxc-freeze on the host if needed. This function is QEMU-only.
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

// GetVMConfig returns the raw VM configuration as key-value pairs.
func (c *Client) GetVMConfig(ctx context.Context, node string, vmid int, vmType string) (map[string]interface{}, error) {
	if err := validateVMType(vmType); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/nodes/%s/%s/%d/config", node, vmType, vmid)
	data, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, fmt.Errorf("get vm config: %w", err)
	}
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("unmarshal vm config: %w", err)
	}
	return config, nil
}

// SetVMConfig updates VM configuration parameters.
func (c *Client) SetVMConfig(ctx context.Context, node string, vmid int, vmType string, params url.Values) error {
	if err := validateVMType(vmType); err != nil {
		return err
	}
	path := fmt.Sprintf("/nodes/%s/%s/%d/config", node, vmType, vmid)
	_, err := c.doRequestWithBody(ctx, http.MethodPut, path, params)
	return err
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
	if vmid <= 0 {
		return "", fmt.Errorf("ungültige VMID %d für vzdump", vmid)
	}
	// Validate vzdump mode (Proxmox supports: stop, suspend, snapshot)
	validModes := map[string]bool{"stop": true, "suspend": true, "snapshot": true}
	if opts.Mode != "" && !validModes[opts.Mode] {
		return "", fmt.Errorf("ungültiger vzdump-Modus %q (erlaubt: stop, suspend, snapshot)", opts.Mode)
	}

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

// GetNodeRRDData returns RRD performance data for a node.
func (c *Client) GetNodeRRDData(ctx context.Context, node string, timeframe string) ([]RRDDataPoint, error) {
	data, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/nodes/%s/rrddata?timeframe=%s", node, timeframe))
	if err != nil {
		return nil, err
	}

	var raw []struct {
		Time      float64  `json:"time"`
		CPU       *float64 `json:"cpu"`
		NetIn     *float64 `json:"netin"`
		NetOut    *float64 `json:"netout"`
		MemUsed   *float64 `json:"memused"`
		MemTotal  *float64 `json:"memtotal"`
		RootUsed  *float64 `json:"rootused"`
		RootTotal *float64 `json:"roottotal"`
		LoadAvg   *float64 `json:"loadavg"`
		IOWait    *float64 `json:"iowait"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal rrd data: %w", err)
	}

	var points []RRDDataPoint
	for _, r := range raw {
		p := RRDDataPoint{
			Time: int64(r.Time),
		}
		// NOTE: nil fields default to 0 — we cannot distinguish "0% usage" from
		// "data unavailable" without changing the RRDDataPoint struct to use pointers.
		// Guard against NaN/Inf values leaking through from Proxmox RRD gaps.
		if r.CPU != nil {
			v := *r.CPU
			if !math.IsNaN(v) && !math.IsInf(v, 0) {
				p.CPU = v
			}
		}
		if r.NetIn != nil {
			v := *r.NetIn
			if !math.IsNaN(v) && !math.IsInf(v, 0) {
				p.NetIn = v
			}
		}
		if r.NetOut != nil {
			v := *r.NetOut
			if !math.IsNaN(v) && !math.IsInf(v, 0) {
				p.NetOut = v
			}
		}
		if r.MemUsed != nil {
			v := *r.MemUsed
			if !math.IsNaN(v) && !math.IsInf(v, 0) {
				p.MemUsed = int64(v)
			}
		}
		if r.MemTotal != nil {
			v := *r.MemTotal
			if !math.IsNaN(v) && !math.IsInf(v, 0) {
				p.MemTotal = int64(v)
			}
		}
		if r.RootUsed != nil {
			v := *r.RootUsed
			if !math.IsNaN(v) && !math.IsInf(v, 0) {
				p.RootUsed = int64(v)
			}
		}
		if r.RootTotal != nil {
			v := *r.RootTotal
			if !math.IsNaN(v) && !math.IsInf(v, 0) {
				p.RootTotal = int64(v)
			}
		}
		if r.LoadAvg != nil {
			v := *r.LoadAvg
			if !math.IsNaN(v) && !math.IsInf(v, 0) {
				p.LoadAvg = v
			}
		}
		if r.IOWait != nil {
			v := *r.IOWait
			if !math.IsNaN(v) && !math.IsInf(v, 0) {
				p.IOWait = v
			}
		}
		points = append(points, p)
	}
	return points, nil
}

// GetVMRRDData returns RRD performance data for a VM/CT.
func (c *Client) GetVMRRDData(ctx context.Context, node string, vmid int, vmType string, timeframe string) ([]VMRRDDataPoint, error) {
	path := fmt.Sprintf("/nodes/%s/%s/%d/rrddata?timeframe=%s", node, vmType, vmid, timeframe)
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

// GetTicket authenticates via username/password and returns a ticket + CSRF token.
// If tlsCfg is nil, TLS verification is skipped (insecure, backward compatible).
func GetTicket(ctx context.Context, hostname string, port int, username, password string, tlsCfg *tls.Config) (ticket string, csrf string, err error) {
	if tlsCfg == nil {
		tlsCfg = &tls.Config{InsecureSkipVerify: true}
	}
	baseURL := fmt.Sprintf("https://%s:%d/api2/json", hostname, port)
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
		},
	}

	params := url.Values{}
	params.Set("username", username)
	params.Set("password", password)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/access/ticket", strings.NewReader(params.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("create ticket request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("execute ticket request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return "", "", fmt.Errorf("read ticket response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("ticket auth failed (HTTP %d): %s", resp.StatusCode, truncateBody(body))
	}

	var result struct {
		Data struct {
			Ticket              string `json:"ticket"`
			CSRFPreventionToken string `json:"CSRFPreventionToken"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("unmarshal ticket response: %w", err)
	}

	if result.Data.Ticket == "" {
		return "", "", fmt.Errorf("empty ticket in response")
	}

	return result.Data.Ticket, result.Data.CSRFPreventionToken, nil
}

// CreateAPITokenWithTicket creates an API token using cookie-based ticket authentication.
// If tlsCfg is nil, TLS verification is skipped (insecure, backward compatible).
func CreateAPITokenWithTicket(ctx context.Context, hostname string, port int, username, ticket, csrf, tokenName string, tlsCfg *tls.Config) (tokenID string, tokenSecret string, err error) {
	if tlsCfg == nil {
		tlsCfg = &tls.Config{InsecureSkipVerify: true}
	}
	baseURL := fmt.Sprintf("https://%s:%d/api2/json", hostname, port)
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
		},
	}

	path := fmt.Sprintf("/access/users/%s/token/%s", url.PathEscape(username), url.PathEscape(tokenName))
	params := url.Values{}
	params.Set("privsep", "0")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, strings.NewReader(params.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("CSRFPreventionToken", csrf)
	req.AddCookie(&http.Cookie{Name: "PVEAuthCookie", Value: ticket})

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("execute token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return "", "", fmt.Errorf("read token response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("create token failed (HTTP %d): %s", resp.StatusCode, truncateBody(body))
	}

	var result struct {
		Data struct {
			FullTokenID string `json:"full-tokenid"`
			Value       string `json:"value"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("unmarshal token response: %w", err)
	}

	if result.Data.FullTokenID == "" || result.Data.Value == "" {
		return "", "", fmt.Errorf("empty token data in response")
	}

	return result.Data.FullTokenID, result.Data.Value, nil
}

// DeleteResource deletes a resource (VM/CT) via its API path. Returns the task UPID.
func (c *Client) DeleteResource(ctx context.Context, path string) (string, error) {
	data, err := c.doRequest(ctx, http.MethodDelete, path)
	if err != nil {
		return "", fmt.Errorf("delete resource %s: %w", path, err)
	}
	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		// Some endpoints don't return a UPID
		return "", nil
	}
	return upid, nil
}

// GetStorageRaw returns the raw JSON response from /nodes/{node}/storage for debugging.
func (c *Client) GetStorageRaw(ctx context.Context, node string) (json.RawMessage, error) {
	return c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/nodes/%s/storage", node))
}

// GetClusterResourcesRaw returns the raw JSON response from /cluster/resources?type=storage for debugging.
func (c *Client) GetClusterResourcesRaw(ctx context.Context) (json.RawMessage, error) {
	return c.doRequest(ctx, http.MethodGet, "/cluster/resources?type=storage")
}

// GetClusterStorages fetches storage info across all cluster nodes via
// /cluster/resources?type=storage. This avoids needing a specific PVE node name.
func (c *Client) GetClusterStorages(ctx context.Context) ([]StorageInfo, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/cluster/resources?type=storage")
	if err != nil {
		return nil, err
	}

	var raw []struct {
		Storage    string `json:"storage"`
		Node       string `json:"node"`
		Type       string `json:"type"`
		Content    string `json:"content"`
		MaxDisk    int64  `json:"maxdisk"`
		Disk       int64  `json:"disk"`
		Status     string `json:"status"`
		Shared     int    `json:"shared"`
		PluginType string `json:"plugintype"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal cluster storages: %w", err)
	}

	// Deduplicate shared storages (appear once per node)
	seen := make(map[string]bool)
	storages := make([]StorageInfo, 0, len(raw))
	for _, r := range raw {
		if seen[r.Storage] {
			continue
		}
		seen[r.Storage] = true

		s := StorageInfo{
			Storage:   r.Storage,
			Type:      r.Type,
			Content:   r.Content,
			Total:     r.MaxDisk,
			Used:      r.Disk,
			Available: r.MaxDisk - r.Disk,
			Active:    r.Status != "unknown",
			Shared:    r.Shared == 1,
		}
		if s.Total > 0 {
			s.UsagePercent = float64(s.Used) / float64(s.Total) * 100
		}
		storages = append(storages, s)
	}

	return storages, nil
}

// GetClusterStoragesForNode fetches storage info for a specific PVE node from
// the cluster resources endpoint. This is used as a fallback when the direct
// node storage endpoint fails, and correctly filters by node name.
func (c *Client) GetClusterStoragesForNode(ctx context.Context, nodeName string) ([]StorageInfo, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/cluster/resources?type=storage")
	if err != nil {
		return nil, err
	}

	var raw []struct {
		Storage    string `json:"storage"`
		Node       string `json:"node"`
		Type       string `json:"type"`
		Content    string `json:"content"`
		MaxDisk    int64  `json:"maxdisk"`
		Disk       int64  `json:"disk"`
		Status     string `json:"status"`
		Shared     int    `json:"shared"`
		PluginType string `json:"plugintype"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal cluster storages: %w", err)
	}

	storages := make([]StorageInfo, 0)
	for _, r := range raw {
		if r.Node != nodeName {
			continue
		}
		s := StorageInfo{
			Storage:   r.Storage,
			Type:      r.Type,
			Content:   r.Content,
			Total:     r.MaxDisk,
			Used:      r.Disk,
			Available: r.MaxDisk - r.Disk,
			Active:    r.Status != "unknown",
			Shared:    r.Shared == 1,
		}
		if s.Total > 0 {
			s.UsagePercent = float64(s.Used) / float64(s.Total) * 100
		}
		storages = append(storages, s)
	}
	return storages, nil
}

// GetStorageContent returns content of a specific type from a storage on a node.
func (c *Client) GetStorageContent(ctx context.Context, node, storage, contentType string) ([]StorageContent, error) {
	path := fmt.Sprintf("/nodes/%s/storage/%s/content?content=%s", node, url.QueryEscape(storage), url.QueryEscape(contentType))
	data, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, fmt.Errorf("get storage content: %w", err)
	}
	var content []StorageContent
	if err := json.Unmarshal(data, &content); err != nil {
		return nil, fmt.Errorf("unmarshal storage content: %w", err)
	}
	return content, nil
}

// DownloadURL downloads a file from a URL to a storage on a node. Returns the task UPID.
func (c *Client) DownloadURL(ctx context.Context, node, storage, filename, downloadURL string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/storage/%s/download-url", node, url.QueryEscape(storage))
	params := url.Values{}
	params.Set("filename", filename)
	params.Set("url", downloadURL)
	params.Set("content", "iso")
	data, err := c.doRequestWithBody(ctx, http.MethodPost, path, params)
	if err != nil {
		return "", fmt.Errorf("download url: %w", err)
	}
	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", fmt.Errorf("unmarshal download upid: %w", err)
	}
	return upid, nil
}

// ListSnapshots returns all snapshots for a VM/CT.
func (c *Client) ListSnapshots(ctx context.Context, node string, vmid int, vmType string) ([]SnapshotInfo, error) {
	path := fmt.Sprintf("/nodes/%s/%s/%d/snapshot", node, vmType, vmid)
	data, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, fmt.Errorf("list snapshots for vm %d: %w", vmid, err)
	}
	var snapshots []SnapshotInfo
	if err := json.Unmarshal(data, &snapshots); err != nil {
		return nil, fmt.Errorf("unmarshal snapshots: %w", err)
	}
	return snapshots, nil
}

// CreateSnapshot creates a snapshot for a VM/CT. Returns the task UPID.
func (c *Client) CreateSnapshot(ctx context.Context, node string, vmid int, vmType string, name string, description string, includeRAM bool) (string, error) {
	path := fmt.Sprintf("/nodes/%s/%s/%d/snapshot", node, vmType, vmid)
	params := url.Values{}
	params.Set("snapname", name)
	if description != "" {
		params.Set("description", description)
	}
	if includeRAM {
		params.Set("vmstate", "1")
	} else {
		params.Set("vmstate", "0")
	}
	data, err := c.doRequestWithBody(ctx, http.MethodPost, path, params)
	if err != nil {
		return "", fmt.Errorf("create snapshot for vm %d: %w", vmid, err)
	}
	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", fmt.Errorf("unmarshal snapshot upid: %w", err)
	}
	return upid, nil
}

// DeleteSnapshot deletes a snapshot for a VM/CT. Returns the task UPID.
func (c *Client) DeleteSnapshot(ctx context.Context, node string, vmid int, vmType string, snapname string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/%s/%d/snapshot/%s", node, vmType, vmid, url.PathEscape(snapname))
	data, err := c.doRequest(ctx, http.MethodDelete, path)
	if err != nil {
		return "", fmt.Errorf("delete snapshot %s for vm %d: %w", snapname, vmid, err)
	}
	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", fmt.Errorf("unmarshal delete snapshot upid: %w", err)
	}
	return upid, nil
}

// RollbackSnapshot rolls back a VM/CT to a snapshot. Returns the task UPID.
func (c *Client) RollbackSnapshot(ctx context.Context, node string, vmid int, vmType string, snapname string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/%s/%d/snapshot/%s/rollback", node, vmType, vmid, url.PathEscape(snapname))
	data, err := c.doRequestWithBody(ctx, http.MethodPost, path, nil)
	if err != nil {
		return "", fmt.Errorf("rollback snapshot %s for vm %d: %w", snapname, vmid, err)
	}
	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", fmt.Errorf("unmarshal rollback upid: %w", err)
	}
	return upid, nil
}

// GetVNCProxy creates a VNC proxy connection for a VM/CT.
func (c *Client) GetVNCProxy(ctx context.Context, node string, vmid int, vmType string) (*VNCProxyResponse, error) {
	path := fmt.Sprintf("/nodes/%s/%s/%d/vncproxy", node, vmType, vmid)
	params := url.Values{}
	params.Set("websocket", "1")
	data, err := c.doRequestWithBody(ctx, http.MethodPost, path, params)
	if err != nil {
		return nil, fmt.Errorf("get vnc proxy for vm %d: %w", vmid, err)
	}
	var resp VNCProxyResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal vnc proxy response: %w", err)
	}
	return &resp, nil
}

// GetGuestOSInfo retrieves OS information from the QEMU guest agent.
func (c *Client) GetGuestOSInfo(ctx context.Context, node string, vmid int) (*GuestOSInfo, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/agent/get-osinfo", node, vmid)
	data, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, fmt.Errorf("get guest os info for vm %d: %w", vmid, err)
	}
	inner := unwrapAgentResult(data)
	var info GuestOSInfo
	if err := json.Unmarshal(inner, &info); err != nil {
		return nil, fmt.Errorf("unmarshal guest os info: %w", err)
	}
	return &info, nil
}

// GetTermProxy creates a terminal proxy connection for a VM/CT (text-based, for xterm.js).
func (c *Client) GetTermProxy(ctx context.Context, node string, vmid int, vmType string) (*VNCProxyResponse, error) {
	path := fmt.Sprintf("/nodes/%s/%s/%d/termproxy", node, vmType, vmid)
	params := url.Values{}
	data, err := c.doRequestWithBody(ctx, http.MethodPost, path, params)
	if err != nil {
		return nil, fmt.Errorf("get term proxy for vm %d: %w", vmid, err)
	}
	var resp VNCProxyResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal term proxy response: %w", err)
	}
	return &resp, nil
}

// RestoreVM restores a VM from a vzdump archive. Returns the task UPID.
// vmType must be "qemu" or "lxc" to select the correct API endpoint.
func (c *Client) RestoreVM(ctx context.Context, node string, archive string, storage string, vmid int, vmType string) (string, error) {
	if err := validateVMType(vmType); err != nil {
		return "", err
	}
	if archive == "" {
		return "", fmt.Errorf("archive-Pfad darf nicht leer sein")
	}
	if storage == "" {
		return "", fmt.Errorf("target-Storage darf nicht leer sein")
	}
	if vmid <= 0 {
		return "", fmt.Errorf("ungültige VMID %d für Restore", vmid)
	}

	params := url.Values{}
	params.Set("archive", archive)
	params.Set("storage", storage)
	params.Set("vmid", fmt.Sprintf("%d", vmid))
	params.Set("force", "1")

	data, err := c.doRequestWithBody(ctx, http.MethodPost, fmt.Sprintf("/nodes/%s/%s", node, vmType), params)
	if err != nil {
		// Provide specific guidance for common restore failures
		errStr := err.Error()
		if strings.Contains(errStr, "already exists") {
			return "", fmt.Errorf("restore VM %d: VMID existiert bereits auf dem Ziel-Node. Bitte andere VMID verwenden oder bestehende VM entfernen: %w", vmid, err)
		}
		if strings.Contains(errStr, "storage") && strings.Contains(errStr, "not available") {
			return "", fmt.Errorf("restore VM %d: Zielspeicher '%s' ist nicht verfügbar auf Node '%s': %w", vmid, storage, node, err)
		}
		if strings.Contains(errStr, "No space left") {
			return "", fmt.Errorf("restore VM %d: Nicht genügend Speicherplatz auf '%s': %w", vmid, storage, err)
		}
		return "", fmt.Errorf("restore VM %d: %w", vmid, err)
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

