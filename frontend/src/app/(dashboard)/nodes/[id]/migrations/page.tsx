"use client";

import { useEffect } from "react";
import { useParams } from "next/navigation";
import { useNodeStore } from "@/stores/node-store";
import { MigrationHistory } from "@/components/migration/migration-history";

export default function NodeMigrationsPage() {
  const params = useParams<{ id: string }>();
  const nodeId = params.id;
  const { nodes, fetchNodes } = useNodeStore();

  useEffect(() => {
    if (nodes.length === 0) {
      fetchNodes();
    }
  }, [nodes.length, fetchNodes]);

  const node = nodes.find((n) => n.id === nodeId);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">
          VM-Migrationen{node ? ` – ${node.name}` : ""}
        </h1>
        <p className="text-muted-foreground text-sm">
          Migrationen für diesen Node
        </p>
      </div>

      <MigrationHistory nodeId={nodeId} />
    </div>
  );
}
