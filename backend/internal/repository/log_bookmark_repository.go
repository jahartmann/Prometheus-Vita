package repository

import (
	"context"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LogBookmarkRepository interface {
	Create(ctx context.Context, bookmark *model.LogBookmark) error
	ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.LogBookmark, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type pgLogBookmarkRepository struct {
	db *pgxpool.Pool
}

func NewLogBookmarkRepository(db *pgxpool.Pool) LogBookmarkRepository {
	return &pgLogBookmarkRepository{db: db}
}

func (r *pgLogBookmarkRepository) Create(ctx context.Context, bookmark *model.LogBookmark) error {
	bookmark.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO log_bookmarks (id, node_id, anomaly_id, log_entry_json, user_note, created_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())`,
		bookmark.ID, bookmark.NodeID, bookmark.AnomalyID, bookmark.LogEntryJSON, bookmark.UserNote,
	)
	if err != nil {
		return fmt.Errorf("create log bookmark: %w", err)
	}
	return nil
}

func (r *pgLogBookmarkRepository) ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.LogBookmark, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, anomaly_id, log_entry_json, user_note, created_at
		 FROM log_bookmarks WHERE node_id = $1 ORDER BY created_at DESC`,
		nodeID)
	if err != nil {
		return nil, fmt.Errorf("list log bookmarks: %w", err)
	}
	defer rows.Close()

	var bookmarks []model.LogBookmark
	for rows.Next() {
		var b model.LogBookmark
		if err := rows.Scan(&b.ID, &b.NodeID, &b.AnomalyID, &b.LogEntryJSON, &b.UserNote, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan log bookmark: %w", err)
		}
		bookmarks = append(bookmarks, b)
	}
	return bookmarks, rows.Err()
}

func (r *pgLogBookmarkRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM log_bookmarks WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete log bookmark: %w", err)
	}
	return nil
}
