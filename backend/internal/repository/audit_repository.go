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
	ListFiltered(ctx context.Context, filter model.QueryFilter) ([]model.AuditLogEntry, error)
	ListWithAgentActions(ctx context.Context, limit, offset int) ([]model.AuditLogEntry, error)
	ListWithAgentActionsFiltered(ctx context.Context, filter model.QueryFilter) ([]model.AuditLogEntry, error)
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
		`SELECT a.id, a.user_id, COALESCE(u.username, ''), a.api_token_id, a.method, a.path, a.status_code,
		        COALESCE(a.ip_address, ''), COALESCE(a.user_agent, ''), a.request_body, a.duration_ms, a.created_at
		 FROM api_audit_log a
		 LEFT JOIN users u ON a.user_id = u.id
		 ORDER BY a.created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list audit log: %w", err)
	}
	defer rows.Close()

	var entries []model.AuditLogEntry
	for rows.Next() {
		var e model.AuditLogEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.Username, &e.APITokenID, &e.Method, &e.Path,
			&e.StatusCode, &e.IPAddress, &e.UserAgent, &e.RequestBody, &e.DurationMS, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *pgAuditRepository) ListFiltered(ctx context.Context, filter model.QueryFilter) ([]model.AuditLogEntry, error) {
	return r.listAuditFiltered(ctx, filter, false)
}

func (r *pgAuditRepository) ListWithAgentActions(ctx context.Context, limit, offset int) ([]model.AuditLogEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, username, api_token_id, method, path, status_code,
		        ip_address, user_agent, request_body, duration_ms, created_at
		 FROM (
		   (SELECT a.id, a.user_id, COALESCE(u.username, '') AS username, a.api_token_id, a.method, a.path, a.status_code,
		           COALESCE(a.ip_address, '') AS ip_address, COALESCE(a.user_agent, '') AS user_agent,
		           a.request_body, a.duration_ms, a.created_at
		    FROM api_audit_log a
		    LEFT JOIN users u ON a.user_id = u.id)
		   UNION ALL
		   (SELECT atc.id, cc.user_id, COALESCE(u2.username, '') AS username, NULL::uuid AS api_token_id,
		           'AGENT' AS method,
		           atc.tool_name AS path,
		           CASE WHEN atc.status = 'success' THEN 200 WHEN atc.status = 'error' THEN 500 ELSE 0 END AS status_code,
		           '' AS ip_address,
		           'KI-Agent' AS user_agent,
		           NULL::jsonb AS request_body,
		           atc.duration_ms,
		           atc.created_at
		    FROM agent_tool_calls atc
		    JOIN chat_messages cm ON atc.message_id = cm.id
		    JOIN chat_conversations cc ON cm.conversation_id = cc.id
		    LEFT JOIN users u2 ON cc.user_id = u2.id)
		 ) combined
		 ORDER BY created_at DESC
		 LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list unified audit log: %w", err)
	}
	defer rows.Close()

	var entries []model.AuditLogEntry
	for rows.Next() {
		var e model.AuditLogEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.Username, &e.APITokenID, &e.Method, &e.Path,
			&e.StatusCode, &e.IPAddress, &e.UserAgent, &e.RequestBody, &e.DurationMS, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan unified audit log: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *pgAuditRepository) ListWithAgentActionsFiltered(ctx context.Context, filter model.QueryFilter) ([]model.AuditLogEntry, error) {
	return r.listAuditFiltered(ctx, filter, true)
}

func (r *pgAuditRepository) listAuditFiltered(ctx context.Context, filter model.QueryFilter, includeAgent bool) ([]model.AuditLogEntry, error) {
	limit := filter.NormalizedLimit(50, 500)
	sourceFilter := filter.Source
	if !includeAgent && sourceFilter == "" {
		sourceFilter = "audit"
	}

	base := `(SELECT a.id, a.user_id, COALESCE(u.username, '') AS username, a.api_token_id, a.method, a.path, a.status_code,
	           COALESCE(a.ip_address, '') AS ip_address, COALESCE(a.user_agent, '') AS user_agent,
	           a.request_body, a.duration_ms, a.created_at, 'audit' AS source
	    FROM api_audit_log a
	    LEFT JOIN users u ON a.user_id = u.id)`
	if includeAgent {
		base += ` UNION ALL
		   (SELECT atc.id, cc.user_id, COALESCE(u2.username, '') AS username, NULL::uuid AS api_token_id,
		           'AGENT' AS method,
		           atc.tool_name AS path,
		           CASE WHEN atc.status = 'success' THEN 200 WHEN atc.status = 'error' THEN 500 ELSE 0 END AS status_code,
		           '' AS ip_address,
		           'KI-Agent' AS user_agent,
		           NULL::jsonb AS request_body,
		           atc.duration_ms,
		           atc.created_at,
		           'agent' AS source
		    FROM agent_tool_calls atc
		    JOIN chat_messages cm ON atc.message_id = cm.id
		    JOIN chat_conversations cc ON cm.conversation_id = cc.id
		    LEFT JOIN users u2 ON cc.user_id = u2.id)`
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, username, api_token_id, method, path, status_code,
		        ip_address, user_agent, request_body, duration_ms, created_at
		 FROM (`+base+`) combined
		 WHERE ($1::timestamptz IS NULL OR created_at >= $1)
		   AND ($2::timestamptz IS NULL OR created_at <= $2)
		   AND ($3::uuid IS NULL OR user_id = $3)
		   AND ($4 = '' OR $4 = 'all' OR source = $4)
		   AND ($5 = '' OR $5 = 'all' OR method = $5)
		   AND ($6 = '' OR $6 = 'all' OR COALESCE(request_body->>'category', '') = $6)
		   AND ($7 = '' OR $7 = 'all' OR
		        CASE
		          WHEN status_code >= 500 THEN 'critical'
		          WHEN status_code >= 400 OR method = 'DELETE' THEN 'warning'
		          ELSE 'info'
		        END = $7)
		   AND ($8 = '' OR path ILIKE '%' || $8 || '%' OR method ILIKE '%' || $8 || '%' OR username ILIKE '%' || $8 || '%')
		 ORDER BY created_at DESC
		 LIMIT $9 OFFSET $10`,
		filter.From, filter.To, filter.UserID, sourceFilter, filter.Method, filter.Category,
		filter.Severity, filter.Query, limit, filter.Offset)
	if err != nil {
		return nil, fmt.Errorf("list filtered audit log: %w", err)
	}
	defer rows.Close()

	var entries []model.AuditLogEntry
	for rows.Next() {
		var e model.AuditLogEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.Username, &e.APITokenID, &e.Method, &e.Path,
			&e.StatusCode, &e.IPAddress, &e.UserAgent, &e.RequestBody, &e.DurationMS, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan filtered audit log: %w", err)
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
		`SELECT a.id, a.user_id, COALESCE(u.username, ''), a.api_token_id, a.method, a.path, a.status_code,
		        COALESCE(a.ip_address, ''), COALESCE(a.user_agent, ''), a.request_body, a.duration_ms, a.created_at
		 FROM api_audit_log a
		 LEFT JOIN users u ON a.user_id = u.id
		 WHERE a.user_id = $1 ORDER BY a.created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list audit log by user: %w", err)
	}
	defer rows.Close()

	var entries []model.AuditLogEntry
	for rows.Next() {
		var e model.AuditLogEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.Username, &e.APITokenID, &e.Method, &e.Path,
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
