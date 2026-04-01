package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/proxmox"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/util"
	"github.com/antigravity/prometheus/internal/service/crypto"
	"github.com/antigravity/prometheus/internal/service/monitor"
	"github.com/antigravity/prometheus/internal/ssh"
	"github.com/google/uuid"
)

// Shell-safe validation patterns to prevent command injection via user-controlled values
// that end up in SSH/SCP commands executed on Proxmox hosts.
var (
	// validStorageName matches Proxmox storage identifiers (alphanumeric, hyphens, underscores)
	validStorageName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)
	// validHostname matches RFC-1123 hostnames and IPv4 addresses
	validHostname = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9.\-]{0,251}[a-zA-Z0-9])?$`)
	// validSSHUser matches standard Unix usernames
	validSSHUser = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)
	// validVzdumpPath matches safe vzdump file paths (no shell metacharacters)
	validVzdumpPath = regexp.MustCompile(`^/[a-zA-Z0-9/_.\-]+$`)
)

// shellQuote wraps a string in single quotes for safe use in POSIX shell commands.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// maxMigrationDuration is the absolute maximum time a migration can run before being cancelled.
// This prevents stuck migrations from consuming resources indefinitely.
const maxMigrationDuration = 6 * time.Hour

type Service struct {
	migrationRepo repository.MigrationRepository
	nodeRepo      repository.NodeRepository
	encryptor     *crypto.Encryptor
	sshPool       *ssh.Pool
	clientFactory proxmox.ClientFactory
	wsHub         *monitor.WSHub
	logRepo       repository.MigrationLogRepository

	mu        sync.Mutex
	cancels   map[uuid.UUID]context.CancelFunc
	// activeVMs tracks VMIDs that are currently being migrated to prevent parallel migrations
	// of the same VM (which would cause VMID conflicts in Proxmox clusters).
	activeVMs map[int]uuid.UUID
}

func NewService(
	migrationRepo repository.MigrationRepository,
	nodeRepo repository.NodeRepository,
	encryptor *crypto.Encryptor,
	sshPool *ssh.Pool,
	clientFactory proxmox.ClientFactory,
	wsHub *monitor.WSHub,
	logRepo repository.MigrationLogRepository,
) *Service {
	return &Service{
		migrationRepo: migrationRepo,
		nodeRepo:      nodeRepo,
		encryptor:     encryptor,
		sshPool:       sshPool,
		clientFactory: clientFactory,
		wsHub:         wsHub,
		logRepo:       logRepo,
		cancels:       make(map[uuid.UUID]context.CancelFunc),
		activeVMs:     make(map[int]uuid.UUID),
	}
}

// RecoverOrphanedMigrations marks all migrations in non-terminal states as failed.
// This should be called on server startup to clean up migrations that were interrupted
// by a server restart.
func (s *Service) RecoverOrphanedMigrations(ctx context.Context) error {
	nonTerminalStatuses := []string{"pending", "preparing", "backing_up", "transferring", "restoring", "cleaning_up"}
	migrations, err := s.migrationRepo.ListByStatus(ctx, nonTerminalStatuses)
	if err != nil {
		return fmt.Errorf("list orphaned migrations: %w", err)
	}

	for _, m := range migrations {
		slog.Warn("recovering orphaned migration",
			slog.String("id", m.ID.String()),
			slog.String("status", string(m.Status)),
		)
		oldStatus := m.Status
		m.Status = "failed"
		m.ErrorMessage = fmt.Sprintf("Migration abgebrochen: Server-Neustart während Status '%s'", oldStatus)
		if err := s.migrationRepo.Update(ctx, &m); err != nil {
			slog.Error("failed to recover migration", slog.String("id", m.ID.String()), slog.Any("error", err))
		}
	}

	if len(migrations) > 0 {
		slog.Info("recovered orphaned migrations", slog.Int("count", len(migrations)))
	}

	// Clear any stale VMID locks from a previous server instance
	s.mu.Lock()
	s.activeVMs = make(map[int]uuid.UUID)
	s.cancels = make(map[uuid.UUID]context.CancelFunc)
	s.mu.Unlock()

	return nil
}

func (s *Service) StartMigration(ctx context.Context, req model.StartMigrationRequest, userID *uuid.UUID) (*model.VMMigration, error) {
	if req.SourceNodeID == req.TargetNodeID {
		return nil, fmt.Errorf("Quell- und Ziel-Node müssen sich unterscheiden")
	}
	if req.VMID <= 0 {
		return nil, fmt.Errorf("ungültige VMID: %d", req.VMID)
	}
	if req.TargetStorage == "" {
		return nil, fmt.Errorf("target_storage ist erforderlich")
	}
	if req.Mode == "" {
		req.Mode = model.MigrationModeSnapshot
	}
	if !req.Mode.IsValid() {
		return nil, fmt.Errorf("ungültiger Migrationsmodus: %s (erlaubt: stop, snapshot, suspend)", req.Mode)
	}

	// VMID-Lock: Prevent parallel migrations of the same VM.
	// We reserve the VMID slot atomically here. If validation fails later, we release it.
	s.mu.Lock()
	if existingMigID, active := s.activeVMs[req.VMID]; active {
		s.mu.Unlock()
		return nil, fmt.Errorf("VM %d wird bereits migriert (Migration %s). Parallele Migrationen derselben VM sind nicht erlaubt", req.VMID, existingMigID)
	}
	// Reserve VMID with a placeholder UUID (will be updated with real migration ID later)
	placeholderID := uuid.New()
	s.activeVMs[req.VMID] = placeholderID
	s.mu.Unlock()

	// Release VMID lock if validation fails (deferred cleanup)
	vmidReserved := true
	defer func() {
		if vmidReserved {
			s.mu.Lock()
			// Only delete if still our placeholder (not replaced by real migration)
			if s.activeVMs[req.VMID] == placeholderID {
				delete(s.activeVMs, req.VMID)
			}
			s.mu.Unlock()
		}
	}()

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
		return nil, fmt.Errorf("VM %d nicht auf Node %s gefunden", req.VMID, sourceNode.Name)
	}

	// LXC containers do not support Proxmox suspend mode — block early with clear error
	if vmInfo.Type == "lxc" && req.Mode == model.MigrationModeSuspend {
		return nil, fmt.Errorf("LXC-Container unterstützen keinen Suspend-Modus. Bitte 'stop' oder 'snapshot' verwenden")
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
		CleanupSource:        req.CleanupSource,
		CleanupTarget:        req.CleanupTarget,
		OverrideStorageCheck: req.OverrideStorageCheck,
		InitiatedBy:          userID,
	}

	if err := s.migrationRepo.Create(ctx, m); err != nil {
		return nil, fmt.Errorf("create migration record: %w", err)
	}

	// Start async execution with overall timeout to prevent stuck migrations
	migCtx, cancel := context.WithTimeout(context.Background(), maxMigrationDuration)
	s.mu.Lock()
	s.cancels[m.ID] = cancel
	s.activeVMs[m.VMID] = m.ID // Replace placeholder with real migration ID
	s.mu.Unlock()
	vmidReserved = false // Prevent defer from cleaning up — SafeGo will handle it

	util.SafeGo("migration-"+m.ID.String(), func() {
		defer func() {
			cancel() // Release timeout context resources
			s.mu.Lock()
			delete(s.activeVMs, m.VMID)
			delete(s.cancels, m.ID)
			s.mu.Unlock()
		}()
		s.executeMigration(migCtx, m.ID)
	})

	return m, nil
}

func (s *Service) CancelMigration(ctx context.Context, id uuid.UUID, userID *uuid.UUID, isAdmin bool) error {
	m, err := s.migrationRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if m.Status.IsTerminal() {
		return fmt.Errorf("migration already in terminal state: %s", m.Status)
	}

	// Ownership check: only the initiator or an admin may cancel a migration
	if !isAdmin {
		if userID == nil || m.InitiatedBy == nil || *userID != *m.InitiatedBy {
			return fmt.Errorf("nur der Ersteller oder ein Admin darf diese Migration abbrechen")
		}
	}

	s.mu.Lock()
	cancel, ok := s.cancels[id]
	s.mu.Unlock()

	if !ok {
		return fmt.Errorf("no cancel function found for migration %s", id)
	}

	cancel()
	return nil
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
	// NOTE: Cleanup of s.cancels and s.activeVMs is done in the SafeGo wrapper in StartMigration.

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
		switch {
		case ctx.Err() == context.Canceled:
			m.Status = model.MigrationStatusCancelled
			m.ErrorMessage = "Migration vom Benutzer abgebrochen"
		case ctx.Err() == context.DeadlineExceeded:
			m.Status = model.MigrationStatusFailed
			m.ErrorMessage = fmt.Sprintf("Migration Timeout: Maximale Laufzeit von %s überschritten in Phase '%s'. "+
				"Die Migration wurde abgebrochen. Bitte Netzwerk-Geschwindigkeit und Speicherplatz prüfen.", maxMigrationDuration, phase)
		default:
			m.Status = model.MigrationStatusFailed
			m.ErrorMessage = fmt.Sprintf("%s: %v", phase, err)
		}
		completedAt := time.Now()
		m.CompletedAt = &completedAt
		_ = s.migrationRepo.Update(context.Background(), m)
		s.broadcastProgress(m, nil)

		// If we stopped/suspended the VM and migration failed, restore it
		if vmWasStopped {
			switch m.Mode {
			case model.MigrationModeStop:
				s.tryRestartSourceVM(context.Background(), m)
			case model.MigrationModeSuspend:
				// Resume the suspended VM
				srcNode, nodeErr := s.nodeRepo.GetByID(context.Background(), m.SourceNodeID)
				if nodeErr == nil {
					srcClient, clientErr := s.clientFactory.CreateClient(srcNode)
					if clientErr == nil {
						pveNode, pveErr := s.resolvePVENodeName(context.Background(), srcClient, srcNode)
						if pveErr == nil {
							_, _ = srcClient.ResumeVM(context.Background(), pveNode, m.VMID)
							slog.Info("migration: resumed suspended VM after failure", slog.Int("vmid", m.VMID))
						}
					}
				}
			}
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
	defer s.sshPool.Return(sourceNode.ID.String(), srcSSHClient)
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
	defer s.sshPool.Return(targetNode.ID.String(), tgtSSHClient)
	s.broadcastLog(m.ID, "✓ SSH zu Target verbunden")

	// 1b. Validate SSH credentials format (prevent command injection via DB-stored values)
	tgtHost := targetNode.Hostname
	tgtPort := targetNode.SSHPort
	if tgtPort == 0 {
		tgtPort = 22
	}
	tgtUser := targetNode.SSHUser
	if tgtUser == "" {
		tgtUser = "root"
	}
	if !validHostname.MatchString(tgtHost) {
		handleError("preflight", fmt.Errorf("ungültiger Hostname für Target-Node: %q. Hostname darf nur Buchstaben, Zahlen, Punkte und Bindestriche enthalten", tgtHost))
		return
	}
	if !validSSHUser.MatchString(tgtUser) {
		handleError("preflight", fmt.Errorf("ungültiger SSH-Benutzername für Target-Node: %q. Erlaubt: Kleinbuchstaben, Zahlen, Unterstrich, Bindestrich", tgtUser))
		return
	}
	if !validStorageName.MatchString(m.TargetStorage) {
		handleError("preflight", fmt.Errorf("ungültiger Storage-Name: %q. Erlaubt: Buchstaben, Zahlen, Unterstrich, Bindestrich", m.TargetStorage))
		return
	}

	// 1c. Set up known_hosts for secure node-to-node SSH/SCP.
	// If we have the target's SSH host key, write it to a temp file on source for StrictHostKeyChecking=yes.
	// Otherwise, fall back to accept-new (TOFU) which is still better than =no.
	hostKeyOpts := "-o StrictHostKeyChecking=accept-new"
	knownHostsCleanup := ""
	if targetNode.SSHHostKey != "" {
		knownHostsFile := fmt.Sprintf("/tmp/.prometheus-known-hosts-%s", m.ID.String())
		// Write the target host key to a temporary known_hosts file on the source node.
		// Format: [hostname]:port ssh-rsa AAAA... (or just hostname if port 22)
		hostEntry := tgtHost
		if tgtPort != 22 {
			hostEntry = fmt.Sprintf("[%s]:%d", tgtHost, tgtPort)
		}
		knownHostsContent := fmt.Sprintf("%s %s\n", hostEntry, strings.TrimSpace(targetNode.SSHHostKey))
		writeResult, writeErr := srcSSHClient.RunCommand(ctx, fmt.Sprintf("printf '%%s' %s > %s && chmod 600 %s",
			shellQuote(knownHostsContent), shellQuote(knownHostsFile), shellQuote(knownHostsFile)))
		if writeErr == nil && writeResult != nil && writeResult.ExitCode == 0 {
			hostKeyOpts = fmt.Sprintf("-o StrictHostKeyChecking=yes -o UserKnownHostsFile=%s", shellQuote(knownHostsFile))
			knownHostsCleanup = knownHostsFile
			s.broadcastLog(m.ID, "✓ SSH-Hostkey des Targets verifiziert (StrictHostKeyChecking=yes)")
		} else {
			slog.Warn("migration: could not write known_hosts file, falling back to accept-new",
				slog.Any("error", writeErr))
			s.broadcastLog(m.ID, "⚠ Konnte known_hosts nicht schreiben, verwende accept-new")
		}
	} else {
		s.broadcastLog(m.ID, "⚠ Kein SSH-Hostkey für Target gespeichert, verwende accept-new (TOFU)")
	}
	// Clean up known_hosts file when done (deferred)
	if knownHostsCleanup != "" {
		defer func() {
			_, _ = srcSSHClient.RunCommand(context.Background(), "rm -f "+shellQuote(knownHostsCleanup))
		}()
	}

	// 1d. Check node-to-node SSH connectivity (required for SCP transfer)
	s.broadcastLog(m.ID, "Prüfe SSH-Verbindung zwischen Source und Target (für SCP-Transfer)...")
	sshCheckCmd := fmt.Sprintf("ssh %s -o ConnectTimeout=10 -o BatchMode=yes -p %d %s@%s echo ok 2>&1",
		hostKeyOpts, tgtPort, tgtUser, tgtHost)
	sshCheckResult, sshCheckErr := srcSSHClient.RunCommand(ctx, sshCheckCmd)
	if sshCheckErr != nil || sshCheckResult == nil || !strings.Contains(sshCheckResult.Stdout, "ok") {
		errDetail := "unbekannt"
		if sshCheckErr != nil {
			errDetail = sshCheckErr.Error()
		} else if sshCheckResult != nil {
			errDetail = sshCheckResult.Stdout + sshCheckResult.Stderr
		}
		handleError("preflight", fmt.Errorf(
			"SSH von Source (%s) zu Target (%s) fehlgeschlagen. SCP-Transfer benötigt SSH-Konnektivität zwischen den Nodes. "+
				"Bitte SSH-Key des Source-Nodes auf dem Target autorisieren (z.B. ssh-copy-id). Detail: %s",
			sourceNode.Name, targetNode.Name, errDetail))
		return
	}
	s.broadcastLog(m.ID, "✓ SSH zwischen Source und Target funktioniert")

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

	// Check VM configuration for migration compatibility
	warnings, blocker := s.checkVMCompatibility(ctx, srcClient, srcPVENode, m.VMID, m.VMType, m.Mode, srcSSHClient, vmInfo.Status)
	for _, w := range warnings {
		s.broadcastLog(m.ID, fmt.Sprintf("⚠ %s", w))
	}
	if blocker != nil {
		handleError("preflight", blocker)
		return
	}

	// Use actual disk usage (not allocated size) for space estimates.
	// vzdump with zstd compresses to ~50% of used data.
	// Prefer config-based disk size calculation (sums all disk volumes) over the reported Disk field,
	// which is 0 for stopped VMs.
	vmDiskUsed := vmInfo.Disk
	vmConfig, configErr := srcClient.GetVMConfig(ctx, srcPVENode, m.VMID, m.VMType)
	if configErr == nil {
		configDiskSize := parseVMDiskSizes(vmConfig)
		if configDiskSize > 0 {
			vmDiskUsed = configDiskSize
			s.broadcastLog(m.ID, fmt.Sprintf("  Disk-Größe aus VM-Config: %s (Summe aller Volumes)", formatBytesLog(configDiskSize)))
		}
	}
	if vmDiskUsed <= 0 {
		vmDiskUsed = vmInfo.MaxDisk
	}
	const maxEstimatedBackupSize = 35 * 1024 * 1024 * 1024 // 35 GB cap — vzdump/zstd compresses heavily
	estimatedBackupSize := int64(float64(vmDiskUsed) * 0.85)
	if estimatedBackupSize > maxEstimatedBackupSize {
		estimatedBackupSize = maxEstimatedBackupSize
	}
	s.broadcastLog(m.ID, fmt.Sprintf("✓ VM %d (%s) gefunden - Disk: %s belegt / %s alloziert, Status: %s",
		vmInfo.VMID, vmInfo.Name, formatBytesLog(vmDiskUsed), formatBytesLog(vmInfo.MaxDisk), vmInfo.Status))
	s.broadcastLog(m.ID, fmt.Sprintf("  Geschätzte Backup-Größe (zstd, max 35 GB): ~%s", formatBytesLog(estimatedBackupSize)))

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
			if m.OverrideStorageCheck {
				s.broadcastLog(m.ID, fmt.Sprintf("⚠ Wenig Speicherplatz auf '%s': benötigt ~%s, verfügbar %s — Override aktiv, fahre fort",
					m.TargetStorage, formatBytesLog(vmDiskUsed), formatBytesLog(targetStorageInfo.Available)))
			} else {
				handleError("preflight", fmt.Errorf("nicht genügend Speicherplatz auf '%s': benötigt ~%s, verfügbar %s",
					m.TargetStorage, formatBytesLog(vmDiskUsed), formatBytesLog(targetStorageInfo.Available)))
				return
			}
		} else {
			s.broadcastLog(m.ID, fmt.Sprintf("✓ Genügend Speicherplatz auf Ziel: benötigt ~%s, verfügbar %s",
				formatBytesLog(vmDiskUsed), formatBytesLog(targetStorageInfo.Available)))
		}
	}

	// 4. Find a vzdump-capable storage on source with enough real disk space
	s.broadcastLog(m.ID, "Suche Vzdump-Storage mit genug Speicherplatz auf Source...")
	vzdumpStorage, vzdumpFreeSpace := s.findVzdumpStorageWithDiskCheck(ctx, srcClient, srcPVENode, srcSSHClient, estimatedBackupSize, m.ID)
	if vzdumpStorage == "" {
		handleError("preflight", fmt.Errorf("kein Vzdump-Storage auf Source-Node mit genuegend Speicherplatz gefunden. "+
			"Geschaetzt ~%s benoetigt. Bitte Speicherplatz freigeben oder einen zusaetzlichen Backup-Storage einrichten",
			formatBytesLog(estimatedBackupSize)))
		return
	}
	s.broadcastLog(m.ID, fmt.Sprintf("✓ Vzdump-Storage: %s (%s frei, benötigt ~%s)",
		vzdumpStorage, formatBytesLog(vzdumpFreeSpace), formatBytesLog(estimatedBackupSize)))

	// 6. Check target dump directory free space (for temporary vzdump file during transfer)
	// This is a warning only - phase 3 will dynamically find a storage with enough space.
	// Note: estimatedBackupSize is a rough upper bound (60% of allocated disk); actual backups
	// are typically much smaller with zstd compression.
	s.broadcastLog(m.ID, "Prüfe freien Speicherplatz auf Target für Backup-Datei...")
	tgtDfResult, err := tgtSSHClient.RunCommand(ctx, "df -B1 /var/lib/vz/dump/ 2>/dev/null | tail -1 | awk '{print $4}'")
	if err == nil && tgtDfResult.ExitCode == 0 {
		var tgtDumpFree int64
		fmt.Sscanf(strings.TrimSpace(tgtDfResult.Stdout), "%d", &tgtDumpFree)
		if tgtDumpFree > 0 && estimatedBackupSize > 0 && tgtDumpFree < estimatedBackupSize {
			// Not a blocker - phase 3 will try to find alternative storage with enough space
			s.broadcastLog(m.ID, fmt.Sprintf("⚠ Wenig Platz auf Target /var/lib/vz/dump/: ~%s benötigt, %s verfügbar. Suche alternative Storages...",
				formatBytesLog(estimatedBackupSize), formatBytesLog(tgtDumpFree)))
			// Check if any vzdump storage on target has enough space
			altStorage := s.findVzdumpStorage(ctx, tgtClient, tgtPVENode, tgtSSHClient, estimatedBackupSize)
			if altStorage == "" {
				handleError("preflight", fmt.Errorf("kein Storage auf Target mit genügend Platz für Backup (~%s) gefunden",
					formatBytesLog(estimatedBackupSize)))
				return
			}
			s.broadcastLog(m.ID, fmt.Sprintf("✓ Alternativer Storage '%s' auf Target hat genug Platz", altStorage))
		} else if tgtDumpFree > 0 {
			s.broadcastLog(m.ID, fmt.Sprintf("✓ Target-Dump-Speicher: %s frei (benötigt ~%s)",
				formatBytesLog(tgtDumpFree), formatBytesLog(estimatedBackupSize)))
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
		if err := s.waitForTask(ctx, srcClient, srcPVENode, upid, 300*time.Second); err != nil {
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
		if err := s.waitForTask(ctx, srcClient, srcPVENode, upid, 180*time.Second); err != nil {
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

	// Re-check vzdump storage free space right before backup (disk may have filled up since pre-flight)
	currentVzdumpFree := s.getDiskFree(ctx, srcSSHClient, vzdumpStorage)
	if currentVzdumpFree > 0 && estimatedBackupSize > 0 && currentVzdumpFree < estimatedBackupSize {
		handleError("backing_up", fmt.Errorf("Vzdump-Storage '%s' hat nicht mehr genug Platz: benötigt ~%s, verfügbar %s. "+
			"Speicherplatz wurde seit Pre-Flight-Check belegt",
			vzdumpStorage, formatBytesLog(estimatedBackupSize), formatBytesLog(currentVzdumpFree)))
		return
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
	sizeResult, err := srcSSHClient.RunCommand(ctx, "stat -c %s "+shellQuote(vzdumpPath))
	if err == nil && sizeResult.ExitCode == 0 {
		var fileSize int64
		fmt.Sscanf(strings.TrimSpace(sizeResult.Stdout), "%d", &fileSize)
		m.VzdumpFileSize = &fileSize
		s.broadcastLog(m.ID, fmt.Sprintf("Backup-Größe: %s", formatBytesLog(fileSize)))
	}

	s.updatePhase(ctx, m, model.MigrationStatusBackingUp, "Backup abgeschlossen", 40)

	// PHASE 3: TRANSFERRING (40-80%)
	s.updatePhase(ctx, m, model.MigrationStatusTransferring, "Datei wird übertragen...", 41)

	// Determine target dump path - find a directory with enough space
	vzdumpFilename := vzdumpPath
	if idx := strings.LastIndex(vzdumpPath, "/"); idx >= 0 {
		vzdumpFilename = vzdumpPath[idx+1:]
	}

	totalSize := int64(0)
	if m.VzdumpFileSize != nil {
		totalSize = *m.VzdumpFileSize
	}

	// Find a vzdump-capable storage on target with enough space for the actual backup
	targetDumpDir := "/var/lib/vz/dump" // default
	tgtVzdumpStorage := s.findVzdumpStorage(ctx, tgtClient, tgtPVENode, tgtSSHClient, totalSize)
	if tgtVzdumpStorage != "" && validStorageName.MatchString(tgtVzdumpStorage) {
		// Resolve the actual filesystem path for this storage (storageName validated above)
		pathResult, pathErr := tgtSSHClient.RunCommand(ctx, fmt.Sprintf("pvesm path %s:backup/test.vma 2>/dev/null | sed 's|/test.vma$||'", tgtVzdumpStorage))
		if pathErr == nil && pathResult.ExitCode == 0 && strings.TrimSpace(pathResult.Stdout) != "" {
			resolvedPath := strings.TrimSpace(pathResult.Stdout)
			// Validate resolved path has no shell metacharacters before using in mkdir
			if validVzdumpPath.MatchString(resolvedPath) {
				safePath := shellQuote(resolvedPath)
				mkdirResult, _ := tgtSSHClient.RunCommand(ctx, fmt.Sprintf("mkdir -p %s && test -w %s && echo ok", safePath, safePath))
				if mkdirResult != nil && strings.TrimSpace(mkdirResult.Stdout) == "ok" {
					targetDumpDir = resolvedPath
				}
			}
		}
		s.broadcastLog(m.ID, fmt.Sprintf("Verwende Target-Dump-Verzeichnis: %s (Storage: %s)", targetDumpDir, tgtVzdumpStorage))
	} else {
		s.broadcastLog(m.ID, "Verwende Standard-Dump-Verzeichnis: /var/lib/vz/dump")
	}
	targetVzdumpPath := targetDumpDir + "/" + vzdumpFilename

	// Ensure target directory exists
	_, _ = tgtSSHClient.RunCommand(ctx, "mkdir -p "+shellQuote(targetDumpDir))

	// Transfer via scp directly between nodes (bypasses Docker networking limitations)
	// tgtHost, tgtPort, tgtUser are already resolved in the pre-flight phase above.
	// SECURITY: Use shellQuote for paths to prevent injection via crafted filenames.
	// tgtUser and tgtHost are validated against strict regexes in pre-flight phase.
	// hostKeyOpts uses StrictHostKeyChecking=yes with known_hosts when available.
	scpCmd := fmt.Sprintf("scp %s -o ConnectTimeout=30 -o ServerAliveInterval=15 -o ServerAliveCountMax=4 -P %d %s %s@%s:%s",
		hostKeyOpts, tgtPort, shellQuote(vzdumpPath), tgtUser, tgtHost, shellQuote(targetVzdumpPath))

	s.broadcastLog(m.ID, fmt.Sprintf("Starte Transfer: %s → %s (%s)", sourceNode.Name, targetNode.Name, formatBytesLog(totalSize)))
	s.broadcastLog(m.ID, "Transfer via scp direkt zwischen Nodes...")

	// Start scp in background on source node
	// We use nohup + background so we can poll progress from the target
	bgScpCmd := fmt.Sprintf("nohup %s > /tmp/scp-mig-%s.log 2>&1 & echo $!", scpCmd, m.ID.String())
	pidResult, err := srcSSHClient.RunCommand(ctx, bgScpCmd)
	if err != nil {
		handleError("transferring", fmt.Errorf("scp starten fehlgeschlagen: %w", err))
		return
	}
	if pidResult.ExitCode != 0 {
		handleError("transferring", fmt.Errorf("scp starten fehlgeschlagen (exit %d): %s", pidResult.ExitCode, pidResult.Stderr))
		return
	}
	scpPID := strings.TrimSpace(pidResult.Stdout)
	// Validate PID is numeric to prevent injection
	if _, err := strconv.Atoi(scpPID); err != nil {
		handleError("transferring", fmt.Errorf("ungültige PID von scp: %q", scpPID))
		return
	}
	s.broadcastLog(m.ID, fmt.Sprintf("scp gestartet (PID: %s)", scpPID))

	// Poll progress by checking file size on target
	transferStart := time.Now()
	lastLogTime := time.Now()
	pollTicker := time.NewTicker(5 * time.Second) // 5s intervals for high-latency networks
	defer pollTicker.Stop()

	consecutivePollErrors := 0
	const maxPollErrors = 10 // allow transient SSH failures during polling

	transferDone := false
	for !transferDone {
		select {
		case <-ctx.Done():
			// Kill scp on cancel
			_, _ = srcSSHClient.RunCommand(context.Background(), fmt.Sprintf("kill %s 2>/dev/null", scpPID))
			_, _ = tgtSSHClient.RunCommand(context.Background(), "rm -f "+shellQuote(targetVzdumpPath))
			if ctx.Err() == context.Canceled {
				m.Status = model.MigrationStatusCancelled
				m.ErrorMessage = "Migration vom Benutzer abgebrochen"
			} else {
				m.Status = model.MigrationStatusFailed
				m.ErrorMessage = fmt.Sprintf("context error: %v", ctx.Err())
			}
			completedAt := time.Now()
			m.CompletedAt = &completedAt
			_ = s.migrationRepo.Update(context.Background(), m)
			s.broadcastProgress(m, nil)
			// Restore VM if it was stopped/suspended
			if vmWasStopped {
				switch m.Mode {
				case model.MigrationModeStop:
					s.tryRestartSourceVM(context.Background(), m)
				case model.MigrationModeSuspend:
					_, _ = srcClient.ResumeVM(context.Background(), srcPVENode, m.VMID)
					slog.Info("migration: resumed suspended VM after cancellation", slog.Int("vmid", m.VMID))
				}
			}
			return
		case <-pollTicker.C:
			// Check if scp is still running (tolerate SSH errors during polling)
			checkResult, checkErr := srcSSHClient.RunCommand(ctx, fmt.Sprintf("kill -0 %s 2>/dev/null; echo $?", scpPID))
			if checkErr != nil {
				consecutivePollErrors++
				if consecutivePollErrors >= maxPollErrors {
					handleError("transferring", fmt.Errorf("SSH-Verbindung zu Source nach %d Versuchen verloren: %w", maxPollErrors, checkErr))
					return
				}
				s.broadcastLog(m.ID, fmt.Sprintf("⚠ SSH-Poll fehlgeschlagen (%d/%d), warte...", consecutivePollErrors, maxPollErrors))
				continue
			}
			consecutivePollErrors = 0
			scpRunning := checkResult != nil && strings.TrimSpace(checkResult.Stdout) == "0"

			// Get current file size on target (tolerate errors)
			statResult, _ := tgtSSHClient.RunCommand(ctx, "stat -c %s "+shellQuote(targetVzdumpPath)+" 2>/dev/null")
			var currentSize int64
			if statResult != nil && statResult.ExitCode == 0 {
				fmt.Sscanf(strings.TrimSpace(statResult.Stdout), "%d", &currentSize)
			}

			m.TransferBytesSent = currentSize
			if totalSize > 0 {
				transferProgress := int(float64(currentSize) / float64(totalSize) * 39)
				m.Progress = 41 + transferProgress
				if m.Progress > 79 {
					m.Progress = 79
				}
			}

			elapsed := time.Since(transferStart).Seconds()
			if elapsed > 0 {
				m.TransferSpeedBps = int64(float64(currentSize) / elapsed)
			}
			s.broadcastProgress(m, nil)

			// Log progress every 15 seconds (less spam for slow networks)
			if time.Since(lastLogTime) > 15*time.Second {
				lastLogTime = time.Now()
				pct := float64(0)
				if totalSize > 0 {
					pct = float64(currentSize) / float64(totalSize) * 100
				}
				s.broadcastLog(m.ID, fmt.Sprintf("Transfer: %s / %s (%.1f%%) - %s/s",
					formatBytesLog(currentSize), formatBytesLog(totalSize), pct, formatBytesLog(m.TransferSpeedBps)))
			}

			if !scpRunning {
				// scp finished - check if it was successful
				logResult, _ := srcSSHClient.RunCommand(ctx, fmt.Sprintf("cat /tmp/scp-mig-%s.log 2>/dev/null", m.ID.String()))
				// Cleanup log file
				_, _ = srcSSHClient.RunCommand(ctx, fmt.Sprintf("rm -f /tmp/scp-mig-%s.log", m.ID.String()))

				// Verify file size matches (with retries for slow filesystems)
				var finalSize int64
				for retry := 0; retry < 3; retry++ {
					finalStatResult, _ := tgtSSHClient.RunCommand(ctx, "stat -c %s "+shellQuote(targetVzdumpPath)+" 2>/dev/null")
					if finalStatResult != nil && finalStatResult.ExitCode == 0 {
						fmt.Sscanf(strings.TrimSpace(finalStatResult.Stdout), "%d", &finalSize)
						if finalSize >= totalSize || totalSize == 0 {
							break
						}
					}
					if retry < 2 {
						time.Sleep(3 * time.Second) // wait for filesystem sync
					}
				}

				if totalSize > 0 && finalSize < totalSize {
					scpLog := ""
					if logResult != nil {
						scpLog = logResult.Stdout
					}
					_, _ = tgtSSHClient.RunCommand(ctx, "rm -f "+shellQuote(targetVzdumpPath))
					handleError("transferring", fmt.Errorf("scp fehlgeschlagen (nur %s von %s übertragen). Log: %s",
						formatBytesLog(finalSize), formatBytesLog(totalSize), scpLog))
					return
				}

				m.TransferBytesSent = finalSize
				transferDone = true
			}
		}
	}

	// Verify file integrity via checksum
	s.broadcastLog(m.ID, "Prüfe Datei-Integrität via SHA256...")
	srcShaResult, srcShaErr := srcSSHClient.RunCommand(ctx, "sha256sum "+shellQuote(vzdumpPath)+" | awk '{print $1}'")
	tgtShaResult, tgtShaErr := tgtSSHClient.RunCommand(ctx, "sha256sum "+shellQuote(targetVzdumpPath)+" | awk '{print $1}'")

	if srcShaErr == nil && tgtShaErr == nil && srcShaResult != nil && tgtShaResult != nil {
		srcHash := strings.TrimSpace(srcShaResult.Stdout)
		tgtHash := strings.TrimSpace(tgtShaResult.Stdout)
		if srcHash != "" && tgtHash != "" && srcHash != tgtHash {
			// Checksum mismatch — file corrupted during transfer
			_, _ = tgtSSHClient.RunCommand(ctx, "rm -f "+shellQuote(targetVzdumpPath))
			handleError("transferring", fmt.Errorf("Prüfsummenfehler! Source: %s, Target: %s. Datei wurde während des Transfers beschädigt", srcHash[:16]+"...", tgtHash[:16]+"..."))
			return
		}
		s.broadcastLog(m.ID, fmt.Sprintf("✓ Prüfsumme stimmt überein (SHA256: %s...)", srcHash[:16]))
	} else {
		// Checksum couldn't be verified — warn but continue (sha256sum might not be available)
		s.broadcastLog(m.ID, "⚠ Prüfsummenverifikation nicht möglich — sha256sum nicht verfügbar")
	}

	s.broadcastLog(m.ID, fmt.Sprintf("✓ Transfer abgeschlossen: %s übertragen", formatBytesLog(m.TransferBytesSent)))
	s.updatePhase(ctx, m, model.MigrationStatusTransferring, "Transfer abgeschlossen", 80)

	// PHASE 4: RESTORING (80-95%)
	s.updatePhase(ctx, m, model.MigrationStatusRestoring, "VM wird auf Ziel wiederhergestellt...", 81)

	restoreVMID := m.VMID
	if m.NewVMID != nil {
		restoreVMID = *m.NewVMID
	}

	// In a Proxmox cluster, VMIDs are unique across ALL nodes.
	// If we restore with the same VMID, we must first unregister the VM from the source.
	// We save the source config so we can re-register on failure.
	var sourceConfigBackup string
	if restoreVMID == m.VMID {
		s.broadcastLog(m.ID, fmt.Sprintf("Entregistriere VM %d von Source-Node (Cluster-VMID-Konflikt)...", m.VMID))

		// Save the config for rollback
		cfgDir := configDir(m.VMType)
		configResult, _ := srcSSHClient.RunCommand(ctx, fmt.Sprintf("cat /etc/pve/nodes/%s/%s/%d.conf 2>/dev/null", srcPVENode, cfgDir, m.VMID))
		if configResult != nil && configResult.ExitCode == 0 {
			sourceConfigBackup = configResult.Stdout
		}

		// Remove VM registration from source (config only, not disks)
		_, _ = srcSSHClient.RunCommand(ctx, fmt.Sprintf("mv /etc/pve/nodes/%s/%s/%d.conf /etc/pve/nodes/%s/%s/%d.conf.mig-backup 2>/dev/null",
			srcPVENode, cfgDir, m.VMID, srcPVENode, cfgDir, m.VMID))
		time.Sleep(2 * time.Second)
		s.broadcastLog(m.ID, fmt.Sprintf("✓ VM %d von Source entregistriert (Config gesichert)", m.VMID))
	}

	// Also check for leftover config on target
	tgtCfgDir := configDir(m.VMType)
	checkResult, _ := tgtSSHClient.RunCommand(ctx, fmt.Sprintf("test -f /etc/pve/nodes/%s/%s/%d.conf && echo exists || echo no", tgtPVENode, tgtCfgDir, restoreVMID))
	if checkResult != nil && strings.Contains(checkResult.Stdout, "exists") {
		s.broadcastLog(m.ID, fmt.Sprintf("⚠ Alte VM %d Config auf Target gefunden - entferne...", restoreVMID))
		_, _ = tgtSSHClient.RunCommand(ctx, fmt.Sprintf("rm -f /etc/pve/nodes/%s/%s/%d.conf", tgtPVENode, tgtCfgDir, restoreVMID))
		time.Sleep(1 * time.Second)
	}

	// Convert filesystem path to Proxmox volume ID format.
	// Only use tgtVzdumpStorage if the file was actually resolved to that storage's path.
	// If targetDumpDir is still the default (/var/lib/vz/dump), the storage path resolution
	// failed (e.g. PBS storage) and the file ended up in local:backup/ instead.
	restoreStorageName := "local"
	if tgtVzdumpStorage != "" && targetDumpDir != "/var/lib/vz/dump" {
		restoreStorageName = tgtVzdumpStorage
	}
	restoreArchive := restoreStorageName + ":backup/" + vzdumpFilename
	s.broadcastLog(m.ID, fmt.Sprintf("Starte Restore auf %s (VMID: %d, Storage: %s, Archive: %s)...",
		targetNode.Name, restoreVMID, m.TargetStorage, restoreArchive))

	restoreUPID, err := tgtClient.RestoreVM(ctx, tgtPVENode, restoreArchive, m.TargetStorage, restoreVMID, m.VMType)
	if err != nil {
		// Restore source config on failure
		s.rollbackSourceConfig(ctx, srcSSHClient, srcPVENode, m.VMID, m.VMType, sourceConfigBackup)
		_, _ = tgtSSHClient.RunCommand(ctx, "rm -f "+shellQuote(targetVzdumpPath))
		handleError("restoring", fmt.Errorf("restore konnte nicht gestartet werden: %w", err))
		return
	}
	m.RestoreTaskUPID = &restoreUPID
	_ = s.migrationRepo.Update(ctx, m)
	s.broadcastLog(m.ID, fmt.Sprintf("Restore Task gestartet: %s", restoreUPID))

	if err := s.pollTaskWithProgress(ctx, tgtClient, tgtPVENode, restoreUPID, m, 81, 94); err != nil {
		taskLog := s.getTaskLogForError(ctx, tgtClient, tgtPVENode, restoreUPID)
		// Restore source config on failure
		s.rollbackSourceConfig(ctx, srcSSHClient, srcPVENode, m.VMID, m.VMType, sourceConfigBackup)
		_, _ = tgtSSHClient.RunCommand(ctx, "rm -f "+shellQuote(targetVzdumpPath))
		handleError("restoring", fmt.Errorf("restore fehlgeschlagen: %w\n\nTask-Log:\n%s", err, taskLog))
		return
	}

	// Restore succeeded - delete the backup config on source
	_, _ = srcSSHClient.RunCommand(ctx, fmt.Sprintf("rm -f /etc/pve/nodes/%s/%s/%d.conf.mig-backup", srcPVENode, configDir(m.VMType), m.VMID))
	s.broadcastLog(m.ID, "✓ VM erfolgreich wiederhergestellt")
	s.updatePhase(ctx, m, model.MigrationStatusRestoring, "Wiederherstellung abgeschlossen", 95)

	// PHASE 5: CLEANING UP (95-100%)
	s.updatePhase(ctx, m, model.MigrationStatusCleaningUp, "Aufräum-Arbeiten...", 96)

	// Delete vzdump on target
	if m.CleanupTarget {
		s.broadcastLog(m.ID, "Lösche Backup-Datei auf Target...")
		if cleanResult, cleanErr := tgtSSHClient.RunCommand(ctx, "rm -f "+shellQuote(targetVzdumpPath)); cleanErr != nil {
			s.broadcastLog(m.ID, fmt.Sprintf("⚠ Backup auf Target konnte nicht gelöscht werden: %v", cleanErr))
		} else if cleanResult != nil && cleanResult.ExitCode != 0 {
			s.broadcastLog(m.ID, fmt.Sprintf("⚠ Backup auf Target konnte nicht gelöscht werden (exit %d): %s", cleanResult.ExitCode, cleanResult.Stderr))
		} else {
			s.broadcastLog(m.ID, "✓ Backup auf Target gelöscht")
		}
	}

	// Delete vzdump on source
	if m.CleanupSource {
		s.broadcastLog(m.ID, "Lösche Backup-Datei auf Source...")
		if cleanResult, cleanErr := srcSSHClient.RunCommand(ctx, "rm -f "+shellQuote(vzdumpPath)); cleanErr != nil {
			s.broadcastLog(m.ID, fmt.Sprintf("⚠ Backup auf Source konnte nicht gelöscht werden: %v", cleanErr))
		} else if cleanResult != nil && cleanResult.ExitCode != 0 {
			s.broadcastLog(m.ID, fmt.Sprintf("⚠ Backup auf Source konnte nicht gelöscht werden (exit %d): %s", cleanResult.ExitCode, cleanResult.Stderr))
		} else {
			s.broadcastLog(m.ID, "✓ Backup auf Source gelöscht")
		}
	}

	// Also clean up the vzdump log/notes file (Proxmox creates these alongside the archive)
	if m.CleanupSource {
		notesPath := strings.TrimSuffix(vzdumpPath, ".zst") + ".log"
		_, _ = srcSSHClient.RunCommand(ctx, "rm -f "+shellQuote(notesPath)+" "+shellQuote(vzdumpPath+".notes"))
	}
	if m.CleanupTarget {
		notesPath := strings.TrimSuffix(targetVzdumpPath, ".zst") + ".log"
		_, _ = tgtSSHClient.RunCommand(ctx, "rm -f "+shellQuote(notesPath)+" "+shellQuote(targetVzdumpPath+".notes"))
	}

	// Resume source VM if suspended — but only if a different VMID was used on target.
	// If same VMID: the source config was unregistered (moved to target), so there's nothing to resume.
	if m.Mode == model.MigrationModeSuspend && m.NewVMID != nil && *m.NewVMID != m.VMID {
		s.broadcastLog(m.ID, "Setze Source-VM fort (neues VMID auf Target, Source bleibt bestehen)...")
		if _, err := srcClient.ResumeVM(ctx, srcPVENode, m.VMID); err != nil {
			slog.Warn("migration: konnte Source-VM nach Migration nicht fortsetzen",
				slog.Int("vmid", m.VMID), slog.Any("error", err))
			s.broadcastLog(m.ID, fmt.Sprintf("⚠ Source-VM konnte nicht fortgesetzt werden: %v", err))
		} else {
			s.broadcastLog(m.ID, "✓ Source-VM fortgesetzt")
		}
	} else if m.Mode == model.MigrationModeSuspend {
		s.broadcastLog(m.ID, "Source-VM nicht fortgesetzt (VMID wurde zum Target verschoben)")
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

	// Determine log level from content
	level := "info"
	if strings.Contains(line, "\u2713") {
		level = "success"
	} else if strings.Contains(line, "\u26a0") {
		level = "warn"
	} else if strings.Contains(line, "ERROR") || strings.Contains(line, "fehlgeschlagen") {
		level = "error"
	} else if strings.HasPrefix(line, "[PVE]") {
		level = "pve"
	}

	// Persist to DB (fire-and-forget, don't block migration)
	if s.logRepo != nil {
		go func() {
			_ = s.logRepo.Append(context.Background(), migrationID, line, level, "")
		}()
	}

	s.wsHub.BroadcastMessage(monitor.WSMessage{
		Type: "migration_log",
		Data: map[string]interface{}{
			"migration_id": migrationID.String(),
			"line":         line,
			"timestamp":    time.Now().Format("15:04:05"),
		},
	})
}

// GetLogs returns persisted migration logs for a given migration.
func (s *Service) GetLogs(ctx context.Context, migrationID uuid.UUID) ([]repository.MigrationLog, error) {
	if s.logRepo == nil {
		return nil, nil
	}
	return s.logRepo.ListByMigration(ctx, migrationID)
}

func (s *Service) waitForTask(ctx context.Context, client *proxmox.Client, node string, upid string, timeout time.Duration) error {
	deadline := time.After(timeout)
	ticker := time.NewTicker(5 * time.Second) // 5s for high-latency networks
	defer ticker.Stop()

	consecutiveErrors := 0
	const maxConsecutiveErrors = 6 // ~30s of failures before giving up

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("task timeout after %s", timeout)
		case <-ticker.C:
			status, err := client.GetTaskStatus(ctx, node, upid)
			if err != nil {
				consecutiveErrors++
				if consecutiveErrors >= maxConsecutiveErrors {
					return fmt.Errorf("task status unavailable after %d attempts: %w", maxConsecutiveErrors, err)
				}
				continue
			}
			consecutiveErrors = 0
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
	ticker := time.NewTicker(5 * time.Second) // 5s for high-latency networks
	defer ticker.Stop()

	progressRange := endProgress - startProgress
	pollCount := 0
	lastLogLine := 0 // Track which log lines we've already sent
	consecutiveErrors := 0
	const maxConsecutiveErrors = 10 // ~50s of failures before giving up

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Stream task log lines (tolerate errors)
			entries, err := client.GetTaskLog(ctx, node, upid, lastLogLine)
			if err == nil && len(entries) > 0 {
				for _, e := range entries {
					if e.LineNum >= lastLogLine && e.Text != "" {
						s.broadcastLog(m.ID, fmt.Sprintf("[PVE] %s", e.Text))
						lastLogLine = e.LineNum + 1
					}
				}
			}

			status, err := client.GetTaskStatus(ctx, node, upid)
			if err != nil {
				consecutiveErrors++
				if consecutiveErrors >= maxConsecutiveErrors {
					return fmt.Errorf("task status unavailable after %d attempts: %w", maxConsecutiveErrors, err)
				}
				continue
			}
			consecutiveErrors = 0
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

// findVzdumpStorageWithDiskCheck finds a backup-capable storage with enough real disk space.
// It queries the Proxmox API for storages, then verifies actual free space via SSH df.
// Returns the storage name and actual free space, or ("", 0) if none found.
func (s *Service) findVzdumpStorageWithDiskCheck(ctx context.Context, client *proxmox.Client, pveNode string, sshClient *ssh.Client, estimatedSize int64, migrationID uuid.UUID) (string, int64) {
	storages, err := client.GetStorage(ctx, pveNode)
	if err != nil {
		// Fallback: try "local" with df check
		freeSpace := s.getDiskFree(ctx, sshClient, "local")
		if freeSpace >= estimatedSize {
			return "local", freeSpace
		}
		return "", 0
	}

	// Collect all backup-capable storages
	type candidate struct {
		name      string
		apiAvail  int64
		diskFree  int64
	}
	var candidates []candidate

	for _, st := range storages {
		if strings.Contains(st.Content, "backup") || strings.Contains(st.Content, "vzdump") {
			diskFree := s.getDiskFree(ctx, sshClient, st.Storage)
			candidates = append(candidates, candidate{
				name:     st.Storage,
				apiAvail: st.Available,
				diskFree: diskFree,
			})
			s.broadcastLog(migrationID, fmt.Sprintf("  Storage '%s': API meldet %s frei, Filesystem: %s frei",
				st.Storage, formatBytesLog(st.Available), formatBytesLog(diskFree)))
		}
	}

	// Pick the first storage with enough real disk space (prefer largest)
	var best *candidate
	for i := range candidates {
		c := &candidates[i]
		if c.diskFree >= estimatedSize {
			if best == nil || c.diskFree > best.diskFree {
				best = c
			}
		}
	}

	if best != nil {
		slog.Info("migration: found vzdump storage with space",
			slog.String("storage", best.name),
			slog.String("disk_free", formatBytesLog(best.diskFree)),
			slog.String("estimated_backup", formatBytesLog(estimatedSize)))
		return best.name, best.diskFree
	}

	// No storage has enough space — log all candidates
	slog.Warn("migration: no vzdump storage with enough space",
		slog.Int("candidates", len(candidates)),
		slog.String("needed", formatBytesLog(estimatedSize)))
	return "", 0
}

// getDiskFree returns the actual free disk space in bytes for a Proxmox storage via SSH df.
func (s *Service) getDiskFree(ctx context.Context, sshClient *ssh.Client, storageName string) int64 {
	// SECURITY: Validate storage name before using in shell command to prevent injection.
	if !validStorageName.MatchString(storageName) {
		slog.Warn("migration: invalid storage name for disk check, skipping",
			slog.String("storage", storageName))
		return 0
	}
	// Try to resolve the storage path via pvesm (storageName is validated above)
	result, err := sshClient.RunCommand(ctx, fmt.Sprintf(
		"df -B1 $(pvesm path %s: 2>/dev/null || echo /var/lib/vz/dump) 2>/dev/null | tail -1 | awk '{print $4}'",
		storageName))
	if err != nil || result.ExitCode != 0 {
		// Fallback: check /var/lib/vz/dump/ directly
		result, err = sshClient.RunCommand(ctx, "df -B1 /var/lib/vz/dump/ 2>/dev/null | tail -1 | awk '{print $4}'")
		if err != nil || result.ExitCode != 0 {
			return 0
		}
	}
	var freeSpace int64
	fmt.Sscanf(strings.TrimSpace(result.Stdout), "%d", &freeSpace)
	return freeSpace
}

// findVzdumpStorage finds a backup-capable storage with enough space for the estimated backup size.
// Returns the storage name with the most available space that fits, or "" if none found.
func (s *Service) findVzdumpStorage(ctx context.Context, client *proxmox.Client, pveNode string, sshClient *ssh.Client, estimatedSize int64) string {
	storages, err := client.GetStorage(ctx, pveNode)
	if err != nil {
		return ""
	}

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

	// Pick the storage with the most available space that can hold the backup
	var best *candidate
	for i := range candidates {
		c := &candidates[i]
		// Skip storages that don't have enough space (if we know the estimated size)
		if estimatedSize > 0 && c.available > 0 && c.available < estimatedSize {
			continue
		}
		if best == nil || c.available > best.available {
			best = c
		}
	}

	if best != nil {
		return best.name
	}

	return ""
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
	// Retry log fetching a few times since the log may not be immediately complete
	var entries []proxmox.TaskLogEntry
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		entries, err = client.GetTaskLog(ctx, node, upid, 0)
		if err == nil && len(entries) > 0 {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return "", fmt.Errorf("task log abrufen: %w", err)
	}

	// Look for vzdump archive paths. Proxmox logs these in various formats:
	// "creating archive '/var/lib/vz/dump/vzdump-qemu-100-2024_01_15-10_30_00.vma.zst'"
	// "creating vzdump archive '/path/to/vzdump-lxc-101-...'"
	// "INFO: archive file size: 1.23GB"
	for _, e := range entries {
		text := e.Text
		if strings.Contains(text, "/vzdump-") {
			parts := strings.Fields(text)
			for _, p := range parts {
				if strings.Contains(p, "/vzdump-") {
					// Clean up path (remove quotes, trailing commas, colons)
					p = strings.Trim(p, "'\",:;")
					// Validate it looks like a real path AND contains no shell metacharacters
					// (this path will be used in SSH commands on Proxmox hosts)
					if strings.HasPrefix(p, "/") && (strings.Contains(p, ".vma") || strings.Contains(p, ".tar")) {
						if !validVzdumpPath.MatchString(p) {
							slog.Warn("migration: rejected vzdump path with unsafe characters",
								slog.String("path", p))
							continue
						}
						return p, nil
					}
				}
			}
		}
	}
	return "", fmt.Errorf("vzdump-Dateipfad nicht im Task-Log gefunden. "+
		"Möglicherweise wurde das Backup in einem unerwarteten Verzeichnis gespeichert. "+
		"Bitte die Proxmox-Task-Logs prüfen (UPID: %s)", upid)
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
		HostKey:    node.SSHHostKey,
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

	// Try matching by hostname
	for _, pn := range pveNodes {
		if strings.EqualFold(pn, node.Hostname) || strings.HasPrefix(strings.ToLower(node.Hostname), strings.ToLower(pn)) {
			slog.Info("migration: resolved PVE node by hostname match", slog.String("pve_node", pn), slog.String("hostname", node.Hostname))
			s.cachePVENodeName(ctx, node, pn)
			return pn, nil
		}
	}
	// If only one node in cluster, use it (safe assumption)
	if len(pveNodes) == 1 {
		slog.Info("migration: single-node cluster, using sole node", slog.String("pve_node", pveNodes[0]))
		s.cachePVENodeName(ctx, node, pveNodes[0])
		return pveNodes[0], nil
	}
	return "", fmt.Errorf("PVE node name for '%s' could not be resolved. Available PVE nodes: %v. Set the correct name in node metadata", node.Name, pveNodes)
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

// configDir returns the Proxmox config directory name for the given VM type.
// For "qemu" it returns "qemu-server", for "lxc" it returns "lxc".
func configDir(vmType string) string {
	if vmType == "lxc" {
		return "lxc"
	}
	return "qemu-server"
}

// rollbackSourceConfig restores the VM config on the source node if restore on target failed.
func (s *Service) rollbackSourceConfig(ctx context.Context, srcSSHClient *ssh.Client, srcPVENode string, vmid int, vmType string, configBackup string) {
	cfgDir := configDir(vmType)
	if configBackup == "" {
		// Try to restore from .mig-backup file
		_, _ = srcSSHClient.RunCommand(ctx, fmt.Sprintf(
			"mv /etc/pve/nodes/%s/%s/%d.conf.mig-backup /etc/pve/nodes/%s/%s/%d.conf 2>/dev/null",
			srcPVENode, cfgDir, vmid, srcPVENode, cfgDir, vmid))
		return
	}
	// Write back the saved config using CopyTo to avoid heredoc data corruption
	path := fmt.Sprintf("/etc/pve/nodes/%s/%s/%d.conf", srcPVENode, cfgDir, vmid)
	if err := srcSSHClient.CopyTo(ctx, []byte(configBackup), path); err != nil {
		slog.Warn("migration: rollback source config failed", slog.Any("error", err))
		return
	}
	// Remove backup
	_, _ = srcSSHClient.RunCommand(ctx, fmt.Sprintf("rm -f %s.mig-backup", path))
	slog.Info("migration: source config restored", slog.Int("vmid", vmid))
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

// checkVMCompatibility reads the VM config and checks for features that are incompatible
// with migration or require special handling (PCI passthrough, EFI disk, TPM, HA, snapshots,
// linked clones, cloud-init). Returns warnings (informational) and a blocker error (migration
// must be aborted).
func (s *Service) checkVMCompatibility(ctx context.Context, client *proxmox.Client, pveNode string, vmid int, vmType string, mode model.MigrationMode, sshClient *ssh.Client, vmStatus string) (warnings []string, blocker error) {
	config, err := client.GetVMConfig(ctx, pveNode, vmid, vmType)
	if err != nil {
		// Non-fatal: we can't check compatibility but shouldn't block migration
		slog.Warn("migration: could not read VM config for compatibility check", slog.Any("error", err))
		warnings = append(warnings, "VM-Konfiguration konnte nicht gelesen werden — Kompatibilitätsprüfung übersprungen")
		return warnings, nil
	}

	// 1. PCI Passthrough check (hostpci0, hostpci1, ...)
	for i := 0; i < 16; i++ {
		key := fmt.Sprintf("hostpci%d", i)
		if val, ok := config[key].(string); ok && val != "" {
			if mode == model.MigrationModeSnapshot {
				return warnings, fmt.Errorf("VM hat PCI-Passthrough konfiguriert (%s). Snapshot-Modus nicht möglich — bitte 'stop' verwenden", key)
			}
			warnings = append(warnings, fmt.Sprintf("VM hat PCI-Passthrough (%s: %s) — Gerät muss auf Ziel-Node verfügbar sein", key, val))
		}
	}

	// 2. EFI Disk
	if val, ok := config["efidisk0"].(string); ok && val != "" {
		warnings = append(warnings, "VM hat EFI-Disk — wird mit migriert")
	}

	// 3. TPM State
	if val, ok := config["tpmstate0"].(string); ok && val != "" {
		warnings = append(warnings, "VM hat TPM-State — Verschlüsselungskeys könnten nach Migration ungültig sein")
	}

	// 4. HA Configuration (check via SSH)
	if sshClient != nil {
		haResult, haErr := sshClient.RunCommand(ctx, fmt.Sprintf("ha-manager status 2>/dev/null | grep 'vm:%d\\|ct:%d'", vmid, vmid))
		if haErr == nil && haResult != nil && haResult.ExitCode == 0 && strings.TrimSpace(haResult.Stdout) != "" {
			return warnings, fmt.Errorf("VM ist in HA-Konfiguration — bitte HA zuerst deaktivieren (ha-manager remove vm:%d)", vmid)
		}
	}

	// 5. Snapshots
	snapshots, snapErr := client.ListSnapshots(ctx, pveNode, vmid, vmType)
	if snapErr == nil {
		realSnapshots := 0
		for _, snap := range snapshots {
			if snap.Name != "current" {
				realSnapshots++
			}
		}
		if realSnapshots > 0 {
			warnings = append(warnings, fmt.Sprintf("VM hat %d Snapshot(s) — diese werden NICHT migriert und gehen verloren", realSnapshots))
		}
	}

	// 6. Linked Clones (VM config has a "parent" key)
	if val, ok := config["parent"].(string); ok && val != "" {
		return warnings, fmt.Errorf("VM ist ein Linked Clone (parent: %s) — kann nicht ohne Base-Image migriert werden", val)
	}

	// 7. Cloud-Init Drive (typically ide2 or another ide/sata with "cloudinit" in value)
	driveKeys := []string{"ide0", "ide1", "ide2", "ide3", "sata0", "sata1", "sata2", "sata3", "scsi0", "scsi1"}
	for _, key := range driveKeys {
		if val, ok := config[key].(string); ok && strings.Contains(val, "cloudinit") {
			warnings = append(warnings, fmt.Sprintf("VM hat Cloud-Init-Drive (%s) — wird neu generiert auf Ziel", key))
			break
		}
	}

	// 8. Running VM + Stop Mode
	if vmStatus == "running" && mode == model.MigrationModeStop {
		warnings = append(warnings, "VM wird vor Migration heruntergefahren — Ausfallzeit erwartet")
	}

	return warnings, nil
}

// parseVMDiskSizes parses VM config to sum up all disk sizes from disk volume entries.
// It looks for keys like scsi0, virtio0, sata0, ide0, efidisk0, etc. and parses the
// size parameter (e.g. "local-zfs:vm-100-disk-0,size=32G" -> 32GB).
// Returns total disk size in bytes.
func parseVMDiskSizes(config map[string]interface{}) int64 {
	var totalBytes int64

	diskPrefixes := []string{"scsi", "virtio", "sata", "ide", "efidisk"}

	for key, raw := range config {
		val, ok := raw.(string)
		if !ok || val == "" {
			continue
		}

		// Check if key matches a disk pattern (scsi0, virtio1, sata2, ide3, efidisk0, etc.)
		isDisk := false
		for _, prefix := range diskPrefixes {
			if strings.HasPrefix(key, prefix) {
				rest := strings.TrimPrefix(key, prefix)
				if _, err := strconv.Atoi(rest); err == nil {
					isDisk = true
					break
				}
			}
		}
		if !isDisk {
			continue
		}

		// Skip non-disk entries (media=cdrom, cloudinit, none)
		if strings.Contains(val, "media=cdrom") || strings.Contains(val, "cloudinit") || val == "none" {
			continue
		}

		// Parse size from config value: "storage:volume,size=32G,other=..."
		size := parseSizeFromDiskValue(val)
		if size > 0 {
			totalBytes += size
		}
	}

	return totalBytes
}

// parseSizeFromDiskValue extracts the disk size in bytes from a Proxmox disk config value.
// Examples:
//   - "local-zfs:vm-100-disk-0,size=32G" -> 32 * 1024^3
//   - "ceph:vm-100-disk-0,size=64G,iothread=1" -> 64 * 1024^3
//   - "local-lvm:vm-200-disk-0,size=512M" -> 512 * 1024^2
//   - "local:100/vm-100-disk-0.qcow2,size=10G" -> 10 * 1024^3
func parseSizeFromDiskValue(val string) int64 {
	// Find size= parameter
	parts := strings.Split(val, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "size=") {
			sizeStr := strings.TrimPrefix(part, "size=")
			return parseSizeString(sizeStr)
		}
	}
	return 0
}

// parseSizeString parses a human-readable size string (e.g. "32G", "512M", "1T", "1024K")
// into bytes.
func parseSizeString(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	// Extract numeric part and unit
	unit := s[len(s)-1]
	numStr := s[:len(s)-1]

	var multiplier int64
	switch unit {
	case 'K', 'k':
		multiplier = 1024
	case 'M', 'm':
		multiplier = 1024 * 1024
	case 'G', 'g':
		multiplier = 1024 * 1024 * 1024
	case 'T', 't':
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		// No unit — try parsing whole string as bytes
		numStr = s
		multiplier = 1
	}

	// Support decimal values (e.g. "1.5G")
	if strings.Contains(numStr, ".") {
		var f float64
		if _, err := fmt.Sscanf(numStr, "%f", &f); err == nil {
			return int64(f * float64(multiplier))
		}
		return 0
	}

	n, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0
	}
	return n * multiplier
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
