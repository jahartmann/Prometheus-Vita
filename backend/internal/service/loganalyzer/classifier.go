package loganalyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/antigravity/prometheus/internal/llm"
	"github.com/antigravity/prometheus/internal/model"
)

// Classifier uses an LLM to assess a batch of log entries and return anomaly
// scores, severity labels, and short summaries.
type Classifier struct {
	llmRegistry *llm.Registry
	model       string
	semaphore   chan struct{}
}

// NewClassifier creates a Classifier with the given model preference and
// maximum concurrency.  If concurrency is <= 0 it defaults to 1.
func NewClassifier(registry *llm.Registry, model string, concurrency int) *Classifier {
	if concurrency <= 0 {
		concurrency = 1
	}
	sem := make(chan struct{}, concurrency)
	for i := 0; i < concurrency; i++ {
		sem <- struct{}{}
	}
	return &Classifier{
		llmRegistry: registry,
		model:       model,
		semaphore:   sem,
	}
}

// ClassifyBatch sends a batch of log entries to the LLM for analysis.
// It returns one LogAssessment per entry.  If the LLM call fails or the
// response is not valid JSON the returned slice will be nil (not an error —
// callers should treat nil as "no assessments available").
func (c *Classifier) ClassifyBatch(ctx context.Context, entries []model.LogEntry) ([]model.LogAssessment, error) {
	if len(entries) == 0 {
		return nil, nil
	}

	// Acquire a semaphore slot, honouring context cancellation.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.semaphore:
	}
	defer func() { c.semaphore <- struct{}{} }()

	prompt := buildClassificationPrompt(entries)

	provider, err := c.resolveProvider()
	if err != nil {
		slog.Warn("loganalyzer: no LLM provider available for classification",
			slog.Any("error", err),
		)
		return nil, nil
	}

	req := llm.CompletionRequest{
		Model: c.model,
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   2048,
		Temperature: 0.0,
	}

	resp, err := provider.Complete(ctx, req)
	if err != nil {
		slog.Warn("loganalyzer: LLM classification failed",
			slog.Any("error", err),
		)
		return nil, nil
	}

	assessments, err := parseAssessments(resp.Content, len(entries))
	if err != nil {
		slog.Warn("loganalyzer: failed to parse LLM assessment response",
			slog.Any("error", err),
			slog.String("raw", truncate(resp.Content, 200)),
		)
		return nil, nil
	}

	return assessments, nil
}

// resolveProvider returns the provider for the configured model, falling back
// to the registry default.
func (c *Classifier) resolveProvider() (llm.Provider, error) {
	if c.model != "" {
		if p, err := c.llmRegistry.GetForModel(c.model); err == nil {
			return p, nil
		}
	}
	// Fall back to the default model's provider.
	defaultModel := c.llmRegistry.DefaultModel()
	if p, err := c.llmRegistry.GetForModel(defaultModel); err == nil {
		return p, nil
	}
	return nil, fmt.Errorf("no LLM provider available")
}

// buildClassificationPrompt constructs the prompt sent to the LLM.
func buildClassificationPrompt(entries []model.LogEntry) string {
	var sb strings.Builder

	sb.WriteString("Analyze these log entries. For each, return a JSON object with:\n")
	sb.WriteString("  severity (normal/info/warning/error/critical),\n")
	sb.WriteString("  anomaly_score (0.0-1.0),\n")
	sb.WriteString("  category (auth_failure/service_crash/disk_issue/network_error/security_threat/performance/unknown),\n")
	sb.WriteString("  summary (brief explanation).\n\n")
	sb.WriteString("Return a JSON array of assessments, one per entry, in the same order.\n")
	sb.WriteString("Respond with ONLY the JSON array — no markdown, no explanation.\n\n")
	sb.WriteString("Entries:\n")

	for i, e := range entries {
		fmt.Fprintf(&sb, "%d. %s\n", i+1, e.Raw)
	}

	return sb.String()
}

// parseAssessments extracts a []model.LogAssessment from the LLM response text.
// It attempts to find the JSON array even when the model wraps it in markdown.
func parseAssessments(content string, expectedCount int) ([]model.LogAssessment, error) {
	// Strip possible markdown fences.
	content = strings.TrimSpace(content)
	if idx := strings.Index(content, "["); idx != -1 {
		content = content[idx:]
	}
	if idx := strings.LastIndex(content, "]"); idx != -1 {
		content = content[:idx+1]
	}

	var assessments []model.LogAssessment
	if err := json.Unmarshal([]byte(content), &assessments); err != nil {
		return nil, fmt.Errorf("unmarshal assessments: %w", err)
	}

	// Pad with zero-value assessments if the model returned fewer items than
	// expected so that callers can always zip entries and assessments by index.
	for len(assessments) < expectedCount {
		assessments = append(assessments, model.LogAssessment{
			Severity:     "unknown",
			AnomalyScore: 0,
			Category:     "unknown",
			Summary:      "",
		})
	}

	return assessments[:expectedCount], nil
}

// truncate returns at most n runes of s.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}
