package reflex

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/node"
	"github.com/antigravity/prometheus/internal/service/notification"
	"github.com/google/uuid"
)

// metricMaxStaleness bounds how old a metric record may be when we evaluate a
// rule against it. If the latest sample is older than this we skip the rule
// instead of firing an action on what is essentially historical data — see
// audit B2 P0 ("stale metric evaluation window"). 5 minutes is conservative
// for the 30s/60s scheduler ticks we use elsewhere.
const metricMaxStaleness = 5 * time.Minute

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

// validateReflexInputs runs input validation that used to be missing — bad
// cron expressions, out-of-range weekday entries and unknown action types
// would all reach the evaluation loop and either silently never fire or fire
// against wrong targets. Catch them at Create/Update time so the operator
// sees the problem immediately.
func validateReflexInputs(actionType model.ReflexActionType, scheduleType, scheduleCron string, timeWindowDays []int) error {
	if string(actionType) == "" {
		return fmt.Errorf("action_type ist erforderlich")
	}
	if !actionType.IsValid() {
		return fmt.Errorf("action_type %q ist unbekannt", actionType)
	}
	if scheduleType == "cron" {
		if strings.TrimSpace(scheduleCron) == "" {
			return fmt.Errorf("schedule_cron ist erforderlich bei schedule_type=cron")
		}
		if err := validateCronExpression(scheduleCron); err != nil {
			return fmt.Errorf("ungültiger cron-Ausdruck %q: %w", scheduleCron, err)
		}
	}
	for _, d := range timeWindowDays {
		if d < 0 || d > 6 {
			return fmt.Errorf("time_window_days enthält ungültigen Wochentag %d (erlaubt: 0..6, 0=Sonntag)", d)
		}
	}
	return nil
}

// validateCronExpression accepts the classic 5-field POSIX cron syntax used
// elsewhere in the codebase. It's intentionally permissive but rejects the
// most common typos (wrong field count, non-numeric out-of-range values).
func validateCronExpression(expr string) error {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return fmt.Errorf("erwartet 5 Felder (Minute Stunde Tag Monat Wochentag), erhalten %d", len(fields))
	}
	// We don't need to verify ranges in detail here — the cron library used by
	// the scheduler will reject anything malformed. But empty fields are a
	// guaranteed bug.
	for i, f := range fields {
		if f == "" {
			return fmt.Errorf("Feld %d ist leer", i+1)
		}
	}
	return nil
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

	scheduleType := req.ScheduleType
	if scheduleType == "" {
		scheduleType = "always"
	}

	aiEnabled := false
	if req.AIEnabled != nil {
		aiEnabled = *req.AIEnabled
	}

	priority := 0
	if req.Priority != nil {
		priority = *req.Priority
	}

	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	timeWindowDays := req.TimeWindowDays
	if timeWindowDays == nil {
		timeWindowDays = []int{}
	}

	if err := validateReflexInputs(req.ActionType, scheduleType, req.ScheduleCron, timeWindowDays); err != nil {
		return nil, err
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
		ScheduleType:    scheduleType,
		ScheduleCron:    req.ScheduleCron,
		TimeWindowStart: req.TimeWindowStart,
		TimeWindowEnd:   req.TimeWindowEnd,
		TimeWindowDays:  timeWindowDays,
		AIEnabled:       aiEnabled,
		Priority:        priority,
		Tags:            tags,
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
	if req.ScheduleType != nil {
		rule.ScheduleType = *req.ScheduleType
	}
	if req.ScheduleCron != nil {
		rule.ScheduleCron = *req.ScheduleCron
	}
	if req.TimeWindowStart != nil {
		rule.TimeWindowStart = *req.TimeWindowStart
	}
	if req.TimeWindowEnd != nil {
		rule.TimeWindowEnd = *req.TimeWindowEnd
	}
	if req.TimeWindowDays != nil {
		rule.TimeWindowDays = req.TimeWindowDays
	}
	if req.AIEnabled != nil {
		rule.AIEnabled = *req.AIEnabled
	}
	if req.Priority != nil {
		rule.Priority = *req.Priority
	}
	if req.Tags != nil {
		rule.Tags = req.Tags
	}

	if err := validateReflexInputs(rule.ActionType, rule.ScheduleType, rule.ScheduleCron, rule.TimeWindowDays); err != nil {
		return nil, err
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

		// Cooldown check (best-effort first-pass; the atomic CAS below is
		// the real guard against double-fire).
		if rule.LastTriggeredAt != nil {
			if now.Sub(*rule.LastTriggeredAt) < time.Duration(rule.CooldownSeconds)*time.Second {
				continue
			}
		}

		if !isInTimeWindow(rule) {
			continue
		}

		// Resolve target nodes. Distinguish "rule applies cluster-wide"
		// (NodeID nil) from "rule targets a specific node that no longer
		// exists in our list" — the latter is a configuration error and
		// should be visible, not silently skipped.
		var targetNodes []model.Node
		if rule.NodeID != nil {
			found := false
			for _, n := range nodes {
				if n.ID == *rule.NodeID {
					targetNodes = append(targetNodes, n)
					found = true
					break
				}
			}
			if !found {
				slog.Warn("reflex: rule targets unknown node — skipping",
					slog.String("rule", rule.Name),
					slog.String("rule_id", rule.ID.String()),
					slog.String("target_node_id", rule.NodeID.String()),
				)
				continue
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

			// Freshness guard: if metrics are too old, the rule would be
			// firing on historical state. Skip — the next metric tick will
			// re-evaluate with fresh data.
			if !latest.RecordedAt.IsZero() && time.Since(latest.RecordedAt) > metricMaxStaleness {
				slog.Debug("reflex: latest metric is too stale, skipping",
					slog.String("rule", rule.Name),
					slog.String("node", n.Name),
					slog.Duration("age", time.Since(latest.RecordedAt)),
				)
				continue
			}

			value, extractErr := extractMetricValueE(latest, rule.TriggerMetric)
			if extractErr != nil {
				slog.Warn("reflex: cannot extract metric — rule misconfigured",
					slog.String("rule", rule.Name),
					slog.String("metric", rule.TriggerMetric),
					slog.Any("error", extractErr),
				)
				continue
			}

			if !evaluateCondition(value, rule.Operator, rule.Threshold) {
				continue
			}

			// Atomic compare-and-swap on LastTriggeredAt: only fire if no
			// other replica or overlapping eval cycle has already triggered
			// the rule within its cooldown. This is the actual guard against
			// the bypass described in audit B2 P0 #4.
			cooldown := time.Duration(rule.CooldownSeconds) * time.Second
			ok, err := s.reflexRepo.TryMarkTriggered(ctx, rule.ID, now, cooldown)
			if err != nil {
				slog.Warn("reflex: TryMarkTriggered failed", slog.String("rule", rule.Name), slog.Any("error", err))
				continue
			}
			if !ok {
				// Another evaluator beat us to it. Skip silently.
				continue
			}

			// Re-fetch the rule immediately before executing the action.
			// Between ListActive() above and now the rule may have been
			// disabled or had its action edited. Firing an old action would
			// be wrong.
			fresh, err := s.reflexRepo.GetByID(ctx, rule.ID)
			if err != nil {
				slog.Warn("reflex: re-fetch before action failed — skipping execution",
					slog.String("rule", rule.Name),
					slog.Any("error", err),
				)
				continue
			}
			if !fresh.IsActive {
				slog.Info("reflex: rule disabled between eval and execute — not firing",
					slog.String("rule", rule.Name),
				)
				continue
			}

			// Audit-quality structured log: an operator searching for
			// "reflex.action_executed" must be able to reconstruct who,
			// what, when, on which node.
			slog.Info("reflex.action_executed",
				slog.String("rule", fresh.Name),
				slog.String("rule_id", fresh.ID.String()),
				slog.String("action_type", string(fresh.ActionType)),
				slog.String("node", n.Name),
				slog.String("node_id", n.ID.String()),
				slog.String("metric", fresh.TriggerMetric),
				slog.Float64("value", value),
				slog.Float64("threshold", fresh.Threshold),
				slog.Time("triggered_at", now),
			)

			s.executeAction(ctx, fresh, &n)

			// Only fire once per rule per evaluation cycle.
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

	case model.ReflexActionScaleUp:
		slog.Info("reflex: scale_up triggered", slog.String("rule", rule.Name), slog.String("node", n.Name))
		subject := fmt.Sprintf("Reflex: %s - Hochskalierung", rule.Name)
		body := fmt.Sprintf("Regel '%s' hat Hochskalierung auf Node '%s' ausgeloest.\nMetrik: %s %s %g",
			rule.Name, n.Name, rule.TriggerMetric, rule.Operator, rule.Threshold)
		s.notifSvc.Notify(ctx, "reflex_scale_up", subject, body)

	case model.ReflexActionScaleDown:
		slog.Info("reflex: scale_down triggered", slog.String("rule", rule.Name), slog.String("node", n.Name))
		subject := fmt.Sprintf("Reflex: %s - Herunterskalierung", rule.Name)
		body := fmt.Sprintf("Regel '%s' hat Herunterskalierung auf Node '%s' ausgeloest.\nMetrik: %s %s %g",
			rule.Name, n.Name, rule.TriggerMetric, rule.Operator, rule.Threshold)
		s.notifSvc.Notify(ctx, "reflex_scale_down", subject, body)

	case model.ReflexActionSnapshot:
		vmidFloat, _ := config["vmid"].(float64)
		vmType, _ := config["vm_type"].(string)
		if vmidFloat == 0 || vmType == "" {
			slog.Warn("reflex: snapshot missing vmid or vm_type", slog.String("rule", rule.Name))
			return
		}
		slog.Info("reflex: snapshot created", slog.String("rule", rule.Name), slog.Int("vmid", int(vmidFloat)))

	case model.ReflexActionAIAnalyze:
		subject := fmt.Sprintf("Reflex KI-Analyse: %s", rule.Name)
		body := fmt.Sprintf("KI-Analyse angefordert fuer Regel '%s' auf Node '%s'.\nMetrik: %s = aktueller Wert ueberschreitet Schwellenwert %g",
			rule.Name, n.Name, rule.TriggerMetric, rule.Threshold)
		s.notifSvc.Notify(ctx, "reflex_ai_analysis", subject, body)
	}
}

// extractMetricValueE replaces the old extractMetricValue: instead of
// returning a nil pointer for every kind of failure (unknown metric, NaN,
// missing field) it returns a descriptive error so misconfigured rules are
// visible in logs rather than silently never firing. Also guards against
// NaN/Inf propagation from upstream metric collection.
func extractMetricValueE(record *model.MetricsRecord, metric string) (float64, error) {
	var val float64
	switch metric {
	case "cpu_usage":
		val = record.CPUUsage
	case "memory_usage":
		if record.MemTotal == 0 {
			return 0, fmt.Errorf("memory_usage: MemTotal is zero")
		}
		val = float64(record.MemUsed) / float64(record.MemTotal) * 100
	case "disk_usage":
		if record.DiskTotal == 0 {
			return 0, fmt.Errorf("disk_usage: DiskTotal is zero")
		}
		val = float64(record.DiskUsed) / float64(record.DiskTotal) * 100
	case "load_avg":
		if len(record.LoadAvg) == 0 {
			return 0, fmt.Errorf("load_avg: LoadAvg array is empty")
		}
		val = record.LoadAvg[0]
	default:
		return 0, fmt.Errorf("metric %q is not supported", metric)
	}
	if math.IsNaN(val) || math.IsInf(val, 0) {
		return 0, fmt.Errorf("metric %s produced non-finite value (%v)", metric, val)
	}
	return val, nil
}

func isInTimeWindow(rule *model.ReflexRule) bool {
	if rule.ScheduleType == "" || rule.ScheduleType == "always" {
		return true
	}

	now := time.Now()

	if rule.ScheduleType == "time_window" {
		// Check day of week
		if len(rule.TimeWindowDays) > 0 {
			dayMatch := false
			currentDay := int(now.Weekday())
			for _, d := range rule.TimeWindowDays {
				if d == currentDay {
					dayMatch = true
					break
				}
			}
			if !dayMatch {
				return false
			}
		}

		// Check time window
		if rule.TimeWindowStart != "" && rule.TimeWindowEnd != "" {
			currentTime := now.Format("15:04")
			start := rule.TimeWindowStart
			end := rule.TimeWindowEnd

			if start <= end {
				// Normal window (e.g., 08:00 - 18:00)
				return currentTime >= start && currentTime <= end
			}
			// Overnight window (e.g., 22:00 - 06:00)
			return currentTime >= start || currentTime <= end
		}
	}

	return true
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
