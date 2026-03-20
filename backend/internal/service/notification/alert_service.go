package notification

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/monitor"
	"github.com/google/uuid"
)

type AlertService struct {
	ruleRepo      repository.AlertRuleRepository
	metricsRepo   repository.MetricsRepository
	nodeRepo      repository.NodeRepository
	notifSvc      *Service
	wsHub         *monitor.WSHub
	escalationSvc *EscalationService
}

func NewAlertService(
	ruleRepo repository.AlertRuleRepository,
	metricsRepo repository.MetricsRepository,
	nodeRepo repository.NodeRepository,
	notifSvc *Service,
	wsHub *monitor.WSHub,
) *AlertService {
	return &AlertService{
		ruleRepo:    ruleRepo,
		metricsRepo: metricsRepo,
		nodeRepo:    nodeRepo,
		notifSvc:    notifSvc,
		wsHub:       wsHub,
	}
}

// SetEscalationService links the escalation service for incident creation on alert trigger.
func (s *AlertService) SetEscalationService(esc *EscalationService) {
	s.escalationSvc = esc
}

// EvaluateRules checks all active alert rules against current metrics.
func (s *AlertService) EvaluateRules(ctx context.Context) error {
	rules, err := s.ruleRepo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("list active alert rules: %w", err)
	}

	for _, rule := range rules {
		s.evaluateRule(ctx, &rule)
	}

	return nil
}

func (s *AlertService) evaluateRule(ctx context.Context, rule *model.AlertRule) {
	latest, err := s.metricsRepo.GetLatest(ctx, rule.NodeID)
	if err != nil {
		slog.Debug("no metrics for node, skipping rule",
			slog.String("rule", rule.Name),
			slog.String("node_id", rule.NodeID.String()),
		)
		return
	}

	value := s.getMetricValue(latest, rule.Metric)
	violated := s.checkThreshold(value, rule.Operator, rule.Threshold)

	if !violated {
		return
	}

	// Check cooldown based on duration_seconds
	if rule.LastTriggeredAt != nil {
		cooldown := time.Duration(rule.DurationSeconds) * time.Second
		if cooldown < 5*time.Minute {
			cooldown = 5 * time.Minute // minimum 5min cooldown to prevent spam
		}
		if time.Since(*rule.LastTriggeredAt) < cooldown {
			return
		}
	}

	// Get node name for the notification
	nodeName := rule.NodeID.String()
	node, err := s.nodeRepo.GetByID(ctx, rule.NodeID)
	if err == nil {
		nodeName = node.Name
	}

	// Build notification
	severityPrefix := fmt.Sprintf("[%s]", rule.Severity)
	subject := fmt.Sprintf("%s Alert: %s on %s", severityPrefix, rule.Name, nodeName)
	body := fmt.Sprintf(
		"Alert Rule: %s\nNode: %s\nMetric: %s\nCurrent Value: %.2f\nThreshold: %s %.2f\nSeverity: %s",
		rule.Name, nodeName, rule.Metric, value, rule.Operator, rule.Threshold, rule.Severity,
	)

	// Send to configured channels
	if len(rule.ChannelIDs) > 0 {
		s.notifSvc.NotifyChannels(ctx, rule.ChannelIDs, "alert_triggered", subject, body)
	} else {
		s.notifSvc.Notify(ctx, "alert_triggered", subject, body)
	}

	// Broadcast via WebSocket
	if s.wsHub != nil {
		s.wsHub.BroadcastMessage(monitor.WSMessage{
			Type: "alert_triggered",
			Data: map[string]any{
				"rule_id":   rule.ID,
				"rule_name": rule.Name,
				"node_id":   rule.NodeID,
				"node_name": nodeName,
				"metric":    rule.Metric,
				"value":     value,
				"threshold": rule.Threshold,
				"severity":  rule.Severity,
			},
		})
	}

	// Create incident if escalation service is available
	if s.escalationSvc != nil && rule.EscalationPolicyID != nil {
		if _, err := s.escalationSvc.CreateIncident(ctx, rule.ID); err != nil {
			slog.Error("failed to create alert incident",
				slog.String("rule_id", rule.ID.String()),
				slog.Any("error", err),
			)
		}
	}

	// Update last triggered
	now := time.Now()
	if err := s.ruleRepo.UpdateLastTriggered(ctx, rule.ID, now); err != nil {
		slog.Error("failed to update last triggered",
			slog.String("rule_id", rule.ID.String()),
			slog.Any("error", err),
		)
	}

	slog.Warn("alert triggered",
		slog.String("rule", rule.Name),
		slog.String("node", nodeName),
		slog.String("metric", rule.Metric),
		slog.Float64("value", value),
		slog.Float64("threshold", rule.Threshold),
	)
}

func (s *AlertService) getMetricValue(record *model.MetricsRecord, metric string) float64 {
	switch metric {
	case "cpu_usage":
		return record.CPUUsage
	case "memory_usage":
		if record.MemTotal > 0 {
			return float64(record.MemUsed) / float64(record.MemTotal) * 100
		}
		return 0
	case "disk_usage":
		if record.DiskTotal > 0 {
			return float64(record.DiskUsed) / float64(record.DiskTotal) * 100
		}
		return 0
	case "load_avg":
		if len(record.LoadAvg) > 0 {
			return record.LoadAvg[0]
		}
		return 0
	case "net_in":
		return float64(record.NetIn)
	case "net_out":
		return float64(record.NetOut)
	default:
		return 0
	}
}

func (s *AlertService) checkThreshold(value float64, operator string, threshold float64) bool {
	switch operator {
	case ">":
		return value > threshold
	case ">=":
		return value >= threshold
	case "<":
		return value < threshold
	case "<=":
		return value <= threshold
	case "==":
		return value == threshold
	case "!=":
		return value != threshold
	default:
		return false
	}
}

// CRUD for alert rules

func (s *AlertService) CreateRule(ctx context.Context, req model.CreateAlertRuleRequest) (*model.AlertRule, error) {
	rule := &model.AlertRule{
		Name:               req.Name,
		NodeID:             req.NodeID,
		Metric:             req.Metric,
		Operator:           req.Operator,
		Threshold:          req.Threshold,
		DurationSeconds:    req.DurationSeconds,
		Severity:           req.Severity,
		ChannelIDs:         req.ChannelIDs,
		EscalationPolicyID: req.EscalationPolicyID,
		IsActive:           true,
	}
	if req.IsActive != nil {
		rule.IsActive = *req.IsActive
	}
	if rule.ChannelIDs == nil {
		rule.ChannelIDs = []uuid.UUID{}
	}

	if err := s.ruleRepo.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("create alert rule: %w", err)
	}
	return rule, nil
}

func (s *AlertService) GetRule(ctx context.Context, id uuid.UUID) (*model.AlertRule, error) {
	return s.ruleRepo.GetByID(ctx, id)
}

func (s *AlertService) ListRules(ctx context.Context) ([]model.AlertRule, error) {
	return s.ruleRepo.List(ctx)
}

func (s *AlertService) UpdateRule(ctx context.Context, id uuid.UUID, req model.UpdateAlertRuleRequest) (*model.AlertRule, error) {
	rule, err := s.ruleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Metric != nil {
		rule.Metric = *req.Metric
	}
	if req.Operator != nil {
		rule.Operator = *req.Operator
	}
	if req.Threshold != nil {
		rule.Threshold = *req.Threshold
	}
	if req.DurationSeconds != nil {
		rule.DurationSeconds = *req.DurationSeconds
	}
	if req.Severity != nil {
		rule.Severity = *req.Severity
	}
	if req.ChannelIDs != nil {
		rule.ChannelIDs = *req.ChannelIDs
	}
	if req.EscalationPolicyID != nil {
		rule.EscalationPolicyID = req.EscalationPolicyID
	}
	if req.IsActive != nil {
		rule.IsActive = *req.IsActive
	}

	if err := s.ruleRepo.Update(ctx, rule); err != nil {
		return nil, err
	}
	return rule, nil
}

func (s *AlertService) DeleteRule(ctx context.Context, id uuid.UUID) error {
	return s.ruleRepo.Delete(ctx, id)
}
