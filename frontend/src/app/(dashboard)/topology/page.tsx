"use client";

import { ReactFlowProvider } from "@xyflow/react";
import { TopologyMap } from "@/components/topology/topology-map";

export default function TopologyPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Cluster-Topologie</h1>
        <p className="text-muted-foreground">
          Interaktive Visualisierung der Cluster-Infrastruktur mit Hosts, VMs, Storage und Netzwerk.
        </p>
      </div>

      <ReactFlowProvider>
        <TopologyMap />
      </ReactFlowProvider>
    </div>
  );
}
