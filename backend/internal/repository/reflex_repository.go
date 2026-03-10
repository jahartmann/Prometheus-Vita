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
		        action_type, action_config, cooldown_seconds, is_active, node_id,
		        schedule_type, schedule_cron, time_window_start, time_window_end, time_window_days,
		        ai_enabled, priority, tags, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, NOW(), NOW())
		 RETURNING created_at, updated_at`,
		rule.ID, rule.Name, rule.Description, rule.TriggerMetric, rule.Operator,
		rule.Threshold, rule.ActionType, rule.ActionConfig, rule.CooldownSeconds,
		rule.IsActive, rule.NodeID,
		rule.ScheduleType, rule.ScheduleCron, rule.TimeWindowStart, rule.TimeWindowEnd, rule.TimeWindowDays,
		rule.AIEnabled, rule.Priority, rule.Tags,
	).Scan(&rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create reflex rule: %w", err)
	}
	return nil
}

func (r *pgReflexRuleRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ReflexRule, error) {
	var rule model.ReflexRule
	err := r.db.QueryRow(ctx,
		`SELECT id, name, COALESCE(description, ''), trigger_metric, operator, threshold,
		        action_type, action_config, cooldown_seconds, is_active, node_id,
		        last_triggered_at, trigger_count,
		        schedule_type, schedule_cron, time_window_start, time_window_end, time_window_days,
		        ai_enabled, ai_severity, ai_recommendation, priority, tags,
		        created_at, updated_at
		 FROM reflex_rules WHERE id = $1`, id,
	).Scan(&rule.ID, &rule.Name, &rule.Description, &rule.TriggerMetric, &rule.Operator,
		&rule.Threshold, &rule.ActionType, &rule.ActionConfig, &rule.CooldownSeconds,
		&rule.IsActive, &rule.NodeID, &rule.LastTriggeredAt, &rule.TriggerCount,
		&rule.ScheduleType, &rule.ScheduleCron, &rule.TimeWindowStart, &rule.TimeWindowEnd, &rule.TimeWindowDays,
		&rule.AIEnabled, &rule.AISeverity, &rule.AIRecommendation, &rule.Priority, &rule.Tags,
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
		`SELECT id, name, COALESCE(description, ''), trigger_metric, operator, threshold,
		        action_type, action_config, cooldown_seconds, is_active, node_id,
		        last_triggered_at, trigger_count,
		        schedule_type, schedule_cron, time_window_start, time_window_end, time_window_days,
		        ai_enabled, ai_severity, ai_recommendation, priority, tags,
		        created_at, updated_at
		 FROM reflex_rules ORDER BY priority ASC, name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list reflex rules: %w", err)
	}
	defer rows.Close()
	return r.scanRules(rows)
}

func (r *pgReflexRuleRepository) ListActive(ctx context.Context) ([]model.ReflexRule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, COALESCE(description, ''), trigger_metric, operator, threshold,
		        action_type, action_config, cooldown_seconds, is_active, node_id,
		        last_triggered_at, trigger_count,
		        schedule_type, schedule_cron, time_window_start, time_window_end, time_window_days,
		        ai_enabled, ai_severity, ai_recommendation, priority, tags,
		        created_at, updated_at
		 FROM reflex_rules WHERE is_active = true ORDER BY priority ASC, name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list active reflex rules: %w", err)
	}
	defer rows.Close()
	return r.scanRules(rows)
}

func (r *pgReflexRuleRepository) Update(ctx context.Context, rule *model.ReflexRule) error {
	_, err := r.db.Exec(ctx,
		`UPDATE reflex_rules SET name=$1, description=$2, trigger_metric=$3, operator=$4, threshold=$5,
		        action_type=$6, action_config=$7, cooldown_seconds=$8, is_active=$9, node_id=$10,
		        schedule_type=$11, schedule_cron=$12, time_window_start=$13, time_window_end=$14,
		        time_window_days=$15, ai_enabled=$16, priority=$17, tags=$18, updated_at=NOW()
		 WHERE id=$19`,
		rule.Name, rule.Description, rule.TriggerMetric, rule.Operator, rule.Threshold,
		rule.ActionType, rule.ActionConfig, rule.CooldownSeconds, rule.IsActive, rule.NodeID,
		rule.ScheduleType, rule.ScheduleCron, rule.TimeWindowStart, rule.TimeWindowEnd,
		rule.TimeWindowDays, rule.AIEnabled, rule.Priority, rule.Tags, rule.ID,
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
			&rule.TriggerCount,
			&rule.ScheduleType, &rule.ScheduleCron, &rule.TimeWindowStart, &rule.TimeWindowEnd, &rule.TimeWindowDays,
			&rule.AIEnabled, &rule.AISeverity, &rule.AIRecommendation, &rule.Priority, &rule.Tags,
			&rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan reflex rule: %w", err)
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}
