"use client";

import { HardDrive } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { formatBytes } from "@/lib/utils";
import type { DiskInfo } from "@/types/api";

interface DiskListProps {
  disks: DiskInfo[];
}

export function DiskList({ disks }: DiskListProps) {
  if (disks.length === 0) return null;

  const healthColor = (health: string | undefined) => {
    if (!health) return "outline" as const;
    if (health === "PASSED" || health === "OK") return "success" as const;
    if (health === "UNKNOWN") return "outline" as const;
    return "destructive" as const;
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Physische Disks</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {disks.map((disk) => (
            <div
              key={disk.devpath}
              className="flex items-center justify-between rounded border p-3"
            >
              <div className="flex items-center gap-3">
                <HardDrive className="h-5 w-5 text-muted-foreground" />
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-sm">{disk.devpath}</span>
                    <Badge variant="outline">{disk.type.toUpperCase()}</Badge>
                    {disk.health && (
                      <Badge variant={healthColor(disk.health)}>{disk.health}</Badge>
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground">
                    {disk.model || disk.vendor || "Unbekannt"} | {formatBytes(disk.size)}
                    {disk.serial && ` | S/N: ${disk.serial}`}
                    {disk.wearout && ` | Wearout: ${disk.wearout}`}
                  </p>
                </div>
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
