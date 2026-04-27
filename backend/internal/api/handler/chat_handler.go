package handler

import (
	"errors"
	"fmt"
	"log/slog"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
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
		slog.Error("chat request failed",
			slog.String("user_id", userID.String()),
			slog.String("message", req.Message),
			slog.String("model", req.Model),
			slog.Any("error", err),
		)
		return apiPkg.InternalError(c, fmt.Sprintf("Chat-Fehler: %v", err))
	}

	return apiPkg.Success(c, resp)
}

func (h *ChatHandler) ToolCatalog(c echo.Context) error {
	return apiPkg.Success(c, h.service.ToolCatalog())
}

// RecentActivity returns the agent's most recent tool calls — used by the
// dashboard "Agent activity" feed to show what the admin-agent has been
// doing without the user having to dig through individual conversations.
func (h *ChatHandler) RecentActivity(c echo.Context) error {
	limit := 50
	if l := c.QueryParam("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	calls, err := h.service.RecentActivity(c.Request().Context(), limit)
	if err != nil {
		return apiPkg.InternalError(c, "failed to load recent agent activity")
	}
	if calls == nil {
		calls = []model.AgentToolCall{}
	}
	return apiPkg.Success(c, calls)
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
