package handler

import (
	"errors"

	apiPkg "github.com/antigravity/prometheus/internal/api"
	"github.com/antigravity/prometheus/internal/api/middleware"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/agent"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type ChatHandler struct {
	service *agent.Service
}

func NewChatHandler(service *agent.Service) *ChatHandler {
	return &ChatHandler{service: service}
}

func (h *ChatHandler) Chat(c echo.Context) error {
	userID, ok := c.Get(middleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		return apiPkg.Unauthorized(c, "user not found in context")
	}

	var req model.ChatRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if req.Message == "" {
		return apiPkg.BadRequest(c, "message is required")
	}

	resp, err := h.service.Chat(c.Request().Context(), userID, req)
	if err != nil {
		return apiPkg.InternalError(c, "chat request failed")
	}

	return apiPkg.Success(c, resp)
}

func (h *ChatHandler) ListConversations(c echo.Context) error {
	userID, ok := c.Get(middleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		return apiPkg.Unauthorized(c, "user not found in context")
	}

	convs, err := h.service.ListConversations(c.Request().Context(), userID)
	if err != nil {
		return apiPkg.InternalError(c, "failed to list conversations")
	}

	if convs == nil {
		convs = []model.ChatConversation{}
	}

	return apiPkg.Success(c, convs)
}

func (h *ChatHandler) GetConversation(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid conversation id")
	}

	conv, err := h.service.GetConversation(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "conversation not found")
		}
		return apiPkg.InternalError(c, "failed to get conversation")
	}

	return apiPkg.Success(c, conv)
}

func (h *ChatHandler) GetMessages(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid conversation id")
	}

	msgs, err := h.service.GetMessages(c.Request().Context(), id)
	if err != nil {
		return apiPkg.InternalError(c, "failed to get messages")
	}

	if msgs == nil {
		msgs = []model.ChatMessage{}
	}

	return apiPkg.Success(c, msgs)
}

func (h *ChatHandler) DeleteConversation(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid conversation id")
	}

	if err := h.service.DeleteConversation(c.Request().Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "conversation not found")
		}
		return apiPkg.InternalError(c, "failed to delete conversation")
	}

	return apiPkg.NoContent(c)
}
