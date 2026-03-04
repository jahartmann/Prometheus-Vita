package reflex

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/node"
	"github.com/antigravity/prometheus/internal/service/notification"
	"github.com/google/uuid"
)

type Service struct {
	reflexRepo  repository.ReflexRuleRepository
	metricsRepo repository.MetricsRepository
	nodeRepo    repository.NodeRepository
	nodeSvc     *node.Service
	notifSvc    *notification.Service
}

func NewService(
	reflexRepo repository.ReflexRuleRepository,
	metricsRepo repository.MetricsRepository,
	nodeRepo repository.NodeRepository,
	nodeSvc *node.Service,
	notifSvc *notification.Service,
) *Service {
	return &Service{
		reflexRepo:  reflexRepo,
		metricsRepo: metricsRepo,
		nodeRepo:    nodeRepo,
		nodeSvc:     nodeSvc,
		notifSvc:    notifSvc,
	}
}

func (s *Service) Create(ctx context.Context, req model.CreateReflexRuleRequest) (*model.ReflexRule, error) {
	actionConfig := req.ActionConfig
	if actionConfig == nil {
		actionConfig = json.RawMessage("{}")
	}

	cooldown := req.CooldownSeconds
	if cooldown == 0 {
		cooldown = 300
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	rule := &model.ReflexRule{
		Name:            req.Name,
		Description:     req.Description,
		TriggerMetric:   req.TriggerMetric,
		Operator:        req.Operator,
		Threshold:       req.Threshold,
		ActionType:      req.ActionType,
		ActionConfig:    actionConfig,
		CooldownSeconds: cooldown,
		IsActive:        isActive,
		NodeID:          req.NodeID,
	}

	if err := s.reflexRepo.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("create reflex rule: %w", err)
	}

	return rule, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*model.ReflexRule, error) {
	return s.reflexRepo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]model.ReflexRule, error) {
	return s.reflexRepo.List(ctx)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req model.UpdateReflexRuleRequest) (*model.ReflexRule, error) {
	rule, err := s.reflexRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Description != nil {
		rule.Description = *req.Description
	}
	if req.TriggerMetric != nil {
		rule.TriggerMetric = *req.TriggerMetric
	}
	if req.Operator != nil {
		rule.Operator = *req.Operator
	}
	if req.Threshold != nil {
		rule.Threshold = *req.Threshold
	}
	if req.ActionType != nil {
		rule.ActionType = *req.ActionType
	}
	if req.ActionConfig != nil {
		rule.ActionConfig = *req.ActionConfig
	}
	if req.CooldownSeconds != nil {
		rule.CooldownSeconds = *req.CooldownSeconds
	}
	if req.IsActive != nil {
		rule.IsActive = *req.IsActive
	}
	if req.NodeID != nil {
		rule.NodeID = req.NodeID
	}

	if err := s.reflexRepo.Update(ctx, rule); err != nil {
		return nil, fmt.Errorf("update reflex rule: %w", err)
	}

	return rule, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.reflexRepo.Delete(ctx, id)
}

func (s *Service) EvaluateRules(ctx context.Context) error {
	rules, err := s.reflexRepo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("list active reflex rules: %w", err)
	}

	nodes, err := s.nodeRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("list nodes: %w", err)
	}

	now := time.Now()

	for i := range rules {
		rule := &rules[i]

		// Cooldown check
		if rule.LastTriggeredAt != nil {
			if now.Sub(*rule.LastTriggeredAt) < time.Duration(rule.CooldownSeconds)*time.Second {
				continue
			}
		}

		// Determine which nodes to check
		var targetNodes []model.Node
		if rule.NodeID != nil {
			for _, n := range nodes {
				if n.ID == *rule.NodeID {
					targetNodes = append(targetNodes, n)
					break
				}
			}
		} else {
			targetNodes = nodes
		}

		for _, n := range targetNodes {
			if !n.IsOnline {
				continue
			}

			latest, err := s.metricsRepo.GetLatest(ctx, n.ID)
			if err != nil || latest == nil {
				continue
			}

			value := extractMetricValue(latest, rule.TriggerMetric)
			if value == nil {
				continue
			}

			if !evaluateCondition(*value, rule.Operator, rule.Threshold) {
				continue
			}

			slog.Info("reflex rule triggered",
				slog.String("rule", rule.Name),
				slog.String("node", n.Name),
				slog.String("metric", rule.TriggerMetric),
				slog.Float64("value", *value),
				slog.Float64("threshold", rule.Threshold),
			)

			s.executeAction(ctx, rule, &n)

			if err := s.reflexRepo.UpdateLastTriggered(ctx, rule.ID, now); err != nil {
				slog.Warn("failed to update reflex rule last triggered",
					slog.String("rule", rule.Name),
					slog.Any("error", err),
				)
			}

			// Only trigger once per rule per evaluation cycle
			break
		}
	}

	return nil
}

func (s *Service) executeAction(ctx context.Context, rule *model.ReflexRule, n *model.Node) {
	var config map[string]interface{}
	if err := json.Unmarshal(rule.ActionConfig, &config); err != nil {
		slog.Warn("failed to parse action config", slog.String("rule", rule.Name), slog.Any("error", err))
		return
	}

	switch rule.ActionType {
	case model.ReflexActionRestartService:
		serviceName, _ := config["service_name"].(string)
		if serviceName == "" {
			slog.Warn("reflex: restart_service missing service_name", slog.String("rule", rule.Name))
			return
		}
		cmd := fmt.Sprintf("systemctl restart %s", serviceName)
		result, err := s.nodeSvc.RunSSHCommand(ctx, n.ID, cmd)
		if err != nil {
			slog.Warn("reflex: restart_service failed", slog.String("rule", rule.Name), slog.Any("error", err))
			return
		}
		slog.Info("reflex: service restarted", slog.String("rule", rule.Name), slog.String("service", serviceName), slog.String("output", result.Stdout))

	case model.ReflexActionClearCache:
		cmd := "sync && echo 3 > /proc/sys/vm/drop_caches"
		result, err := s.nodeSvc.RunSSHCommand(ctx, n.ID, cmd)
		if err != nil {
			slog.Warn("reflex: clear_cache failed", slog.String("rule", rule.Name), slog.Any("error", err))
			return
		}
		slog.Info("reflex: cache cleared", slog.String("rule", rule.Name), slog.String("output", result.Stdout))

	case model.ReflexActionNotify:
		subject := fmt.Sprintf("Reflex: %s ausgeloest", rule.Name)
		body := fmt.Sprintf("Regel '%s' wurde auf Node '%s' ausgeloest.\nMetrik: %s %s %g",
			rule.Name, n.Name, rule.TriggerMetric, rule.Operator, rule.Threshold)
		s.notifSvc.Notify(ctx, "reflex_triggered", subject, body)

	case model.ReflexActionRunCommand:
		cmd, _ := config["command"].(string)
		if cmd == "" {
			slog.Warn("reflex: run_command missing command", slog.String("rule", rule.Name))
			return
		}
		result, err := s.nodeSvc.RunSSHCommand(ctx, n.ID, cmd)
		if err != nil {
			slog.Warn("reflex: run_command failed", slog.String("rule", rule.Name), slog.Any("error", err))
			return
		}
		slog.Info("reflex: command executed", slog.String("rule", rule.Name), slog.String("output", result.Stdout))

	case model.ReflexActionStartVM:
		vmidFloat, _ := config["vmid"].(float64)
		vmType, _ := config["vm_type"].(string)
		if vmidFloat == 0 || vmType == "" {
			slog.Warn("reflex: start_vm missing vmid or vm_type", slog.String("rule", rule.Name))
			return
		}
		_, err := s.nodeSvc.StartVM(ctx, n.ID, int(vmidFloat), vmType)
		if err != nil {
			slog.Warn("reflex: start_vm failed", slog.String("rule", rule.Name), slog.Any("error", err))
			return
		}
		slog.Info("reflex: VM started", slog.String("rule", rule.Name), slog.Int("vmid", int(vmidFloat)))

	case model.ReflexActionStopVM:
		vmidFloat, _ := config["vmid"].(float64)
		vmType, _ := config["vm_type"].(string)
		if vmidFloat == 0 || vmType == "" {
			slog.Warn("reflex: stop_vm missing vmid or vm_type", slog.String("rule", rule.Name))
			return
		}
		_, err := s.nodeSvc.StopVM(ctx, n.ID, int(vmidFloat), vmType)
		if err != nil {
			slog.Warn("reflex: stop_vm failed", slog.String("rule", rule.Name), slog.Any("error", err))
			return
		}
		slog.Info("reflex: VM stopped", slog.String("rule", rule.Name), slog.Int("vmid", int(vmidFloat)))
	}
}

func extractMetricValue(record *model.MetricsRecord, metric string) *float64 {
	var val float64
	switch metric {
	case "cpu_usage":
		val = record.CPUUsage
	case "memory_usage":
		if record.MemTotal == 0 {
			return nil
		}
		val = float64(record.MemUsed) / float64(record.MemTotal) * 100
	case "disk_usage":
		if record.DiskTotal == 0 {
			return nil
		}
		val = float64(record.DiskUsed) / float64(record.DiskTotal) * 100
	case "load_avg":
		if len(record.LoadAvg) == 0 {
			return nil
		}
		val = record.LoadAvg[0]
	default:
		return nil
	}
	return &val
}

func evaluateCondition(value float64, operator string, threshold float64) bool {
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
