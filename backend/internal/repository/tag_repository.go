package repository

import (
	"context"
	"fmt"

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
