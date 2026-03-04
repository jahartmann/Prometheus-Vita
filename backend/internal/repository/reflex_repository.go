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

type ReflexRuleRepository interface {
	Create(ctx context.Context, rule *model.ReflexRule) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.ReflexRule, error)
	List(ctx context.Context) ([]model.ReflexRule, error)
	ListActive(ctx context.Context) ([]model.ReflexRule, error)
	Update(ctx context.Context, rule *model.ReflexRule) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateLastTriggered(ctx context.Context, id uuid.UUID, t time.Time) error
}

type pgReflexRuleRepository struct {
	db *pgxpool.Pool
}

func NewReflexRuleRepository(db *pgxpool.Pool) ReflexRuleRepository {
	return &pgReflexRuleRepository{db: db}
}

func (r *pgReflexRuleRepository) Create(ctx context.Context, rule *model.ReflexRule) error {
	rule.ID = uuid.New()
	err := r.db.QueryRow(ctx,
		`INSERT INTO reflex_rules (id, name, description, trigger_metric, operator, threshold,
		        action_type, action_config, cooldown_seconds, is_active, node_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		 RETURNING created_at, updated_at`,
		rule.ID, rule.Name, rule.Description, rule.TriggerMetric, rule.Operator,
		rule.Threshold, rule.ActionType, rule.ActionConfig, rule.CooldownSeconds,
		rule.IsActive, rule.NodeID,
	).Scan(&rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create reflex rule: %w", err)
	}
	return nil
}

func (r *pgReflexRuleRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ReflexRule, error) {
	var rule model.ReflexRule
	err := r.db.QueryRow(ctx,
		`SELECT id, name, description, trigger_metric, operator, threshold,
		        action_type, action_config, cooldown_seconds, is_active, node_id,
		        last_triggered_at, trigger_count, created_at, updated_at
		 FROM reflex_rules WHERE id = $1`, id,
	).Scan(&rule.ID, &rule.Name, &rule.Description, &rule.TriggerMetric, &rule.Operator,
		&rule.Threshold, &rule.ActionType, &rule.ActionConfig, &rule.CooldownSeconds,
		&rule.IsActive, &rule.NodeID, &rule.LastTriggeredAt, &rule.TriggerCount,
		&rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get reflex rule by id: %w", err)
	}
	return &rule, nil
}

func (r *pgReflexRuleRepository) List(ctx context.Context) ([]model.ReflexRule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, description, trigger_metric, operator, threshold,
		        action_type, action_config, cooldown_seconds, is_active, node_id,
		        last_triggered_at, trigger_count, created_at, updated_at
		 FROM reflex_rules ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list reflex rules: %w", err)
	}
	defer rows.Close()
	return r.scanRules(rows)
}

func (r *pgReflexRuleRepository) ListActive(ctx context.Context) ([]model.ReflexRule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, description, trigger_metric, operator, threshold,
		        action_type, action_config, cooldown_seconds, is_active, node_id,
		        last_triggered_at, trigger_count, created_at, updated_at
		 FROM reflex_rules WHERE is_active = true ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list active reflex rules: %w", err)
	}
	defer rows.Close()
	return r.scanRules(rows)
}

func (r *pgReflexRuleRepository) Update(ctx context.Context, rule *model.ReflexRule) error {
	_, err := r.db.Exec(ctx,
		`UPDATE reflex_rules SET name=$1, description=$2, trigger_metric=$3, operator=$4, threshold=$5,
		        action_type=$6, action_config=$7, cooldown_seconds=$8, is_active=$9, node_id=$10, updated_at=NOW()
		 WHERE id=$11`,
		rule.Name, rule.Description, rule.TriggerMetric, rule.Operator, rule.Threshold,
		rule.ActionType, rule.ActionConfig, rule.CooldownSeconds, rule.IsActive, rule.NodeID, rule.ID,
	)
	if err != nil {
		return fmt.Errorf("update reflex rule: %w", err)
	}
	return nil
}

func (r *pgReflexRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM reflex_rules WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("delete reflex rule: %w", err)
	}
	return nil
}

func (r *pgReflexRuleRepository) UpdateLastTriggered(ctx context.Context, id uuid.UUID, t time.Time) error {
	_, err := r.db.Exec(ctx,
		"UPDATE reflex_rules SET last_triggered_at=$1, trigger_count = trigger_count + 1, updated_at=NOW() WHERE id=$2", t, id,
	)
	if err != nil {
		return fmt.Errorf("update reflex rule last triggered: %w", err)
	}
	return nil
}

func (r *pgReflexRuleRepository) scanRules(rows pgx.Rows) ([]model.ReflexRule, error) {
	var rules []model.ReflexRule
	for rows.Next() {
		var rule model.ReflexRule
		if err := rows.Scan(&rule.ID, &rule.Name, &rule.Description, &rule.TriggerMetric,
			&rule.Operator, &rule.Threshold, &rule.ActionType, &rule.ActionConfig,
			&rule.CooldownSeconds, &rule.IsActive, &rule.NodeID, &rule.LastTriggeredAt,
			&rule.TriggerCount, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan reflex rule: %w", err)
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}
