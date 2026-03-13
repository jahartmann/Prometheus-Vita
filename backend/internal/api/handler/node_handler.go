package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/apierror"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/proxmox"
	"github.com/antigravity/prometheus/internal/repository"
	nodeService "github.com/antigravity/prometheus/internal/service/node"
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

// commonScanPorts are well-known ports scanned from the node for QEMU VMs without a guest agent.
var commonScanPorts = []int{
	22, 25, 53, 80, 443, 993, 995, 3128, 3306, 3389,
	5432, 5900, 6379, 8006, 8080, 8443, 9090, 27017,
}

func (h *NodeHandler) GetPorts(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	ctx := c.Request().Context()

	// Get node info + VM list from Proxmox API
	node, err := h.service.GetByID(ctx, id)
	if err != nil {
		return handleNodeError(c, err, "failed to get node")
	}

	vms, _ := h.service.GetVMs(ctx, id)

	// Build a single SSH script that gathers all port data at once.
	// This avoids per-VM API calls and works without the guest agent.
	script := h.buildPortScanScript(node.Name, vms)
	sshResult, sshErr := h.service.RunSSHCommand(ctx, id, script)
	if sshErr != nil {
		slog.Warn("port scan SSH failed", slog.Any("error", sshErr))
		// Return empty result rather than error so the UI still renders
		return apiPkg.Success(c, map[string]interface{}{
			"groups":      []VMPortGroup{},
			"listening":   []NodePort{},
			"established": []NodePort{},
		})
	}

	// Parse structured output into groups
	groups := h.parsePortScanOutput(sshResult.Stdout, node.Name, vms)

	// Compute flat summaries for overview cards
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

// buildPortScanScript creates a bash script that runs on the Proxmox node
// and collects port data for the node itself, all LXC containers (via pct exec),
// and all QEMU VMs (via nmap/nc scan from the node using the ARP table).
func (h *NodeHandler) buildPortScanScript(nodeName string, vms []proxmox.VMInfo) string {
	var sb strings.Builder
	portList := joinInts(commonScanPorts)
	portListComma := strings.ReplaceAll(portList, " ", ",")

	// Node ports
	sb.WriteString("echo '###NODE:0###'\n")
	sb.WriteString("ss -tunap 2>/dev/null\n")

	for _, vm := range vms {
		if vm.Status != "running" {
			continue
		}
		if vm.Type == "lxc" {
			// LXC: pct exec always works, no guest agent needed
			fmt.Fprintf(&sb, "echo '###LXC:%d###'\n", vm.VMID)
			fmt.Fprintf(&sb, "timeout 5 pct exec %d -- sh -c 'ss -tunap 2>/dev/null || netstat -tunap 2>/dev/null' 2>/dev/null\n", vm.VMID)
		} else {
			// QEMU: find VM IP via MAC in ARP table, scan with nmap or nc
			fmt.Fprintf(&sb, "echo '###QEMU:%d###'\n", vm.VMID)
			fmt.Fprintf(&sb, `_mac=$(qm config %d 2>/dev/null | grep -oP '([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}' | head -1 | tr '[:upper:]' '[:lower:]')
if [ -n "$_mac" ]; then
  _ip=$(ip neigh 2>/dev/null | tr '[:upper:]' '[:lower:]' | grep "$_mac" | awk '$NF!="FAILED"{print $1}' | head -1)
  if [ -n "$_ip" ]; then
    if command -v nmap >/dev/null 2>&1; then
      timeout 10 nmap -sT -T4 -p %s --open "$_ip" 2>/dev/null | awk -v ip="$_ip" '/^[0-9]+\/tcp.*open/{split($1,a,"/"); print "tcp LISTEN 0 0 " ip ":" a[1] " 0.0.0.0:*"}'
    elif command -v nc >/dev/null 2>&1; then
      for _p in %s; do
        nc -z -w1 "$_ip" "$_p" 2>/dev/null && echo "tcp LISTEN 0 0 $_ip:$_p 0.0.0.0:*"
      done
    fi
  fi
fi
`, vm.VMID, portListComma, portList)
		}
	}

	return sb.String()
}

func joinInts(nums []int) string {
	parts := make([]string, len(nums))
	for i, n := range nums {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, " ")
}

// parsePortScanOutput parses the structured output from the SSH port scan script.
// Sections are delimited by ###TYPE:VMID### markers.
func (h *NodeHandler) parsePortScanOutput(output string, nodeName string, vms []proxmox.VMInfo) []VMPortGroup {
	// Build VM lookup by VMID
	vmLookup := map[int]proxmox.VMInfo{}
	for _, vm := range vms {
		vmLookup[vm.VMID] = vm
	}

	var groups []VMPortGroup
	lines := strings.Split(output, "\n")
	var currentType, currentContent string
	currentVMID := 0

	flushGroup := func() {
		if currentType == "" {
			return
		}
		ports := parseSSOutput(currentContent)
		name := nodeName
		vmType := currentType
		if currentType != "node" {
			if vm, ok := vmLookup[currentVMID]; ok {
				name = vm.Name
			} else {
				name = fmt.Sprintf("VM %d", currentVMID)
			}
			vmType = strings.ToLower(currentType)
		}
		groups = append(groups, VMPortGroup{
			VMID:  currentVMID,
			Name:  name,
			Type:  vmType,
			Ports: ports,
		})
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "###") && strings.HasSuffix(line, "###") {
			flushGroup()
			marker := strings.Trim(line, "#")
			parts := strings.SplitN(marker, ":", 2)
			currentType = parts[0]
			currentContent = ""
			if len(parts) >= 2 {
				currentVMID, _ = strconv.Atoi(parts[1])
			}
			continue
		}
		currentContent += line + "\n"
	}
	flushGroup()

	return groups
}

// parseSSOutput parses ss/netstat output into NodePort slice.
func parseSSOutput(output string) []NodePort {
	ports := []NodePort{}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		proto := strings.ToLower(fields[0])
		if proto != "tcp" && proto != "udp" {
			continue
		}
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
