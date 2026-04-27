package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// Notifier is the minimal contract the push pipeline needs from a
// transport. The Telegram bot service satisfies this interface; future
// transports (Slack, Matrix, email digest) can plug in the same way.
type Notifier interface {
	BroadcastToLinkedUsers(ctx context.Context, text string) int
}

// PushService is the agent's "active voice". When the rest of the system
// detects something noteworthy — a critical security finding, a kritische
// Anomalie, the daily briefing — it calls into PushService, which formats a
// human-readable message and broadcasts it via every registered Notifier.
//
// Throttling guards against runaway loops: the same logical event (deduped by
// key) cannot be pushed more often than minInterval. This keeps the agent
// from spamming the admin if a flapping metric fires the same hook many times.
type PushService struct {
	notifiers []Notifier

	mu          sync.Mutex
	lastPushed  map[string]time.Time
	minInterval time.Duration
}

func NewPushService(notifiers ...Notifier) *PushService {
	return &PushService{
		notifiers:   filterNonNilNotifiers(notifiers),
		lastPushed:  make(map[string]time.Time),
		minInterval: 5 * time.Minute,
	}
}

func filterNonNilNotifiers(in []Notifier) []Notifier {
	out := make([]Notifier, 0, len(in))
	for _, n := range in {
		if n != nil {
			out = append(out, n)
		}
	}
	return out
}

// HasNotifiers reports whether at least one transport is wired up. Callers
// can short-circuit expensive formatting work if the answer is false.
func (s *PushService) HasNotifiers() bool {
	return s != nil && len(s.notifiers) > 0
}

// PushRaw broadcasts the given text immediately, bypassing throttling.
// Use this sparingly — only for messages the user explicitly requested.
func (s *PushService) PushRaw(ctx context.Context, text string) {
	if s == nil || text == "" {
		return
	}
	for _, n := range s.notifiers {
		count := n.BroadcastToLinkedUsers(ctx, text)
		slog.Debug("agent push", slog.Int("recipients", count))
	}
}

// pushThrottled deduplicates: if `key` was already broadcast within
// minInterval, the call is silently dropped. This is the right default for
// alert hooks that may fire repeatedly during a single incident.
func (s *PushService) pushThrottled(ctx context.Context, key, text string) {
	if s == nil || text == "" {
		return
	}
	s.mu.Lock()
	if last, ok := s.lastPushed[key]; ok && time.Since(last) < s.minInterval {
		s.mu.Unlock()
		return
	}
	s.lastPushed[key] = time.Now()
	s.mu.Unlock()

	s.PushRaw(ctx, text)
}

// SecurityFinding is the shape PushService expects when intelligence detects
// a critical or emergency event. Keeping this struct local to the agent
// package avoids a hard dependency on the model layer.
type SecurityFinding struct {
	ID          string
	NodeName    string
	Severity    string
	Title       string
	Description string
	Recommendation string
}

// PushSecurity formats and sends a single security finding. Throttled per
// finding ID, so re-sends from the same scheduler tick are coalesced.
func (s *PushService) PushSecurity(ctx context.Context, f SecurityFinding) {
	if !s.HasNotifiers() {
		return
	}
	emoji := severityEmoji(f.Severity)
	var b strings.Builder
	fmt.Fprintf(&b, "%s *Sicherheits-Befund — %s*\n", emoji, strings.ToUpper(f.Severity))
	if f.NodeName != "" {
		fmt.Fprintf(&b, "Node: `%s`\n", f.NodeName)
	}
	fmt.Fprintf(&b, "\n*%s*\n", f.Title)
	if f.Description != "" {
		fmt.Fprintf(&b, "\n%s\n", f.Description)
	}
	if f.Recommendation != "" {
		fmt.Fprintf(&b, "\n_Empfehlung:_ %s\n", f.Recommendation)
	}
	s.pushThrottled(ctx, "security:"+f.ID, b.String())
}

// AnomalyAlert describes one anomaly that's important enough to push out.
type AnomalyAlert struct {
	ID         string
	NodeName   string
	Metric     string  // cpu, ram, disk, network…
	Severity   string  // warning, critical
	Value      float64 // current value
	Threshold  float64 // breached threshold
	Trend      string  // optional: "steigend seit 5 Min"
}

// PushAnomaly formats and sends an anomaly alert.
func (s *PushService) PushAnomaly(ctx context.Context, a AnomalyAlert) {
	if !s.HasNotifiers() {
		return
	}
	emoji := severityEmoji(a.Severity)
	var b strings.Builder
	fmt.Fprintf(&b, "%s *Anomalie — %s auf %s*\n", emoji, a.Metric, a.NodeName)
	fmt.Fprintf(&b, "Wert: `%.1f` (Schwelle `%.1f`)\n", a.Value, a.Threshold)
	if a.Trend != "" {
		fmt.Fprintf(&b, "Trend: %s\n", a.Trend)
	}
	s.pushThrottled(ctx, "anomaly:"+a.ID, b.String())
}

// PushBriefing sends a daily briefing summary. Not throttled — the briefing
// scheduler already runs at most once per day, and the user explicitly opted
// in to receive it.
func (s *PushService) PushBriefing(ctx context.Context, headline, body string) {
	if !s.HasNotifiers() {
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "📋 *Tages-Briefing*\n%s\n\n", headline)
	if body != "" {
		b.WriteString(body)
	}
	s.PushRaw(ctx, b.String())
}

// PushApprovalRequest tells the user that an action is waiting for their
// approval. The actual approve/deny flow currently lives in the Telegram bot
// or in the UI; this push is purely a heads-up.
func (s *PushService) PushApprovalRequest(ctx context.Context, toolName, summary string) {
	if !s.HasNotifiers() {
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "🛂 *Freigabe nötig*\nDer Agent möchte `%s` ausführen.\n\n", toolName)
	if summary != "" {
		fmt.Fprintf(&b, "%s\n", summary)
	}
	b.WriteString("\nÖffne die Anwendung unter /approvals oder antworte hier.")
	s.PushRaw(ctx, b.String())
}

// summarizeApprovalArgs formats tool arguments into a short, human-readable
// preview for the Telegram approval prompt. Strips deeply nested objects
// and truncates long strings — the full payload is still available in the
// approvals UI.
func summarizeApprovalArgs(args string, security ToolSecurity) string {
	parts := []string{}
	if security.Risk != "" {
		parts = append(parts, fmt.Sprintf("Risiko: %s", security.Risk))
	}
	if args == "" || args == "{}" {
		return strings.Join(parts, " · ")
	}
	// Prefer compact key=value rendering of the top-level object. We don't
	// pretty-print JSON because Telegram strips most whitespace anyway.
	preview := args
	if len(preview) > 240 {
		preview = preview[:240] + "…"
	}
	parts = append(parts, "Args: "+preview)
	return strings.Join(parts, "\n")
}

func severityEmoji(severity string) string {
	switch strings.ToLower(severity) {
	case "emergency":
		return "🚨"
	case "critical":
		return "🔴"
	case "warning":
		return "⚠️"
	case "info":
		return "ℹ️"
	default:
		return "•"
	}
}
