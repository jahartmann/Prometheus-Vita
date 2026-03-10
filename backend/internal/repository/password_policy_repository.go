package repository

import (
	"context"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PasswordPolicyRepository interface {
	Get(ctx context.Context) (*model.PasswordPolicy, error)
	Update(ctx context.Context, policy *model.PasswordPolicy) error
}

type pgPasswordPolicyRepository struct {
	db *pgxpool.Pool
}

func NewPasswordPolicyRepository(db *pgxpool.Pool) PasswordPolicyRepository {
	return &pgPasswordPolicyRepository{db: db}
}

func (r *pgPasswordPolicyRepository) Get(ctx context.Context) (*model.PasswordPolicy, error) {
	var p model.PasswordPolicy
	err := r.db.QueryRow(ctx,
		`SELECT id, min_length, require_uppercase, require_lowercase, require_digit, require_special, max_length, disallow_username, updated_at, updated_by
		 FROM password_policy LIMIT 1`,
	).Scan(&p.ID, &p.MinLength, &p.RequireUppercase, &p.RequireLowercase, &p.RequireDigit, &p.RequireSpecial, &p.MaxLength, &p.DisallowUsername, &p.UpdatedAt, &p.UpdatedBy)
	if err != nil {
		return nil, fmt.Errorf("get password policy: %w", err)
	}
	return &p, nil
}

func (r *pgPasswordPolicyRepository) Update(ctx context.Context, policy *model.PasswordPolicy) error {
	_, err := r.db.Exec(ctx,
		`UPDATE password_policy SET min_length=$1, require_uppercase=$2, require_lowercase=$3, require_digit=$4, require_special=$5, max_length=$6, disallow_username=$7, updated_at=NOW(), updated_by=$8 WHERE id=$9`,
		policy.MinLength, policy.RequireUppercase, policy.RequireLowercase, policy.RequireDigit, policy.RequireSpecial, policy.MaxLength, policy.DisallowUsername, policy.UpdatedBy, policy.ID,
	)
	if err != nil {
		return fmt.Errorf("update password policy: %w", err)
	}
	return nil
}
