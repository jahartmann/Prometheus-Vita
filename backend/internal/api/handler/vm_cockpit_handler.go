package handler

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/antigravity/prometheus/internal/api/middleware"
	apiPkg "github.com/antigravity/prometheus/internal/api/response"
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

	return apiPkg.Success(c, result)
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

	result, err := h.nodeSvc.ExecVMCommand(c.Request().Context(), nodeID, vmid, vmType, []string{"ps", "aux", "--sort=-pcpu"})
	if err != nil {
		return handleNodeError(c, err, "failed to get processes")
	}

	processes := parseProcesses(result.OutData)
	return apiPkg.Success(c, processes)
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

	result, err := h.nodeSvc.ExecVMCommand(c.Request().Context(), nodeID, vmid, vmType,
		[]string{"systemctl", "list-units", "--type=service", "--all", "--no-pager", "--plain"})
	if err != nil {
		return handleNodeError(c, err, "failed to get services")
	}

	services := parseServices(result.OutData)
	return apiPkg.Success(c, services)
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

	result, err := h.nodeSvc.ExecVMCommand(c.Request().Context(), nodeID, vmid, vmType, []string{"ss", "-tlnp"})
	if err != nil {
		return handleNodeError(c, err, "failed to get ports")
	}

	ports := parsePorts(result.OutData)
	return apiPkg.Success(c, ports)
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

	result, err := h.nodeSvc.ExecVMCommand(c.Request().Context(), nodeID, vmid, vmType,
		[]string{"df", "-h", "--output=target,size,used,avail,pcent"})
	if err != nil {
		return handleNodeError(c, err, "failed to get disk usage")
	}

	disks := parseDisk(result.OutData)
	return apiPkg.Success(c, disks)
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

	result, err := h.nodeSvc.ExecVMCommand(c.Request().Context(), nodeID, vmid, vmType,
		[]string{"systemctl", req.Action, req.Service})
	if err != nil {
		return handleNodeError(c, err, "failed to execute service action")
	}

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

	result, err := h.nodeSvc.ExecVMCommand(c.Request().Context(), nodeID, vmid, vmType,
		[]string{"kill", fmt.Sprintf("-%s", req.Signal), strconv.Itoa(req.PID)})
	if err != nil {
		return handleNodeError(c, err, "failed to kill process")
	}

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

	vmType := c.QueryParam("type")
	if vmType == "" {
		vmType = "lxc"
	}

	// Check permission
	allowed, err := h.permSvc.CheckPermission(c.Request().Context(), claims.UserID, nodeID, strconv.Itoa(vmid), model.PermVMShell)
	if err != nil || !allowed {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "insufficient VM permissions"})
	}

	// Get VNC proxy ticket from Proxmox
	vncProxy, err := h.nodeSvc.GetVNCProxy(c.Request().Context(), nodeID, vmid, vmType)
	if err != nil {
		slog.Error("failed to get VNC proxy", slog.Any("error", err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create terminal session"})
	}

	// Get node info for constructing the Proxmox WebSocket URL
	node, err := h.nodeSvc.GetByID(c.Request().Context(), nodeID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get node"})
	}

	// Construct Proxmox VNC WebSocket URL
	pveWSURL := fmt.Sprintf("wss://%s:%d/api2/json/nodes/%s/%s/%d/vncwebsocket?port=%s&vncticket=%s",
		node.Hostname, node.Port, node.Name, vmType, vmid,
		url.QueryEscape(vncProxy.Port), url.QueryEscape(vncProxy.Ticket))

	// Upgrade browser connection
	browserConn, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error("vm shell ws upgrade failed", slog.Any("error", err))
		return err
	}
	defer browserConn.Close()

	// Connect to Proxmox VNC WebSocket
	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	pveHeaders := http.Header{}
	pveHeaders.Set("Cookie", fmt.Sprintf("PVEAuthCookie=%s", vncProxy.Ticket))

	pveConn, _, err := dialer.Dial(pveWSURL, pveHeaders)
	if err != nil {
		slog.Error("failed to connect to Proxmox VNC WebSocket", slog.Any("error", err))
		browserConn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "failed to connect to terminal"))
		return nil
	}
	defer pveConn.Close()

	// Bidirectional proxy
	done := make(chan struct{})

	// Browser -> Proxmox
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			msgType, msg, err := browserConn.ReadMessage()
			if err != nil {
				return
			}
			if err := pveConn.WriteMessage(msgType, msg); err != nil {
				return
			}
		}
	}()

	// Proxmox -> Browser
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			msgType, msg, err := pveConn.ReadMessage()
			if err != nil {
				return
			}
			if err := browserConn.WriteMessage(msgType, msg); err != nil {
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
	path := c.QueryParam("path")
	if path == "" {
		path = "/"
	}
	entries, err := h.nodeSvc.ListVMDirectory(c.Request().Context(), nodeID, vmid, vmType, path)
	if err != nil {
		return handleNodeError(c, err, "Verzeichnis konnte nicht gelesen werden")
	}
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
	path := c.QueryParam("path")
	if path == "" {
		return apiPkg.BadRequest(c, "path is required")
	}
	content, err := h.nodeSvc.ReadVMFile(c.Request().Context(), nodeID, vmid, vmType, path)
	if err != nil {
		return handleNodeError(c, err, "Datei konnte nicht gelesen werden")
	}
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
	if err := h.nodeSvc.WriteVMFile(c.Request().Context(), nodeID, vmid, vmType, req.Path, req.Content); err != nil {
		return handleNodeError(c, err, "Datei konnte nicht geschrieben werden")
	}
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
	if err := h.nodeSvc.WriteVMFile(c.Request().Context(), nodeID, vmid, vmType, req.Path, req.Content); err != nil {
		return handleNodeError(c, err, "Datei konnte nicht hochgeladen werden")
	}
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
	path := c.QueryParam("path")
	if path == "" {
		return apiPkg.BadRequest(c, "path is required")
	}
	if err := h.nodeSvc.DeleteVMFile(c.Request().Context(), nodeID, vmid, vmType, path); err != nil {
		return handleNodeError(c, err, "Datei konnte nicht geloescht werden")
	}
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
	if err := h.nodeSvc.MakeVMDirectory(c.Request().Context(), nodeID, vmid, vmType, req.Path); err != nil {
		return handleNodeError(c, err, "Verzeichnis konnte nicht erstellt werden")
	}
	return apiPkg.Success(c, map[string]string{"path": req.Path, "message": "Verzeichnis erfolgreich erstellt"})
}
