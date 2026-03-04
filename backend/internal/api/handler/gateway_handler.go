package handler

import (
	"errors"
	"strconv"

	apiPkg "github.com/antigravity/prometheus/internal/api"
	"github.com/antigravity/prometheus/internal/api/middleware"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/gateway"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type GatewayHandler struct {
	gatewaySvc *gateway.Service
	auditRepo  repository.AuditRepository
}

func NewGatewayHandler(gatewaySvc *gateway.Service, auditRepo repository.AuditRepository) *GatewayHandler {
	return &GatewayHandler{gatewaySvc: gatewaySvc, auditRepo: auditRepo}
}

func (h *GatewayHandler) CreateToken(c echo.Context) error {
	userID, ok := c.Get(middleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		return apiPkg.Unauthorized(c, "user not found in context")
	}

	var req model.CreateAPITokenRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if req.Name == "" {
		return apiPkg.BadRequest(c, "name is required")
	}

	resp, err := h.gatewaySvc.CreateToken(c.Request().Context(), userID, req)
	if err != nil {
		return apiPkg.InternalError(c, "failed to create api token")
	}

	return apiPkg.Created(c, resp)
}

func (h *GatewayHandler) ListTokens(c echo.Context) error {
	userID, ok := c.Get(middleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		return apiPkg.Unauthorized(c, "user not found in context")
	}

	tokens, err := h.gatewaySvc.ListTokens(c.Request().Context(), userID)
	if err != nil {
		return apiPkg.InternalError(c, "failed to list tokens")
	}

	return apiPkg.Success(c, tokens)
}

func (h *GatewayHandler) RevokeToken(c echo.Context) error {
	tokenID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid token id")
	}

	if err := h.gatewaySvc.RevokeToken(c.Request().Context(), tokenID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "token not found")
		}
		return apiPkg.InternalError(c, "failed to revoke token")
	}

	return apiPkg.Success(c, map[string]string{"status": "revoked"})
}

func (h *GatewayHandler) DeleteToken(c echo.Context) error {
	tokenID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid token id")
	}

	if err := h.gatewaySvc.DeleteToken(c.Request().Context(), tokenID); err != nil {
		return apiPkg.InternalError(c, "failed to delete token")
	}

	return apiPkg.NoContent(c)
}

func (h *GatewayHandler) ListAuditLog(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	entries, err := h.auditRepo.List(c.Request().Context(), limit, offset)
	if err != nil {
		return apiPkg.InternalError(c, "failed to list audit log")
	}

	return apiPkg.Success(c, entries)
}
