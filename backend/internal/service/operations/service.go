package operations

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/antigravity/prometheus/internal/llm"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/proxmox"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
)

type NodeVMService interface {
	GetVMs(ctx context.Context, id uuid.UUID) ([]proxmox.VMInfo, error)
}

type Query struct {
	Limit    int
	Source   string
	Severity string
	Status   string
	NodeID   *uuid.UUID
	UserID   *uuid.UUID
	From     *time.Time
	To       *time.Time
	Query    string
}

type Service struct {
	nodeRepo         repository.NodeRepository
	backupRepo       repository.BackupRepository
	migrationRepo    repository.MigrationRepository
	auditRepo        repository.AuditRepository
	securityRepo     repository.SecurityEventRepository
	anomalyRepo      repository.AnomalyRepository
	predictionRepo   repository.PredictionRepository
	incidentRepo     repository.AlertIncidentRepository
	approvalRepo     repository.ApprovalRepository
	notificationRepo repository.NotificationHistoryRepository
	backupScheduleRepo repository.ScheduleRepository
	logReportScheduleRepo repository.LogReportScheduleRepository
	snapshotPolicyRepo repository.SnapshotPolicyRepository
	vmDependencyRepo repository.VMDependencyRepository
	scheduledActionRepo repository.ScheduledActionRepository
	networkDeviceRepo repository.NetworkDeviceRepository
	networkPortRepo repository.NetworkPortRepository
	nodeVMService    NodeVMService
	llmRegistry      *llm.Registry
}

func NewService(
	nodeRepo repository.NodeRepository,
	backupRepo repository.BackupRepository,
	migrationRepo repository.MigrationRepository,
	auditRepo repository.AuditRepository,
	securityRepo repository.SecurityEventRepository,
	anomalyRepo repository.AnomalyRepository,
	predictionRepo repository.PredictionRepository,
	incidentRepo repository.AlertIncidentRepository,
	approvalRepo repository.ApprovalRepository,
	notificationRepo repository.NotificationHistoryRepository,
	backupScheduleRepo repository.ScheduleRepository,
	logReportScheduleRepo repository.LogReportScheduleRepository,
	snapshotPolicyRepo repository.SnapshotPolicyRepository,
	vmDependencyRepo repository.VMDependencyRepository,
	scheduledActionRepo repository.ScheduledActionRepository,
	networkDeviceRepo repository.NetworkDeviceRepository,
	networkPortRepo repository.NetworkPortRepository,
	nodeVMService NodeVMService,
	llmRegistry *llm.Registry,
) *Service {
	return &Service{
		nodeRepo: nodeRepo,
		backupRepo: backupRepo,
		migrationRepo: migrationRepo,
		auditRepo: auditRepo,
		securityRepo: securityRepo,
		anomalyRepo: anomalyRepo,
		predictionRepo: predictionRepo,
		incidentRepo: incidentRepo,
		approvalRepo: approvalRepo,
		notificationRepo: notificationRepo,
		backupScheduleRepo: backupScheduleRepo,
		logReportScheduleRepo: logReportScheduleRepo,
		snapshotPolicyRepo: snapshotPolicyRepo,
		vmDependencyRepo: vmDependencyRepo,
		scheduledActionRepo: scheduledActionRepo,
		networkDeviceRepo: networkDeviceRepo,
		networkPortRepo: networkPortRepo,
		nodeVMService: nodeVMService,
		llmRegistry: llmRegistry,
	}
}

func (s *Service) ListTasks(ctx context.Context, q Query) ([]model.OperationTask, error) {
	if q.Limit <= 0 || q.Limit > 200 {
		q.Limit = 80
	}

	var tasks []model.OperationTask
	if s.migrationRepo != nil {
		migrations, err := s.migrationRepo.List(ctx)
		if err != nil {
			return nil, err
		}
		for _, m := range migrations {
			status, severity := taskStatusForMigration(m.Status)
			task := model.OperationTask{
				ID: fmt.Sprintf("migration-%s", m.ID),
				Type: "migration",
				Title: fmt.Sprintf("%s migrieren", fallback(m.VMName, fmt.Sprintf("VM %d", m.VMID))),
				Detail: fmt.Sprintf("%s -> %s · %s", m.SourceNodeID, m.TargetNodeID, fallback(m.CurrentStep, string(m.Status))),
				Status: status,
				Severity: severity,
				Progress: m.Progress,
				EntityID: m.ID.String(),
				NodeID: &m.SourceNodeID,
				Href: "/migrations",
				CreatedAt: m.CreatedAt,
			}
			if includeTask(task, q) {
				tasks = append(tasks, task)
			}
		}
	}

	if s.backupRepo != nil {
		backups, err := s.backupRepo.ListAll(ctx)
		if err != nil {
			return nil, err
		}
		for _, b := range backups {
			status, severity := taskStatusForBackup(b.Status)
			progress := 10
			if b.Status == model.BackupStatusRunning {
				progress = 50
			}
			if b.Status == model.BackupStatusCompleted || b.Status == model.BackupStatusFailed {
				progress = 100
			}
			task := model.OperationTask{
				ID: fmt.Sprintf("backup-%s", b.ID),
				Type: "backup",
				Title: fmt.Sprintf("Backup %d auf %s", b.Version, b.NodeID),
				Detail: fmt.Sprintf("%s · %d Dateien%s", b.BackupType, b.FileCount, suffix(b.ErrorMessage)),
				Status: status,
				Severity: severity,
				Progress: progress,
				EntityID: b.ID.String(),
				NodeID: &b.NodeID,
				Href: "/backups",
				CreatedAt: b.CreatedAt,
			}
			if includeTask(task, q) {
				tasks = append(tasks, task)
			}
		}
	}

	if s.incidentRepo != nil {
		incidents, err := s.incidentRepo.List(ctx, 80, 0)
		if err != nil {
			return nil, err
		}
		for _, inc := range incidents {
			if inc.Status == model.IncidentStatusResolved {
				continue
			}
			status := "running"
			severity := "warning"
			if inc.Status == model.IncidentStatusTriggered {
				status = "warning"
			}
			task := model.OperationTask{
				ID: fmt.Sprintf("incident-%s", inc.ID),
				Type: "incident",
				Title: fmt.Sprintf("Incident %s", inc.Status),
				Detail: fmt.Sprintf("Stufe %d · Regel %s", inc.CurrentStep, inc.AlertRuleID),
				Status: status,
				Severity: severity,
				Progress: 25,
				EntityID: inc.ID.String(),
				Href: "/alerts",
				CreatedAt: inc.TriggeredAt,
			}
			if inc.Status == model.IncidentStatusAcknowledged {
				task.Progress = 65
			}
			if includeTask(task, q) {
				tasks = append(tasks, task)
			}
		}
	}

	if s.approvalRepo != nil && q.UserID != nil {
		approvals, err := s.approvalRepo.ListPending(ctx, *q.UserID)
		if err != nil {
			return nil, err
		}
		for _, approval := range approvals {
			task := model.OperationTask{
				ID: "approval-" + approval.ID.String(),
				Type: "approval",
				Title: "Agent-Approval: " + approval.ToolName,
				Detail: "Konversation " + approval.ConversationID.String(),
				Status: "pending",
				Severity: "warning",
				Progress: 10,
				EntityID: approval.ID.String(),
				Href: "/chat",
				CreatedAt: approval.CreatedAt,
			}
			if includeTask(task, q) {
				tasks = append(tasks, task)
			}
		}
	}

	if s.scheduledActionRepo != nil {
		actions, err := s.scheduledActionRepo.ListActive(ctx)
		if err != nil {
			return nil, err
		}
		for _, action := range actions {
			detail := action.ScheduleCron
			if action.Description != "" {
				detail += " · " + action.Description
			}
			task := model.OperationTask{
				ID: "scheduled-action-" + action.ID.String(),
				Type: "scheduled_action",
				Title: "Geplante Aktion: " + action.Action,
				Detail: detail,
				Status: "pending",
				Severity: "info",
				Progress: 0,
				EntityID: action.ID.String(),
				NodeID: &action.NodeID,
				Href: fmt.Sprintf("/nodes/%s/vms", action.NodeID),
				CreatedAt: action.CreatedAt,
			}
			if includeTask(task, q) {
				tasks = append(tasks, task)
			}
		}
	}

	if err := s.appendScheduledJobTasks(ctx, &tasks, q); err != nil {
		return nil, err
	}

	if s.notificationRepo != nil {
		entries, err := s.notificationRepo.List(ctx, 80, 0)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.Status != model.NotifStatusFailed && entry.Status != model.NotifStatusPending {
				continue
			}
			status := "pending"
			severity := "info"
			progress := 15
			if entry.Status == model.NotifStatusFailed {
				status = "failed"
				severity = "warning"
				progress = 100
			}
			task := model.OperationTask{
				ID: fmt.Sprintf("notification-%s", entry.ID),
				Type: "notification",
				Title: fallback(entry.Subject, "Benachrichtigung"),
				Detail: fmt.Sprintf("%s · %s", entry.EventType, fallback(entry.ErrorMessage, string(entry.Status))),
				Status: status,
				Severity: severity,
				Progress: progress,
				EntityID: entry.ID.String(),
				Href: "/settings/notifications",
				CreatedAt: entry.CreatedAt,
			}
			if includeTask(task, q) {
				tasks = append(tasks, task)
			}
		}
	}

	sort.Slice(tasks, func(i, j int) bool { return tasks[i].CreatedAt.After(tasks[j].CreatedAt) })
	return limitTasks(tasks, q.Limit), nil
}

func (s *Service) appendScheduledJobTasks(ctx context.Context, tasks *[]model.OperationTask, q Query) error {
	if s.backupScheduleRepo != nil && s.nodeRepo != nil {
		nodes, err := s.nodeRepo.List(ctx)
		if err != nil {
			return err
		}
		for _, node := range nodes {
			schedules, err := s.backupScheduleRepo.ListByNode(ctx, node.ID)
			if err != nil {
				return err
			}
			for _, schedule := range schedules {
				if !schedule.IsActive {
					continue
				}
				task := model.OperationTask{
					ID: "backup-schedule-" + schedule.ID.String(),
					Type: "scheduled_job",
					Title: "Backup-Job: " + node.Name,
					Detail: fmt.Sprintf("%s - Retention %d", schedule.CronExpression, schedule.RetentionCount),
					Status: scheduledStatus(schedule.NextRunAt),
					Severity: "info",
					Progress: 0,
					EntityID: schedule.ID.String(),
					NodeID: &node.ID,
					Href: "/backups",
					DueAt: schedule.NextRunAt,
					CreatedAt: schedule.CreatedAt,
				}
				if includeTask(task, q) {
					*tasks = append(*tasks, task)
				}
			}
		}
	}

	if s.logReportScheduleRepo != nil {
		schedules, err := s.logReportScheduleRepo.List(ctx)
		if err != nil {
			return err
		}
		for _, schedule := range schedules {
			if !schedule.IsActive {
				continue
			}
			task := model.OperationTask{
				ID: "log-report-schedule-" + schedule.ID.String(),
				Type: "scheduled_report",
				Title: "Geplanter Log-Report",
				Detail: fmt.Sprintf("%s - %d Nodes - %dh Fenster", schedule.CronExpression, len(schedule.NodeIDs), schedule.TimeWindowHours),
				Status: scheduledStatus(schedule.NextRunAt),
				Severity: "info",
				Progress: 0,
				EntityID: schedule.ID.String(),
				Href: "/reports",
				DueAt: schedule.NextRunAt,
				CreatedAt: schedule.CreatedAt,
			}
			if includeTask(task, q) {
				*tasks = append(*tasks, task)
			}
		}
	}

	if s.snapshotPolicyRepo != nil {
		policies, err := s.snapshotPolicyRepo.ListActive(ctx)
		if err != nil {
			return err
		}
		for _, policy := range policies {
			task := model.OperationTask{
				ID: "snapshot-policy-" + policy.ID.String(),
				Type: "scheduled_job",
				Title: fmt.Sprintf("Snapshot-Policy: %s", policy.Name),
				Detail: fmt.Sprintf("VM %d - %s", policy.VMID, policy.ScheduleCron),
				Status: "pending",
				Severity: "info",
				Progress: 0,
				EntityID: policy.ID.String(),
				NodeID: &policy.NodeID,
				Href: fmt.Sprintf("/nodes/%s/vms", policy.NodeID),
				CreatedAt: policy.CreatedAt,
			}
			if includeTask(task, q) {
				*tasks = append(*tasks, task)
			}
		}
	}
	return nil
}

func (s *Service) Timeline(ctx context.Context, q Query) ([]model.TimelineEvent, error) {
	if q.Limit <= 0 || q.Limit > 300 {
		q.Limit = 120
	}

	var events []model.TimelineEvent
	if s.auditRepo != nil {
		entries, err := s.auditRepo.ListWithAgentActions(ctx, q.Limit, 0)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			event := model.TimelineEvent{
				ID: fmt.Sprintf("audit-%s", entry.ID),
				Source: "audit",
				Severity: severityForAudit(entry),
				Title: fmt.Sprintf("%s %d", entry.Method, entry.StatusCode),
				Detail: entry.Path,
				Actor: fallback(entry.Username, "System"),
				EntityID: entry.ID.String(),
				Href: "/settings/audit-log",
				CreatedAt: entry.CreatedAt,
			}
			if entry.APITokenID != nil && event.Actor == "System" {
				event.Actor = "API-Token"
			}
			if includeTimeline(event, q) {
				events = append(events, event)
			}
		}
	}

	if s.securityRepo != nil {
		securityEvents, err := s.securityRepo.ListRecent(ctx, q.Limit)
		if err != nil {
			return nil, err
		}
		for _, sec := range securityEvents {
			event := model.TimelineEvent{
				ID: fmt.Sprintf("security-%s", sec.ID),
				Source: "security",
				Severity: normalizeSeverity(sec.Severity),
				Title: sec.Title,
				Detail: fmt.Sprintf("%s · %s", fallback(sec.NodeName, sec.NodeID.String()), sec.Category),
				Actor: fallback(sec.AnalysisModel, "Security Engine"),
				EntityID: sec.ID.String(),
				NodeID: &sec.NodeID,
				Href: "/security",
				CreatedAt: sec.DetectedAt,
			}
			if includeTimeline(event, q) {
				events = append(events, event)
			}
		}
	}

	tasks, err := s.ListTasks(ctx, Query{Limit: q.Limit, UserID: q.UserID, From: q.From, To: q.To})
	if err != nil {
		return nil, err
	}
	for _, task := range tasks {
		source := task.Type
		if source == "incident" {
			source = "alert"
		}
		event := model.TimelineEvent{
			ID: fmt.Sprintf("task-%s", task.ID),
			Source: source,
			Severity: task.Severity,
			Title: task.Title,
			Detail: task.Detail,
			Actor: "Prometheus",
			EntityID: task.EntityID,
			NodeID: task.NodeID,
			Href: task.Href,
			CreatedAt: task.CreatedAt,
		}
		if includeTimeline(event, q) {
			events = append(events, event)
		}
	}

	sort.Slice(events, func(i, j int) bool { return events[i].CreatedAt.After(events[j].CreatedAt) })
	return limitTimeline(events, q.Limit), nil
}

func (s *Service) AnalyzeRCA(ctx context.Context, req model.RCAAnalyzeRequest) (*model.RCAAnalyzeResponse, error) {
	limit := req.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	candidates, err := s.buildCandidates(ctx, req.NodeID)
	if err != nil {
		return nil, err
	}
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	timeline, err := s.Timeline(ctx, Query{Limit: 30, NodeID: req.NodeID})
	if err != nil {
		return nil, err
	}
	summary := fallback(req.Prompt, "Root-Cause-Analyse")
	summary = fmt.Sprintf("%s: %d Ursachen-Kandidaten, %d Timeline-Ereignisse.", summary, len(candidates), len(timeline))
	modelUsed := ""
	if req.UseLLM && s.llmRegistry != nil {
		if generated, modelName := s.generateLLMSummary(ctx, req.Model, "RCA", candidates, timeline); generated != "" {
			summary = generated
			modelUsed = modelName
		}
	}
	return &model.RCAAnalyzeResponse{
		Summary: summary,
		ModelUsed: modelUsed,
		Candidates: candidates,
		Timeline: timeline,
		GeneratedAt: time.Now(),
	}, nil
}

func (s *Service) KnowledgeGraph(ctx context.Context) (*model.KnowledgeGraphResponse, error) {
	nodes, err := s.nodeRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	graphNodes := make([]model.KnowledgeGraphNode, 0)
	edges := make([]model.KnowledgeGraphEdge, 0)
	stats := model.KnowledgeGraphStats{}

	for _, n := range nodes {
		nodeID := n.ID
		status := "offline"
		if n.IsOnline {
			status = "online"
		}
		graphNodes = append(graphNodes, model.KnowledgeGraphNode{
			ID: "node:" + n.ID.String(),
			Type: "node",
			Label: n.Name,
			Status: status,
			NodeID: &nodeID,
			Metadata: map[string]string{"hostname": n.Hostname, "type": string(n.Type)},
		})
		stats.Nodes++

		if s.nodeVMService != nil && n.IsOnline {
			if vms, vmErr := s.nodeVMService.GetVMs(ctx, n.ID); vmErr == nil {
				for _, vm := range vms {
					vmNodeID := fmt.Sprintf("vm:%s:%d", n.ID, vm.VMID)
					graphNodes = append(graphNodes, model.KnowledgeGraphNode{
						ID: vmNodeID,
						Type: "vm",
						Label: fallback(vm.Name, fmt.Sprintf("VM %d", vm.VMID)),
						Status: vm.Status,
						NodeID: &nodeID,
						Metadata: map[string]string{"vmid": fmt.Sprintf("%d", vm.VMID), "type": vm.Type},
					})
					edges = append(edges, model.KnowledgeGraphEdge{
						ID: "hosts:" + n.ID.String() + ":" + fmt.Sprintf("%d", vm.VMID),
						From: "node:" + n.ID.String(),
						To: vmNodeID,
						Type: "hosts",
						Label: "hostet",
					})
					stats.VMs++
				}
			}
		}

		if s.networkDeviceRepo != nil && s.networkPortRepo != nil {
			devices, _ := s.networkDeviceRepo.ListByNode(ctx, n.ID)
			for _, device := range devices {
				deviceID := "device:" + device.ID.String()
				graphNodes = append(graphNodes, model.KnowledgeGraphNode{
					ID: deviceID,
					Type: "device",
					Label: fallback(device.Hostname, device.IP),
					Status: knownStatus(device.IsKnown),
					NodeID: &nodeID,
					Metadata: map[string]string{"ip": device.IP, "mac": device.MAC},
				})
				edges = append(edges, model.KnowledgeGraphEdge{
					ID: "sees:" + n.ID.String() + ":" + device.ID.String(),
					From: "node:" + n.ID.String(),
					To: deviceID,
					Type: "sees",
					Label: "sieht",
				})
				stats.Devices++
				ports, _ := s.networkPortRepo.ListByDevice(ctx, device.ID)
				for _, port := range ports {
					serviceID := fmt.Sprintf("service:%s:%d:%s", device.ID, port.Port, port.Protocol)
					graphNodes = append(graphNodes, model.KnowledgeGraphNode{
						ID: serviceID,
						Type: "service",
						Label: fallback(port.ServiceName, fmt.Sprintf("%s/%d", port.Protocol, port.Port)),
						Status: port.State,
						NodeID: &nodeID,
						Metadata: map[string]string{"port": fmt.Sprintf("%d", port.Port), "protocol": port.Protocol, "version": port.ServiceVersion},
					})
					edges = append(edges, model.KnowledgeGraphEdge{
						ID: "exposes:" + device.ID.String() + ":" + fmt.Sprintf("%d", port.Port) + ":" + port.Protocol,
						From: deviceID,
						To: serviceID,
						Type: "exposes",
						Label: "exponiert",
						Status: port.State,
					})
					stats.Services++
				}
			}
		}
	}

	if s.vmDependencyRepo != nil {
		deps, err := s.vmDependencyRepo.List(ctx)
		if err != nil {
			return nil, err
		}
		for _, dep := range deps {
			edges = append(edges, model.KnowledgeGraphEdge{
				ID: "depends:" + dep.ID.String(),
				From: fmt.Sprintf("vm:%s:%d", dep.SourceNodeID, dep.SourceVMID),
				To: fmt.Sprintf("vm:%s:%d", dep.TargetNodeID, dep.TargetVMID),
				Type: "depends_on",
				Label: fallback(dep.DependencyType, "depends_on"),
			})
			stats.Dependencies++
		}
	}

	return &model.KnowledgeGraphResponse{
		Nodes: graphNodes,
		Edges: edges,
		Stats: stats,
		GeneratedAt: time.Now(),
	}, nil
}

func (s *Service) GenerateReport(ctx context.Context, req model.OperationsReportRequest) (*model.OperationsReportResponse, error) {
	events, err := s.Timeline(ctx, Query{Limit: 120, Severity: req.Severity, Query: req.Query})
	if err != nil {
		return nil, err
	}
	events = filterReportEvents(events, req.Domain)
	candidates, err := s.buildCandidates(ctx, nil)
	if err != nil {
		return nil, err
	}
	counts := map[string]int{"security": 0, "capacity": 0, "operations": 0, "timeline": len(events)}
	for _, event := range events {
		switch event.Source {
		case "security":
			counts["security"]++
		case "audit", "migration", "backup", "notification", "alert":
			counts["operations"]++
		default:
			counts[event.Source]++
		}
	}
	for _, candidate := range candidates {
		if strings.Contains(strings.ToLower(candidate.Title), "anomalie") || strings.Contains(strings.ToLower(candidate.Title), "schwelle") {
			counts["capacity"]++
		}
	}

	text := s.buildReportText(req, events, candidates, counts)
	modelUsed := ""
	if req.UseLLM && s.llmRegistry != nil {
		if generated, modelName := s.generateReportWithLLM(ctx, req.Model, req.Prompt, events, candidates, counts); generated != "" {
			text = generated
			modelUsed = modelName
		}
	}
	return &model.OperationsReportResponse{Text: text, ModelUsed: modelUsed, Counts: counts, GeneratedAt: time.Now()}, nil
}

func (s *Service) buildCandidates(ctx context.Context, nodeID *uuid.UUID) ([]model.RCACandidate, error) {
	nodes, err := s.nodeRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	var candidates []model.RCACandidate
	for _, node := range nodes {
		if nodeID != nil && node.ID != *nodeID {
			continue
		}
		if !node.IsOnline {
			nid := node.ID
			candidates = append(candidates, model.RCACandidate{
				ID: "offline-" + node.ID.String(),
				Title: node.Name + " ist offline",
				Severity: "critical",
				NodeID: &nid,
				Evidence: []string{"Node meldet sich nicht online", "Letzter Kontakt: " + timeString(node.LastSeen)},
				Recommendation: "Node-Verbindung, API-Token, Netzwerkpfad und Proxmox-Service pruefen.",
				Href: "/nodes/" + node.ID.String(),
			})
		}
	}

	if s.securityRepo != nil {
		events, err := s.securityRepo.ListRecent(ctx, 80)
		if err != nil {
			return nil, err
		}
		for _, event := range events {
			if nodeID != nil && event.NodeID != *nodeID {
				continue
			}
			if event.IsAcknowledged {
				continue
			}
			if event.Severity != "critical" && event.Severity != "emergency" && event.Severity != "warning" {
				continue
			}
			nid := event.NodeID
			candidates = append(candidates, model.RCACandidate{
				ID: "security-" + event.ID.String(),
				Title: event.Title,
				Severity: normalizeSeverity(event.Severity),
				NodeID: &nid,
				Evidence: compactEvidence(event.Description, event.Impact),
				Recommendation: fallback(event.Recommendation, "Security-Ereignis validieren und bestaetigen."),
				Href: "/security",
			})
		}
	}

	if s.anomalyRepo != nil {
		anomalies, err := s.anomalyRepo.ListUnresolved(ctx)
		if err != nil {
			return nil, err
		}
		for _, anomaly := range anomalies {
			if nodeID != nil && anomaly.NodeID != *nodeID {
				continue
			}
			nid := anomaly.NodeID
			candidates = append(candidates, model.RCACandidate{
				ID: "anomaly-" + anomaly.ID.String(),
				Title: fmt.Sprintf("%s Anomalie", anomaly.Metric),
				Severity: normalizeSeverity(anomaly.Severity),
				NodeID: &nid,
				Evidence: compactEvidence(fmt.Sprintf("Wert %.2f", anomaly.Value), fmt.Sprintf("Z-Score %.2f", anomaly.ZScore)),
				Recommendation: fallback(anomaly.Recommendation, "Metriken mit Flight Recorder und aktuellen Changes korrelieren."),
				Href: "/monitoring",
			})
		}
	}

	if s.predictionRepo != nil {
		predictions, err := s.predictionRepo.ListCritical(ctx)
		if err != nil {
			return nil, err
		}
		for _, prediction := range predictions {
			if nodeID != nil && prediction.NodeID != *nodeID {
				continue
			}
			nid := prediction.NodeID
			evidence := []string{fmt.Sprintf("Aktuell %.2f", prediction.CurrentValue), fmt.Sprintf("Prognose %.2f", prediction.PredictedValue)}
			if prediction.DaysUntilThreshold != nil {
				evidence = append(evidence, fmt.Sprintf("%.0f Tage bis Schwelle", *prediction.DaysUntilThreshold))
			}
			candidates = append(candidates, model.RCACandidate{
				ID: "prediction-" + prediction.ID.String(),
				Title: prediction.Metric + " erreicht Schwelle",
				Severity: normalizeSeverity(prediction.Severity),
				NodeID: &nid,
				Evidence: evidence,
				Recommendation: fallback(prediction.Recommendation, "Kapazitaetsplanung ausloesen und betroffene VMs pruefen."),
				Href: "/recommendations",
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return severityRank(candidates[i].Severity) > severityRank(candidates[j].Severity)
	})
	return candidates, nil
}

func (s *Service) buildReportText(req model.OperationsReportRequest, events []model.TimelineEvent, candidates []model.RCACandidate, counts map[string]int) string {
	lines := []string{
		fallback(req.Prompt, "Betriebsbericht"),
		"",
		fmt.Sprintf("Scope: %s, Severity: %s, Suche: %s", fallback(req.Domain, "all"), fallback(req.Severity, "all"), fallback(req.Query, "-")),
		fmt.Sprintf("Timeline-Ereignisse: %d, Ursachen-Kandidaten: %d", len(events), len(candidates)),
		"",
		"Zaehler",
		fmt.Sprintf("- Security: %d", counts["security"]),
		fmt.Sprintf("- Kapazitaet: %d", counts["capacity"]),
		fmt.Sprintf("- Operations: %d", counts["operations"]),
		"",
		"Top-Signale",
	}
	if len(candidates) == 0 {
		lines = append(lines, "- Keine offenen Ursachen-Kandidaten.")
	} else {
		for _, candidate := range firstCandidates(candidates, 8) {
			lines = append(lines, fmt.Sprintf("- %s: %s", candidate.Severity, candidate.Title))
		}
	}
	lines = append(lines, "", "Aktuelle Timeline")
	if len(events) == 0 {
		lines = append(lines, "- Keine passenden Timeline-Ereignisse.")
	} else {
		for _, event := range firstEvents(events, 10) {
			lines = append(lines, fmt.Sprintf("- %s: %s (%s)", event.Source, event.Title, event.Severity))
		}
	}
	lines = append(lines, "", "Empfohlene Reihenfolge", "- Kritische Security- und Offline-Signale zuerst bestaetigen.", "- Danach fehlgeschlagene Jobs mit Flight Recorder gegen letzte Changes pruefen.", "- Kapazitaets-Signale in Monitoring und Recommendations vertiefen.")
	return strings.Join(lines, "\n")
}

func (s *Service) generateLLMSummary(ctx context.Context, modelName, title string, candidates []model.RCACandidate, timeline []model.TimelineEvent) (string, string) {
	if modelName == "" {
		modelName = s.llmRegistry.DefaultModel()
	}
	provider, err := s.llmRegistry.GetForModel(modelName)
	if err != nil {
		return "", ""
	}
	payload, _ := json.Marshal(struct {
		Candidates []model.RCACandidate `json:"candidates"`
		Timeline []model.TimelineEvent `json:"timeline"`
	}{Candidates: firstCandidates(candidates, 12), Timeline: firstEvents(timeline, 12)})
	resp, err := provider.Complete(ctx, llm.CompletionRequest{
		Model: modelName,
		Messages: []llm.Message{{Role: "user", Content: fmt.Sprintf("Erstelle eine knappe deutsche %s mit Ursache, Evidenz und naechsten Schritten. Daten: %s", title, string(payload))}},
		MaxTokens: 500,
		Temperature: 0.2,
	})
	if err != nil {
		return "", ""
	}
	return resp.Content, modelName
}

func (s *Service) generateReportWithLLM(ctx context.Context, modelName, prompt string, events []model.TimelineEvent, candidates []model.RCACandidate, counts map[string]int) (string, string) {
	if modelName == "" {
		modelName = s.llmRegistry.DefaultModel()
	}
	provider, err := s.llmRegistry.GetForModel(modelName)
	if err != nil {
		return "", ""
	}
	payload, _ := json.Marshal(struct {
		Prompt string `json:"prompt"`
		Counts map[string]int `json:"counts"`
		Events []model.TimelineEvent `json:"events"`
		Candidates []model.RCACandidate `json:"candidates"`
	}{Prompt: prompt, Counts: counts, Events: firstEvents(events, 16), Candidates: firstCandidates(candidates, 12)})
	resp, err := provider.Complete(ctx, llm.CompletionRequest{
		Model: modelName,
		Messages: []llm.Message{{Role: "user", Content: "Erstelle einen strukturierten deutschen Infrastruktur-Report mit Lage, Risiken und naechsten Schritten. Daten: " + string(payload)}},
		MaxTokens: 700,
		Temperature: 0.2,
	})
	if err != nil {
		return "", ""
	}
	return resp.Content, modelName
}

func includeTask(task model.OperationTask, q Query) bool {
	if q.Status != "" && q.Status != "all" && task.Status != q.Status {
		return false
	}
	if q.Source != "" && q.Source != "all" && task.Type != q.Source {
		return false
	}
	if q.Severity != "" && q.Severity != "all" && task.Severity != q.Severity {
		return false
	}
	if q.NodeID != nil && (task.NodeID == nil || *task.NodeID != *q.NodeID) {
		return false
	}
	if !inTimeRange(task.CreatedAt, q.From, q.To) {
		return false
	}
	if q.Query != "" && !strings.Contains(strings.ToLower(task.Title+" "+task.Detail), strings.ToLower(q.Query)) {
		return false
	}
	return true
}

func includeTimeline(event model.TimelineEvent, q Query) bool {
	if q.Source != "" && q.Source != "all" && event.Source != q.Source {
		return false
	}
	if q.Severity != "" && q.Severity != "all" && event.Severity != q.Severity {
		return false
	}
	if q.NodeID != nil && (event.NodeID == nil || *event.NodeID != *q.NodeID) {
		return false
	}
	if !inTimeRange(event.CreatedAt, q.From, q.To) {
		return false
	}
	if q.Query != "" && !strings.Contains(strings.ToLower(event.Title+" "+event.Detail+" "+event.Actor), strings.ToLower(q.Query)) {
		return false
	}
	return true
}

func inTimeRange(value time.Time, from, to *time.Time) bool {
	if from != nil && value.Before(*from) {
		return false
	}
	if to != nil && value.After(*to) {
		return false
	}
	return true
}

func filterReportEvents(events []model.TimelineEvent, domain string) []model.TimelineEvent {
	if domain == "" || domain == "all" {
		return events
	}
	filtered := make([]model.TimelineEvent, 0, len(events))
	for _, event := range events {
		switch domain {
		case "security":
			if event.Source == "security" {
				filtered = append(filtered, event)
			}
		case "operations":
			if event.Source == "audit" || event.Source == "migration" || event.Source == "backup" || event.Source == "notification" || event.Source == "alert" {
				filtered = append(filtered, event)
			}
		case "capacity":
			if event.Source == "anomaly" || event.Source == "prediction" {
				filtered = append(filtered, event)
			}
		default:
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func taskStatusForMigration(status model.MigrationStatus) (string, string) {
	switch status {
	case model.MigrationStatusFailed, model.MigrationStatusCancelled:
		return "failed", "critical"
	case model.MigrationStatusCompleted:
		return "completed", "info"
	default:
		return "running", "warning"
	}
}

func taskStatusForBackup(status model.BackupStatus) (string, string) {
	switch status {
	case model.BackupStatusPending:
		return "pending", "info"
	case model.BackupStatusRunning:
		return "running", "warning"
	case model.BackupStatusFailed:
		return "failed", "critical"
	default:
		return "completed", "info"
	}
}

func severityForAudit(entry model.AuditLogEntry) string {
	if entry.StatusCode >= 500 {
		return "critical"
	}
	if entry.StatusCode >= 400 || entry.Method == "DELETE" {
		return "warning"
	}
	return "info"
}

func normalizeSeverity(severity string) string {
	switch strings.ToLower(severity) {
	case "critical", "emergency":
		return "critical"
	case "warning", "medium":
		return "warning"
	default:
		return "info"
	}
}

func severityRank(severity string) int {
	switch severity {
	case "critical":
		return 3
	case "warning":
		return 2
	default:
		return 1
	}
}

func fallback(value, fallbackValue string) string {
	if strings.TrimSpace(value) == "" {
		return fallbackValue
	}
	return value
}

func suffix(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return " · " + value
}

func compactEvidence(values ...string) []string {
	var result []string
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, value)
		}
	}
	return result
}

func timeString(value *time.Time) string {
	if value == nil {
		return "unbekannt"
	}
	return value.Format(time.RFC3339)
}

func knownStatus(isKnown bool) string {
	if isKnown {
		return "known"
	}
	return "unknown"
}

func scheduledStatus(nextRun *time.Time) string {
	if nextRun != nil && nextRun.Before(time.Now()) {
		return "running"
	}
	return "pending"
}

func limitTasks(tasks []model.OperationTask, limit int) []model.OperationTask {
	if len(tasks) <= limit {
		return tasks
	}
	return tasks[:limit]
}

func limitTimeline(events []model.TimelineEvent, limit int) []model.TimelineEvent {
	if len(events) <= limit {
		return events
	}
	return events[:limit]
}

func firstEvents(events []model.TimelineEvent, limit int) []model.TimelineEvent {
	if len(events) <= limit {
		return events
	}
	return events[:limit]
}

func firstCandidates(candidates []model.RCACandidate, limit int) []model.RCACandidate {
	if len(candidates) <= limit {
		return candidates
	}
	return candidates[:limit]
}
