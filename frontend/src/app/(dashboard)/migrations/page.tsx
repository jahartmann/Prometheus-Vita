"use client";

import { useEffect } from "react";
import { useNodeStore } from "@/stores/node-store";
import { MigrationHistory } from "@/components/migration/migration-history";

export default function MigrationsPage() {
  const { nodes, fetchNodes } = useNodeStore();

  useEffect(() => {
    if (nodes.length === 0) {
      fetchNodes();
    }
  }, [nodes.length, fetchNodes]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">VM-Migrationen</h1>
        <p className="text-muted-foreground text-sm">
          Uebersicht aller VM-Migrationen zwischen Nodes
        </p>
      </div>

      <MigrationHistory />
    </div>
  );
}
