"use client";

import { AlertTriangle, CheckCircle2, Info } from "lucide-react";
import { useNodeStore } from "@/stores/node-store";
import { cn } from "@/lib/utils";

interface AttentionItem {
  title: string;
  description: string;
  severity: "critical" | "warning" | "info";
}

const severityMeta = {
  critical: {
    label: "Kritisch",
    icon: AlertTriangle,
    itemClass: "border-red-200 bg-red-50 text-red-950 dark:border-red-900/70 dark:bg-red-950/30 dark:text-red-100",
    badgeClass: "bg-red-600 text-white",
    rank: 0,
  },
  warning: {
    label: "Warnung",
    icon: AlertTriangle,
    itemClass: "border-orange-200 bg-orange-50 text-orange-950 dark:border-orange-900/70 dark:bg-orange-950/30 dark:text-orange-100",
    badgeClass: "bg-orange-500 text-white",
    rank: 1,
  },
  info: {
    label: "Hinweis",
    icon: Info,
    itemClass: "border-blue-200 bg-blue-50 text-blue-950 dark:border-blue-900/70 dark:bg-blue-950/30 dark:text-blue-100",
    badgeClass: "bg-blue-500 text-white",
    rank: 2,
  },
};

export function AttentionBanner() {
  const { nodes, nodeStatus, nodeErrors } = useNodeStore();

  const items: AttentionItem[] = [];

  const offlineNodes = nodes.filter((n) => !n.is_online);
  offlineNodes.forEach((n) => {
    items.push({
      title: `${n.name} ist offline`,
      description: "Server ist nicht erreichbar.",
      severity: "critical",
    });
  });

  Object.entries(nodeErrors).forEach(([nodeId, error]) => {
    if (error) {
      const node = nodes.find((n) => n.id === nodeId);
      items.push({
        title: `${node?.name ?? nodeId}: Verbindungsfehler`,
        description: error,
        severity: "warning",
      });
    }
  });

  Object.entries(nodeStatus).forEach(([nodeId, status]) => {
    if (status && status.cpu_usage > 85) {
      const node = nodes.find((n) => n.id === nodeId);
      items.push({
        title: `Hohe CPU-Last auf ${node?.name ?? nodeId}`,
        description: `CPU bei ${status.cpu_usage.toFixed(0)}%.`,
        severity: status.cpu_usage > 95 ? "critical" : "warning",
      });
    }
  });

  Object.entries(nodeStatus).forEach(([nodeId, status]) => {
    if (status && status.memory_total > 0) {
      const memPercent = (status.memory_used / status.memory_total) * 100;
      if (memPercent > 90) {
        const node = nodes.find((n) => n.id === nodeId);
        items.push({
          title: `Hoher RAM-Verbrauch auf ${node?.name ?? nodeId}`,
          description: `RAM bei ${memPercent.toFixed(0)}%.`,
          severity: memPercent > 95 ? "critical" : "warning",
        });
      }
    }
  });

  const sortedItems = [...items].sort((a, b) => severityMeta[a.severity].rank - severityMeta[b.severity].rank);
  const criticalCount = items.filter((item) => item.severity === "critical").length;
  const warningCount = items.filter((item) => item.severity === "warning").length;

  if (items.length === 0) {
    return (
      <section className="rounded-lg border bg-card p-4">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-start gap-3">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-muted text-green-600 dark:text-green-400">
              <CheckCircle2 className="h-4 w-4" />
            </div>
            <div>
              <h2 className="text-sm font-semibold">Keine akuten Eingriffe</h2>
              <p className="text-sm text-muted-foreground">
                Nodes, Verbindungen und Basismetriken liefern aktuell keine priorisierten Warnsignale.
              </p>
            </div>
          </div>
          <span className="w-fit rounded-full border px-2.5 py-1 text-xs font-medium text-muted-foreground">
            Lage ruhig
          </span>
        </div>
      </section>
    );
  }

  return (
    <section className="rounded-lg border bg-card p-4">
      <div className="mb-3 flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
        <div className="flex items-start gap-3">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-muted text-orange-600 dark:text-orange-300">
            <AlertTriangle className="h-4 w-4" />
          </div>
          <div>
            <h2 className="text-sm font-semibold">Aufmerksamkeit zuerst</h2>
            <p className="text-sm text-muted-foreground">
              Priorisierte Signale, bevor sie Workloads oder Wartungsfenster beeinträchtigen.
            </p>
          </div>
        </div>
        <div className="flex flex-wrap gap-2 text-xs">
          <span className="rounded-full bg-red-600 px-2.5 py-1 font-medium text-white">{criticalCount} kritisch</span>
          <span className="rounded-full bg-orange-500 px-2.5 py-1 font-medium text-white">{warningCount} Warnungen</span>
        </div>
      </div>

      <div className="grid gap-2 md:grid-cols-2 xl:grid-cols-4">
        {sortedItems.slice(0, 4).map((item, i) => {
          const meta = severityMeta[item.severity];
          const Icon = meta.icon;

          return (
            <article
              key={`${item.title}-${i}`}
              className={cn("min-w-0 rounded-md border p-3", meta.itemClass)}
            >
              <div className="mb-2 flex items-start justify-between gap-2">
                <div className="flex min-w-0 items-center gap-2">
                  <Icon className="h-3.5 w-3.5 shrink-0" />
                  <h3 className="truncate text-sm font-semibold">{item.title}</h3>
                </div>
                <span className={cn("shrink-0 rounded-full px-1.5 py-0.5 text-[10px] font-medium", meta.badgeClass)}>
                  {meta.label}
                </span>
              </div>
              <p className="text-xs opacity-85">{item.description}</p>
            </article>
          );
        })}
      </div>
    </section>
  );
}
