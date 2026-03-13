package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/apierror"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/proxmox"
	"github.com/antigravity/prometheus/internal/repository"
	nodeService "github.com/antigravity/prometheus/internal/service/node"
	"github.com/antigravity/prometheus/internal/ssh"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// handleNodeError maps service-layer errors to the appropriate HTTP response.
func handleNodeError(c echo.Context, err error, fallbackMsg string) error {
	// Check for structured APIError first
	var apiErr *apierror.APIError
	if errors.As(err, &apiErr) {
		slog.Warn("vm cockpit error",
			slog.String("path", c.Path()),
			slog.String("error_code", string(apiErr.Code)),
			slog.String("error", apiErr.Error()),
		)
		return apiPkg.FromAPIError(c, apiErr)
	}

	if errors.Is(err, repository.ErrNotFound) {
		return apiPkg.NotFound(c, "node not found")
	}
	if errors.Is(err, nodeService.ErrNodeUnreachable) {
		nodeID := c.Param("id")
		slog.Warn("node unreachable",
			slog.String("path", c.Path()),
			slog.String("node_id", nodeID),
			slog.String("error_detail", err.Error()))
		return apiPkg.FromAPIError(c, apierror.NodeUnreachable(err))
	}

	// Detect common error patterns from unstructured errors
	errMsg := err.Error()
	if strings.Contains(errMsg, "timeout") {
		return apiPkg.FromAPIError(c, apierror.VMCommandTimeout(err))
	}
	if strings.Contains(errMsg, "agent") || strings.Contains(errMsg, "QEMU guest agent") {
		return apiPkg.FromAPIError(c, apierror.GuestAgentUnavailable(err))
	}
	if strings.Contains(errMsg, "ssh") || strings.Contains(errMsg, "SSH") {
		return apiPkg.FromAPIError(c, apierror.NodeSSHFailed(err))
	}

	slog.Error("node handler error", slog.String("path", c.Path()), slog.Any("error", err))
	return apiPkg.FromAPIError(c, apierror.VMExecFailed(err))
}

type NodeHandler struct {
	service *nodeService.Service
}

func NewNodeHandler(service *nodeService.Service) *NodeHandler {
	return &NodeHandler{service: service}
}

func (h *NodeHandler) Create(c echo.Context) error {
	var req model.CreateNodeRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if req.Name == "" || req.Hostname == "" || req.APITokenID == "" || req.APITokenSecret == "" {
		return apiPkg.BadRequest(c, "name, hostname, api_token_id, and api_token_secret are required")
	}
	if !req.Type.IsValid() {
		return apiPkg.BadRequest(c, "type must be 'pve' or 'pbs'")
	}

	node, err := h.service.Create(c.Request().Context(), req)
	if err != nil {
		return apiPkg.InternalError(c, "failed to create node")
	}

	return apiPkg.Created(c, node.ToResponse())
}

func (h *NodeHandler) List(c echo.Context) error {
	nodes, err := h.service.List(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "failed to list nodes")
	}

	responses := make([]model.NodeResponse, 0, len(nodes))
	for _, n := range nodes {
		responses = append(responses, n.ToResponse())
	}

	return apiPkg.Success(c, responses)
}

func (h *NodeHandler) Get(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	node, err := h.service.GetByID(c.Request().Context(), id)
	if err != nil {
		return handleNodeError(c, err, "failed to get node")
	}

	return apiPkg.Success(c, node.ToResponse())
}

func (h *NodeHandler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	var req model.UpdateNodeRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	node, err := h.service.Update(c.Request().Context(), id, req)
	if err != nil {
		return handleNodeError(c, err, "failed to update node")
	}

	return apiPkg.Success(c, node.ToResponse())
}

func (h *NodeHandler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	if err := h.service.Delete(c.Request().Context(), id); err != nil {
		return handleNodeError(c, err, "failed to delete node")
	}

	return apiPkg.NoContent(c)
}

func (h *NodeHandler) GetStatus(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	status, err := h.service.GetStatus(c.Request().Context(), id)
	if err != nil {
		return handleNodeError(c, err, "failed to get node status")
	}

	return apiPkg.Success(c, status)
}

func (h *NodeHandler) GetVMs(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	vms, err := h.service.GetVMs(c.Request().Context(), id)
	if err != nil {
		return handleNodeError(c, err, "failed to get VMs")
	}

	responses := make([]proxmox.VMResponse, 0, len(vms))
	for _, vm := range vms {
		responses = append(responses, vm.ToResponse())
	}

	return apiPkg.Success(c, responses)
}

func (h *NodeHandler) GetStorage(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	slog.Info("handler GetStorage called", slog.String("node_id", id.String()))

	storage, err := h.service.GetStorage(c.Request().Context(), id)
	if err != nil {
		// Include full error details for debugging connectivity issues
		slog.Error("handler GetStorage failed",
			slog.String("node_id", id.String()),
			slog.Any("error", err))
		return handleNodeError(c, err, fmt.Sprintf("failed to get storage: %v", err))
	}

	slog.Info("handler GetStorage returning",
		slog.String("node_id", id.String()),
		slog.Int("storage_count", len(storage)))

	return apiPkg.Success(c, storage)
}

func (h *NodeHandler) GetClusterStorage(c echo.Context) error {
	storage, err := h.service.GetClusterStorage(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, fmt.Sprintf("failed to get cluster storage: %v", err))
	}

	return apiPkg.Success(c, storage)
}

func (h *NodeHandler) TestConnection(c echo.Context) error {
	var req model.TestConnectionRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if req.Hostname == "" || req.APITokenID == "" || req.APITokenSecret == "" {
		return apiPkg.BadRequest(c, "hostname, api_token_id, and api_token_secret are required")
	}
	if !req.Type.IsValid() {
		return apiPkg.BadRequest(c, "type must be 'pve' or 'pbs'")
	}

	result := h.service.TestConnection(c.Request().Context(), req)
	return apiPkg.Success(c, result)
}

func (h *NodeHandler) GetNetworkInterfaces(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	ifaces, err := h.service.GetNetworkInterfaces(c.Request().Context(), id)
	if err != nil {
		return handleNodeError(c, err, "failed to get network interfaces")
	}

	return apiPkg.Success(c, ifaces)
}

// NodePort represents a port on the node (listening, established, etc.)
type NodePort struct {
	Protocol  string `json:"protocol"`
	State     string `json:"state"`
	LocalAddr string `json:"local_address"`
	LocalPort int    `json:"local_port"`
	PeerAddr  string `json:"peer_address,omitempty"`
	PeerPort  int    `json:"peer_port,omitempty"`
	Process   string `json:"process,omitempty"`
}

// VMPortGroup contains ports for a single VM or the node itself.
type VMPortGroup struct {
	VMID   int        `json:"vmid"`
	Name   string     `json:"name"`
	Type   string     `json:"type"` // "node", "qemu", "lxc"
	Status string     `json:"status,omitempty"`
	Ports  []NodePort `json:"ports"`
}

func (h *NodeHandler) GetPorts(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	ctx := c.Request().Context()

	// Get node info for the name
	node, err := h.service.GetByID(ctx, id)
	if err != nil {
		return handleNodeError(c, err, "failed to get node")
	}

	// Fetch node ports via SSH
	nodeResult, nodeErr := h.service.RunSSHCommand(ctx, id, "ss -tunap 2>/dev/null || ss -tuna 2>/dev/null")
	nodePorts := parseNodePortsFlat(nodeResult, nodeErr)

	// Build grouped response
	groups := []VMPortGroup{{
		VMID:  0,
		Name:  node.Name,
		Type:  "node",
		Ports: nodePorts,
	}}

	// Fetch running VMs and get their ports in parallel with a timeout
	vms, err := h.service.GetVMs(ctx, id)
	if err == nil {
		var runningVMs []proxmox.VM
		for _, vm := range vms {
			if vm.Status == "running" {
				runningVMs = append(runningVMs, vm)
			}
		}

		if len(runningVMs) > 0 {
			vmGroups := make([]VMPortGroup, len(runningVMs))
			var wg sync.WaitGroup
			// Per-VM timeout so one stuck VM doesn't block everything
			vmCtx, vmCancel := context.WithTimeout(ctx, 10*time.Second)
			defer vmCancel()

			for i, vm := range runningVMs {
				wg.Add(1)
				go func(idx int, v proxmox.VM) {
					defer wg.Done()
					vmPorts := h.fetchVMPorts(vmCtx, id, v.VMID, v.Type)
					vmGroups[idx] = VMPortGroup{
						VMID:   v.VMID,
						Name:   v.Name,
						Type:   v.Type,
						Status: v.Status,
						Ports:  vmPorts,
					}
				}(i, vm)
			}
			wg.Wait()
			groups = append(groups, vmGroups...)
		}
	}

	// Also compute the flat summary for the overview cards
	allListening := []NodePort{}
	allEstablished := []NodePort{}
	for _, g := range groups {
		for _, p := range g.Ports {
			switch strings.ToUpper(p.State) {
			case "LISTEN", "LISTENING":
				allListening = append(allListening, p)
			case "ESTAB", "ESTABLISHED":
				allEstablished = append(allEstablished, p)
			}
		}
	}

	return apiPkg.Success(c, map[string]interface{}{
		"groups":      groups,
		"listening":   allListening,
		"established": allEstablished,
	})
}

// fetchVMPorts gets ports from a single VM via exec.
func (h *NodeHandler) fetchVMPorts(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string) []NodePort {
	osFamily := h.service.GetGuestOSFamily(ctx, nodeID, vmid, vmType)
	var cmd []string
	if osFamily == "windows" {
		cmd = []string{"netstat", "-ano"}
	} else {
		// Default to Linux commands for both "linux" and "unknown"
		cmd = []string{"sh", "-c", "ss -tunap 2>/dev/null || netstat -tunap 2>/dev/null"}
	}

	result, err := h.service.ExecVMCommand(ctx, nodeID, vmid, vmType, cmd)
	if err != nil {
		slog.Debug("could not fetch VM ports",
			slog.Int("vmid", vmid), slog.String("type", vmType),
			slog.Any("error", err))
		return nil
	}

	if osFamily == "windows" {
		return parseWindowsNetstatToNodePorts(result.OutData)
	}
	return parseSSOutputToNodePorts(result.OutData)
}

// parseNodePortsFlat parses ss output into a flat list of NodePort.
func parseNodePortsFlat(result *ssh.CommandResult, err error) []NodePort {
	if err != nil || result == nil {
		return nil
	}
	var ports []NodePort
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	if len(lines) < 2 {
		return ports
	}
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		proto := fields[0]
		state := fields[1]
		localAddr := fields[4]
		peerAddr := ""
		if len(fields) >= 6 {
			peerAddr = fields[5]
		}
		process := ""
		if len(fields) >= 7 {
			process = fields[6]
		}
		lAddr, lPort := splitAddrPort(localAddr)
		pAddr, pPort := splitAddrPort(peerAddr)
		if state == "UNCONN" && lPort == 0 {
			continue
		}
		ports = append(ports, NodePort{
			Protocol:  proto,
			State:     state,
			LocalAddr: lAddr,
			LocalPort: lPort,
			PeerAddr:  pAddr,
			PeerPort:  pPort,
			Process:   cleanProcessName(process),
		})
	}
	return ports
}

// parseSSOutputToNodePorts parses VM ss output to NodePort slice.
func parseSSOutputToNodePorts(output string) []NodePort {
	var ports []NodePort
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return ports
	}
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		proto := fields[0]
		state := fields[1]
		localAddr := fields[4]
		peerAddr := ""
		if len(fields) >= 6 {
			peerAddr = fields[5]
		}
		process := ""
		if len(fields) >= 7 {
			process = fields[6]
		}
		lAddr, lPort := splitAddrPort(localAddr)
		pAddr, pPort := splitAddrPort(peerAddr)
		if state == "UNCONN" && lPort == 0 {
			continue
		}
		ports = append(ports, NodePort{
			Protocol:  proto,
			State:     state,
			LocalAddr: lAddr,
			LocalPort: lPort,
			PeerAddr:  pAddr,
			PeerPort:  pPort,
			Process:   cleanProcessName(process),
		})
	}
	return ports
}

// parseWindowsNetstatToNodePorts parses Windows netstat -ano to NodePort.
func parseWindowsNetstatToNodePorts(output string) []NodePort {
	var ports []NodePort
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Active") || strings.HasPrefix(line, "Proto") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		proto := strings.ToLower(fields[0])
		localAddr := fields[1]
		state := ""
		pid := ""
		if proto == "tcp" && len(fields) >= 5 {
			state = fields[3]
			pid = fields[4]
		} else if proto == "udp" && len(fields) >= 4 {
			state = "LISTEN"
			pid = fields[3]
		}
		lAddr, lPort := splitAddrPort(localAddr)
		ports = append(ports, NodePort{
			Protocol:  proto,
			State:     state,
			LocalAddr: lAddr,
			LocalPort: lPort,
			Process:   pid,
		})
	}
	return ports
}

// parseNodePorts parses `ss -tunap` output into structured port data.
func splitAddrPort(addr string) (string, int) {
	if addr == "" || addr == "*:*" {
		return "*", 0
	}
	lastColon := strings.LastIndex(addr, ":")
	if lastColon < 0 {
		return addr, 0
	}
	host := addr[:lastColon]
	portNum, _ := strconv.Atoi(addr[lastColon+1:])
	return host, portNum
}

func cleanProcessName(raw string) string {
	// ss outputs: users:(("sshd",pid=1234,fd=3))
	if raw == "" {
		return ""
	}
	raw = strings.TrimPrefix(raw, "users:((")
	raw = strings.TrimSuffix(raw, "))")
	parts := strings.Split(raw, ",")
	if len(parts) > 0 {
		return strings.Trim(parts[0], "\"")
	}
	return raw
}

func (h *NodeHandler) SetNetworkAlias(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	iface := c.Param("iface")
	if iface == "" {
		return apiPkg.BadRequest(c, "interface name is required")
	}

	var req model.UpsertAliasRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if req.DisplayName == "" {
		return apiPkg.BadRequest(c, "display_name is required")
	}

	if err := h.service.SetAlias(c.Request().Context(), id, iface, req); err != nil {
		return handleNodeError(c, err, "failed to set network alias")
	}

	return apiPkg.Success(c, map[string]string{"status": "ok"})
}

func (h *NodeHandler) Onboard(c echo.Context) error {
	var req model.OnboardNodeRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if req.Name == "" || req.Hostname == "" || req.Password == "" {
		return apiPkg.BadRequest(c, "name, hostname, and password are required")
	}
	if !req.Type.IsValid() {
		return apiPkg.BadRequest(c, "type must be 'pve' or 'pbs'")
	}

	node, err := h.service.Onboard(c.Request().Context(), req)
	if err != nil {
		slog.Error("failed to onboard node", slog.Any("error", err))
		return apiPkg.InternalError(c, err.Error())
	}

	return apiPkg.Created(c, node.ToResponse())
}

func (h *NodeHandler) GetDisks(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	disks, err := h.service.GetDisks(c.Request().Context(), id)
	if err != nil {
		return handleNodeError(c, err, "failed to get disks")
	}

	return apiPkg.Success(c, disks)
}

// parseVMParams extracts node ID, vmid, and vmType from the request.
func parseVMParams(c echo.Context) (uuid.UUID, int, string, error) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.UUID{}, 0, "", err
	}

	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		return uuid.UUID{}, 0, "", err
	}

	vmType := c.QueryParam("type")
	if vmType == "" {
		vmType = "qemu"
	}
	if vmType != "qemu" && vmType != "lxc" {
		return uuid.UUID{}, 0, "", fmt.Errorf("type must be 'qemu' or 'lxc'")
	}

	return id, vmid, vmType, nil
}

func (h *NodeHandler) StartVM(c echo.Context) error {
	id, vmid, vmType, err := parseVMParams(c)
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id or vmid")
	}

	upid, err := h.service.StartVM(c.Request().Context(), id, vmid, vmType)
	if err != nil {
		return handleNodeError(c, err, "failed to start VM")
	}

	return apiPkg.Success(c, map[string]string{"upid": upid})
}

func (h *NodeHandler) StopVM(c echo.Context) error {
	id, vmid, vmType, err := parseVMParams(c)
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id or vmid")
	}

	upid, err := h.service.StopVM(c.Request().Context(), id, vmid, vmType)
	if err != nil {
		return handleNodeError(c, err, "failed to stop VM")
	}

	return apiPkg.Success(c, map[string]string{"upid": upid})
}

func (h *NodeHandler) ShutdownVM(c echo.Context) error {
	id, vmid, vmType, err := parseVMParams(c)
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id or vmid")
	}

	upid, err := h.service.ShutdownVM(c.Request().Context(), id, vmid, vmType)
	if err != nil {
		return handleNodeError(c, err, "failed to shutdown VM")
	}

	return apiPkg.Success(c, map[string]string{"upid": upid})
}

func (h *NodeHandler) SuspendVM(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid vmid")
	}

	upid, err := h.service.SuspendVM(c.Request().Context(), id, vmid)
	if err != nil {
		return handleNodeError(c, err, "failed to suspend VM")
	}

	return apiPkg.Success(c, map[string]string{"upid": upid})
}

func (h *NodeHandler) ResumeVM(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid vmid")
	}

	upid, err := h.service.ResumeVM(c.Request().Context(), id, vmid)
	if err != nil {
		return handleNodeError(c, err, "failed to resume VM")
	}

	return apiPkg.Success(c, map[string]string{"upid": upid})
}

func (h *NodeHandler) ListSnapshots(c echo.Context) error {
	id, vmid, vmType, err := parseVMParams(c)
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id or vmid")
	}

	snapshots, err := h.service.ListSnapshots(c.Request().Context(), id, vmid, vmType)
	if err != nil {
		return handleNodeError(c, err, "failed to list snapshots")
	}

	return apiPkg.Success(c, snapshots)
}

func (h *NodeHandler) CreateSnapshot(c echo.Context) error {
	id, vmid, vmType, err := parseVMParams(c)
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id or vmid")
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		IncludeRAM  bool   `json:"include_ram"`
	}
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if req.Name == "" {
		return apiPkg.BadRequest(c, "snapshot name is required")
	}

	upid, err := h.service.CreateSnapshot(c.Request().Context(), id, vmid, vmType, req.Name, req.Description, req.IncludeRAM)
	if err != nil {
		return handleNodeError(c, err, "failed to create snapshot")
	}

	return apiPkg.Success(c, map[string]string{"upid": upid})
}

func (h *NodeHandler) DeleteSnapshot(c echo.Context) error {
	id, vmid, vmType, err := parseVMParams(c)
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id or vmid")
	}

	snapname := c.Param("snapname")
	if snapname == "" {
		return apiPkg.BadRequest(c, "snapshot name is required")
	}

	upid, err := h.service.DeleteSnapshot(c.Request().Context(), id, vmid, vmType, snapname)
	if err != nil {
		return handleNodeError(c, err, "failed to delete snapshot")
	}

	return apiPkg.Success(c, map[string]string{"upid": upid})
}

func (h *NodeHandler) RollbackSnapshot(c echo.Context) error {
	id, vmid, vmType, err := parseVMParams(c)
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id or vmid")
	}

	snapname := c.Param("snapname")
	if snapname == "" {
		return apiPkg.BadRequest(c, "snapshot name is required")
	}

	upid, err := h.service.RollbackSnapshot(c.Request().Context(), id, vmid, vmType, snapname)
	if err != nil {
		return handleNodeError(c, err, "failed to rollback snapshot")
	}

	return apiPkg.Success(c, map[string]string{"upid": upid})
}

func (h *NodeHandler) BulkVMAction(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	var req model.BulkVMRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if len(req.VMIDs) == 0 {
		return apiPkg.BadRequest(c, "vmids is required")
	}
	if req.Action != "start" && req.Action != "stop" && req.Action != "shutdown" {
		return apiPkg.BadRequest(c, "action must be 'start', 'stop', or 'shutdown'")
	}

	results, err := h.service.BulkVMAction(c.Request().Context(), id, req)
	if err != nil {
		return handleNodeError(c, err, "failed to execute bulk VM action")
	}

	return apiPkg.Success(c, results)
}

func (h *NodeHandler) SyncTags(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	count, err := h.service.SyncTagsFromProxmox(c.Request().Context(), id)
	if err != nil {
		return handleNodeError(c, err, "failed to sync tags")
	}

	return apiPkg.Success(c, map[string]int{"imported": count})
}

func (h *NodeHandler) DebugStorage(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	result, err := h.service.DebugStorage(c.Request().Context(), id)
	if err != nil {
		return handleNodeError(c, err, "failed to debug storage")
	}

	return apiPkg.Success(c, result)
}

func (h *NodeHandler) DiagnoseConnectivity(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	result := h.service.DiagnoseConnectivity(c.Request().Context(), id)
	return apiPkg.Success(c, result)
}

func (h *NodeHandler) ListISOs(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	isos, err := h.service.ListISOs(c.Request().Context(), id)
	if err != nil {
		return handleNodeError(c, err, "failed to list ISOs")
	}

	return apiPkg.Success(c, isos)
}

func (h *NodeHandler) ListTemplates(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	templates, err := h.service.ListTemplates(c.Request().Context(), id)
	if err != nil {
		return handleNodeError(c, err, "failed to list templates")
	}

	return apiPkg.Success(c, templates)
}

func (h *NodeHandler) SyncContent(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	var req nodeService.SyncContentRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if req.SourceNodeID == "" || req.Volid == "" {
		return apiPkg.BadRequest(c, "source_node_id and volid are required")
	}

	upid, err := h.service.SyncContent(c.Request().Context(), id, req)
	if err != nil {
		return handleNodeError(c, err, "failed to sync content")
	}

	return apiPkg.Success(c, map[string]string{"upid": upid})
}

func (h *NodeHandler) GetVNCProxy(c echo.Context) error {
	id, vmid, vmType, err := parseVMParams(c)
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id or vmid")
	}

	proxy, err := h.service.GetVNCProxy(c.Request().Context(), id, vmid, vmType)
	if err != nil {
		return handleNodeError(c, err, "failed to get VNC proxy")
	}

	return apiPkg.Success(c, proxy)
}
