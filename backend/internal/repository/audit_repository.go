package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditRepository interface {
	Create(ctx context.Context, entry *model.AuditLogEntry) error
	List(ctx context.Context, limit, offset int) ([]model.AuditLogEntry, error)
	ListByUser(ctx context.Context, userID uuid.UUID, limit int) ([]model.AuditLogEntry, error)
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

type pgAuditRepository struct {
	db *pgxpool.Pool
}

func NewAuditRepository(db *pgxpool.Pool) AuditRepository {
	return &pgAuditRepository{db: db}
}

func (r *pgAuditRepository) Create(ctx context.Context, entry *model.AuditLogEntry) error {
	entry.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO api_audit_log (id, user_id, api_token_id, method, path, status_code, ip_address, user_agent, request_body, duration_ms, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())`,
		entry.ID, entry.UserID, entry.APITokenID, entry.Method, entry.Path,
		entry.StatusCode, entry.IPAddress, entry.UserAgent, entry.RequestBody, entry.DurationMS,
	)
	if err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}
	return nil
}

func (r *pgAuditRepository) List(ctx context.Context, limit, offset int) ([]model.AuditLogEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, api_token_id, method, path, status_code, COALESCE(ip_address, ''), COALESCE(user_agent, ''), request_body, duration_ms, created_at
		 FROM api_audit_log ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list audit log: %w", err)
	}
	defer rows.Close()

	var entries []model.AuditLogEntry
	for rows.Next() {
		var e model.AuditLogEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.APITokenID, &e.Method, &e.Path,
			&e.StatusCode, &e.IPAddress, &e.UserAgent, &e.RequestBody, &e.DurationMS, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *pgAuditRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit int) ([]model.AuditLogEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, api_token_id, method, path, status_code, COALESCE(ip_address, ''), COALESCE(user_agent, ''), request_body, duration_ms, created_at
		 FROM api_audit_log WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list audit log by user: %w", err)
	}
	defer rows.Close()

	var entries []model.AuditLogEntry
	for rows.Next() {
		var e model.AuditLogEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.APITokenID, &e.Method, &e.Path,
			&e.StatusCode, &e.IPAddress, &e.UserAgent, &e.RequestBody, &e.DurationMS, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *pgAuditRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.Exec(ctx, "DELETE FROM api_audit_log WHERE created_at < $1", before)
	if err != nil {
		return 0, fmt.Errorf("delete old audit logs: %w", err)
	}
	return result.RowsAffected(), nil
}
