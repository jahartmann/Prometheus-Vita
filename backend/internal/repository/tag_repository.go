package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TagRepository interface {
	Create(ctx context.Context, tag *model.Tag) error
	List(ctx context.Context) ([]model.Tag, error)
	Delete(ctx context.Context, id uuid.UUID) error
	AddToNode(ctx context.Context, nodeID, tagID uuid.UUID) error
	RemoveFromNode(ctx context.Context, nodeID, tagID uuid.UUID) error
	GetByNode(ctx context.Context, nodeID uuid.UUID) ([]model.Tag, error)
	GetNodesByTag(ctx context.Context, tagID uuid.UUID) ([]uuid.UUID, error)

	// VM tag methods
	AddToVM(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string, tagID uuid.UUID) error
	RemoveFromVM(ctx context.Context, nodeID uuid.UUID, vmid int, tagID uuid.UUID) error
	GetByVM(ctx context.Context, nodeID uuid.UUID, vmid int) ([]model.Tag, error)
	GetVMsByTag(ctx context.Context, tagID uuid.UUID) ([]model.VMTag, error)
	BulkAddToVMs(ctx context.Context, vmTags []model.VMTag) error
	BulkRemoveTagFromVMs(ctx context.Context, tagID uuid.UUID, nodeID uuid.UUID, vmids []int) error
}

type pgTagRepository struct {
	db *pgxpool.Pool
}

func NewTagRepository(db *pgxpool.Pool) TagRepository {
	return &pgTagRepository{db: db}
}

func (r *pgTagRepository) Create(ctx context.Context, tag *model.Tag) error {
	tag.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO tags (id, name, color, category, created_at)
		 VALUES ($1, $2, $3, $4, NOW())`,
		tag.ID, tag.Name, tag.Color, tag.Category,
	)
	if err != nil {
		return fmt.Errorf("create tag: %w", err)
	}
	return nil
}

func (r *pgTagRepository) List(ctx context.Context) ([]model.Tag, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, color, category, created_at
		 FROM tags ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	var tags []model.Tag
	for rows.Next() {
		var t model.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.Category, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (r *pgTagRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM tags WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}
	return nil
}

func (r *pgTagRepository) AddToNode(ctx context.Context, nodeID, tagID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO node_tags (node_id, tag_id) VALUES ($1, $2)
		 ON CONFLICT DO NOTHING`,
		nodeID, tagID,
	)
	if err != nil {
		return fmt.Errorf("add tag to node: %w", err)
	}
	return nil
}

func (r *pgTagRepository) RemoveFromNode(ctx context.Context, nodeID, tagID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		"DELETE FROM node_tags WHERE node_id=$1 AND tag_id=$2",
		nodeID, tagID,
	)
	if err != nil {
		return fmt.Errorf("remove tag from node: %w", err)
	}
	return nil
}

func (r *pgTagRepository) GetByNode(ctx context.Context, nodeID uuid.UUID) ([]model.Tag, error) {
	rows, err := r.db.Query(ctx,
		`SELECT t.id, t.name, t.color, t.category, t.created_at
		 FROM tags t JOIN node_tags nt ON t.id = nt.tag_id
		 WHERE nt.node_id = $1 ORDER BY t.name ASC`, nodeID)
	if err != nil {
		return nil, fmt.Errorf("get tags by node: %w", err)
	}
	defer rows.Close()

	var tags []model.Tag
	for rows.Next() {
		var t model.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.Category, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan node tag: %w", err)
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (r *pgTagRepository) GetNodesByTag(ctx context.Context, tagID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx,
		"SELECT node_id FROM node_tags WHERE tag_id = $1", tagID)
	if err != nil {
		return nil, fmt.Errorf("get nodes by tag: %w", err)
	}
	defer rows.Close()

	var nodeIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan node id: %w", err)
		}
		nodeIDs = append(nodeIDs, id)
	}
	return nodeIDs, rows.Err()
}

func (r *pgTagRepository) AddToVM(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string, tagID uuid.UUID) error {
	if vmType == "" {
		vmType = "qemu"
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO vm_tags (node_id, vmid, vm_type, tag_id) VALUES ($1, $2, $3, $4)
		 ON CONFLICT DO NOTHING`,
		nodeID, vmid, vmType, tagID,
	)
	if err != nil {
		return fmt.Errorf("add tag to vm: %w", err)
	}
	return nil
}

func (r *pgTagRepository) RemoveFromVM(ctx context.Context, nodeID uuid.UUID, vmid int, tagID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		"DELETE FROM vm_tags WHERE node_id=$1 AND vmid=$2 AND tag_id=$3",
		nodeID, vmid, tagID,
	)
	if err != nil {
		return fmt.Errorf("remove tag from vm: %w", err)
	}
	return nil
}

func (r *pgTagRepository) GetByVM(ctx context.Context, nodeID uuid.UUID, vmid int) ([]model.Tag, error) {
	rows, err := r.db.Query(ctx,
		`SELECT t.id, t.name, t.color, COALESCE(t.category, ''), t.created_at
		 FROM tags t JOIN vm_tags vt ON t.id = vt.tag_id
		 WHERE vt.node_id = $1 AND vt.vmid = $2 ORDER BY t.name ASC`, nodeID, vmid)
	if err != nil {
		return nil, fmt.Errorf("get tags by vm: %w", err)
	}
	defer rows.Close()

	var tags []model.Tag
	for rows.Next() {
		var t model.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.Category, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan vm tag: %w", err)
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (r *pgTagRepository) GetVMsByTag(ctx context.Context, tagID uuid.UUID) ([]model.VMTag, error) {
	rows, err := r.db.Query(ctx,
		`SELECT node_id, vmid, vm_type, tag_id, created_at
		 FROM vm_tags WHERE tag_id = $1 ORDER BY node_id, vmid`, tagID)
	if err != nil {
		return nil, fmt.Errorf("get vms by tag: %w", err)
	}
	defer rows.Close()

	var vmTags []model.VMTag
	for rows.Next() {
		var vt model.VMTag
		if err := rows.Scan(&vt.NodeID, &vt.VMID, &vt.VMType, &vt.TagID, &vt.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan vm tag: %w", err)
		}
		vmTags = append(vmTags, vt)
	}
	return vmTags, rows.Err()
}

func (r *pgTagRepository) BulkAddToVMs(ctx context.Context, vmTags []model.VMTag) error {
	if len(vmTags) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("INSERT INTO vm_tags (node_id, vmid, vm_type, tag_id) VALUES ")
	args := make([]interface{}, 0, len(vmTags)*4)
	for i, vt := range vmTags {
		if i > 0 {
			sb.WriteString(", ")
		}
		base := i * 4
		sb.WriteString(fmt.Sprintf("($%d, $%d, $%d, $%d)", base+1, base+2, base+3, base+4))
		vmType := vt.VMType
		if vmType == "" {
			vmType = "qemu"
		}
		args = append(args, vt.NodeID, vt.VMID, vmType, vt.TagID)
	}
	sb.WriteString(" ON CONFLICT DO NOTHING")

	_, err := r.db.Exec(ctx, sb.String(), args...)
	if err != nil {
		return fmt.Errorf("bulk add tags to vms: %w", err)
	}
	return nil
}

func (r *pgTagRepository) BulkRemoveTagFromVMs(ctx context.Context, tagID uuid.UUID, nodeID uuid.UUID, vmids []int) error {
	if len(vmids) == 0 {
		return nil
	}

	args := []interface{}{tagID, nodeID}
	placeholders := make([]string, len(vmids))
	for i, vmid := range vmids {
		args = append(args, vmid)
		placeholders[i] = fmt.Sprintf("$%d", i+3)
	}

	query := fmt.Sprintf(
		"DELETE FROM vm_tags WHERE tag_id=$1 AND node_id=$2 AND vmid IN (%s)",
		strings.Join(placeholders, ", "),
	)

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("bulk remove tag from vms: %w", err)
	}
	return nil
}
