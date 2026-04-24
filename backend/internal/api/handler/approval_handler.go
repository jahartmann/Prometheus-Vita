package handler

import (
	"errors"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/api/middleware"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/agent"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type ApprovalHandler struct {
	approvalRepo repository.ApprovalRepository
	agentService *agent.Service
}

func NewApprovalHandler(approvalRepo repository.ApprovalRepository, agentSvc *agent.Service) *ApprovalHandler {
	return &ApprovalHandler{
		approvalRepo: approvalRepo,
		agentService: agentSvc,
	}
}

func (h *ApprovalHandler) ListPending(c echo.Context) error {
	userID, ok := c.Get(middleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		return apiPkg.Unauthorized(c, "user not found in context")
	}

	approvals, err := h.approvalRepo.ListPending(c.Request().Context(), userID)
	if err != nil {
		return apiPkg.InternalError(c, "Fehler beim Abrufen der Genehmigungen")
	}
	if approvals == nil {
		approvals = []model.AgentPendingApproval{}
	}
	return apiPkg.Success(c, approvals)
}

func (h *ApprovalHandler) Approve(c echo.Context) error {
	userID, ok := c.Get(middleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		return apiPkg.Unauthorized(c, "user not found in context")
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "Ungueltige Approval-ID")
	}

	approval, err := h.approvalRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "Genehmigung nicht gefunden")
		}
		return apiPkg.InternalError(c, "Fehler beim Abrufen der Genehmigung")
	}

	if approval.UserID != userID {
		return apiPkg.NotFound(c, "Genehmigung nicht gefunden")
	}

	// Atomically resolve as approved (WHERE status='pending' prevents double-approve)
	if err := h.approvalRepo.Resolve(c.Request().Context(), id, model.ApprovalApproved, userID); err != nil {
		if errors.Is(err, repository.ErrAlreadyResolved) {
			return apiPkg.BadRequest(c, "Genehmigung wurde bereits bearbeitet")
		}
		return apiPkg.InternalError(c, "Fehler beim Genehmigen")
	}

	result, err := h.agentService.ExecuteApprovedTool(c.Request().Context(), userID, approval)
	if err != nil {
		return apiPkg.Success(c, map[string]interface{}{
			"status": "approved",
			"error":  err.Error(),
		})
	}

	return apiPkg.Success(c, map[string]interface{}{
		"status": "approved",
		"result": result,
	})
}

func (h *ApprovalHandler) Reject(c echo.Context) error {
	userID, ok := c.Get(middleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		return apiPkg.Unauthorized(c, "user not found in context")
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "Ungueltige Approval-ID")
	}

	approval, err := h.approvalRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "Genehmigung nicht gefunden")
		}
		return apiPkg.InternalError(c, "Fehler beim Abrufen der Genehmigung")
	}

	if approval.UserID != userID {
		return apiPkg.NotFound(c, "Genehmigung nicht gefunden")
	}

	// Atomically resolve as rejected (WHERE status='pending' prevents double-reject)
	if err := h.approvalRepo.Resolve(c.Request().Context(), id, model.ApprovalRejected, userID); err != nil {
		if errors.Is(err, repository.ErrAlreadyResolved) {
			return apiPkg.BadRequest(c, "Genehmigung wurde bereits bearbeitet")
		}
		return apiPkg.InternalError(c, "Fehler beim Ablehnen")
	}

	return apiPkg.Success(c, map[string]string{"status": "rejected"})
}
