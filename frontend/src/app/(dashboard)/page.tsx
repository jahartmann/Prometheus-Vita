"use client";

import { useEffect } from "react";
import { Activity, AlertTriangle, Server } from "lucide-react";
import { useNodeStore } from "@/stores/node-store";
import { DashboardOverview } from "@/components/dashboard/dashboard-overview";
import { BriefingWidget } from "@/components/dashboard/briefing-widget";
import { SecurityWidget } from "@/components/dashboard/security-widget";
import { AttentionBanner } from "@/components/dashboard/attention-banner";
import { cn } from "@/lib/utils";

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
      <header className="rounded-lg border bg-card p-4">
        <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div>
            <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Prometheus Operations</p>
            <h1 className="text-2xl font-bold tracking-tight">Operations-Cockpit</h1>
            <p className="text-sm text-muted-foreground">
              Priorisierte Übersicht über Infrastruktur, Risiken und nächste Eingriffe.
            </p>
          </div>
          <div className="flex flex-wrap gap-2 text-xs">
            <span
              className={cn(
                "inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 font-medium",
                isHealthy
                  ? "border-green-200 bg-green-50 text-green-800 dark:border-green-900/60 dark:bg-green-950/20 dark:text-green-200"
                  : "border-orange-200 bg-orange-50 text-orange-800 dark:border-orange-900/60 dark:bg-orange-950/20 dark:text-orange-200"
              )}
            >
              {isHealthy ? <Activity className="h-3.5 w-3.5" /> : <AlertTriangle className="h-3.5 w-3.5" />}
              {isHealthy ? "Cluster operativ" : `${offlineNodes} offline`}
            </span>
            <span className="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 font-medium text-muted-foreground">
              <Server className="h-3.5 w-3.5" />
              {onlineNodes}/{nodes.length} Nodes online
            </span>
          </div>
        </div>
      </header>

      <AttentionBanner />
      <DashboardOverview />

      <section className="grid gap-4 xl:grid-cols-2">
        <BriefingWidget />
        <SecurityWidget />
      </section>
    </div>
  );
}
