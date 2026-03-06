package intelligence

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/antigravity/prometheus/internal/llm"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/proxmox"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/monitor"
)

// NodeServiceInterface allows fetching VMs for context.
type NodeServiceInterface interface {
	GetVMs(ctx context.Context, id uuid.UUID) ([]proxmox.VMInfo, error)
}

// AnalysisMode controls whether to use LLM, rule-based, or both.
type AnalysisMode string

const (
	ModeHybrid   AnalysisMode = "hybrid"    // Rule-based pre-filter + LLM (default)
	ModeFullLLM  AnalysisMode = "full_llm"  // Send ALL metrics to LLM without pre-filter
	ModeRuleOnly AnalysisMode = "rule_only" // Only rule-based, no LLM
)

type Service struct {
	securityRepo   repository.SecurityEventRepository
	metricsRepo    repository.MetricsRepository
	nodeRepo       repository.NodeRepository
	anomalyRepo    repository.AnomalyRepository
	predictionRepo repository.PredictionRepository
	llmRegistry    *llm.Registry
	wsHub          *monitor.WSHub
	nodeSvc        NodeServiceInterface
	mode           AnalysisMode
}

func NewAnalysisService(
	securityRepo repository.SecurityEventRepository,
	metricsRepo repository.MetricsRepository,
	nodeRepo repository.NodeRepository,
	anomalyRepo repository.AnomalyRepository,
	predictionRepo repository.PredictionRepository,
	llmRegistry *llm.Registry,
	wsHub *monitor.WSHub,
) *Service {
	return &Service{
		securityRepo:   securityRepo,
		metricsRepo:    metricsRepo,
		nodeRepo:       nodeRepo,
		anomalyRepo:    anomalyRepo,
		predictionRepo: predictionRepo,
		llmRegistry:    llmRegistry,
		wsHub:          wsHub,
		mode:           ModeHybrid,
	}
}

func (s *Service) SetNodeService(svc NodeServiceInterface) {
	s.nodeSvc = svc
}

func (s *Service) SetMode(mode AnalysisMode) {
	s.mode = mode
}

func (s *Service) GetMode() AnalysisMode {
	return s.mode
}

// nodeContext holds collected data for a single node analysis.
type nodeContext struct {
	Node       *model.Node
	Metrics    []model.MetricsRecord
	VMs        []proxmox.VMInfo
	RunningVMs []string
	Anomalies  []model.AnomalyRecord
	Findings   []finding
}

// finding is a rule-based pre-filter result.
type finding struct {
	Category    string
	Severity    string
	Title       string
	Description string
	Metrics     map[string]float64
}

// llmAnalysisResult is the expected JSON response from the LLM.
type llmAnalysisResult struct {
	Events []llmEvent `json:"events"`
}

type llmEvent struct {
	Category       string `json:"category"`
	Severity       string `json:"severity"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	Impact         string `json:"impact"`
	Recommendation string `json:"recommendation"`
}

// RunAnalysis performs the full analysis cycle for all online nodes.
func (s *Service) RunAnalysis(ctx context.Context) error {
	nodes, err := s.nodeRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("intelligence: list nodes: %w", err)
	}

	now := time.Now()
	since := now.Add(-24 * time.Hour)

	for _, node := range nodes {
		if !node.IsOnline {
			s.checkOfflineNode(ctx, &node)
			continue
		}

		nc, err := s.collectNodeContext(ctx, &node, since, now)
		if err != nil {
			slog.Warn("intelligence: failed to collect context", slog.String("node", node.Name), slog.Any("error", err))
			continue
		}

		var events []model.SecurityEvent

		switch s.mode {
		case ModeFullLLM:
			// Send ALL data to LLM without pre-filter
			events = s.analyzeFullLLM(ctx, nc)
		case ModeRuleOnly:
			// Only rule-based detection
			s.detectPerformanceIssues(nc)
			s.detectCapacityIssues(nc)
			s.detectSecurityAnomalies(nc)
			if len(nc.Findings) > 0 {
				events = s.findingsToEvents(nc)
			}
		default: // ModeHybrid
			// Rule-based pre-filter, then LLM
			s.detectPerformanceIssues(nc)
			s.detectCapacityIssues(nc)
			s.detectSecurityAnomalies(nc)
			if len(nc.Findings) == 0 {
				continue
			}
			events = s.analyzeWithLLM(ctx, nc)
		}

		// Store events and broadcast
		for i := range events {
			events[i].NodeID = node.ID
			events[i].NodeName = node.Name
			events[i].AffectedVMs = nc.RunningVMs

			if err := s.securityRepo.Create(ctx, &events[i]); err != nil {
				slog.Warn("intelligence: failed to store event", slog.String("node", node.Name), slog.Any("error", err))
				continue
			}

			if s.wsHub != nil {
				s.wsHub.BroadcastMessage(monitor.WSMessage{
					Type: "security_event",
					Data: events[i],
				})
			}

			slog.Info("intelligence: event created",
				slog.String("node", node.Name),
				slog.String("category", events[i].Category),
				slog.String("severity", events[i].Severity),
				slog.String("title", events[i].Title))
		}
	}

	return nil
}

func (s *Service) collectNodeContext(ctx context.Context, node *model.Node, since, until time.Time) (*nodeContext, error) {
	nc := &nodeContext{Node: node}

	records, err := s.metricsRepo.GetByNode(ctx, node.ID, since, until)
	if err != nil {
		return nil, fmt.Errorf("get metrics: %w", err)
	}
	nc.Metrics = records

	if s.nodeSvc != nil {
		vms, err := s.nodeSvc.GetVMs(ctx, node.ID)
		if err == nil {
			nc.VMs = vms
			for _, vm := range vms {
				if vm.Status == "running" {
					name := vm.Name
					if name == "" {
						name = fmt.Sprintf("VM %d", vm.VMID)
					}
					nc.RunningVMs = append(nc.RunningVMs, name)
				}
			}
		}
	}

	anomalies, err := s.anomalyRepo.ListByNode(ctx, node.ID)
	if err == nil {
		nc.Anomalies = anomalies
	}

	return nc, nil
}

func (s *Service) checkOfflineNode(ctx context.Context, node *model.Node) {
	event := model.SecurityEvent{
		NodeID:   node.ID,
		NodeName: node.Name,
		Category: "availability",
		Severity: "critical",
		Title:    fmt.Sprintf("Node %s ist offline", node.Name),
		Description: fmt.Sprintf("Der Node %s ist nicht erreichbar. Letzte Verbindung: %s.",
			node.Name, formatTimeAgo(node.LastSeen)),
		Impact:         "Alle VMs auf diesem Node sind nicht erreichbar. Dienste sind unterbrochen.",
		Recommendation: "Pruefen Sie die Netzwerkverbindung und den physischen Server. Ueberpruefen Sie Proxmox-Dienste (systemctl status pveproxy pvedaemon).",
		AnalysisModel:  "rule-based",
	}

	if err := s.securityRepo.Create(ctx, &event); err != nil {
		slog.Warn("intelligence: failed to store offline event", slog.String("node", node.Name), slog.Any("error", err))
		return
	}

	if s.wsHub != nil {
		s.wsHub.BroadcastMessage(monitor.WSMessage{Type: "security_event", Data: event})
	}
}

// detectPerformanceIssues checks for sustained high resource usage.
func (s *Service) detectPerformanceIssues(nc *nodeContext) {
	if len(nc.Metrics) < 15 {
		return
	}

	recent := nc.Metrics[len(nc.Metrics)-15:]

	// Sustained CPU > 85%
	cpuSum := 0.0
	for _, r := range recent {
		cpuSum += r.CPUUsage
	}
	cpuAvg := cpuSum / float64(len(recent))

	if cpuAvg > 85 {
		severity := "warning"
		if cpuAvg > 95 {
			severity = "critical"
		}
		nc.Findings = append(nc.Findings, finding{
			Category:    "performance",
			Severity:    severity,
			Title:       fmt.Sprintf("Anhaltend hohe CPU-Auslastung auf %s (%.1f%%)", nc.Node.Name, cpuAvg),
			Description: fmt.Sprintf("CPU-Auslastung lag in den letzten 15 Minuten durchschnittlich bei %.1f%%.", cpuAvg),
			Metrics:     map[string]float64{"cpu_avg_15m": cpuAvg},
		})
	}

	// Sustained RAM > 85%
	ramSum := 0.0
	validRAM := 0
	for _, r := range recent {
		if r.MemTotal > 0 {
			ramSum += float64(r.MemUsed) / float64(r.MemTotal) * 100
			validRAM++
		}
	}
	if validRAM > 0 {
		ramAvg := ramSum / float64(validRAM)
		if ramAvg > 85 {
			severity := "warning"
			if ramAvg > 95 {
				severity = "critical"
			}
			nc.Findings = append(nc.Findings, finding{
				Category:    "performance",
				Severity:    severity,
				Title:       fmt.Sprintf("Hohe RAM-Auslastung auf %s (%.1f%%)", nc.Node.Name, ramAvg),
				Description: fmt.Sprintf("RAM-Auslastung lag in den letzten 15 Minuten durchschnittlich bei %.1f%%.", ramAvg),
				Metrics:     map[string]float64{"ram_avg_15m": ramAvg},
			})
		}
	}

	// CPU spike without RAM correlation — potential cryptominer
	if cpuAvg > 80 {
		ramAvgNorm := 0.0
		if validRAM > 0 {
			ramAvgNorm = ramSum / float64(validRAM)
		}
		if ramAvgNorm < 40 && cpuAvg > 85 {
			nc.Findings = append(nc.Findings, finding{
				Category:    "security",
				Severity:    "warning",
				Title:       fmt.Sprintf("Verdaechtige CPU-Aktivitaet auf %s", nc.Node.Name),
				Description: fmt.Sprintf("Hohe CPU (%.1f%%) bei niedrigem RAM (%.1f%%). Moegliches Cryptomining oder unerwarteter Rechenprozess.", cpuAvg, ramAvgNorm),
				Metrics:     map[string]float64{"cpu_avg": cpuAvg, "ram_avg": ramAvgNorm},
			})
		}
	}
}

// detectCapacityIssues checks for disk capacity problems.
func (s *Service) detectCapacityIssues(nc *nodeContext) {
	if len(nc.Metrics) < 2 {
		return
	}

	last := nc.Metrics[len(nc.Metrics)-1]
	if last.DiskTotal == 0 {
		return
	}

	diskPct := float64(last.DiskUsed) / float64(last.DiskTotal) * 100

	if diskPct > 85 {
		severity := "warning"
		if diskPct > 95 {
			severity = "emergency"
		} else if diskPct > 90 {
			severity = "critical"
		}
		nc.Findings = append(nc.Findings, finding{
			Category:    "capacity",
			Severity:    severity,
			Title:       fmt.Sprintf("Speicherplatz auf %s bei %.1f%%", nc.Node.Name, diskPct),
			Description: fmt.Sprintf("Disk-Auslastung bei %.1f%%. Verbleibend: %s.", diskPct, formatBytes(last.DiskTotal-last.DiskUsed)),
			Metrics:     map[string]float64{"disk_pct": diskPct},
		})
	}

	// Check disk growth trend
	if len(nc.Metrics) > 100 {
		first := nc.Metrics[0]
		if first.DiskTotal > 0 {
			firstPct := float64(first.DiskUsed) / float64(first.DiskTotal) * 100
			hoursDiff := last.RecordedAt.Sub(first.RecordedAt).Hours()
			if hoursDiff > 0 {
				growthPerDay := (diskPct - firstPct) / hoursDiff * 24
				if growthPerDay > 0.5 && diskPct > 70 {
					nc.Findings = append(nc.Findings, finding{
						Category:    "capacity",
						Severity:    "warning",
						Title:       fmt.Sprintf("Disk-Wachstum auf %s: %.2f%%/Tag", nc.Node.Name, growthPerDay),
						Description: fmt.Sprintf("Speicher waechst um %.2f%% pro Tag. Bei aktuellem Tempo in %.0f Tagen voll.", growthPerDay, (100-diskPct)/growthPerDay),
						Metrics:     map[string]float64{"disk_pct": diskPct, "growth_per_day": growthPerDay},
					})
				}
			}
		}
	}
}

// detectSecurityAnomalies checks for unusual network patterns.
func (s *Service) detectSecurityAnomalies(nc *nodeContext) {
	if len(nc.Metrics) < 30 {
		return
	}

	var totalNetIn, totalNetOut float64
	validNet := 0
	for _, r := range nc.Metrics {
		if r.NetIn > 0 || r.NetOut > 0 {
			totalNetIn += float64(r.NetIn)
			totalNetOut += float64(r.NetOut)
			validNet++
		}
	}

	if validNet == 0 {
		return
	}

	avgNetIn := totalNetIn / float64(validNet)
	avgNetOut := totalNetOut / float64(validNet)

	recent := nc.Metrics[len(nc.Metrics)-5:]
	for _, r := range recent {
		if avgNetIn > 0 && float64(r.NetIn) > avgNetIn*5 {
			nc.Findings = append(nc.Findings, finding{
				Category:    "security",
				Severity:    "warning",
				Title:       fmt.Sprintf("Ungewoehnlicher eingehender Traffic auf %s", nc.Node.Name),
				Description: fmt.Sprintf("Eingehender Netzwerktraffic ist %.1fx hoeher als der Durchschnitt. Moeglicherweise DDoS oder unerwarteter Datentransfer.", float64(r.NetIn)/avgNetIn),
				Metrics:     map[string]float64{"net_in_current": float64(r.NetIn), "net_in_avg": avgNetIn},
			})
			break
		}

		if avgNetOut > 0 && float64(r.NetOut) > avgNetOut*5 {
			nc.Findings = append(nc.Findings, finding{
				Category:    "security",
				Severity:    "warning",
				Title:       fmt.Sprintf("Ungewoehnlicher ausgehender Traffic auf %s", nc.Node.Name),
				Description: fmt.Sprintf("Ausgehender Netzwerktraffic ist %.1fx hoeher als der Durchschnitt. Moeglicherweise Datenexfiltration oder unkontrollierter Backup-Prozess.", float64(r.NetOut)/avgNetOut),
				Metrics:     map[string]float64{"net_out_current": float64(r.NetOut), "net_out_avg": avgNetOut},
			})
			break
		}
	}
}

// analyzeFullLLM sends ALL node data to the LLM without pre-filtering.
// Ideal for local models (Ollama) where LLM calls are free.
func (s *Service) analyzeFullLLM(ctx context.Context, nc *nodeContext) []model.SecurityEvent {
	if s.llmRegistry == nil {
		// Fallback to rule-based
		s.detectPerformanceIssues(nc)
		s.detectCapacityIssues(nc)
		s.detectSecurityAnomalies(nc)
		return s.findingsToEvents(nc)
	}

	modelName := s.llmRegistry.DefaultModel()
	provider, err := s.llmRegistry.GetForModel(modelName)
	if err != nil {
		slog.Warn("intelligence: no LLM provider for full analysis, falling back to rules", slog.Any("error", err))
		s.detectPerformanceIssues(nc)
		s.detectCapacityIssues(nc)
		s.detectSecurityAnomalies(nc)
		return s.findingsToEvents(nc)
	}

	contextData := s.buildFullLLMContext(nc)

	req := llm.CompletionRequest{
		Model: modelName,
		Messages: []llm.Message{
			{Role: "system", Content: fullLLMSystemPrompt},
			{Role: "user", Content: contextData},
		},
		MaxTokens:   3000,
		Temperature: 0.3,
	}

	resp, err := provider.Complete(ctx, req)
	if err != nil {
		slog.Warn("intelligence: full LLM analysis failed, falling back to rules", slog.Any("error", err))
		s.detectPerformanceIssues(nc)
		s.detectCapacityIssues(nc)
		s.detectSecurityAnomalies(nc)
		return s.findingsToEvents(nc)
	}

	events, err := s.parseLLMResponse(resp.Content, modelName)
	if err != nil {
		slog.Warn("intelligence: failed to parse full LLM response", slog.Any("error", err))
		s.detectPerformanceIssues(nc)
		s.detectCapacityIssues(nc)
		s.detectSecurityAnomalies(nc)
		return s.findingsToEvents(nc)
	}

	return events
}

// analyzeWithLLM sends pre-filtered findings to the LLM for intelligent analysis.
func (s *Service) analyzeWithLLM(ctx context.Context, nc *nodeContext) []model.SecurityEvent {
	if s.llmRegistry == nil {
		return s.findingsToEvents(nc)
	}

	modelName := s.llmRegistry.DefaultModel()
	provider, err := s.llmRegistry.GetForModel(modelName)
	if err != nil {
		slog.Warn("intelligence: no LLM provider, using rule-based fallback", slog.Any("error", err))
		return s.findingsToEvents(nc)
	}

	contextData := s.buildHybridLLMContext(nc)

	req := llm.CompletionRequest{
		Model: modelName,
		Messages: []llm.Message{
			{Role: "system", Content: hybridSystemPrompt},
			{Role: "user", Content: contextData},
		},
		MaxTokens:   2000,
		Temperature: 0.3,
	}

	resp, err := provider.Complete(ctx, req)
	if err != nil {
		slog.Warn("intelligence: LLM analysis failed, using rule-based fallback", slog.Any("error", err))
		return s.findingsToEvents(nc)
	}

	events, err := s.parseLLMResponse(resp.Content, modelName)
	if err != nil {
		slog.Warn("intelligence: failed to parse LLM response, using rule-based fallback", slog.Any("error", err))
		return s.findingsToEvents(nc)
	}

	return events
}

// buildFullLLMContext sends comprehensive metrics for full LLM analysis.
func (s *Service) buildFullLLMContext(nc *nodeContext) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## Node: %s (Typ: %s)\n\n", nc.Node.Name, nc.Node.Type))

	// Metrics summary over 24h
	if len(nc.Metrics) > 0 {
		sb.WriteString("### Metriken-Verlauf (letzte 24h):\n")

		// Compute stats
		var cpuVals, ramVals, diskVals []float64
		var netInVals, netOutVals []float64
		for _, r := range nc.Metrics {
			cpuVals = append(cpuVals, r.CPUUsage)
			if r.MemTotal > 0 {
				ramVals = append(ramVals, float64(r.MemUsed)/float64(r.MemTotal)*100)
			}
			if r.DiskTotal > 0 {
				diskVals = append(diskVals, float64(r.DiskUsed)/float64(r.DiskTotal)*100)
			}
			netInVals = append(netInVals, float64(r.NetIn))
			netOutVals = append(netOutVals, float64(r.NetOut))
		}

		writeStats(&sb, "CPU", cpuVals, "%")
		writeStats(&sb, "RAM", ramVals, "%")
		writeStats(&sb, "Disk", diskVals, "%")

		// Last record details
		last := nc.Metrics[len(nc.Metrics)-1]
		sb.WriteString(fmt.Sprintf("\n### Aktueller Stand:\n"))
		sb.WriteString(fmt.Sprintf("- CPU: %.1f%%\n", last.CPUUsage))
		if last.MemTotal > 0 {
			sb.WriteString(fmt.Sprintf("- RAM: %.1f%% (%s / %s)\n",
				float64(last.MemUsed)/float64(last.MemTotal)*100,
				formatBytes(last.MemUsed), formatBytes(last.MemTotal)))
		}
		if last.DiskTotal > 0 {
			sb.WriteString(fmt.Sprintf("- Disk: %.1f%% (%s / %s)\n",
				float64(last.DiskUsed)/float64(last.DiskTotal)*100,
				formatBytes(last.DiskUsed), formatBytes(last.DiskTotal)))
		}
		sb.WriteString(fmt.Sprintf("- Netzwerk In: %s, Out: %s\n", formatBytes(last.NetIn), formatBytes(last.NetOut)))

		// Trend: last 5 vs first 5
		if len(nc.Metrics) > 10 {
			first5CPU := avg(cpuVals[:5])
			last5CPU := avg(cpuVals[len(cpuVals)-5:])
			sb.WriteString(fmt.Sprintf("\n### Trends:\n"))
			sb.WriteString(fmt.Sprintf("- CPU-Trend: %.1f%% -> %.1f%% (%+.1f%%)\n", first5CPU, last5CPU, last5CPU-first5CPU))
			if len(ramVals) > 10 {
				first5RAM := avg(ramVals[:5])
				last5RAM := avg(ramVals[len(ramVals)-5:])
				sb.WriteString(fmt.Sprintf("- RAM-Trend: %.1f%% -> %.1f%% (%+.1f%%)\n", first5RAM, last5RAM, last5RAM-first5RAM))
			}
		}
	}

	// VMs
	if len(nc.VMs) > 0 {
		running := 0
		stopped := 0
		for _, vm := range nc.VMs {
			if vm.Status == "running" {
				running++
			} else {
				stopped++
			}
		}
		sb.WriteString(fmt.Sprintf("\n### VMs: %d laufend, %d gestoppt\n", running, stopped))
		for _, vm := range nc.VMs {
			name := vm.Name
			if name == "" {
				name = fmt.Sprintf("VM %d", vm.VMID)
			}
			sb.WriteString(fmt.Sprintf("- %s (ID: %d, Status: %s, CPU: %.1f%%, RAM: %s/%s)\n",
				name, vm.VMID, vm.Status,
				vm.CPUUsage*100,
				formatBytes(int64(vm.MemUsed)), formatBytes(int64(vm.MemTotal))))
		}
	}

	// Existing anomalies
	if len(nc.Anomalies) > 0 {
		unresolved := 0
		for _, a := range nc.Anomalies {
			if !a.IsResolved {
				unresolved++
			}
		}
		if unresolved > 0 {
			sb.WriteString(fmt.Sprintf("\n### Offene Anomalien (%d):\n", unresolved))
			for _, a := range nc.Anomalies {
				if !a.IsResolved {
					sb.WriteString(fmt.Sprintf("- %s: %.1f%% (Z-Score: %.1f, Severity: %s)\n", a.Metric, a.Value, a.ZScore, a.Severity))
				}
			}
		}
	}

	sb.WriteString(fmt.Sprintf("\nDatenpunkte: %d (Zeitraum: 24h)\n", len(nc.Metrics)))

	return sb.String()
}

// buildHybridLLMContext builds context with pre-filter findings for hybrid mode.
func (s *Service) buildHybridLLMContext(nc *nodeContext) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## Node: %s\n\n", nc.Node.Name))

	// Current metrics
	if len(nc.Metrics) > 0 {
		last := nc.Metrics[len(nc.Metrics)-1]
		sb.WriteString("### Aktuelle Metriken:\n")
		sb.WriteString(fmt.Sprintf("- CPU: %.1f%%\n", last.CPUUsage))
		if last.MemTotal > 0 {
			sb.WriteString(fmt.Sprintf("- RAM: %.1f%% (%s / %s)\n",
				float64(last.MemUsed)/float64(last.MemTotal)*100,
				formatBytes(last.MemUsed), formatBytes(last.MemTotal)))
		}
		if last.DiskTotal > 0 {
			sb.WriteString(fmt.Sprintf("- Disk: %.1f%% (%s / %s)\n",
				float64(last.DiskUsed)/float64(last.DiskTotal)*100,
				formatBytes(last.DiskUsed), formatBytes(last.DiskTotal)))
		}
		sb.WriteString(fmt.Sprintf("- Netzwerk In: %s, Out: %s\n\n", formatBytes(last.NetIn), formatBytes(last.NetOut)))
	}

	// Running VMs
	if len(nc.RunningVMs) > 0 {
		sb.WriteString(fmt.Sprintf("### Laufende VMs (%d):\n", len(nc.RunningVMs)))
		for _, vm := range nc.RunningVMs {
			sb.WriteString(fmt.Sprintf("- %s\n", vm))
		}
		sb.WriteString("\n")
	}

	// Findings
	sb.WriteString("### Erkannte Auffaelligkeiten (Rule-Based):\n")
	for i, f := range nc.Findings {
		sb.WriteString(fmt.Sprintf("%d. [%s/%s] %s\n   %s\n", i+1, f.Category, f.Severity, f.Title, f.Description))
		if len(f.Metrics) > 0 {
			sb.WriteString("   Metriken: ")
			first := true
			for k, v := range f.Metrics {
				if !first {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("%s=%.2f", k, v))
				first = false
			}
			sb.WriteString("\n")
		}
	}

	// Existing anomalies
	if len(nc.Anomalies) > 0 {
		sb.WriteString(fmt.Sprintf("\n### Bestehende Anomalien (%d):\n", len(nc.Anomalies)))
		for _, a := range nc.Anomalies {
			if !a.IsResolved {
				sb.WriteString(fmt.Sprintf("- %s: %.1f%% (Z-Score: %.1f, Severity: %s)\n", a.Metric, a.Value, a.ZScore, a.Severity))
			}
		}
	}

	return sb.String()
}

func (s *Service) parseLLMResponse(content string, modelName string) ([]model.SecurityEvent, error) {
	jsonStr := content
	if idx := strings.Index(content, "{"); idx >= 0 {
		jsonStr = content[idx:]
	}
	if idx := strings.LastIndex(jsonStr, "}"); idx >= 0 {
		jsonStr = jsonStr[:idx+1]
	}

	var result llmAnalysisResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("parse LLM JSON: %w", err)
	}

	var events []model.SecurityEvent
	for _, e := range result.Events {
		if !isValidCategory(e.Category) || !isValidSeverity(e.Severity) {
			continue
		}
		events = append(events, model.SecurityEvent{
			Category:       e.Category,
			Severity:       e.Severity,
			Title:          e.Title,
			Description:    e.Description,
			Impact:         e.Impact,
			Recommendation: e.Recommendation,
			AnalysisModel:  modelName,
		})
	}

	return events, nil
}

// findingsToEvents converts rule-based findings to SecurityEvents (fallback without LLM).
func (s *Service) findingsToEvents(nc *nodeContext) []model.SecurityEvent {
	var events []model.SecurityEvent
	for _, f := range nc.Findings {
		metricsJSON, _ := json.Marshal(f.Metrics)
		events = append(events, model.SecurityEvent{
			Category:       f.Category,
			Severity:       f.Severity,
			Title:          f.Title,
			Description:    f.Description,
			Impact:         generateDefaultImpact(f),
			Recommendation: generateDefaultRecommendation(f),
			Metrics:        metricsJSON,
			AnalysisModel:  "rule-based",
		})
	}
	return events
}

func generateDefaultImpact(f finding) string {
	switch f.Category {
	case "performance":
		return "Laufende VMs koennen Performance-Einbussen erfahren. Dienste reagieren moeglicherweise langsamer."
	case "capacity":
		return "Bei weiterem Wachstum koennen Schreibvorgaenge fehlschlagen und VMs nicht mehr gestartet werden."
	case "security":
		return "Moeglicherweise kompromittiertes System. Weitere Untersuchung dringend empfohlen."
	case "availability":
		return "Dienste auf diesem Node sind nicht erreichbar."
	default:
		return "Abweichung vom Normalbetrieb erkannt."
	}
}

func generateDefaultRecommendation(f finding) string {
	switch f.Category {
	case "performance":
		return "Pruefen Sie die ressourcenintensivsten Prozesse (top/htop). Erwaegen Sie VM-Migration oder Ressourcen-Erweiterung."
	case "capacity":
		return "Alte Snapshots und Backups bereinigen. Log-Rotation pruefen. Disk-Erweiterung planen."
	case "security":
		return "Pruefen Sie laufende Prozesse, offene Verbindungen (ss -tnlp) und Netzwerk-Traffic. Bei Verdacht: Node isolieren."
	case "availability":
		return "Netzwerkverbindung und physischen Server pruefen. Proxmox-Dienste neustarten."
	default:
		return "Metrik weiter beobachten und bei Bedarf eingreifen."
	}
}

func isValidCategory(c string) bool {
	switch c {
	case "performance", "security", "capacity", "availability", "config":
		return true
	}
	return false
}

func isValidSeverity(s string) bool {
	switch s {
	case "info", "warning", "critical", "emergency":
		return true
	}
	return false
}

func formatTimeAgo(t *time.Time) string {
	if t == nil {
		return "unbekannt"
	}
	diff := time.Since(*t)
	if diff < time.Minute {
		return "gerade eben"
	}
	if diff < time.Hour {
		return fmt.Sprintf("vor %d Min.", int(diff.Minutes()))
	}
	if diff < 24*time.Hour {
		return fmt.Sprintf("vor %d Std.", int(diff.Hours()))
	}
	return fmt.Sprintf("vor %d Tagen", int(diff.Hours()/24))
}

func formatBytes(b int64) string {
	bf := float64(b)
	units := []string{"B", "KB", "MB", "GB", "TB"}
	for _, u := range units {
		if math.Abs(bf) < 1024 {
			return fmt.Sprintf("%.1f %s", bf, u)
		}
		bf /= 1024
	}
	return fmt.Sprintf("%.1f PB", bf)
}

func avg(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func writeStats(sb *strings.Builder, name string, vals []float64, unit string) {
	if len(vals) == 0 {
		return
	}
	min, max, mean := vals[0], vals[0], 0.0
	for _, v := range vals {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
		mean += v
	}
	mean /= float64(len(vals))
	sb.WriteString(fmt.Sprintf("- %s: Avg=%.1f%s, Min=%.1f%s, Max=%.1f%s\n", name, mean, unit, min, unit, max, unit))
}

const hybridSystemPrompt = `Du bist ein erfahrener Infrastruktur-Sicherheitsanalyst fuer Proxmox-Rechenzentren.
Analysiere die folgenden Metriken und rule-based Befunde und erstelle eine priorisierte Bewertung.

Erkenne und bewerte:
1. SICHERHEITSANOMALIEN: Cryptomining, DDoS, unautorisierter Zugriff, Datenexfiltration
2. PERFORMANCE-PROBLEME: Resource Contention, Memory Leaks, I/O-Bottlenecks, Noisy Neighbors
3. KAPAZITAETSRISIKEN: Wachstumstrends, bevorstehende Speicherknappheit
4. VERFUEGBARKEITSRISIKEN: Single Points of Failure, ueberlastete Nodes
5. KONFIGURATIONSPROBLEME: Suboptimale Ressourcenverteilung

Antworte AUSSCHLIESSLICH als valides JSON:
{
  "events": [{
    "category": "security|performance|capacity|availability|config",
    "severity": "info|warning|critical|emergency",
    "title": "Kurzer Titel auf Deutsch",
    "description": "Detaillierte Analyse auf Deutsch",
    "impact": "Konkrete Auswirkungen",
    "recommendation": "Spezifische Handlungsempfehlung"
  }]
}

REGELN:
- NUR relevante Befunde melden. Keine falschen Alarme.
- Alle Texte auf Deutsch.
- Maximal 5 Events pro Analyse.
- Bei normaler Auslastung: leeres events Array.`

const fullLLMSystemPrompt = `Du bist ein erfahrener Infrastruktur-Sicherheitsanalyst fuer Proxmox-Rechenzentren.
Du erhaeltst umfassende Metriken und VM-Informationen eines Nodes. Analysiere ALLES selbststaendig.

Suche nach:
1. SICHERHEITSANOMALIEN:
   - Cryptomining: Hohe CPU bei niedrigem RAM-Verbrauch
   - DDoS: Ungewoehnliche Netzwerk-Spikes (>3x Durchschnitt)
   - Datenexfiltration: Hoher ausgehender Traffic
   - Unerwartete Prozesse: CPU-Spikes ohne erklaerbare Ursache

2. PERFORMANCE-PROBLEME:
   - Sustained high CPU/RAM (>85% ueber 15+ Minuten)
   - Memory Leaks: Stetig steigender RAM ohne Reset
   - I/O-Bottlenecks: Korrelation zwischen hoher Disk-Nutzung und Performance
   - Noisy Neighbors: Eine VM verbraucht ueberproportional viele Ressourcen

3. KAPAZITAETSRISIKEN:
   - Disk >85% oder steigender Trend (>0.5%/Tag)
   - RAM-Engpaesse
   - Wachstumsprognosen basierend auf Trends

4. VERFUEGBARKEITSRISIKEN:
   - Ueberladene Nodes (zu viele VMs)
   - Single Points of Failure
   - VMs die gestoppt sind aber laufen sollten

5. KONFIGURATIONSPROBLEME:
   - Ueberprovisioned VMs (hohe Zuweisung, niedrige Nutzung)
   - Underprovisioned VMs (konstant am Limit)
   - Ungleichmaessige Lastverteilung

Antworte AUSSCHLIESSLICH als valides JSON:
{
  "events": [{
    "category": "security|performance|capacity|availability|config",
    "severity": "info|warning|critical|emergency",
    "title": "Kurzer, praegnanter Titel auf Deutsch",
    "description": "Detaillierte Analyse: Was passiert, warum ist es auffaellig, Kontext zu den VMs",
    "impact": "Konkrete Auswirkungen auf den Betrieb und betroffene VMs",
    "recommendation": "Spezifische, umsetzbare Handlungsempfehlung mit konkreten Befehlen/Schritten"
  }]
}

REGELN:
- NUR melden wenn wirklich relevant. Bei normaler Auslastung: leeres events Array.
- Beruecksichtige den VM-Typ (Name-basiert: db/mysql = Datenbank, nginx/apache = Webserver, docker/k8s = Container-Host)
- Gib kontextbezogene Empfehlungen basierend auf VM-Typ
- Alle Texte auf Deutsch
- Maximal 7 Events pro Analyse
- Severity "emergency" nur bei echten Notfaellen`
