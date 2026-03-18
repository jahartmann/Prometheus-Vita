package logstream

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/ssh"
	"github.com/redis/go-redis/v9"
)

// StreamConfig holds configuration for the StreamManager.
type StreamConfig struct {
	// WorkerPoolSize is the maximum number of concurrent workers (one per node).
	WorkerPoolSize int
	// RotationInterval is how often non-priority workers poll for new log lines.
	RotationInterval time.Duration
	// RedisMaxLen is the approximate maximum number of entries kept per stream.
	RedisMaxLen int64
	// RedisMaxAge is the TTL applied to Redis stream keys.
	RedisMaxAge time.Duration
}

// StreamManager orchestrates log streaming workers for all registered nodes.
type StreamManager struct {
	sshPool     *ssh.Pool
	redisClient *redis.Client
	nodeRepo    repository.NodeRepository
	sourceRepo  repository.LogSourceRepository
	cfg         StreamConfig

	workers     map[string]*Worker
	subscribers map[string][]chan model.LogEntry // nodeID -> subscriber channels
	mu          sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
}

// NewStreamManager creates a new StreamManager. Call Start to begin streaming.
func NewStreamManager(
	sshPool *ssh.Pool,
	redisClient *redis.Client,
	nodeRepo repository.NodeRepository,
	sourceRepo repository.LogSourceRepository,
	cfg StreamConfig,
) *StreamManager {
	if cfg.WorkerPoolSize <= 0 {
		cfg.WorkerPoolSize = 10
	}
	if cfg.RotationInterval <= 0 {
		cfg.RotationInterval = 30 * time.Second
	}
	if cfg.RedisMaxLen <= 0 {
		cfg.RedisMaxLen = 10_000
	}
	if cfg.RedisMaxAge <= 0 {
		cfg.RedisMaxAge = 24 * time.Hour
	}
	return &StreamManager{
		sshPool:     sshPool,
		redisClient: redisClient,
		nodeRepo:    nodeRepo,
		sourceRepo:  sourceRepo,
		cfg:         cfg,
		workers:     make(map[string]*Worker),
		subscribers: make(map[string][]chan model.LogEntry),
	}
}

// Start initialises the manager and launches streaming workers for all nodes.
// It returns immediately; work is done in background goroutines.
func (m *StreamManager) Start(ctx context.Context) {
	m.ctx, m.cancel = context.WithCancel(ctx)

	go m.run()
}

// run is the main background loop — bootstraps workers and handles rotation.
func (m *StreamManager) run() {
	if err := m.bootstrapWorkers(); err != nil {
		slog.Error("logstream: failed to bootstrap workers", slog.Any("error", err))
	}

	ticker := time.NewTicker(m.cfg.RotationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.rotateNonPriorityWorkers()
		}
	}
}

// bootstrapWorkers lists nodes and starts a worker for each, up to WorkerPoolSize.
func (m *StreamManager) bootstrapWorkers() error {
	nodes, err := m.nodeRepo.List(m.ctx)
	if err != nil {
		return fmt.Errorf("list nodes: %w", err)
	}

	slots := m.cfg.WorkerPoolSize
	for i := range nodes {
		if slots <= 0 {
			slog.Warn("logstream: worker pool limit reached, remaining nodes skipped",
				slog.Int("limit", m.cfg.WorkerPoolSize),
				slog.Int("total_nodes", len(nodes)),
			)
			break
		}

		node := &nodes[i]
		if err := m.startWorkerForNode(node, true); err != nil {
			slog.Error("logstream: failed to start worker",
				slog.String("node_id", node.ID.String()),
				slog.String("node_name", node.Name),
				slog.Any("error", err),
			)
			continue
		}
		slots--
	}
	return nil
}

// startWorkerForNode creates and starts a Worker for the given node.
func (m *StreamManager) startWorkerForNode(node *model.Node, priority bool) error {
	nodeID := node.ID.String()

	sources, err := m.sourceRepo.ListByNode(m.ctx, node.ID)
	if err != nil {
		return fmt.Errorf("list log sources for node %s: %w", nodeID, err)
	}

	// Filter to only enabled sources.
	var enabled []model.LogSource
	for _, s := range sources {
		if s.Enabled {
			enabled = append(enabled, s)
		}
	}

	sshCfg := ssh.SSHConfig{
		Host:       node.Hostname,
		Port:       node.SSHPort,
		User:       node.SSHUser,
		PrivateKey: node.SSHPrivateKey,
	}
	if sshCfg.Port == 0 {
		sshCfg.Port = 22
	}
	if sshCfg.User == "" {
		sshCfg.User = "root"
	}

	w := &Worker{
		nodeID:   nodeID,
		nodeName: node.Name,
		sshPool:  m.sshPool,
		sshCfg:   sshCfg,
		sources:  enabled,
		manager:  m,
		priority: priority,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}

	m.mu.Lock()
	// Stop any existing worker for this node before replacing it.
	if old, ok := m.workers[nodeID]; ok {
		old.Stop()
	}
	m.workers[nodeID] = w
	m.mu.Unlock()

	w.Start(m.ctx)

	slog.Info("logstream: worker started",
		slog.String("node_id", nodeID),
		slog.String("node_name", node.Name),
		slog.Bool("priority", priority),
		slog.Int("sources", len(enabled)),
	)
	return nil
}

// rotateNonPriorityWorkers re-creates workers for non-priority nodes so that
// they can pick up any newly enabled/disabled log sources.
func (m *StreamManager) rotateNonPriorityWorkers() {
	nodes, err := m.nodeRepo.List(m.ctx)
	if err != nil {
		slog.Error("logstream: rotation failed to list nodes", slog.Any("error", err))
		return
	}

	m.mu.RLock()
	toRotate := make([]*model.Node, 0)
	for i := range nodes {
		node := &nodes[i]
		w, ok := m.workers[node.ID.String()]
		if ok && !w.priority {
			toRotate = append(toRotate, node)
		}
	}
	m.mu.RUnlock()

	for _, node := range toRotate {
		if err := m.startWorkerForNode(node, false); err != nil {
			slog.Error("logstream: rotation failed for node",
				slog.String("node_id", node.ID.String()),
				slog.Any("error", err),
			)
		}
	}
}

// Shutdown stops all workers and releases resources.
// It waits for workers to finish up to the context deadline.
func (m *StreamManager) Shutdown(ctx context.Context) error {
	m.cancel()

	m.mu.Lock()
	workers := make([]*Worker, 0, len(m.workers))
	for _, w := range m.workers {
		workers = append(workers, w)
	}
	m.mu.Unlock()

	done := make(chan struct{})
	go func() {
		for _, w := range workers {
			w.Stop()
		}
		close(done)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("shutdown timed out: %w", ctx.Err())
	case <-done:
		return nil
	}
}

// Subscribe returns a channel that receives log entries for the given node.
// The caller must call Unsubscribe when done to avoid a goroutine/channel leak.
func (m *StreamManager) Subscribe(nodeID string) <-chan model.LogEntry {
	ch := make(chan model.LogEntry, 100)

	m.mu.Lock()
	m.subscribers[nodeID] = append(m.subscribers[nodeID], ch)
	m.mu.Unlock()

	return ch
}

// Unsubscribe removes a previously subscribed channel.
func (m *StreamManager) Unsubscribe(nodeID string, ch <-chan model.LogEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()

	subs := m.subscribers[nodeID]
	for i, sub := range subs {
		if sub == ch {
			m.subscribers[nodeID] = append(subs[:i], subs[i+1:]...)
			close(sub)
			return
		}
	}
}

// notifySubscribers performs a non-blocking fan-out to all subscriber channels
// registered for the given node.
func (m *StreamManager) notifySubscribers(nodeID string, entry model.LogEntry) {
	m.mu.RLock()
	subs := m.subscribers[nodeID]
	m.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- entry:
		default:
			// Drop the entry rather than blocking — slow consumers miss data.
		}
	}
}

// writeToRedis persists a log entry to the Redis Stream `logs:{nodeID}`.
// It uses approximate MAXLEN trimming to cap memory usage.
func (m *StreamManager) writeToRedis(ctx context.Context, nodeID string, entry model.LogEntry) error {
	key := fmt.Sprintf("logs:%s", nodeID)

	args := &redis.XAddArgs{
		Stream: key,
		MaxLen: m.cfg.RedisMaxLen,
		Approx: true,
		ID:     "*",
		Values: map[string]interface{}{
			"id":        entry.ID,
			"timestamp": entry.Timestamp.UnixMilli(),
			"node_id":   entry.NodeID,
			"source":    entry.Source,
			"severity":  entry.Severity,
			"process":   entry.Process,
			"pid":       entry.PID,
			"message":   entry.Message,
			"raw":       entry.Raw,
		},
	}

	if err := m.redisClient.XAdd(ctx, args).Err(); err != nil {
		return fmt.Errorf("xadd logs:%s: %w", nodeID, err)
	}

	// Refresh TTL so the stream ages out when the node goes quiet.
	m.redisClient.Expire(ctx, key, m.cfg.RedisMaxAge)

	return nil
}
