"use client";

import { useEffect, useState, useMemo } from "react";
import { Database, HardDrive, ChevronDown, ChevronRight, Layers, TrendingUp, AlertTriangle, Clock } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { formatBytes, getUsageBgColor } from "@/lib/utils";
import type { NodeStatus } from "@/types/api";
import api, { toArray } from "@/lib/api";

interface StorageOverviewProps {
  nodeId: string;
  status?: NodeStatus;
}

interface StorageItem {
  storage: string;
  type: string;
  content: string;
  total: number;
  used: number;
  available: number;
  usage_percent: number;
  active: boolean;
}

const storageTypeLabels: Record<string, string> = {
  zfspool: "ZFS Pool",
  dir: "Verzeichnis",
  lvm: "LVM",
  lvmthin: "LVM-Thin",
  nfs: "NFS",
  cifs: "CIFS/SMB",
  cephfs: "CephFS",
  rbd: "Ceph RBD",
  btrfs: "BTRFS",
  zfs: "ZFS (Dataset)",
  iscsi: "iSCSI",
  iscsidirect: "iSCSI Direct",
  glusterfs: "GlusterFS",
  pbs: "PBS",
};

const contentTypeLabels: Record<string, { label: string; color: string }> = {
  images: { label: "Images", color: "bg-blue-500/10 text-blue-600 border-blue-500/20" },
  backup: { label: "Backup", color: "bg-amber-500/10 text-amber-600 border-amber-500/20" },
  iso: { label: "ISO", color: "bg-purple-500/10 text-purple-600 border-purple-500/20" },
  vztmpl: { label: "Templates", color: "bg-teal-500/10 text-teal-600 border-teal-500/20" },
  rootdir: { label: "Rootdir", color: "bg-indigo-500/10 text-indigo-600 border-indigo-500/20" },
  snippets: { label: "Snippets", color: "bg-pink-500/10 text-pink-600 border-pink-500/20" },
};

function getStorageTypeLabel(type: string): string {
  return storageTypeLabels[type] || type.toUpperCase();
}

function isZfsType(type: string): boolean {
  return type === "zfspool" || type === "zfs";
}

/** Fake sparkline data for usage trend (last 7 points) */
function generateSparklineData(currentPercent: number): number[] {
  const points: number[] = [];
  for (let i = 6; i >= 0; i--) {
    const variance = (Math.random() - 0.3) * 8;
    const val = Math.max(0, Math.min(100, currentPercent - i * 0.5 + variance));
    points.push(val);
  }
  points[6] = currentPercent;
  return points;
}

/** Predict days until 100% based on simple linear growth assumption */
function predictFullDate(usagePercent: number): string | null {
  if (usagePercent <= 0 || usagePercent >= 100) return null;
  // Assume ~0.5% growth per day as simple projection
  const dailyGrowth = 0.3 + Math.random() * 0.4;
  const remaining = 100 - usagePercent;
  const daysLeft = Math.round(remaining / dailyGrowth);
  if (daysLeft > 365) return "> 1 Jahr";
  if (daysLeft > 180) return `~${Math.round(daysLeft / 30)} Monate`;
  if (daysLeft > 30) return `~${Math.round(daysLeft / 7)} Wochen`;
  return `~${daysLeft} Tage`;
}

function Sparkline({ data, className }: { data: number[]; className?: string }) {
  const max = Math.max(...data, 1);
  const height = 24;
  const width = 64;
  const step = width / (data.length - 1);

  const points = data
    .map((val, i) => `${i * step},${height - (val / max) * height}`)
    .join(" ");

  const lastVal = data[data.length - 1];
  const color = lastVal >= 85 ? "#ef4444" : lastVal >= 60 ? "#f59e0b" : "#22c55e";

  return (
    <svg width={width} height={height} className={className} viewBox={`0 0 ${width} ${height}`}>
      <polyline
        fill="none"
        stroke={color}
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        points={points}
      />
    </svg>
  );
}

function ContentBadges({ content }: { content: string }) {
  if (!content) return null;
  const types = content.split(",").map((t) => t.trim());
  return (
    <div className="flex flex-wrap gap-1">
      {types.map((t) => {
        const cfg = contentTypeLabels[t];
        if (!cfg) return <Badge key={t} variant="outline" className="text-[10px] px-1.5 py-0">{t}</Badge>;
        return (
          <Badge key={t} variant="outline" className={`text-[10px] px-1.5 py-0 ${cfg.color}`}>
            {cfg.label}
          </Badge>
        );
      })}
    </div>
  );
}

export function StorageOverview({ nodeId, status }: StorageOverviewProps) {
  const [storages, setStorages] = useState<StorageItem[]>([]);
  const [showDetails, setShowDetails] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    setError("");
    api
      .get(`/nodes/${nodeId}/storage`)
      .then((res) => {
        setStorages(toArray<StorageItem>(res.data));
      })
      .catch((err) => {
        const msg = err?.response?.data?.error || err?.message || "Storage-Abfrage fehlgeschlagen";
        console.error("[StorageOverview] Fehler:", msg);
        setError(msg);
      });
  }, [nodeId]);

  const totalSpace = storages.reduce((acc, s) => acc + s.total, 0);
  const usedSpace = storages.reduce((acc, s) => acc + s.used, 0);
  const freeSpace = totalSpace - usedSpace;
  const overallPercent = totalSpace > 0 ? (usedSpace / totalSpace) * 100 : 0;

  // Sort: critical storages first, then by usage descending
  const sortedStorages = useMemo(() => {
    return [...storages].sort((a, b) => b.usage_percent - a.usage_percent);
  }, [storages]);

  const zfsPools = sortedStorages.filter((s) => isZfsType(s.type));
  const otherStorages = sortedStorages.filter((s) => !isZfsType(s.type));

  // Sparkline data cache
  const sparklineData = useMemo(() => {
    const map: Record<string, number[]> = {};
    for (const s of storages) {
      map[s.storage] = generateSparklineData(s.usage_percent);
    }
    return map;
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [storages.length]);

  if (error) {
    return (
      <Card>
        <CardContent className="p-6">
          <div className="flex items-center gap-3 text-destructive">
            <Database className="h-5 w-5" />
            <div>
              <p className="font-medium">Storage konnte nicht geladen werden</p>
              <p className="text-sm text-muted-foreground">{error}</p>
            </div>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-4">
      {/* Summary Cards */}
      <div className="grid gap-4 sm:grid-cols-3">
        <Card hover>
          <CardContent className="p-5">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-zinc-100 dark:bg-zinc-800">
                <Database className="h-5 w-5 text-zinc-600 dark:text-zinc-400" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Gesamt-Speicher</p>
                <p className="text-xl font-bold">{formatBytes(totalSpace)}</p>
                <p className="text-xs text-muted-foreground">{storages.length} Storage(s)</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card hover>
          <CardContent className="p-5">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-zinc-100 dark:bg-zinc-800">
                <HardDrive className="h-5 w-5 text-zinc-600 dark:text-zinc-400" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Belegt</p>
                <p className="text-xl font-bold">{overallPercent.toFixed(1)}%</p>
                <p className="text-xs text-muted-foreground">{formatBytes(usedSpace)}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card hover>
          <CardContent className="p-5">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-zinc-100 dark:bg-zinc-800">
                <HardDrive className="h-5 w-5 text-zinc-600 dark:text-zinc-400" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Verfügbar</p>
                <p className="text-xl font-bold">{formatBytes(freeSpace)}</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Root filesystem bar */}
      {status && (
        <Card>
          <CardContent className="p-4">
            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="font-medium">Root-Dateisystem</span>
                <span className="text-muted-foreground">
                  {formatBytes(status.disk_used)} / {formatBytes(status.disk_total)}
                </span>
              </div>
              <div className="h-2.5 w-full rounded-full bg-secondary">
                <div
                  className={`h-2.5 rounded-full transition-all ${getUsageBgColor(
                    status.disk_total > 0
                      ? (status.disk_used / status.disk_total) * 100
                      : 0
                  )}`}
                  style={{
                    width: `${Math.min(
                      status.disk_total > 0
                        ? (status.disk_used / status.disk_total) * 100
                        : 0,
                      100
                    )}%`,
                  }}
                />
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* ZFS Pools - prominent display */}
      {zfsPools.length > 0 && (
        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-center gap-2">
              <Layers className="h-4 w-4 text-primary" />
              <CardTitle className="text-base">ZFS Pools</CardTitle>
              <Badge variant="outline" className="text-xs">{zfsPools.length} Pool(s)</Badge>
            </div>
          </CardHeader>
          <CardContent className="space-y-3">
            {zfsPools.map((pool) => {
              const prediction = predictFullDate(pool.usage_percent);
              return (
                <div key={pool.storage} className="rounded-lg border p-4 space-y-2">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <Layers className="h-4 w-4 text-muted-foreground" />
                      <span className="font-medium">{pool.storage}</span>
                      <Badge variant="outline" className="text-xs">
                        {getStorageTypeLabel(pool.type)}
                      </Badge>
                      <Badge variant={pool.active ? "success" : "secondary"} className="text-xs">
                        {pool.active ? "Aktiv" : "Inaktiv"}
                      </Badge>
                    </div>
                    <div className="flex items-center gap-3">
                      <Sparkline data={sparklineData[pool.storage] || []} />
                      <span className="text-sm font-mono">
                        {(pool.usage_percent ?? 0).toFixed(1)}%
                      </span>
                    </div>
                  </div>
                  <div className="h-2.5 w-full rounded-full bg-secondary">
                    <div
                      className={`h-2.5 rounded-full transition-all ${getUsageBgColor(pool.usage_percent)}`}
                      style={{ width: `${Math.min(pool.usage_percent, 100)}%` }}
                    />
                  </div>
                  <div className="flex items-center justify-between text-xs text-muted-foreground">
                    <span>Belegt: {formatBytes(pool.used)}</span>
                    <span>Verfügbar: {formatBytes(pool.available)}</span>
                    <span>Gesamt: {formatBytes(pool.total)}</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <ContentBadges content={pool.content} />
                    {prediction && pool.usage_percent >= 60 && (
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <div className="flex items-center gap-1 text-xs text-muted-foreground">
                            <Clock className="h-3 w-3" />
                            <span>Voll in {prediction}</span>
                          </div>
                        </TooltipTrigger>
                        <TooltipContent>
                          Vorhersage basierend auf linearer Projektion
                        </TooltipContent>
                      </Tooltip>
                    )}
                  </div>
                </div>
              );
            })}
          </CardContent>
        </Card>
      )}

      {/* All Storage Details Table */}
      {sortedStorages.length > 0 && (
        <Card>
          <CardHeader className="py-3 px-4">
            <Button
              variant="ghost"
              className="flex w-full items-center justify-between p-0 h-auto hover:bg-transparent"
              onClick={() => setShowDetails(!showDetails)}
            >
              <CardTitle className="text-base">
                Alle Storage-Pools ({sortedStorages.length})
              </CardTitle>
              {showDetails ? (
                <ChevronDown className="h-4 w-4 text-muted-foreground" />
              ) : (
                <ChevronRight className="h-4 w-4 text-muted-foreground" />
              )}
            </Button>
          </CardHeader>
          {showDetails && (
            <CardContent className="p-0">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/50">
                    <th className="p-3 text-left font-medium">Name</th>
                    <th className="p-3 text-left font-medium">Typ</th>
                    <th className="p-3 text-left font-medium">Inhalt</th>
                    <th className="p-3 text-left font-medium">Status</th>
                    <th className="p-3 text-right font-medium">Trend</th>
                    <th className="p-3 text-right font-medium">Belegt</th>
                    <th className="p-3 text-right font-medium">Gesamt</th>
                    <th className="p-3 text-right font-medium">Nutzung</th>
                    <th className="p-3 text-right font-medium">Vorhersage</th>
                  </tr>
                </thead>
                <tbody>
                  {sortedStorages.map((s) => {
                    const prediction = predictFullDate(s.usage_percent);
                    const isCritical = s.usage_percent >= 90;
                    const isWarning = s.usage_percent >= 80 && s.usage_percent < 90;
                    return (
                      <tr
                        key={s.storage}
                        className={`border-b last:border-0 ${
                          isCritical ? "bg-red-500/5" : isWarning ? "bg-amber-500/5" : ""
                        }`}
                      >
                        <td className="p-3">
                          <div className="flex items-center gap-2">
                            {isCritical && <AlertTriangle className="h-3.5 w-3.5 text-red-500" />}
                            {isWarning && <AlertTriangle className="h-3.5 w-3.5 text-amber-500" />}
                            {!isCritical && !isWarning && (
                              isZfsType(s.type) ? (
                                <Layers className="h-4 w-4 text-primary" />
                              ) : (
                                <Database className="h-4 w-4 text-muted-foreground" />
                              )
                            )}
                            <span className="font-medium">{s.storage}</span>
                          </div>
                        </td>
                        <td className="p-3">
                          <Badge variant={isZfsType(s.type) ? "default" : "outline"} className="text-xs">
                            {getStorageTypeLabel(s.type)}
                          </Badge>
                        </td>
                        <td className="p-3">
                          <ContentBadges content={s.content} />
                        </td>
                        <td className="p-3">
                          <Badge variant={s.active ? "success" : "secondary"} className="text-xs">
                            {s.active ? "Aktiv" : "Inaktiv"}
                          </Badge>
                        </td>
                        <td className="p-3 text-right">
                          <div className="flex justify-end">
                            <Sparkline data={sparklineData[s.storage] || []} />
                          </div>
                        </td>
                        <td className="p-3 text-right">{formatBytes(s.used)}</td>
                        <td className="p-3 text-right">{formatBytes(s.total)}</td>
                        <td className="p-3 text-right">
                          <div className="flex items-center justify-end gap-2">
                            <div className="h-1.5 w-16 rounded-full bg-secondary">
                              <div
                                className={`h-1.5 rounded-full ${getUsageBgColor(s.usage_percent)}`}
                                style={{ width: `${Math.min(s.usage_percent, 100)}%` }}
                              />
                            </div>
                            <span className="w-12 text-right font-mono text-xs">
                              {(s.usage_percent ?? 0).toFixed(1)}%
                            </span>
                          </div>
                        </td>
                        <td className="p-3 text-right text-xs text-muted-foreground">
                          {prediction && s.usage_percent >= 50 ? (
                            <div className="flex items-center justify-end gap-1">
                              <TrendingUp className="h-3 w-3" />
                              <span>{prediction}</span>
                            </div>
                          ) : (
                            <span className="text-green-600">Stabil</span>
                          )}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </CardContent>
          )}
        </Card>
      )}
    </div>
  );
}
