package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RolePermissionRepository interface {
	List(ctx context.Context) ([]model.RolePermissionOverride, error)
	Get(ctx context.Context, role model.UserRole) (*model.RolePermissionOverride, error)
	Update(ctx context.Context, role model.UserRole, permissions []model.Permission, updatedBy *uuid.UUID) (*model.RolePermissionOverride, error)
}

type pgRolePermissionRepository struct {
	db *pgxpool.Pool
}

func NewRolePermissionRepository(db *pgxpool.Pool) RolePermissionRepository {
	return &pgRolePermissionRepository{db: db}
}

func (r *pgRolePermissionRepository) List(ctx context.Context) ([]model.RolePermissionOverride, error) {
	rows, err := r.db.Query(ctx,
		`SELECT role, permissions, updated_at, updated_by
		 FROM role_permissions
		 ORDER BY CASE role WHEN 'admin' THEN 1 WHEN 'operator' THEN 2 WHEN 'viewer' THEN 3 ELSE 4 END`,
	)
	if err != nil {
		return nil, fmt.Errorf("list role permissions: %w", err)
	}
	defer rows.Close()

	var roles []model.RolePermissionOverride
	for rows.Next() {
		role, err := scanRolePermission(rows)
		if err != nil {
			return nil, err
		}
		roles = append(roles, *role)
	}
	return roles, rows.Err()
}

func (r *pgRolePermissionRepository) Get(ctx context.Context, role model.UserRole) (*model.RolePermissionOverride, error) {
	row := r.db.QueryRow(ctx,
		`SELECT role, permissions, updated_at, updated_by
		 FROM role_permissions
		 WHERE role = $1`,
		role,
	)
	permissions, err := scanRolePermission(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get role permissions: %w", err)
	}
	return permissions, nil
}

func (r *pgRolePermissionRepository) Update(ctx context.Context, role model.UserRole, permissions []model.Permission, updatedBy *uuid.UUID) (*model.RolePermissionOverride, error) {
	raw, err := json.Marshal(permissions)
	if err != nil {
		return nil, fmt.Errorf("marshal role permissions: %w", err)
	}

	row := r.db.QueryRow(ctx,
		`INSERT INTO role_permissions (role, permissions, updated_at, updated_by)
		 VALUES ($1, $2, NOW(), $3)
		 ON CONFLICT (role) DO UPDATE
		 SET permissions = EXCLUDED.permissions,
		     updated_at = NOW(),
		     updated_by = EXCLUDED.updated_by
		 RETURNING role, permissions, updated_at, updated_by`,
		role, json.RawMessage(raw), updatedBy,
	)

	updated, err := scanRolePermission(row)
	if err != nil {
		return nil, fmt.Errorf("update role permissions: %w", err)
	}
	return updated, nil
}

type rolePermissionScanner interface {
	Scan(dest ...any) error
}

func scanRolePermission(scanner rolePermissionScanner) (*model.RolePermissionOverride, error) {
	var role model.RolePermissionOverride
	var raw json.RawMessage
	if err := scanner.Scan(&role.Role, &raw, &role.UpdatedAt, &role.UpdatedBy); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &role.Permissions); err != nil {
		return nil, fmt.Errorf("decode role permissions: %w", err)
	}
	return &role, nil
}
