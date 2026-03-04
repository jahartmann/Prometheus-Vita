"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Archive, CheckCircle, XCircle, Clock } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useNodeStore } from "@/stores/node-store";
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

export default function BackupsPage() {
  const { nodes, fetchNodes } = useNodeStore();
  const [backups, setBackups] = useState<ConfigBackup[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    fetchNodes();
  }, [fetchNodes]);

  useEffect(() => {
    const loadBackups = async () => {
      setIsLoading(true);
      try {
        const response = await backupApi.listAll();
        setBackups(response.data || []);
      } catch {
        setBackups([]);
      } finally {
        setIsLoading(false);
      }
    };
    loadBackups();
  }, []);

  const completedCount = backups.filter((b) => b.status === "completed").length;
  const failedCount = backups.filter((b) => b.status === "failed").length;

  const getNodeName = (nodeId: string) => {
    const node = nodes.find((n) => n.id === nodeId);
    return node?.name ?? nodeId.slice(0, 8);
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Backups</h1>
        <p className="text-muted-foreground">
          Uebersicht aller Konfigurations-Backups.
        </p>
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

      {/* Backup List */}
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
                  <Clock className="h-5 w-5 text-muted-foreground" />
                  <div>
                    <div className="flex items-center gap-2">
                      <Link
                        href={`/nodes/${backup.node_id}/backups`}
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
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
