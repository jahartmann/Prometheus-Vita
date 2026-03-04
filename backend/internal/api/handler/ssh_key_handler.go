package handler

import (
	"errors"

	apiPkg "github.com/antigravity/prometheus/internal/api"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/sshkeys"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type SSHKeyHandler struct {
	sshkeySvc *sshkeys.Service
}

func NewSSHKeyHandler(sshkeySvc *sshkeys.Service) *SSHKeyHandler {
	return &SSHKeyHandler{sshkeySvc: sshkeySvc}
}

func (h *SSHKeyHandler) ListByNode(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}
	keys, err := h.sshkeySvc.ListByNode(c.Request().Context(), nodeID)
	if err != nil {
		return apiPkg.InternalError(c, "failed to list ssh keys")
	}
	return apiPkg.Success(c, keys)
}

func (h *SSHKeyHandler) Generate(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}
	var req model.CreateSSHKeyRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if req.Name == "" {
		return apiPkg.BadRequest(c, "name is required")
	}
	key, err := h.sshkeySvc.GenerateKeyPair(c.Request().Context(), nodeID, req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to generate ssh key")
	}
	return apiPkg.Created(c, key)
}

func (h *SSHKeyHandler) Deploy(c echo.Context) error {
	keyID, err := uuid.Parse(c.Param("keyId"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid key id")
	}
	if err := h.sshkeySvc.DeployKey(c.Request().Context(), keyID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "key not found")
		}
		return apiPkg.InternalError(c, "failed to deploy ssh key")
	}
	return apiPkg.Success(c, map[string]string{"status": "deployed"})
}

func (h *SSHKeyHandler) Rotate(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}
	key, err := h.sshkeySvc.RotateKey(c.Request().Context(), nodeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to rotate key")
	}
	return apiPkg.Success(c, key)
}

func (h *SSHKeyHandler) Delete(c echo.Context) error {
	keyID, err := uuid.Parse(c.Param("keyId"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid key id")
	}
	if err := h.sshkeySvc.Delete(c.Request().Context(), keyID); err != nil {
		return apiPkg.InternalError(c, "failed to delete ssh key")
	}
	return apiPkg.NoContent(c)
}

func (h *SSHKeyHandler) GetRotationSchedule(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}
	sched, err := h.sshkeySvc.GetRotationSchedule(c.Request().Context(), nodeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "rotation schedule not found")
		}
		return apiPkg.InternalError(c, "failed to get rotation schedule")
	}
	return apiPkg.Success(c, sched)
}

func (h *SSHKeyHandler) CreateRotationSchedule(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}
	var req model.CreateRotationScheduleRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if req.IntervalDays <= 0 {
		return apiPkg.BadRequest(c, "interval_days must be positive")
	}
	sched, err := h.sshkeySvc.CreateRotationSchedule(c.Request().Context(), nodeID, req)
	if err != nil {
		return apiPkg.InternalError(c, "failed to create rotation schedule")
	}
	return apiPkg.Created(c, sched)
}
