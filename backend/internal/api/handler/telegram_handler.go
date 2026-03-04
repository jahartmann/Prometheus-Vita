package handler

import (
	"context"
	"errors"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/api/middleware"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	telegramSvc "github.com/antigravity/prometheus/internal/service/telegram"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type TelegramHandler struct {
	linkRepo   repository.TelegramLinkRepository
	botSvc     *telegramSvc.BotService
	botEnabled bool
}

func NewTelegramHandler(linkRepo repository.TelegramLinkRepository, botSvc *telegramSvc.BotService, botEnabled bool) *TelegramHandler {
	return &TelegramHandler{
		linkRepo:   linkRepo,
		botSvc:     botSvc,
		botEnabled: botEnabled,
	}
}

func (h *TelegramHandler) LinkTelegram(c echo.Context) error {
	if !h.botEnabled {
		return apiPkg.BadRequest(c, "Telegram bot is not configured")
	}

	userID, ok := c.Get(middleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		return apiPkg.Unauthorized(c, "user not found in context")
	}

	ctx := c.Request().Context()

	// Check if already linked
	existing, err := h.linkRepo.GetByUserID(ctx, userID)
	if err == nil && existing.IsVerified {
		return apiPkg.BadRequest(c, "Telegram account already linked")
	}

	// Delete existing unverified link if any
	if err == nil && !existing.IsVerified {
		_ = h.linkRepo.Delete(ctx, existing.ID)
	}

	// Generate verification code
	code := telegramSvc.GenerateVerificationCode()

	link := &model.TelegramUserLink{
		UserID:           userID,
		VerificationCode: code,
	}
	if err := h.linkRepo.Create(ctx, link); err != nil {
		return apiPkg.InternalError(c, "failed to create telegram link")
	}

	// Get bot username
	botUsername := ""
	if h.botSvc != nil {
		botUsername, _ = h.botSvc.GetBotUsername(context.Background())
	}

	return apiPkg.Created(c, model.TelegramLinkResponse{
		VerificationCode: code,
		BotUsername:       botUsername,
		IsVerified:       false,
	})
}

func (h *TelegramHandler) GetTelegramStatus(c echo.Context) error {
	userID, ok := c.Get(middleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		return apiPkg.Unauthorized(c, "user not found in context")
	}

	link, err := h.linkRepo.GetByUserID(c.Request().Context(), userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.Success(c, map[string]interface{}{
				"linked":     false,
				"is_verified": false,
				"bot_enabled": h.botEnabled,
			})
		}
		return apiPkg.InternalError(c, "failed to get telegram link status")
	}

	return apiPkg.Success(c, map[string]interface{}{
		"linked":            true,
		"is_verified":       link.IsVerified,
		"telegram_username": link.TelegramUsername,
		"verification_code": link.VerificationCode,
		"bot_enabled":       h.botEnabled,
	})
}

func (h *TelegramHandler) UnlinkTelegram(c echo.Context) error {
	userID, ok := c.Get(middleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		return apiPkg.Unauthorized(c, "user not found in context")
	}

	if err := h.linkRepo.DeleteByUserID(c.Request().Context(), userID); err != nil {
		return apiPkg.InternalError(c, "failed to unlink telegram")
	}

	return apiPkg.NoContent(c)
}
