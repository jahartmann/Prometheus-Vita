package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
)

type Service struct {
	repo repository.BrainRepository
}

func NewService(repo repository.BrainRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req model.CreateBrainEntryRequest, createdBy *uuid.UUID) (*model.BrainEntry, error) {
	entry := &model.BrainEntry{
		Category:       req.Category,
		Subject:        req.Subject,
		Content:        req.Content,
		Metadata:       json.RawMessage("{}"),
		RelevanceScore: 1.0,
		CreatedBy:      createdBy,
	}

	if err := s.repo.Create(ctx, entry); err != nil {
		return nil, fmt.Errorf("create brain entry: %w", err)
	}

	slog.Info("brain entry created",
		slog.String("id", entry.ID.String()),
		slog.String("category", entry.Category),
		slog.String("subject", entry.Subject),
	)

	return entry, nil
}

func (s *Service) List(ctx context.Context) ([]model.BrainEntry, error) {
	return s.repo.List(ctx)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) Search(ctx context.Context, query string) ([]model.BrainEntry, error) {
	entries, err := s.repo.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	// Increment access count for found entries
	for _, e := range entries {
		if err := s.repo.IncrementAccessCount(ctx, e.ID); err != nil {
			slog.Warn("failed to increment access count",
				slog.String("id", e.ID.String()),
				slog.Any("error", err),
			)
		}
	}

	return entries, nil
}

func (s *Service) Cleanup(ctx context.Context, minScore float64, minAccessCount int) error {
	if err := s.repo.DeleteLowRelevance(ctx, minScore, minAccessCount); err != nil {
		return fmt.Errorf("cleanup brain entries: %w", err)
	}

	slog.Info("brain cleanup completed",
		slog.Float64("min_score", minScore),
		slog.Int("min_access_count", minAccessCount),
	)

	return nil
}
