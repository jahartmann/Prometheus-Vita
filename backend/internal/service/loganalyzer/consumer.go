package loganalyzer

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/monitor"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ConsumerConfig holds tuning parameters for the Consumer.
type ConsumerConfig struct {
	// BatchSize is the maximum number of log entries per classification batch.
	BatchSize int
	// BatchTimeout is the maximum time to wait before flushing an incomplete batch.
	BatchTimeout time.Duration
	// AnomalyThreshold is the minimum AnomalyScore to persist to PostgreSQL.
	AnomalyThreshold float64
	// AlertThreshold is the minimum AnomalyScore to broadcast via WebSocket.
	AlertThreshold float64
	// DedupWindow is the minimum interval between duplicate alerts for the same
	// (nodeID, category) combination.
	DedupWindow time.Duration
}

// dedupKey uniquely identifies an anomaly category on a node for deduplication.
type dedupKey struct {
	nodeID   string
	category string
}

// Consumer reads log entries from Redis Streams, classifies them with the LLM,
// persists anomalies, and broadcasts high-severity alerts via WebSocket.
type Consumer struct {
	redisClient *redis.Client
	classifier  *Classifier
	anomalyRepo repository.LogAnomalyRepository
	wsHub       *monitor.WSHub
	cfg         ConsumerConfig

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// lastAlert tracks the most recent alert time per (nodeID, category).
	lastAlert   map[dedupKey]time.Time
	lastAlertMu sync.Mutex
}

// NewConsumer creates a Consumer ready to be started.
func NewConsumer(
	redisClient *redis.Client,
	classifier *Classifier,
	anomalyRepo repository.LogAnomalyRepository,
	wsHub *monitor.WSHub,
	cfg ConsumerConfig,
) *Consumer {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}
	if cfg.BatchTimeout <= 0 {
		cfg.BatchTimeout = 5 * time.Second
	}
	if cfg.DedupWindow <= 0 {
		cfg.DedupWindow = 5 * time.Minute
	}
	return &Consumer{
		redisClient: redisClient,
		classifier:  classifier,
		anomalyRepo: anomalyRepo,
		wsHub:       wsHub,
		cfg:         cfg,
		lastAlert:   make(map[dedupKey]time.Time),
	}
}

// Start launches the consumer loop in a background goroutine.
func (c *Consumer) Start(ctx context.Context) {
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.run()
	}()
}

// Shutdown stops the consumer and waits for the background goroutine to exit.
func (c *Consumer) Shutdown(_ context.Context) error {
	c.cancel()
	c.wg.Wait()
	return nil
}

// run is the main consumer loop.
func (c *Consumer) run() {
	// lastIDs tracks the last-read Redis Stream entry ID per stream key.
	// "$" means "entries from now on"; "0" would mean "from the start".
	lastIDs := make(map[string]string)

	batchTimer := time.NewTimer(c.cfg.BatchTimeout)
	defer batchTimer.Stop()

	var batch []model.LogEntry

	flushBatch := func() {
		if len(batch) == 0 {
			return
		}
		c.processBatch(batch)
		batch = batch[:0]

		if !batchTimer.Stop() {
			select {
			case <-batchTimer.C:
			default:
			}
		}
		batchTimer.Reset(c.cfg.BatchTimeout)
	}

	for {
		select {
		case <-c.ctx.Done():
			flushBatch()
			return
		case <-batchTimer.C:
			flushBatch()
			batchTimer.Reset(c.cfg.BatchTimeout)
		default:
		}

		entries, err := c.readStreams(lastIDs)
		if err != nil {
			if c.ctx.Err() != nil {
				return
			}
			slog.Warn("loganalyzer: redis XREAD error", slog.Any("error", err))
			// Small back-off to avoid a tight error loop.
			select {
			case <-c.ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
			continue
		}

		batch = append(batch, entries...)

		if len(batch) >= c.cfg.BatchSize {
			flushBatch()
		}
	}
}

// readStreams performs a blocking XREAD across all active `logs:*` streams,
// waiting up to 1 second for new data.  It updates lastIDs in-place.
func (c *Consumer) readStreams(lastIDs map[string]string) ([]model.LogEntry, error) {
	// Discover stream keys.  Redis KEYS is fine here since it's a background
	// task and the key space is small (one stream per node).
	keys, err := c.redisClient.Keys(c.ctx, "logs:*").Result()
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		// No streams yet — wait a moment before retrying.
		select {
		case <-c.ctx.Done():
		case <-time.After(1 * time.Second):
		}
		return nil, nil
	}

	// Build the XREAD argument list: for each key use the last seen ID or "$"
	// if this is the first read.
	streams := make([]string, 0, len(keys)*2)
	for _, k := range keys {
		streams = append(streams, k)
	}
	for _, k := range keys {
		id, ok := lastIDs[k]
		if !ok {
			id = "$"
		}
		streams = append(streams, id)
	}

	results, err := c.redisClient.XRead(c.ctx, &redis.XReadArgs{
		Streams: streams,
		Count:   int64(c.cfg.BatchSize),
		Block:   1000 * time.Millisecond,
	}).Result()
	if err != nil {
		if err == redis.Nil {
			// Timeout — no new messages.
			return nil, nil
		}
		return nil, err
	}

	var entries []model.LogEntry
	for _, stream := range results {
		for _, msg := range stream.Messages {
			// Advance the cursor for this stream.
			lastIDs[stream.Stream] = msg.ID

			entry := decodeStreamMessage(msg.Values)
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

// processBatch classifies a batch of entries and acts on anomalies.
func (c *Consumer) processBatch(entries []model.LogEntry) {
	assessments, err := c.classifier.ClassifyBatch(c.ctx, entries)
	if err != nil {
		slog.Warn("loganalyzer: classifier returned error", slog.Any("error", err))
		return
	}
	if assessments == nil {
		// LLM unavailable — skip silently.
		return
	}

	for i, entry := range entries {
		if i >= len(assessments) {
			break
		}
		a := assessments[i]

		if a.AnomalyScore <= c.cfg.AnomalyThreshold {
			continue
		}

		nodeID, err := uuid.Parse(entry.NodeID)
		if err != nil {
			slog.Warn("loganalyzer: invalid node_id in log entry",
				slog.String("node_id", entry.NodeID),
			)
			continue
		}

		anomaly := &model.LogAnomaly{
			NodeID:       nodeID,
			Timestamp:    entry.Timestamp,
			Source:       entry.Source,
			Severity:     a.Severity,
			AnomalyScore: a.AnomalyScore,
			Category:     a.Category,
			Summary:      a.Summary,
			RawLog:       entry.Raw,
		}

		// Persist to PostgreSQL.
		if err := c.anomalyRepo.Create(c.ctx, anomaly); err != nil {
			slog.Warn("loganalyzer: failed to persist anomaly",
				slog.String("node_id", entry.NodeID),
				slog.Any("error", err),
			)
		}

		// Broadcast high-severity anomalies via WebSocket with deduplication.
		if a.AnomalyScore > c.cfg.AlertThreshold {
			c.maybeAlert(anomaly)
		}
	}
}

// maybeAlert broadcasts an anomaly alert unless a recent alert for the same
// (nodeID, category) was already sent within DedupWindow.
func (c *Consumer) maybeAlert(anomaly *model.LogAnomaly) {
	key := dedupKey{nodeID: anomaly.NodeID.String(), category: anomaly.Category}

	c.lastAlertMu.Lock()
	last, seen := c.lastAlert[key]
	if seen && time.Since(last) < c.cfg.DedupWindow {
		c.lastAlertMu.Unlock()
		return
	}
	c.lastAlert[key] = time.Now()
	c.lastAlertMu.Unlock()

	c.wsHub.BroadcastMessage(monitor.WSMessage{
		Type: "log_anomaly",
		Data: anomaly,
	})

	slog.Info("loganalyzer: anomaly alert broadcast",
		slog.String("node_id", anomaly.NodeID.String()),
		slog.String("category", anomaly.Category),
		slog.Float64("score", anomaly.AnomalyScore),
	)
}

// decodeStreamMessage converts a Redis XMessage Values map into a LogEntry.
func decodeStreamMessage(values map[string]interface{}) model.LogEntry {
	str := func(key string) string {
		if v, ok := values[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}

	var ts time.Time
	if raw := str("timestamp"); raw != "" {
		if ms, err := strconv.ParseInt(raw, 10, 64); err == nil {
			ts = time.UnixMilli(ms)
		}
	}

	pid, _ := strconv.Atoi(str("pid"))

	return model.LogEntry{
		ID:        str("id"),
		Timestamp: ts,
		NodeID:    str("node_id"),
		Source:    str("source"),
		Severity:  str("severity"),
		Process:   str("process"),
		PID:       pid,
		Message:   str("message"),
		Raw:       str("raw"),
	}
}
