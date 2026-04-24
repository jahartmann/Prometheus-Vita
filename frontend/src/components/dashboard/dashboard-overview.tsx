"use client";

import { Activity, AlertTriangle, Cpu, Plus, Server, ServerCog, ShieldCheck } from "lucide-react";
import Link from "next/link";
import { useNodeStore } from "@/stores/node-store";
import { KpiCard } from "@/components/ui/kpi-card";
import { NodeGrid } from "./node-grid";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export function DashboardOverview() {
  const { nodes, nodeStatus } = useNodeStore();

  const onlineNodes = nodes.filter((n) => n.is_online).length;
  const totalVMs = Object.values(nodeStatus).reduce(
    (acc, s) => acc + (s?.vm_count ?? 0) + (s?.ct_count ?? 0),
    0
  );
  const runningVMs = Object.values(nodeStatus).reduce(
    (acc, s) => acc + (s?.vm_running ?? 0) + (s?.ct_running ?? 0),
    0
  );
  const offlineNodes = nodes.length - onlineNodes;

  const statusValues = Object.values(nodeStatus).filter(Boolean);
  const avgCpu = statusValues.length > 0
    ? statusValues.reduce((acc, s) => acc + (s?.cpu_usage ?? 0), 0) / statusValues.length
    : 0;
  const avgMemory = statusValues.length > 0
    ? statusValues.reduce((acc, s) => {
      if (!s || s.memory_total <= 0) return acc;
      return acc + (s.memory_used / s.memory_total) * 100;
    }, 0) / statusValues.length
    : 0;

  const healthTone = offlineNodes > 0 || avgCpu > 85 || avgMemory > 90 ? "warning" : "healthy";
  const healthLabel = healthTone === "healthy" ? "Operativ" : "Prüfen";
  const healthText = offlineNodes > 0
    ? `${offlineNodes} Node${offlineNodes === 1 ? "" : "s"} offline`
    : avgCpu > 85
      ? "CPU-Auslastung erhöht"
      : avgMemory > 90
        ? "RAM-Auslastung erhöht"
        : "Keine Blocker im Lagebild";

  const healthItems = [
    { label: "Cluster", value: healthLabel, detail: healthText, tone: healthTone },
    { label: "Nodes online", value: `${onlineNodes}/${nodes.length}`, detail: "Erreichbare Hosts", tone: offlineNodes > 0 ? "warning" : "healthy" },
    { label: "Workloads aktiv", value: runningVMs, detail: `${totalVMs} VMs/Container`, tone: "neutral" },
    { label: "CPU Ø", value: `${avgCpu.toFixed(1)}%`, detail: `RAM Ø ${avgMemory.toFixed(1)}%`, tone: avgCpu > 85 || avgMemory > 90 ? "warning" : "neutral" },
  ];

  return (
    <div className="flex flex-col gap-5">
      <section className="rounded-lg border bg-card p-4">
        <div className="mb-4 flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
          <div>
            <h2 className="text-base font-semibold">Lagebild</h2>
            <p className="text-sm text-muted-foreground">
              Kompakte Sicht auf Betriebszustand, Kapazität und laufende Workloads.
            </p>
          </div>
          <Button variant="outline" size="sm" asChild>
            <Link href="/settings/nodes">
              <Plus className="mr-2 h-4 w-4" />
              Server hinzufügen
            </Link>
          </Button>
        </div>

        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          {healthItems.map((item) => (
            <div
              key={item.label}
              className={cn(
                "rounded-md border p-3",
                item.tone === "healthy" && "border-green-200 bg-green-50/70 dark:border-green-900/60 dark:bg-green-950/20",
                item.tone === "warning" && "border-orange-200 bg-orange-50/70 dark:border-orange-900/60 dark:bg-orange-950/20",
                item.tone === "neutral" && "bg-background"
              )}
            >
              <p className="text-xs font-medium text-muted-foreground">{item.label}</p>
              <p className="mt-1 text-xl font-semibold">{item.value}</p>
              <p className="mt-1 text-xs text-muted-foreground">{item.detail}</p>
            </div>
          ))}
        </div>
      </section>

      <div className="grid gap-3 grid-cols-2 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard
          title="Server"
          value={nodes.length}
          subtitle={`${onlineNodes} online`}
          icon={Server}
          color="blue"
        />
        <KpiCard
          title="VMs / Container"
          value={totalVMs}
          subtitle={`${runningVMs} aktiv`}
          icon={ServerCog}
          color="green"
        />
        <KpiCard
          title="CPU-Durchschnitt"
          value={`${avgCpu.toFixed(1)}%`}
          subtitle="Alle Server"
          icon={Cpu}
          color={avgCpu > 80 ? "red" : avgCpu > 60 ? "orange" : "blue"}
        />
        <KpiCard
          title="Status"
          value={offlineNodes === 0 ? "Gesund" : `${offlineNodes} offline`}
          subtitle={offlineNodes === 0 ? "Alle Systeme operativ" : "Achtung erforderlich"}
          icon={offlineNodes === 0 ? Activity : AlertTriangle}
          color={offlineNodes === 0 ? "green" : "red"}
        />
      </div>

      <section className="flex flex-col gap-3">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 className="text-base font-semibold">Server-Flotte</h2>
            <p className="text-sm text-muted-foreground">Nodes und Workload-Zustand für den schnellen Drill-down.</p>
          </div>
          <Button variant="ghost" size="sm" asChild>
            <Link href="/health">
              <ShieldCheck className="mr-2 h-4 w-4" />
              VM-Gesundheit
            </Link>
          </Button>
        </div>
        <NodeGrid />
      </section>
    </div>
  );
}
