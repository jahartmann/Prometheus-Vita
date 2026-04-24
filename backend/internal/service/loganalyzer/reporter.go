package loganalyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/antigravity/prometheus/internal/llm"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Reporter reads log data from Redis Streams and produces a comprehensive
// analysis using the LLM, then persists the result.
type Reporter struct {
	redisClient  *redis.Client
	llmRegistry  *llm.Registry
	analysisRepo repository.LogAnalysisRepository
}

// NewReporter creates a new Reporter.
func NewReporter(
	redisClient *redis.Client,
	llmRegistry *llm.Registry,
	analysisRepo repository.LogAnalysisRepository,
) *Reporter {
	return &Reporter{
		redisClient:  redisClient,
		llmRegistry:  llmRegistry,
		analysisRepo: analysisRepo,
	}
}

// Analyze reads logs from Redis Streams for the requested nodes and time range,
// sends them to the LLM for deep analysis, persists the resulting
// LogAnalysis, and returns it.
func (r *Reporter) Analyze(ctx context.Context, req model.AnalyzeLogsRequest) (*model.LogAnalysis, error) {
	return r.analyze(ctx, req, nil)
}

func (r *Reporter) AnalyzeScheduled(ctx context.Context, req model.AnalyzeLogsRequest, scheduleID uuid.UUID) (*model.LogAnalysis, error) {
	return r.analyze(ctx, req, &scheduleID)
}

func (r *Reporter) analyze(ctx context.Context, req model.AnalyzeLogsRequest, scheduleID *uuid.UUID) (*model.LogAnalysis, error) {
	rawLogs, err := r.collectLogs(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("collect logs: %w", err)
	}

	slog.Info("loganalyzer: analyzing logs",
		slog.Int("node_count", len(req.NodeIDs)),
		slog.Int("log_lines", len(rawLogs)),
		slog.Time("from", req.TimeFrom),
		slog.Time("to", req.TimeTo),
	)

	provider, modelName, err := r.resolveProvider()
	if err != nil {
		return nil, fmt.Errorf("no LLM provider available: %w", err)
	}

	prompt := buildAnalysisPrompt(req, rawLogs)

	llmResp, err := provider.Complete(ctx, llm.CompletionRequest{
		Model: modelName,
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   4096,
		Temperature: 0.2,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM completion: %w", err)
	}

	report, err := parseAnalysisReport(llmResp.Content)
	if err != nil {
		slog.Warn("loganalyzer: failed to parse analysis report, storing raw",
			slog.Any("error", err),
		)
		// Store the raw text as a minimal report so the result is not lost.
		report = &model.LogAnalysisReport{
			Summary:   llmResp.Content,
			ModelUsed: modelName,
			TimeRange: model.TimeRange{From: req.TimeFrom, To: req.TimeTo},
		}
	}

	report.ModelUsed = modelName
	report.TimeRange = model.TimeRange{From: req.TimeFrom, To: req.TimeTo}
	for _, id := range req.NodeIDs {
		report.NodesAnalyzed = append(report.NodesAnalyzed, id.String())
	}

	reportJSON, err := json.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("marshal report: %w", err)
	}

	analysis := &model.LogAnalysis{
		NodeIDs:    req.NodeIDs,
		TimeFrom:   req.TimeFrom,
		TimeTo:     req.TimeTo,
		ReportJSON: reportJSON,
		ModelUsed:  modelName,
		ScheduleID: scheduleID,
	}

	if err := r.analysisRepo.Create(ctx, analysis); err != nil {
		return nil, fmt.Errorf("persist analysis: %w", err)
	}

	slog.Info("loganalyzer: analysis complete",
		slog.String("analysis_id", analysis.ID.String()),
		slog.String("model", modelName),
	)

	return analysis, nil
}

// collectLogs reads log entries from the `logs:{nodeID}` Redis Stream for each
// node in the request, filtered to the requested time range.
func (r *Reporter) collectLogs(ctx context.Context, req model.AnalyzeLogsRequest) ([]string, error) {
	// Redis Stream IDs are millisecond timestamps; use them to bound the read.
	minID := fmt.Sprintf("%d-0", req.TimeFrom.UnixMilli())
	maxID := fmt.Sprintf("%d-+", req.TimeTo.UnixMilli())

	var lines []string

	for _, nodeID := range req.NodeIDs {
		key := fmt.Sprintf("logs:%s", nodeID.String())

		msgs, err := r.redisClient.XRange(ctx, key, minID, maxID).Result()
		if err != nil {
			slog.Warn("loganalyzer: failed to read stream",
				slog.String("key", key),
				slog.Any("error", err),
			)
			continue
		}

		for _, msg := range msgs {
			raw, _ := msg.Values["raw"].(string)
			if raw == "" {
				// Fall back to message field.
				raw, _ = msg.Values["message"].(string)
			}
			if raw != "" {
				lines = append(lines, raw)
			}
		}
	}

	return lines, nil
}

// resolveProvider picks the best available LLM provider and returns it along
// with the model name that should be used in the request.
func (r *Reporter) resolveProvider() (llm.Provider, string, error) {
	defaultModel := r.llmRegistry.DefaultModel()
	if p, err := r.llmRegistry.GetForModel(defaultModel); err == nil {
		return p, defaultModel, nil
	}
	return nil, "", fmt.Errorf("no LLM provider registered")
}

// buildAnalysisPrompt constructs the deep-analysis prompt.
func buildAnalysisPrompt(req model.AnalyzeLogsRequest, logs []string) string {
	var sb strings.Builder

	sb.WriteString("You are an expert infrastructure operations analyst.\n\n")
	sb.WriteString("Analyze the following log data collected from Proxmox nodes ")
	fmt.Fprintf(&sb, "between %s and %s.\n", req.TimeFrom.Format(time.RFC3339), req.TimeTo.Format(time.RFC3339))
	if req.Context != "" {
		fmt.Fprintf(&sb, "Additional context: %s\n", req.Context)
	}
	sb.WriteString("\nReturn ONLY a JSON object with these fields:\n")
	sb.WriteString("  summary           — one-paragraph executive summary\n")
	sb.WriteString("  anomalies         — array of objects: {node_id, timestamp, source, severity, anomaly_score, category, summary, raw_log}\n")
	sb.WriteString("  patterns          — array of objects: {pattern, occurrences, severity, description}\n")
	sb.WriteString("  root_cause_hypotheses — array of strings\n")
	sb.WriteString("  recommendations   — array of strings\n\n")
	sb.WriteString("No markdown fences, no explanation outside the JSON.\n\n")
	sb.WriteString("Log data:\n")

	// Limit the number of lines sent to avoid exceeding token limits.
	const maxLines = 500
	if len(logs) > maxLines {
		logs = logs[:maxLines]
	}
	for _, line := range logs {
		sb.WriteString(line)
		sb.WriteByte('\n')
	}

	return sb.String()
}

// parseAnalysisReport extracts a LogAnalysisReport from the LLM response text.
func parseAnalysisReport(content string) (*model.LogAnalysisReport, error) {
	content = strings.TrimSpace(content)
	// Strip markdown fences if present.
	if idx := strings.Index(content, "{"); idx != -1 {
		content = content[idx:]
	}
	if idx := strings.LastIndex(content, "}"); idx != -1 {
		content = content[:idx+1]
	}

	var report model.LogAnalysisReport
	if err := json.Unmarshal([]byte(content), &report); err != nil {
		return nil, fmt.Errorf("unmarshal analysis report: %w", err)
	}
	return &report, nil
}
