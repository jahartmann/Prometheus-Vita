"use client";

import { useEffect, useRef } from "react";
import { AgentActivityFeed } from "@/components/dashboard/agent-activity-feed";
import { AttentionBanner } from "@/components/dashboard/attention-banner";
import { BriefingWidget } from "@/components/dashboard/briefing-widget";
import { DashboardOverview } from "@/components/dashboard/dashboard-overview";
import { SecurityWidget } from "@/components/dashboard/security-widget";
import { StatusBadge } from "@/components/ui/status-badge";
import { useNodeStore } from "@/stores/node-store";

export default function DashboardPage() {
  const { nodes, nodeStatus, fetchNodes, fetchNodeStatus } = useNodeStore();
  const requestedStatusRef = useRef<Set<string>>(new Set());

  useEffect(() => {
    fetchNodes();
  }, [fetchNodes]);

  useEffect(() => {
    nodes.forEach((node) => {
      if (node.is_online && !nodeStatus[node.id] && !requestedStatusRef.current.has(node.id)) {
        requestedStatusRef.current.add(node.id);
        void fetchNodeStatus(node.id);
      }
    });
  }, [nodes, nodeStatus, fetchNodeStatus]);

  const onlineNodes = nodes.filter((node) => node.is_online).length;
  const offlineNodes = nodes.length - onlineNodes;
  const statusTone = nodes.length === 0 ? "muted" : offlineNodes === 0 ? "ok" : "warning";
  const statusLabel =
    nodes.length === 0 ? "Keine Nodes" : offlineNodes === 0 ? "Cluster operativ" : `${offlineNodes} offline`;

  return (
    <div className="flex flex-col gap-4">
      <section className="flex flex-col gap-3 border-b ops-divider pb-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="eyebrow">{greetingFor(new Date())}</p>
          <h1 className="mt-1 text-2xl font-semibold tracking-tight">Lagezentrum</h1>
          <p className="mt-1 max-w-2xl text-sm text-muted-foreground">
            Prioritäten, Clusterzustand und nächste Aktionen - ohne Funktionslärm.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <StatusBadge tone={statusTone}>{statusLabel}</StatusBadge>
          <StatusBadge tone="muted" withIcon={false}>
            <span className="tabular">{onlineNodes}</span>/<span className="tabular">{nodes.length}</span> Nodes online
          </StatusBadge>
        </div>
      </section>

      <AttentionBanner />
      <DashboardOverview />

      <section className="grid gap-4 xl:grid-cols-2">
        <BriefingWidget />
        <SecurityWidget />
      </section>

      <AgentActivityFeed limit={12} pollInterval={15000} />
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
