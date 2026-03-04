package environment

import (
	"context"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
)

type Service struct {
	envRepo  repository.EnvironmentRepository
	nodeRepo repository.NodeRepository
}

func NewService(envRepo repository.EnvironmentRepository, nodeRepo repository.NodeRepository) *Service {
	return &Service{envRepo: envRepo, nodeRepo: nodeRepo}
}

func (s *Service) Create(ctx context.Context, req model.CreateEnvironmentRequest) (*model.Environment, error) {
	env := &model.Environment{
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
	}
	if env.Color == "" {
		env.Color = "#6366f1"
	}
	if err := s.envRepo.Create(ctx, env); err != nil {
		return nil, fmt.Errorf("create environment: %w", err)
	}
	return env, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*model.Environment, error) {
	return s.envRepo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]model.Environment, error) {
	return s.envRepo.List(ctx)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req model.UpdateEnvironmentRequest) (*model.Environment, error) {
	env, err := s.envRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		env.Name = *req.Name
	}
	if req.Description != nil {
		env.Description = *req.Description
	}
	if req.Color != nil {
		env.Color = *req.Color
	}
	if err := s.envRepo.Update(ctx, env); err != nil {
		return nil, fmt.Errorf("update environment: %w", err)
	}
	return env, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.envRepo.Delete(ctx, id)
}

func (s *Service) AssignNode(ctx context.Context, nodeID uuid.UUID, envID *uuid.UUID) error {
	// Verify node exists
	if _, err := s.nodeRepo.GetByID(ctx, nodeID); err != nil {
		return err
	}
	// Verify environment exists (if not nil)
	if envID != nil {
		if _, err := s.envRepo.GetByID(ctx, *envID); err != nil {
			return err
		}
	}
	return s.nodeRepo.UpdateEnvironment(ctx, nodeID, envID)
}

func (s *Service) ListNodesByEnvironment(ctx context.Context, envID uuid.UUID) ([]model.Node, error) {
	return s.nodeRepo.ListByEnvironment(ctx, envID)
}
