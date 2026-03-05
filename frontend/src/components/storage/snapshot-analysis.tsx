"use client";

import { useMemo } from "react";
import { Camera, Clock, Trash2, AlertTriangle } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { formatBytes } from "@/lib/utils";

interface SnapshotEntry {
  vmName: string;
  vmid: number;
  snapshotName: string;
  date: Date;
  estimatedSize: number;
  storage: string;
}

interface SnapshotAnalysisProps {
  /** Optional static data; if not provided, generates demo data */
  snapshots?: SnapshotEntry[];
}

function generateDemoSnapshots(): SnapshotEntry[] {
  const now = Date.now();
  const day = 86400000;
  const vmNames = ["web-server-01", "db-primary", "mail-gw", "dev-sandbox", "monitoring", "proxy-lb"];
  const storageNames = ["local-zfs", "ceph-pool", "nfs-backup"];
  const snapNames = ["daily-auto", "pre-update", "manual-backup", "weekly", "pre-migration"];

  const entries: SnapshotEntry[] = [];
  for (let i = 0; i < 12; i++) {
    const ageInDays = Math.floor(Math.random() * 180);
    entries.push({
      vmName: vmNames[i % vmNames.length],
      vmid: 100 + i,
      snapshotName: `${snapNames[i % snapNames.length]}-${i}`,
      date: new Date(now - ageInDays * day),
      estimatedSize: Math.floor(Math.random() * 50 * 1024 * 1024 * 1024) + 512 * 1024 * 1024,
      storage: storageNames[i % storageNames.length],
    });
  }
  return entries.sort((a, b) => b.date.getTime() - a.date.getTime());
}

function getAgeCategory(date: Date): { label: string; color: string; variant: "success" | "outline" | "destructive" } {
  const ageMs = Date.now() - date.getTime();
  const ageDays = ageMs / 86400000;
  if (ageDays <= 7) return { label: "Aktuell", color: "text-green-600", variant: "success" };
  if (ageDays <= 30) return { label: "Normal", color: "text-amber-600", variant: "outline" };
  if (ageDays <= 90) return { label: "Alt", color: "text-orange-600", variant: "outline" };
  return { label: "Sehr alt", color: "text-red-600", variant: "destructive" };
}

function formatRelativeDate(date: Date): string {
  const ageDays = Math.floor((Date.now() - date.getTime()) / 86400000);
  if (ageDays === 0) return "Heute";
  if (ageDays === 1) return "Gestern";
  if (ageDays < 7) return `Vor ${ageDays} Tagen`;
  if (ageDays < 30) return `Vor ${Math.floor(ageDays / 7)} Wochen`;
  if (ageDays < 365) return `Vor ${Math.floor(ageDays / 30)} Monaten`;
  return `Vor ${Math.floor(ageDays / 365)} Jahren`;
}

export function SnapshotAnalysis({ snapshots: propSnapshots }: SnapshotAnalysisProps) {
  const snapshots = useMemo(() => propSnapshots || generateDemoSnapshots(), [propSnapshots]);

  const totalSize = useMemo(() => snapshots.reduce((acc, s) => acc + s.estimatedSize, 0), [snapshots]);
  const oldSnapshots = useMemo(
    () => snapshots.filter((s) => (Date.now() - s.date.getTime()) / 86400000 > 30),
    [snapshots]
  );
  const oldTotalSize = useMemo(() => oldSnapshots.reduce((acc, s) => acc + s.estimatedSize, 0), [oldSnapshots]);

  // Group by storage
  const byStorage = useMemo(() => {
    const map: Record<string, SnapshotEntry[]> = {};
    for (const s of snapshots) {
      if (!map[s.storage]) map[s.storage] = [];
      map[s.storage].push(s);
    }
    return map;
  }, [snapshots]);

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Camera className="h-4 w-4 text-primary" />
            <CardTitle className="text-base">Snapshot-Analyse</CardTitle>
            <Badge variant="outline" className="text-xs">{snapshots.length} Snapshots</Badge>
          </div>
          {oldSnapshots.length > 0 && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="outline" size="sm" className="gap-1.5 text-xs">
                  <Trash2 className="h-3.5 w-3.5" />
                  Alte Snapshots bereinigen ({oldSnapshots.length})
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                {oldSnapshots.length} Snapshots aelter als 30 Tage ({formatBytes(oldTotalSize)})
              </TooltipContent>
            </Tooltip>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Summary */}
        <div className="grid grid-cols-3 gap-3">
          <div className="rounded-lg border p-3 text-center">
            <p className="text-lg font-bold">{snapshots.length}</p>
            <p className="text-xs text-muted-foreground">Gesamt</p>
          </div>
          <div className="rounded-lg border p-3 text-center">
            <p className="text-lg font-bold">{formatBytes(totalSize)}</p>
            <p className="text-xs text-muted-foreground">Geschaetzte Groesse</p>
          </div>
          <div className="rounded-lg border p-3 text-center">
            <p className="text-lg font-bold text-amber-600">{oldSnapshots.length}</p>
            <p className="text-xs text-muted-foreground">Aelter als 30 Tage</p>
          </div>
        </div>

        {/* By storage */}
        {Object.entries(byStorage).map(([storageName, snaps]) => (
          <div key={storageName} className="space-y-2">
            <div className="flex items-center gap-2 text-sm font-medium">
              <span>{storageName}</span>
              <Badge variant="outline" className="text-[10px]">{snaps.length}</Badge>
            </div>
            <div className="space-y-1">
              {snaps.map((snap, i) => {
                const age = getAgeCategory(snap.date);
                return (
                  <div
                    key={`${snap.vmid}-${snap.snapshotName}-${i}`}
                    className="flex items-center justify-between rounded-md border px-3 py-2 text-sm"
                  >
                    <div className="flex items-center gap-3 min-w-0">
                      <div className="flex items-center gap-1.5 min-w-0">
                        <span className="font-medium truncate">{snap.vmName}</span>
                        <span className="text-muted-foreground text-xs">({snap.vmid})</span>
                      </div>
                      <span className="text-muted-foreground text-xs truncate">{snap.snapshotName}</span>
                    </div>
                    <div className="flex items-center gap-3 shrink-0">
                      <Badge variant={age.variant} className="text-[10px]">
                        {age.label}
                      </Badge>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span className={`flex items-center gap-1 text-xs ${age.color}`}>
                            <Clock className="h-3 w-3" />
                            {formatRelativeDate(snap.date)}
                          </span>
                        </TooltipTrigger>
                        <TooltipContent>
                          {snap.date.toLocaleDateString("de-DE", {
                            day: "2-digit",
                            month: "2-digit",
                            year: "numeric",
                            hour: "2-digit",
                            minute: "2-digit",
                          })}
                        </TooltipContent>
                      </Tooltip>
                      <span className="text-xs text-muted-foreground w-16 text-right">
                        {formatBytes(snap.estimatedSize)}
                      </span>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        ))}

        {/* Warning for old snapshots */}
        {oldSnapshots.length > 3 && (
          <div className="flex items-start gap-2 rounded-lg border border-amber-500/30 bg-amber-500/5 p-3">
            <AlertTriangle className="h-4 w-4 text-amber-500 mt-0.5 shrink-0" />
            <div className="text-sm">
              <p className="font-medium text-amber-700 dark:text-amber-400">
                {oldSnapshots.length} veraltete Snapshots gefunden
              </p>
              <p className="text-xs text-muted-foreground mt-0.5">
                Diese belegen ca. {formatBytes(oldTotalSize)} Speicherplatz.
                Eine regelmaessige Bereinigung wird empfohlen.
              </p>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
