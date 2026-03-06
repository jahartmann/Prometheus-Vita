package prediction

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
	diskThreshold   = 90.0
	memoryThreshold = 90.0
)

// NodeServiceInterface allows fetching VMs for context enrichment.
type NodeServiceInterface interface {
	GetVMs(ctx context.Context, id uuid.UUID) ([]proxmox.VMInfo, error)
}

type Service struct {
	predictionRepo repository.PredictionRepository
	metricsRepo    repository.MetricsRepository
	nodeRepo       repository.NodeRepository
	nodeSvc        NodeServiceInterface
}

func NewService(
	predictionRepo repository.PredictionRepository,
	metricsRepo repository.MetricsRepository,
	nodeRepo repository.NodeRepository,
) *Service {
	return &Service{
		predictionRepo: predictionRepo,
		metricsRepo:    metricsRepo,
		nodeRepo:       nodeRepo,
	}
}

func (s *Service) SetNodeService(svc NodeServiceInterface) {
	s.nodeSvc = svc
}

func (s *Service) RunPredictions(ctx context.Context) error {
	nodes, err := s.nodeRepo.List(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	since := now.Add(-7 * 24 * time.Hour)

	for _, node := range nodes {
		if !node.IsOnline {
			continue
		}

		records, err := s.metricsRepo.GetByNode(ctx, node.ID, since, now)
		if err != nil {
			slog.Warn("prediction: failed to get metrics", slog.String("node", node.Name), slog.Any("error", err))
			continue
		}

		if len(records) < 20 {
			continue
		}

		s.predictMetric(ctx, node.ID, "disk_usage", diskThreshold, records, func(r model.MetricsRecord) float64 {
			if r.DiskTotal == 0 {
				return 0
			}
			return float64(r.DiskUsed) / float64(r.DiskTotal) * 100
		})

		s.predictMetric(ctx, node.ID, "memory_usage", memoryThreshold, records, func(r model.MetricsRecord) float64 {
			if r.MemTotal == 0 {
				return 0
			}
			return float64(r.MemUsed) / float64(r.MemTotal) * 100
		})
	}

	return nil
}

func (s *Service) predictMetric(ctx context.Context, nodeID uuid.UUID, metric string, threshold float64, records []model.MetricsRecord, extract func(model.MetricsRecord) float64) {
	n := len(records)
	xs := make([]float64, n)
	ys := make([]float64, n)
	baseTime := records[0].RecordedAt.Unix()

	for i, r := range records {
		xs[i] = float64(r.RecordedAt.Unix()-baseTime) / 3600.0 // hours
		ys[i] = extract(r)
	}

	slope, intercept, rSquared := linearRegression(xs, ys)

	currentValue := ys[n-1]
	currentX := xs[n-1]
	// Predict 30 days ahead
	predictedValue := slope*(currentX+30*24) + intercept

	var daysUntilThreshold *float64
	if slope > 0 && currentValue < threshold {
		hoursUntil := (threshold - intercept) / slope
		hoursFromNow := hoursUntil - currentX
		if hoursFromNow > 0 {
			days := hoursFromNow / 24.0
			daysUntilThreshold = &days
		}
	}

	severity := "info"
	if daysUntilThreshold != nil {
		if *daysUntilThreshold <= 7 {
			severity = "critical"
		} else if *daysUntilThreshold <= 30 {
			severity = "warning"
		}
	}

	pred := &model.MaintenancePrediction{
		NodeID:             nodeID,
		Metric:             metric,
		CurrentValue:       currentValue,
		PredictedValue:     predictedValue,
		Threshold:          threshold,
		DaysUntilThreshold: daysUntilThreshold,
		Slope:              slope,
		Intercept:          intercept,
		RSquared:           rSquared,
		Severity:           severity,
	}

	if err := s.predictionRepo.Upsert(ctx, pred); err != nil {
		slog.Warn("prediction: failed to upsert",
			slog.String("node", nodeID.String()),
			slog.String("metric", metric),
			slog.Any("error", err))
	}
}

func (s *Service) ListCritical(ctx context.Context) ([]model.MaintenancePrediction, error) {
	preds, err := s.predictionRepo.ListCritical(ctx)
	if err != nil {
		return nil, err
	}
	s.enrichPredictions(ctx, preds)
	return preds, nil
}

func (s *Service) ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.MaintenancePrediction, error) {
	preds, err := s.predictionRepo.ListByNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	s.enrichPredictions(ctx, preds)
	return preds, nil
}

// enrichPredictions adds node names, descriptions, recommendations, and VM context.
func (s *Service) enrichPredictions(ctx context.Context, preds []model.MaintenancePrediction) {
	nodeCache := make(map[uuid.UUID]*model.Node)
	vmCache := make(map[uuid.UUID][]proxmox.VMInfo)

	for i := range preds {
		p := &preds[i]

		// Resolve node name
		node, ok := nodeCache[p.NodeID]
		if !ok {
			n, err := s.nodeRepo.GetByID(ctx, p.NodeID)
			if err == nil {
				node = n
			}
			nodeCache[p.NodeID] = node
		}
		if node != nil {
			p.NodeName = node.Name
		} else {
			p.NodeName = p.NodeID.String()[:8]
		}

		// Resolve VMs
		vms, ok := vmCache[p.NodeID]
		if !ok && s.nodeSvc != nil {
			fetchedVMs, err := s.nodeSvc.GetVMs(ctx, p.NodeID)
			if err == nil {
				vms = fetchedVMs
			}
			vmCache[p.NodeID] = vms
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
		p.AffectedVMs = runningVMs
		p.VMCount = len(runningVMs)

		// Trend direction
		if p.Slope > 0.01 {
			p.TrendDirection = "rising"
		} else if p.Slope < -0.01 {
			p.TrendDirection = "falling"
		} else {
			p.TrendDirection = "stable"
		}

		// Generate description and recommendation
		p.Description = s.generateDescription(p)
		p.Recommendation = s.generateRecommendation(p, runningVMs)
	}
}

func (s *Service) generateDescription(p *model.MaintenancePrediction) string {
	metricName := metricDisplayName(p.Metric)
	days := float64(0)
	if p.DaysUntilThreshold != nil {
		days = *p.DaysUntilThreshold
	}

	ratePerDay := p.Slope * 24 // percent per day

	if p.TrendDirection == "rising" {
		if days > 0 {
			return fmt.Sprintf(
				"%s auf %s steigt mit %.2f%%/Tag. Bei diesem Trend wird der Schwellwert von %.0f%% in %.0f Tagen erreicht. "+
					"Aktuell: %.1f%%, Prognose (30d): %.1f%%.",
				metricName, p.NodeName, ratePerDay, p.Threshold, days,
				p.CurrentValue, p.PredictedValue,
			)
		}
		return fmt.Sprintf(
			"%s auf %s steigt mit %.2f%%/Tag. Aktuell bei %.1f%%. "+
				"Der Schwellwert von %.0f%% wird voraussichtlich nicht innerhalb von 30 Tagen erreicht.",
			metricName, p.NodeName, ratePerDay, p.CurrentValue, p.Threshold,
		)
	}
	if p.TrendDirection == "falling" {
		return fmt.Sprintf(
			"%s auf %s faellt mit %.2f%%/Tag. Aktuell bei %.1f%%. Kein Engpass absehbar.",
			metricName, p.NodeName, math.Abs(ratePerDay), p.CurrentValue,
		)
	}
	return fmt.Sprintf(
		"%s auf %s ist stabil bei %.1f%%. Keine wesentliche Aenderung prognostiziert.",
		metricName, p.NodeName, p.CurrentValue,
	)
}

func (s *Service) generateRecommendation(p *model.MaintenancePrediction, vmNames []string) string {
	hasDB := containsAny(vmNames, []string{"db", "mysql", "postgres", "mongo", "redis", "mariadb", "sql"})
	hasDocker := containsAny(vmNames, []string{"docker", "k8s", "kubernetes", "container", "portainer"})
	hasBackup := containsAny(vmNames, []string{"backup", "borg", "restic", "pbs"})

	days := float64(999)
	if p.DaysUntilThreshold != nil {
		days = *p.DaysUntilThreshold
	}

	switch p.Metric {
	case "disk_usage":
		if days <= 7 {
			if hasBackup {
				return "Dringend: Backup-Storage fast voll. Alte Backup-Generationen bereinigen, Retention-Policy pruefen. Disk-Erweiterung sofort einplanen."
			}
			if hasDocker {
				return "Dringend: Docker-Images und nicht genutzte Volumes bereinigen (`docker system prune -a`). Disk-Erweiterung einplanen bevor Container nicht mehr starten koennen."
			}
			return fmt.Sprintf("Dringend: Speicher wird in %.0f Tagen voll. Sofort bereinigen: alte Snapshots loeschen, Logs rotieren, grosse Dateien identifizieren. Disk-Erweiterung planen.", days)
		}
		if days <= 30 {
			return fmt.Sprintf("Speicherplatz wird in ca. %.0f Tagen knapp. Praeventiv: Snapshot-Retention pruefen, unnoetige ISOs/Templates entfernen, Disk-Erweiterung einplanen.", days)
		}
		return "Speicherverbrauch steigt langsam. Regelmaessig pruefen und bei Bedarf erweitern."

	case "memory_usage":
		if days <= 7 {
			if hasDB {
				return "Kritisch: RAM-Engpass in Kuerze. Datenbank-Caches (innodb_buffer_pool_size, shared_buffers) verkleinern oder RAM erweitern. OOM-Killer koennte Datenbank-Prozesse beenden!"
			}
			return fmt.Sprintf("Kritisch: RAM wird in %.0f Tagen erschoepft. Memory-Leaks pruefen, ueberdimensionierte VMs identifizieren (siehe Empfehlungen), oder RAM-Upgrade durchfuehren.", days)
		}
		if days <= 30 {
			if hasDB {
				return "RAM-Nutzung steigt. Bei Datenbank-VMs ist das oft durch wachsende Datenbankgroesse bedingt. Pruefen Sie ob DB-Caches optimal konfiguriert sind und planen Sie ggf. RAM-Erweiterung."
			}
			return fmt.Sprintf("RAM-Nutzung steigt stetig. In ca. %.0f Tagen wird der Schwellwert erreicht. VMs auf Memory-Leaks pruefen, Ballooning aktivieren, oder RAM erweitern.", days)
		}
		return "RAM-Verbrauch steigt leicht. Trend beobachten und bei Bedarf handeln."
	}

	return "Entwicklung weiter beobachten und praeventiv Kapazitaet planen."
}

func containsAny(names []string, keywords []string) bool {
	for _, name := range names {
		lower := toLower(name)
		for _, kw := range keywords {
			if findSubstring(lower, kw) {
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

func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func metricDisplayName(metric string) string {
	switch metric {
	case "disk_usage":
		return "Disk-Auslastung"
	case "memory_usage":
		return "RAM-Auslastung"
	case "cpu_usage":
		return "CPU-Auslastung"
	default:
		return metric
	}
}

func linearRegression(xs, ys []float64) (slope, intercept, rSquared float64) {
	n := float64(len(xs))
	if n == 0 {
		return 0, 0, 0
	}

	var sumX, sumY, sumXY, sumX2, sumY2 float64
	for i := range xs {
		sumX += xs[i]
		sumY += ys[i]
		sumXY += xs[i] * ys[i]
		sumX2 += xs[i] * xs[i]
		sumY2 += ys[i] * ys[i]
	}
	_ = sumY2

	denom := n*sumX2 - sumX*sumX
	if denom == 0 {
		return 0, sumY / n, 0
	}

	slope = (n*sumXY - sumX*sumY) / denom
	intercept = (sumY - slope*sumX) / n

	// R-squared
	meanY := sumY / n
	var ssTot, ssRes float64
	for i := range xs {
		predicted := slope*xs[i] + intercept
		ssRes += (ys[i] - predicted) * (ys[i] - predicted)
		ssTot += (ys[i] - meanY) * (ys[i] - meanY)
	}

	if ssTot == 0 {
		rSquared = 1
	} else {
		rSquared = math.Max(0, 1-ssRes/ssTot)
	}

	return slope, intercept, rSquared
}
