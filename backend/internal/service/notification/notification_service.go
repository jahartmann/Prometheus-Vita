package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/crypto"
	"github.com/google/uuid"
)

// sensitiveFields are config keys whose values must be encrypted at rest.
var sensitiveFields = map[string]bool{
	"smtp_password": true,
	"bot_token":     true,
	"secret":        true,
}

type Service struct {
	channelRepo repository.NotificationChannelRepository
	historyRepo repository.NotificationHistoryRepository
	encryptor   *crypto.Encryptor
}

func NewService(
	channelRepo repository.NotificationChannelRepository,
	historyRepo repository.NotificationHistoryRepository,
	encryptor *crypto.Encryptor,
) *Service {
	return &Service{
		channelRepo: channelRepo,
		historyRepo: historyRepo,
		encryptor:   encryptor,
	}
}

// Notify sends a notification to all active channels.
func (s *Service) Notify(ctx context.Context, eventType, subject, body string) {
	channels, err := s.channelRepo.ListActive(ctx)
	if err != nil {
		slog.Error("failed to list active channels for notification", slog.Any("error", err))
		return
	}

	for _, ch := range channels {
		s.sendToChannel(ctx, &ch, eventType, subject, body)
	}
}

// NotifyChannels sends a notification to specific channels by ID.
func (s *Service) NotifyChannels(ctx context.Context, channelIDs []uuid.UUID, eventType, subject, body string) {
	for _, id := range channelIDs {
		ch, err := s.channelRepo.GetByID(ctx, id)
		if err != nil {
			slog.Warn("failed to get channel for notification",
				slog.String("channel_id", id.String()),
				slog.Any("error", err),
			)
			continue
		}
		if !ch.IsActive {
			continue
		}
		s.sendToChannel(ctx, ch, eventType, subject, body)
	}
}

// TestChannel sends a test message to a specific channel.
func (s *Service) TestChannel(ctx context.Context, channelID uuid.UUID, message string) error {
	ch, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return fmt.Errorf("get channel: %w", err)
	}

	decryptedConfig, err := s.decryptConfig(ch.Config)
	if err != nil {
		return fmt.Errorf("decrypt config: %w", err)
	}

	sender, err := NewSender(ch.Type, decryptedConfig)
	if err != nil {
		return fmt.Errorf("create sender: %w", err)
	}

	return sender.Send(ctx, "Prometheus Test", message)
}

// CreateChannel creates a new notification channel, encrypting sensitive config fields.
func (s *Service) CreateChannel(ctx context.Context, req model.CreateChannelRequest, createdBy *uuid.UUID) (*model.NotificationChannel, error) {
	encryptedConfig, err := s.encryptConfig(req.Config)
	if err != nil {
		return nil, fmt.Errorf("encrypt config: %w", err)
	}

	channel := &model.NotificationChannel{
		Name:      req.Name,
		Type:      req.Type,
		Config:    encryptedConfig,
		IsActive:  true,
		CreatedBy: createdBy,
	}

	if err := s.channelRepo.Create(ctx, channel); err != nil {
		return nil, fmt.Errorf("create channel: %w", err)
	}

	// Return with masked config
	channel.Config = s.maskConfig(req.Config)
	return channel, nil
}

// GetChannel returns a channel by ID with masked config.
func (s *Service) GetChannel(ctx context.Context, id uuid.UUID) (*model.NotificationChannel, error) {
	ch, err := s.channelRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	decrypted, err := s.decryptConfig(ch.Config)
	if err == nil {
		ch.Config = s.maskConfig(decrypted)
	}
	return ch, nil
}

// ListChannels returns all channels with masked configs.
func (s *Service) ListChannels(ctx context.Context) ([]model.NotificationChannel, error) {
	channels, err := s.channelRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range channels {
		decrypted, err := s.decryptConfig(channels[i].Config)
		if err == nil {
			channels[i].Config = s.maskConfig(decrypted)
		}
	}
	return channels, nil
}

// UpdateChannel updates a channel, encrypting sensitive config fields if provided.
func (s *Service) UpdateChannel(ctx context.Context, id uuid.UUID, req model.UpdateChannelRequest) (*model.NotificationChannel, error) {
	ch, err := s.channelRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		ch.Name = *req.Name
	}
	if req.IsActive != nil {
		ch.IsActive = *req.IsActive
	}
	if req.Config != nil {
		encrypted, err := s.encryptConfig(*req.Config)
		if err != nil {
			return nil, fmt.Errorf("encrypt config: %w", err)
		}
		ch.Config = encrypted
	}

	if err := s.channelRepo.Update(ctx, ch); err != nil {
		return nil, err
	}

	// Mask config in response
	if req.Config != nil {
		ch.Config = s.maskConfig(*req.Config)
	} else {
		decrypted, err := s.decryptConfig(ch.Config)
		if err == nil {
			ch.Config = s.maskConfig(decrypted)
		}
	}
	return ch, nil
}

// DeleteChannel deletes a channel by ID.
func (s *Service) DeleteChannel(ctx context.Context, id uuid.UUID) error {
	return s.channelRepo.Delete(ctx, id)
}

// ListHistory returns notification history entries.
func (s *Service) ListHistory(ctx context.Context, limit, offset int) ([]model.NotificationHistoryEntry, error) {
	return s.historyRepo.List(ctx, limit, offset)
}

// ListHistoryByChannel returns notification history for a specific channel.
func (s *Service) ListHistoryByChannel(ctx context.Context, channelID uuid.UUID) ([]model.NotificationHistoryEntry, error) {
	return s.historyRepo.ListByChannel(ctx, channelID)
}

func (s *Service) sendToChannel(ctx context.Context, ch *model.NotificationChannel, eventType, subject, body string) {
	entry := &model.NotificationHistoryEntry{
		ChannelID: &ch.ID,
		EventType: eventType,
		Subject:   subject,
		Body:      body,
		Status:    model.NotifStatusPending,
	}

	if err := s.historyRepo.Create(ctx, entry); err != nil {
		slog.Error("failed to create notification history entry", slog.Any("error", err))
	}

	decryptedConfig, err := s.decryptConfig(ch.Config)
	if err != nil {
		slog.Error("failed to decrypt channel config",
			slog.String("channel_id", ch.ID.String()),
			slog.Any("error", err),
		)
		s.markFailed(ctx, entry.ID, fmt.Sprintf("decrypt config: %v", err))
		return
	}

	sender, err := NewSender(ch.Type, decryptedConfig)
	if err != nil {
		slog.Error("failed to create sender",
			slog.String("channel_id", ch.ID.String()),
			slog.Any("error", err),
		)
		s.markFailed(ctx, entry.ID, fmt.Sprintf("create sender: %v", err))
		return
	}

	if err := Dispatch(ctx, sender, subject, body); err != nil {
		s.markFailed(ctx, entry.ID, err.Error())
		return
	}

	now := time.Now()
	_ = s.historyRepo.UpdateStatus(ctx, entry.ID, model.NotifStatusSent, "", &now)
}

func (s *Service) markFailed(ctx context.Context, id uuid.UUID, errMsg string) {
	_ = s.historyRepo.UpdateStatus(ctx, id, model.NotifStatusFailed, errMsg, nil)
}

func (s *Service) encryptConfig(raw json.RawMessage) (json.RawMessage, error) {
	var configMap map[string]interface{}
	if err := json.Unmarshal(raw, &configMap); err != nil {
		return raw, nil
	}

	for key := range sensitiveFields {
		if val, ok := configMap[key]; ok {
			if strVal, isStr := val.(string); isStr && strVal != "" {
				encrypted, err := s.encryptor.Encrypt(strVal)
				if err != nil {
					return nil, fmt.Errorf("encrypt field %s: %w", key, err)
				}
				configMap[key] = encrypted
			}
		}
	}

	return json.Marshal(configMap)
}

func (s *Service) decryptConfig(raw json.RawMessage) (json.RawMessage, error) {
	var configMap map[string]interface{}
	if err := json.Unmarshal(raw, &configMap); err != nil {
		return raw, nil
	}

	for key := range sensitiveFields {
		if val, ok := configMap[key]; ok {
			if strVal, isStr := val.(string); isStr && strVal != "" {
				decrypted, err := s.encryptor.Decrypt(strVal)
				if err != nil {
					// Value might not be encrypted (e.g. old data)
					continue
				}
				configMap[key] = decrypted
			}
		}
	}

	return json.Marshal(configMap)
}

func (s *Service) maskConfig(raw json.RawMessage) json.RawMessage {
	var configMap map[string]interface{}
	if err := json.Unmarshal(raw, &configMap); err != nil {
		return raw
	}

	for key := range sensitiveFields {
		if val, ok := configMap[key]; ok {
			if strVal, isStr := val.(string); isStr && strVal != "" {
				if len(strVal) > 4 {
					configMap[key] = "****" + strVal[len(strVal)-4:]
				} else {
					configMap[key] = "****"
				}
			}
		}
	}

	masked, err := json.Marshal(configMap)
	if err != nil {
		return raw
	}
	return masked
}
