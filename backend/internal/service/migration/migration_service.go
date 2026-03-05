package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
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

	srcPVENode, err := s.resolvePVENodeName(ctx, srcClient, sourceNode)
	if err != nil {
		return nil, fmt.Errorf("resolve source PVE node: %w", err)
	}

	vms, err := srcClient.GetVMs(ctx, srcPVENode)
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
		s.broadcastProgress(m, nil)

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

	// Resolve actual PVE cluster node names (critical: DB name != PVE name)
	srcPVENode, err := s.resolvePVENodeName(ctx, srcClient, sourceNode)
	if err != nil {
		handleError("preparing", fmt.Errorf("resolve source PVE node: %w", err))
		return
	}
	tgtPVENode, err := s.resolvePVENodeName(ctx, tgtClient, targetNode)
	if err != nil {
		handleError("preparing", fmt.Errorf("resolve target PVE node: %w", err))
		return
	}

	slog.Info("migration: resolved PVE node names",
		slog.String("source_db_name", sourceNode.Name), slog.String("source_pve", srcPVENode),
		slog.String("target_db_name", targetNode.Name), slog.String("target_pve", tgtPVENode))

	// ── PRE-FLIGHT CHECKS ──
	s.updatePhase(ctx, m, model.MigrationStatusPreparing, "Pre-Flight-Checks...", 2)
	s.broadcastLog(m.ID, "Pre-Flight-Checks starten...")

	// 1. Check SSH connectivity
	s.broadcastLog(m.ID, "Prüfe SSH-Verbindung zu Source...")
	srcSSHCfg, err := s.getSSHConfig(sourceNode)
	if err != nil {
		handleError("preflight", fmt.Errorf("source SSH config: %w", err))
		return
	}
	srcSSHClient, err := s.sshPool.Get(sourceNode.ID.String(), srcSSHCfg)
	if err != nil {
		handleError("preflight", fmt.Errorf("SSH-Verbindung zu Source %s fehlgeschlagen: %w", sourceNode.Name, err))
		return
	}
	s.broadcastLog(m.ID, "✓ SSH zu Source verbunden")

	s.broadcastLog(m.ID, "Prüfe SSH-Verbindung zu Target...")
	tgtSSHCfg, err := s.getSSHConfig(targetNode)
	if err != nil {
		handleError("preflight", fmt.Errorf("target SSH config: %w", err))
		return
	}
	tgtSSHClient, err := s.sshPool.Get(targetNode.ID.String(), tgtSSHCfg)
	if err != nil {
		handleError("preflight", fmt.Errorf("SSH-Verbindung zu Target %s fehlgeschlagen: %w", targetNode.Name, err))
		return
	}
	s.broadcastLog(m.ID, "✓ SSH zu Target verbunden")

	// 2. Get VM info to check disk size
	s.broadcastLog(m.ID, fmt.Sprintf("Prüfe VM %d...", m.VMID))
	vms, err := srcClient.GetVMs(ctx, srcPVENode)
	if err != nil {
		handleError("preflight", fmt.Errorf("get VMs: %w", err))
		return
	}
	var vmInfo *proxmox.VMInfo
	for _, v := range vms {
		if v.VMID == m.VMID {
			vm := v
			vmInfo = &vm
			break
		}
	}
	if vmInfo == nil {
		handleError("preflight", fmt.Errorf("VM %d nicht auf Source-Node gefunden", m.VMID))
		return
	}
	// Use actual disk usage (not allocated size) for space estimates.
	// vzdump with zstd compresses to ~50% of used data.
	vmDiskUsed := vmInfo.Disk
	if vmDiskUsed <= 0 {
		vmDiskUsed = vmInfo.MaxDisk
	}
	estimatedBackupSize := int64(float64(vmDiskUsed) * 0.6) // 60% of used = safety margin with zstd
	s.broadcastLog(m.ID, fmt.Sprintf("✓ VM %d (%s) gefunden - Disk: %s belegt / %s alloziert, Status: %s",
		vmInfo.VMID, vmInfo.Name, formatBytesLog(vmDiskUsed), formatBytesLog(vmInfo.MaxDisk), vmInfo.Status))
	s.broadcastLog(m.ID, fmt.Sprintf("  Geschätzte Backup-Größe (zstd): ~%s", formatBytesLog(estimatedBackupSize)))

	// 3. Check target storage exists and has enough space
	s.broadcastLog(m.ID, fmt.Sprintf("Prüfe Zielspeicher '%s'...", m.TargetStorage))
	storages, err := tgtClient.GetClusterStorages(ctx)
	if err != nil {
		storages, err = tgtClient.GetStorage(ctx, tgtPVENode)
		if err != nil {
			handleError("preflight", fmt.Errorf("Zielspeicher konnte nicht abgefragt werden: %w", err))
			return
		}
	}
	var targetStorageInfo *proxmox.StorageInfo
	for _, st := range storages {
		if st.Storage == m.TargetStorage {
			stCopy := st
			targetStorageInfo = &stCopy
			break
		}
	}
	if targetStorageInfo == nil {
		handleError("preflight", fmt.Errorf("Speicher '%s' nicht auf Ziel-Node gefunden. Verfügbar: %s",
			m.TargetStorage, listStorageNames(storages)))
		return
	}
	s.broadcastLog(m.ID, fmt.Sprintf("✓ Zielspeicher '%s' gefunden - Frei: %s / Gesamt: %s (%.1f%% belegt)",
		targetStorageInfo.Storage, formatBytesLog(targetStorageInfo.Available),
		formatBytesLog(targetStorageInfo.Total), targetStorageInfo.UsagePercent))

	// Check if target storage has enough space for the VM disk (used size)
	if targetStorageInfo.Total > 0 && vmDiskUsed > 0 {
		if targetStorageInfo.Available < vmDiskUsed {
			handleError("preflight", fmt.Errorf("nicht genügend Speicherplatz auf '%s': benötigt ~%s, verfügbar %s",
				m.TargetStorage, formatBytesLog(vmDiskUsed), formatBytesLog(targetStorageInfo.Available)))
			return
		}
		s.broadcastLog(m.ID, fmt.Sprintf("✓ Genügend Speicherplatz auf Ziel: benötigt ~%s, verfügbar %s",
			formatBytesLog(vmDiskUsed), formatBytesLog(targetStorageInfo.Available)))
	}

	// 4. Find a vzdump-capable storage on source for backup (with enough space)
	vzdumpStorage := s.findVzdumpStorage(ctx, srcClient, srcPVENode, srcSSHClient, estimatedBackupSize)
	s.broadcastLog(m.ID, fmt.Sprintf("✓ Vzdump-Storage: %s", vzdumpStorage))

	// 5. Check source node disk space for vzdump file (warning, not blocking)
	s.broadcastLog(m.ID, "Prüfe freien Speicherplatz auf Source für Backup...")
	result, err := srcSSHClient.RunCommand(ctx, fmt.Sprintf("df -B1 $(pvesm path %s:) 2>/dev/null | tail -1 | awk '{print $4}'", vzdumpStorage))
	if err != nil || result.ExitCode != 0 {
		// Fallback: check /var/lib/vz/dump/
		result, err = srcSSHClient.RunCommand(ctx, "df -B1 /var/lib/vz/dump/ 2>/dev/null | tail -1 | awk '{print $4}'")
	}
	if err == nil && result.ExitCode == 0 {
		var freeSpace int64
		fmt.Sscanf(strings.TrimSpace(result.Stdout), "%d", &freeSpace)
		if freeSpace > 0 && estimatedBackupSize > 0 && freeSpace < estimatedBackupSize {
			s.broadcastLog(m.ID, fmt.Sprintf("⚠ Wenig Speicherplatz: geschätzt ~%s benötigt, %s verfügbar. Versuche trotzdem...",
				formatBytesLog(estimatedBackupSize), formatBytesLog(freeSpace)))
		} else if freeSpace > 0 {
			s.broadcastLog(m.ID, fmt.Sprintf("✓ Source-Speicher für Backup: %s frei (benötigt ~%s)",
				formatBytesLog(freeSpace), formatBytesLog(estimatedBackupSize)))
		}
	}

	s.broadcastLog(m.ID, "✓ Alle Pre-Flight-Checks bestanden!")
	s.updatePhase(ctx, m, model.MigrationStatusPreparing, "Pre-Flight-Checks bestanden", 4)

	// Handle VM state based on mode
	switch m.Mode {
	case model.MigrationModeStop:
		s.updatePhase(ctx, m, model.MigrationStatusPreparing, "VM wird heruntergefahren...", 4)
		s.broadcastLog(m.ID, fmt.Sprintf("Fahre VM %d herunter (Mode: stop)...", m.VMID))
		upid, err := srcClient.ShutdownVM(ctx, srcPVENode, m.VMID, m.VMType)
		if err != nil {
			// Fallback to stop
			s.broadcastLog(m.ID, "Graceful Shutdown fehlgeschlagen, erzwinge Stop...")
			upid, err = srcClient.StopVM(ctx, srcPVENode, m.VMID, m.VMType)
			if err != nil {
				handleError("preparing", fmt.Errorf("stop vm: %w", err))
				return
			}
		}
		if err := s.waitForTask(ctx, srcClient, srcPVENode, upid, 120*time.Second); err != nil {
			handleError("preparing", fmt.Errorf("wait for vm stop: %w", err))
			return
		}
		vmWasStopped = true
		s.broadcastLog(m.ID, "✓ VM heruntergefahren")
	case model.MigrationModeSuspend:
		s.updatePhase(ctx, m, model.MigrationStatusPreparing, "VM wird pausiert...", 4)
		s.broadcastLog(m.ID, fmt.Sprintf("Pausiere VM %d (Mode: suspend)...", m.VMID))
		upid, err := srcClient.SuspendVM(ctx, srcPVENode, m.VMID)
		if err != nil {
			handleError("preparing", fmt.Errorf("suspend vm: %w", err))
			return
		}
		if err := s.waitForTask(ctx, srcClient, srcPVENode, upid, 60*time.Second); err != nil {
			handleError("preparing", fmt.Errorf("wait for vm suspend: %w", err))
			return
		}
		vmWasStopped = true
		s.broadcastLog(m.ID, "✓ VM pausiert")
	case model.MigrationModeSnapshot:
		s.broadcastLog(m.ID, "Snapshot-Modus: VM läuft weiter während Backup")
	}

	s.updatePhase(ctx, m, model.MigrationStatusPreparing, "Vorbereitung abgeschlossen", 5)

	// PHASE 2: BACKING UP (5-40%)
	s.updatePhase(ctx, m, model.MigrationStatusBackingUp, "Vzdump-Backup wird erstellt...", 6)

	// Detach ISO/CD-ROM drives that may reference non-existing volumes (causes vzdump to fail)
	detachedMedia := s.detachMissingMedia(ctx, srcClient, srcPVENode, m.VMID, m.VMType, m.ID)
	if len(detachedMedia) > 0 {
		s.broadcastLog(m.ID, fmt.Sprintf("✓ %d CD/ISO-Laufwerk(e) temporär entfernt (werden nach Backup wiederhergestellt)", len(detachedMedia)))
	}

	s.broadcastLog(m.ID, fmt.Sprintf("Starte vzdump Backup (Mode: %s, Compress: zstd, Storage: %s)...", m.Mode, vzdumpStorage))

	vzdumpOpts := proxmox.VzdumpOptions{
		Mode:     string(m.Mode),
		Compress: "zstd",
		Storage:  vzdumpStorage,
	}
	vzdumpUPID, err := srcClient.CreateVzdump(ctx, srcPVENode, m.VMID, vzdumpOpts)
	if err != nil {
		handleError("backing_up", fmt.Errorf("vzdump konnte nicht gestartet werden: %w", err))
		return
	}
	m.VzdumpTaskUPID = &vzdumpUPID
	_ = s.migrationRepo.Update(ctx, m)
	s.broadcastLog(m.ID, fmt.Sprintf("Vzdump Task gestartet: %s", vzdumpUPID))

	// Poll vzdump task with live log streaming
	if err := s.pollTaskWithProgress(ctx, srcClient, srcPVENode, vzdumpUPID, m, 6, 38); err != nil {
		// Restore detached media before reporting error
		s.restoreMedia(ctx, srcClient, srcPVENode, m.VMID, m.VMType, detachedMedia)
		// Fetch task log for detailed error
		taskLog := s.getTaskLogForError(ctx, srcClient, srcPVENode, vzdumpUPID)
		handleError("backing_up", fmt.Errorf("vzdump fehlgeschlagen: %w\n\nTask-Log:\n%s", err, taskLog))
		return
	}
	// Restore detached media on source VM
	s.restoreMedia(ctx, srcClient, srcPVENode, m.VMID, m.VMType, detachedMedia)
	s.broadcastLog(m.ID, "✓ Vzdump Backup abgeschlossen")

	// Find the vzdump file path from task log
	vzdumpPath, err := s.findVzdumpPath(ctx, srcClient, srcPVENode, vzdumpUPID)
	if err != nil {
		handleError("backing_up", fmt.Errorf("vzdump-Pfad nicht gefunden: %w", err))
		return
	}
	m.VzdumpFilePath = &vzdumpPath
	s.broadcastLog(m.ID, fmt.Sprintf("Backup-Datei: %s", vzdumpPath))

	// Get file size
	sizeResult, err := srcSSHClient.RunCommand(ctx, fmt.Sprintf("stat -c %%s %q", vzdumpPath))
	if err == nil && sizeResult.ExitCode == 0 {
		var fileSize int64
		fmt.Sscanf(strings.TrimSpace(sizeResult.Stdout), "%d", &fileSize)
		m.VzdumpFileSize = &fileSize
		s.broadcastLog(m.ID, fmt.Sprintf("Backup-Größe: %s", formatBytesLog(fileSize)))
	}

	s.updatePhase(ctx, m, model.MigrationStatusBackingUp, "Backup abgeschlossen", 40)

	// PHASE 3: TRANSFERRING (40-80%)
	s.updatePhase(ctx, m, model.MigrationStatusTransferring, "Datei wird übertragen...", 41)
	s.broadcastLog(m.ID, "Starte Transfer von Source zu Target...")

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

	lastLogTime := time.Now()
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
			s.broadcastProgress(m, nil)

			// Log progress every 10 seconds
			if time.Since(lastLogTime) > 10*time.Second {
				lastLogTime = time.Now()
				pct := float64(0)
				if totalSize > 0 {
					pct = float64(bytesSent) / float64(totalSize) * 100
				}
				s.broadcastLog(m.ID, fmt.Sprintf("Transfer: %s / %s (%.1f%%) - %s/s",
					formatBytesLog(bytesSent), formatBytesLog(totalSize), pct, formatBytesLog(m.TransferSpeedBps)))
			}
		})
	if err != nil {
		// Cleanup partial file on target
		_, _ = tgtSSHClient.RunCommand(ctx, fmt.Sprintf("rm -f %q", targetVzdumpPath))
		handleError("transferring", fmt.Errorf("transfer fehlgeschlagen: %w", err))
		return
	}
	m.TransferBytesSent = transferred
	s.broadcastLog(m.ID, fmt.Sprintf("✓ Transfer abgeschlossen: %s übertragen", formatBytesLog(transferred)))
	s.updatePhase(ctx, m, model.MigrationStatusTransferring, "Transfer abgeschlossen", 80)

	// PHASE 4: RESTORING (80-95%)
	s.updatePhase(ctx, m, model.MigrationStatusRestoring, "VM wird auf Ziel wiederhergestellt...", 81)

	restoreVMID := m.VMID
	if m.NewVMID != nil {
		restoreVMID = *m.NewVMID
	}

	// Check if VM already exists on target (e.g. from a previous failed attempt) and remove it
	tgtVMs, _ := tgtClient.GetVMs(ctx, tgtPVENode)
	for _, v := range tgtVMs {
		if v.VMID == restoreVMID {
			s.broadcastLog(m.ID, fmt.Sprintf("⚠ VM %d existiert bereits auf Target - wird entfernt...", restoreVMID))
			// Stop if running
			if v.Status == "running" {
				stopUPID, err := tgtClient.StopVM(ctx, tgtPVENode, restoreVMID, m.VMType)
				if err == nil {
					_ = s.waitForTask(ctx, tgtClient, tgtPVENode, stopUPID, 60*time.Second)
				}
			}
			// Delete the VM
			delPath := fmt.Sprintf("/nodes/%s/%s/%d", tgtPVENode, m.VMType, restoreVMID)
			_, delErr := tgtClient.DeleteResource(ctx, delPath)
			if delErr != nil {
				s.broadcastLog(m.ID, fmt.Sprintf("⚠ VM %d konnte nicht gelöscht werden: %v", restoreVMID, delErr))
			} else {
				// Wait for delete to complete
				time.Sleep(3 * time.Second)
				s.broadcastLog(m.ID, fmt.Sprintf("✓ Alte VM %d auf Target entfernt", restoreVMID))
			}
			break
		}
	}

	// Convert filesystem path to Proxmox volume ID format.
	// API tokens cannot use raw filesystem paths (only root can).
	// /var/lib/vz/dump/vzdump-qemu-11000-... → local:backup/vzdump-qemu-11000-...
	restoreArchive := "local:backup/" + vzdumpFilename
	s.broadcastLog(m.ID, fmt.Sprintf("Starte Restore auf %s (VMID: %d, Storage: %s, Archive: %s)...",
		targetNode.Name, restoreVMID, m.TargetStorage, restoreArchive))

	restoreUPID, err := tgtClient.RestoreVM(ctx, tgtPVENode, restoreArchive, m.TargetStorage, restoreVMID)
	if err != nil {
		// Cleanup vzdump on target
		_, _ = tgtSSHClient.RunCommand(ctx, fmt.Sprintf("rm -f %q", targetVzdumpPath))
		handleError("restoring", fmt.Errorf("restore konnte nicht gestartet werden: %w", err))
		return
	}
	m.RestoreTaskUPID = &restoreUPID
	_ = s.migrationRepo.Update(ctx, m)
	s.broadcastLog(m.ID, fmt.Sprintf("Restore Task gestartet: %s", restoreUPID))

	if err := s.pollTaskWithProgress(ctx, tgtClient, tgtPVENode, restoreUPID, m, 81, 94); err != nil {
		taskLog := s.getTaskLogForError(ctx, tgtClient, tgtPVENode, restoreUPID)
		_, _ = tgtSSHClient.RunCommand(ctx, fmt.Sprintf("rm -f %q", targetVzdumpPath))
		handleError("restoring", fmt.Errorf("restore fehlgeschlagen: %w\n\nTask-Log:\n%s", err, taskLog))
		return
	}
	s.broadcastLog(m.ID, "✓ VM erfolgreich wiederhergestellt")
	s.updatePhase(ctx, m, model.MigrationStatusRestoring, "Wiederherstellung abgeschlossen", 95)

	// PHASE 5: CLEANING UP (95-100%)
	s.updatePhase(ctx, m, model.MigrationStatusCleaningUp, "Aufräum-Arbeiten...", 96)

	// Delete vzdump on target
	if m.CleanupTarget {
		s.broadcastLog(m.ID, "Lösche Backup-Datei auf Target...")
		_, _ = tgtSSHClient.RunCommand(ctx, fmt.Sprintf("rm -f %q", targetVzdumpPath))
	}

	// Delete vzdump on source
	if m.CleanupSource {
		s.broadcastLog(m.ID, "Lösche Backup-Datei auf Source...")
		_, _ = srcSSHClient.RunCommand(ctx, fmt.Sprintf("rm -f %q", vzdumpPath))
	}

	// Resume source VM if suspended
	if m.Mode == model.MigrationModeSuspend {
		s.broadcastLog(m.ID, "Setze Source-VM fort...")
		_, _ = srcClient.ResumeVM(ctx, srcPVENode, m.VMID)
	}

	// Mark completed
	m.Status = model.MigrationStatusCompleted
	m.Progress = 100
	m.CurrentStep = "Migration abgeschlossen"
	completedAt := time.Now()
	m.CompletedAt = &completedAt
	_ = s.migrationRepo.Update(ctx, m)

	duration := completedAt.Sub(*m.StartedAt)
	s.broadcastLog(m.ID, fmt.Sprintf("✓ Migration abgeschlossen! Dauer: %s, Übertragen: %s",
		duration.Round(time.Second), formatBytesLog(m.TransferBytesSent)))
	s.broadcastProgress(m, nil)

	slog.Info("migration completed",
		slog.String("id", m.ID.String()),
		slog.Int("vmid", m.VMID),
		slog.Int64("bytes_transferred", m.TransferBytesSent),
		slog.String("duration", duration.String()),
	)
}

func (s *Service) updatePhase(ctx context.Context, m *model.VMMigration, status model.MigrationStatus, step string, progress int) {
	m.Status = status
	m.CurrentStep = step
	m.Progress = progress
	_ = s.migrationRepo.Update(ctx, m)
	s.broadcastProgress(m, nil)
}

func (s *Service) broadcastProgress(m *model.VMMigration, logEntries []string) {
	resp := m.ToResponse()
	resp.LogEntries = logEntries
	s.wsHub.BroadcastMessage(monitor.WSMessage{
		Type: "migration_progress",
		Data: resp,
	})
}

// broadcastLog sends a single log line via WebSocket.
func (s *Service) broadcastLog(migrationID uuid.UUID, line string) {
	slog.Info("migration log", slog.String("migration_id", migrationID.String()), slog.String("line", line))
	s.wsHub.BroadcastMessage(monitor.WSMessage{
		Type: "migration_log",
		Data: map[string]interface{}{
			"migration_id": migrationID.String(),
			"line":         line,
			"timestamp":    time.Now().Format("15:04:05"),
		},
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
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	progressRange := endProgress - startProgress
	pollCount := 0
	lastLogLine := 0 // Track which log lines we've already sent

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Stream task log lines
			entries, err := client.GetTaskLog(ctx, node, upid, lastLogLine)
			if err == nil && len(entries) > 0 {
				for _, e := range entries {
					if e.LineNum >= lastLogLine && e.Text != "" {
						s.broadcastLog(m.ID, fmt.Sprintf("[PVE] %s", e.Text))
						if e.LineNum >= lastLogLine {
							lastLogLine = e.LineNum + 1
						}
					}
				}
			}

			status, err := client.GetTaskStatus(ctx, node, upid)
			if err != nil {
				continue
			}
			if !status.IsRunning() {
				// Fetch remaining log lines
				finalEntries, err := client.GetTaskLog(ctx, node, upid, lastLogLine)
				if err == nil {
					for _, e := range finalEntries {
						if e.Text != "" {
							s.broadcastLog(m.ID, fmt.Sprintf("[PVE] %s", e.Text))
						}
					}
				}

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
			s.broadcastProgress(m, nil)
		}
	}
}

// getTaskLogForError fetches the task log and returns the last N lines for error context.
func (s *Service) getTaskLogForError(ctx context.Context, client *proxmox.Client, node string, upid string) string {
	entries, err := client.GetTaskLog(ctx, node, upid, 0)
	if err != nil {
		return fmt.Sprintf("(Task-Log nicht verfügbar: %v)", err)
	}
	if len(entries) == 0 {
		return "(Keine Log-Einträge)"
	}

	var lines []string
	// Get last 20 lines max
	start := 0
	if len(entries) > 20 {
		start = len(entries) - 20
	}
	for _, e := range entries[start:] {
		if e.Text != "" {
			lines = append(lines, e.Text)
		}
	}
	return strings.Join(lines, "\n")
}

// findVzdumpStorage finds a storage capable of holding vzdump backups on the source node.
// It prefers storages with enough available space for the estimated backup size.
func (s *Service) findVzdumpStorage(ctx context.Context, client *proxmox.Client, pveNode string, sshClient *ssh.Client, estimatedSize int64) string {
	storages, err := client.GetStorage(ctx, pveNode)
	if err != nil {
		return "local" // fallback
	}

	// Collect all backup-capable storages, sorted by available space (largest first)
	type candidate struct {
		name      string
		available int64
	}
	var candidates []candidate

	for _, st := range storages {
		if strings.Contains(st.Content, "backup") || strings.Contains(st.Content, "vzdump") {
			candidates = append(candidates, candidate{name: st.Storage, available: st.Available})
		}
	}

	// Pick the one with the most available space
	if len(candidates) > 0 {
		best := candidates[0]
		for _, c := range candidates[1:] {
			if c.available > best.available {
				best = c
			}
		}
		slog.Info("migration: found vzdump storage",
			slog.String("storage", best.name),
			slog.String("available", formatBytesLog(best.available)),
			slog.String("estimated_backup", formatBytesLog(estimatedSize)))
		return best.name
	}

	return "local"
}

// detachMissingMedia checks VM config for CD/ISO drives and temporarily removes them
// to prevent vzdump from failing due to missing ISO files.
// Returns a map of drive -> original value for restoration.
func (s *Service) detachMissingMedia(ctx context.Context, client *proxmox.Client, node string, vmid int, vmType string, migrationID uuid.UUID) map[string]string {
	config, err := client.GetVMConfig(ctx, node, vmid, vmType)
	if err != nil {
		slog.Warn("migration: could not read VM config for media check", slog.Any("error", err))
		return nil
	}

	detached := make(map[string]string)
	// Check common CD/ISO drive keys: ide0-3, scsi*, sata*
	driveKeys := []string{"ide0", "ide1", "ide2", "ide3", "sata0", "sata1", "sata2", "sata3"}
	for _, key := range driveKeys {
		val, ok := config[key].(string)
		if !ok || val == "" {
			continue
		}
		// Check if it references an ISO file
		if strings.Contains(val, ",media=cdrom") || strings.Contains(val, ".iso") {
			s.broadcastLog(migrationID, fmt.Sprintf("  Entferne CD/ISO: %s = %s", key, val))
			// Set to none,media=cdrom
			params := url.Values{}
			params.Set(key, "none,media=cdrom")
			if err := client.SetVMConfig(ctx, node, vmid, vmType, params); err != nil {
				slog.Warn("migration: could not detach media",
					slog.String("drive", key), slog.Any("error", err))
				continue
			}
			detached[key] = val
		}
	}

	return detached
}

// restoreMedia re-attaches previously detached CD/ISO drives.
func (s *Service) restoreMedia(ctx context.Context, client *proxmox.Client, node string, vmid int, vmType string, detached map[string]string) {
	for key, val := range detached {
		params := url.Values{}
		params.Set(key, val)
		if err := client.SetVMConfig(ctx, node, vmid, vmType, params); err != nil {
			slog.Warn("migration: could not restore media",
				slog.String("drive", key), slog.String("value", val), slog.Any("error", err))
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

// resolvePVENodeName determines the correct Proxmox cluster node name for a
// given database node. It first checks the metadata cache, then falls back to
// querying the Proxmox API. This is critical because Proxmox API endpoints
// require the exact cluster node name (e.g. "pve1"), not our user-defined name.
func (s *Service) resolvePVENodeName(ctx context.Context, client *proxmox.Client, node *model.Node) (string, error) {
	// Fast path: use cached pve_node from metadata
	if len(node.Metadata) > 0 {
		var meta map[string]interface{}
		if err := json.Unmarshal(node.Metadata, &meta); err == nil {
			if pn, ok := meta["pve_node"].(string); ok && pn != "" {
				return pn, nil
			}
		}
	}

	// Slow path: resolve via Proxmox API
	pveNodes, err := client.GetNodes(ctx)
	if err != nil {
		return "", fmt.Errorf("get cluster nodes: %w", err)
	}
	if len(pveNodes) == 0 {
		return "", fmt.Errorf("no nodes found in cluster")
	}

	// Try to match node.Name against PVE node names
	for _, pn := range pveNodes {
		if strings.EqualFold(pn, node.Name) {
			s.cachePVENodeName(ctx, node, pn)
			return pn, nil
		}
	}

	// Fallback to first node
	pveNode := pveNodes[0]
	s.cachePVENodeName(ctx, node, pveNode)
	return pveNode, nil
}

// cachePVENodeName stores the resolved PVE node name in the node's metadata.
func (s *Service) cachePVENodeName(ctx context.Context, node *model.Node, pveNode string) {
	var meta map[string]interface{}
	if len(node.Metadata) > 0 {
		if err := json.Unmarshal(node.Metadata, &meta); err != nil {
			meta = make(map[string]interface{})
		}
	} else {
		meta = make(map[string]interface{})
	}
	meta["pve_node"] = pveNode
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return
	}
	node.Metadata = metaBytes
	if err := s.nodeRepo.Update(ctx, node); err != nil {
		slog.Warn("migration: failed to cache pve_node", slog.Any("error", err))
	}
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
	pveNode, err := s.resolvePVENodeName(ctx, srcClient, sourceNode)
	if err != nil {
		slog.Error("migration: failed to resolve PVE node for restart", slog.Any("error", err))
		return
	}
	_, err = srcClient.StartVM(ctx, pveNode, m.VMID, m.VMType)
	if err != nil {
		slog.Error("migration: failed to restart source VM", slog.Int("vmid", m.VMID), slog.Any("error", err))
	} else {
		slog.Info("migration: restarted source VM after failure", slog.Int("vmid", m.VMID))
	}
}

func listStorageNames(storages []proxmox.StorageInfo) string {
	names := make([]string, 0, len(storages))
	for _, s := range storages {
		names = append(names, s.Storage)
	}
	return strings.Join(names, ", ")
}

func formatBytesLog(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
