package handler

import (
	"errors"

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
