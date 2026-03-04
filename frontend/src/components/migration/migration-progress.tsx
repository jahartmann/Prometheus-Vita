"use client";

import { useCallback } from "react";
import { useWebSocket } from "@/hooks/use-websocket";
import { useMigrationStore } from "@/stores/migration-store";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatBytes } from "@/lib/utils";
import { X, Loader2 } from "lucide-react";
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
  const { cancelMigration } = useMigrationStore();

  const isActive = ![
    "completed",
    "failed",
    "cancelled",
  ].includes(migration.status);

  const bytesRemaining = totalSize
    ? totalSize - migration.transfer_bytes_sent
    : 0;

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between text-sm">
        <div className="flex items-center gap-2">
          <span className="font-medium">
            VM {migration.vmid}
            {migration.vm_name && ` (${migration.vm_name})`}
          </span>
          <Badge variant={STATUS_VARIANTS[migration.status]}>
            {STATUS_LABELS[migration.status]}
          </Badge>
        </div>
        <span className="text-muted-foreground">{migration.progress}%</span>
      </div>

      {/* Progress bar */}
      <div className="h-2 rounded-full bg-muted overflow-hidden">
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

      {isActive && (
        <div className="flex justify-end">
          <Button
            variant="outline"
            size="sm"
            onClick={() => cancelMigration(migration.id)}
          >
            <X className="mr-1 h-3 w-3" />
            Abbrechen
          </Button>
        </div>
      )}

      {migration.error_message && (
        <p className="text-xs text-destructive">{migration.error_message}</p>
      )}
    </div>
  );
}

export function ActiveMigrations() {
  const { activeMigrations, updateMigrationProgress } = useMigrationStore();

  const handleMessage = useCallback(
    (data: unknown) => {
      const msg = data as { type?: string; data?: VMMigration };
      if (msg?.type === "migration_progress" && msg.data) {
        updateMigrationProgress(msg.data);
      }
    },
    [updateMigrationProgress]
  );

  useWebSocket({
    url: "/api/v1/ws",
    onMessage: handleMessage,
    enabled: activeMigrations.length > 0,
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
      <CardContent className="space-y-4">
        {activeMigrations.map((m) => (
          <MigrationProgress key={m.id} migration={m} />
        ))}
      </CardContent>
    </Card>
  );
}
