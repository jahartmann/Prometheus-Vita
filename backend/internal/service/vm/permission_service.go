package vm

import (
	"context"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
)

type PermissionService struct {
	repo     repository.VMPermissionRepository
	userRepo repository.UserRepository
}

func NewPermissionService(repo repository.VMPermissionRepository, userRepo repository.UserRepository) *PermissionService {
	return &PermissionService{repo: repo, userRepo: userRepo}
}

// CheckPermission returns true if user has the given permission on the VM.
// Admins always have all permissions.
func (s *PermissionService) CheckPermission(ctx context.Context, userID uuid.UUID, nodeID uuid.UUID, vmid string, perm string) (bool, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	if user.Role == model.RoleAdmin {
		return true, nil
	}
	return s.repo.HasPermission(ctx, userID, nodeID, vmid, perm)
}

func (s *PermissionService) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.VMPermission, error) {
	return s.repo.ListByUser(ctx, userID)
}

func (s *PermissionService) Grant(ctx context.Context, perm *model.VMPermission) error {
	return s.repo.Create(ctx, perm)
}

func (s *PermissionService) Revoke(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *PermissionService) Update(ctx context.Context, perm *model.VMPermission) error {
	return s.repo.Update(ctx, perm)
}

func (s *PermissionService) ListByTarget(ctx context.Context, targetType, targetID string, nodeID uuid.UUID) ([]model.VMPermission, error) {
	return s.repo.ListByTarget(ctx, targetType, targetID, nodeID)
}

func (s *PermissionService) GetByID(ctx context.Context, id uuid.UUID) (*model.VMPermission, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *PermissionService) List(ctx context.Context) ([]model.VMPermission, error) {
	return s.repo.List(ctx)
}

func (s *PermissionService) Upsert(ctx context.Context, perm *model.VMPermission) error {
	return s.repo.Upsert(ctx, perm)
}

func (s *PermissionService) GetEffectivePermissions(ctx context.Context, userID, nodeID uuid.UUID, vmid int) ([]string, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.Role == model.RoleAdmin {
		return model.AllVMPermissions, nil
	}
	return s.repo.GetEffectivePermissions(ctx, userID, nodeID, vmid)
}
