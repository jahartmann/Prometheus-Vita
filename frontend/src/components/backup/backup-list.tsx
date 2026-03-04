"use client";

import { useState } from "react";
import { Download, Trash2, Eye, Plus, Clock } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useBackupStore } from "@/stores/backup-store";
import { CreateBackupDialog } from "./create-backup-dialog";
import { BackupDetailDialog } from "./backup-detail-dialog";
import { BackupScheduleDialog } from "./backup-schedule-dialog";
import { backupApi } from "@/lib/api";
import { formatBytes } from "@/lib/utils";
import type { ConfigBackup } from "@/types/api";

const statusVariant: Record<string, "default" | "success" | "destructive" | "outline"> = {
  pending: "outline",
  running: "default",
  completed: "success",
  failed: "destructive",
};

const statusLabel: Record<string, string> = {
  pending: "Ausstehend",
  running: "Laeuft",
  completed: "Abgeschlossen",
  failed: "Fehlgeschlagen",
};

interface BackupListProps {
  nodeId: string;
}

export function BackupList({ nodeId }: BackupListProps) {
  const { backups, isLoading } = useBackupStore();
  const [createOpen, setCreateOpen] = useState(false);
  const [scheduleOpen, setScheduleOpen] = useState(false);
  const [selectedBackup, setSelectedBackup] = useState<ConfigBackup | null>(null);

  const handleDownload = async (backupId: string) => {
    try {
      const response = await backupApi.downloadBackup(backupId);
      const url = window.URL.createObjectURL(new Blob([response.data]));
      const a = document.createElement("a");
      a.href = url;
      a.download = `backup-${backupId}.tar.gz`;
      a.click();
      window.URL.revokeObjectURL(url);
    } catch {
      /* ignore */
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          {backups.length} Backup(s) vorhanden
        </p>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => setScheduleOpen(true)}>
            <Clock className="mr-2 h-4 w-4" />
            Zeitplan
          </Button>
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Backup erstellen
          </Button>
        </div>
      </div>

      {isLoading ? (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-16 animate-pulse rounded-lg bg-muted" />
          ))}
        </div>
      ) : backups.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <p className="text-muted-foreground">Noch keine Backups vorhanden.</p>
            <Button variant="outline" className="mt-4" onClick={() => setCreateOpen(true)}>
              Erstes Backup erstellen
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-2">
          {backups.map((backup) => (
            <Card key={backup.id}>
              <CardContent className="flex items-center justify-between p-4">
                <div className="flex items-center gap-4">
                  <div>
                    <div className="flex items-center gap-2">
                      <p className="font-medium">v{backup.version}</p>
                      <Badge variant={statusVariant[backup.status] || "outline"}>
                        {statusLabel[backup.status] || backup.status}
                      </Badge>
                      <Badge variant="outline">{backup.backup_type}</Badge>
                    </div>
                    <p className="text-sm text-muted-foreground">
                      {backup.file_count} Dateien | {formatBytes(backup.total_size)} |{" "}
                      {new Date(backup.created_at).toLocaleString("de-DE")}
                    </p>
                    {backup.notes && (
                      <p className="text-xs text-muted-foreground mt-1">{backup.notes}</p>
                    )}
                  </div>
                </div>
                <div className="flex gap-1">
                  <Button variant="ghost" size="icon" onClick={() => setSelectedBackup(backup)}>
                    <Eye className="h-4 w-4" />
                  </Button>
                  {backup.status === "completed" && (
                    <Button variant="ghost" size="icon" onClick={() => handleDownload(backup.id)}>
                      <Download className="h-4 w-4" />
                    </Button>
                  )}
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => {
                      useBackupStore.getState().deleteBackup(backup.id);
                    }}
                  >
                    <Trash2 className="h-4 w-4 text-destructive" />
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <CreateBackupDialog nodeId={nodeId} open={createOpen} onOpenChange={setCreateOpen} />
      <BackupScheduleDialog nodeId={nodeId} open={scheduleOpen} onOpenChange={setScheduleOpen} />
      {selectedBackup && (
        <BackupDetailDialog
          backup={selectedBackup}
          open={!!selectedBackup}
          onOpenChange={(open) => {
            if (!open) setSelectedBackup(null);
          }}
        />
      )}
    </div>
  );
}
