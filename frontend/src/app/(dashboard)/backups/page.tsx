"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import {
  Archive,
  CheckCircle,
  XCircle,
  Clock,
  Plus,
  Download,
  Trash2,
  Eye,
  RotateCcw,
  HardDrive,
} from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useNodeStore } from "@/stores/node-store";
import { backupApi, scheduleApi, nodeApi, toArray } from "@/lib/api";
import { formatBytes } from "@/lib/utils";
import type { ConfigBackup, BackupSchedule, Node } from "@/types/api";
import { BackupDetailDialog } from "@/components/backup/backup-detail-dialog";
import { RestoreDialog } from "@/components/backup/restore-dialog";
import { VzdumpDialog } from "@/components/backup/vzdump-dialog";
import { ScheduleForm } from "@/components/backup/schedule-form";

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

export default function BackupsPage() {
  const { nodes, fetchNodes } = useNodeStore();
  const [backups, setBackups] = useState<ConfigBackup[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [selectedBackup, setSelectedBackup] = useState<ConfigBackup | null>(null);
  const [restoreBackupId, setRestoreBackupId] = useState<string | null>(null);
  const [vzdumpOpen, setVzdumpOpen] = useState(false);

  // Create backup state
  const [createNodeId, setCreateNodeId] = useState("");
  const [createNotes, setCreateNotes] = useState("");
  const [isCreating, setIsCreating] = useState(false);

  // Schedules state
  const [allSchedules, setAllSchedules] = useState<(BackupSchedule & { _nodeId: string })[]>([]);
  const [schedulesLoading, setSchedulesLoading] = useState(false);
  const [scheduleNodeId, setScheduleNodeId] = useState("");
  const [isCreatingSchedule, setIsCreatingSchedule] = useState(false);

  useEffect(() => {
    fetchNodes();
  }, [fetchNodes]);

  const loadBackups = async () => {
    setIsLoading(true);
    try {
      const response = await backupApi.listAll();
      const data = toArray<ConfigBackup>(response.data);
      setBackups(data);
    } catch {
      setBackups([]);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadBackups();
  }, []);

  const loadAllSchedules = async () => {
    if (nodes.length === 0) return;
    setSchedulesLoading(true);
    try {
      const results = await Promise.all(
        nodes.map(async (n) => {
          try {
            const res = await scheduleApi.listSchedules(n.id);
            return toArray<BackupSchedule>(res.data).map((s) => ({ ...s, _nodeId: n.id }));
          } catch {
            return [];
          }
        })
      );
      setAllSchedules(results.flat());
    } catch {
      setAllSchedules([]);
    } finally {
      setSchedulesLoading(false);
    }
  };

  useEffect(() => {
    if (nodes.length > 0) {
      loadAllSchedules();
    }
  }, [nodes]);

  const completedCount = backups.filter((b) => b.status === "completed").length;
  const failedCount = backups.filter((b) => b.status === "failed").length;

  const getNodeName = (nodeId: string) => {
    const node = nodes.find((n) => n.id === nodeId);
    return node?.name ?? nodeId.slice(0, 8);
  };

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
    try {
      await backupApi.deleteBackup(backupId);
      setBackups((prev) => prev.filter((b) => b.id !== backupId));
    } catch {
      /* ignore */
    }
  };

  const handleCreateBackup = async () => {
    if (!createNodeId) return;
    setIsCreating(true);
    try {
      await backupApi.createBackup(createNodeId, {
        backup_type: "manual",
        notes: createNotes || undefined,
      });
      setCreateNotes("");
      setCreateNodeId("");
      loadBackups();
    } catch {
      /* ignore */
    }
    setIsCreating(false);
  };

  const handleCreateSchedule = async (cronExpression: string, retentionDays: number) => {
    if (!scheduleNodeId) return;
    setIsCreatingSchedule(true);
    try {
      await scheduleApi.createSchedule(scheduleNodeId, {
        cron_expression: cronExpression,
        is_active: true,
        retention_count: retentionDays,
      });
      loadAllSchedules();
    } catch {
      /* ignore */
    }
    setIsCreatingSchedule(false);
  };

  const handleDeleteSchedule = async (id: string) => {
    try {
      await scheduleApi.deleteSchedule(id);
      loadAllSchedules();
    } catch {
      /* ignore */
    }
  };

  const handleToggleSchedule = async (schedule: BackupSchedule & { _nodeId: string }) => {
    try {
      await scheduleApi.updateSchedule(schedule.id, { is_active: !schedule.is_active });
      loadAllSchedules();
    } catch {
      /* ignore */
    }
  };

  const onlineNodes = nodes.filter((n) => n.is_online);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Backups</h1>
          <p className="text-muted-foreground">
            Zentrale Verwaltung aller Backups, Zeitplaene und Vzdump-Sicherungen.
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => setVzdumpOpen(true)}>
            <HardDrive className="mr-2 h-4 w-4" />
            Vzdump
          </Button>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardContent className="flex items-center gap-4 p-4">
            <Archive className="h-8 w-8 text-muted-foreground" />
            <div>
              <p className="text-2xl font-bold">{backups.length}</p>
              <p className="text-sm text-muted-foreground">Backups gesamt</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-4 p-4">
            <CheckCircle className="h-8 w-8 text-green-500" />
            <div>
              <p className="text-2xl font-bold">{completedCount}</p>
              <p className="text-sm text-muted-foreground">Erfolgreich</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-4 p-4">
            <XCircle className="h-8 w-8 text-red-500" />
            <div>
              <p className="text-2xl font-bold">{failedCount}</p>
              <p className="text-sm text-muted-foreground">Fehlgeschlagen</p>
            </div>
          </CardContent>
        </Card>
      </div>

      <Tabs defaultValue="backups">
        <TabsList>
          <TabsTrigger value="backups">Konfig-Backups</TabsTrigger>
          <TabsTrigger value="schedules">Zeitplaene</TabsTrigger>
          <TabsTrigger value="create">Backup erstellen</TabsTrigger>
        </TabsList>

        {/* Backups Tab */}
        <TabsContent value="backups" className="space-y-4">
          {isLoading ? (
            <div className="space-y-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-16 w-full" />
              ))}
            </div>
          ) : backups.length === 0 ? (
            <Card>
              <CardContent className="flex flex-col items-center justify-center py-12">
                <Archive className="mb-3 h-10 w-10 text-muted-foreground" />
                <p className="text-muted-foreground">Noch keine Backups vorhanden.</p>
              </CardContent>
            </Card>
          ) : (
            <div className="space-y-2">
              {backups.map((backup) => (
                <Card key={backup.id}>
                  <CardContent className="flex items-center justify-between p-4">
                    <div className="flex items-center gap-4">
                      <Clock className="h-5 w-5 text-muted-foreground shrink-0" />
                      <div>
                        <div className="flex items-center gap-2 flex-wrap">
                          <Link
                            href={`/nodes/${backup.node_id}`}
                            className="font-medium hover:underline"
                          >
                            {getNodeName(backup.node_id)}
                          </Link>
                          <span className="text-muted-foreground">v{backup.version}</span>
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
                    <div className="flex gap-1 shrink-0">
                      <Button
                        variant="ghost"
                        size="icon"
                        title="Details"
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
                      <Button
                        variant="ghost"
                        size="icon"
                        title="Loeschen"
                        onClick={() => handleDelete(backup.id)}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </TabsContent>

        {/* Schedules Tab */}
        <TabsContent value="schedules" className="space-y-4">
          {/* Existing schedules */}
          {schedulesLoading ? (
            <div className="space-y-3">
              {Array.from({ length: 2 }).map((_, i) => (
                <Skeleton key={i} className="h-16 w-full" />
              ))}
            </div>
          ) : allSchedules.length === 0 ? (
            <Card>
              <CardContent className="flex flex-col items-center justify-center py-12">
                <Clock className="mb-3 h-10 w-10 text-muted-foreground" />
                <p className="text-muted-foreground">Keine Zeitplaene vorhanden.</p>
              </CardContent>
            </Card>
          ) : (
            <div className="space-y-2">
              {allSchedules.map((s) => (
                <Card key={s.id}>
                  <CardContent className="flex items-center justify-between p-4">
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-medium">{getNodeName(s._nodeId)}</span>
                        <code className="text-sm bg-muted px-2 py-0.5 rounded">
                          {s.cron_expression}
                        </code>
                        <Badge variant={s.is_active ? "success" : "outline"}>
                          {s.is_active ? "Aktiv" : "Inaktiv"}
                        </Badge>
                      </div>
                      <p className="text-xs text-muted-foreground mt-1">
                        Aufbewahrung: {s.retention_count} |{" "}
                        {s.next_run_at
                          ? `Naechster Lauf: ${new Date(s.next_run_at).toLocaleString("de-DE")}`
                          : ""}
                      </p>
                    </div>
                    <div className="flex gap-1">
                      <Button variant="ghost" size="sm" onClick={() => handleToggleSchedule(s)}>
                        {s.is_active ? "Pause" : "Start"}
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleDeleteSchedule(s.id)}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}

          {/* Create new schedule */}
          <Card>
            <CardContent className="p-4 space-y-4">
              <h3 className="text-sm font-medium">Neuen Zeitplan erstellen</h3>
              <div className="space-y-2">
                <Select value={scheduleNodeId} onValueChange={setScheduleNodeId}>
                  <SelectTrigger>
                    <SelectValue placeholder="Node waehlen..." />
                  </SelectTrigger>
                  <SelectContent>
                    {onlineNodes.map((n) => (
                      <SelectItem key={n.id} value={n.id}>
                        {n.name} ({n.hostname})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              {scheduleNodeId && (
                <ScheduleForm onSubmit={handleCreateSchedule} isSubmitting={isCreatingSchedule} />
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Create Backup Tab */}
        <TabsContent value="create" className="space-y-4">
          <Card>
            <CardContent className="p-4 space-y-4">
              <h3 className="text-sm font-medium">Konfigurations-Backup erstellen</h3>
              <div className="space-y-2">
                <Select value={createNodeId} onValueChange={setCreateNodeId}>
                  <SelectTrigger>
                    <SelectValue placeholder="Node waehlen..." />
                  </SelectTrigger>
                  <SelectContent>
                    {onlineNodes.map((n) => (
                      <SelectItem key={n.id} value={n.id}>
                        {n.name} ({n.hostname})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <textarea
                  className="w-full rounded-md border bg-background px-3 py-2 text-sm"
                  rows={3}
                  value={createNotes}
                  onChange={(e) => setCreateNotes(e.target.value)}
                  placeholder="Notizen (optional), z.B. Vor Kernel-Update..."
                />
              </div>
              <Button onClick={handleCreateBackup} disabled={isCreating || !createNodeId}>
                {isCreating ? "Erstelle..." : "Backup erstellen"}
              </Button>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Dialogs */}
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
