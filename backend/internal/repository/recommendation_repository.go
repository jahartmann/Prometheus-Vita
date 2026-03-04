package repository

import (
	"context"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RecommendationRepository interface {
	Create(ctx context.Context, rec *model.ResourceRecommendation) error
	ListByNode(ctx context.Context, nodeID uuid.UUID, limit int) ([]model.ResourceRecommendation, error)
	ListAll(ctx context.Context, limit int) ([]model.ResourceRecommendation, error)
	DeleteByNode(ctx context.Context, nodeID uuid.UUID) error
}

type pgRecommendationRepository struct {
	db *pgxpool.Pool
}

func NewRecommendationRepository(db *pgxpool.Pool) RecommendationRepository {
	return &pgRecommendationRepository{db: db}
}

func (r *pgRecommendationRepository) Create(ctx context.Context, rec *model.ResourceRecommendation) error {
	rec.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO resource_recommendations (id, node_id, vmid, vm_name, vm_type, resource_type, current_value, recommended_value, avg_usage, max_usage, recommendation_type, reason, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())`,
		rec.ID, rec.NodeID, rec.VMID, rec.VMName, rec.VMType, rec.ResourceType,
		rec.CurrentValue, rec.RecommendedValue, rec.AvgUsage, rec.MaxUsage,
		rec.RecommendationType, rec.Reason,
	)
	if err != nil {
		return fmt.Errorf("create recommendation: %w", err)
	}
	return nil
}

func (r *pgRecommendationRepository) ListByNode(ctx context.Context, nodeID uuid.UUID, limit int) ([]model.ResourceRecommendation, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, vmid, vm_name, vm_type, resource_type, current_value, recommended_value, avg_usage, max_usage, recommendation_type, COALESCE(reason, ''), created_at
		 FROM resource_recommendations WHERE node_id = $1 ORDER BY created_at DESC LIMIT $2`, nodeID, limit)
	if err != nil {
		return nil, fmt.Errorf("list recommendations by node: %w", err)
	}
	defer rows.Close()

	var recs []model.ResourceRecommendation
	for rows.Next() {
		var rec model.ResourceRecommendation
		if err := rows.Scan(&rec.ID, &rec.NodeID, &rec.VMID, &rec.VMName, &rec.VMType,
			&rec.ResourceType, &rec.CurrentValue, &rec.RecommendedValue, &rec.AvgUsage, &rec.MaxUsage,
			&rec.RecommendationType, &rec.Reason, &rec.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan recommendation: %w", err)
		}
		recs = append(recs, rec)
	}
	return recs, rows.Err()
}

func (r *pgRecommendationRepository) ListAll(ctx context.Context, limit int) ([]model.ResourceRecommendation, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, vmid, vm_name, vm_type, resource_type, current_value, recommended_value, avg_usage, max_usage, recommendation_type, COALESCE(reason, ''), created_at
		 FROM resource_recommendations ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list all recommendations: %w", err)
	}
	defer rows.Close()

	var recs []model.ResourceRecommendation
	for rows.Next() {
		var rec model.ResourceRecommendation
		if err := rows.Scan(&rec.ID, &rec.NodeID, &rec.VMID, &rec.VMName, &rec.VMType,
			&rec.ResourceType, &rec.CurrentValue, &rec.RecommendedValue, &rec.AvgUsage, &rec.MaxUsage,
			&rec.RecommendationType, &rec.Reason, &rec.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan recommendation: %w", err)
		}
		recs = append(recs, rec)
	}
	return recs, rows.Err()
}

func (r *pgRecommendationRepository) DeleteByNode(ctx context.Context, nodeID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM resource_recommendations WHERE node_id=$1", nodeID)
	if err != nil {
		return fmt.Errorf("delete recommendations by node: %w", err)
	}
	return nil
}
