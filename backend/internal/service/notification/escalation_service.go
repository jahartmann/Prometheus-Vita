package notification

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
)

type EscalationService struct {
	policyRepo   repository.EscalationPolicyRepository
	incidentRepo repository.AlertIncidentRepository
	ruleRepo     repository.AlertRuleRepository
	notifSvc     *Service
}

func NewEscalationService(
	policyRepo repository.EscalationPolicyRepository,
	incidentRepo repository.AlertIncidentRepository,
	ruleRepo repository.AlertRuleRepository,
	notifSvc *Service,
) *EscalationService {
	return &EscalationService{
		policyRepo:   policyRepo,
		incidentRepo: incidentRepo,
		ruleRepo:     ruleRepo,
		notifSvc:     notifSvc,
	}
}

// CreateIncident creates a new alert incident for a triggered rule.
func (s *EscalationService) CreateIncident(ctx context.Context, ruleID uuid.UUID) (*model.AlertIncident, error) {
	// Check for existing open incidents
	open, err := s.incidentRepo.ListOpenByRule(ctx, ruleID)
	if err != nil {
		return nil, fmt.Errorf("list open incidents: %w", err)
	}
	if len(open) > 0 {
		return &open[0], nil
	}

	incident := &model.AlertIncident{
		AlertRuleID: ruleID,
		Status:      model.IncidentStatusTriggered,
		CurrentStep: 0,
	}
	if err := s.incidentRepo.Create(ctx, incident); err != nil {
		return nil, fmt.Errorf("create incident: %w", err)
	}
	return incident, nil
}

// ProcessEscalations checks all triggered incidents and escalates as needed.
func (s *EscalationService) ProcessEscalations(ctx context.Context) error {
	incidents, err := s.incidentRepo.ListByStatus(ctx, model.IncidentStatusTriggered)
	if err != nil {
		return fmt.Errorf("list triggered incidents: %w", err)
	}

	for _, inc := range incidents {
		s.processIncident(ctx, &inc)
	}

	return nil
}

func (s *EscalationService) processIncident(ctx context.Context, incident *model.AlertIncident) {
	rule, err := s.ruleRepo.GetByID(ctx, incident.AlertRuleID)
	if err != nil {
		slog.Error("failed to get rule for incident",
			slog.String("incident_id", incident.ID.String()),
			slog.Any("error", err))
		return
	}

	if rule.EscalationPolicyID == nil {
		return
	}

	policy, err := s.policyRepo.GetByID(ctx, *rule.EscalationPolicyID)
	if err != nil {
		slog.Error("failed to get escalation policy",
			slog.String("policy_id", rule.EscalationPolicyID.String()),
			slog.Any("error", err))
		return
	}

	if !policy.IsActive || len(policy.Steps) == 0 {
		return
	}

	// Find next step to execute
	for _, step := range policy.Steps {
		if step.StepOrder <= incident.CurrentStep {
			continue
		}

		// Check if enough time has passed
		referenceTime := incident.TriggeredAt
		if incident.LastEscalatedAt != nil {
			referenceTime = *incident.LastEscalatedAt
		}

		if time.Since(referenceTime) < time.Duration(step.DelaySeconds)*time.Second {
			break
		}

		// Execute escalation step
		subject := fmt.Sprintf("[Eskalation Stufe %d] Alert: %s", step.StepOrder, rule.Name)
		body := fmt.Sprintf(
			"Eskalation fuer Alert-Regel: %s\nStufe: %d\nVorfall-ID: %s\nStatus: %s",
			rule.Name, step.StepOrder, incident.ID.String(), incident.Status,
		)

		if len(step.ChannelIDs) > 0 {
			s.notifSvc.NotifyChannels(ctx, step.ChannelIDs, "escalation", subject, body)
		}

		now := time.Now()
		if err := s.incidentRepo.UpdateEscalation(ctx, incident.ID, step.StepOrder, now); err != nil {
			slog.Error("failed to update escalation",
				slog.String("incident_id", incident.ID.String()),
				slog.Any("error", err))
		}

		slog.Info("escalation step executed",
			slog.String("incident_id", incident.ID.String()),
			slog.Int("step", step.StepOrder),
			slog.String("rule", rule.Name))

		break
	}
}

// AcknowledgeIncident marks an incident as acknowledged.
func (s *EscalationService) AcknowledgeIncident(ctx context.Context, incidentID uuid.UUID, userID uuid.UUID) error {
	incident, err := s.incidentRepo.GetByID(ctx, incidentID)
	if err != nil {
		return err
	}
	if incident.Status != model.IncidentStatusTriggered {
		return fmt.Errorf("incident is not in triggered state")
	}
	return s.incidentRepo.Acknowledge(ctx, incidentID, userID)
}

// ResolveIncident marks an incident as resolved.
func (s *EscalationService) ResolveIncident(ctx context.Context, incidentID uuid.UUID, userID uuid.UUID) error {
	incident, err := s.incidentRepo.GetByID(ctx, incidentID)
	if err != nil {
		return err
	}
	if incident.Status == model.IncidentStatusResolved {
		return fmt.Errorf("incident is already resolved")
	}
	return s.incidentRepo.Resolve(ctx, incidentID, userID)
}

// GetIncident retrieves a single incident.
func (s *EscalationService) GetIncident(ctx context.Context, id uuid.UUID) (*model.AlertIncident, error) {
	return s.incidentRepo.GetByID(ctx, id)
}

// ListIncidents returns all incidents.
func (s *EscalationService) ListIncidents(ctx context.Context, limit, offset int) ([]model.AlertIncident, error) {
	return s.incidentRepo.List(ctx, limit, offset)
}

// CRUD for escalation policies

func (s *EscalationService) CreatePolicy(ctx context.Context, req model.CreateEscalationPolicyRequest) (*model.EscalationPolicy, error) {
	policy := &model.EscalationPolicy{
		Name:        req.Name,
		Description: req.Description,
		IsActive:    true,
	}
	if err := s.policyRepo.Create(ctx, policy); err != nil {
		return nil, fmt.Errorf("create policy: %w", err)
	}

	for _, stepReq := range req.Steps {
		step := &model.EscalationStep{
			PolicyID:     policy.ID,
			StepOrder:    stepReq.StepOrder,
			DelaySeconds: stepReq.DelaySeconds,
			ChannelIDs:   stepReq.ChannelIDs,
		}
		if step.ChannelIDs == nil {
			step.ChannelIDs = []uuid.UUID{}
		}
		if err := s.policyRepo.CreateStep(ctx, step); err != nil {
			return nil, fmt.Errorf("create step: %w", err)
		}
	}

	return s.policyRepo.GetByID(ctx, policy.ID)
}

func (s *EscalationService) GetPolicy(ctx context.Context, id uuid.UUID) (*model.EscalationPolicy, error) {
	return s.policyRepo.GetByID(ctx, id)
}

func (s *EscalationService) ListPolicies(ctx context.Context) ([]model.EscalationPolicy, error) {
	return s.policyRepo.List(ctx)
}

func (s *EscalationService) UpdatePolicy(ctx context.Context, id uuid.UUID, req model.UpdateEscalationPolicyRequest) (*model.EscalationPolicy, error) {
	policy, err := s.policyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		policy.Name = *req.Name
	}
	if req.Description != nil {
		policy.Description = *req.Description
	}
	if req.IsActive != nil {
		policy.IsActive = *req.IsActive
	}

	if err := s.policyRepo.Update(ctx, policy); err != nil {
		return nil, err
	}

	// Replace steps if provided
	if req.Steps != nil {
		if err := s.policyRepo.DeleteStepsByPolicy(ctx, id); err != nil {
			return nil, err
		}
		for _, stepReq := range req.Steps {
			step := &model.EscalationStep{
				PolicyID:     id,
				StepOrder:    stepReq.StepOrder,
				DelaySeconds: stepReq.DelaySeconds,
				ChannelIDs:   stepReq.ChannelIDs,
			}
			if step.ChannelIDs == nil {
				step.ChannelIDs = []uuid.UUID{}
			}
			if err := s.policyRepo.CreateStep(ctx, step); err != nil {
				return nil, err
			}
		}
	}

	return s.policyRepo.GetByID(ctx, id)
}

func (s *EscalationService) DeletePolicy(ctx context.Context, id uuid.UUID) error {
	return s.policyRepo.Delete(ctx, id)
}
