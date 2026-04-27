package notification

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/crypto"
	"github.com/google/uuid"
)

func TestUpdateChannelPreservesExistingSMTPPassword(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		incomingConfig json.RawMessage
	}{
		{
			name:           "password omitted",
			incomingConfig: json.RawMessage(`{"smtp_host":"smtp2.example.com","smtp_port":587,"smtp_user":"ops","from_address":"ops@example.com","to_addresses":["admin@example.com"],"use_tls":true}`),
		},
		{
			name:           "masked password ignored",
			incomingConfig: json.RawMessage(`{"smtp_host":"smtp2.example.com","smtp_port":587,"smtp_user":"ops","smtp_password":"****cret","from_address":"ops@example.com","to_addresses":["admin@example.com"],"use_tls":true}`),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			enc, err := crypto.NewEncryptor(strings.Repeat("1", 64))
			if err != nil {
				t.Fatalf("new encryptor: %v", err)
			}

			channelRepo := &memoryNotificationChannelRepo{}
			svc := NewService(channelRepo, &noopNotificationHistoryRepo{}, enc)
			storedConfig, err := svc.encryptConfig(json.RawMessage(`{"smtp_host":"smtp.example.com","smtp_port":587,"smtp_user":"ops","smtp_password":"real-secret","from_address":"old@example.com","to_addresses":["admin@example.com"],"use_tls":true}`))
			if err != nil {
				t.Fatalf("encrypt stored config: %v", err)
			}

			channelID := uuid.New()
			channelRepo.channel = &model.NotificationChannel{
				ID:       channelID,
				Name:     "SMTP E-Mail",
				Type:     model.ChannelTypeEmail,
				Config:   storedConfig,
				IsActive: true,
			}

			_, err = svc.UpdateChannel(context.Background(), channelID, model.UpdateChannelRequest{
				Config: &tc.incomingConfig,
			})
			if err != nil {
				t.Fatalf("update channel: %v", err)
			}

			decrypted, err := svc.decryptConfig(channelRepo.channel.Config)
			if err != nil {
				t.Fatalf("decrypt updated config: %v", err)
			}

			var config map[string]interface{}
			if err := json.Unmarshal(decrypted, &config); err != nil {
				t.Fatalf("unmarshal updated config: %v", err)
			}

			if got := config["smtp_password"]; got != "real-secret" {
				t.Fatalf("smtp_password = %v, want real-secret", got)
			}
			if got := config["from_address"]; got != "ops@example.com" {
				t.Fatalf("from_address = %v, want ops@example.com", got)
			}
		})
	}
}

type memoryNotificationChannelRepo struct {
	channel *model.NotificationChannel
}

func (r *memoryNotificationChannelRepo) Create(ctx context.Context, channel *model.NotificationChannel) error {
	r.channel = channel
	return nil
}

func (r *memoryNotificationChannelRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.NotificationChannel, error) {
	if r.channel == nil || r.channel.ID != id {
		return nil, repository.ErrNotFound
	}
	copy := *r.channel
	return &copy, nil
}

func (r *memoryNotificationChannelRepo) List(ctx context.Context) ([]model.NotificationChannel, error) {
	if r.channel == nil {
		return nil, nil
	}
	return []model.NotificationChannel{*r.channel}, nil
}

func (r *memoryNotificationChannelRepo) ListActive(ctx context.Context) ([]model.NotificationChannel, error) {
	if r.channel == nil || !r.channel.IsActive {
		return nil, nil
	}
	return []model.NotificationChannel{*r.channel}, nil
}

func (r *memoryNotificationChannelRepo) Update(ctx context.Context, channel *model.NotificationChannel) error {
	copy := *channel
	r.channel = &copy
	return nil
}

func (r *memoryNotificationChannelRepo) Delete(ctx context.Context, id uuid.UUID) error {
	r.channel = nil
	return nil
}

type noopNotificationHistoryRepo struct{}

func (r *noopNotificationHistoryRepo) Create(ctx context.Context, entry *model.NotificationHistoryEntry) error {
	return nil
}

func (r *noopNotificationHistoryRepo) List(ctx context.Context, limit, offset int) ([]model.NotificationHistoryEntry, error) {
	return nil, nil
}

func (r *noopNotificationHistoryRepo) ListByChannel(ctx context.Context, channelID uuid.UUID) ([]model.NotificationHistoryEntry, error) {
	return nil, nil
}

func (r *noopNotificationHistoryRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.NotificationStatus, errorMsg string, sentAt *time.Time) error {
	return nil
}

func (r *noopNotificationHistoryRepo) DeleteOlderThan(ctx context.Context, before time.Time) error {
	return nil
}
