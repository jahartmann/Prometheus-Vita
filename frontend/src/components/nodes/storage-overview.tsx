"use client";

import { useEffect, useState } from "react";
import { Database } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
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

export function StorageOverview({ nodeId, status }: StorageOverviewProps) {
  const [storages, setStorages] = useState<StorageItem[]>([]);

  useEffect(() => {
    api
      .get(`/nodes/${nodeId}/storage`)
      .then((res) => {
        setStorages(toArray<StorageItem>(res.data));
      })
      .catch(() => {});
  }, [nodeId]);

  return (
    <div className="space-y-4">
      {status && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Root-Dateisystem</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span>Belegt</span>
                <span>
                  {formatBytes(status.disk_used)} / {formatBytes(status.disk_total)}
                </span>
              </div>
              <div className="h-2 w-full rounded-full bg-secondary">
                <div
                  className={`h-2 rounded-full transition-all ${getUsageBgColor(
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

      {storages.length > 0 && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {storages.map((s) => (
            <Card key={s.storage}>
              <CardContent className="p-4">
                <div className="flex items-center gap-2 mb-3">
                  <Database className="h-4 w-4 text-primary" />
                  <span className="font-medium">{s.storage}</span>
                  <span className="text-xs text-muted-foreground">({s.type})</span>
                </div>
                <div className="space-y-2">
                  <div className="flex justify-between text-xs text-muted-foreground">
                    <span>{formatBytes(s.used)}</span>
                    <span>{formatBytes(s.total)}</span>
                  </div>
                  <div className="h-1.5 w-full rounded-full bg-secondary">
                    <div
                      className={`h-1.5 rounded-full ${getUsageBgColor(s.usage_percent)}`}
                      style={{ width: `${Math.min(s.usage_percent, 100)}%` }}
                    />
                  </div>
                  <p className="text-xs text-muted-foreground">{s.content}</p>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
