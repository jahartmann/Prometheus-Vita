package handler

import (
	"errors"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/api/middleware"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	userService "github.com/antigravity/prometheus/internal/service/user"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type UserHandler struct {
	service *userService.Service
}

func NewUserHandler(service *userService.Service) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) List(c echo.Context) error {
	users, err := h.service.List(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "failed to list users")
	}
	return apiPkg.Success(c, users)
}

func (h *UserHandler) GetByID(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid user id")
	}

	user, err := h.service.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "user not found")
		}
		return apiPkg.InternalError(c, "failed to get user")
	}
	return apiPkg.Success(c, user)
}

func (h *UserHandler) Create(c echo.Context) error {
	var req model.CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if req.Username == "" || req.Password == "" {
		return apiPkg.BadRequest(c, "username and password are required")
	}
	if !req.Role.IsValid() {
		return apiPkg.BadRequest(c, "role must be 'admin', 'operator', or 'viewer'")
	}

	user, err := h.service.Create(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, userService.ErrUsernameTaken) {
			return apiPkg.BadRequest(c, "username already taken")
		}
		return apiPkg.InternalError(c, "failed to create user")
	}
	return apiPkg.Created(c, user)
}

func (h *UserHandler) ListInvitations(c echo.Context) error {
	invitations, err := h.service.ListInvitations(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "failed to list invitations")
	}
	return apiPkg.Success(c, invitations)
}

func (h *UserHandler) CreateInvitation(c echo.Context) error {
	var req model.CreateUserInvitationRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if req.Username == "" {
		return apiPkg.BadRequest(c, "username is required")
	}
	if !req.Role.IsValid() {
		return apiPkg.BadRequest(c, "role must be 'admin', 'operator', or 'viewer'")
	}
	currentUserID, ok := c.Get(middleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		return apiPkg.Unauthorized(c, "user not found in context")
	}
	resp, err := h.service.CreateInvitation(c.Request().Context(), req, currentUserID)
	if err != nil {
		if errors.Is(err, userService.ErrUsernameTaken) {
			return apiPkg.BadRequest(c, "username already taken")
		}
		return apiPkg.InternalError(c, "failed to create invitation")
	}
	return apiPkg.Created(c, resp)
}

func (h *UserHandler) DeleteInvitation(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid invitation id")
	}
	if err := h.service.DeleteInvitation(c.Request().Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "invitation not found")
		}
		return apiPkg.InternalError(c, "failed to delete invitation")
	}
	return apiPkg.NoContent(c)
}

func (h *UserHandler) AcceptInvitation(c echo.Context) error {
	var req model.AcceptUserInvitationRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	user, err := h.service.AcceptInvitation(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, userService.ErrInvitationInvalid) {
			return apiPkg.BadRequest(c, "invitation is invalid")
		}
		if errors.Is(err, userService.ErrInvitationExpired) {
			return apiPkg.BadRequest(c, "invitation is expired")
		}
		if errors.Is(err, userService.ErrUsernameTaken) {
			return apiPkg.BadRequest(c, "username already taken")
		}
		return apiPkg.InternalError(c, "failed to accept invitation")
	}
	return apiPkg.Created(c, user)
}

func (h *UserHandler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid user id")
	}

	var req model.UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	user, err := h.service.Update(c.Request().Context(), id, req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "user not found")
		}
		if errors.Is(err, userService.ErrUsernameTaken) {
			return apiPkg.BadRequest(c, "username already taken")
		}
		return apiPkg.InternalError(c, "failed to update user")
	}
	return apiPkg.Success(c, user)
}

func (h *UserHandler) ListSessions(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid user id")
	}
	sessions, err := h.service.ListSessions(c.Request().Context(), id)
	if err != nil {
		return apiPkg.InternalError(c, "failed to list sessions")
	}
	return apiPkg.Success(c, sessions)
}

func (h *UserHandler) RevokeSession(c echo.Context) error {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid user id")
	}
	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid session id")
	}
	if err := h.service.RevokeSession(c.Request().Context(), userID, sessionID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "session not found")
		}
		return apiPkg.InternalError(c, "failed to revoke session")
	}
	return apiPkg.Success(c, map[string]string{"status": "revoked"})
}

func (h *UserHandler) RevokeAllAccess(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid user id")
	}
	if err := h.service.RevokeAllAccess(c.Request().Context(), id); err != nil {
		return apiPkg.InternalError(c, "failed to revoke access")
	}
	return apiPkg.Success(c, map[string]string{"status": "revoked"})
}

func (h *UserHandler) ListAPITokens(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid user id")
	}
	tokens, err := h.service.ListAPITokens(c.Request().Context(), id)
	if err != nil {
		return apiPkg.InternalError(c, "failed to list api tokens")
	}
	return apiPkg.Success(c, tokens)
}

func (h *UserHandler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid user id")
	}

	currentUserID, ok := c.Get(middleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		return apiPkg.Unauthorized(c, "user not found in context")
	}

	if err := h.service.Delete(c.Request().Context(), id, currentUserID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "user not found")
		}
		if errors.Is(err, userService.ErrSelfDelete) {
			return apiPkg.BadRequest(c, "cannot delete own account")
		}
		if errors.Is(err, userService.ErrLastAdmin) {
			return apiPkg.BadRequest(c, "cannot delete last admin")
		}
		return apiPkg.InternalError(c, "failed to delete user")
	}
	return apiPkg.NoContent(c)
}

func (h *UserHandler) ChangePassword(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid user id")
	}

	var req model.ChangePasswordRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if req.NewPassword == "" {
		return apiPkg.BadRequest(c, "new_password is required")
	}

	currentUserID, ok := c.Get(middleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		return apiPkg.Unauthorized(c, "user not found in context")
	}
	currentRole, _ := c.Get(middleware.ContextKeyRole).(model.UserRole)

	if err := h.service.ChangePassword(c.Request().Context(), id, req, currentUserID, currentRole); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "user not found")
		}
		if errors.Is(err, userService.ErrWrongPassword) {
			return apiPkg.BadRequest(c, "current password is incorrect")
		}
		if errors.Is(err, userService.ErrPasswordRequired) {
			return apiPkg.BadRequest(c, "current password is required")
		}
		return apiPkg.InternalError(c, "failed to change password")
	}
	return apiPkg.Success(c, map[string]string{"message": "password changed"})
}
