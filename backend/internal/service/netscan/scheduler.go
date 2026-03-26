package netscan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/monitor"
	"github.com/antigravity/prometheus/internal/ssh"
	"github.com/google/uuid"
)

const (
	// anomalyThreshold is the minimum risk score that causes an anomaly to be persisted.
	anomalyThreshold = 0.3

	// scanTypeQuick is the scan_type stored in network_scans for quick scans.
	scanTypeQuick = "quick"
	// scanTypeFull is the scan_type stored in network_scans for full nmap scans.
	scanTypeFull = "full"
)

// ScanConfig holds tuning parameters for the scheduler.
type ScanConfig struct {
	QuickInterval time.Duration
	FullInterval  time.Duration
	// MaxParallel limits how many nodes are scanned concurrently.
	MaxParallel int
	// TopPorts controls how many ports nmap scans during a full scan.
	TopPorts int
}

// ScanScheduler orchestrates network scanning across all nodes.
// It runs quick (ss-based) and full (nmap-based) scans, persists results,
// computes diffs, and broadcasts anomalies over WebSocket.
type ScanScheduler struct {
	sshPool     *ssh.Pool
	nodeRepo    repository.NodeRepository
	scanRepo    repository.NetworkScanRepository
	deviceRepo  repository.NetworkDeviceRepository
	portRepo    repository.NetworkPortRepository
	anomalyRepo repository.NetworkAnomalyRepository
	baselineSvc *BaselineService
	wsHub       *monitor.WSHub
	cfg         ScanConfig

	// nmapStatus caches whether nmap is available per node (nodeID string → bool).
	nmapStatus map[string]bool
	mu         sync.RWMutex
}

// NewScanScheduler creates a ScanScheduler wired to all required dependencies.
func NewScanScheduler(
	sshPool *ssh.Pool,
	nodeRepo repository.NodeRepository,
	scanRepo repository.NetworkScanRepository,
	deviceRepo repository.NetworkDeviceRepository,
	portRepo repository.NetworkPortRepository,
	anomalyRepo repository.NetworkAnomalyRepository,
	baselineRepo repository.ScanBaselineRepository,
	wsHub *monitor.WSHub,
	cfg ScanConfig,
) *ScanScheduler {
	if cfg.MaxParallel <= 0 {
		cfg.MaxParallel = 4
	}
	if cfg.TopPorts <= 0 {
		cfg.TopPorts = 100
	}
	return &ScanScheduler{
		sshPool:     sshPool,
		nodeRepo:    nodeRepo,
		scanRepo:    scanRepo,
		deviceRepo:  deviceRepo,
		portRepo:    portRepo,
		anomalyRepo: anomalyRepo,
		baselineSvc: NewBaselineService(baselineRepo),
		wsHub:       wsHub,
		cfg:         cfg,
		nmapStatus:  make(map[string]bool),
	}
}

// RunQuickScans fetches all nodes and runs a quick (ss) scan on each in parallel.
// Results are stored, diffed, and anomalies persisted/broadcast.
func (s *ScanScheduler) RunQuickScans(ctx context.Context) error {
	nodes, err := s.nodeRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("netscan: list nodes for quick scan: %w", err)
	}

	sem := make(chan struct{}, s.cfg.MaxParallel)
	var wg sync.WaitGroup

	for _, n := range nodes {
		n := n // capture
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if err := s.scanNode(ctx, n.ID, scanTypeQuick); err != nil {
				slog.Error("netscan: quick scan failed",
					slog.String("node_id", n.ID.String()),
					slog.Any("error", err),
				)
			}
		}()
	}

	wg.Wait()
	return nil
}

// RunFullScans fetches all nodes and runs a full nmap scan on each node where
// nmap is available.
func (s *ScanScheduler) RunFullScans(ctx context.Context) error {
	nodes, err := s.nodeRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("netscan: list nodes for full scan: %w", err)
	}

	sem := make(chan struct{}, s.cfg.MaxParallel)
	var wg sync.WaitGroup

	for _, n := range nodes {
		n := n
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if err := s.scanNode(ctx, n.ID, scanTypeFull); err != nil {
				slog.Error("netscan: full scan failed",
					slog.String("node_id", n.ID.String()),
					slog.Any("error", err),
				)
			}
		}()
	}

	wg.Wait()
	return nil
}

// TriggerScan runs an immediate scan of the given type for a single node and
// returns the persisted NetworkScan record.
func (s *ScanScheduler) TriggerScan(ctx context.Context, nodeID uuid.UUID, scanType string) (*model.NetworkScan, error) {
	if scanType != scanTypeQuick && scanType != scanTypeFull {
		return nil, fmt.Errorf("netscan: unknown scan type %q", scanType)
	}

	// Create the scan record upfront so callers can track it.
	scan := &model.NetworkScan{
		NodeID:   nodeID,
		ScanType: scanType,
	}
	if err := s.scanRepo.Create(ctx, scan); err != nil {
		return nil, fmt.Errorf("netscan: create scan record: %w", err)
	}

	runner, err := s.runnerForNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	var resultsJSON json.RawMessage

	switch scanType {
	case scanTypeQuick:
		qsr, qErr := RunQuickScan(ctx, runner, nodeID.String())
		if qErr != nil {
			return scan, qErr
		}
		resultsJSON, err = json.Marshal(qsr)
		if err != nil {
			return scan, fmt.Errorf("netscan: marshal quick scan results: %w", err)
		}
		s.processDiff(ctx, nodeID, &scan.ID, qsr)

	case scanTypeFull:
		fsr, fErr := RunFullScan(ctx, runner, nodeID.String(), s.cfg.TopPorts)
		if fErr != nil {
			return scan, fErr
		}
		resultsJSON, err = json.Marshal(fsr)
		if err != nil {
			return scan, fmt.Errorf("netscan: marshal full scan results: %w", err)
		}
		s.persistFullScanDevices(ctx, nodeID, fsr)
	}

	if compErr := s.scanRepo.Complete(ctx, scan.ID, resultsJSON); compErr != nil {
		slog.Error("netscan: complete scan record", slog.Any("error", compErr))
	}
	scan.ResultsJSON = resultsJSON
	now := time.Now()
	scan.CompletedAt = &now

	return scan, nil
}

// GetDiff loads two scans by ID and computes the diff between them.
// Both scans must be quick scans whose results can be unmarshalled into
// QuickScanResult; full scan diffs are not supported via this path.
func (s *ScanScheduler) GetDiff(ctx context.Context, scanID1, scanID2 uuid.UUID) (*model.ScanDiff, error) {
	s1, err := s.scanRepo.GetByID(ctx, scanID1)
	if err != nil {
		return nil, fmt.Errorf("netscan: get scan %s: %w", scanID1, err)
	}
	s2, err := s.scanRepo.GetByID(ctx, scanID2)
	if err != nil {
		return nil, fmt.Errorf("netscan: get scan %s: %w", scanID2, err)
	}

	var qsr1, qsr2 QuickScanResult
	if err := json.Unmarshal(s1.ResultsJSON, &qsr1); err != nil {
		return nil, fmt.Errorf("netscan: unmarshal scan %s: %w", scanID1, err)
	}
	if err := json.Unmarshal(s2.ResultsJSON, &qsr2); err != nil {
		return nil, fmt.Errorf("netscan: unmarshal scan %s: %w", scanID2, err)
	}

	diff := ComputeDiff(&qsr1, &qsr2)
	return &diff, nil
}

// Shutdown performs a graceful shutdown. Currently a no-op placeholder for
// future cleanup (closing goroutines, draining channels, etc.).
func (s *ScanScheduler) Shutdown(_ context.Context) error {
	return nil
}

// scanNode performs a single node scan of the given type, stores results,
// runs diff detection, persists anomalies above threshold, and broadcasts.
func (s *ScanScheduler) scanNode(ctx context.Context, nodeID uuid.UUID, scanType string) error {
	runner, err := s.runnerForNode(ctx, nodeID)
	if err != nil {
		return err
	}

	// For full scans, check nmap availability (cached per node).
	if scanType == scanTypeFull {
		s.mu.Lock()
		avail, known := s.nmapStatus[nodeID.String()]
		if !known {
			s.mu.Unlock()
			avail = CheckNmapAvailable(ctx, runner)
			s.mu.Lock()
			s.nmapStatus[nodeID.String()] = avail
		}
		s.mu.Unlock()
		if !avail {
			slog.Debug("netscan: nmap not available, skipping full scan",
				slog.String("node_id", nodeID.String()),
			)
			return nil
		}
	}

	scan := &model.NetworkScan{
		NodeID:   nodeID,
		ScanType: scanType,
	}
	if err := s.scanRepo.Create(ctx, scan); err != nil {
		return fmt.Errorf("netscan: create scan record for node %s: %w", nodeID, err)
	}

	var resultsJSON json.RawMessage

	switch scanType {
	case scanTypeQuick:
		qsr, qErr := RunQuickScan(ctx, runner, nodeID.String())
		if qErr != nil {
			return qErr
		}
		resultsJSON, err = json.Marshal(qsr)
		if err != nil {
			return fmt.Errorf("netscan: marshal quick results: %w", err)
		}
		s.processDiff(ctx, nodeID, &scan.ID, qsr)

	case scanTypeFull:
		fsr, fErr := RunFullScan(ctx, runner, nodeID.String(), s.cfg.TopPorts)
		if fErr != nil {
			return fErr
		}
		resultsJSON, err = json.Marshal(fsr)
		if err != nil {
			return fmt.Errorf("netscan: marshal full results: %w", err)
		}
		s.persistFullScanDevices(ctx, nodeID, fsr)
	}

	if compErr := s.scanRepo.Complete(ctx, scan.ID, resultsJSON); compErr != nil {
		slog.Error("netscan: complete scan", slog.Any("error", compErr))
	}

	slog.Info("netscan: scan completed",
		slog.String("node_id", nodeID.String()),
		slog.String("type", scanType),
	)
	return nil
}

// processDiff fetches the previous scan result, computes the diff, applies the
// active baseline whitelist, and persists anomalies above the threshold.
func (s *ScanScheduler) processDiff(ctx context.Context, nodeID uuid.UUID, scanID *uuid.UUID, current *QuickScanResult) {
	// Load the previous quick scan for this node (most recent, limit 2 so we skip current).
	scans, err := s.scanRepo.ListByNode(ctx, nodeID, 2, 0)
	if err != nil || len(scans) < 2 {
		// No previous scan to diff against.
		return
	}

	var previous QuickScanResult
	// scans[0] is the just-completed scan; scans[1] is the one before.
	if err := json.Unmarshal(scans[1].ResultsJSON, &previous); err != nil {
		slog.Warn("netscan: cannot unmarshal previous scan for diff",
			slog.String("node_id", nodeID.String()),
			slog.Any("error", err),
		)
		return
	}

	diff := ComputeDiff(current, &previous)

	// Apply active baseline whitelist if one exists.
	baseline, bErr := s.baselineSvc.GetActive(ctx, nodeID)
	if bErr == nil && baseline != nil && len(baseline.WhitelistJSON) > 0 {
		if wErr := ApplyWhitelist(&diff, baseline.WhitelistJSON); wErr != nil {
			slog.Warn("netscan: apply whitelist", slog.Any("error", wErr))
		}
	}

	score := ComputeRiskScore(diff)
	if score < anomalyThreshold {
		return
	}

	detailsJSON, err := json.Marshal(diff)
	if err != nil {
		slog.Error("netscan: marshal diff for anomaly", slog.Any("error", err))
		return
	}

	anomaly := &model.NetworkAnomaly{
		NodeID:      nodeID,
		AnomalyType: "scan_diff",
		RiskScore:   score,
		DetailsJSON: json.RawMessage(detailsJSON),
		ScanID:      scanID,
	}
	if err := s.anomalyRepo.Create(ctx, anomaly); err != nil {
		slog.Error("netscan: persist anomaly", slog.Any("error", err))
		return
	}

	// Broadcast the anomaly over WebSocket so the UI reacts in real time.
	s.wsHub.BroadcastMessage(monitor.WSMessage{
		Type: "network_anomaly",
		Data: anomaly,
	})

	slog.Warn("netscan: anomaly detected",
		slog.String("node_id", nodeID.String()),
		slog.Float64("risk_score", score),
		slog.Int("new_ports", len(diff.NewPorts)),
		slog.Int("new_connections", len(diff.NewConnections)),
	)
}

// persistFullScanDevices upserts devices and ports discovered by a full nmap scan.
func (s *ScanScheduler) persistFullScanDevices(ctx context.Context, nodeID uuid.UUID, fsr *FullScanResult) {
	// For a localhost full scan we treat the node itself as the device.
	device := &model.NetworkDevice{
		NodeID:  nodeID,
		IP:      "127.0.0.1",
		IsKnown: true,
	}
	if err := s.deviceRepo.Upsert(ctx, device); err != nil {
		slog.Error("netscan: upsert localhost device", slog.Any("error", err))
		return
	}

	for _, sp := range fsr.Ports {
		port := &model.NetworkPort{
			DeviceID:       device.ID,
			Port:           sp.Port,
			Protocol:       sp.Protocol,
			State:          sp.State,
			ServiceName:    sp.ServiceName,
			ServiceVersion: sp.ServiceVersion,
		}
		if err := s.portRepo.Upsert(ctx, port); err != nil {
			slog.Error("netscan: upsert port",
				slog.Int("port", sp.Port),
				slog.Any("error", err),
			)
		}
	}
}

// runnerForNode fetches the node's SSH config from the repository and returns
// an SSHRunner backed by the pool.
//
// TODO: The ssh.Pool.Get requires an SSHConfig. We build it from the node's
// stored SSH fields (Hostname, SSHPort, SSHUser, SSHPrivateKey). Adjust if the
// decryption of SSHPrivateKey is handled elsewhere in the call chain.
func (s *ScanScheduler) runnerForNode(ctx context.Context, nodeID uuid.UUID) (SSHRunner, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("netscan: get node %s: %w", nodeID, err)
	}

	cfg := ssh.SSHConfig{
		Host:       node.Hostname,
		Port:       node.SSHPort,
		User:       node.SSHUser,
		PrivateKey: node.SSHPrivateKey,
		HostKey:    node.SSHHostKey,
	}

	client, err := s.sshPool.Get(nodeID.String(), cfg)
	if err != nil {
		return nil, fmt.Errorf("netscan: get ssh client for node %s: %w", nodeID, err)
	}

	return &poolClientRunner{client: client}, nil
}

// poolClientRunner adapts *ssh.Client to the SSHRunner interface expected by
// RunQuickScan, RunFullScan, etc.
type poolClientRunner struct {
	client interface {
		RunCommand(ctx context.Context, cmd string) (*ssh.CommandResult, error)
	}
}

func (r *poolClientRunner) Run(ctx context.Context, cmd string) (string, int, error) {
	result, err := r.client.RunCommand(ctx, cmd)
	if err != nil {
		// Check if it's a context cancellation — propagate as-is.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return "", -1, err
		}
		return "", -1, err
	}
	return result.Stdout, result.ExitCode, nil
}
