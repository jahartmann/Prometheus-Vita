package recovery

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
)

type ReadinessService struct {
	readinessRepo repository.DRReadinessRepository
	profileRepo   repository.NodeProfileRepository
	backupRepo    repository.BackupRepository
	nodeRepo      repository.NodeRepository
}

func NewReadinessService(
	readinessRepo repository.DRReadinessRepository,
	profileRepo repository.NodeProfileRepository,
	backupRepo repository.BackupRepository,
	nodeRepo repository.NodeRepository,
) *ReadinessService {
	return &ReadinessService{
		readinessRepo: readinessRepo,
		profileRepo:   profileRepo,
		backupRepo:    backupRepo,
		nodeRepo:      nodeRepo,
	}
}

func (s *ReadinessService) CalculateScore(ctx context.Context, nodeID uuid.UUID) (*model.DRReadinessScore, error) {
	// Verify node exists
	if _, err := s.nodeRepo.GetByID(ctx, nodeID); err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}

	details := make(map[string]interface{})

	// Backup Score (40% weight)
	backupScore := 0
	latestBackup, err := s.backupRepo.GetLatestByNode(ctx, nodeID)
	if err == nil && latestBackup != nil {
		if latestBackup.Status == model.BackupStatusCompleted {
			age := time.Since(latestBackup.CreatedAt)
			switch {
			case age < 24*time.Hour:
				backupScore = 100
				details["backup_status"] = "recent backup within 24h"
			case age < 72*time.Hour:
				backupScore = 70
				details["backup_status"] = "backup within 72h"
			case age < 7*24*time.Hour:
				backupScore = 40
				details["backup_status"] = "backup within 7 days"
			default:
				backupScore = 10
				details["backup_status"] = "backup older than 7 days"
			}
			details["backup_age_hours"] = int(age.Hours())
		} else {
			backupScore = 5
			details["backup_status"] = "latest backup not completed"
		}
	} else {
		details["backup_status"] = "no backup found"
	}

	// Profile Score (30% weight)
	profileScore := 0
	latestProfile, err := s.profileRepo.GetLatest(ctx, nodeID)
	if err == nil && latestProfile != nil {
		age := time.Since(latestProfile.CollectedAt)
		switch {
		case age < 7*24*time.Hour:
			profileScore = 100
			details["profile_status"] = "recent profile within 7 days"
		case age < 30*24*time.Hour:
			profileScore = 60
			details["profile_status"] = "profile within 30 days"
		default:
			profileScore = 20
			details["profile_status"] = "profile older than 30 days"
		}
		details["profile_age_hours"] = int(age.Hours())
	} else {
		details["profile_status"] = "no profile found"
	}

	// Config Score (30% weight) - based on profile data completeness
	configScore := 0
	if latestProfile != nil {
		completeness := 0
		total := 7
		if latestProfile.CPUModel != "" {
			completeness++
		}
		if latestProfile.CPUCores > 0 {
			completeness++
		}
		if latestProfile.MemoryTotalBytes > 0 {
			completeness++
		}
		if len(latestProfile.Disks) > 0 {
			completeness++
		}
		if len(latestProfile.NetworkInterfaces) > 0 {
			completeness++
		}
		if latestProfile.PVEVersion != "" {
			completeness++
		}
		if latestProfile.KernelVersion != "" {
			completeness++
		}
		configScore = (completeness * 100) / total
		details["config_completeness"] = fmt.Sprintf("%d/%d fields", completeness, total)
	} else {
		details["config_completeness"] = "no profile data"
	}

	// Calculate overall score with weights
	overallScore := (backupScore*40 + profileScore*30 + configScore*30) / 100

	detailsJSON, _ := json.Marshal(details)

	score := &model.DRReadinessScore{
		NodeID:       nodeID,
		OverallScore: overallScore,
		BackupScore:  backupScore,
		ProfileScore: profileScore,
		ConfigScore:  configScore,
		Details:      detailsJSON,
	}

	if err := s.readinessRepo.Create(ctx, score); err != nil {
		return nil, fmt.Errorf("create readiness score: %w", err)
	}

	return score, nil
}

func (s *ReadinessService) GetLatestScore(ctx context.Context, nodeID uuid.UUID) (*model.DRReadinessScore, error) {
	return s.readinessRepo.GetLatest(ctx, nodeID)
}

func (s *ReadinessService) ListAllScores(ctx context.Context) ([]model.DRReadinessScore, error) {
	return s.readinessRepo.ListAll(ctx)
}
