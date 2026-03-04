package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationHistoryRepository interface {
	Create(ctx context.Context, entry *model.NotificationHistoryEntry) error
	List(ctx context.Context, limit, offset int) ([]model.NotificationHistoryEntry, error)
	ListByChannel(ctx context.Context, channelID uuid.UUID) ([]model.NotificationHistoryEntry, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.NotificationStatus, errorMsg string, sentAt *time.Time) error
	DeleteOlderThan(ctx context.Context, before time.Time) error
}

type pgNotificationHistoryRepository struct {
	db *pgxpool.Pool
}

func NewNotificationHistoryRepository(db *pgxpool.Pool) NotificationHistoryRepository {
	return &pgNotificationHistoryRepository{db: db}
}

func (r *pgNotificationHistoryRepository) Create(ctx context.Context, entry *model.NotificationHistoryEntry) error {
	entry.ID = uuid.New()
	err := r.db.QueryRow(ctx,
		`INSERT INTO notification_history (id, channel_id, event_type, subject, body, status, error_message, metadata, created_at, sent_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), $9)
		 RETURNING created_at`,
		entry.ID, entry.ChannelID, entry.EventType, entry.Subject, entry.Body,
		entry.Status, entry.ErrorMessage, entry.Metadata, entry.SentAt,
	).Scan(&entry.CreatedAt)
	if err != nil {
		return fmt.Errorf("create notification history: %w", err)
	}
	return nil
}

func (r *pgNotificationHistoryRepository) List(ctx context.Context, limit, offset int) ([]model.NotificationHistoryEntry, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, channel_id, event_type, subject, body, status, COALESCE(error_message, ''), metadata, created_at, sent_at
		 FROM notification_history ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list notification history: %w", err)
	}
	defer rows.Close()

	var entries []model.NotificationHistoryEntry
	for rows.Next() {
		var e model.NotificationHistoryEntry
		if err := rows.Scan(&e.ID, &e.ChannelID, &e.EventType, &e.Subject, &e.Body,
			&e.Status, &e.ErrorMessage, &e.Metadata, &e.CreatedAt, &e.SentAt); err != nil {
			return nil, fmt.Errorf("scan notification history: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *pgNotificationHistoryRepository) ListByChannel(ctx context.Context, channelID uuid.UUID) ([]model.NotificationHistoryEntry, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, channel_id, event_type, subject, body, status, COALESCE(error_message, ''), metadata, created_at, sent_at
		 FROM notification_history WHERE channel_id = $1 ORDER BY created_at DESC`, channelID)
	if err != nil {
		return nil, fmt.Errorf("list notification history by channel: %w", err)
	}
	defer rows.Close()

	var entries []model.NotificationHistoryEntry
	for rows.Next() {
		var e model.NotificationHistoryEntry
		if err := rows.Scan(&e.ID, &e.ChannelID, &e.EventType, &e.Subject, &e.Body,
			&e.Status, &e.ErrorMessage, &e.Metadata, &e.CreatedAt, &e.SentAt); err != nil {
			return nil, fmt.Errorf("scan notification history: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *pgNotificationHistoryRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.NotificationStatus, errorMsg string, sentAt *time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE notification_history SET status=$1, error_message=$2, sent_at=$3 WHERE id=$4`,
		status, errorMsg, sentAt, id,
	)
	if err != nil {
		return fmt.Errorf("update notification history status: %w", err)
	}
	return nil
}

func (r *pgNotificationHistoryRepository) DeleteOlderThan(ctx context.Context, before time.Time) error {
	_, err := r.db.Exec(ctx, "DELETE FROM notification_history WHERE created_at < $1", before)
	if err != nil {
		return fmt.Errorf("delete old notification history: %w", err)
	}
	return nil
}
