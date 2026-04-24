package repository

import (
	"context"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SecurityEventRepository interface {
	Create(ctx context.Context, event *model.SecurityEvent) error
	ListUnacknowledged(ctx context.Context) ([]model.SecurityEvent, error)
	ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.SecurityEvent, error)
	ListFiltered(ctx context.Context, filter model.QueryFilter) ([]model.SecurityEvent, error)
	Acknowledge(ctx context.Context, id uuid.UUID) error
	ListRecent(ctx context.Context, limit int) ([]model.SecurityEvent, error)
	CountBySeverity(ctx context.Context) (map[string]int, error)
	CountByCategory(ctx context.Context) (map[string]int, error)
}

type pgSecurityEventRepository struct {
	db *pgxpool.Pool
}

func NewSecurityEventRepository(db *pgxpool.Pool) SecurityEventRepository {
	return &pgSecurityEventRepository{db: db}
}

func (r *pgSecurityEventRepository) Create(ctx context.Context, event *model.SecurityEvent) error {
	event.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO security_events (id, node_id, category, severity, title, description, impact,
		        recommendation, metrics, affected_vms, node_name, is_acknowledged, detected_at, analysis_model)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), $13)`,
		event.ID, event.NodeID, event.Category, event.Severity, event.Title, event.Description,
		event.Impact, event.Recommendation, event.Metrics, event.AffectedVMs, event.NodeName,
		event.IsAcknowledged, event.AnalysisModel,
	)
	if err != nil {
		return fmt.Errorf("create security event: %w", err)
	}
	return nil
}

func (r *pgSecurityEventRepository) ListUnacknowledged(ctx context.Context) ([]model.SecurityEvent, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, category, severity, title, description, impact, recommendation,
		        metrics, affected_vms, node_name, is_acknowledged, detected_at, acknowledged_at, analysis_model
		 FROM security_events WHERE is_acknowledged = false ORDER BY detected_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list unacknowledged security events: %w", err)
	}
	defer rows.Close()
	return r.scanEvents(rows)
}

func (r *pgSecurityEventRepository) ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.SecurityEvent, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, category, severity, title, description, impact, recommendation,
		        metrics, affected_vms, node_name, is_acknowledged, detected_at, acknowledged_at, analysis_model
		 FROM security_events WHERE node_id = $1 ORDER BY detected_at DESC LIMIT 100`, nodeID)
	if err != nil {
		return nil, fmt.Errorf("list security events by node: %w", err)
	}
	defer rows.Close()
	return r.scanEvents(rows)
}

func (r *pgSecurityEventRepository) ListFiltered(ctx context.Context, filter model.QueryFilter) ([]model.SecurityEvent, error) {
	limit := filter.NormalizedLimit(100, 500)
	status := filter.Status
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, category, severity, title, description, impact, recommendation,
		        metrics, affected_vms, node_name, is_acknowledged, detected_at, acknowledged_at, analysis_model
		 FROM security_events
		 WHERE ($1::timestamptz IS NULL OR detected_at >= $1)
		   AND ($2::timestamptz IS NULL OR detected_at <= $2)
		   AND ($3::uuid IS NULL OR node_id = $3)
		   AND ($4 = '' OR $4 = 'all' OR severity = $4)
		   AND ($5 = '' OR $5 = 'all' OR category = $5)
		   AND ($6 = '' OR $6 = 'all' OR ($6 = 'acknowledged' AND is_acknowledged = true) OR ($6 = 'open' AND is_acknowledged = false) OR ($6 = 'unacknowledged' AND is_acknowledged = false))
		   AND ($7 = '' OR title ILIKE '%' || $7 || '%' OR description ILIKE '%' || $7 || '%' OR impact ILIKE '%' || $7 || '%' OR recommendation ILIKE '%' || $7 || '%')
		 ORDER BY detected_at DESC
		 LIMIT $8 OFFSET $9`,
		filter.From, filter.To, filter.NodeID, filter.Severity, filter.Category, status, filter.Query, limit, filter.Offset)
	if err != nil {
		return nil, fmt.Errorf("list filtered security events: %w", err)
	}
	defer rows.Close()
	return r.scanEvents(rows)
}

func (r *pgSecurityEventRepository) Acknowledge(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE security_events SET is_acknowledged = true, acknowledged_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("acknowledge security event: %w", err)
	}
	return nil
}

func (r *pgSecurityEventRepository) ListRecent(ctx context.Context, limit int) ([]model.SecurityEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, category, severity, title, description, impact, recommendation,
		        metrics, affected_vms, node_name, is_acknowledged, detected_at, acknowledged_at, analysis_model
		 FROM security_events ORDER BY detected_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent security events: %w", err)
	}
	defer rows.Close()
	return r.scanEvents(rows)
}

func (r *pgSecurityEventRepository) CountBySeverity(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.Query(ctx,
		`SELECT severity, COUNT(*) FROM security_events WHERE is_acknowledged = false GROUP BY severity`)
	if err != nil {
		return nil, fmt.Errorf("count security events by severity: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var severity string
		var count int
		if err := rows.Scan(&severity, &count); err != nil {
			return nil, fmt.Errorf("scan severity count: %w", err)
		}
		counts[severity] = count
	}
	return counts, rows.Err()
}

func (r *pgSecurityEventRepository) CountByCategory(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.Query(ctx,
		`SELECT category, COUNT(*) FROM security_events WHERE is_acknowledged = false GROUP BY category`)
	if err != nil {
		return nil, fmt.Errorf("count security events by category: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			return nil, fmt.Errorf("scan category count: %w", err)
		}
		counts[category] = count
	}
	return counts, rows.Err()
}

type eventRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func (r *pgSecurityEventRepository) scanEvents(rows eventRows) ([]model.SecurityEvent, error) {
	var events []model.SecurityEvent
	for rows.Next() {
		var e model.SecurityEvent
		if err := rows.Scan(&e.ID, &e.NodeID, &e.Category, &e.Severity, &e.Title,
			&e.Description, &e.Impact, &e.Recommendation, &e.Metrics,
			&e.AffectedVMs, &e.NodeName, &e.IsAcknowledged, &e.DetectedAt,
			&e.AcknowledgedAt, &e.AnalysisModel); err != nil {
			return nil, fmt.Errorf("scan security event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
