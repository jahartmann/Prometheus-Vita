"use client";

import { useEffect } from "react";
import { useNodeStore } from "@/stores/node-store";
import { DashboardOverview } from "@/components/dashboard/dashboard-overview";
import { BriefingWidget } from "@/components/dashboard/briefing-widget";
import { SecurityWidget } from "@/components/dashboard/security-widget";
import { AttentionBanner } from "@/components/dashboard/attention-banner";
import { StatusBadge } from "@/components/ui/status-badge";

export default function DashboardPage() {
  const { nodes, fetchNodes } = useNodeStore();

  useEffect(() => {
    fetchNodes();
  }, [fetchNodes]);

  const onlineNodes = nodes.filter((node) => node.is_online).length;
  const offlineNodes = nodes.length - onlineNodes;
  const isHealthy = offlineNodes === 0;

  return (
    <div className="flex flex-col gap-5">
      <section className="surface-panel-strong overflow-hidden p-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Prometheus Operations</p>
            <h1 className="mt-1 text-3xl font-bold tracking-tight">Lagezentrum</h1>
            <p className="mt-1 max-w-2xl text-sm text-muted-foreground">
              Prioritaeten, Clusterzustand und naechste Aktionen in einer ruhigen Operations-Ansicht.
            </p>
          </div>
          <div className="flex flex-wrap gap-2">
            <StatusBadge tone={isHealthy ? "ok" : "warning"}>
              {isHealthy ? "Cluster operativ" : `${offlineNodes} offline`}
            </StatusBadge>
            <StatusBadge tone="muted" withIcon={false}>
              {onlineNodes}/{nodes.length} Nodes online
            </StatusBadge>
          </div>
        </div>
      </section>

      <AttentionBanner />
      <DashboardOverview />

      <section className="grid gap-4 xl:grid-cols-2">
        <BriefingWidget />
        <SecurityWidget />
      </section>
    </div>
  );
}
