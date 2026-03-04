"use client";

import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { backupApi } from "@/lib/api";
import { formatBytes } from "@/lib/utils";
import type { ConfigBackup, BackupFile, FileDiff } from "@/types/api";

interface BackupDetailDialogProps {
  backup: ConfigBackup;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function BackupDetailDialog({ backup, open, onOpenChange }: BackupDetailDialogProps) {
  const [files, setFiles] = useState<BackupFile[]>([]);
  const [diffs, setDiffs] = useState<FileDiff[]>([]);
  const [tab, setTab] = useState<"files" | "diff">("files");

  useEffect(() => {
    if (open && backup.id) {
      backupApi.getBackupFiles(backup.id).then((res) => {
        setFiles(res.data?.data || res.data || []);
      });
      backupApi.diffBackup(backup.id).then((res) => {
        setDiffs(res.data?.data || res.data || []);
      }).catch(() => {});
    }
  }, [open, backup.id]);

  if (!open) return null;

  const diffStatusColor: Record<string, string> = {
    added: "text-green-500",
    removed: "text-red-500",
    modified: "text-amber-500",
    unchanged: "text-muted-foreground",
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <Card className="w-full max-w-3xl max-h-[80vh] overflow-auto">
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Backup v{backup.version}</CardTitle>
            <Button variant="ghost" onClick={() => onOpenChange(false)}>
              Schliessen
            </Button>
          </div>
          <p className="text-sm text-muted-foreground">
            {backup.file_count} Dateien | {formatBytes(backup.total_size)} |{" "}
            {new Date(backup.created_at).toLocaleString("de-DE")}
          </p>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-2">
            <Button
              variant={tab === "files" ? "default" : "outline"}
              size="sm"
              onClick={() => setTab("files")}
            >
              Dateien
            </Button>
            <Button
              variant={tab === "diff" ? "default" : "outline"}
              size="sm"
              onClick={() => setTab("diff")}
            >
              Aenderungen
            </Button>
          </div>

          {tab === "files" ? (
            <div className="space-y-1">
              {files.map((f) => (
                <div
                  key={f.id}
                  className="flex items-center justify-between rounded px-3 py-2 text-sm hover:bg-muted"
                >
                  <span className="font-mono text-xs">{f.file_path}</span>
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <span>{formatBytes(f.file_size)}</span>
                    <span>{f.file_permissions}</span>
                    <span>{f.file_owner}</span>
                  </div>
                </div>
              ))}
              {files.length === 0 && (
                <p className="text-sm text-muted-foreground">Keine Dateien.</p>
              )}
            </div>
          ) : (
            <div className="space-y-2">
              {diffs
                .filter((d) => d.status !== "unchanged")
                .map((d) => (
                  <div key={d.file_path} className="rounded border p-3">
                    <div className="flex items-center gap-2 mb-2">
                      <span className={`font-mono text-xs ${diffStatusColor[d.status]}`}>
                        {d.status}
                      </span>
                      <span className="font-mono text-xs">{d.file_path}</span>
                    </div>
                    {d.diff && (
                      <pre className="rounded bg-muted p-2 text-xs overflow-x-auto whitespace-pre-wrap">
                        {d.diff}
                      </pre>
                    )}
                  </div>
                ))}
              {diffs.filter((d) => d.status !== "unchanged").length === 0 && (
                <p className="text-sm text-muted-foreground">
                  Keine Aenderungen seit letztem Backup.
                </p>
              )}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
