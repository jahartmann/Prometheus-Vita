package vm

import (
	"context"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
)

type ScheduledActionService struct {
	actionRepo repository.ScheduledActionRepository
}

func NewScheduledActionService(
	actionRepo repository.ScheduledActionRepository,
) *ScheduledActionService {
	return &ScheduledActionService{
		actionRepo: actionRepo,
	}
}

func (s *ScheduledActionService) Create(ctx context.Context, req model.CreateScheduledActionRequest) (*model.ScheduledAction, error) {
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	a := &model.ScheduledAction{
		NodeID:       req.NodeID,
		VMID:         req.VMID,
		VMType:       req.VMType,
		Action:       req.Action,
		ScheduleCron: req.ScheduleCron,
		IsActive:     isActive,
		Description:  req.Description,
	}

	if err := s.actionRepo.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("create scheduled action: %w", err)
	}
	return a, nil
}

func (s *ScheduledActionService) GetByID(ctx context.Context, id uuid.UUID) (*model.ScheduledAction, error) {
	return s.actionRepo.GetByID(ctx, id)
}

func (s *ScheduledActionService) List(ctx context.Context) ([]model.ScheduledAction, error) {
	return s.actionRepo.List(ctx)
}

func (s *ScheduledActionService) ListByVM(ctx context.Context, nodeID uuid.UUID, vmid int) ([]model.ScheduledAction, error) {
	return s.actionRepo.ListByVM(ctx, nodeID, vmid)
}

func (s *ScheduledActionService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.actionRepo.Delete(ctx, id)
}
