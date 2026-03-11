package vm

import (
	"context"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
)

type GroupService struct {
	repo repository.VMGroupRepository
}

func NewGroupService(repo repository.VMGroupRepository) *GroupService {
	return &GroupService{repo: repo}
}

func (s *GroupService) Create(ctx context.Context, req *model.CreateVMGroupRequest, createdBy uuid.UUID) (*model.VMGroup, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	group := &model.VMGroup{
		Name:        req.Name,
		Description: req.Description,
		TagFilter:   req.TagFilter,
		CreatedBy:   createdBy,
	}
	if err := s.repo.Create(ctx, group); err != nil {
		return nil, err
	}
	return group, nil
}

func (s *GroupService) GetByID(ctx context.Context, id uuid.UUID) (*model.VMGroup, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *GroupService) List(ctx context.Context) ([]model.VMGroup, error) {
	return s.repo.List(ctx)
}

func (s *GroupService) Update(ctx context.Context, id uuid.UUID, req *model.UpdateVMGroupRequest) (*model.VMGroup, error) {
	group, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		group.Name = *req.Name
	}
	if req.Description != nil {
		group.Description = *req.Description
	}
	if req.TagFilter != nil {
		group.TagFilter = *req.TagFilter
	}
	if err := s.repo.Update(ctx, group); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *GroupService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *GroupService) ListMembers(ctx context.Context, groupID uuid.UUID) ([]model.VMGroupMember, error) {
	return s.repo.ListMembers(ctx, groupID)
}

func (s *GroupService) AddMember(ctx context.Context, groupID, nodeID uuid.UUID, vmid int) error {
	member := &model.VMGroupMember{
		GroupID: groupID,
		NodeID:  nodeID,
		VMID:    vmid,
	}
	return s.repo.AddMember(ctx, member)
}

func (s *GroupService) RemoveMember(ctx context.Context, groupID, nodeID uuid.UUID, vmid int) error {
	return s.repo.RemoveMember(ctx, groupID, nodeID, vmid)
}

func (s *GroupService) GetGroupsForVM(ctx context.Context, nodeID uuid.UUID, vmid int) ([]model.VMGroup, error) {
	return s.repo.GetGroupsForVM(ctx, nodeID, vmid)
}
