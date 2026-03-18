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

var (
	ErrNotFound        = errors.New("not found")
	ErrAlreadyResolved = errors.New("already resolved")
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	List(ctx context.Context) ([]model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	Count(ctx context.Context) (int, error)
	CountByRole(ctx context.Context, role model.UserRole) (int, error)
}

type pgUserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &pgUserRepository{db: db}
}

func (r *pgUserRepository) Create(ctx context.Context, user *model.User) error {
	user.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO users (id, username, email, password_hash, role, is_active, autonomy_level, must_change_password, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())`,
		user.ID, user.Username, user.Email, user.PasswordHash, user.Role, user.IsActive, user.AutonomyLevel, user.MustChangePassword,
	)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *pgUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var u model.User
	err := r.db.QueryRow(ctx,
		`SELECT id, username, email, password_hash, role, is_active, autonomy_level, created_at, updated_at, last_login, must_change_password
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.AutonomyLevel,
		&u.CreatedAt, &u.UpdatedAt, &u.LastLogin, &u.MustChangePassword)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}

func (r *pgUserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var u model.User
	err := r.db.QueryRow(ctx,
		`SELECT id, username, email, password_hash, role, is_active, autonomy_level, created_at, updated_at, last_login, must_change_password
		 FROM users WHERE username = $1`, username,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.AutonomyLevel,
		&u.CreatedAt, &u.UpdatedAt, &u.LastLogin, &u.MustChangePassword)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return &u, nil
}

func (r *pgUserRepository) List(ctx context.Context) ([]model.User, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, username, email, password_hash, role, is_active, autonomy_level, created_at, updated_at, last_login, must_change_password
		 FROM users ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.AutonomyLevel,
			&u.CreatedAt, &u.UpdatedAt, &u.LastLogin, &u.MustChangePassword); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *pgUserRepository) Update(ctx context.Context, user *model.User) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET username=$1, email=$2, password_hash=$3, role=$4, is_active=$5, autonomy_level=$6, must_change_password=$7, updated_at=NOW()
		 WHERE id=$8`,
		user.Username, user.Email, user.PasswordHash, user.Role, user.IsActive, user.AutonomyLevel, user.MustChangePassword, user.ID,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (r *pgUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM users WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (r *pgUserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "UPDATE users SET last_login=NOW() WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("update last login: %w", err)
	}
	return nil
}

func (r *pgUserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	_, err := r.db.Exec(ctx, "UPDATE users SET password_hash=$1, updated_at=NOW() WHERE id=$2", passwordHash, id)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

func (r *pgUserRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

func (r *pgUserRepository) CountByRole(ctx context.Context, role model.UserRole) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE role=$1 AND is_active=true", role).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count users by role: %w", err)
	}
	return count, nil
}
