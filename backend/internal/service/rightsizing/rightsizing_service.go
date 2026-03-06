package rightsizing

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/proxmox"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
)

// VM context types
const (
	VMContextDatabase  = "Datenbank"
	VMContextWebserver = "Webserver"
	VMContextMail      = "Mailserver"
	VMContextDNS       = "DNS"
	VMContextBackup    = "Backup"
	VMContextMonitor   = "Monitoring"
	VMContextContainer = "Container-Host"
	VMContextGeneral   = "Allgemein"
)

// Context-aware thresholds per VM type
type thresholds struct {
	cpuOverprovision  float64
	cpuUnderprovision float64
	memOverprovision  float64
	memUnderprovision float64
}

var defaultThresholds = thresholds{
	cpuOverprovision:  0.10,
	cpuUnderprovision: 0.80,
	memOverprovision:  0.20,
	memUnderprovision: 0.85,
}

var contextThresholds = map[string]thresholds{
	VMContextDatabase: {
		cpuOverprovision:  0.05,
		cpuUnderprovision: 0.85,
		memOverprovision:  0.30, // DBs need more RAM headroom
		memUnderprovision: 0.90,
	},
	VMContextWebserver: {
		cpuOverprovision:  0.10,
		cpuUnderprovision: 0.60, // recommend upsize earlier for webservers
		memOverprovision:  0.20,
		memUnderprovision: 0.85,
	},
	VMContextContainer: {
		cpuOverprovision:  0.15, // only downsize if <15%
		cpuUnderprovision: 0.85,
		memOverprovision:  0.15, // more headroom for containers
		memUnderprovision: 0.90,
	},
	VMContextBackup: {
		cpuOverprovision:  0.05,
		cpuUnderprovision: 0.90,
		memOverprovision:  0.10,
		memUnderprovision: 0.95,
	},
	VMContextMonitor: {
		cpuOverprovision:  0.10,
		cpuUnderprovision: 0.75,
		memOverprovision:  0.25,
		memUnderprovision: 0.85,
	},
}

var vmContextPatterns = []struct {
	keywords []string
	context  string
}{
	{[]string{"db", "mysql", "postgres", "pgsql", "mongo", "redis", "mariadb", "elastic", "influx"}, VMContextDatabase},
	{[]string{"web", "nginx", "apache", "proxy", "haproxy", "caddy", "traefik"}, VMContextWebserver},
	{[]string{"mail", "smtp", "imap", "postfix", "dovecot"}, VMContextMail},
	{[]string{"dns", "pihole", "bind", "unbound", "adguard"}, VMContextDNS},
	{[]string{"backup", "borg", "restic", "pbs", "veeam"}, VMContextBackup},
	{[]string{"monitor", "grafana", "prometheus", "zabbix", "nagios", "loki"}, VMContextMonitor},
	{[]string{"docker", "k8s", "kubernetes", "container", "podman", "rancher"}, VMContextContainer},
}

type Service struct {
	recRepo       repository.RecommendationRepository
	nodeRepo      repository.NodeRepository
	clientFactory proxmox.ClientFactory
}

func NewService(
	recRepo repository.RecommendationRepository,
	nodeRepo repository.NodeRepository,
	clientFactory proxmox.ClientFactory,
) *Service {
	return &Service{
		recRepo:       recRepo,
		nodeRepo:      nodeRepo,
		clientFactory: clientFactory,
	}
}

func detectVMContext(vmName string) string {
	lower := strings.ToLower(vmName)
	for _, p := range vmContextPatterns {
		for _, kw := range p.keywords {
			if strings.Contains(lower, kw) {
				return p.context
			}
		}
	}
	return VMContextGeneral
}

func getThresholds(vmCtx string) thresholds {
	if t, ok := contextThresholds[vmCtx]; ok {
		return t
	}
	return defaultThresholds
}

func contextReasonForType(vmCtx, resourceType, recType string) string {
	switch vmCtx {
	case VMContextDatabase:
		if resourceType == "memory" && recType == "optimal" {
			return "Als Datenbank-VM ist eine hoehere RAM-Nutzung normal und erwartet."
		}
		if resourceType == "memory" && recType == "downsize" {
			return "Als Datenbank-VM wird RAM-Reduktion nur bei sehr niedriger Nutzung (<30%) empfohlen."
		}
		if resourceType == "cpu" {
			return "Datenbank-VMs sind typischerweise I/O-gebunden, CPU ist weniger kritisch."
		}
	case VMContextWebserver:
		if resourceType == "cpu" && recType == "upsize" {
			return "Webserver benoetigen ausreichend CPU fuer Request-Verarbeitung."
		}
		if resourceType == "memory" {
			return "Webserver benoetigen weniger RAM als Datenbanken."
		}
	case VMContextContainer:
		if recType == "downsize" {
			return "Container-Hosts benoetigen mehr Headroom fuer dynamische Workloads."
		}
		return "Container-Hosts haben variable Auslastung - mehr Puffer wird empfohlen."
	case VMContextBackup:
		if resourceType == "cpu" || resourceType == "memory" {
			return "Backup-VMs benoetigen primaer Disk-Ressourcen, CPU/RAM ist weniger kritisch."
		}
	case VMContextMonitor:
		if resourceType == "memory" {
			return "Monitoring-VMs benoetigen RAM fuer Metriken-Speicherung und Dashboards."
		}
	case VMContextMail:
		return "Mailserver benoetigen stabile Ressourcen fuer die Zustellung."
	case VMContextDNS:
		return "DNS-Server sind typischerweise ressourcenschonend."
	}
	return ""
}

func (s *Service) AnalyzeNode(ctx context.Context, nodeID uuid.UUID) ([]model.ResourceRecommendation, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}

	client, err := s.clientFactory.CreateClient(node)
	if err != nil {
		return nil, fmt.Errorf("create proxmox client: %w", err)
	}

	pveNodes, err := client.GetNodes(ctx)
	if err != nil || len(pveNodes) == 0 {
		return nil, fmt.Errorf("get pve nodes: %w", err)
	}

	vms, err := client.GetVMs(ctx, pveNodes[0])
	if err != nil {
		return nil, fmt.Errorf("get vms: %w", err)
	}

	// Clear old recommendations
	_ = s.recRepo.DeleteByNode(ctx, nodeID)

	var recommendations []model.ResourceRecommendation

	for _, vm := range vms {
		if vm.Status != "running" {
			continue
		}

		rrdData, err := client.GetVMRRDData(ctx, pveNodes[0], vm.VMID, vm.Type, "day")
		if err != nil {
			slog.Warn("failed to get RRD data for VM",
				slog.Int("vmid", vm.VMID),
				slog.Any("error", err),
			)
			continue
		}

		if len(rrdData) == 0 {
			continue
		}

		vmCtx := detectVMContext(vm.Name)
		t := getThresholds(vmCtx)

		// Analyze CPU
		var totalCPU, maxCPU float64
		for _, dp := range rrdData {
			totalCPU += dp.CPU
			if dp.CPU > maxCPU {
				maxCPU = dp.CPU
			}
		}
		avgCPU := totalCPU / float64(len(rrdData))

		if avgCPU < t.cpuOverprovision && vm.CPUs > 1 {
			recType := string(model.RecommendationDownsize)
			rec := model.ResourceRecommendation{
				NodeID:             nodeID,
				VMID:               vm.VMID,
				VMName:             vm.Name,
				VMType:             vm.Type,
				ResourceType:       "cpu",
				CurrentValue:       int64(vm.CPUs),
				RecommendedValue:   max(int64(float64(vm.CPUs)*0.5), 1),
				AvgUsage:           avgCPU * 100,
				MaxUsage:           maxCPU * 100,
				RecommendationType: model.RecommendationDownsize,
				Reason:             fmt.Sprintf("Durchschnittliche CPU-Auslastung %.1f%% - VM ist ueberprovisioniert", avgCPU*100),
				VMContext:          vmCtx,
				ContextReason:      contextReasonForType(vmCtx, "cpu", recType),
			}
			_ = s.recRepo.Create(ctx, &rec)
			recommendations = append(recommendations, rec)
		} else if avgCPU > t.cpuUnderprovision {
			recType := string(model.RecommendationUpsize)
			rec := model.ResourceRecommendation{
				NodeID:             nodeID,
				VMID:               vm.VMID,
				VMName:             vm.Name,
				VMType:             vm.Type,
				ResourceType:       "cpu",
				CurrentValue:       int64(vm.CPUs),
				RecommendedValue:   int64(float64(vm.CPUs) * 1.5),
				AvgUsage:           avgCPU * 100,
				MaxUsage:           maxCPU * 100,
				RecommendationType: model.RecommendationUpsize,
				Reason:             fmt.Sprintf("Durchschnittliche CPU-Auslastung %.1f%% - VM benoetigt mehr Ressourcen", avgCPU*100),
				VMContext:          vmCtx,
				ContextReason:      contextReasonForType(vmCtx, "cpu", recType),
			}
			_ = s.recRepo.Create(ctx, &rec)
			recommendations = append(recommendations, rec)
		}

		// Analyze Memory
		if vm.MaxMem > 0 {
			var totalMem, maxMem float64
			for _, dp := range rrdData {
				if dp.MaxMem > 0 {
					usage := dp.Mem / dp.MaxMem
					totalMem += usage
					if usage > maxMem {
						maxMem = usage
					}
				}
			}
			avgMem := totalMem / float64(len(rrdData))

			if avgMem < t.memOverprovision && vm.MaxMem > 512*1024*1024 {
				recType := string(model.RecommendationDownsize)
				rec := model.ResourceRecommendation{
					NodeID:             nodeID,
					VMID:               vm.VMID,
					VMName:             vm.Name,
					VMType:             vm.Type,
					ResourceType:       "memory",
					CurrentValue:       vm.MaxMem,
					RecommendedValue:   max(int64(float64(vm.MaxMem)*0.5), 512*1024*1024),
					AvgUsage:           avgMem * 100,
					MaxUsage:           maxMem * 100,
					RecommendationType: model.RecommendationDownsize,
					Reason:             fmt.Sprintf("Durchschnittliche Speichernutzung %.1f%% - VM ist ueberprovisioniert", avgMem*100),
					VMContext:          vmCtx,
					ContextReason:      contextReasonForType(vmCtx, "memory", recType),
				}
				_ = s.recRepo.Create(ctx, &rec)
				recommendations = append(recommendations, rec)
			} else if avgMem > t.memUnderprovision {
				recType := string(model.RecommendationUpsize)
				rec := model.ResourceRecommendation{
					NodeID:             nodeID,
					VMID:               vm.VMID,
					VMName:             vm.Name,
					VMType:             vm.Type,
					ResourceType:       "memory",
					CurrentValue:       vm.MaxMem,
					RecommendedValue:   int64(float64(vm.MaxMem) * 1.5),
					AvgUsage:           avgMem * 100,
					MaxUsage:           maxMem * 100,
					RecommendationType: model.RecommendationUpsize,
					Reason:             fmt.Sprintf("Durchschnittliche Speichernutzung %.1f%% - VM benoetigt mehr RAM", avgMem*100),
					VMContext:          vmCtx,
					ContextReason:      contextReasonForType(vmCtx, "memory", recType),
				}
				_ = s.recRepo.Create(ctx, &rec)
				recommendations = append(recommendations, rec)
			}
		}
	}

	slog.Info("rightsizing analysis completed",
		slog.String("node_id", nodeID.String()),
		slog.Int("recommendations", len(recommendations)),
	)

	return recommendations, nil
}

func (s *Service) ListByNode(ctx context.Context, nodeID uuid.UUID, limit int) ([]model.ResourceRecommendation, error) {
	return s.recRepo.ListByNode(ctx, nodeID, limit)
}

func (s *Service) ListAll(ctx context.Context, limit int) ([]model.ResourceRecommendation, error) {
	return s.recRepo.ListAll(ctx, limit)
}
