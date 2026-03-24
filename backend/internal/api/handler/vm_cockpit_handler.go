package handler

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/antigravity/prometheus/internal/api/middleware"
	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/apierror"
	"github.com/antigravity/prometheus/internal/model"
	nodeService "github.com/antigravity/prometheus/internal/service/node"
	"github.com/antigravity/prometheus/internal/service/auth"
	vmService "github.com/antigravity/prometheus/internal/service/vm"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

// VMProcess represents a process running inside a VM.
type VMProcess struct {
	User    string  `json:"user"`
	PID     int     `json:"pid"`
	CPU     float64 `json:"cpu"`
	Mem     float64 `json:"mem"`
	VSZ     string  `json:"vsz"`
	RSS     string  `json:"rss"`
	Command string  `json:"command"`
}

// VMServiceInfo represents a systemd service.
type VMServiceInfo struct {
	Unit        string `json:"unit"`
	LoadState   string `json:"load_state"`
	ActiveState string `json:"active_state"`
	SubState    string `json:"sub_state"`
	Description string `json:"description"`
}

// VMPort represents a listening port.
type VMPort struct {
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
	Process  string `json:"process"`
}

// VMDisk represents disk usage for a mountpoint.
type VMDisk struct {
	Target  string `json:"target"`
	Size    string `json:"size"`
	Used    string `json:"used"`
	Avail   string `json:"avail"`
	Percent string `json:"percent"`
}

type VMCockpitHandler struct {
	nodeSvc        *nodeService.Service
	permSvc        *vmService.PermissionService
	jwtSvc         *auth.JWTService
	upgrader       websocket.Upgrader
}

func NewVMCockpitHandler(nodeSvc *nodeService.Service, permSvc *vmService.PermissionService, jwtSvc *auth.JWTService, allowedOrigins []string) *VMCockpitHandler {
	h := &VMCockpitHandler{
		nodeSvc: nodeSvc,
		permSvc: permSvc,
		jwtSvc:  jwtSvc,
	}
	h.upgrader = websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			if len(allowedOrigins) == 0 {
				return true
			}
			for _, allowed := range allowedOrigins {
				if allowed == "*" || allowed == origin {
					return true
				}
			}
			slog.Warn("vm cockpit ws origin rejected", slog.String("origin", origin))
			return false
		},
	}
	return h
}

// parseCockpitParams extracts nodeID, vmid, vmType, and userID from the request.
func parseCockpitParams(c echo.Context) (uuid.UUID, int, string, uuid.UUID, error) {
	// Support both :id (from nodes group) and :nodeId (from standalone routes)
	nodeIDStr := c.Param("id")
	if nodeIDStr == "" {
		nodeIDStr = c.Param("nodeId")
	}
	nodeID, err := uuid.Parse(nodeIDStr)
	if err != nil {
		return uuid.UUID{}, 0, "", uuid.UUID{}, fmt.Errorf("invalid node id")
	}

	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		return uuid.UUID{}, 0, "", uuid.UUID{}, fmt.Errorf("invalid vmid")
	}
	if vmid <= 0 {
		return uuid.UUID{}, 0, "", uuid.UUID{}, fmt.Errorf("vmid must be positive")
	}

	vmType := c.QueryParam("type")
	if vmType == "" {
		vmType = "lxc"
	}
	if vmType != "qemu" && vmType != "lxc" {
		return uuid.UUID{}, 0, "", uuid.UUID{}, fmt.Errorf("type must be 'qemu' or 'lxc'")
	}

	userID, ok := c.Get(middleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		return uuid.UUID{}, 0, "", uuid.UUID{}, fmt.Errorf("user not found in context")
	}

	return nodeID, vmid, vmType, userID, nil
}

func (h *VMCockpitHandler) ExecCommand(c echo.Context) error {
	nodeID, vmid, vmType, userID, err := parseCockpitParams(c)
	if err != nil {
		return apiPkg.BadRequest(c, err.Error())
	}

	allowed, err := h.permSvc.CheckPermission(c.Request().Context(), userID, nodeID, strconv.Itoa(vmid), model.PermVMShell)
	if err != nil {
		return apiPkg.InternalError(c, "failed to check permission")
	}
	if !allowed {
		return apiPkg.Forbidden(c, "insufficient VM permissions")
	}

	var req struct {
		Command []string `json:"command"`
	}
	if err := c.Bind(&req); err != nil || len(req.Command) == 0 {
		return apiPkg.BadRequest(c, "command is required (array of strings)")
	}

	for i, arg := range req.Command {
		if arg == "" {
			return apiPkg.BadRequest(c, fmt.Sprintf("command element %d is empty", i))
		}
	}

	result, err := h.nodeSvc.ExecVMCommand(c.Request().Context(), nodeID, vmid, vmType, req.Command)
	if err != nil {
		return handleNodeError(c, err, "failed to execute command")
	}

	slog.Info("vm cockpit operation",
		slog.String("op", "exec_command"),
		slog.String("node_id", nodeID.String()),
		slog.Int("vmid", vmid),
		slog.String("vm_type", vmType),
		slog.String("user_id", userID.String()),
	)
	return apiPkg.Success(c, result)
}

// GetOSInfo returns the detected OS family of a VM.
func (h *VMCockpitHandler) GetOSInfo(c echo.Context) error {
	nodeID, vmid, vmType, userID, err := parseCockpitParams(c)
	if err != nil {
		return apiPkg.BadRequest(c, err.Error())
	}
	allowed, err := h.permSvc.CheckPermission(c.Request().Context(), userID, nodeID, strconv.Itoa(vmid), model.PermVMSystemView)
	if err != nil {
		return apiPkg.InternalError(c, "failed to check permission")
	}
	if !allowed {
		return apiPkg.Forbidden(c, "insufficient VM permissions")
	}
	osFamily := h.nodeSvc.GetGuestOSFamily(c.Request().Context(), nodeID, vmid, vmType)
	return apiPkg.Success(c, map[string]string{"os_family": osFamily})
}

// detectOS is a helper that detects the OS for the current VM.
func (h *VMCockpitHandler) detectOS(c echo.Context, nodeID uuid.UUID, vmid int, vmType string) string {
	return h.nodeSvc.GetGuestOSFamily(c.Request().Context(), nodeID, vmid, vmType)
}

func (h *VMCockpitHandler) GetProcesses(c echo.Context) error {
	nodeID, vmid, vmType, userID, err := parseCockpitParams(c)
	if err != nil {
		return apiPkg.BadRequest(c, err.Error())
	}

	allowed, err := h.permSvc.CheckPermission(c.Request().Context(), userID, nodeID, strconv.Itoa(vmid), model.PermVMSystemView)
	if err != nil {
		return apiPkg.InternalError(c, "failed to check permission")
	}
	if !allowed {
		return apiPkg.Forbidden(c, "insufficient VM permissions")
	}

	osFamily := h.detectOS(c, nodeID, vmid, vmType)
	var cmd []string
	if osFamily == "windows" {
		cmd = []string{"powershell", "-Command", "Get-Process | Sort-Object CPU -Descending | Select-Object -First 50 Id,ProcessName,CPU,@{N='Mem';E={[math]::Round($_.WorkingSet64/1MB,1)}} | Format-Table -AutoSize | Out-String -Width 200"}
	} else {
		cmd = []string{"ps", "aux", "--sort=-pcpu"}
	}

	result, err := h.nodeSvc.ExecVMCommand(c.Request().Context(), nodeID, vmid, vmType, cmd)
	if err != nil {
		return handleNodeError(c, err, "failed to get processes")
	}

	var response interface{}
	if osFamily == "windows" {
		response = parseWindowsProcesses(result.OutData)
	} else {
		response = parseProcesses(result.OutData)
	}
	slog.Info("vm cockpit operation",
		slog.String("op", "get_processes"),
		slog.String("node_id", nodeID.String()),
		slog.Int("vmid", vmid),
		slog.String("os_family", osFamily),
	)
	return apiPkg.Success(c, response)
}

func (h *VMCockpitHandler) GetServices(c echo.Context) error {
	nodeID, vmid, vmType, userID, err := parseCockpitParams(c)
	if err != nil {
		return apiPkg.BadRequest(c, err.Error())
	}

	allowed, err := h.permSvc.CheckPermission(c.Request().Context(), userID, nodeID, strconv.Itoa(vmid), model.PermVMSystemView)
	if err != nil {
		return apiPkg.InternalError(c, "failed to check permission")
	}
	if !allowed {
		return apiPkg.Forbidden(c, "insufficient VM permissions")
	}

	osFamily := h.detectOS(c, nodeID, vmid, vmType)
	var cmd []string
	if osFamily == "windows" {
		cmd = []string{"powershell", "-Command", "Get-Service | Select-Object Name,Status,DisplayName | Format-Table -AutoSize | Out-String -Width 300"}
	} else {
		cmd = []string{"systemctl", "list-units", "--type=service", "--all", "--no-pager", "--plain"}
	}

	result, err := h.nodeSvc.ExecVMCommand(c.Request().Context(), nodeID, vmid, vmType, cmd)
	if err != nil {
		return handleNodeError(c, err, "failed to get services")
	}

	var response interface{}
	if osFamily == "windows" {
		response = parseWindowsServices(result.OutData)
	} else {
		response = parseServices(result.OutData)
	}
	slog.Info("vm cockpit operation",
		slog.String("op", "get_services"),
		slog.String("node_id", nodeID.String()),
		slog.Int("vmid", vmid),
		slog.String("os_family", osFamily),
	)
	return apiPkg.Success(c, response)
}

func (h *VMCockpitHandler) GetPorts(c echo.Context) error {
	nodeID, vmid, vmType, userID, err := parseCockpitParams(c)
	if err != nil {
		return apiPkg.BadRequest(c, err.Error())
	}

	allowed, err := h.permSvc.CheckPermission(c.Request().Context(), userID, nodeID, strconv.Itoa(vmid), model.PermVMSystemView)
	if err != nil {
		return apiPkg.InternalError(c, "failed to check permission")
	}
	if !allowed {
		return apiPkg.Forbidden(c, "insufficient VM permissions")
	}

	osFamily := h.detectOS(c, nodeID, vmid, vmType)
	var cmd []string
	if osFamily == "windows" {
		cmd = []string{"netstat", "-ano"}
	} else {
		cmd = []string{"ss", "-tlnp"}
	}

	result, err := h.nodeSvc.ExecVMCommand(c.Request().Context(), nodeID, vmid, vmType, cmd)
	if err != nil {
		return handleNodeError(c, err, "failed to get ports")
	}

	var response interface{}
	if osFamily == "windows" {
		response = parseWindowsPorts(result.OutData)
	} else {
		response = parsePorts(result.OutData)
	}
	slog.Info("vm cockpit operation",
		slog.String("op", "get_ports"),
		slog.String("node_id", nodeID.String()),
		slog.Int("vmid", vmid),
		slog.String("os_family", osFamily),
	)
	return apiPkg.Success(c, response)
}

func (h *VMCockpitHandler) GetDiskUsage(c echo.Context) error {
	nodeID, vmid, vmType, userID, err := parseCockpitParams(c)
	if err != nil {
		return apiPkg.BadRequest(c, err.Error())
	}

	allowed, err := h.permSvc.CheckPermission(c.Request().Context(), userID, nodeID, strconv.Itoa(vmid), model.PermVMSystemView)
	if err != nil {
		return apiPkg.InternalError(c, "failed to check permission")
	}
	if !allowed {
		return apiPkg.Forbidden(c, "insufficient VM permissions")
	}

	osFamily := h.detectOS(c, nodeID, vmid, vmType)
	var cmd []string
	if osFamily == "windows" {
		cmd = []string{"powershell", "-Command", "Get-PSDrive -PSProvider FileSystem | Select-Object Name,@{N='Size';E={[math]::Round($_.Used/1GB+$_.Free/1GB,1)}},@{N='Used';E={[math]::Round($_.Used/1GB,1)}},@{N='Free';E={[math]::Round($_.Free/1GB,1)}},@{N='Pct';E={if(($_.Used+$_.Free) -gt 0){[math]::Round($_.Used/($_.Used+$_.Free)*100)}else{0}}} | Format-Table -AutoSize | Out-String -Width 200"}
	} else {
		cmd = []string{"df", "-h", "--output=target,size,used,avail,pcent"}
	}

	result, err := h.nodeSvc.ExecVMCommand(c.Request().Context(), nodeID, vmid, vmType, cmd)
	if err != nil {
		return handleNodeError(c, err, "failed to get disk usage")
	}

	var response interface{}
	if osFamily == "windows" {
		response = parseWindowsDisk(result.OutData)
	} else {
		response = parseDisk(result.OutData)
	}
	slog.Info("vm cockpit operation",
		slog.String("op", "get_disk_usage"),
		slog.String("node_id", nodeID.String()),
		slog.Int("vmid", vmid),
		slog.String("os_family", osFamily),
	)
	return apiPkg.Success(c, response)
}

func (h *VMCockpitHandler) ServiceAction(c echo.Context) error {
	nodeID, vmid, vmType, userID, err := parseCockpitParams(c)
	if err != nil {
		return apiPkg.BadRequest(c, err.Error())
	}

	allowed, err := h.permSvc.CheckPermission(c.Request().Context(), userID, nodeID, strconv.Itoa(vmid), model.PermVMSystemService)
	if err != nil {
		return apiPkg.InternalError(c, "failed to check permission")
	}
	if !allowed {
		return apiPkg.Forbidden(c, "insufficient VM permissions")
	}

	var req struct {
		Service string `json:"service"`
		Action  string `json:"action"`
	}
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if req.Service == "" || req.Action == "" {
		return apiPkg.BadRequest(c, "service and action are required")
	}

	validActions := map[string]bool{"start": true, "stop": true, "restart": true, "enable": true, "disable": true}
	if !validActions[req.Action] {
		return apiPkg.BadRequest(c, "action must be start, stop, restart, enable, or disable")
	}

	osFamily := h.detectOS(c, nodeID, vmid, vmType)
	var cmd []string
	if osFamily == "windows" {
		// Windows: use sc or net command
		switch req.Action {
		case "start":
			cmd = []string{"net", "start", req.Service}
		case "stop":
			cmd = []string{"net", "stop", req.Service}
		case "restart":
			cmd = []string{"powershell", "-Command", fmt.Sprintf("Restart-Service -Name '%s' -Force", req.Service)}
		default:
			return apiPkg.BadRequest(c, "enable/disable not supported on Windows via this interface")
		}
	} else {
		cmd = []string{"systemctl", req.Action, req.Service}
	}

	result, err := h.nodeSvc.ExecVMCommand(c.Request().Context(), nodeID, vmid, vmType, cmd)
	if err != nil {
		return handleNodeError(c, err, "failed to execute service action")
	}

	slog.Info("vm cockpit operation",
		slog.String("op", "service_action"),
		slog.String("node_id", nodeID.String()),
		slog.Int("vmid", vmid),
		slog.String("vm_type", vmType),
		slog.String("user_id", userID.String()),
	)
	return apiPkg.Success(c, result)
}

func (h *VMCockpitHandler) KillProcess(c echo.Context) error {
	nodeID, vmid, vmType, userID, err := parseCockpitParams(c)
	if err != nil {
		return apiPkg.BadRequest(c, err.Error())
	}

	allowed, err := h.permSvc.CheckPermission(c.Request().Context(), userID, nodeID, strconv.Itoa(vmid), model.PermVMSystemKill)
	if err != nil {
		return apiPkg.InternalError(c, "failed to check permission")
	}
	if !allowed {
		return apiPkg.Forbidden(c, "insufficient VM permissions")
	}

	var req struct {
		PID    int    `json:"pid"`
		Signal string `json:"signal"`
	}
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if req.PID <= 0 {
		return apiPkg.BadRequest(c, "valid pid is required")
	}
	if req.Signal == "" {
		req.Signal = "TERM"
	}
	if req.Signal != "TERM" && req.Signal != "KILL" {
		return apiPkg.BadRequest(c, "signal must be TERM or KILL")
	}

	osFamily := h.detectOS(c, nodeID, vmid, vmType)
	var killCmd []string
	if osFamily == "windows" {
		if req.Signal == "KILL" {
			killCmd = []string{"taskkill", "/PID", strconv.Itoa(req.PID), "/F"}
		} else {
			killCmd = []string{"taskkill", "/PID", strconv.Itoa(req.PID)}
		}
	} else {
		killCmd = []string{"kill", fmt.Sprintf("-%s", req.Signal), strconv.Itoa(req.PID)}
	}

	result, err := h.nodeSvc.ExecVMCommand(c.Request().Context(), nodeID, vmid, vmType, killCmd)
	if err != nil {
		return handleNodeError(c, err, "failed to kill process")
	}

	slog.Info("vm cockpit operation",
		slog.String("op", "kill_process"),
		slog.String("node_id", nodeID.String()),
		slog.Int("vmid", vmid),
		slog.String("vm_type", vmType),
		slog.String("user_id", userID.String()),
	)
	return apiPkg.Success(c, result)
}

// HandleShell proxies a WebSocket connection between the browser and a Proxmox VNC terminal.
func (h *VMCockpitHandler) HandleShell(c echo.Context) error {
	// Authenticate via query parameter
	token := c.QueryParam("token")
	if token == "" {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 {
				token = parts[1]
			}
		}
	}
	if token == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "token required"})
	}

	claims, err := h.jwtSvc.ValidateAccessToken(token)
	if err != nil {
		slog.Warn("vm shell ws auth failed", slog.Any("error", err))
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	nodeID, err := uuid.Parse(c.Param("nodeId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid node id"})
	}

	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid vmid"})
	}
	if vmid <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "vmid must be positive"})
	}

	vmType := c.QueryParam("type")
	if vmType == "" {
		vmType = "lxc"
	}

	// Check permission
	allowed, err := h.permSvc.CheckPermission(c.Request().Context(), claims.UserID, nodeID, strconv.Itoa(vmid), model.PermVMShell)
	if err != nil || !allowed {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "insufficient VM permissions"})
	}

	// Get terminal proxy ticket from Proxmox (text-based terminal, not VNC)
	termProxy, err := h.nodeSvc.GetTermProxy(c.Request().Context(), nodeID, vmid, vmType)
	if err != nil {
		slog.Error("failed to get terminal proxy", slog.Any("error", err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create terminal session"})
	}

	// Get node info for constructing the Proxmox WebSocket URL
	node, err := h.nodeSvc.GetByID(c.Request().Context(), nodeID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get node"})
	}

	// Construct Proxmox terminal WebSocket URL (uses vncwebsocket endpoint)
	pveWSURL := fmt.Sprintf("wss://%s:%d/api2/json/nodes/%s/%s/%d/vncwebsocket?port=%s&vncticket=%s",
		node.Hostname, node.Port, node.Name, vmType, vmid,
		url.QueryEscape(termProxy.Port), url.QueryEscape(termProxy.Ticket))

	// Upgrade browser connection
	browserConn, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error("vm shell ws upgrade failed", slog.Any("error", err))
		return err
	}
	defer browserConn.Close()

	// Connect to Proxmox terminal WebSocket
	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	pveHeaders := http.Header{}
	pveHeaders.Set("Cookie", fmt.Sprintf("PVEAuthCookie=%s", termProxy.Ticket))

	pveConn, _, err := dialer.Dial(pveWSURL, pveHeaders)
	if err != nil {
		slog.Error("failed to connect to Proxmox terminal WebSocket", slog.Any("error", err))
		browserConn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "failed to connect to terminal"))
		return nil
	}
	defer pveConn.Close()

	// Bidirectional proxy with Proxmox terminal protocol translation.
	// Proxmox termproxy protocol:
	//   Browser → PVE: "0:<length>:<data>" for stdin, "1:<cols>:<rows>:" for resize
	//   PVE → Browser: raw terminal output text
	done := make(chan struct{})

	// Browser -> Proxmox: wrap xterm.js input in Proxmox framing
	go func() {
		defer func() { done <- struct{}{} }()
		defer func() {
			if r := recover(); r != nil {
				slog.Error("terminal browser->pve goroutine panicked",
					slog.Any("panic", r),
					slog.String("stack", string(debug.Stack())),
				)
			}
		}()
		for {
			_, msg, err := browserConn.ReadMessage()
			if err != nil {
				return
			}
			// Frame as Proxmox stdin: "0:<length>:<data>"
			framed := fmt.Sprintf("0:%d:%s", len(msg), string(msg))
			if err := pveConn.WriteMessage(websocket.TextMessage, []byte(framed)); err != nil {
				return
			}
		}
	}()

	// Proxmox -> Browser: forward terminal output directly
	go func() {
		defer func() { done <- struct{}{} }()
		defer func() {
			if r := recover(); r != nil {
				slog.Error("terminal pve->browser goroutine panicked",
					slog.Any("panic", r),
					slog.String("stack", string(debug.Stack())),
				)
			}
		}()
		for {
			_, msg, err := pveConn.ReadMessage()
			if err != nil {
				return
			}
			if err := browserConn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}()

	// Wait for either direction to close
	<-done
	return nil
}

// parseProcesses parses the output of `ps aux --sort=-pcpu`.
func parseProcesses(output string) []VMProcess {
	var processes []VMProcess
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return processes
	}

	for _, line := range lines[1:] { // skip header
		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}
		cpu, _ := strconv.ParseFloat(fields[2], 64)
		mem, _ := strconv.ParseFloat(fields[3], 64)
		pid, _ := strconv.Atoi(fields[1])

		processes = append(processes, VMProcess{
			User:    fields[0],
			PID:     pid,
			CPU:     cpu,
			Mem:     mem,
			VSZ:     fields[4],
			RSS:     fields[5],
			Command: strings.Join(fields[10:], " "),
		})
	}
	return processes
}

// parseServices parses the output of `systemctl list-units --type=service --all --no-pager --plain`.
func parseServices(output string) []VMServiceInfo {
	var services []VMServiceInfo
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "UNIT") {
			continue
		}
		// Stop at the summary line
		if strings.HasPrefix(line, "LOAD") || strings.Contains(line, "loaded units listed") {
			break
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		description := ""
		if len(fields) > 4 {
			description = strings.Join(fields[4:], " ")
		}

		services = append(services, VMServiceInfo{
			Unit:        fields[0],
			LoadState:   fields[1],
			ActiveState: fields[2],
			SubState:    fields[3],
			Description: description,
		})
	}
	return services
}

// parsePorts parses the output of `ss -tlnp`.
func parsePorts(output string) []VMPort {
	var ports []VMPort
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return ports
	}

	for _, line := range lines[1:] { // skip header
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		localAddr := fields[3]
		// Parse address:port
		lastColon := strings.LastIndex(localAddr, ":")
		if lastColon < 0 {
			continue
		}
		addr := localAddr[:lastColon]
		portNum, err := strconv.Atoi(localAddr[lastColon+1:])
		if err != nil {
			continue
		}

		process := ""
		if len(fields) >= 6 {
			process = fields[5]
		}

		ports = append(ports, VMPort{
			Protocol: fields[0],
			Address:  addr,
			Port:     portNum,
			Process:  process,
		})
	}
	return ports
}

// parseDisk parses the output of `df -h --output=target,size,used,avail,pcent`.
func parseDisk(output string) []VMDisk {
	var disks []VMDisk
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return disks
	}

	for _, line := range lines[1:] { // skip header
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		disks = append(disks, VMDisk{
			Target:  fields[0],
			Size:    fields[1],
			Used:    fields[2],
			Avail:   fields[3],
			Percent: fields[4],
		})
	}
	return disks
}

// --- Windows Parsers ---

// parseWindowsProcesses parses PowerShell Get-Process output.
func parseWindowsProcesses(output string) []VMProcess {
	var processes []VMProcess
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Id") || strings.HasPrefix(line, "--") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		pid, _ := strconv.Atoi(fields[0])
		name := fields[1]
		var cpu float64
		if len(fields) >= 3 {
			// CPU might have comma as decimal separator
			cpuStr := strings.ReplaceAll(fields[2], ",", ".")
			fmt.Sscanf(cpuStr, "%f", &cpu)
		}
		var mem float64
		if len(fields) >= 4 {
			memStr := strings.ReplaceAll(fields[3], ",", ".")
			fmt.Sscanf(memStr, "%f", &mem)
		}

		processes = append(processes, VMProcess{
			User:    "-",
			PID:     pid,
			CPU:     cpu,
			Mem:     mem,
			Command: name,
		})
	}
	return processes
}

// parseWindowsServices parses PowerShell Get-Service output.
func parseWindowsServices(output string) []VMServiceInfo {
	var services []VMServiceInfo
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Name") || strings.HasPrefix(line, "----") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		name := fields[0]
		status := fields[1]
		displayName := ""
		if len(fields) > 2 {
			displayName = strings.Join(fields[2:], " ")
		}

		activeState := "inactive"
		subState := "dead"
		if strings.ToLower(status) == "running" {
			activeState = "active"
			subState = "running"
		} else if strings.ToLower(status) == "stopped" {
			activeState = "inactive"
			subState = "dead"
		}

		services = append(services, VMServiceInfo{
			Unit:        name,
			LoadState:   "loaded",
			ActiveState: activeState,
			SubState:    subState,
			Description: displayName,
		})
	}
	return services
}

// parseWindowsPorts parses `netstat -ano` output.
func parseWindowsPorts(output string) []VMPort {
	var ports []VMPort
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

		// Only show LISTENING and ESTABLISHED
		if state != "LISTENING" && state != "ESTABLISHED" && state != "LISTEN" {
			continue
		}

		addr, portNum := splitNetstatAddr(localAddr)

		ports = append(ports, VMPort{
			Protocol: proto,
			Address:  addr,
			Port:     portNum,
			Process:  pid,
		})
	}
	return ports
}

func splitNetstatAddr(addr string) (string, int) {
	lastColon := strings.LastIndex(addr, ":")
	if lastColon < 0 {
		return addr, 0
	}
	host := addr[:lastColon]
	port, _ := strconv.Atoi(addr[lastColon+1:])
	return host, port
}

// parseWindowsDisk parses PowerShell Get-PSDrive output.
func parseWindowsDisk(output string) []VMDisk {
	var disks []VMDisk
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Name") || strings.HasPrefix(line, "----") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		name := fields[0] + ":\\"
		sizeStr := strings.ReplaceAll(fields[1], ",", ".")
		usedStr := strings.ReplaceAll(fields[2], ",", ".")
		freeStr := strings.ReplaceAll(fields[3], ",", ".")
		pctStr := strings.ReplaceAll(fields[4], ",", ".")

		disks = append(disks, VMDisk{
			Target:  name,
			Size:    sizeStr + "G",
			Used:    usedStr + "G",
			Avail:   freeStr + "G",
			Percent: pctStr + "%",
		})
	}
	return disks
}

// --- Files Tab Endpoints (Phase 2) ---

// ListFiles handles GET /nodes/:nodeId/vms/:vmid/cockpit/files
func (h *VMCockpitHandler) ListFiles(c echo.Context) error {
	nodeID, vmid, vmType, userID, err := parseCockpitParams(c)
	if err != nil {
		return apiPkg.BadRequest(c, err.Error())
	}
	allowed, err := h.permSvc.CheckPermission(c.Request().Context(), userID, nodeID, strconv.Itoa(vmid), model.PermVMFilesRead)
	if err != nil {
		return apiPkg.InternalError(c, "Berechtigungspruefung fehlgeschlagen")
	}
	if !allowed {
		return apiPkg.Forbidden(c, "Keine Berechtigung fuer Dateizugriff")
	}
	rawPath := c.QueryParam("path")
	if rawPath == "" {
		rawPath = "/"
	}
	path, pathErr := apierror.ValidatePath(rawPath)
	if pathErr != nil {
		return apiPkg.FromAPIError(c, pathErr.(*apierror.APIError))
	}
	entries, err := h.nodeSvc.ListVMDirectory(c.Request().Context(), nodeID, vmid, vmType, path)
	if err != nil {
		return handleNodeError(c, err, "Verzeichnis konnte nicht gelesen werden")
	}
	slog.Info("vm cockpit operation",
		slog.String("op", "list_files"),
		slog.String("node_id", nodeID.String()),
		slog.Int("vmid", vmid),
		slog.String("vm_type", vmType),
		slog.String("user_id", userID.String()),
	)
	return apiPkg.Success(c, entries)
}

// ReadFile handles GET /nodes/:nodeId/vms/:vmid/cockpit/files/read
func (h *VMCockpitHandler) ReadFile(c echo.Context) error {
	nodeID, vmid, vmType, userID, err := parseCockpitParams(c)
	if err != nil {
		return apiPkg.BadRequest(c, err.Error())
	}
	allowed, err := h.permSvc.CheckPermission(c.Request().Context(), userID, nodeID, strconv.Itoa(vmid), model.PermVMFilesRead)
	if err != nil {
		return apiPkg.InternalError(c, "Berechtigungspruefung fehlgeschlagen")
	}
	if !allowed {
		return apiPkg.Forbidden(c, "Keine Berechtigung fuer Dateizugriff")
	}
	rawPath := c.QueryParam("path")
	if rawPath == "" {
		return apiPkg.BadRequest(c, "path is required")
	}
	path, pathErr := apierror.ValidatePath(rawPath)
	if pathErr != nil {
		return apiPkg.FromAPIError(c, pathErr.(*apierror.APIError))
	}
	content, err := h.nodeSvc.ReadVMFile(c.Request().Context(), nodeID, vmid, vmType, path)
	if err != nil {
		return handleNodeError(c, err, "Datei konnte nicht gelesen werden")
	}
	slog.Info("vm cockpit operation",
		slog.String("op", "read_file"),
		slog.String("node_id", nodeID.String()),
		slog.Int("vmid", vmid),
		slog.String("vm_type", vmType),
		slog.String("user_id", userID.String()),
	)
	return apiPkg.Success(c, map[string]string{"path": path, "content": content})
}

// WriteFile handles POST /nodes/:nodeId/vms/:vmid/cockpit/files/write
func (h *VMCockpitHandler) WriteFile(c echo.Context) error {
	nodeID, vmid, vmType, userID, err := parseCockpitParams(c)
	if err != nil {
		return apiPkg.BadRequest(c, err.Error())
	}
	allowed, err := h.permSvc.CheckPermission(c.Request().Context(), userID, nodeID, strconv.Itoa(vmid), model.PermVMFilesWrite)
	if err != nil {
		return apiPkg.InternalError(c, "Berechtigungspruefung fehlgeschlagen")
	}
	if !allowed {
		return apiPkg.Forbidden(c, "Keine Schreibberechtigung fuer Dateien")
	}
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := c.Bind(&req); err != nil || req.Path == "" {
		return apiPkg.BadRequest(c, "path and content are required")
	}
	validPath, pathErr := apierror.ValidatePath(req.Path)
	if pathErr != nil {
		return apiPkg.FromAPIError(c, pathErr.(*apierror.APIError))
	}
	req.Path = validPath
	if err := h.nodeSvc.WriteVMFile(c.Request().Context(), nodeID, vmid, vmType, req.Path, req.Content); err != nil {
		return handleNodeError(c, err, "Datei konnte nicht geschrieben werden")
	}
	slog.Info("vm cockpit operation",
		slog.String("op", "write_file"),
		slog.String("node_id", nodeID.String()),
		slog.Int("vmid", vmid),
		slog.String("vm_type", vmType),
		slog.String("user_id", userID.String()),
	)
	return apiPkg.Success(c, map[string]string{"path": req.Path, "message": "Datei erfolgreich gespeichert"})
}

// UploadFile handles POST /nodes/:nodeId/vms/:vmid/cockpit/files/upload
func (h *VMCockpitHandler) UploadFile(c echo.Context) error {
	nodeID, vmid, vmType, userID, err := parseCockpitParams(c)
	if err != nil {
		return apiPkg.BadRequest(c, err.Error())
	}
	allowed, err := h.permSvc.CheckPermission(c.Request().Context(), userID, nodeID, strconv.Itoa(vmid), model.PermVMFilesWrite)
	if err != nil {
		return apiPkg.InternalError(c, "Berechtigungspruefung fehlgeschlagen")
	}
	if !allowed {
		return apiPkg.Forbidden(c, "Keine Schreibberechtigung fuer Dateien")
	}
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := c.Bind(&req); err != nil || req.Path == "" {
		return apiPkg.BadRequest(c, "path and content are required")
	}
	validPath, pathErr := apierror.ValidatePath(req.Path)
	if pathErr != nil {
		return apiPkg.FromAPIError(c, pathErr.(*apierror.APIError))
	}
	req.Path = validPath
	if err := h.nodeSvc.WriteVMFile(c.Request().Context(), nodeID, vmid, vmType, req.Path, req.Content); err != nil {
		return handleNodeError(c, err, "Datei konnte nicht hochgeladen werden")
	}
	slog.Info("vm cockpit operation",
		slog.String("op", "upload_file"),
		slog.String("node_id", nodeID.String()),
		slog.Int("vmid", vmid),
		slog.String("vm_type", vmType),
		slog.String("user_id", userID.String()),
	)
	return apiPkg.Success(c, map[string]string{"path": req.Path, "message": "Datei erfolgreich hochgeladen"})
}

// DeleteFile handles DELETE /nodes/:nodeId/vms/:vmid/cockpit/files
func (h *VMCockpitHandler) DeleteFile(c echo.Context) error {
	nodeID, vmid, vmType, userID, err := parseCockpitParams(c)
	if err != nil {
		return apiPkg.BadRequest(c, err.Error())
	}
	allowed, err := h.permSvc.CheckPermission(c.Request().Context(), userID, nodeID, strconv.Itoa(vmid), model.PermVMFilesWrite)
	if err != nil {
		return apiPkg.InternalError(c, "Berechtigungspruefung fehlgeschlagen")
	}
	if !allowed {
		return apiPkg.Forbidden(c, "Keine Schreibberechtigung fuer Dateien")
	}
	rawPath := c.QueryParam("path")
	if rawPath == "" {
		return apiPkg.BadRequest(c, "path is required")
	}
	path, pathErr := apierror.ValidatePath(rawPath)
	if pathErr != nil {
		return apiPkg.FromAPIError(c, pathErr.(*apierror.APIError))
	}
	if err := h.nodeSvc.DeleteVMFile(c.Request().Context(), nodeID, vmid, vmType, path); err != nil {
		return handleNodeError(c, err, "Datei konnte nicht geloescht werden")
	}
	slog.Info("vm cockpit operation",
		slog.String("op", "delete_file"),
		slog.String("node_id", nodeID.String()),
		slog.Int("vmid", vmid),
		slog.String("vm_type", vmType),
		slog.String("user_id", userID.String()),
	)
	return apiPkg.Success(c, map[string]string{"path": path, "message": "Datei erfolgreich geloescht"})
}

// MakeDir handles POST /nodes/:nodeId/vms/:vmid/cockpit/files/mkdir
func (h *VMCockpitHandler) MakeDir(c echo.Context) error {
	nodeID, vmid, vmType, userID, err := parseCockpitParams(c)
	if err != nil {
		return apiPkg.BadRequest(c, err.Error())
	}
	allowed, err := h.permSvc.CheckPermission(c.Request().Context(), userID, nodeID, strconv.Itoa(vmid), model.PermVMFilesWrite)
	if err != nil {
		return apiPkg.InternalError(c, "Berechtigungspruefung fehlgeschlagen")
	}
	if !allowed {
		return apiPkg.Forbidden(c, "Keine Schreibberechtigung fuer Dateien")
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := c.Bind(&req); err != nil || req.Path == "" {
		return apiPkg.BadRequest(c, "path is required")
	}
	validPath, pathErr := apierror.ValidatePath(req.Path)
	if pathErr != nil {
		return apiPkg.FromAPIError(c, pathErr.(*apierror.APIError))
	}
	req.Path = validPath
	if err := h.nodeSvc.MakeVMDirectory(c.Request().Context(), nodeID, vmid, vmType, req.Path); err != nil {
		return handleNodeError(c, err, "Verzeichnis konnte nicht erstellt werden")
	}
	slog.Info("vm cockpit operation",
		slog.String("op", "make_dir"),
		slog.String("node_id", nodeID.String()),
		slog.Int("vmid", vmid),
		slog.String("vm_type", vmType),
		slog.String("user_id", userID.String()),
	)
	return apiPkg.Success(c, map[string]string{"path": req.Path, "message": "Verzeichnis erfolgreich erstellt"})
}
