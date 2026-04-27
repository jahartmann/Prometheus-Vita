"use client";

import { useEffect } from "react";
import { useNodeStore } from "@/stores/node-store";
import { DashboardOverview } from "@/components/dashboard/dashboard-overview";
import { BriefingWidget } from "@/components/dashboard/briefing-widget";
import { SecurityWidget } from "@/components/dashboard/security-widget";
import { AttentionBanner } from "@/components/dashboard/attention-banner";
import { AgentActivityFeed } from "@/components/dashboard/agent-activity-feed";
import { StatusBadge } from "@/components/ui/status-badge";

export default function DashboardPage() {
  const { nodes, fetchNodes } = useNodeStore();

  useEffect(() => {
    fetchNodes();
  }, [fetchNodes]);

  const onlineNodes = nodes.filter((node) => node.is_online).length;
  const offlineNodes = nodes.length - onlineNodes;
  const isHealthy = offlineNodes === 0;
  const greeting = greetingFor(new Date());

  return (
    <div className="flex flex-col gap-6">
      {/* Hero — quiet, single column. The status pills live to the right
          so the eye scans title → data without crossing the page. */}
      <section className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="eyebrow">{greeting}</p>
          <h1 className="mt-1 text-3xl font-semibold tracking-tight">Lagezentrum</h1>
          <p className="mt-1 max-w-2xl text-sm text-muted-foreground">
            Prioritäten, Clusterzustand und nächste Aktionen — auf einen Blick.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <StatusBadge tone={isHealthy ? "ok" : "warning"}>
            {isHealthy ? "Cluster operativ" : `${offlineNodes} offline`}
          </StatusBadge>
          <StatusBadge tone="muted" withIcon={false}>
            <span className="tabular">{onlineNodes}</span>/<span className="tabular">{nodes.length}</span> Nodes online
          </StatusBadge>
        </div>
      </section>

      <AttentionBanner />

      {/* KPIs */}
      <DashboardOverview />

      {/* Two-column: briefing (the agent's voice) + security (its eyes). */}
      <section className="grid gap-4 xl:grid-cols-2">
        <BriefingWidget />
        <SecurityWidget />
      </section>

      {/* What the agent has been doing — the "live admin" feed. */}
      <AgentActivityFeed limit={20} pollInterval={15000} />
    </div>
  );
}

function greetingFor(d: Date): string {
  const hour = d.getHours();
  if (hour < 5) return "Gute Nacht";
  if (hour < 11) return "Guten Morgen";
  if (hour < 14) return "Mittag";
  if (hour < 18) return "Guten Tag";
  if (hour < 22) return "Guten Abend";
  return "Späte Stunde";
}
