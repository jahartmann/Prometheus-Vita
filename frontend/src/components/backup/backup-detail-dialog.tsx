"use client";

import { useEffect, useState } from "react";
import {
  Download,
  FileText,
  GitBranch,
  BookOpen,
  File,
  Plus,
  Minus,
  Edit3,
  CheckCircle,
  HardDrive,
  Clock,
  Shield,
  Hash,
} from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { backupApi, toArray } from "@/lib/api";
import { formatBytes } from "@/lib/utils";
import type { ConfigBackup, BackupFile, FileDiff } from "@/types/api";

interface BackupDetailDialogProps {
  backup: ConfigBackup;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

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

const diffStatusConfig: Record<string, { label: string; color: string; bg: string; icon: typeof Plus }> = {
  added: { label: "Hinzugefügt", color: "text-green-600", bg: "bg-green-500/10 border-green-500/20", icon: Plus },
  removed: { label: "Entfernt", color: "text-red-600", bg: "bg-red-500/10 border-red-500/20", icon: Minus },
  modified: { label: "Geändert", color: "text-amber-600", bg: "bg-amber-500/10 border-amber-500/20", icon: Edit3 },
  unchanged: { label: "Unverändert", color: "text-muted-foreground", bg: "bg-muted", icon: CheckCircle },
};

export function BackupDetailDialog({ backup, open, onOpenChange }: BackupDetailDialogProps) {
  const [files, setFiles] = useState<BackupFile[]>([]);
  const [diffs, setDiffs] = useState<FileDiff[]>([]);
  const [recoveryGuide, setRecoveryGuide] = useState<string>("");
  const [isLoadingFiles, setIsLoadingFiles] = useState(true);
  const [isLoadingDiffs, setIsLoadingDiffs] = useState(true);
  const [isLoadingGuide, setIsLoadingGuide] = useState(true);

  useEffect(() => {
    if (open && backup.id) {
      setIsLoadingFiles(true);
      setIsLoadingDiffs(true);
      setIsLoadingGuide(true);

      backupApi.getBackupFiles(backup.id).then((res) => {
        setFiles(toArray<BackupFile>(res.data));
      }).finally(() => setIsLoadingFiles(false));

      backupApi.diffBackup(backup.id).then((res) => {
        setDiffs(toArray<FileDiff>(res.data));
      }).catch(() => setDiffs([])).finally(() => setIsLoadingDiffs(false));

      backupApi.getRecoveryGuide(backup.id).then((res) => {
        const data = res.data as { recovery_guide?: string };
        setRecoveryGuide(data?.recovery_guide || "");
      }).catch(() => setRecoveryGuide("")).finally(() => setIsLoadingGuide(false));
    }
  }, [open, backup.id]);

  const downloadRecoveryGuide = () => {
    const blob = new Blob([recoveryGuide], { type: "text/markdown" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `recovery-guide-v${backup.version}.md`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const changedDiffs = diffs.filter((d) => d.status !== "unchanged");
  const addedCount = diffs.filter((d) => d.status === "added").length;
  const removedCount = diffs.filter((d) => d.status === "removed").length;
  const modifiedCount = diffs.filter((d) => d.status === "modified").length;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-4xl max-h-[85vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-3">
            <Shield className="h-5 w-5 text-primary" />
            Backup v{backup.version}
            <Badge variant={statusVariant[backup.status] || "outline"}>
              {statusLabel[backup.status] || backup.status}
            </Badge>
          </DialogTitle>
          <DialogDescription className="sr-only">
            Details zum Backup Version {backup.version}
          </DialogDescription>
        </DialogHeader>

        {/* Backup Info Bar */}
        <div className="flex flex-wrap gap-4 text-sm text-muted-foreground border rounded-lg p-3 bg-muted/30">
          <span className="flex items-center gap-1.5">
            <FileText className="h-3.5 w-3.5" />
            {backup.file_count} Dateien
          </span>
          <span className="flex items-center gap-1.5">
            <HardDrive className="h-3.5 w-3.5" />
            {formatBytes(backup.total_size)}
          </span>
          <span className="flex items-center gap-1.5">
            <Clock className="h-3.5 w-3.5" />
            {new Date(backup.created_at).toLocaleString("de-DE")}
          </span>
          {backup.completed_at && (
            <span className="flex items-center gap-1.5">
              <CheckCircle className="h-3.5 w-3.5" />
              Abgeschlossen: {new Date(backup.completed_at).toLocaleString("de-DE")}
            </span>
          )}
          <span className="flex items-center gap-1.5">
            <Hash className="h-3.5 w-3.5" />
            {backup.backup_type === "manual" ? "Manuell" : backup.backup_type === "scheduled" ? "Geplant" : backup.backup_type}
          </span>
        </div>

        {backup.notes && (
          <div className="text-sm bg-muted/50 rounded-lg px-3 py-2">
            <span className="font-medium">Notizen:</span> {backup.notes}
          </div>
        )}
        {backup.error_message && (
          <div className="text-sm bg-red-500/10 border border-red-500/20 rounded-lg px-3 py-2 text-red-600">
            <span className="font-medium">Fehler:</span> {backup.error_message}
          </div>
        )}

        {/* Tabs */}
        <Tabs defaultValue="files" className="flex-1 overflow-hidden flex flex-col">
          <TabsList className="w-full justify-start">
            <TabsTrigger value="files" className="flex items-center gap-1.5">
              <FileText className="h-3.5 w-3.5" />
              Dateien
              <Badge variant="outline" className="ml-1 text-[10px] px-1 py-0">
                {files.length}
              </Badge>
            </TabsTrigger>
            <TabsTrigger value="diff" className="flex items-center gap-1.5">
              <GitBranch className="h-3.5 w-3.5" />
              Änderungen
              {changedDiffs.length > 0 && (
                <Badge variant="outline" className="ml-1 text-[10px] px-1 py-0">
                  {changedDiffs.length}
                </Badge>
              )}
            </TabsTrigger>
            <TabsTrigger value="recovery" className="flex items-center gap-1.5">
              <BookOpen className="h-3.5 w-3.5" />
              Recovery Guide
            </TabsTrigger>
          </TabsList>

          <div className="flex-1 overflow-auto mt-2">
            {/* Files Tab */}
            <TabsContent value="files" className="m-0">
              {isLoadingFiles ? (
                <div className="space-y-2">
                  {Array.from({ length: 5 }).map((_, i) => (
                    <div key={i} className="h-10 animate-pulse rounded bg-muted" />
                  ))}
                </div>
              ) : files.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
                  <FileText className="h-10 w-10 mb-2" />
                  <p>Keine Dateien in diesem Backup.</p>
                </div>
              ) : (
                <div className="rounded-lg border overflow-hidden">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b bg-muted/50">
                        <th className="text-left px-3 py-2 font-medium">Dateipfad</th>
                        <th className="text-right px-3 py-2 font-medium w-24">Größe</th>
                        <th className="text-center px-3 py-2 font-medium w-24">Rechte</th>
                        <th className="text-center px-3 py-2 font-medium w-24">Besitzer</th>
                      </tr>
                    </thead>
                    <tbody>
                      {files.map((f) => (
                        <tr key={f.id} className="border-b last:border-0 hover:bg-muted/30 transition-colors">
                          <td className="px-3 py-2">
                            <div className="flex items-center gap-2">
                              <File className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                              <span className="font-mono text-xs break-all">{f.file_path}</span>
                            </div>
                          </td>
                          <td className="text-right px-3 py-2 text-xs text-muted-foreground whitespace-nowrap">
                            {formatBytes(f.file_size)}
                          </td>
                          <td className="text-center px-3 py-2">
                            {f.file_permissions && (
                              <code className="text-xs bg-muted px-1.5 py-0.5 rounded">{f.file_permissions}</code>
                            )}
                          </td>
                          <td className="text-center px-3 py-2 text-xs text-muted-foreground">
                            {f.file_owner || "-"}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </TabsContent>

            {/* Diff Tab */}
            <TabsContent value="diff" className="m-0">
              {isLoadingDiffs ? (
                <div className="space-y-2">
                  {Array.from({ length: 3 }).map((_, i) => (
                    <div key={i} className="h-16 animate-pulse rounded bg-muted" />
                  ))}
                </div>
              ) : (
                <div className="space-y-3">
                  {/* Summary */}
                  {changedDiffs.length > 0 && (
                    <div className="flex gap-3 text-sm">
                      {addedCount > 0 && (
                        <span className="flex items-center gap-1 text-green-600">
                          <Plus className="h-3.5 w-3.5" /> {addedCount} hinzugefügt
                        </span>
                      )}
                      {modifiedCount > 0 && (
                        <span className="flex items-center gap-1 text-amber-600">
                          <Edit3 className="h-3.5 w-3.5" /> {modifiedCount} geändert
                        </span>
                      )}
                      {removedCount > 0 && (
                        <span className="flex items-center gap-1 text-red-600">
                          <Minus className="h-3.5 w-3.5" /> {removedCount} entfernt
                        </span>
                      )}
                    </div>
                  )}

                  {changedDiffs.length === 0 ? (
                    <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
                      <CheckCircle className="h-10 w-10 mb-2 text-green-500" />
                      <p className="font-medium">Keine Änderungen</p>
                      <p className="text-sm">Seit dem letzten Backup hat sich nichts geändert.</p>
                    </div>
                  ) : (
                    changedDiffs.map((d) => {
                      const config = diffStatusConfig[d.status] || diffStatusConfig.unchanged;
                      const DiffIcon = config.icon;
                      return (
                        <div key={d.file_path} className={`rounded-lg border p-3 ${config.bg}`}>
                          <div className="flex items-center gap-2 mb-2">
                            <DiffIcon className={`h-4 w-4 ${config.color}`} />
                            <Badge variant="outline" className={`text-xs ${config.color}`}>
                              {config.label}
                            </Badge>
                            <span className="font-mono text-xs">{d.file_path}</span>
                          </div>
                          {d.diff && (
                            <pre className="rounded-md bg-background border p-3 text-xs overflow-x-auto whitespace-pre-wrap font-mono leading-relaxed max-h-64 overflow-y-auto">
                              {d.diff}
                            </pre>
                          )}
                        </div>
                      );
                    })
                  )}
                </div>
              )}
            </TabsContent>

            {/* Recovery Guide Tab */}
            <TabsContent value="recovery" className="m-0">
              {isLoadingGuide ? (
                <div className="space-y-2">
                  {Array.from({ length: 4 }).map((_, i) => (
                    <div key={i} className="h-6 animate-pulse rounded bg-muted" />
                  ))}
                </div>
              ) : recoveryGuide ? (
                <div className="space-y-3">
                  <div className="flex items-center justify-between">
                    <p className="text-sm text-muted-foreground">
                      Automatisch generierter Wiederherstellungsleitfaden
                    </p>
                    <Button variant="outline" size="sm" onClick={downloadRecoveryGuide}>
                      <Download className="mr-2 h-3.5 w-3.5" />
                      Als Markdown herunterladen
                    </Button>
                  </div>
                  <div className="rounded-lg border bg-muted/30 p-4">
                    <pre className="text-xs font-mono overflow-x-auto whitespace-pre-wrap leading-relaxed">
                      {recoveryGuide}
                    </pre>
                  </div>
                </div>
              ) : (
                <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
                  <BookOpen className="h-10 w-10 mb-2" />
                  <p className="font-medium">Kein Recovery Guide verfügbar</p>
                  <p className="text-sm mt-1">
                    Dieser wird automatisch nach Abschluss eines Backups generiert.
                  </p>
                </div>
              )}
            </TabsContent>
          </div>
        </Tabs>
      </DialogContent>
    </Dialog>
  );
}
