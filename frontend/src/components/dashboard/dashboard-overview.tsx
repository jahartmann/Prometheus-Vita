"use client";

import { Server, ServerCog, Activity, AlertTriangle } from "lucide-react";
import { useNodeStore } from "@/stores/node-store";
import { KpiCard } from "./kpi-card";
import { NodeGrid } from "./node-grid";

export function DashboardOverview() {
  const { nodes, nodeStatus } = useNodeStore();

  const onlineNodes = nodes.filter((n) => n.is_online).length;
  const totalVMs = Object.values(nodeStatus).reduce(
    (acc, s) => acc + (s?.vm_count ?? 0),
    0
  );
  const runningVMs = Object.values(nodeStatus).reduce(
    (acc, s) => acc + (s?.vm_running ?? 0),
    0
  );
  const offlineNodes = nodes.length - onlineNodes;

  return (
    <div className="space-y-6">
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard
          title="Nodes"
          value={nodes.length}
          subtitle={`${onlineNodes} online`}
          icon={Server}
        />
        <KpiCard
          title="VMs / Container"
          value={totalVMs}
          subtitle={`${runningVMs} aktiv`}
          icon={ServerCog}
        />
        <KpiCard
          title="Status"
          value={offlineNodes === 0 ? "Gesund" : `${offlineNodes} offline`}
          subtitle={offlineNodes === 0 ? "Alle Systeme operativ" : "Achtung erforderlich"}
          icon={offlineNodes === 0 ? Activity : AlertTriangle}
        />
        <KpiCard
          title="Uptime"
          value="--"
          subtitle="Durchschnitt aller Nodes"
          icon={Activity}
        />
      </div>

      <div>
        <h2 className="mb-4 text-lg font-semibold">Nodes</h2>
        <NodeGrid />
      </div>
    </div>
  );
}
