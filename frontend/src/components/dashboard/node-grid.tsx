"use client";

import { useEffect } from "react";
import { useNodeStore } from "@/stores/node-store";
import { NodeCard } from "./node-card";
import { Skeleton } from "@/components/ui/skeleton";

export function NodeGrid() {
  const { nodes, nodeStatus, isLoading, fetchNodeStatus } = useNodeStore();

  useEffect(() => {
    nodes.forEach((node) => {
      if (node.is_online) {
        fetchNodeStatus(node.id);
      }
    });
  }, [nodes, fetchNodeStatus]);

  if (isLoading) {
    return (
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-64 rounded-xl" />
        ))}
      </div>
    );
  }

  if (nodes.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center rounded-xl border border-dashed py-12">
        <p className="text-muted-foreground">Keine Nodes konfiguriert.</p>
        <p className="mt-1 text-sm text-muted-foreground">
          Fuegen Sie Ihren ersten Proxmox Node unter Einstellungen hinzu.
        </p>
      </div>
    );
  }

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      {nodes.map((node) => (
        <NodeCard
          key={node.id}
          node={node}
          status={nodeStatus[node.id]}
        />
      ))}
    </div>
  );
}
