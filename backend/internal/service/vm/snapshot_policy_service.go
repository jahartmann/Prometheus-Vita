package vm

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/proxmox"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
)

type SnapshotPolicyService struct {
	policyRepo    repository.SnapshotPolicyRepository
	nodeRepo      repository.NodeRepository
	clientFactory proxmox.ClientFactory
}

func NewSnapshotPolicyService(
	policyRepo repository.SnapshotPolicyRepository,
	nodeRepo repository.NodeRepository,
	clientFactory proxmox.ClientFactory,
) *SnapshotPolicyService {
	return &SnapshotPolicyService{
		policyRepo:    policyRepo,
		nodeRepo:      nodeRepo,
		clientFactory: clientFactory,
	}
}

func (s *SnapshotPolicyService) Create(ctx context.Context, req model.CreateSnapshotPolicyRequest) (*model.SnapshotPolicy, error) {
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	scheduleCron := req.ScheduleCron
	if scheduleCron == "" {
		scheduleCron = "0 2 * * *"
	}

	keepDaily := req.KeepDaily
	if keepDaily == 0 {
		keepDaily = 5
	}

	p := &model.SnapshotPolicy{
		NodeID:       req.NodeID,
		VMID:         req.VMID,
		VMType:       req.VMType,
		Name:         req.Name,
		KeepDaily:    keepDaily,
		KeepWeekly:   req.KeepWeekly,
		KeepMonthly:  req.KeepMonthly,
		ScheduleCron: scheduleCron,
		IsActive:     isActive,
	}

	if err := s.policyRepo.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("create snapshot policy: %w", err)
	}
	return p, nil
}

func (s *SnapshotPolicyService) GetByID(ctx context.Context, id uuid.UUID) (*model.SnapshotPolicy, error) {
	return s.policyRepo.GetByID(ctx, id)
}

func (s *SnapshotPolicyService) List(ctx context.Context) ([]model.SnapshotPolicy, error) {
	return s.policyRepo.List(ctx)
}

func (s *SnapshotPolicyService) ListByVM(ctx context.Context, nodeID uuid.UUID, vmid int) ([]model.SnapshotPolicy, error) {
	return s.policyRepo.ListByVM(ctx, nodeID, vmid)
}

func (s *SnapshotPolicyService) Update(ctx context.Context, id uuid.UUID, req model.UpdateSnapshotPolicyRequest) (*model.SnapshotPolicy, error) {
	p, err := s.policyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.KeepDaily != nil {
		p.KeepDaily = *req.KeepDaily
	}
	if req.KeepWeekly != nil {
		p.KeepWeekly = *req.KeepWeekly
	}
	if req.KeepMonthly != nil {
		p.KeepMonthly = *req.KeepMonthly
	}
	if req.ScheduleCron != nil {
		p.ScheduleCron = *req.ScheduleCron
	}
	if req.IsActive != nil {
		p.IsActive = *req.IsActive
	}

	if err := s.policyRepo.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("update snapshot policy: %w", err)
	}
	return p, nil
}

func (s *SnapshotPolicyService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.policyRepo.Delete(ctx, id)
}

// ExecutePolicy creates a new snapshot and enforces retention by deleting old ones.
func (s *SnapshotPolicyService) ExecutePolicy(ctx context.Context, policy *model.SnapshotPolicy) error {
	node, err := s.nodeRepo.GetByID(ctx, policy.NodeID)
	if err != nil {
		return fmt.Errorf("get node: %w", err)
	}

	client, err := s.clientFactory.CreateClient(node)
	if err != nil {
		return fmt.Errorf("create proxmox client: %w", err)
	}

	pveNodes, err := client.GetNodes(ctx)
	if err != nil || len(pveNodes) == 0 {
		return fmt.Errorf("get pve nodes: %w", err)
	}

	// Create snapshot
	snapName := fmt.Sprintf("auto-%s-%s", policy.Name, time.Now().Format("20060102-150405"))
	description := fmt.Sprintf("Automatischer Snapshot (Richtlinie: %s)", policy.Name)

	_, err = client.CreateSnapshot(ctx, pveNodes[0], policy.VMID, policy.VMType, snapName, description, false)
	if err != nil {
		return fmt.Errorf("create snapshot: %w", err)
	}

	slog.Info("snapshot created by policy",
		slog.String("policy", policy.Name),
		slog.Int("vmid", policy.VMID),
		slog.String("snapshot", snapName),
	)

	// Update last run
	_ = s.policyRepo.UpdateLastRun(ctx, policy.ID, time.Now())

	// Enforce retention
	snapshots, err := client.ListSnapshots(ctx, pveNodes[0], policy.VMID, policy.VMType)
	if err != nil {
		slog.Warn("failed to list snapshots for retention", slog.Any("error", err))
		return nil
	}

	// Filter auto-created snapshots only (prefix "auto-")
	type snapEntry struct {
		name string
		time int64
	}
	var autoSnaps []snapEntry
	for _, snap := range snapshots {
		if strings.HasPrefix(snap.Name, "auto-") && snap.Name != "current" {
			autoSnaps = append(autoSnaps, snapEntry{name: snap.Name, time: snap.Snaptime})
		}
	}

	// Sort by time descending (newest first)
	sort.Slice(autoSnaps, func(i, j int) bool {
		return autoSnaps[i].time > autoSnaps[j].time
	})

	// Keep only the configured number of daily snapshots
	maxKeep := policy.KeepDaily + policy.KeepWeekly + policy.KeepMonthly
	if maxKeep <= 0 {
		maxKeep = policy.KeepDaily
	}
	if maxKeep <= 0 {
		maxKeep = 5
	}

	if len(autoSnaps) > maxKeep {
		toDelete := autoSnaps[maxKeep:]
		for _, snap := range toDelete {
			_, err := client.DeleteSnapshot(ctx, pveNodes[0], policy.VMID, policy.VMType, snap.name)
			if err != nil {
				slog.Warn("failed to delete old snapshot",
					slog.String("snapshot", snap.name),
					slog.Any("error", err),
				)
			} else {
				slog.Info("old snapshot deleted by retention policy",
					slog.String("snapshot", snap.name),
					slog.Int("vmid", policy.VMID),
				)
			}
		}
	}

	return nil
}
