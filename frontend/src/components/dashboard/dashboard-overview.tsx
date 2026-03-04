"use client";

import { Server, ServerCog, Activity, AlertTriangle, Cpu, Plus } from "lucide-react";
import Link from "next/link";
import { useNodeStore } from "@/stores/node-store";
import { KpiCard } from "@/components/ui/kpi-card";
import { NodeGrid } from "./node-grid";
import { Button } from "@/components/ui/button";

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

  return (
    <div className="space-y-6">
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
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

      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Server</h2>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" asChild>
            <Link href="/settings/nodes">
              <Plus className="mr-2 h-4 w-4" />
              Server hinzufuegen
            </Link>
          </Button>
        </div>
      </div>
      <NodeGrid />
    </div>
  );
}
