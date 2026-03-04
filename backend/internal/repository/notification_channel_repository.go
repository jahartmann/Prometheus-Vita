package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationChannelRepository interface {
	Create(ctx context.Context, channel *model.NotificationChannel) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.NotificationChannel, error)
	List(ctx context.Context) ([]model.NotificationChannel, error)
	ListActive(ctx context.Context) ([]model.NotificationChannel, error)
	Update(ctx context.Context, channel *model.NotificationChannel) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type pgNotificationChannelRepository struct {
	db *pgxpool.Pool
}

func NewNotificationChannelRepository(db *pgxpool.Pool) NotificationChannelRepository {
	return &pgNotificationChannelRepository{db: db}
}

func (r *pgNotificationChannelRepository) Create(ctx context.Context, channel *model.NotificationChannel) error {
	channel.ID = uuid.New()
	err := r.db.QueryRow(ctx,
		`INSERT INTO notification_channels (id, name, type, config, is_active, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		 RETURNING created_at, updated_at`,
		channel.ID, channel.Name, channel.Type, channel.Config, channel.IsActive, channel.CreatedBy,
	).Scan(&channel.CreatedAt, &channel.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create notification channel: %w", err)
	}
	return nil
}

func (r *pgNotificationChannelRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.NotificationChannel, error) {
	var ch model.NotificationChannel
	err := r.db.QueryRow(ctx,
		`SELECT id, name, type, config, is_active, created_by, created_at, updated_at
		 FROM notification_channels WHERE id = $1`, id,
	).Scan(&ch.ID, &ch.Name, &ch.Type, &ch.Config, &ch.IsActive, &ch.CreatedBy, &ch.CreatedAt, &ch.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get notification channel by id: %w", err)
	}
	return &ch, nil
}

func (r *pgNotificationChannelRepository) List(ctx context.Context) ([]model.NotificationChannel, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, type, config, is_active, created_by, created_at, updated_at
		 FROM notification_channels ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list notification channels: %w", err)
	}
	defer rows.Close()

	var channels []model.NotificationChannel
	for rows.Next() {
		var ch model.NotificationChannel
		if err := rows.Scan(&ch.ID, &ch.Name, &ch.Type, &ch.Config, &ch.IsActive, &ch.CreatedBy, &ch.CreatedAt, &ch.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan notification channel: %w", err)
		}
		channels = append(channels, ch)
	}
	return channels, rows.Err()
}

func (r *pgNotificationChannelRepository) ListActive(ctx context.Context) ([]model.NotificationChannel, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, type, config, is_active, created_by, created_at, updated_at
		 FROM notification_channels WHERE is_active = true ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list active notification channels: %w", err)
	}
	defer rows.Close()

	var channels []model.NotificationChannel
	for rows.Next() {
		var ch model.NotificationChannel
		if err := rows.Scan(&ch.ID, &ch.Name, &ch.Type, &ch.Config, &ch.IsActive, &ch.CreatedBy, &ch.CreatedAt, &ch.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan notification channel: %w", err)
		}
		channels = append(channels, ch)
	}
	return channels, rows.Err()
}

func (r *pgNotificationChannelRepository) Update(ctx context.Context, channel *model.NotificationChannel) error {
	_, err := r.db.Exec(ctx,
		`UPDATE notification_channels SET name=$1, config=$2, is_active=$3, updated_at=NOW()
		 WHERE id=$4`,
		channel.Name, channel.Config, channel.IsActive, channel.ID,
	)
	if err != nil {
		return fmt.Errorf("update notification channel: %w", err)
	}
	return nil
}

func (r *pgNotificationChannelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM notification_channels WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("delete notification channel: %w", err)
	}
	return nil
}
