package migration

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/proxmox"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/crypto"
	"github.com/antigravity/prometheus/internal/service/monitor"
	"github.com/antigravity/prometheus/internal/ssh"
	"github.com/google/uuid"
)

type Service struct {
	migrationRepo repository.MigrationRepository
	nodeRepo      repository.NodeRepository
	encryptor     *crypto.Encryptor
	sshPool       *ssh.Pool
	clientFactory proxmox.ClientFactory
	wsHub         *monitor.WSHub

	mu        sync.Mutex
	cancels   map[uuid.UUID]context.CancelFunc
}

func NewService(
	migrationRepo repository.MigrationRepository,
	nodeRepo repository.NodeRepository,
	encryptor *crypto.Encryptor,
	sshPool *ssh.Pool,
	clientFactory proxmox.ClientFactory,
	wsHub *monitor.WSHub,
) *Service {
	return &Service{
		migrationRepo: migrationRepo,
		nodeRepo:      nodeRepo,
		encryptor:     encryptor,
		sshPool:       sshPool,
		clientFactory: clientFactory,
		wsHub:         wsHub,
		cancels:       make(map[uuid.UUID]context.CancelFunc),
	}
}

func (s *Service) StartMigration(ctx context.Context, req model.StartMigrationRequest, userID *uuid.UUID) (*model.VMMigration, error) {
	if req.SourceNodeID == req.TargetNodeID {
		return nil, fmt.Errorf("source and target node must differ")
	}
	if req.VMID <= 0 {
		return nil, fmt.Errorf("invalid vmid")
	}
	if req.TargetStorage == "" {
		return nil, fmt.Errorf("target_storage is required")
	}
	if req.Mode == "" {
		req.Mode = model.MigrationModeSnapshot
	}
	if !req.Mode.IsValid() {
		return nil, fmt.Errorf("invalid migration mode: %s", req.Mode)
	}

	// Validate nodes exist
	sourceNode, err := s.nodeRepo.GetByID(ctx, req.SourceNodeID)
	if err != nil {
		return nil, fmt.Errorf("source node: %w", err)
	}
	targetNode, err := s.nodeRepo.GetByID(ctx, req.TargetNodeID)
	if err != nil {
		return nil, fmt.Errorf("target node: %w", err)
	}
	if !sourceNode.IsOnline {
		return nil, fmt.Errorf("source node %s is offline", sourceNode.Name)
	}
	if !targetNode.IsOnline {
		return nil, fmt.Errorf("target node %s is offline", targetNode.Name)
	}

	// Get VM info to populate vm_name and vm_type
	srcClient, err := s.clientFactory.CreateClient(sourceNode)
	if err != nil {
		return nil, fmt.Errorf("create source client: %w", err)
	}

	vms, err := srcClient.GetVMs(ctx, sourceNode.Name)
	if err != nil {
		return nil, fmt.Errorf("get vms: %w", err)
	}

	var vmInfo *proxmox.VMInfo
	for _, v := range vms {
		if v.VMID == req.VMID {
			vm := v
			vmInfo = &vm
			break
		}
	}
	if vmInfo == nil {
		return nil, fmt.Errorf("VM %d not found on node %s", req.VMID, sourceNode.Name)
	}

	vmid := req.VMID
	if req.NewVMID != nil {
		vmid = *req.NewVMID
	}

	m := &model.VMMigration{
		SourceNodeID:  req.SourceNodeID,
		TargetNodeID:  req.TargetNodeID,
		VMID:          req.VMID,
		VMName:        vmInfo.Name,
		VMType:        vmInfo.Type,
		Status:        model.MigrationStatusPending,
		Mode:          req.Mode,
		TargetStorage: req.TargetStorage,
		NewVMID:       &vmid,
		CleanupSource: req.CleanupSource,
		CleanupTarget: req.CleanupTarget,
		InitiatedBy:   userID,
	}

	if err := s.migrationRepo.Create(ctx, m); err != nil {
		return nil, fmt.Errorf("create migration record: %w", err)
	}

	// Start async execution
	migCtx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.cancels[m.ID] = cancel
	s.mu.Unlock()

	go s.executeMigration(migCtx, m.ID)

	return m, nil
}

func (s *Service) CancelMigration(ctx context.Context, id uuid.UUID) error {
	m, err := s.migrationRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if m.Status.IsTerminal() {
		return fmt.Errorf("migration already in terminal state: %s", m.Status)
	}

	s.mu.Lock()
	cancel, ok := s.cancels[id]
	s.mu.Unlock()

	if ok {
		cancel()
	}

	m.Status = model.MigrationStatusCancelled
	m.ErrorMessage = "cancelled by user"
	now := time.Now()
	m.CompletedAt = &now
	return s.migrationRepo.Update(ctx, m)
}

func (s *Service) GetMigration(ctx context.Context, id uuid.UUID) (*model.VMMigration, error) {
	return s.migrationRepo.GetByID(ctx, id)
}

func (s *Service) ListMigrations(ctx context.Context) ([]model.VMMigration, error) {
	return s.migrationRepo.List(ctx)
}

func (s *Service) ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.VMMigration, error) {
	return s.migrationRepo.ListByNode(ctx, nodeID)
}

func (s *Service) DeleteMigration(ctx context.Context, id uuid.UUID) error {
	m, err := s.migrationRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !m.Status.IsTerminal() {
		return fmt.Errorf("cannot delete active migration")
	}
	return s.migrationRepo.Delete(ctx, id)
}

// executeMigration runs the 5-phase migration pipeline.
func (s *Service) executeMigration(ctx context.Context, migrationID uuid.UUID) {
	defer func() {
		s.mu.Lock()
		delete(s.cancels, migrationID)
		s.mu.Unlock()
	}()

	m, err := s.migrationRepo.GetByID(ctx, migrationID)
	if err != nil {
		slog.Error("migration: load failed", slog.String("id", migrationID.String()), slog.Any("error", err))
		return
	}

	now := time.Now()
	m.StartedAt = &now

	var vmWasStopped bool

	// Error handler: update status + restart VM if needed
	handleError := func(phase string, err error) {
		slog.Error("migration failed", slog.String("phase", phase), slog.Any("error", err))
		m.Status = model.MigrationStatusFailed
		m.ErrorMessage = fmt.Sprintf("%s: %v", phase, err)
		completedAt := time.Now()
		m.CompletedAt = &completedAt
		_ = s.migrationRepo.Update(ctx, m)
		s.broadcastProgress(m)

		// If we stopped the VM and migration failed, restart it
		if vmWasStopped && m.Mode == model.MigrationModeStop {
			s.tryRestartSourceVM(context.Background(), m)
		}
	}

	// PHASE 1: PREPARING (0-5%)
	s.updatePhase(ctx, m, model.MigrationStatusPreparing, "Vorbereitung...", 1)

	sourceNode, err := s.nodeRepo.GetByID(ctx, m.SourceNodeID)
	if err != nil {
		handleError("preparing", fmt.Errorf("load source node: %w", err))
		return
	}
	targetNode, err := s.nodeRepo.GetByID(ctx, m.TargetNodeID)
	if err != nil {
		handleError("preparing", fmt.Errorf("load target node: %w", err))
		return
	}

	srcClient, err := s.clientFactory.CreateClient(sourceNode)
	if err != nil {
		handleError("preparing", fmt.Errorf("create source client: %w", err))
		return
	}

	tgtClient, err := s.clientFactory.CreateClient(targetNode)
	if err != nil {
		handleError("preparing", fmt.Errorf("create target client: %w", err))
		return
	}

	// Verify target storage exists
	storages, err := tgtClient.GetStorage(ctx, targetNode.Name)
	if err != nil {
		handleError("preparing", fmt.Errorf("get target storage: %w", err))
		return
	}
	storageFound := false
	for _, st := range storages {
		if st.Storage == m.TargetStorage {
			storageFound = true
			break
		}
	}
	if !storageFound {
		handleError("preparing", fmt.Errorf("storage %q not found on target node", m.TargetStorage))
		return
	}

	// Handle VM state based on mode
	switch m.Mode {
	case model.MigrationModeStop:
		s.updatePhase(ctx, m, model.MigrationStatusPreparing, "VM wird heruntergefahren...", 3)
		upid, err := srcClient.ShutdownVM(ctx, sourceNode.Name, m.VMID, m.VMType)
		if err != nil {
			// Fallback to stop
			upid, err = srcClient.StopVM(ctx, sourceNode.Name, m.VMID, m.VMType)
			if err != nil {
				handleError("preparing", fmt.Errorf("stop vm: %w", err))
				return
			}
		}
		if err := s.waitForTask(ctx, srcClient, sourceNode.Name, upid, 120*time.Second); err != nil {
			handleError("preparing", fmt.Errorf("wait for vm stop: %w", err))
			return
		}
		vmWasStopped = true
	case model.MigrationModeSuspend:
		s.updatePhase(ctx, m, model.MigrationStatusPreparing, "VM wird pausiert...", 3)
		upid, err := srcClient.SuspendVM(ctx, sourceNode.Name, m.VMID)
		if err != nil {
			handleError("preparing", fmt.Errorf("suspend vm: %w", err))
			return
		}
		if err := s.waitForTask(ctx, srcClient, sourceNode.Name, upid, 60*time.Second); err != nil {
			handleError("preparing", fmt.Errorf("wait for vm suspend: %w", err))
			return
		}
		vmWasStopped = true
	}

	s.updatePhase(ctx, m, model.MigrationStatusPreparing, "Vorbereitung abgeschlossen", 5)

	// PHASE 2: BACKING UP (5-40%)
	s.updatePhase(ctx, m, model.MigrationStatusBackingUp, "Vzdump-Backup wird erstellt...", 6)

	vzdumpOpts := proxmox.VzdumpOptions{
		Mode:     string(m.Mode),
		Compress: "zstd",
	}
	vzdumpUPID, err := srcClient.CreateVzdump(ctx, sourceNode.Name, m.VMID, vzdumpOpts)
	if err != nil {
		handleError("backing_up", fmt.Errorf("create vzdump: %w", err))
		return
	}
	m.VzdumpTaskUPID = &vzdumpUPID
	_ = s.migrationRepo.Update(ctx, m)

	// Poll vzdump task
	if err := s.pollTaskWithProgress(ctx, srcClient, sourceNode.Name, vzdumpUPID, m, 6, 38); err != nil {
		handleError("backing_up", fmt.Errorf("vzdump task: %w", err))
		return
	}

	// Find the vzdump file path from task log
	vzdumpPath, err := s.findVzdumpPath(ctx, srcClient, sourceNode.Name, vzdumpUPID)
	if err != nil {
		handleError("backing_up", fmt.Errorf("find vzdump path: %w", err))
		return
	}
	m.VzdumpFilePath = &vzdumpPath

	// Get file size
	srcSSHCfg, err := s.getSSHConfig(sourceNode)
	if err != nil {
		handleError("backing_up", fmt.Errorf("source ssh config: %w", err))
		return
	}
	srcSSHClient, err := s.sshPool.Get(sourceNode.ID.String(), srcSSHCfg)
	if err != nil {
		handleError("backing_up", fmt.Errorf("source ssh connect: %w", err))
		return
	}

	sizeResult, err := srcSSHClient.RunCommand(ctx, fmt.Sprintf("stat -c %%s %q", vzdumpPath))
	if err == nil && sizeResult.ExitCode == 0 {
		var fileSize int64
		fmt.Sscanf(strings.TrimSpace(sizeResult.Stdout), "%d", &fileSize)
		m.VzdumpFileSize = &fileSize
	}

	s.updatePhase(ctx, m, model.MigrationStatusBackingUp, "Backup abgeschlossen", 40)

	// PHASE 3: TRANSFERRING (40-80%)
	s.updatePhase(ctx, m, model.MigrationStatusTransferring, "Datei wird uebertragen...", 41)

	tgtSSHCfg, err := s.getSSHConfig(targetNode)
	if err != nil {
		handleError("transferring", fmt.Errorf("target ssh config: %w", err))
		return
	}
	tgtSSHClient, err := s.sshPool.Get(targetNode.ID.String(), tgtSSHCfg)
	if err != nil {
		handleError("transferring", fmt.Errorf("target ssh connect: %w", err))
		return
	}

	// Target path: /var/lib/vz/dump/<filename>
	vzdumpFilename := vzdumpPath
	if idx := strings.LastIndex(vzdumpPath, "/"); idx >= 0 {
		vzdumpFilename = vzdumpPath[idx+1:]
	}
	targetVzdumpPath := "/var/lib/vz/dump/" + vzdumpFilename

	// Ensure target directory exists
	_, _ = tgtSSHClient.RunCommand(ctx, "mkdir -p /var/lib/vz/dump")

	totalSize := int64(0)
	if m.VzdumpFileSize != nil {
		totalSize = *m.VzdumpFileSize
	}

	transferred, err := ssh.StreamCopyNodeToNode(ctx, srcSSHClient, tgtSSHClient, vzdumpPath, targetVzdumpPath,
		func(bytesSent int64) {
			m.TransferBytesSent = bytesSent
			if totalSize > 0 {
				transferProgress := int(float64(bytesSent) / float64(totalSize) * 39) // 40% range
				m.Progress = 41 + transferProgress
				if m.Progress > 79 {
					m.Progress = 79
				}
			}
			// Calculate speed
			if m.StartedAt != nil {
				elapsed := time.Since(*m.StartedAt).Seconds()
				if elapsed > 0 {
					m.TransferSpeedBps = int64(float64(bytesSent) / elapsed)
				}
			}
			s.broadcastProgress(m)
		})
	if err != nil {
		// Cleanup partial file on target
		_, _ = tgtSSHClient.RunCommand(ctx, fmt.Sprintf("rm -f %q", targetVzdumpPath))
		handleError("transferring", fmt.Errorf("stream copy: %w", err))
		return
	}
	m.TransferBytesSent = transferred
	s.updatePhase(ctx, m, model.MigrationStatusTransferring, "Transfer abgeschlossen", 80)

	// PHASE 4: RESTORING (80-95%)
	s.updatePhase(ctx, m, model.MigrationStatusRestoring, "VM wird auf Ziel wiederhergestellt...", 81)

	restoreVMID := m.VMID
	if m.NewVMID != nil {
		restoreVMID = *m.NewVMID
	}

	restoreUPID, err := tgtClient.RestoreVM(ctx, targetNode.Name, targetVzdumpPath, m.TargetStorage, restoreVMID)
	if err != nil {
		// Cleanup vzdump on target
		_, _ = tgtSSHClient.RunCommand(ctx, fmt.Sprintf("rm -f %q", targetVzdumpPath))
		handleError("restoring", fmt.Errorf("restore vm: %w", err))
		return
	}
	m.RestoreTaskUPID = &restoreUPID
	_ = s.migrationRepo.Update(ctx, m)

	if err := s.pollTaskWithProgress(ctx, tgtClient, targetNode.Name, restoreUPID, m, 81, 94); err != nil {
		_, _ = tgtSSHClient.RunCommand(ctx, fmt.Sprintf("rm -f %q", targetVzdumpPath))
		handleError("restoring", fmt.Errorf("restore task: %w", err))
		return
	}
	s.updatePhase(ctx, m, model.MigrationStatusRestoring, "Wiederherstellung abgeschlossen", 95)

	// PHASE 5: CLEANING UP (95-100%)
	s.updatePhase(ctx, m, model.MigrationStatusCleaningUp, "Aufraeum-Arbeiten...", 96)

	// Delete vzdump on target
	if m.CleanupTarget {
		_, _ = tgtSSHClient.RunCommand(ctx, fmt.Sprintf("rm -f %q", targetVzdumpPath))
	}

	// Delete vzdump on source
	if m.CleanupSource {
		_, _ = srcSSHClient.RunCommand(ctx, fmt.Sprintf("rm -f %q", vzdumpPath))
	}

	// Resume source VM if suspended
	if m.Mode == model.MigrationModeSuspend {
		_, _ = srcClient.ResumeVM(ctx, sourceNode.Name, m.VMID)
	}

	// Mark completed
	m.Status = model.MigrationStatusCompleted
	m.Progress = 100
	m.CurrentStep = "Migration abgeschlossen"
	completedAt := time.Now()
	m.CompletedAt = &completedAt
	_ = s.migrationRepo.Update(ctx, m)
	s.broadcastProgress(m)

	slog.Info("migration completed",
		slog.String("id", m.ID.String()),
		slog.Int("vmid", m.VMID),
		slog.Int64("bytes_transferred", m.TransferBytesSent),
	)
}

func (s *Service) updatePhase(ctx context.Context, m *model.VMMigration, status model.MigrationStatus, step string, progress int) {
	m.Status = status
	m.CurrentStep = step
	m.Progress = progress
	_ = s.migrationRepo.Update(ctx, m)
	s.broadcastProgress(m)
}

func (s *Service) broadcastProgress(m *model.VMMigration) {
	s.wsHub.BroadcastMessage(monitor.WSMessage{
		Type: "migration_progress",
		Data: m.ToResponse(),
	})
}

func (s *Service) waitForTask(ctx context.Context, client *proxmox.Client, node string, upid string, timeout time.Duration) error {
	deadline := time.After(timeout)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("task timeout after %s", timeout)
		case <-ticker.C:
			status, err := client.GetTaskStatus(ctx, node, upid)
			if err != nil {
				continue
			}
			if !status.IsRunning() {
				if status.IsSuccess() {
					return nil
				}
				return fmt.Errorf("task failed: %s", status.ExitStatus)
			}
		}
	}
}

func (s *Service) pollTaskWithProgress(ctx context.Context, client *proxmox.Client, node string, upid string, m *model.VMMigration, startProgress, endProgress int) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	progressRange := endProgress - startProgress
	pollCount := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			status, err := client.GetTaskStatus(ctx, node, upid)
			if err != nil {
				continue
			}
			if !status.IsRunning() {
				if status.IsSuccess() {
					return nil
				}
				return fmt.Errorf("task failed: %s", status.ExitStatus)
			}
			// Increment progress slowly
			pollCount++
			fakeProgress := startProgress + int(float64(progressRange)*float64(pollCount)/(float64(pollCount)+10))
			if fakeProgress > endProgress-1 {
				fakeProgress = endProgress - 1
			}
			m.Progress = fakeProgress
			s.broadcastProgress(m)
		}
	}
}

func (s *Service) findVzdumpPath(ctx context.Context, client *proxmox.Client, node string, upid string) (string, error) {
	entries, err := client.GetTaskLog(ctx, node, upid, 0)
	if err != nil {
		return "", err
	}
	// Look for "creating archive" or the .vma/.tar file path
	for _, e := range entries {
		text := e.Text
		if strings.Contains(text, "/vzdump-") && (strings.Contains(text, ".vma") || strings.Contains(text, ".tar")) {
			// Extract the path
			parts := strings.Fields(text)
			for _, p := range parts {
				if strings.Contains(p, "/vzdump-") {
					// Clean up path (remove quotes, trailing chars)
					p = strings.Trim(p, "'\"")
					return p, nil
				}
			}
		}
	}
	return "", fmt.Errorf("vzdump file path not found in task log")
}

func (s *Service) getSSHConfig(node *model.Node) (ssh.SSHConfig, error) {
	privateKey, err := s.encryptor.Decrypt(node.SSHPrivateKey)
	if err != nil {
		return ssh.SSHConfig{}, fmt.Errorf("decrypt ssh key: %w", err)
	}

	port := node.SSHPort
	if port == 0 {
		port = 22
	}
	user := node.SSHUser
	if user == "" {
		user = "root"
	}

	return ssh.SSHConfig{
		Host:       node.Hostname,
		Port:       port,
		User:       user,
		PrivateKey: privateKey,
	}, nil
}

func (s *Service) tryRestartSourceVM(ctx context.Context, m *model.VMMigration) {
	sourceNode, err := s.nodeRepo.GetByID(ctx, m.SourceNodeID)
	if err != nil {
		slog.Error("migration: failed to load source node for restart", slog.Any("error", err))
		return
	}
	srcClient, err := s.clientFactory.CreateClient(sourceNode)
	if err != nil {
		slog.Error("migration: failed to create client for restart", slog.Any("error", err))
		return
	}
	_, err = srcClient.StartVM(ctx, sourceNode.Name, m.VMID, m.VMType)
	if err != nil {
		slog.Error("migration: failed to restart source VM", slog.Int("vmid", m.VMID), slog.Any("error", err))
	} else {
		slog.Info("migration: restarted source VM after failure", slog.Int("vmid", m.VMID))
	}
}
