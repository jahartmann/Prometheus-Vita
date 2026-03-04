"use client";

import { useState } from "react";
import {
  Download,
  Trash2,
  Eye,
  Plus,
  Clock,
  RotateCcw,
  HardDrive,
  FileText,
  Archive,
  CheckCircle,
  XCircle,
  RefreshCw,
  AlertCircle,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useBackupStore } from "@/stores/backup-store";
import { CreateBackupDialog } from "./create-backup-dialog";
import { BackupDetailDialog } from "./backup-detail-dialog";
import { BackupScheduleDialog } from "./backup-schedule-dialog";
import { RestoreDialog } from "./restore-dialog";
import { VzdumpDialog } from "./vzdump-dialog";
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

const statusIcon: Record<string, typeof Clock> = {
  pending: Clock,
  running: RefreshCw,
  completed: CheckCircle,
  failed: XCircle,
};

const typeLabel: Record<string, string> = {
  manual: "Manuell",
  scheduled: "Geplant",
  pre_update: "Vor Update",
};

interface BackupListProps {
  nodeId: string;
}

export function BackupList({ nodeId }: BackupListProps) {
  const { backups, isLoading } = useBackupStore();
  const [createOpen, setCreateOpen] = useState(false);
  const [scheduleOpen, setScheduleOpen] = useState(false);
  const [selectedBackup, setSelectedBackup] = useState<ConfigBackup | null>(null);
  const [restoreBackupId, setRestoreBackupId] = useState<string | null>(null);
  const [vzdumpOpen, setVzdumpOpen] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);

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

  const handleDelete = async (backupId: string) => {
    await useBackupStore.getState().deleteBackup(backupId);
    setDeleteConfirm(null);
  };

  const formatTimeAgo = (dateStr: string) => {
    const diff = Date.now() - new Date(dateStr).getTime();
    const minutes = Math.floor(diff / 60000);
    if (minutes < 60) return `vor ${minutes} Min.`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `vor ${hours} Std.`;
    const days = Math.floor(hours / 24);
    return `vor ${days} Tag(en)`;
  };

  const completedCount = backups.filter((b) => b.status === "completed").length;
  const totalSize = backups.reduce((acc, b) => acc + (b.total_size || 0), 0);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <p className="text-sm text-muted-foreground">
            {backups.length} Backup(s) | {completedCount} erfolgreich | {formatBytes(totalSize)}
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={() => setScheduleOpen(true)}>
            <Clock className="mr-2 h-4 w-4" />
            Zeitplan
          </Button>
          <Button variant="outline" size="sm" onClick={() => setVzdumpOpen(true)}>
            <HardDrive className="mr-2 h-4 w-4" />
            Vzdump
          </Button>
          <Button size="sm" onClick={() => setCreateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Backup erstellen
          </Button>
        </div>
      </div>

      {isLoading ? (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-20 animate-pulse rounded-lg bg-muted" />
          ))}
        </div>
      ) : backups.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-16">
            <Archive className="mb-4 h-12 w-12 text-muted-foreground" />
            <p className="text-lg font-medium">Noch keine Backups vorhanden</p>
            <p className="mt-1 text-sm text-muted-foreground">
              Erstellen Sie Ihr erstes Backup dieses Servers.
            </p>
            <Button className="mt-4" onClick={() => setCreateOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              Erstes Backup erstellen
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {backups.map((backup) => {
            const StatusIcon = statusIcon[backup.status] || Clock;
            return (
              <Card key={backup.id} className="transition-all hover:shadow-md">
                <CardContent className="p-0">
                  <div className="flex items-stretch">
                    {/* Status indicator bar */}
                    <div className={`w-1 rounded-l-lg shrink-0 ${
                      backup.status === "completed" ? "bg-green-500" :
                      backup.status === "failed" ? "bg-red-500" :
                      backup.status === "running" ? "bg-blue-500" : "bg-gray-300"
                    }`} />
                    <div className="flex-1 p-4">
                      <div className="flex items-start justify-between">
                        <div className="space-y-1.5">
                          <div className="flex items-center gap-2 flex-wrap">
                            <StatusIcon className={`h-4 w-4 shrink-0 ${
                              backup.status === "completed" ? "text-green-500" :
                              backup.status === "failed" ? "text-red-500" :
                              backup.status === "running" ? "text-blue-500 animate-spin" : "text-muted-foreground"
                            }`} />
                            <span className="font-semibold font-mono">v{backup.version}</span>
                            <Badge variant={statusVariant[backup.status] || "outline"}>
                              {statusLabel[backup.status] || backup.status}
                            </Badge>
                            <Badge variant="outline" className="text-xs">
                              {typeLabel[backup.backup_type] || backup.backup_type}
                            </Badge>
                          </div>
                          <div className="flex items-center gap-3 text-sm text-muted-foreground">
                            <span className="flex items-center gap-1">
                              <FileText className="h-3 w-3" />
                              {backup.file_count} Dateien
                            </span>
                            <span className="flex items-center gap-1">
                              <HardDrive className="h-3 w-3" />
                              {formatBytes(backup.total_size)}
                            </span>
                            <span className="flex items-center gap-1">
                              <Clock className="h-3 w-3" />
                              {new Date(backup.created_at).toLocaleString("de-DE")}
                            </span>
                            <span className="text-xs">
                              ({formatTimeAgo(backup.created_at)})
                            </span>
                          </div>
                          {backup.notes && (
                            <p className="text-xs text-muted-foreground bg-muted/50 rounded px-2 py-1 mt-1 inline-block">
                              {backup.notes}
                            </p>
                          )}
                          {backup.error_message && (
                            <p className="text-xs text-red-500 flex items-center gap-1 mt-1">
                              <AlertCircle className="h-3 w-3" />
                              {backup.error_message}
                            </p>
                          )}
                        </div>
                        <div className="flex gap-1 shrink-0 ml-4">
                          <Button
                            variant="ghost"
                            size="icon"
                            title="Details anzeigen"
                            onClick={() => setSelectedBackup(backup)}
                          >
                            <Eye className="h-4 w-4" />
                          </Button>
                          {backup.status === "completed" && (
                            <>
                              <Button
                                variant="ghost"
                                size="icon"
                                title="Wiederherstellen"
                                onClick={() => setRestoreBackupId(backup.id)}
                              >
                                <RotateCcw className="h-4 w-4" />
                              </Button>
                              <Button
                                variant="ghost"
                                size="icon"
                                title="Herunterladen"
                                onClick={() => handleDownload(backup.id)}
                              >
                                <Download className="h-4 w-4" />
                              </Button>
                            </>
                          )}
                          {deleteConfirm === backup.id ? (
                            <div className="flex items-center gap-1">
                              <Button
                                variant="destructive"
                                size="sm"
                                className="text-xs h-8"
                                onClick={() => handleDelete(backup.id)}
                              >
                                Ja
                              </Button>
                              <Button
                                variant="ghost"
                                size="sm"
                                className="text-xs h-8"
                                onClick={() => setDeleteConfirm(null)}
                              >
                                Nein
                              </Button>
                            </div>
                          ) : (
                            <Button
                              variant="ghost"
                              size="icon"
                              title="Loeschen"
                              onClick={() => setDeleteConfirm(backup.id)}
                            >
                              <Trash2 className="h-4 w-4 text-destructive" />
                            </Button>
                          )}
                        </div>
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            );
          })}
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
      {restoreBackupId && (
        <RestoreDialog
          backupId={restoreBackupId}
          open={!!restoreBackupId}
          onOpenChange={(open) => {
            if (!open) setRestoreBackupId(null);
          }}
        />
      )}
      <VzdumpDialog open={vzdumpOpen} onOpenChange={setVzdumpOpen} />
    </div>
  );
}
