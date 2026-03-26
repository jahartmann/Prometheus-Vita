"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import {
  Archive,
  CheckCircle,
  XCircle,
  Clock,
  Download,
  Trash2,
  Eye,
  RotateCcw,
  HardDrive,
  Shield,
  Calendar,
  FileText,
  Server,
  AlertCircle,
  RefreshCw,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
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
import { Label } from "@/components/ui/label";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { KpiCard } from "@/components/ui/kpi-card";
import { useNodeStore } from "@/stores/node-store";
import { backupApi, scheduleApi, toArray } from "@/lib/api";
import { formatBytes } from "@/lib/utils";
import type { ConfigBackup, BackupSchedule } from "@/types/api";
import { BackupDetailDialog } from "@/components/backup/backup-detail-dialog";
import { RestoreDialog } from "@/components/backup/restore-dialog";
import { VzdumpDialog } from "@/components/backup/vzdump-dialog";
import { ScheduleForm } from "@/components/backup/schedule-form";
import { toast } from "sonner";

const statusVariant: Record<string, "default" | "success" | "destructive" | "outline"> = {
  pending: "outline",
  running: "default",
  completed: "success",
  failed: "destructive",
};

const statusLabel: Record<string, string> = {
  pending: "Ausstehend",
  running: "Läuft",
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

export default function BackupsPage() {
  const { nodes, fetchNodes } = useNodeStore();
  const [backups, setBackups] = useState<ConfigBackup[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [selectedBackup, setSelectedBackup] = useState<ConfigBackup | null>(null);
  const [restoreBackupId, setRestoreBackupId] = useState<string | null>(null);
  const [vzdumpOpen, setVzdumpOpen] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);

  // Create backup state
  const [createNodeId, setCreateNodeId] = useState("");
  const [createNotes, setCreateNotes] = useState("");
  const [isCreating, setIsCreating] = useState(false);

  // Schedules state
  const [allSchedules, setAllSchedules] = useState<(BackupSchedule & { _nodeId: string })[]>([]);
  const [schedulesLoading, setSchedulesLoading] = useState(false);
  const [scheduleNodeId, setScheduleNodeId] = useState("");
  const [isCreatingSchedule, setIsCreatingSchedule] = useState(false);
  const [activeTab, setActiveTab] = useState("backups");

  useEffect(() => {
    fetchNodes();
  }, [fetchNodes]);

  const loadBackups = async () => {
    setIsLoading(true);
    try {
      const response = await backupApi.listAll();
      const data = toArray<ConfigBackup>(response.data);
      setBackups(data);
      setLoadError(null);
    } catch (e) {
      console.error('Failed to load backups:', e);
      const message = e instanceof Error ? e.message : "Fehler beim Laden der Backups";
      setLoadError(message);
      setBackups([]);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadBackups();
  }, []);

  // Auto-refresh when backups are running or pending
  useEffect(() => {
    const hasRunning = backups.some(b => b.status === 'running' || b.status === 'pending');
    if (!hasRunning) return;
    const timer = setInterval(loadBackups, 5000);
    return () => clearInterval(timer);
  }, [backups]);

  const loadAllSchedules = async () => {
    if (nodes.length === 0) return;
    setSchedulesLoading(true);
    try {
      const results = await Promise.all(
        nodes.map(async (n) => {
          try {
            const res = await scheduleApi.listSchedules(n.id);
            return toArray<BackupSchedule>(res.data).map((s) => ({ ...s, _nodeId: n.id }));
          } catch (e) {
            console.error(`Failed to load schedules for node ${n.id}:`, e);
            return [];
          }
        })
      );
      setAllSchedules(results.flat());
    } catch (e) {
      console.error('Failed to load schedules:', e);
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
  const totalSize = backups.reduce((acc, b) => acc + (b.total_size || 0), 0);
  const activeSchedules = allSchedules.filter((s) => s.is_active).length;

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
    } catch (e) {
      console.error('Failed to download backup:', e);
    }
  };

  const handleDelete = async (backupId: string) => {
    try {
      await backupApi.deleteBackup(backupId);
      setBackups((prev) => prev.filter((b) => b.id !== backupId));
      setDeleteConfirm(null);
      toast.success("Backup gelöscht");
    } catch (e) {
      console.error('Failed to delete backup:', e);
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
      toast.success("Backup wird erstellt", {
        description: "Der Fortschritt wird in der Liste angezeigt.",
      });
      setCreateNotes("");
      setCreateNodeId("");
      setActiveTab("backups");
      loadBackups();
    } catch (e) {
      console.error('Failed to create backup:', e);
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
      toast.success("Backup-Zeitplan erstellt");
      loadAllSchedules();
    } catch (e) {
      console.error('Failed to create schedule:', e);
    }
    setIsCreatingSchedule(false);
  };

  const handleDeleteSchedule = async (id: string) => {
    try {
      await scheduleApi.deleteSchedule(id);
      toast.success("Zeitplan gelöscht");
      loadAllSchedules();
    } catch (e) {
      console.error('Failed to delete schedule:', e);
    }
  };

  const handleToggleSchedule = async (schedule: BackupSchedule & { _nodeId: string }) => {
    try {
      await scheduleApi.updateSchedule(schedule.id, { is_active: !schedule.is_active });
      toast.success(schedule.is_active ? "Zeitplan pausiert" : "Zeitplan aktiviert");
      loadAllSchedules();
    } catch (e) {
      console.error('Failed to toggle schedule:', e);
    }
  };

  const onlineNodes = nodes.filter((n) => n.is_online);

  const formatTimeAgo = (dateStr: string) => {
    const diff = Date.now() - new Date(dateStr).getTime();
    const minutes = Math.floor(diff / 60000);
    if (minutes < 60) return `vor ${minutes} Min.`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `vor ${hours} Std.`;
    const days = Math.floor(hours / 24);
    return `vor ${days} Tag(en)`;
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Backups</h1>
          <p className="text-muted-foreground">
            Zentrale Verwaltung aller Konfigurations-Backups, Zeitpläne und Vzdump-Sicherungen.
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={() => loadBackups()}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Aktualisieren
          </Button>
          <Button variant="outline" size="sm" onClick={() => setVzdumpOpen(true)}>
            <HardDrive className="mr-2 h-4 w-4" />
            Vzdump
          </Button>
        </div>
      </div>

      {/* KPI Cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard
          title="Backups gesamt"
          value={backups.length}
          subtitle={`${formatBytes(totalSize)} Gesamtgröße`}
          icon={Archive}
          color="blue"
        />
        <KpiCard
          title="Erfolgreich"
          value={completedCount}
          subtitle={backups.length > 0 ? `${Math.round((completedCount / backups.length) * 100)}% Erfolgsrate` : "Keine Backups"}
          icon={CheckCircle}
          color="green"
        />
        <KpiCard
          title="Fehlgeschlagen"
          value={failedCount}
          subtitle={failedCount > 0 ? "Achtung erforderlich" : "Alles in Ordnung"}
          icon={failedCount > 0 ? AlertCircle : Shield}
          color={failedCount > 0 ? "red" : "green"}
        />
        <KpiCard
          title="Aktive Zeitpläne"
          value={activeSchedules}
          subtitle={`${allSchedules.length} Zeitpläne gesamt`}
          icon={Calendar}
          color="orange"
        />
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="backups">
            Konfig-Backups
            {backups.length > 0 && (
              <Badge variant="outline" className="ml-2 text-[10px] px-1.5 py-0">
                {backups.length}
              </Badge>
            )}
          </TabsTrigger>
          <TabsTrigger value="schedules">
            Zeitpläne
            {allSchedules.length > 0 && (
              <Badge variant="outline" className="ml-2 text-[10px] px-1.5 py-0">
                {allSchedules.length}
              </Badge>
            )}
          </TabsTrigger>
          <TabsTrigger value="create">Backup erstellen</TabsTrigger>
        </TabsList>

        {/* Backups Tab */}
        <TabsContent value="backups" className="space-y-4">
          {!isLoading && loadError && (
            <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-4 flex items-center gap-3">
              <AlertCircle className="h-5 w-5 text-destructive shrink-0" />
              <div className="flex-1">
                <p className="font-medium text-destructive">Fehler beim Laden der Backups</p>
                <p className="text-sm text-muted-foreground">{loadError}</p>
              </div>
              <Button variant="outline" size="sm" onClick={() => loadBackups()}>
                <RefreshCw className="mr-2 h-4 w-4" />
                Erneut versuchen
              </Button>
            </div>
          )}
          {isLoading ? (
            <div className="space-y-3" aria-busy="true" aria-label="Backups werden geladen">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-24 w-full" />
              ))}
            </div>
          ) : backups.length === 0 ? (
            <Card>
              <CardContent className="flex flex-col items-center justify-center py-16">
                <Archive className="mb-4 h-12 w-12 text-muted-foreground" />
                <p className="text-lg font-medium">Noch keine Backups vorhanden</p>
                <p className="mt-1 text-sm text-muted-foreground">
                  Erstellen Sie Ihr erstes Backup, um Ihre Konfiguration zu sichern.
                </p>
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
                                <Link
                                  href={`/nodes/${backup.node_id}`}
                                  className="font-semibold hover:underline flex items-center gap-1"
                                >
                                  <Server className="h-3.5 w-3.5 text-muted-foreground" />
                                  {getNodeName(backup.node_id)}
                                </Link>
                                <span className="text-muted-foreground font-mono text-sm">v{backup.version}</span>
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
                                <p className="text-xs text-foreground/70 bg-muted/50 rounded px-2 py-1 mt-1 inline-block">
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
                                aria-label="Details anzeigen"
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
                                    aria-label="Wiederherstellen"
                                    onClick={() => setRestoreBackupId(backup.id)}
                                  >
                                    <RotateCcw className="h-4 w-4" />
                                  </Button>
                                  <Button
                                    variant="ghost"
                                    size="icon"
                                    title="Herunterladen"
                                    aria-label="Herunterladen"
                                    onClick={() => handleDownload(backup.id)}
                                  >
                                    <Download className="h-4 w-4" />
                                  </Button>
                                </>
                              )}
                              <Button
                                variant="ghost"
                                size="icon"
                                title="Löschen"
                                aria-label="Löschen"
                                onClick={() => setDeleteConfirm(backup.id)}
                              >
                                <Trash2 className="h-4 w-4 text-destructive" />
                              </Button>
                              <AlertDialog open={deleteConfirm === backup.id} onOpenChange={(open) => !open && setDeleteConfirm(null)}>
                                <AlertDialogContent>
                                  <AlertDialogHeader>
                                    <AlertDialogTitle>Backup löschen?</AlertDialogTitle>
                                    <AlertDialogDescription>
                                      Backup v{backup.version} von {getNodeName(backup.node_id)} wird unwiderruflich gelöscht.
                                    </AlertDialogDescription>
                                  </AlertDialogHeader>
                                  <AlertDialogFooter>
                                    <AlertDialogCancel>Abbrechen</AlertDialogCancel>
                                    <AlertDialogAction onClick={() => handleDelete(backup.id)} className="bg-destructive text-destructive-foreground hover:bg-destructive/90">
                                      Löschen
                                    </AlertDialogAction>
                                  </AlertDialogFooter>
                                </AlertDialogContent>
                              </AlertDialog>
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
        </TabsContent>

        {/* Schedules Tab */}
        <TabsContent value="schedules" className="space-y-4">
          {schedulesLoading ? (
            <div className="space-y-3">
              {Array.from({ length: 2 }).map((_, i) => (
                <Skeleton key={i} className="h-20 w-full" />
              ))}
            </div>
          ) : allSchedules.length === 0 ? (
            <Card>
              <CardContent className="flex flex-col items-center justify-center py-16">
                <Calendar className="mb-4 h-12 w-12 text-muted-foreground" />
                <p className="text-lg font-medium">Keine Zeitpläne vorhanden</p>
                <p className="mt-1 text-sm text-muted-foreground">
                  Erstellen Sie einen automatischen Backup-Zeitplan.
                </p>
              </CardContent>
            </Card>
          ) : (
            <div className="space-y-3">
              {allSchedules.map((s) => (
                <Card key={s.id} className="transition-all hover:shadow-md">
                  <CardContent className="p-0">
                    <div className="flex items-stretch">
                      <div className={`w-1 rounded-l-lg shrink-0 ${s.is_active ? "bg-green-500" : "bg-gray-300"}`} />
                      <div className="flex-1 p-4">
                        <div className="flex items-center justify-between">
                          <div className="space-y-1.5">
                            <div className="flex items-center gap-2">
                              <Server className="h-3.5 w-3.5 text-muted-foreground" />
                              <span className="font-semibold">{getNodeName(s._nodeId)}</span>
                              <code className="text-sm bg-muted px-2 py-0.5 rounded font-mono">
                                {s.cron_expression}
                              </code>
                              <Badge variant={s.is_active ? "success" : "outline"}>
                                {s.is_active ? "Aktiv" : "Pausiert"}
                              </Badge>
                            </div>
                            <div className="flex items-center gap-3 text-sm text-muted-foreground">
                              <span>Aufbewahrung: {s.retention_count} Backups</span>
                              {s.last_run_at && (
                                <span>Letzter Lauf: {new Date(s.last_run_at).toLocaleString("de-DE")}</span>
                              )}
                              {s.next_run_at && (
                                <span className="text-primary">
                                  Nächster Lauf: {new Date(s.next_run_at).toLocaleString("de-DE")}
                                </span>
                              )}
                            </div>
                          </div>
                          <div className="flex gap-1 items-center">
                            <Button variant="outline" size="sm" onClick={() => handleToggleSchedule(s)}>
                              {s.is_active ? "Pausieren" : "Aktivieren"}
                            </Button>
                            <Button
                              variant="ghost"
                              size="icon"
                              aria-label="Zeitplan löschen"
                              onClick={() => handleDeleteSchedule(s.id)}
                            >
                              <Trash2 className="h-4 w-4 text-destructive" />
                            </Button>
                          </div>
                        </div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}

          {/* Create new schedule */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Neuen Zeitplan erstellen</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="schedule-node-select">Server</Label>
                <Select value={scheduleNodeId} onValueChange={setScheduleNodeId}>
                  <SelectTrigger id="schedule-node-select">
                    <SelectValue placeholder="Server auswählen..." />
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
            <CardHeader>
              <CardTitle className="text-base">Konfigurations-Backup erstellen</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <p className="text-sm text-muted-foreground">
                Sichert die Konfigurationsdateien des ausgewählten Servers (z.B. /etc/pve, /etc/network, Crontabs).
              </p>
              <div className="space-y-2">
                <Label htmlFor="create-node-select">Server</Label>
                <Select value={createNodeId} onValueChange={setCreateNodeId}>
                  <SelectTrigger id="create-node-select">
                    <SelectValue placeholder="Server auswählen..." />
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
                <Label htmlFor="create-notes">Notizen (optional)</Label>
                <textarea
                  id="create-notes"
                  className="w-full rounded-md border bg-background px-3 py-2 text-sm min-h-[80px] resize-y"
                  rows={3}
                  value={createNotes}
                  onChange={(e) => setCreateNotes(e.target.value)}
                  placeholder="z.B. Vor Kernel-Update, Nach Netzwerk-Umbau..."
                />
              </div>
              <Button
                onClick={handleCreateBackup}
                disabled={isCreating || !createNodeId}
                className="w-full sm:w-auto"
              >
                {isCreating ? (
                  <>
                    <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                    Erstelle Backup...
                  </>
                ) : (
                  <>
                    <Archive className="mr-2 h-4 w-4" />
                    Backup erstellen
                  </>
                )}
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
