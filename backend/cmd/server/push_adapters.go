package main

import (
	"context"

	"github.com/antigravity/prometheus/internal/service/agent"
	"github.com/antigravity/prometheus/internal/service/briefing"
	"github.com/antigravity/prometheus/internal/service/intelligence"
)

// The intelligence and briefing packages each define a local PushNotifier
// interface so they don't have to depend on the agent package directly. This
// file provides the bridging adapters that the main wiring uses to plug the
// shared *agent.PushService into both consumers.

type intelligencePushAdapter struct {
	push *agent.PushService
}

func (a intelligencePushAdapter) PushSecurity(ctx context.Context, f intelligence.PushFinding) {
	if a.push == nil {
		return
	}
	a.push.PushSecurity(ctx, agent.SecurityFinding{
		ID:             f.ID,
		NodeName:       f.NodeName,
		Severity:       f.Severity,
		Title:          f.Title,
		Description:    f.Description,
		Recommendation: f.Recommendation,
	})
}

type briefingPushAdapter struct {
	push *agent.PushService
}

func (a briefingPushAdapter) PushBriefing(ctx context.Context, headline, body string) {
	if a.push == nil {
		return
	}
	a.push.PushBriefing(ctx, headline, body)
}

// Compile-time guards: ensure the adapters keep matching the local notifier
// interfaces in their respective packages. If those interfaces change, this
// will break the build before the runtime would.
var (
	_ intelligence.PushNotifier = intelligencePushAdapter{}
	_ briefing.PushNotifier     = briefingPushAdapter{}
)
