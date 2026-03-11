package repository

import (
	"context"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VMGroupRepository interface {
	Create(ctx context.Context, group *model.VMGroup) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.VMGroup, error)
	List(ctx context.Context) ([]model.VMGroup, error)
	Update(ctx context.Context, group *model.VMGroup) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListMembers(ctx context.Context, groupID uuid.UUID) ([]model.VMGroupMember, error)
	AddMember(ctx context.Context, member *model.VMGroupMember) error
	RemoveMember(ctx context.Context, groupID, nodeID uuid.UUID, vmid int) error
	GetGroupsForVM(ctx context.Context, nodeID uuid.UUID, vmid int) ([]model.VMGroup, error)
}

type pgVMGroupRepository struct {
	db *pgxpool.Pool
}

func NewVMGroupRepository(db *pgxpool.Pool) VMGroupRepository {
	return &pgVMGroupRepository{db: db}
}

func (r *pgVMGroupRepository) Create(ctx context.Context, group *model.VMGroup) error {
	group.ID = uuid.New()
	err := r.db.QueryRow(ctx,
		`INSERT INTO vm_groups (id, name, description, tag_filter, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		 RETURNING created_at, updated_at`,
		group.ID, group.Name, group.Description, group.TagFilter, group.CreatedBy,
	).Scan(&group.CreatedAt, &group.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create vm group: %w", err)
	}
	return nil
}

func (r *pgVMGroupRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.VMGroup, error) {
	var g model.VMGroup
	err := r.db.QueryRow(ctx,
		`SELECT g.id, g.name, COALESCE(g.description, ''), COALESCE(g.tag_filter, ''),
		        COALESCE(g.created_by, '00000000-0000-0000-0000-000000000000'), g.created_at, g.updated_at,
		        (SELECT COUNT(*) FROM vm_group_members m WHERE m.group_id = g.id) as member_count
		 FROM vm_groups g WHERE g.id = $1`, id,
	).Scan(&g.ID, &g.Name, &g.Description, &g.TagFilter, &g.CreatedBy, &g.CreatedAt, &g.UpdatedAt, &g.MemberCount)
	if err != nil {
		return nil, fmt.Errorf("get vm group: %w", err)
	}
	return &g, nil
}

func (r *pgVMGroupRepository) List(ctx context.Context) ([]model.VMGroup, error) {
	rows, err := r.db.Query(ctx,
		`SELECT g.id, g.name, COALESCE(g.description, ''), COALESCE(g.tag_filter, ''),
		        COALESCE(g.created_by, '00000000-0000-0000-0000-000000000000'), g.created_at, g.updated_at,
		        (SELECT COUNT(*) FROM vm_group_members m WHERE m.group_id = g.id) as member_count
		 FROM vm_groups g ORDER BY g.name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list vm groups: %w", err)
	}
	defer rows.Close()

	var groups []model.VMGroup
	for rows.Next() {
		var g model.VMGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.TagFilter, &g.CreatedBy, &g.CreatedAt, &g.UpdatedAt, &g.MemberCount); err != nil {
			return nil, fmt.Errorf("scan vm group: %w", err)
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func (r *pgVMGroupRepository) Update(ctx context.Context, group *model.VMGroup) error {
	_, err := r.db.Exec(ctx,
		`UPDATE vm_groups SET name = $2, description = $3, tag_filter = $4, updated_at = NOW()
		 WHERE id = $1`,
		group.ID, group.Name, group.Description, group.TagFilter,
	)
	if err != nil {
		return fmt.Errorf("update vm group: %w", err)
	}
	return nil
}

func (r *pgVMGroupRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM vm_groups WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete vm group: %w", err)
	}
	return nil
}

func (r *pgVMGroupRepository) ListMembers(ctx context.Context, groupID uuid.UUID) ([]model.VMGroupMember, error) {
	rows, err := r.db.Query(ctx,
		`SELECT group_id, node_id, vmid FROM vm_group_members
		 WHERE group_id = $1 ORDER BY node_id, vmid`, groupID)
	if err != nil {
		return nil, fmt.Errorf("list vm group members: %w", err)
	}
	defer rows.Close()

	var members []model.VMGroupMember
	for rows.Next() {
		var m model.VMGroupMember
		if err := rows.Scan(&m.GroupID, &m.NodeID, &m.VMID); err != nil {
			return nil, fmt.Errorf("scan vm group member: %w", err)
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *pgVMGroupRepository) AddMember(ctx context.Context, member *model.VMGroupMember) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO vm_group_members (group_id, node_id, vmid)
		 VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		member.GroupID, member.NodeID, member.VMID,
	)
	if err != nil {
		return fmt.Errorf("add vm group member: %w", err)
	}
	return nil
}

func (r *pgVMGroupRepository) RemoveMember(ctx context.Context, groupID, nodeID uuid.UUID, vmid int) error {
	_, err := r.db.Exec(ctx,
		"DELETE FROM vm_group_members WHERE group_id = $1 AND node_id = $2 AND vmid = $3",
		groupID, nodeID, vmid,
	)
	if err != nil {
		return fmt.Errorf("remove vm group member: %w", err)
	}
	return nil
}

func (r *pgVMGroupRepository) GetGroupsForVM(ctx context.Context, nodeID uuid.UUID, vmid int) ([]model.VMGroup, error) {
	rows, err := r.db.Query(ctx,
		`SELECT g.id, g.name, COALESCE(g.description, ''), COALESCE(g.tag_filter, ''),
		        COALESCE(g.created_by, '00000000-0000-0000-0000-000000000000'), g.created_at, g.updated_at,
		        (SELECT COUNT(*) FROM vm_group_members m2 WHERE m2.group_id = g.id) as member_count
		 FROM vm_groups g
		 JOIN vm_group_members m ON g.id = m.group_id
		 WHERE m.node_id = $1 AND m.vmid = $2
		 ORDER BY g.name ASC`, nodeID, vmid)
	if err != nil {
		return nil, fmt.Errorf("get groups for vm: %w", err)
	}
	defer rows.Close()

	var groups []model.VMGroup
	for rows.Next() {
		var g model.VMGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.TagFilter, &g.CreatedBy, &g.CreatedAt, &g.UpdatedAt, &g.MemberCount); err != nil {
			return nil, fmt.Errorf("scan vm group: %w", err)
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}
