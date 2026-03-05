"use client";

import { useEffect } from "react";
import { GitCompare, FileWarning, AlertTriangle, CheckCircle } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { useDriftStore } from "@/stores/drift-store";
import { useNodeStore } from "@/stores/node-store";
import { DriftList } from "@/components/drift/drift-list";

export default function DriftPage() {
  const { checks, isLoading, fetchAll } = useDriftStore();
  const { nodes, fetchNodes } = useNodeStore();

  useEffect(() => {
    fetchAll();
    fetchNodes();
  }, [fetchAll, fetchNodes]);

  const nodeNames: Record<string, string> = {};
  for (const node of nodes) {
    nodeNames[node.id] = node.name;
  }

  const completedChecks = checks.filter((c) => c.status === "completed");
  const withDrift = completedChecks.filter(
    (c) => c.changed_files + c.added_files + c.removed_files > 0
  );
  const failedChecks = checks.filter((c) => c.status === "failed");

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Drift Detection</h1>
        <p className="text-muted-foreground">
          Konfigurationsabweichungen zwischen Backups und aktuellem Zustand.
        </p>
      </div>

      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardContent className="flex items-center gap-4 p-4">
            <GitCompare className="h-8 w-8 text-blue-500" />
            <div>
              <p className="text-2xl font-bold">{checks.length}</p>
              <p className="text-sm text-muted-foreground">Checks gesamt</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-4 p-4">
            <CheckCircle className="h-8 w-8 text-green-500" />
            <div>
              <p className="text-2xl font-bold">{completedChecks.length - withDrift.length}</p>
              <p className="text-sm text-muted-foreground">Ohne Drift</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-4 p-4">
            <AlertTriangle className="h-8 w-8 text-yellow-500" />
            <div>
              <p className="text-2xl font-bold">{withDrift.length}</p>
              <p className="text-sm text-muted-foreground">Mit Drift</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-4 p-4">
            <FileWarning className="h-8 w-8 text-red-500" />
            <div>
              <p className="text-2xl font-bold">{failedChecks.length}</p>
              <p className="text-sm text-muted-foreground">Fehlgeschlagen</p>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardContent className="p-0">
          <DriftList checks={checks} nodeNames={nodeNames} isLoading={isLoading} />
        </CardContent>
      </Card>
    </div>
  );
}
