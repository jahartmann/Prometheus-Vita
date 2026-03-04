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

type EscalationPolicyRepository interface {
	Create(ctx context.Context, policy *model.EscalationPolicy) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.EscalationPolicy, error)
	List(ctx context.Context) ([]model.EscalationPolicy, error)
	Update(ctx context.Context, policy *model.EscalationPolicy) error
	Delete(ctx context.Context, id uuid.UUID) error
	CreateStep(ctx context.Context, step *model.EscalationStep) error
	ListStepsByPolicy(ctx context.Context, policyID uuid.UUID) ([]model.EscalationStep, error)
	DeleteStepsByPolicy(ctx context.Context, policyID uuid.UUID) error
}

type pgEscalationPolicyRepository struct {
	db *pgxpool.Pool
}

func NewEscalationPolicyRepository(db *pgxpool.Pool) EscalationPolicyRepository {
	return &pgEscalationPolicyRepository{db: db}
}

func (r *pgEscalationPolicyRepository) Create(ctx context.Context, policy *model.EscalationPolicy) error {
	policy.ID = uuid.New()
	err := r.db.QueryRow(ctx,
		`INSERT INTO escalation_policies (id, name, description, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, NOW(), NOW())
		 RETURNING created_at, updated_at`,
		policy.ID, policy.Name, policy.Description, policy.IsActive,
	).Scan(&policy.CreatedAt, &policy.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create escalation policy: %w", err)
	}
	return nil
}

func (r *pgEscalationPolicyRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.EscalationPolicy, error) {
	var p model.EscalationPolicy
	err := r.db.QueryRow(ctx,
		`SELECT id, name, COALESCE(description, ''), is_active, created_at, updated_at
		 FROM escalation_policies WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.Description, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get escalation policy: %w", err)
	}

	steps, err := r.ListStepsByPolicy(ctx, id)
	if err != nil {
		return nil, err
	}
	p.Steps = steps

	return &p, nil
}

func (r *pgEscalationPolicyRepository) List(ctx context.Context) ([]model.EscalationPolicy, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, COALESCE(description, ''), is_active, created_at, updated_at
		 FROM escalation_policies ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list escalation policies: %w", err)
	}
	defer rows.Close()

	var policies []model.EscalationPolicy
	for rows.Next() {
		var p model.EscalationPolicy
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan escalation policy: %w", err)
		}
		policies = append(policies, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load steps for each policy
	for i := range policies {
		steps, err := r.ListStepsByPolicy(ctx, policies[i].ID)
		if err != nil {
			return nil, err
		}
		policies[i].Steps = steps
	}

	return policies, nil
}

func (r *pgEscalationPolicyRepository) Update(ctx context.Context, policy *model.EscalationPolicy) error {
	_, err := r.db.Exec(ctx,
		`UPDATE escalation_policies SET name=$1, description=$2, is_active=$3, updated_at=NOW()
		 WHERE id=$4`,
		policy.Name, policy.Description, policy.IsActive, policy.ID,
	)
	if err != nil {
		return fmt.Errorf("update escalation policy: %w", err)
	}
	return nil
}

func (r *pgEscalationPolicyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM escalation_policies WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("delete escalation policy: %w", err)
	}
	return nil
}

func (r *pgEscalationPolicyRepository) CreateStep(ctx context.Context, step *model.EscalationStep) error {
	step.ID = uuid.New()
	err := r.db.QueryRow(ctx,
		`INSERT INTO escalation_steps (id, policy_id, step_order, delay_seconds, channel_ids, created_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())
		 RETURNING created_at`,
		step.ID, step.PolicyID, step.StepOrder, step.DelaySeconds, step.ChannelIDs,
	).Scan(&step.CreatedAt)
	if err != nil {
		return fmt.Errorf("create escalation step: %w", err)
	}
	return nil
}

func (r *pgEscalationPolicyRepository) ListStepsByPolicy(ctx context.Context, policyID uuid.UUID) ([]model.EscalationStep, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, policy_id, step_order, delay_seconds, channel_ids, created_at
		 FROM escalation_steps WHERE policy_id = $1
		 ORDER BY step_order ASC`, policyID)
	if err != nil {
		return nil, fmt.Errorf("list escalation steps: %w", err)
	}
	defer rows.Close()

	var steps []model.EscalationStep
	for rows.Next() {
		var s model.EscalationStep
		if err := rows.Scan(&s.ID, &s.PolicyID, &s.StepOrder, &s.DelaySeconds, &s.ChannelIDs, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan escalation step: %w", err)
		}
		steps = append(steps, s)
	}
	return steps, rows.Err()
}

func (r *pgEscalationPolicyRepository) DeleteStepsByPolicy(ctx context.Context, policyID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM escalation_steps WHERE policy_id=$1", policyID)
	if err != nil {
		return fmt.Errorf("delete escalation steps: %w", err)
	}
	return nil
}
