"use client";

import { useEffect, useCallback } from "react";
import { useMigrationStore } from "@/stores/migration-store";
import { useNodeStore } from "@/stores/node-store";
import { useWebSocket } from "@/hooks/use-websocket";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { MigrationProgress } from "./migration-progress";
import { Trash2 } from "lucide-react";
import type { VMMigration, MigrationStatus } from "@/types/api";

const STATUS_LABELS: Record<MigrationStatus, string> = {
  pending: "Wartend",
  preparing: "Vorbereitung",
  backing_up: "Backup",
  transferring: "Transfer",
  restoring: "Wiederherstellung",
  cleaning_up: "Aufraeumen",
  completed: "Abgeschlossen",
  failed: "Fehlgeschlagen",
  cancelled: "Abgebrochen",
};

const STATUS_VARIANTS: Record<
  MigrationStatus,
  "success" | "secondary" | "warning" | "destructive" | "outline"
> = {
  pending: "secondary",
  preparing: "outline",
  backing_up: "outline",
  transferring: "warning",
  restoring: "outline",
  cleaning_up: "outline",
  completed: "success",
  failed: "destructive",
  cancelled: "secondary",
};

interface MigrationHistoryProps {
  nodeId?: string;
}

function formatDuration(startedAt?: string, completedAt?: string): string {
  if (!startedAt) return "--";
  const start = new Date(startedAt).getTime();
  const end = completedAt ? new Date(completedAt).getTime() : Date.now();
  const seconds = Math.floor((end - start) / 1000);
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
  return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
}

export function MigrationHistory({ nodeId }: MigrationHistoryProps) {
  const {
    migrations,
    activeMigrations,
    fetchMigrations,
    fetchByNode,
    deleteMigration,
    updateMigrationProgress,
    addMigrationLog,
    isLoading,
  } = useMigrationStore();
  const { nodes } = useNodeStore();

  useEffect(() => {
    if (nodeId) {
      fetchByNode(nodeId);
    } else {
      fetchMigrations();
    }
  }, [nodeId, fetchMigrations, fetchByNode]);

  const handleMessage = useCallback(
    (data: unknown) => {
      const msg = data as { type?: string; data?: Record<string, unknown> };
      if (!msg?.type || !msg.data) return;

      if (msg.type === "migration_progress") {
        updateMigrationProgress(msg.data as unknown as VMMigration);
      } else if (msg.type === "migration_log") {
        const logData = msg.data as { migration_id: string; line: string; timestamp: string };
        if (logData.migration_id && logData.line) {
          addMigrationLog(logData.migration_id, logData.line, logData.timestamp || "");
        }
      }
    },
    [updateMigrationProgress, addMigrationLog]
  );

  useWebSocket({
    url: "/api/v1/ws",
    onMessage: handleMessage,
    enabled: activeMigrations.length > 0,
  });

  const getNodeName = (id: string) =>
    nodes.find((n) => n.id === id)?.name || id.substring(0, 8);

  const isActive = (status: MigrationStatus) =>
    !["completed", "failed", "cancelled"].includes(status);

  // Sort: active first, then by date
  const sorted = [...migrations].sort((a, b) => {
    const aActive = isActive(a.status) ? 0 : 1;
    const bActive = isActive(b.status) ? 0 : 1;
    if (aActive !== bActive) return aActive - bActive;
    return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
  });

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">Lade Migrationen...</p>;
  }

  if (migrations.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center rounded-xl border border-dashed py-12">
        <p className="text-muted-foreground">Keine Migrationen vorhanden.</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Active migrations with live progress */}
      {activeMigrations.length > 0 && (
        <div className="space-y-3">
          {activeMigrations.map((m) => (
            <div key={m.id} className="rounded-lg border p-4">
              <MigrationProgress migration={m} />
            </div>
          ))}
        </div>
      )}

      {/* History table */}
      <div className="rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>VM</TableHead>
              <TableHead>Source</TableHead>
              <TableHead>Target</TableHead>
              <TableHead>Modus</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Fortschritt</TableHead>
              <TableHead>Dauer</TableHead>
              <TableHead>Datum</TableHead>
              <TableHead></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {sorted.map((m) => (
              <TableRow key={m.id}>
                <TableCell className="font-medium">
                  {m.vmid}
                  {m.vm_name && (
                    <span className="text-muted-foreground ml-1 text-xs">
                      ({m.vm_name})
                    </span>
                  )}
                </TableCell>
                <TableCell>{getNodeName(m.source_node_id)}</TableCell>
                <TableCell>{getNodeName(m.target_node_id)}</TableCell>
                <TableCell>
                  <Badge variant="outline">{m.mode}</Badge>
                </TableCell>
                <TableCell>
                  <Badge variant={STATUS_VARIANTS[m.status]}>
                    {STATUS_LABELS[m.status]}
                  </Badge>
                </TableCell>
                <TableCell>
                  {isActive(m.status) ? (
                    <div className="w-20 h-1.5 rounded-full bg-muted overflow-hidden">
                      <div
                        className="h-full bg-primary rounded-full"
                        style={{ width: `${m.progress}%` }}
                      />
                    </div>
                  ) : (
                    <span className="text-xs text-muted-foreground">
                      {m.progress}%
                    </span>
                  )}
                </TableCell>
                <TableCell className="text-xs text-muted-foreground">
                  {formatDuration(m.started_at, m.completed_at)}
                </TableCell>
                <TableCell className="text-xs text-muted-foreground">
                  {new Date(m.created_at).toLocaleDateString("de-DE")}
                </TableCell>
                <TableCell>
                  {!isActive(m.status) && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => deleteMigration(m.id)}
                    >
                      <Trash2 className="h-3 w-3" />
                    </Button>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
