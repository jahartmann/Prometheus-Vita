package recovery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
)

type RunbookService struct {
	runbookRepo repository.RunbookRepository
	profileRepo repository.NodeProfileRepository
	nodeRepo    repository.NodeRepository
}

func NewRunbookService(
	runbookRepo repository.RunbookRepository,
	profileRepo repository.NodeProfileRepository,
	nodeRepo repository.NodeRepository,
) *RunbookService {
	return &RunbookService{
		runbookRepo: runbookRepo,
		profileRepo: profileRepo,
		nodeRepo:    nodeRepo,
	}
}

func (s *RunbookService) GenerateRunbook(ctx context.Context, nodeID uuid.UUID, scenario string) (*model.RecoveryRunbook, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}

	profile, _ := s.profileRepo.GetLatest(ctx, nodeID)

	steps := s.generateSteps(scenario, node, profile)
	stepsJSON, err := json.Marshal(steps)
	if err != nil {
		return nil, fmt.Errorf("marshal steps: %w", err)
	}

	runbook := &model.RecoveryRunbook{
		NodeID:   &nodeID,
		Title:    fmt.Sprintf("%s - %s", s.scenarioTitle(scenario), node.Name),
		Scenario: scenario,
		Steps:    stepsJSON,
	}

	if err := s.runbookRepo.Create(ctx, runbook); err != nil {
		return nil, fmt.Errorf("create runbook: %w", err)
	}

	return runbook, nil
}

func (s *RunbookService) GetRunbook(ctx context.Context, id uuid.UUID) (*model.RecoveryRunbook, error) {
	return s.runbookRepo.GetByID(ctx, id)
}

func (s *RunbookService) ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.RecoveryRunbook, error) {
	return s.runbookRepo.ListByNode(ctx, nodeID)
}

func (s *RunbookService) ListTemplates(ctx context.Context) ([]model.RecoveryRunbook, error) {
	return s.runbookRepo.ListTemplates(ctx)
}

func (s *RunbookService) UpdateRunbook(ctx context.Context, id uuid.UUID, req model.UpdateRunbookRequest) (*model.RecoveryRunbook, error) {
	runbook, err := s.runbookRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get runbook: %w", err)
	}

	if req.Title != nil {
		runbook.Title = *req.Title
	}
	if req.Steps != nil {
		runbook.Steps = *req.Steps
	}

	if err := s.runbookRepo.Update(ctx, runbook); err != nil {
		return nil, fmt.Errorf("update runbook: %w", err)
	}

	return runbook, nil
}

func (s *RunbookService) DeleteRunbook(ctx context.Context, id uuid.UUID) error {
	return s.runbookRepo.Delete(ctx, id)
}

func (s *RunbookService) SimulateDR(ctx context.Context, req model.DRSimulationRequest) (*model.DRSimulationResult, error) {
	node, err := s.nodeRepo.GetByID(ctx, req.NodeID)
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}

	var checks []model.DRSimulationCheck

	// Check 1: Node is online
	checks = append(checks, model.DRSimulationCheck{
		Name:    "Node Online",
		Passed:  node.IsOnline,
		Message: boolMessage(node.IsOnline, "Node is online and reachable", "Node is offline"),
	})

	// Check 2: Profile exists
	profile, profileErr := s.profileRepo.GetLatest(ctx, req.NodeID)
	hasProfile := profileErr == nil && profile != nil
	checks = append(checks, model.DRSimulationCheck{
		Name:    "Hardware Profile",
		Passed:  hasProfile,
		Message: boolMessage(hasProfile, "Hardware profile available", "No hardware profile collected"),
	})

	// Check 3: SSH configured
	hasSSH := node.SSHPort > 0 && node.SSHUser != ""
	checks = append(checks, model.DRSimulationCheck{
		Name:    "SSH Configuration",
		Passed:  hasSSH,
		Message: boolMessage(hasSSH, "SSH access configured", "SSH access not configured"),
	})

	// Check 4: Runbook exists for scenario
	runbooks, _ := s.runbookRepo.ListByNode(ctx, req.NodeID)
	hasRunbook := false
	for _, rb := range runbooks {
		if rb.Scenario == req.Scenario {
			hasRunbook = true
			break
		}
	}
	checks = append(checks, model.DRSimulationCheck{
		Name:    "Recovery Runbook",
		Passed:  hasRunbook,
		Message: boolMessage(hasRunbook, "Runbook available for scenario", "No runbook for this scenario"),
	})

	// Determine overall readiness
	allPassed := true
	passedCount := 0
	for _, c := range checks {
		if !c.Passed {
			allPassed = false
		} else {
			passedCount++
		}
	}

	summary := fmt.Sprintf("%d/%d checks passed", passedCount, len(checks))
	if allPassed {
		summary = "All prerequisite checks passed. System is ready for DR scenario: " + req.Scenario
	}

	return &model.DRSimulationResult{
		NodeID:   req.NodeID,
		Scenario: req.Scenario,
		Ready:    allPassed,
		Checks:   checks,
		Summary:  summary,
	}, nil
}

func (s *RunbookService) scenarioTitle(scenario string) string {
	switch scenario {
	case "node_replacement":
		return "Node Replacement Recovery"
	case "disk_failure":
		return "Disk Failure Recovery"
	case "network_failure":
		return "Network Failure Recovery"
	case "cluster_recovery":
		return "Cluster Recovery"
	case "full_restore":
		return "Full System Restore"
	default:
		return "Recovery Runbook"
	}
}

func (s *RunbookService) generateSteps(scenario string, node *model.Node, profile *model.NodeProfile) []model.RunbookStep {
	switch scenario {
	case "node_replacement":
		return s.nodeReplacementSteps(node, profile)
	case "disk_failure":
		return s.diskFailureSteps(node, profile)
	case "network_failure":
		return s.networkFailureSteps(node)
	case "cluster_recovery":
		return s.clusterRecoverySteps(node, profile)
	case "full_restore":
		return s.fullRestoreSteps(node, profile)
	default:
		return []model.RunbookStep{
			{Title: "Custom Recovery", Description: "Define custom recovery steps for this scenario.", IsManual: true},
		}
	}
}

func (s *RunbookService) nodeReplacementSteps(node *model.Node, profile *model.NodeProfile) []model.RunbookStep {
	steps := []model.RunbookStep{
		{
			Title:       "Prepare Replacement Hardware",
			Description: fmt.Sprintf("Provision replacement hardware matching node '%s' specifications.", node.Name),
			IsManual:    true,
		},
		{
			Title:       "Install Proxmox VE",
			Description: "Install Proxmox VE on the replacement hardware.",
			IsManual:    true,
		},
	}

	if profile != nil && profile.PVEVersion != "" {
		steps = append(steps, model.RunbookStep{
			Title:       "Verify PVE Version",
			Description: fmt.Sprintf("Ensure PVE version matches: %s", profile.PVEVersion),
			Command:     "pveversion",
			IsManual:    false,
		})
	}

	steps = append(steps,
		model.RunbookStep{
			Title:       "Configure Network",
			Description: fmt.Sprintf("Configure network with hostname %s on port %d.", node.Hostname, node.Port),
			Command:     fmt.Sprintf("hostnamectl set-hostname %s", node.Name),
			IsManual:    false,
		},
		model.RunbookStep{
			Title:       "Restore Configuration Backup",
			Description: "Restore the latest configuration backup to the new node.",
			IsManual:    true,
		},
		model.RunbookStep{
			Title:       "Verify Services",
			Description: "Verify all Proxmox services are running correctly.",
			Command:     "systemctl status pve-cluster pvedaemon pveproxy",
			IsManual:    false,
		},
		model.RunbookStep{
			Title:       "Validate Connectivity",
			Description: "Validate the node is accessible via API and SSH.",
			IsManual:    true,
		},
	)

	return steps
}

func (s *RunbookService) diskFailureSteps(node *model.Node, profile *model.NodeProfile) []model.RunbookStep {
	steps := []model.RunbookStep{
		{
			Title:       "Identify Failed Disk",
			Description: "Identify which disk has failed using system logs and SMART data.",
			Command:     "dmesg | grep -i 'error\\|fail' | tail -20",
			IsManual:    false,
		},
		{
			Title:       "Check RAID Status",
			Description: "Check current RAID status if applicable.",
			Command:     "cat /proc/mdstat 2>/dev/null; zpool status 2>/dev/null",
			IsManual:    false,
		},
		{
			Title:       "Replace Failed Disk",
			Description: fmt.Sprintf("Physically replace the failed disk in node '%s'.", node.Name),
			IsManual:    true,
		},
		{
			Title:       "Rebuild RAID/ZFS",
			Description: "Add new disk to the array and start rebuild.",
			IsManual:    true,
		},
		{
			Title:       "Verify Disk Health",
			Description: "Verify the new disk is healthy and rebuild is progressing.",
			Command:     "lsblk && smartctl -H /dev/sdX",
			IsManual:    false,
		},
		{
			Title:       "Monitor Rebuild Progress",
			Description: "Monitor RAID/ZFS rebuild until completion.",
			Command:     "watch -n 5 'cat /proc/mdstat 2>/dev/null; zpool status 2>/dev/null'",
			IsManual:    false,
		},
	}
	return steps
}

func (s *RunbookService) networkFailureSteps(node *model.Node) []model.RunbookStep {
	return []model.RunbookStep{
		{
			Title:       "Check Physical Connectivity",
			Description: "Verify network cables and switch ports are functioning.",
			IsManual:    true,
		},
		{
			Title:       "Check Interface Status",
			Description: "Check the status of all network interfaces.",
			Command:     "ip link show && ip addr show",
			IsManual:    false,
		},
		{
			Title:       "Restart Networking",
			Description: "Restart the networking service.",
			Command:     "systemctl restart networking",
			IsManual:    false,
		},
		{
			Title:       "Verify Connectivity",
			Description: fmt.Sprintf("Verify node '%s' is reachable.", node.Name),
			Command:     fmt.Sprintf("ping -c 3 %s", node.Hostname),
			IsManual:    false,
		},
		{
			Title:       "Check Cluster Communication",
			Description: "Verify cluster communication is restored.",
			Command:     "pvecm status",
			IsManual:    false,
		},
	}
}

func (s *RunbookService) clusterRecoverySteps(node *model.Node, profile *model.NodeProfile) []model.RunbookStep {
	return []model.RunbookStep{
		{
			Title:       "Assess Cluster State",
			Description: "Check the current state of the Proxmox cluster.",
			Command:     "pvecm status",
			IsManual:    false,
		},
		{
			Title:       "Check Quorum",
			Description: "Verify cluster quorum status.",
			Command:     "pvecm expected 1",
			IsManual:    false,
		},
		{
			Title:       "Verify Corosync",
			Description: "Check Corosync status and fix if needed.",
			Command:     "systemctl status corosync",
			IsManual:    false,
		},
		{
			Title:       "Restart Cluster Services",
			Description: "Restart all cluster-related services.",
			Command:     "systemctl restart pve-cluster corosync pvedaemon pveproxy",
			IsManual:    false,
		},
		{
			Title:       "Verify VM Migration",
			Description: "Verify VMs can be migrated between nodes.",
			IsManual:    true,
		},
		{
			Title:       "Validate Storage Replication",
			Description: "Verify storage replication is functioning.",
			Command:     "pvesr status",
			IsManual:    false,
		},
	}
}

func (s *RunbookService) fullRestoreSteps(node *model.Node, profile *model.NodeProfile) []model.RunbookStep {
	steps := []model.RunbookStep{
		{
			Title:       "Prepare Bare Metal",
			Description: fmt.Sprintf("Prepare bare metal hardware for full restore of node '%s'.", node.Name),
			IsManual:    true,
		},
		{
			Title:       "Install Base OS",
			Description: "Install Proxmox VE from ISO.",
			IsManual:    true,
		},
	}

	if profile != nil {
		if profile.PVEVersion != "" {
			steps = append(steps, model.RunbookStep{
				Title:       "Match PVE Version",
				Description: fmt.Sprintf("Install matching PVE version: %s", profile.PVEVersion),
				Command:     "pveversion",
				IsManual:    false,
			})
		}
		if profile.KernelVersion != "" {
			steps = append(steps, model.RunbookStep{
				Title:       "Verify Kernel",
				Description: fmt.Sprintf("Expected kernel: %s", profile.KernelVersion),
				Command:     "uname -r",
				IsManual:    false,
			})
		}
	}

	steps = append(steps,
		model.RunbookStep{
			Title:       "Configure Networking",
			Description: "Restore network configuration from backup.",
			IsManual:    true,
		},
		model.RunbookStep{
			Title:       "Restore Configuration Files",
			Description: "Restore all configuration files from the latest backup.",
			IsManual:    true,
		},
		model.RunbookStep{
			Title:       "Restore VMs and Containers",
			Description: "Restore all VMs and containers from PBS backup.",
			IsManual:    true,
		},
		model.RunbookStep{
			Title:       "Rejoin Cluster",
			Description: "Rejoin the Proxmox cluster if applicable.",
			Command:     "pvecm add <cluster-node-ip>",
			IsManual:    false,
		},
		model.RunbookStep{
			Title:       "Final Verification",
			Description: "Run full system verification.",
			Command:     "pveversion && pvecm status && systemctl status pve-cluster pvedaemon pveproxy",
			IsManual:    false,
		},
	)

	return steps
}

func boolMessage(condition bool, trueMsg, falseMsg string) string {
	if condition {
		return trueMsg
	}
	return falseMsg
}
