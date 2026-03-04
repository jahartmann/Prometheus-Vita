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

type BriefingRepository interface {
	Create(ctx context.Context, briefing *model.MorningBriefing) error
	GetLatest(ctx context.Context) (*model.MorningBriefing, error)
	List(ctx context.Context, limit int) ([]model.MorningBriefing, error)
}

type pgBriefingRepository struct {
	db *pgxpool.Pool
}

func NewBriefingRepository(db *pgxpool.Pool) BriefingRepository {
	return &pgBriefingRepository{db: db}
}

func (r *pgBriefingRepository) Create(ctx context.Context, briefing *model.MorningBriefing) error {
	briefing.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO morning_briefings (id, summary, data, generated_at)
		 VALUES ($1, $2, $3, NOW())`,
		briefing.ID, briefing.Summary, briefing.Data,
	)
	if err != nil {
		return fmt.Errorf("create briefing: %w", err)
	}
	return nil
}

func (r *pgBriefingRepository) GetLatest(ctx context.Context) (*model.MorningBriefing, error) {
	var b model.MorningBriefing
	err := r.db.QueryRow(ctx,
		`SELECT id, summary, data, generated_at
		 FROM morning_briefings ORDER BY generated_at DESC LIMIT 1`,
	).Scan(&b.ID, &b.Summary, &b.Data, &b.GeneratedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get latest briefing: %w", err)
	}
	return &b, nil
}

func (r *pgBriefingRepository) List(ctx context.Context, limit int) ([]model.MorningBriefing, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, summary, data, generated_at
		 FROM morning_briefings ORDER BY generated_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list briefings: %w", err)
	}
	defer rows.Close()

	var briefings []model.MorningBriefing
	for rows.Next() {
		var b model.MorningBriefing
		if err := rows.Scan(&b.ID, &b.Summary, &b.Data, &b.GeneratedAt); err != nil {
			return nil, fmt.Errorf("scan briefing: %w", err)
		}
		briefings = append(briefings, b)
	}
	return briefings, rows.Err()
}
