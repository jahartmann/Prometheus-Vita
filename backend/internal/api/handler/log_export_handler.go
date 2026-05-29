package handler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type LogExportHandler struct {
	redisClient  *redis.Client
	anomalyRepo  repository.LogAnomalyRepository
	bookmarkRepo repository.LogBookmarkRepository
}

func NewLogExportHandler(
	redisClient *redis.Client,
	anomalyRepo repository.LogAnomalyRepository,
	bookmarkRepo repository.LogBookmarkRepository,
) *LogExportHandler {
	return &LogExportHandler{
		redisClient:  redisClient,
		anomalyRepo:  anomalyRepo,
		bookmarkRepo: bookmarkRepo,
	}
}

type exportLogEntry struct {
	StreamID  string            `json:"stream_id"`
	Timestamp string            `json:"timestamp"`
	NodeID    string            `json:"node_id"`
	Source    string            `json:"source"`
	Severity  string            `json:"severity"`
	Message   string            `json:"message"`
	Raw       string            `json:"raw"`
	Values    map[string]string `json:"values,omitempty"`
}

func (h *LogExportHandler) Export(c echo.Context) error {
	ctx := c.Request().Context()

	nodeIDsParam := c.QueryParam("node_ids")
	nodeIDParam := c.QueryParam("node_id")
	sourcesParam := c.QueryParam("sources")
	fromParam := c.QueryParam("from")
	toParam := c.QueryParam("to")
	format := c.QueryParam("format")
	// Accept both the documented "include_ai" and the frontend's
	// "include_annotations" spelling.
	includeAI := c.QueryParam("include_ai") == "true" || c.QueryParam("include_annotations") == "true"

	if format == "" {
		format = "text"
	}

	// Parse time bounds; default to last 24h if absent.
	now := time.Now()
	fromTime := now.Add(-24 * time.Hour)
	toTime := now

	if fromParam != "" {
		if t, err := time.Parse(time.RFC3339, fromParam); err == nil {
			fromTime = t
		}
	}
	if toParam != "" {
		if t, err := time.Parse(time.RFC3339, toParam); err == nil {
			toTime = t
		}
	}

	// Build the Redis stream min/max IDs from time bounds.
	minID, maxID := redisStreamBounds(fromTime, toTime)

	// Parse requested node IDs (accept both node_ids plural and node_id singular).
	nodeIDs, err := parseNodeIDParams(nodeIDsParam, nodeIDParam)
	if err != nil {
		return response.BadRequest(c, "invalid node id")
	}

	// Parse optional source filter.
	var sourceFilter map[string]struct{}
	if sourcesParam != "" {
		sourceFilter = make(map[string]struct{})
		for _, s := range strings.Split(sourcesParam, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				sourceFilter[s] = struct{}{}
			}
		}
	}

	// Read log entries from Redis Streams.
	var entries []exportLogEntry
	for _, nodeID := range nodeIDs {
		key := fmt.Sprintf("logs:%s", nodeID.String())
		msgs, err := h.redisClient.XRange(ctx, key, minID, maxID).Result()
		if err != nil {
			slog.Warn("log export: failed to read stream",
				slog.String("key", key),
				slog.Any("error", err),
			)
			continue
		}

		for _, msg := range msgs {
			source, _ := msg.Values["source"].(string)
			if sourceFilter != nil {
				if _, ok := sourceFilter[source]; !ok {
					continue
				}
			}

			message, _ := msg.Values["message"].(string)
			raw, _ := msg.Values["raw"].(string)
			severity, _ := msg.Values["severity"].(string)
			tsMillis, _ := msg.Values["timestamp"].(string)

			tsFormatted := ""
			if tsMillis != "" {
				if ms, err := parseInt64(tsMillis); err == nil {
					tsFormatted = time.UnixMilli(ms).UTC().Format(time.RFC3339)
				}
			}

			entry := exportLogEntry{
				StreamID:  msg.ID,
				Timestamp: tsFormatted,
				NodeID:    nodeID.String(),
				Source:    source,
				Severity:  severity,
				Message:   message,
				Raw:       raw,
			}
			entries = append(entries, entry)
		}
	}

	// Optionally attach AI anomaly annotations as additional entries so they
	// flow into every export format uniformly.
	if includeAI {
		for _, nodeID := range nodeIDs {
			anomalies, err := h.anomalyRepo.ListByNode(ctx, nodeID, 1000, 0)
			if err != nil {
				slog.Warn("log export: failed to read anomalies",
					slog.String("node_id", nodeID.String()),
					slog.Any("error", err),
				)
				continue
			}
			for _, a := range anomalies {
				if a.Timestamp.Before(fromTime) || a.Timestamp.After(toTime) {
					continue
				}
				entries = append(entries, exportLogEntry{
					Timestamp: a.Timestamp.UTC().Format(time.RFC3339),
					NodeID:    nodeID.String(),
					Source:    "ai:" + a.Source,
					Severity:  a.Severity,
					Message:   a.Summary,
					Raw:       a.RawLog,
					Values: map[string]string{
						"category":      a.Category,
						"anomaly_score": fmt.Sprintf("%.2f", a.AnomalyScore),
					},
				})
			}
		}
	}

	// Determine filename and content-type.
	ts := time.Now().UTC().Format("20060102-150405")
	ext := format
	if ext == "text" {
		ext = "txt"
	}
	filename := fmt.Sprintf("logs-%s.%s", ts, ext)
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	switch format {
	case "json":
		c.Response().Header().Set("Content-Type", "application/json")
		return json.NewEncoder(c.Response()).Encode(entries)

	case "csv":
		c.Response().Header().Set("Content-Type", "text/csv")
		w := csv.NewWriter(c.Response())
		_ = w.Write([]string{"stream_id", "timestamp", "node_id", "source", "severity", "message", "raw"})
		for _, e := range entries {
			_ = w.Write([]string{e.StreamID, e.Timestamp, e.NodeID, e.Source, e.Severity, e.Message, e.Raw})
		}
		w.Flush()
		return w.Error()

	default: // "text"
		c.Response().Header().Set("Content-Type", "text/plain")
		sb := &strings.Builder{}
		for _, e := range entries {
			fmt.Fprintf(sb, "[%s] [%s] [%s] %s: %s\n",
				e.Timestamp, e.NodeID, e.Severity, e.Source, e.Raw)
		}
		_, err := c.Response().Write([]byte(sb.String()))
		return err
	}
}

func parseInt64(s string) (int64, error) {
	var v int64
	_, err := fmt.Sscanf(s, "%d", &v)
	return v, err
}

// parseNodeIDParams parses node IDs from the plural ("node_ids", comma-joined)
// and singular ("node_id") query parameters, returning the deduplicated union.
func parseNodeIDParams(plural, singular string) ([]uuid.UUID, error) {
	seen := make(map[uuid.UUID]struct{})
	var ids []uuid.UUID
	add := func(raw string) error {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil
		}
		id, err := uuid.Parse(raw)
		if err != nil {
			return err
		}
		if _, ok := seen[id]; ok {
			return nil
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
		return nil
	}
	for _, raw := range strings.Split(plural, ",") {
		if err := add(raw); err != nil {
			return nil, err
		}
	}
	if err := add(singular); err != nil {
		return nil, err
	}
	return ids, nil
}

// redisStreamBounds converts a time range into valid Redis stream IDs for
// XRANGE. The end bound uses the bare "<ms>" form (Redis auto-fills the
// sequence to the maximum); the previous "<ms>-+" form was an invalid stream
// ID that made XRANGE error out and silently return nothing.
func redisStreamBounds(from, to time.Time) (string, string) {
	return fmt.Sprintf("%d-0", from.UnixMilli()), fmt.Sprintf("%d", to.UnixMilli())
}
