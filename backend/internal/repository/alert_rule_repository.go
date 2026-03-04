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

type AlertRuleRepository interface {
	Create(ctx context.Context, rule *model.AlertRule) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.AlertRule, error)
	List(ctx context.Context) ([]model.AlertRule, error)
	ListActive(ctx context.Context) ([]model.AlertRule, error)
	ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.AlertRule, error)
	Update(ctx context.Context, rule *model.AlertRule) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateLastTriggered(ctx context.Context, id uuid.UUID, t time.Time) error
}

type pgAlertRuleRepository struct {
	db *pgxpool.Pool
}

func NewAlertRuleRepository(db *pgxpool.Pool) AlertRuleRepository {
	return &pgAlertRuleRepository{db: db}
}

func (r *pgAlertRuleRepository) Create(ctx context.Context, rule *model.AlertRule) error {
	rule.ID = uuid.New()
	err := r.db.QueryRow(ctx,
		`INSERT INTO alert_rules (id, name, node_id, metric, operator, threshold, duration_seconds,
		        severity, channel_ids, escalation_policy_id, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		 RETURNING created_at, updated_at`,
		rule.ID, rule.Name, rule.NodeID, rule.Metric, rule.Operator,
		rule.Threshold, rule.DurationSeconds, rule.Severity, rule.ChannelIDs,
		rule.EscalationPolicyID, rule.IsActive,
	).Scan(&rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create alert rule: %w", err)
	}
	return nil
}

func (r *pgAlertRuleRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.AlertRule, error) {
	var rule model.AlertRule
	err := r.db.QueryRow(ctx,
		`SELECT id, name, node_id, metric, operator, threshold, duration_seconds,
		        severity, channel_ids, escalation_policy_id, is_active, last_triggered_at, created_at, updated_at
		 FROM alert_rules WHERE id = $1`, id,
	).Scan(&rule.ID, &rule.Name, &rule.NodeID, &rule.Metric, &rule.Operator,
		&rule.Threshold, &rule.DurationSeconds, &rule.Severity, &rule.ChannelIDs,
		&rule.EscalationPolicyID, &rule.IsActive, &rule.LastTriggeredAt, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get alert rule by id: %w", err)
	}
	return &rule, nil
}

func (r *pgAlertRuleRepository) List(ctx context.Context) ([]model.AlertRule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, node_id, metric, operator, threshold, duration_seconds,
		        severity, channel_ids, escalation_policy_id, is_active, last_triggered_at, created_at, updated_at
		 FROM alert_rules ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list alert rules: %w", err)
	}
	defer rows.Close()
	return r.scanRules(rows)
}

func (r *pgAlertRuleRepository) ListActive(ctx context.Context) ([]model.AlertRule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, node_id, metric, operator, threshold, duration_seconds,
		        severity, channel_ids, escalation_policy_id, is_active, last_triggered_at, created_at, updated_at
		 FROM alert_rules WHERE is_active = true ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list active alert rules: %w", err)
	}
	defer rows.Close()
	return r.scanRules(rows)
}

func (r *pgAlertRuleRepository) ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.AlertRule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, node_id, metric, operator, threshold, duration_seconds,
		        severity, channel_ids, escalation_policy_id, is_active, last_triggered_at, created_at, updated_at
		 FROM alert_rules WHERE node_id = $1 ORDER BY name ASC`, nodeID)
	if err != nil {
		return nil, fmt.Errorf("list alert rules by node: %w", err)
	}
	defer rows.Close()
	return r.scanRules(rows)
}

func (r *pgAlertRuleRepository) Update(ctx context.Context, rule *model.AlertRule) error {
	_, err := r.db.Exec(ctx,
		`UPDATE alert_rules SET name=$1, metric=$2, operator=$3, threshold=$4,
		        duration_seconds=$5, severity=$6, channel_ids=$7, escalation_policy_id=$8, is_active=$9, updated_at=NOW()
		 WHERE id=$10`,
		rule.Name, rule.Metric, rule.Operator, rule.Threshold,
		rule.DurationSeconds, rule.Severity, rule.ChannelIDs, rule.EscalationPolicyID, rule.IsActive, rule.ID,
	)
	if err != nil {
		return fmt.Errorf("update alert rule: %w", err)
	}
	return nil
}

func (r *pgAlertRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM alert_rules WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("delete alert rule: %w", err)
	}
	return nil
}

func (r *pgAlertRuleRepository) UpdateLastTriggered(ctx context.Context, id uuid.UUID, t time.Time) error {
	_, err := r.db.Exec(ctx,
		"UPDATE alert_rules SET last_triggered_at=$1, updated_at=NOW() WHERE id=$2", t, id,
	)
	if err != nil {
		return fmt.Errorf("update alert rule last triggered: %w", err)
	}
	return nil
}

func (r *pgAlertRuleRepository) scanRules(rows pgx.Rows) ([]model.AlertRule, error) {
	var rules []model.AlertRule
	for rows.Next() {
		var rule model.AlertRule
		if err := rows.Scan(&rule.ID, &rule.Name, &rule.NodeID, &rule.Metric, &rule.Operator,
			&rule.Threshold, &rule.DurationSeconds, &rule.Severity, &rule.ChannelIDs,
			&rule.EscalationPolicyID, &rule.IsActive, &rule.LastTriggeredAt, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan alert rule: %w", err)
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}
