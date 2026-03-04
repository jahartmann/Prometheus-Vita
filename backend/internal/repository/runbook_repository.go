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

type RunbookRepository interface {
	Create(ctx context.Context, runbook *model.RecoveryRunbook) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.RecoveryRunbook, error)
	ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.RecoveryRunbook, error)
	ListTemplates(ctx context.Context) ([]model.RecoveryRunbook, error)
	Update(ctx context.Context, runbook *model.RecoveryRunbook) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type pgRunbookRepository struct {
	db *pgxpool.Pool
}

func NewRunbookRepository(db *pgxpool.Pool) RunbookRepository {
	return &pgRunbookRepository{db: db}
}

func (r *pgRunbookRepository) Create(ctx context.Context, runbook *model.RecoveryRunbook) error {
	runbook.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO recovery_runbooks (id, node_id, title, scenario, steps, is_template, generated_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())`,
		runbook.ID, runbook.NodeID, runbook.Title, runbook.Scenario,
		runbook.Steps, runbook.IsTemplate,
	)
	if err != nil {
		return fmt.Errorf("create recovery runbook: %w", err)
	}
	return nil
}

func (r *pgRunbookRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.RecoveryRunbook, error) {
	var rb model.RecoveryRunbook
	err := r.db.QueryRow(ctx,
		`SELECT id, node_id, title, scenario, steps, is_template, generated_at, updated_at
		 FROM recovery_runbooks WHERE id = $1`, id,
	).Scan(&rb.ID, &rb.NodeID, &rb.Title, &rb.Scenario,
		&rb.Steps, &rb.IsTemplate, &rb.GeneratedAt, &rb.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get runbook by id: %w", err)
	}
	return &rb, nil
}

func (r *pgRunbookRepository) ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.RecoveryRunbook, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, title, scenario, steps, is_template, generated_at, updated_at
		 FROM recovery_runbooks WHERE node_id = $1 ORDER BY generated_at DESC`, nodeID)
	if err != nil {
		return nil, fmt.Errorf("list runbooks by node: %w", err)
	}
	defer rows.Close()

	var runbooks []model.RecoveryRunbook
	for rows.Next() {
		var rb model.RecoveryRunbook
		if err := rows.Scan(&rb.ID, &rb.NodeID, &rb.Title, &rb.Scenario,
			&rb.Steps, &rb.IsTemplate, &rb.GeneratedAt, &rb.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan runbook: %w", err)
		}
		runbooks = append(runbooks, rb)
	}
	return runbooks, rows.Err()
}

func (r *pgRunbookRepository) ListTemplates(ctx context.Context) ([]model.RecoveryRunbook, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, title, scenario, steps, is_template, generated_at, updated_at
		 FROM recovery_runbooks WHERE is_template = true ORDER BY scenario ASC`)
	if err != nil {
		return nil, fmt.Errorf("list runbook templates: %w", err)
	}
	defer rows.Close()

	var runbooks []model.RecoveryRunbook
	for rows.Next() {
		var rb model.RecoveryRunbook
		if err := rows.Scan(&rb.ID, &rb.NodeID, &rb.Title, &rb.Scenario,
			&rb.Steps, &rb.IsTemplate, &rb.GeneratedAt, &rb.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan runbook template: %w", err)
		}
		runbooks = append(runbooks, rb)
	}
	return runbooks, rows.Err()
}

func (r *pgRunbookRepository) Update(ctx context.Context, runbook *model.RecoveryRunbook) error {
	_, err := r.db.Exec(ctx,
		`UPDATE recovery_runbooks SET title=$1, steps=$2, updated_at=NOW() WHERE id=$3`,
		runbook.Title, runbook.Steps, runbook.ID,
	)
	if err != nil {
		return fmt.Errorf("update runbook: %w", err)
	}
	return nil
}

func (r *pgRunbookRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM recovery_runbooks WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("delete runbook: %w", err)
	}
	return nil
}
