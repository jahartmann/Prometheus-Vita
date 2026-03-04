package handler

import (
	"errors"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/environment"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type EnvironmentHandler struct {
	envSvc *environment.Service
}

func NewEnvironmentHandler(envSvc *environment.Service) *EnvironmentHandler {
	return &EnvironmentHandler{envSvc: envSvc}
}

func (h *EnvironmentHandler) List(c echo.Context) error {
	envs, err := h.envSvc.List(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "failed to list environments")
	}
	return apiPkg.Success(c, envs)
}

func (h *EnvironmentHandler) Get(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid environment id")
	}
	env, err := h.envSvc.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "environment not found")
		}
		return apiPkg.InternalError(c, "failed to get environment")
	}
	return apiPkg.Success(c, env)
}

func (h *EnvironmentHandler) Create(c echo.Context) error {
	var req model.CreateEnvironmentRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if req.Name == "" {
		return apiPkg.BadRequest(c, "name is required")
	}
	env, err := h.envSvc.Create(c.Request().Context(), req)
	if err != nil {
		return apiPkg.InternalError(c, "failed to create environment")
	}
	return apiPkg.Created(c, env)
}

func (h *EnvironmentHandler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid environment id")
	}
	var req model.UpdateEnvironmentRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	env, err := h.envSvc.Update(c.Request().Context(), id, req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "environment not found")
		}
		return apiPkg.InternalError(c, "failed to update environment")
	}
	return apiPkg.Success(c, env)
}

func (h *EnvironmentHandler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid environment id")
	}
	if err := h.envSvc.Delete(c.Request().Context(), id); err != nil {
		return apiPkg.InternalError(c, "failed to delete environment")
	}
	return apiPkg.NoContent(c)
}

func (h *EnvironmentHandler) AssignNode(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}
	var req struct {
		EnvironmentID *uuid.UUID `json:"environment_id"`
	}
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if err := h.envSvc.AssignNode(c.Request().Context(), nodeID, req.EnvironmentID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node or environment not found")
		}
		return apiPkg.InternalError(c, "failed to assign node to environment")
	}
	return apiPkg.Success(c, map[string]string{"status": "assigned"})
}
