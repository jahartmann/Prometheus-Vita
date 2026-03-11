package repository

import (
	"context"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VMPermissionRepository interface {
	Create(ctx context.Context, perm *model.VMPermission) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.VMPermission, error)
	List(ctx context.Context) ([]model.VMPermission, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]model.VMPermission, error)
	ListByTarget(ctx context.Context, targetType, targetID string, nodeID uuid.UUID) ([]model.VMPermission, error)
	Update(ctx context.Context, perm *model.VMPermission) error
	Delete(ctx context.Context, id uuid.UUID) error
	Upsert(ctx context.Context, perm *model.VMPermission) error
	HasPermission(ctx context.Context, userID uuid.UUID, nodeID uuid.UUID, vmid string, permission string) (bool, error)
	GetEffectivePermissions(ctx context.Context, userID, nodeID uuid.UUID, vmid int) ([]string, error)
}

type pgVMPermissionRepository struct {
	db *pgxpool.Pool
}

func NewVMPermissionRepository(db *pgxpool.Pool) VMPermissionRepository {
	return &pgVMPermissionRepository{db: db}
}

func (r *pgVMPermissionRepository) Create(ctx context.Context, perm *model.VMPermission) error {
	perm.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO vm_permissions (id, user_id, target_type, target_id, node_id, permissions, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())`,
		perm.ID, perm.UserID, perm.TargetType, perm.TargetID, perm.NodeID, perm.Permissions, perm.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("create vm permission: %w", err)
	}
	return nil
}

func (r *pgVMPermissionRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.VMPermission, error) {
	var p model.VMPermission
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, target_type, target_id, node_id, permissions, created_by, created_at, updated_at
		 FROM vm_permissions WHERE id = $1`, id,
	).Scan(&p.ID, &p.UserID, &p.TargetType, &p.TargetID, &p.NodeID, &p.Permissions, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get vm permission: %w", err)
	}
	return &p, nil
}

func (r *pgVMPermissionRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.VMPermission, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, target_type, target_id, node_id, permissions, created_by, created_at, updated_at
		 FROM vm_permissions WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("list vm permissions by user: %w", err)
	}
	defer rows.Close()
	var perms []model.VMPermission
	for rows.Next() {
		var p model.VMPermission
		if err := rows.Scan(&p.ID, &p.UserID, &p.TargetType, &p.TargetID, &p.NodeID, &p.Permissions, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan vm permission: %w", err)
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func (r *pgVMPermissionRepository) ListByTarget(ctx context.Context, targetType, targetID string, nodeID uuid.UUID) ([]model.VMPermission, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, target_type, target_id, node_id, permissions, created_by, created_at, updated_at
		 FROM vm_permissions WHERE target_type = $1 AND target_id = $2 AND node_id = $3 ORDER BY created_at`,
		targetType, targetID, nodeID)
	if err != nil {
		return nil, fmt.Errorf("list vm permissions by target: %w", err)
	}
	defer rows.Close()
	var perms []model.VMPermission
	for rows.Next() {
		var p model.VMPermission
		if err := rows.Scan(&p.ID, &p.UserID, &p.TargetType, &p.TargetID, &p.NodeID, &p.Permissions, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan vm permission: %w", err)
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func (r *pgVMPermissionRepository) Update(ctx context.Context, perm *model.VMPermission) error {
	_, err := r.db.Exec(ctx,
		`UPDATE vm_permissions SET permissions = $1, updated_at = NOW() WHERE id = $2`,
		perm.Permissions, perm.ID)
	if err != nil {
		return fmt.Errorf("update vm permission: %w", err)
	}
	return nil
}

func (r *pgVMPermissionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM vm_permissions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete vm permission: %w", err)
	}
	return nil
}

func (r *pgVMPermissionRepository) List(ctx context.Context) ([]model.VMPermission, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, target_type, target_id, node_id, permissions, created_by, created_at, updated_at
		 FROM vm_permissions ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list vm permissions: %w", err)
	}
	defer rows.Close()
	var perms []model.VMPermission
	for rows.Next() {
		var p model.VMPermission
		if err := rows.Scan(&p.ID, &p.UserID, &p.TargetType, &p.TargetID, &p.NodeID, &p.Permissions, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan vm permission: %w", err)
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func (r *pgVMPermissionRepository) Upsert(ctx context.Context, perm *model.VMPermission) error {
	if perm.ID == uuid.Nil {
		perm.ID = uuid.New()
	}
	err := r.db.QueryRow(ctx,
		`INSERT INTO vm_permissions (id, user_id, target_type, target_id, node_id, permissions, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		 ON CONFLICT (user_id, target_type, target_id, node_id)
		 DO UPDATE SET permissions = EXCLUDED.permissions, updated_at = NOW()
		 RETURNING id, created_at, updated_at`,
		perm.ID, perm.UserID, perm.TargetType, perm.TargetID, perm.NodeID, perm.Permissions, perm.CreatedBy,
	).Scan(&perm.ID, &perm.CreatedAt, &perm.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert vm permission: %w", err)
	}
	return nil
}

// HasPermission checks if a user has a specific permission for a VM.
// It checks direct VM permissions AND group-based permissions (via vm_group_members).
func (r *pgVMPermissionRepository) HasPermission(ctx context.Context, userID uuid.UUID, nodeID uuid.UUID, vmid string, permission string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(
			-- Direct VM permission
			SELECT 1 FROM vm_permissions
			WHERE user_id = $1 AND node_id = $2
			  AND target_type = 'vm' AND target_id = $3
			  AND $4 = ANY(permissions)
			UNION
			-- Group-based permission: user has permission on a group that contains this VM
			SELECT 1 FROM vm_permissions vp
			JOIN vm_group_members gm ON vp.target_id = gm.group_id::text
			WHERE vp.user_id = $1 AND vp.target_type = 'group'
			  AND gm.node_id = $2 AND gm.vmid = $3::integer
			  AND $4 = ANY(vp.permissions)
		)`, userID, nodeID, vmid, permission,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check vm permission: %w", err)
	}
	return exists, nil
}

// GetEffectivePermissions returns all permissions a user has for a specific VM,
// combining direct and group-inherited permissions.
func (r *pgVMPermissionRepository) GetEffectivePermissions(ctx context.Context, userID, nodeID uuid.UUID, vmid int) ([]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT DISTINCT unnest(permissions) FROM (
			-- Direct VM permissions
			SELECT permissions FROM vm_permissions
			WHERE user_id = $1 AND target_type = 'vm' AND target_id = $3::text AND node_id = $2
			UNION ALL
			-- Group-based permissions
			SELECT vp.permissions FROM vm_permissions vp
			JOIN vm_group_members gm ON vp.target_id = gm.group_id::text
			WHERE vp.user_id = $1 AND vp.target_type = 'group'
			  AND gm.node_id = $2 AND gm.vmid = $3
		) sub`, userID, nodeID, vmid,
	)
	if err != nil {
		return nil, fmt.Errorf("get effective vm permissions: %w", err)
	}
	defer rows.Close()

	var perms []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("scan permission: %w", err)
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}
