package handler

import (
	"errors"
	"strconv"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	nodeService "github.com/antigravity/prometheus/internal/service/node"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

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
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to get node")
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
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to update node")
	}

	return apiPkg.Success(c, node.ToResponse())
}

func (h *NodeHandler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	if err := h.service.Delete(c.Request().Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to delete node")
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
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to get node status")
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
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to get VMs")
	}

	return apiPkg.Success(c, vms)
}

func (h *NodeHandler) GetStorage(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	storage, err := h.service.GetStorage(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to get storage")
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
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to get network interfaces")
	}

	return apiPkg.Success(c, ifaces)
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
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to set network alias")
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
		return apiPkg.InternalError(c, "failed to onboard node: "+err.Error())
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
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to get disks")
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

	return id, vmid, vmType, nil
}

func (h *NodeHandler) StartVM(c echo.Context) error {
	id, vmid, vmType, err := parseVMParams(c)
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id or vmid")
	}

	upid, err := h.service.StartVM(c.Request().Context(), id, vmid, vmType)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to start VM")
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
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to stop VM")
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
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to shutdown VM")
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
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to suspend VM")
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
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to resume VM")
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
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to list snapshots")
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
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to create snapshot")
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
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to delete snapshot")
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
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to rollback snapshot")
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
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to get VNC proxy")
	}

	return apiPkg.Success(c, proxy)
}
