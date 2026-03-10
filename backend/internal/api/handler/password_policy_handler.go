package handler

import (
	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/api/middleware"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type PasswordPolicyHandler struct {
	repo repository.PasswordPolicyRepository
}

func NewPasswordPolicyHandler(repo repository.PasswordPolicyRepository) *PasswordPolicyHandler {
	return &PasswordPolicyHandler{repo: repo}
}

func (h *PasswordPolicyHandler) Get(c echo.Context) error {
	policy, err := h.repo.Get(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "failed to get password policy")
	}
	return apiPkg.Success(c, policy)
}

func (h *PasswordPolicyHandler) Update(c echo.Context) error {
	var req model.UpdatePasswordPolicyRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	policy, err := h.repo.Get(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "failed to get password policy")
	}

	if req.MinLength != nil {
		if *req.MinLength < 1 || *req.MinLength > 128 {
			return apiPkg.BadRequest(c, "min_length must be between 1 and 128")
		}
		policy.MinLength = *req.MinLength
	}
	if req.MaxLength != nil {
		if *req.MaxLength < 1 || *req.MaxLength > 1024 {
			return apiPkg.BadRequest(c, "max_length must be between 1 and 1024")
		}
		policy.MaxLength = *req.MaxLength
	}
	if policy.MinLength > policy.MaxLength {
		return apiPkg.BadRequest(c, "min_length cannot exceed max_length")
	}
	if req.RequireUppercase != nil {
		policy.RequireUppercase = *req.RequireUppercase
	}
	if req.RequireLowercase != nil {
		policy.RequireLowercase = *req.RequireLowercase
	}
	if req.RequireDigit != nil {
		policy.RequireDigit = *req.RequireDigit
	}
	if req.RequireSpecial != nil {
		policy.RequireSpecial = *req.RequireSpecial
	}
	if req.DisallowUsername != nil {
		policy.DisallowUsername = *req.DisallowUsername
	}

	userID, _ := c.Get(middleware.ContextKeyUserID).(uuid.UUID)
	policy.UpdatedBy = &userID

	if err := h.repo.Update(c.Request().Context(), policy); err != nil {
		return apiPkg.InternalError(c, "failed to update password policy")
	}

	return apiPkg.Success(c, policy)
}
