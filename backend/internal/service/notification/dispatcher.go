package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/antigravity/prometheus/internal/model"
)

// Sender is the interface that each notification channel must implement.
type Sender interface {
	Send(ctx context.Context, subject, body string) error
	Type() model.NotificationChannelType
}

// NewSender creates a Sender from a notification channel's type and config.
func NewSender(channelType model.NotificationChannelType, config json.RawMessage) (Sender, error) {
	switch channelType {
	case model.ChannelTypeEmail:
		return NewEmailSender(config)
	case model.ChannelTypeTelegram:
		return NewTelegramSender(config)
	case model.ChannelTypeWebhook:
		return NewWebhookSender(config)
	default:
		return nil, fmt.Errorf("unsupported channel type: %s", channelType)
	}
}

// Dispatch sends a message through the given sender and logs the result.
func Dispatch(ctx context.Context, sender Sender, subject, body string) error {
	if err := sender.Send(ctx, subject, body); err != nil {
		slog.Error("notification dispatch failed",
			slog.String("type", string(sender.Type())),
			slog.String("subject", subject),
			slog.Any("error", err),
		)
		return err
	}

	slog.Info("notification dispatched",
		slog.String("type", string(sender.Type())),
		slog.String("subject", subject),
	)
	return nil
}
