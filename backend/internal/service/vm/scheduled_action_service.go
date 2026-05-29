package vm

import (
	"context"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
)

// VMPowerController abstracts the VM power operations the scheduled-action
// executor needs. *node.Service satisfies it.
type VMPowerController interface {
	StartVM(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string) (string, error)
	StopVM(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string) (string, error)
	ShutdownVM(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string) (string, error)
	RebootVM(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string) (string, error)
}

type ScheduledActionService struct {
	actionRepo repository.ScheduledActionRepository
	power      VMPowerController
}

func NewScheduledActionService(
	actionRepo repository.ScheduledActionRepository,
	power VMPowerController,
) *ScheduledActionService {
	return &ScheduledActionService{
		actionRepo: actionRepo,
		power:      power,
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

// ListActive returns all active scheduled actions (used by the scheduler).
func (s *ScheduledActionService) ListActive(ctx context.Context) ([]model.ScheduledAction, error) {
	return s.actionRepo.ListActive(ctx)
}

func (s *ScheduledActionService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.actionRepo.Delete(ctx, id)
}

// Execute performs the action's VM power operation. It does NOT update
// last_run_at — the scheduler records that after a successful run.
func (s *ScheduledActionService) Execute(ctx context.Context, a model.ScheduledAction) error {
	if a.VMID == nil {
		return fmt.Errorf("scheduled action %s has no vmid", a.ID)
	}
	if s.power == nil {
		return fmt.Errorf("scheduled action executor not configured")
	}
	vmType := a.VMType
	if vmType == "" {
		vmType = "qemu"
	}
	var err error
	switch a.Action {
	case "start":
		_, err = s.power.StartVM(ctx, a.NodeID, *a.VMID, vmType)
	case "stop":
		_, err = s.power.StopVM(ctx, a.NodeID, *a.VMID, vmType)
	case "shutdown":
		_, err = s.power.ShutdownVM(ctx, a.NodeID, *a.VMID, vmType)
	case "restart", "reboot":
		_, err = s.power.RebootVM(ctx, a.NodeID, *a.VMID, vmType)
	default:
		return fmt.Errorf("unknown scheduled action %q", a.Action)
	}
	return err
}
