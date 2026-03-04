package repository

import (
	"context"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PredictionRepository interface {
	Upsert(ctx context.Context, pred *model.MaintenancePrediction) error
	ListCritical(ctx context.Context) ([]model.MaintenancePrediction, error)
	ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.MaintenancePrediction, error)
}

type pgPredictionRepository struct {
	db *pgxpool.Pool
}

func NewPredictionRepository(db *pgxpool.Pool) PredictionRepository {
	return &pgPredictionRepository{db: db}
}

func (r *pgPredictionRepository) Upsert(ctx context.Context, pred *model.MaintenancePrediction) error {
	pred.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO maintenance_predictions (id, node_id, metric, current_value, predicted_value, threshold,
		        days_until_threshold, slope, intercept, r_squared, severity, predicted_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
		 ON CONFLICT (node_id, metric) DO UPDATE SET
		    current_value = EXCLUDED.current_value,
		    predicted_value = EXCLUDED.predicted_value,
		    threshold = EXCLUDED.threshold,
		    days_until_threshold = EXCLUDED.days_until_threshold,
		    slope = EXCLUDED.slope,
		    intercept = EXCLUDED.intercept,
		    r_squared = EXCLUDED.r_squared,
		    severity = EXCLUDED.severity,
		    predicted_at = NOW()`,
		pred.ID, pred.NodeID, pred.Metric, pred.CurrentValue, pred.PredictedValue,
		pred.Threshold, pred.DaysUntilThreshold, pred.Slope, pred.Intercept,
		pred.RSquared, pred.Severity,
	)
	if err != nil {
		return fmt.Errorf("upsert prediction: %w", err)
	}
	return nil
}

func (r *pgPredictionRepository) ListCritical(ctx context.Context) ([]model.MaintenancePrediction, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, metric, current_value, predicted_value, threshold,
		        days_until_threshold, slope, intercept, r_squared, severity, predicted_at
		 FROM maintenance_predictions
		 WHERE severity IN ('warning', 'critical')
		 ORDER BY days_until_threshold ASC NULLS LAST`)
	if err != nil {
		return nil, fmt.Errorf("list critical predictions: %w", err)
	}
	defer rows.Close()

	var preds []model.MaintenancePrediction
	for rows.Next() {
		var p model.MaintenancePrediction
		if err := rows.Scan(&p.ID, &p.NodeID, &p.Metric, &p.CurrentValue, &p.PredictedValue,
			&p.Threshold, &p.DaysUntilThreshold, &p.Slope, &p.Intercept,
			&p.RSquared, &p.Severity, &p.PredictedAt); err != nil {
			return nil, fmt.Errorf("scan prediction: %w", err)
		}
		preds = append(preds, p)
	}
	return preds, rows.Err()
}

func (r *pgPredictionRepository) ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.MaintenancePrediction, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, metric, current_value, predicted_value, threshold,
		        days_until_threshold, slope, intercept, r_squared, severity, predicted_at
		 FROM maintenance_predictions WHERE node_id = $1
		 ORDER BY severity DESC, predicted_at DESC`, nodeID)
	if err != nil {
		return nil, fmt.Errorf("list predictions by node: %w", err)
	}
	defer rows.Close()

	var preds []model.MaintenancePrediction
	for rows.Next() {
		var p model.MaintenancePrediction
		if err := rows.Scan(&p.ID, &p.NodeID, &p.Metric, &p.CurrentValue, &p.PredictedValue,
			&p.Threshold, &p.DaysUntilThreshold, &p.Slope, &p.Intercept,
			&p.RSquared, &p.Severity, &p.PredictedAt); err != nil {
			return nil, fmt.Errorf("scan prediction: %w", err)
		}
		preds = append(preds, p)
	}
	return preds, rows.Err()
}
