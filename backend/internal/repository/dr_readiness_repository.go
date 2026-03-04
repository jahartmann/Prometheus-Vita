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

type DRReadinessRepository interface {
	Create(ctx context.Context, score *model.DRReadinessScore) error
	GetLatest(ctx context.Context, nodeID uuid.UUID) (*model.DRReadinessScore, error)
	ListAll(ctx context.Context) ([]model.DRReadinessScore, error)
}

type pgDRReadinessRepository struct {
	db *pgxpool.Pool
}

func NewDRReadinessRepository(db *pgxpool.Pool) DRReadinessRepository {
	return &pgDRReadinessRepository{db: db}
}

func (r *pgDRReadinessRepository) Create(ctx context.Context, score *model.DRReadinessScore) error {
	score.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO dr_readiness_scores (id, node_id, overall_score, backup_score,
		        profile_score, config_score, details, calculated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`,
		score.ID, score.NodeID, score.OverallScore,
		score.BackupScore, score.ProfileScore, score.ConfigScore,
		score.Details,
	)
	if err != nil {
		return fmt.Errorf("create dr readiness score: %w", err)
	}
	return nil
}

func (r *pgDRReadinessRepository) GetLatest(ctx context.Context, nodeID uuid.UUID) (*model.DRReadinessScore, error) {
	var s model.DRReadinessScore
	err := r.db.QueryRow(ctx,
		`SELECT id, node_id, overall_score, backup_score, profile_score,
		        config_score, details, calculated_at
		 FROM dr_readiness_scores WHERE node_id = $1 ORDER BY calculated_at DESC LIMIT 1`, nodeID,
	).Scan(&s.ID, &s.NodeID, &s.OverallScore,
		&s.BackupScore, &s.ProfileScore, &s.ConfigScore,
		&s.Details, &s.CalculatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get latest dr readiness score: %w", err)
	}
	return &s, nil
}

func (r *pgDRReadinessRepository) ListAll(ctx context.Context) ([]model.DRReadinessScore, error) {
	rows, err := r.db.Query(ctx,
		`SELECT DISTINCT ON (node_id) id, node_id, overall_score, backup_score,
		        profile_score, config_score, details, calculated_at
		 FROM dr_readiness_scores ORDER BY node_id, calculated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list all dr readiness scores: %w", err)
	}
	defer rows.Close()

	var scores []model.DRReadinessScore
	for rows.Next() {
		var s model.DRReadinessScore
		if err := rows.Scan(&s.ID, &s.NodeID, &s.OverallScore,
			&s.BackupScore, &s.ProfileScore, &s.ConfigScore,
			&s.Details, &s.CalculatedAt); err != nil {
			return nil, fmt.Errorf("scan dr readiness score: %w", err)
		}
		scores = append(scores, s)
	}
	return scores, rows.Err()
}
