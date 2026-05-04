"use client";

import Link from "next/link";
import { Server } from "lucide-react";
import { OpsPanel, OpsPanelContent, OpsPanelHeader, OpsPanelTitle } from "@/components/ops/ops-panel";
import { StatusIndicator } from "@/components/ops/status-indicator";
import { Skeleton } from "@/components/ui/skeleton";
import { formatBytes, formatPercentage } from "@/lib/utils";
import type { Node, NodeStatus } from "@/types/api";

interface NodeFleetTableProps {
  nodes: Node[];
  nodeStatus: Record<string, NodeStatus | undefined>;
  isLoading: boolean;
}

export function NodeFleetTable({ nodes, nodeStatus, isLoading }: NodeFleetTableProps) {
  if (isLoading) {
    return (
      <OpsPanel>
        <OpsPanelHeader>
          <OpsPanelTitle>Server-Flotte</OpsPanelTitle>
        </OpsPanelHeader>
        <OpsPanelContent className="space-y-2">
          {Array.from({ length: 4 }).map((_, index) => (
            <Skeleton key={index} className="h-12 rounded-md" />
          ))}
        </OpsPanelContent>
      </OpsPanel>
    );
  }

  return (
    <OpsPanel>
      <OpsPanelHeader className="flex-row items-center justify-between">
        <div>
          <OpsPanelTitle>Server-Flotte</OpsPanelTitle>
          <p className="mt-1 text-xs text-muted-foreground">
            Kompakte Übersicht. Details bleiben per Klick auf die Node erreichbar.
          </p>
        </div>
        <Link href="/nodes" className="text-xs font-medium text-primary hover:underline">
          Alle Nodes
        </Link>
      </OpsPanelHeader>
      <OpsPanelContent className="space-y-2">
        {nodes.length === 0 ? (
          <Link
            href="/settings/nodes"
            className="ops-row ops-focus-ring flex items-center justify-between px-3 py-3 text-sm hover:bg-accent/60"
          >
            <span>Keine Nodes konfiguriert.</span>
            <span className="text-primary">Node hinzufügen</span>
          </Link>
        ) : (
          nodes.map((node) => {
            const status = nodeStatus[node.id];
            const memUsage =
              status && status.memory_total > 0
                ? (status.memory_used / status.memory_total) * 100
                : 0;
            const diskUsage =
              status && status.disk_total > 0
                ? (status.disk_used / status.disk_total) * 100
                : 0;

            return (
              <Link
                key={node.id}
                href={`/nodes/${node.id}`}
                className="ops-row ops-focus-ring grid gap-3 px-3 py-2.5 transition-colors hover:bg-accent/60 md:grid-cols-[1.3fr_repeat(4,minmax(0,1fr))]"
              >
                <div className="flex min-w-0 items-center gap-2">
                  <Server className="h-4 w-4 shrink-0 text-muted-foreground" />
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium">{node.name}</p>
                    <p className="truncate text-xs text-muted-foreground">
                      {node.hostname}:{node.port}
                    </p>
                  </div>
                </div>
                <StatusIndicator
                  tone={node.is_online ? "ok" : "critical"}
                  label={node.is_online ? "Online" : "Offline"}
                />
                <FleetMetric label="CPU" value={status ? formatPercentage(status.cpu_usage) : "-"} />
                <FleetMetric label="RAM" value={status ? formatPercentage(memUsage) : "-"} />
                <FleetMetric
                  label="Disk"
                  value={status ? formatPercentage(diskUsage) : "-"}
                  helper={status ? formatBytes(status.disk_total) : undefined}
                />
              </Link>
            );
          })
        )}
      </OpsPanelContent>
    </OpsPanel>
  );
}

function FleetMetric({
  label,
  value,
  helper,
}: {
  label: string;
  value: string;
  helper?: string;
}) {
  return (
    <div className="min-w-0">
      <p className="text-[11px] uppercase tracking-wide text-muted-foreground">{label}</p>
      <p className="truncate text-sm font-medium tabular">{value}</p>
      {helper && <p className="truncate text-[11px] text-muted-foreground">{helper}</p>}
    </div>
  );
}
