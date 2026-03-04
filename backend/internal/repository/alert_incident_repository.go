package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AlertIncidentRepository interface {
	Create(ctx context.Context, incident *model.AlertIncident) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.AlertIncident, error)
	List(ctx context.Context, limit, offset int) ([]model.AlertIncident, error)
	ListByStatus(ctx context.Context, status model.IncidentStatus) ([]model.AlertIncident, error)
	ListOpenByRule(ctx context.Context, ruleID uuid.UUID) ([]model.AlertIncident, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.IncidentStatus) error
	Acknowledge(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	Resolve(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	UpdateEscalation(ctx context.Context, id uuid.UUID, step int, escalatedAt time.Time) error
}

type pgAlertIncidentRepository struct {
	db *pgxpool.Pool
}

func NewAlertIncidentRepository(db *pgxpool.Pool) AlertIncidentRepository {
	return &pgAlertIncidentRepository{db: db}
}

func (r *pgAlertIncidentRepository) Create(ctx context.Context, incident *model.AlertIncident) error {
	incident.ID = uuid.New()
	err := r.db.QueryRow(ctx,
		`INSERT INTO alert_incidents (id, alert_rule_id, status, current_step, triggered_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, NOW(), NOW(), NOW())
		 RETURNING triggered_at, created_at, updated_at`,
		incident.ID, incident.AlertRuleID, incident.Status, incident.CurrentStep,
	).Scan(&incident.TriggeredAt, &incident.CreatedAt, &incident.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create alert incident: %w", err)
	}
	return nil
}

func (r *pgAlertIncidentRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.AlertIncident, error) {
	var inc model.AlertIncident
	err := r.db.QueryRow(ctx,
		`SELECT id, alert_rule_id, status, current_step, triggered_at,
		        acknowledged_at, acknowledged_by, resolved_at, resolved_by,
		        last_escalated_at, created_at, updated_at
		 FROM alert_incidents WHERE id = $1`, id,
	).Scan(&inc.ID, &inc.AlertRuleID, &inc.Status, &inc.CurrentStep, &inc.TriggeredAt,
		&inc.AcknowledgedAt, &inc.AcknowledgedBy, &inc.ResolvedAt, &inc.ResolvedBy,
		&inc.LastEscalatedAt, &inc.CreatedAt, &inc.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get alert incident: %w", err)
	}
	return &inc, nil
}

func (r *pgAlertIncidentRepository) List(ctx context.Context, limit, offset int) ([]model.AlertIncident, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, alert_rule_id, status, current_step, triggered_at,
		        acknowledged_at, acknowledged_by, resolved_at, resolved_by,
		        last_escalated_at, created_at, updated_at
		 FROM alert_incidents ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list alert incidents: %w", err)
	}
	defer rows.Close()
	return r.scanIncidents(rows)
}

func (r *pgAlertIncidentRepository) ListByStatus(ctx context.Context, status model.IncidentStatus) ([]model.AlertIncident, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, alert_rule_id, status, current_step, triggered_at,
		        acknowledged_at, acknowledged_by, resolved_at, resolved_by,
		        last_escalated_at, created_at, updated_at
		 FROM alert_incidents WHERE status = $1 ORDER BY created_at DESC`, status)
	if err != nil {
		return nil, fmt.Errorf("list alert incidents by status: %w", err)
	}
	defer rows.Close()
	return r.scanIncidents(rows)
}

func (r *pgAlertIncidentRepository) ListOpenByRule(ctx context.Context, ruleID uuid.UUID) ([]model.AlertIncident, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, alert_rule_id, status, current_step, triggered_at,
		        acknowledged_at, acknowledged_by, resolved_at, resolved_by,
		        last_escalated_at, created_at, updated_at
		 FROM alert_incidents
		 WHERE alert_rule_id = $1 AND status IN ('triggered', 'acknowledged')
		 ORDER BY created_at DESC`, ruleID)
	if err != nil {
		return nil, fmt.Errorf("list open incidents by rule: %w", err)
	}
	defer rows.Close()
	return r.scanIncidents(rows)
}

func (r *pgAlertIncidentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.IncidentStatus) error {
	_, err := r.db.Exec(ctx,
		`UPDATE alert_incidents SET status=$1, updated_at=NOW() WHERE id=$2`, status, id)
	if err != nil {
		return fmt.Errorf("update incident status: %w", err)
	}
	return nil
}

func (r *pgAlertIncidentRepository) Acknowledge(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE alert_incidents SET status='acknowledged', acknowledged_at=NOW(), acknowledged_by=$1, updated_at=NOW()
		 WHERE id=$2`, userID, id)
	if err != nil {
		return fmt.Errorf("acknowledge incident: %w", err)
	}
	return nil
}

func (r *pgAlertIncidentRepository) Resolve(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE alert_incidents SET status='resolved', resolved_at=NOW(), resolved_by=$1, updated_at=NOW()
		 WHERE id=$2`, userID, id)
	if err != nil {
		return fmt.Errorf("resolve incident: %w", err)
	}
	return nil
}

func (r *pgAlertIncidentRepository) UpdateEscalation(ctx context.Context, id uuid.UUID, step int, escalatedAt time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE alert_incidents SET current_step=$1, last_escalated_at=$2, updated_at=NOW()
		 WHERE id=$3`, step, escalatedAt, id)
	if err != nil {
		return fmt.Errorf("update incident escalation: %w", err)
	}
	return nil
}

func (r *pgAlertIncidentRepository) scanIncidents(rows pgx.Rows) ([]model.AlertIncident, error) {
	var incidents []model.AlertIncident
	for rows.Next() {
		var inc model.AlertIncident
		if err := rows.Scan(&inc.ID, &inc.AlertRuleID, &inc.Status, &inc.CurrentStep, &inc.TriggeredAt,
			&inc.AcknowledgedAt, &inc.AcknowledgedBy, &inc.ResolvedAt, &inc.ResolvedBy,
			&inc.LastEscalatedAt, &inc.CreatedAt, &inc.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan alert incident: %w", err)
		}
		incidents = append(incidents, inc)
	}
	return incidents, rows.Err()
}
