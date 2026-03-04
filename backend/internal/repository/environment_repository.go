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

type EnvironmentRepository interface {
	Create(ctx context.Context, env *model.Environment) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Environment, error)
	List(ctx context.Context) ([]model.Environment, error)
	Update(ctx context.Context, env *model.Environment) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type pgEnvironmentRepository struct {
	db *pgxpool.Pool
}

func NewEnvironmentRepository(db *pgxpool.Pool) EnvironmentRepository {
	return &pgEnvironmentRepository{db: db}
}

func (r *pgEnvironmentRepository) Create(ctx context.Context, env *model.Environment) error {
	env.ID = uuid.New()
	err := r.db.QueryRow(ctx,
		`INSERT INTO environments (id, name, description, color, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, NOW(), NOW()) RETURNING created_at, updated_at`,
		env.ID, env.Name, env.Description, env.Color,
	).Scan(&env.CreatedAt, &env.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create environment: %w", err)
	}
	return nil
}

func (r *pgEnvironmentRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Environment, error) {
	var e model.Environment
	err := r.db.QueryRow(ctx,
		`SELECT id, name, description, color, created_at, updated_at FROM environments WHERE id = $1`, id,
	).Scan(&e.ID, &e.Name, &e.Description, &e.Color, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get environment: %w", err)
	}
	return &e, nil
}

func (r *pgEnvironmentRepository) List(ctx context.Context) ([]model.Environment, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, description, color, created_at, updated_at FROM environments ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list environments: %w", err)
	}
	defer rows.Close()

	var envs []model.Environment
	for rows.Next() {
		var e model.Environment
		if err := rows.Scan(&e.ID, &e.Name, &e.Description, &e.Color, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan environment: %w", err)
		}
		envs = append(envs, e)
	}
	return envs, rows.Err()
}

func (r *pgEnvironmentRepository) Update(ctx context.Context, env *model.Environment) error {
	_, err := r.db.Exec(ctx,
		`UPDATE environments SET name=$1, description=$2, color=$3, updated_at=NOW() WHERE id=$4`,
		env.Name, env.Description, env.Color, env.ID,
	)
	if err != nil {
		return fmt.Errorf("update environment: %w", err)
	}
	return nil
}

func (r *pgEnvironmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM environments WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("delete environment: %w", err)
	}
	return nil
}
