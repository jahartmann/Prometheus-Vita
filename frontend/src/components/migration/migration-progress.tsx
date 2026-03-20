"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useWebSocket } from "@/hooks/use-websocket";
import { useMigrationStore } from "@/stores/migration-store";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatBytes } from "@/lib/utils";
import {
  X,
  Loader2,
  ChevronDown,
  ChevronUp,
  Terminal,
  CheckCircle2,
  XCircle,
  RotateCw,
} from "lucide-react";
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

function formatSpeed(bps: number): string {
  if (bps <= 0) return "--";
  return `${formatBytes(bps)}/s`;
}

function formatETA(bytesRemaining: number, speedBps: number): string {
  if (speedBps <= 0 || bytesRemaining <= 0) return "--";
  const seconds = bytesRemaining / speedBps;
  if (seconds < 60) return `${Math.ceil(seconds)}s`;
  if (seconds < 3600) return `${Math.ceil(seconds / 60)}min`;
  return `${(seconds / 3600).toFixed(1)}h`;
}

interface MigrationProgressProps {
  migration: VMMigration;
  totalSize?: number;
}

export function MigrationProgress({
  migration,
  totalSize,
}: MigrationProgressProps) {
  const { cancelMigration, retryMigration, migrationLogs, loadMigrationLogs } = useMigrationStore();
  const [showLogs, setShowLogs] = useState(true);
  const logRef = useRef<HTMLDivElement>(null);

  const isActive = ![
    "completed",
    "failed",
    "cancelled",
  ].includes(migration.status);

  const bytesRemaining = totalSize
    ? totalSize - migration.transfer_bytes_sent
    : 0;

  const logs = migrationLogs[migration.id] || [];

  // Auto-load persisted logs for terminal migrations that have no in-memory logs
  useEffect(() => {
    if (
      ["completed", "failed", "cancelled"].includes(migration.status) &&
      logs.length === 0
    ) {
      loadMigrationLogs(migration.id);
    }
  }, [migration.id, migration.status, logs.length, loadMigrationLogs]);

  // Auto-scroll log to bottom
  useEffect(() => {
    if (logRef.current && showLogs) {
      logRef.current.scrollTop = logRef.current.scrollHeight;
    }
  }, [logs.length, showLogs]);

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between text-sm">
        <div className="flex items-center gap-2">
          {migration.status === "completed" ? (
            <CheckCircle2 className="h-4 w-4 text-green-500" />
          ) : migration.status === "failed" ? (
            <XCircle className="h-4 w-4 text-destructive" />
          ) : isActive ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : null}
          <span className="font-medium">
            VM {migration.vmid}
            {migration.vm_name && ` (${migration.vm_name})`}
          </span>
          <Badge variant={STATUS_VARIANTS[migration.status]}>
            {STATUS_LABELS[migration.status]}
          </Badge>
        </div>
        <span className="text-muted-foreground font-mono">{migration.progress}%</span>
      </div>

      {/* Progress bar */}
      <div className="h-2.5 rounded-full bg-muted overflow-hidden">
        <div
          className={`h-full rounded-full transition-all duration-500 ${
            migration.status === "failed"
              ? "bg-destructive"
              : migration.status === "completed"
              ? "bg-green-500"
              : "bg-primary"
          }`}
          style={{ width: `${migration.progress}%` }}
        />
      </div>

      <div className="flex items-center justify-between text-xs text-muted-foreground">
        <span>{migration.current_step}</span>
        {migration.status === "transferring" && (
          <div className="flex items-center gap-3">
            <span>{formatBytes(migration.transfer_bytes_sent)} uebertragen</span>
            <span>{formatSpeed(migration.transfer_speed_bps)}</span>
            {totalSize && totalSize > 0 && (
              <span>
                ETA: {formatETA(bytesRemaining, migration.transfer_speed_bps)}
              </span>
            )}
          </div>
        )}
      </div>

      {/* Live Log Output */}
      {logs.length > 0 && (
        <div className="space-y-1">
          <button
            onClick={() => setShowLogs(!showLogs)}
            className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            <Terminal className="h-3 w-3" />
            <span>Log ({logs.length} Einträge)</span>
            {showLogs ? (
              <ChevronUp className="h-3 w-3" />
            ) : (
              <ChevronDown className="h-3 w-3" />
            )}
          </button>

          {showLogs && (
            <div
              ref={logRef}
              className="max-h-48 overflow-y-auto rounded-md border bg-black/90 p-3 font-mono text-xs leading-relaxed"
            >
              {logs.map((log, i) => (
                <div key={i} className="flex gap-2">
                  <span className="text-muted-foreground shrink-0 select-none">
                    {log.timestamp}
                  </span>
                  <span
                    className={
                      log.line.startsWith("✓")
                        ? "text-green-400"
                        : log.line.startsWith("[PVE]")
                        ? "text-blue-400"
                        : log.line.includes("fehlgeschlagen") || log.line.includes("ERROR")
                        ? "text-red-400"
                        : "text-gray-300"
                    }
                  >
                    {log.line}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {migration.error_message && (
        <div className="text-sm text-destructive bg-destructive/10 rounded p-2 mt-2">
          <span className="font-medium">Fehler:</span> {migration.error_message}
        </div>
      )}

      <div className="flex items-center justify-between">

        <div className="flex gap-2 ml-auto">
          {migration.status === "failed" && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => retryMigration(migration)}
            >
              <RotateCw className="mr-1 h-3 w-3" />
              Wiederholen
            </Button>
          )}
          {isActive && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => cancelMigration(migration.id)}
            >
              <X className="mr-1 h-3 w-3" />
              Abbrechen
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}

export function ActiveMigrations() {
  const { activeMigrations, updateMigrationProgress, addMigrationLog } =
    useMigrationStore();

  const handleMessage = useCallback(
    (data: unknown) => {
      const msg = data as {
        type?: string;
        data?: VMMigration & { migration_id?: string; line?: string; timestamp?: string };
      };
      if (!msg?.type || !msg.data) return;

      if (msg.type === "migration_progress") {
        updateMigrationProgress(msg.data as VMMigration);
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
    enabled: true,
  });

  if (activeMigrations.length === 0) return null;

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm flex items-center gap-2">
          <Loader2 className="h-4 w-4 animate-spin" />
          Aktive Migrationen ({activeMigrations.length})
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        {activeMigrations.map((m) => (
          <MigrationProgress key={m.id} migration={m} />
        ))}
      </CardContent>
    </Card>
  );
}
