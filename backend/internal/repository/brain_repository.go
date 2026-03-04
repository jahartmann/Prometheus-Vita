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

type BrainRepository interface {
	Create(ctx context.Context, entry *model.BrainEntry) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.BrainEntry, error)
	List(ctx context.Context) ([]model.BrainEntry, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Search(ctx context.Context, query string) ([]model.BrainEntry, error)
	IncrementAccessCount(ctx context.Context, id uuid.UUID) error
	DeleteLowRelevance(ctx context.Context, minScore float64, minAccessCount int) error
}

type pgBrainRepository struct {
	db *pgxpool.Pool
}

func NewBrainRepository(db *pgxpool.Pool) BrainRepository {
	return &pgBrainRepository{db: db}
}

func (r *pgBrainRepository) Create(ctx context.Context, entry *model.BrainEntry) error {
	entry.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO brain_entries (id, category, subject, content, metadata, relevance_score, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())`,
		entry.ID, entry.Category, entry.Subject, entry.Content,
		entry.Metadata, entry.RelevanceScore, entry.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("create brain entry: %w", err)
	}
	return nil
}

func (r *pgBrainRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.BrainEntry, error) {
	var e model.BrainEntry
	err := r.db.QueryRow(ctx,
		`SELECT id, category, subject, content, metadata, relevance_score, access_count,
		        last_accessed_at, created_by, created_at, updated_at
		 FROM brain_entries WHERE id = $1`, id,
	).Scan(&e.ID, &e.Category, &e.Subject, &e.Content, &e.Metadata,
		&e.RelevanceScore, &e.AccessCount, &e.LastAccessedAt,
		&e.CreatedBy, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get brain entry by id: %w", err)
	}
	return &e, nil
}

func (r *pgBrainRepository) List(ctx context.Context) ([]model.BrainEntry, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, category, subject, content, metadata, relevance_score, access_count,
		        last_accessed_at, created_by, created_at, updated_at
		 FROM brain_entries ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list brain entries: %w", err)
	}
	defer rows.Close()

	var entries []model.BrainEntry
	for rows.Next() {
		var e model.BrainEntry
		if err := rows.Scan(&e.ID, &e.Category, &e.Subject, &e.Content, &e.Metadata,
			&e.RelevanceScore, &e.AccessCount, &e.LastAccessedAt,
			&e.CreatedBy, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan brain entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *pgBrainRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM brain_entries WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("delete brain entry: %w", err)
	}
	return nil
}

func (r *pgBrainRepository) Search(ctx context.Context, query string) ([]model.BrainEntry, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, category, subject, content, metadata, relevance_score, access_count,
		        last_accessed_at, created_by, created_at, updated_at
		 FROM brain_entries
		 WHERE to_tsvector('german', subject || ' ' || content) @@ plainto_tsquery('german', $1)
		 ORDER BY ts_rank(to_tsvector('german', subject || ' ' || content), plainto_tsquery('german', $1)) DESC,
		          relevance_score DESC
		 LIMIT 20`, query)
	if err != nil {
		return nil, fmt.Errorf("search brain entries: %w", err)
	}
	defer rows.Close()

	var entries []model.BrainEntry
	for rows.Next() {
		var e model.BrainEntry
		if err := rows.Scan(&e.ID, &e.Category, &e.Subject, &e.Content, &e.Metadata,
			&e.RelevanceScore, &e.AccessCount, &e.LastAccessedAt,
			&e.CreatedBy, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan brain entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *pgBrainRepository) IncrementAccessCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE brain_entries SET access_count = access_count + 1, last_accessed_at = NOW(), updated_at = NOW()
		 WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("increment brain entry access count: %w", err)
	}
	return nil
}

func (r *pgBrainRepository) DeleteLowRelevance(ctx context.Context, minScore float64, minAccessCount int) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM brain_entries WHERE relevance_score < $1 AND access_count < $2`,
		minScore, minAccessCount)
	if err != nil {
		return fmt.Errorf("delete low relevance brain entries: %w", err)
	}
	return nil
}
