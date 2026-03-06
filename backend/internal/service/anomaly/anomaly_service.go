package anomaly

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/proxmox"
	"github.com/antigravity/prometheus/internal/repository"
)

const (
	// Only flag truly significant deviations — avoid noise from normal fluctuations
	zScoreWarning  = 3.5
	zScoreCritical = 5.0
	minDataPoints  = 30 // Need at least 30 data points (30+ minutes of data)

	// Minimum absolute value thresholds — don't flag low-usage anomalies
	minCPUForAnomaly  = 70.0  // CPU must be >70% to be flagged
	minRAMForAnomaly  = 75.0  // RAM must be >75% to be flagged
	minDiskForAnomaly = 80.0  // Disk must be >80% to be flagged
)

// NodeServiceInterface allows fetching VMs for context enrichment.
type NodeServiceInterface interface {
	GetVMs(ctx context.Context, id uuid.UUID) ([]proxmox.VMInfo, error)
}

type Service struct {
	anomalyRepo repository.AnomalyRepository
	metricsRepo repository.MetricsRepository
	nodeRepo    repository.NodeRepository
	nodeSvc     NodeServiceInterface
}

func NewService(
	anomalyRepo repository.AnomalyRepository,
	metricsRepo repository.MetricsRepository,
	nodeRepo repository.NodeRepository,
) *Service {
	return &Service{
		anomalyRepo: anomalyRepo,
		metricsRepo: metricsRepo,
		nodeRepo:    nodeRepo,
	}
}

func (s *Service) SetNodeService(svc NodeServiceInterface) {
	s.nodeSvc = svc
}

func (s *Service) DetectAnomalies(ctx context.Context) error {
	nodes, err := s.nodeRepo.List(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	since := now.Add(-24 * time.Hour)

	for _, node := range nodes {
		if !node.IsOnline {
			continue
		}

		records, err := s.metricsRepo.GetByNode(ctx, node.ID, since, now)
		if err != nil {
			slog.Warn("anomaly detection: failed to get metrics", slog.String("node", node.Name), slog.Any("error", err))
			continue
		}

		if len(records) < minDataPoints {
			continue
		}

		s.checkMetric(ctx, node.ID, "cpu_usage", records, func(r model.MetricsRecord) float64 { return r.CPUUsage })

		s.checkMetric(ctx, node.ID, "memory_usage", records, func(r model.MetricsRecord) float64 {
			if r.MemTotal == 0 {
				return 0
			}
			return float64(r.MemUsed) / float64(r.MemTotal) * 100
		})

		s.checkMetric(ctx, node.ID, "disk_usage", records, func(r model.MetricsRecord) float64 {
			if r.DiskTotal == 0 {
				return 0
			}
			return float64(r.DiskUsed) / float64(r.DiskTotal) * 100
		})
	}

	return nil
}

func (s *Service) checkMetric(ctx context.Context, nodeID uuid.UUID, metric string, records []model.MetricsRecord, extract func(model.MetricsRecord) float64) {
	values := make([]float64, len(records))
	for i, r := range records {
		values[i] = extract(r)
	}

	mean, stddev := meanStdDev(values)
	if stddev == 0 {
		return
	}

	// Check the last 5 data points (sustained anomaly, not a single spike)
	recentCount := 5
	if len(values) < recentCount {
		recentCount = len(values)
	}
	recentValues := values[len(values)-recentCount:]
	recentMean := 0.0
	for _, v := range recentValues {
		recentMean += v
	}
	recentMean /= float64(len(recentValues))

	zScore := (recentMean - mean) / stddev

	if math.Abs(zScore) < zScoreWarning {
		return
	}

	// Enforce minimum absolute thresholds — don't flag low-usage "anomalies"
	minThreshold := 0.0
	switch metric {
	case "cpu_usage":
		minThreshold = minCPUForAnomaly
	case "memory_usage":
		minThreshold = minRAMForAnomaly
	case "disk_usage":
		minThreshold = minDiskForAnomaly
	}

	// Only flag if the value is actually in a concerning range
	if recentMean < minThreshold {
		return
	}

	severity := "warning"
	if math.Abs(zScore) >= zScoreCritical {
		severity = "critical"
	}

	record := &model.AnomalyRecord{
		NodeID:   nodeID,
		Metric:   metric,
		Value:    recentMean,
		ZScore:   zScore,
		Mean:     mean,
		StdDev:   stddev,
		Severity: severity,
	}

	if err := s.anomalyRepo.Create(ctx, record); err != nil {
		slog.Warn("anomaly detection: failed to create record",
			slog.String("node", nodeID.String()),
			slog.String("metric", metric),
			slog.Any("error", err))
	} else {
		slog.Info("anomaly detected",
			slog.String("node", nodeID.String()),
			slog.String("metric", metric),
			slog.Float64("z_score", zScore),
			slog.String("severity", severity))
	}
}

func (s *Service) ListUnresolved(ctx context.Context) ([]model.AnomalyRecord, error) {
	records, err := s.anomalyRepo.ListUnresolved(ctx)
	if err != nil {
		return nil, err
	}
	s.enrichAnomalies(ctx, records)
	return records, nil
}

func (s *Service) ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.AnomalyRecord, error) {
	records, err := s.anomalyRepo.ListByNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	s.enrichAnomalies(ctx, records)
	return records, nil
}

func (s *Service) Resolve(ctx context.Context, id uuid.UUID) error {
	return s.anomalyRepo.Resolve(ctx, id)
}

// enrichAnomalies adds node names, descriptions, impact, recommendations, and affected VMs.
func (s *Service) enrichAnomalies(ctx context.Context, records []model.AnomalyRecord) {
	// Cache node info
	nodeCache := make(map[uuid.UUID]*model.Node)
	vmCache := make(map[uuid.UUID][]proxmox.VMInfo)

	for i := range records {
		r := &records[i]

		// Resolve node name
		node, ok := nodeCache[r.NodeID]
		if !ok {
			n, err := s.nodeRepo.GetByID(ctx, r.NodeID)
			if err == nil {
				node = n
			}
			nodeCache[r.NodeID] = node
		}
		if node != nil {
			r.NodeName = node.Name
		} else {
			r.NodeName = r.NodeID.String()[:8]
		}

		// Resolve VMs running on this node
		vms, ok := vmCache[r.NodeID]
		if !ok && s.nodeSvc != nil {
			fetchedVMs, err := s.nodeSvc.GetVMs(ctx, r.NodeID)
			if err == nil {
				vms = fetchedVMs
			}
			vmCache[r.NodeID] = vms
		}

		runningVMs := []string{}
		for _, vm := range vms {
			if vm.Status == "running" {
				name := vm.Name
				if name == "" {
					name = fmt.Sprintf("VM %d", vm.VMID)
				}
				runningVMs = append(runningVMs, name)
			}
		}
		r.AffectedVMs = runningVMs

		// Generate context-aware description, impact, and recommendation
		r.Description = s.generateDescription(r)
		r.Impact = s.generateImpact(r, len(runningVMs))
		r.Recommendation = s.generateRecommendation(r, runningVMs)
	}
}

func (s *Service) generateDescription(r *model.AnomalyRecord) string {
	metricName := metricDisplayName(r.Metric)
	deviation := math.Abs(r.Value - r.Mean)

	if r.Value > r.Mean {
		return fmt.Sprintf(
			"%s auf %s liegt bei %.1f%% — das sind %.1f Prozentpunkte ueber dem 24h-Durchschnitt von %.1f%%. "+
				"Diese Abweichung (Z-Score: %.1f) ist statistisch %s.",
			metricName, r.NodeName, r.Value, deviation, r.Mean,
			math.Abs(r.ZScore), severityText(r.Severity),
		)
	}
	return fmt.Sprintf(
		"%s auf %s ist auf %.1f%% gefallen — %.1f Prozentpunkte unter dem 24h-Durchschnitt von %.1f%%. "+
			"Diese Abweichung (Z-Score: %.1f) ist ungewoehnlich.",
		metricName, r.NodeName, r.Value, deviation, r.Mean,
		math.Abs(r.ZScore),
	)
}

func (s *Service) generateImpact(r *model.AnomalyRecord, vmCount int) string {
	switch r.Metric {
	case "cpu_usage":
		if r.Value > 90 {
			return fmt.Sprintf("Kritische CPU-Auslastung — %d laufende VMs koennen Leistungseinbussen erfahren. Prozesse werden moeglicherweise gedrosselt.", vmCount)
		}
		if r.Value > 80 {
			return fmt.Sprintf("Hohe CPU-Last — bei %d laufenden VMs koennte die Reaktionszeit steigen.", vmCount)
		}
		return "Erhoehte CPU-Nutzung, aber noch im handhabbaren Bereich."
	case "memory_usage":
		if r.Value > 90 {
			return fmt.Sprintf("Kritischer RAM-Engpass — %d VMs koennten von OOM-Killer betroffen werden. Swap-Nutzung wahrscheinlich.", vmCount)
		}
		if r.Value > 80 {
			return fmt.Sprintf("Hohe RAM-Auslastung bei %d aktiven VMs — Performance-Degradation moeglich.", vmCount)
		}
		return "RAM-Nutzung ueber dem Normalwert, aber noch ausreichend Puffer."
	case "disk_usage":
		if r.Value > 95 {
			return "Kritisch: Speicher fast voll — Schreibvorgaenge koennen fehlschlagen, VMs koennen nicht mehr starten."
		}
		if r.Value > 85 {
			return "Speicherplatz wird knapp — Backups und Snapshots koennten scheitern."
		}
		return "Disk-Nutzung ueber dem Durchschnitt, noch kein akuter Engpass."
	}
	return "Abweichung vom Normalwert erkannt."
}

func (s *Service) generateRecommendation(r *model.AnomalyRecord, vmNames []string) string {
	// Check for known VM types to give context-specific advice
	hasDB := containsAny(vmNames, []string{"db", "mysql", "postgres", "mongo", "redis", "mariadb", "sql"})
	hasWeb := containsAny(vmNames, []string{"web", "nginx", "apache", "proxy", "haproxy", "caddy"})
	hasDocker := containsAny(vmNames, []string{"docker", "k8s", "kubernetes", "container", "portainer"})

	switch r.Metric {
	case "cpu_usage":
		if hasDB && r.Value > 80 {
			return "Datenbank-VMs auf diesem Node koennten die Ursache sein. Pruefen Sie langsame Queries, fehlende Indizes oder laufende Backups. Erwaeegen Sie CPU-Ressourcen zu erhoehen oder Workloads zu verteilen."
		}
		if hasDocker && r.Value > 80 {
			return "Container-Host mit hoher CPU-Last. Pruefen Sie `docker stats` auf ressourcenhungrige Container. Erwaeegen Sie Resource-Limits oder horizontales Scaling."
		}
		if hasWeb && r.Value > 80 {
			return "Webserver unter hoher Last. Pruefen Sie Zugriffslogs auf Traffic-Spitzen. Erwaeegen Sie Load-Balancing oder Caching-Optimierung."
		}
		if r.Severity == "critical" {
			return "Sofortige Aktion: Identifizieren Sie den ressourcenintensivsten Prozess (`top`/`htop`). Erwaeegen Sie VM-Migration auf weniger ausgelastete Nodes."
		}
		return "Beobachten Sie den Trend. Falls anhaltend, pruefen Sie ob VMs umverteilt oder CPU-Ressourcen erweitert werden sollten."

	case "memory_usage":
		if hasDB {
			return "Datenbank-VMs benoetigen typischerweise viel RAM fuer Caches. Pruefen Sie ob die DB-Konfiguration (innodb_buffer_pool, shared_buffers) zum verfuegbaren RAM passt. RAM-Erweiterung bevorzugen statt VMs zu verschieben."
		}
		if hasDocker {
			return "Container-Host mit hohem RAM-Verbrauch. Pruefen Sie Container-Memory-Limits (`docker stats`). Nicht genutzte Container/Images bereinigen (`docker system prune`)."
		}
		if r.Severity == "critical" {
			return "Akuter RAM-Engpass: Pruefen Sie ob Memory-Leaks vorliegen. Gestoppte VMs mit reserviertem RAM identifizieren. Notfalls VM-Migration einleiten."
		}
		return "RAM-Verbrauch ueberwachen. Falls steigend, pruefen Sie ob einzelne VMs ueberdimensioniert sind (siehe Empfehlungen)."

	case "disk_usage":
		if r.Value > 90 {
			return "Dringend: Alte Snapshots und Backups bereinigen. Log-Rotation pruefen. Grosse Dateien identifizieren (`du -sh /*`). Disk-Erweiterung planen."
		}
		return "Speichertrend beobachten. Alte Snapshots bereinigen und Log-Rotation sicherstellen. Siehe Vorhersagen fuer Kapazitaetsplanung."
	}
	return "Metrik weiter beobachten und bei anhaltender Abweichung eingreifen."
}

func containsAny(names []string, keywords []string) bool {
	for _, name := range names {
		lower := toLower(name)
		for _, kw := range keywords {
			if contains(lower, kw) {
				return true
			}
		}
	}
	return false
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func metricDisplayName(metric string) string {
	switch metric {
	case "cpu_usage":
		return "CPU-Auslastung"
	case "memory_usage":
		return "RAM-Auslastung"
	case "disk_usage":
		return "Disk-Auslastung"
	default:
		return metric
	}
}

func severityText(severity string) string {
	switch severity {
	case "critical":
		return "stark auffaellig"
	case "warning":
		return "auffaellig"
	default:
		return "leicht auffaellig"
	}
}

func meanStdDev(values []float64) (float64, float64) {
	n := float64(len(values))
	if n == 0 {
		return 0, 0
	}

	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / n

	var varianceSum float64
	for _, v := range values {
		diff := v - mean
		varianceSum += diff * diff
	}
	stddev := math.Sqrt(varianceSum / n)

	return mean, stddev
}
