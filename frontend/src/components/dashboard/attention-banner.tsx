"use client";

import type { LucideIcon } from "lucide-react";
import { AlertTriangle, CheckCircle2, Cpu, Info, MemoryStick, ServerOff } from "lucide-react";
import { useNodeStore } from "@/stores/node-store";
import { ActionCard } from "@/components/ui/action-card";
import { Card, CardContent } from "@/components/ui/card";
import type { StatusTone } from "@/components/ui/status-badge";

interface AttentionItem {
  title: string;
  description: string;
  severity: "critical" | "warning" | "info";
  kind: "offline" | "error" | "cpu" | "memory" | "info";
  href: string;
}

const severityMeta = {
  critical: {
    label: "Kritisch",
    tone: "critical",
    rank: 0,
  },
  warning: {
    label: "Warnung",
    tone: "warning",
    rank: 1,
  },
  info: {
    label: "Hinweis",
    tone: "info",
    rank: 2,
  },
} satisfies Record<AttentionItem["severity"], { label: string; tone: StatusTone; rank: number }>;

const kindMeta: Record<AttentionItem["kind"], { icon: LucideIcon; actionLabel: string }> = {
  offline: { icon: ServerOff, actionLabel: "Node pruefen" },
  error: { icon: AlertTriangle, actionLabel: "Signal pruefen" },
  cpu: { icon: Cpu, actionLabel: "Auslastung pruefen" },
  memory: { icon: MemoryStick, actionLabel: "Auslastung pruefen" },
  info: { icon: Info, actionLabel: "Aufgaben öffnen" },
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
      kind: "offline",
      href: `/nodes/${n.id}`,
    });
  });

  Object.entries(nodeErrors).forEach(([nodeId, error]) => {
    if (error) {
      const node = nodes.find((n) => n.id === nodeId);
      items.push({
        title: `${node?.name ?? nodeId}: Verbindungsfehler`,
        description: error,
        severity: "warning",
        kind: "error",
        href: node ? `/nodes/${node.id}` : "/monitoring",
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
        kind: "cpu",
        href: node ? `/nodes/${node.id}` : "/monitoring",
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
          kind: "memory",
          href: node ? `/nodes/${node.id}` : "/monitoring",
        });
      }
    }
  });

  const sortedItems = [...items].sort((a, b) => severityMeta[a.severity].rank - severityMeta[b.severity].rank);
  const criticalCount = items.filter((item) => item.severity === "critical").length;
  const warningCount = items.filter((item) => item.severity === "warning").length;

  if (items.length === 0) {
    return (
      <Card>
        <CardContent className="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between">
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
        </CardContent>
      </Card>
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
              Priorisierte Signale, bevor sie Workloads oder Wartungsfenster beeintraechtigen.
            </p>
          </div>
        </div>
        <div className="flex flex-wrap gap-2 text-xs">
          <span className="rounded-full bg-red-600 px-2.5 py-1 font-medium text-white">{criticalCount} kritisch</span>
          <span className="rounded-full bg-orange-500 px-2.5 py-1 font-medium text-white">{warningCount} Warnungen</span>
        </div>
      </div>

      <div className="grid gap-3 md:grid-cols-3">
        {sortedItems.slice(0, 3).map((item, i) => {
          const meta = severityMeta[item.severity];
          const action = kindMeta[item.kind];

          return (
            <ActionCard
              key={`${item.title}-${i}`}
              tone={meta.tone}
              icon={action.icon}
              title={item.title}
              description={item.description}
              badge={meta.label}
              href={item.href}
              actionLabel={action.actionLabel}
            />
          );
        })}
      </div>
    </section>
  );
}
