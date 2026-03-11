package vm

import (
	"context"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/proxmox"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
)

type DependencyService struct {
	depRepo       repository.VMDependencyRepository
	nodeRepo      repository.NodeRepository
	clientFactory proxmox.ClientFactory
}

func NewDependencyService(
	depRepo repository.VMDependencyRepository,
	nodeRepo repository.NodeRepository,
	clientFactory proxmox.ClientFactory,
) *DependencyService {
	return &DependencyService{
		depRepo:       depRepo,
		nodeRepo:      nodeRepo,
		clientFactory: clientFactory,
	}
}

func (s *DependencyService) Create(ctx context.Context, req model.CreateVMDependencyRequest) (*model.VMDependency, error) {
	depType := req.DependencyType
	if depType == "" {
		depType = "depends_on"
	}

	d := &model.VMDependency{
		SourceNodeID:   req.SourceNodeID,
		SourceVMID:     req.SourceVMID,
		TargetNodeID:   req.TargetNodeID,
		TargetVMID:     req.TargetVMID,
		DependencyType: depType,
		Description:    req.Description,
	}

	if err := s.depRepo.Create(ctx, d); err != nil {
		return nil, fmt.Errorf("create vm dependency: %w", err)
	}
	return d, nil
}

func (s *DependencyService) List(ctx context.Context) ([]model.VMDependency, error) {
	deps, err := s.depRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	return s.enrichDependencies(ctx, deps), nil
}

func (s *DependencyService) ListByVM(ctx context.Context, nodeID uuid.UUID, vmid int) ([]model.VMDependency, error) {
	deps, err := s.depRepo.ListByVM(ctx, nodeID, vmid)
	if err != nil {
		return nil, err
	}
	return s.enrichDependencies(ctx, deps), nil
}

func (s *DependencyService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.depRepo.Delete(ctx, id)
}

// enrichDependencies adds VM names, types, and statuses to dependency records.
func (s *DependencyService) enrichDependencies(ctx context.Context, deps []model.VMDependency) []model.VMDependency {
	// Collect unique node IDs
	nodeIDs := make(map[uuid.UUID]bool)
	for _, d := range deps {
		nodeIDs[d.SourceNodeID] = true
		nodeIDs[d.TargetNodeID] = true
	}

	// Build VM lookup per node
	type vmKey struct {
		nodeID uuid.UUID
		vmid   int
	}
	type vmInfo struct {
		name   string
		vmType string
		status string
	}
	vmMap := make(map[vmKey]vmInfo)

	for nodeID := range nodeIDs {
		node, err := s.nodeRepo.GetByID(ctx, nodeID)
		if err != nil {
			continue
		}
		client, err := s.clientFactory.CreateClient(node)
		if err != nil {
			continue
		}
		pveNodes, err := client.GetNodes(ctx)
		if err != nil || len(pveNodes) == 0 {
			continue
		}
		vms, err := client.GetVMs(ctx, pveNodes[0])
		if err != nil {
			continue
		}
		for _, vm := range vms {
			vmMap[vmKey{nodeID: nodeID, vmid: vm.VMID}] = vmInfo{
				name:   vm.Name,
				vmType: vm.Type,
				status: vm.Status,
			}
		}
	}

	// Enrich
	for i := range deps {
		if info, ok := vmMap[vmKey{nodeID: deps[i].SourceNodeID, vmid: deps[i].SourceVMID}]; ok {
			deps[i].SourceVMName = info.name
			deps[i].SourceVMType = info.vmType
			deps[i].SourceStatus = info.status
		}
		if info, ok := vmMap[vmKey{nodeID: deps[i].TargetNodeID, vmid: deps[i].TargetVMID}]; ok {
			deps[i].TargetVMName = info.name
			deps[i].TargetVMType = info.vmType
			deps[i].TargetStatus = info.status
		}
	}

	return deps
}
