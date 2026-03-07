"use client";

import { AlertTriangle } from "lucide-react";
import { useNodeStore } from "@/stores/node-store";

interface AttentionItem {
  title: string;
  description: string;
  severity: "critical" | "warning" | "info";
}

const severityStyles = {
  critical: "border-red-200 bg-red-50 text-red-900 dark:border-red-800 dark:bg-red-500/10 dark:text-red-200",
  warning: "border-orange-200 bg-orange-50 text-orange-900 dark:border-orange-800 dark:bg-orange-500/10 dark:text-orange-200",
  info: "border-blue-200 bg-blue-50 text-blue-900 dark:border-blue-800 dark:bg-blue-500/10 dark:text-blue-200",
};

const severityBadge = {
  critical: "bg-red-500 text-white",
  warning: "bg-orange-500 text-white",
  info: "bg-blue-500 text-white",
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

  if (items.length === 0) return null;

  return (
    <div className="rounded-xl border bg-card p-4">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <AlertTriangle className="h-4 w-4 text-orange-500" />
          <h3 className="text-sm font-semibold">Was braucht Aufmerksamkeit</h3>
          <p className="text-xs text-muted-foreground">
            Signale die Beachtung brauchen bevor sie Workloads beeintraechtigen.
          </p>
        </div>
        <span className="flex h-5 min-w-5 items-center justify-center rounded-full bg-red-500 px-1.5 text-[11px] font-medium text-white">
          {items.length}
        </span>
      </div>
      <div className="flex gap-3 overflow-x-auto pb-1">
        {items.slice(0, 4).map((item, i) => (
          <div
            key={i}
            className={`flex-1 min-w-[200px] rounded-lg border p-3 ${severityStyles[item.severity]}`}
          >
            <div className="flex items-center justify-between mb-1">
              <span className="text-sm font-medium">{item.title}</span>
              <span className={`rounded-full px-1.5 py-0.5 text-[10px] font-medium ${severityBadge[item.severity]}`}>
                {item.severity === "critical" ? "!" : "i"}
              </span>
            </div>
            <p className="text-xs opacity-80">{item.description}</p>
          </div>
        ))}
      </div>
    </div>
  );
}
