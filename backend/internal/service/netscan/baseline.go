package netscan

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
)

// BaselineService manages scan baselines for nodes.
type BaselineService struct {
	repo repository.ScanBaselineRepository
}

// NewBaselineService creates a new BaselineService backed by the given repository.
func NewBaselineService(repo repository.ScanBaselineRepository) *BaselineService {
	return &BaselineService{repo: repo}
}

// CreateFromCurrentScan marshals scan into JSON and persists a new baseline
// for the given node. The baseline is created as inactive; call Activate to
// make it the reference point for future diffs.
func (s *BaselineService) CreateFromCurrentScan(
	ctx context.Context,
	nodeID uuid.UUID,
	scan *QuickScanResult,
	label string,
) (*model.ScanBaseline, error) {
	data, err := json.Marshal(scan)
	if err != nil {
		return nil, fmt.Errorf("netscan: marshal baseline scan: %w", err)
	}

	baseline := &model.ScanBaseline{
		NodeID:       nodeID,
		Label:        label,
		IsActive:     false,
		BaselineJSON: json.RawMessage(data),
	}

	if err := s.repo.Create(ctx, baseline); err != nil {
		return nil, fmt.Errorf("netscan: create baseline: %w", err)
	}

	return baseline, nil
}

// GetActive returns the currently active baseline for the given node.
// Returns repository.ErrNotFound when no active baseline exists.
func (s *BaselineService) GetActive(ctx context.Context, nodeID uuid.UUID) (*model.ScanBaseline, error) {
	return s.repo.GetActive(ctx, nodeID)
}

// Activate sets the given baseline as the active one for the node,
// deactivating any previously active baseline in the same transaction.
func (s *BaselineService) Activate(ctx context.Context, nodeID, baselineID uuid.UUID) error {
	return s.repo.Activate(ctx, nodeID, baselineID)
}

// UnmarshalBaseline decodes the BaselineJSON of a ScanBaseline back into a
// QuickScanResult so it can be used for diff computation.
func UnmarshalBaseline(b *model.ScanBaseline) (*QuickScanResult, error) {
	var qsr QuickScanResult
	if err := json.Unmarshal(b.BaselineJSON, &qsr); err != nil {
		return nil, fmt.Errorf("netscan: unmarshal baseline: %w", err)
	}
	return &qsr, nil
}
