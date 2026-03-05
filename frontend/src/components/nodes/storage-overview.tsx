"use client";

import { useEffect, useState } from "react";
import { Database, HardDrive, ChevronDown, ChevronRight, Layers } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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

function getStorageTypeLabel(type: string): string {
  return storageTypeLabels[type] || type.toUpperCase();
}

function isZfsType(type: string): boolean {
  return type === "zfspool" || type === "zfs";
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

  const zfsPools = storages.filter((s) => isZfsType(s.type));
  const otherStorages = storages.filter((s) => !isZfsType(s.type));

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
        <Card hover className="gradient-blue">
          <CardContent className="p-5">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-kpi-blue/15">
                <Database className="h-5 w-5 text-kpi-blue" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Gesamt-Speicher</p>
                <p className="text-xl font-bold">{formatBytes(totalSpace)}</p>
                <p className="text-xs text-muted-foreground">{storages.length} Storage(s)</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card hover className="gradient-orange">
          <CardContent className="p-5">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-kpi-orange/15">
                <HardDrive className="h-5 w-5 text-kpi-orange" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Belegt</p>
                <p className="text-xl font-bold">{overallPercent.toFixed(1)}%</p>
                <p className="text-xs text-muted-foreground">{formatBytes(usedSpace)}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card hover className="gradient-green">
          <CardContent className="p-5">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-kpi-green/15">
                <HardDrive className="h-5 w-5 text-kpi-green" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Verfuegbar</p>
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
            {zfsPools.map((pool) => (
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
                  <span className="text-sm font-mono">
                    {(pool.usage_percent ?? 0).toFixed(1)}%
                  </span>
                </div>
                <div className="h-2.5 w-full rounded-full bg-secondary">
                  <div
                    className={`h-2.5 rounded-full transition-all ${getUsageBgColor(pool.usage_percent)}`}
                    style={{ width: `${Math.min(pool.usage_percent, 100)}%` }}
                  />
                </div>
                <div className="flex items-center justify-between text-xs text-muted-foreground">
                  <span>Belegt: {formatBytes(pool.used)}</span>
                  <span>Verfuegbar: {formatBytes(pool.available)}</span>
                  <span>Gesamt: {formatBytes(pool.total)}</span>
                </div>
                {pool.content && (
                  <p className="text-xs text-muted-foreground">Inhalt: {pool.content}</p>
                )}
              </div>
            ))}
          </CardContent>
        </Card>
      )}

      {/* All Storage Details Table */}
      {storages.length > 0 && (
        <Card>
          <CardHeader className="py-3 px-4">
            <Button
              variant="ghost"
              className="flex w-full items-center justify-between p-0 h-auto hover:bg-transparent"
              onClick={() => setShowDetails(!showDetails)}
            >
              <CardTitle className="text-base">
                Alle Storage-Pools ({storages.length})
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
                    <th className="p-3 text-right font-medium">Belegt</th>
                    <th className="p-3 text-right font-medium">Gesamt</th>
                    <th className="p-3 text-right font-medium">Nutzung</th>
                  </tr>
                </thead>
                <tbody>
                  {storages.map((s) => (
                    <tr key={s.storage} className="border-b last:border-0">
                      <td className="p-3">
                        <div className="flex items-center gap-2">
                          {isZfsType(s.type) ? (
                            <Layers className="h-4 w-4 text-primary" />
                          ) : (
                            <Database className="h-4 w-4 text-muted-foreground" />
                          )}
                          <span className="font-medium">{s.storage}</span>
                        </div>
                      </td>
                      <td className="p-3">
                        <Badge variant={isZfsType(s.type) ? "default" : "outline"} className="text-xs">
                          {getStorageTypeLabel(s.type)}
                        </Badge>
                      </td>
                      <td className="p-3 text-muted-foreground">{s.content}</td>
                      <td className="p-3">
                        <Badge variant={s.active ? "success" : "secondary"} className="text-xs">
                          {s.active ? "Aktiv" : "Inaktiv"}
                        </Badge>
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
                    </tr>
                  ))}
                </tbody>
              </table>
            </CardContent>
          )}
        </Card>
      )}
    </div>
  );
}
