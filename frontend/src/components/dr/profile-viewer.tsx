"use client";

import type { NodeProfile } from "@/types/api";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Cpu, HardDrive, MemoryStick, Network } from "lucide-react";

interface ProfileViewerProps {
  profile: NodeProfile;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
}

export function ProfileViewer({ profile }: ProfileViewerProps) {
  return (
    <div className="grid gap-4 md:grid-cols-2">
      <Card>
        <CardHeader className="flex flex-row items-center gap-2 pb-2">
          <Cpu className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-sm font-medium">CPU</CardTitle>
        </CardHeader>
        <CardContent className="space-y-1 text-sm">
          <div className="flex justify-between">
            <span className="text-muted-foreground">Modell</span>
            <span>{profile.cpu_model || "Unbekannt"}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Kerne</span>
            <span>{profile.cpu_cores || "-"}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Threads</span>
            <span>{profile.cpu_threads || "-"}</span>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center gap-2 pb-2">
          <MemoryStick className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-sm font-medium">Arbeitsspeicher</CardTitle>
        </CardHeader>
        <CardContent className="space-y-1 text-sm">
          <div className="flex justify-between">
            <span className="text-muted-foreground">Gesamt</span>
            <span>{profile.memory_total_bytes ? formatBytes(profile.memory_total_bytes) : "-"}</span>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center gap-2 pb-2">
          <HardDrive className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-sm font-medium">System</CardTitle>
        </CardHeader>
        <CardContent className="space-y-1 text-sm">
          <div className="flex justify-between">
            <span className="text-muted-foreground">PVE Version</span>
            <span>{profile.pve_version || "-"}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Kernel</span>
            <span>{profile.kernel_version || "-"}</span>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center gap-2 pb-2">
          <Network className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-sm font-medium">Erfassung</CardTitle>
        </CardHeader>
        <CardContent className="space-y-1 text-sm">
          <div className="flex justify-between">
            <span className="text-muted-foreground">Erfasst am</span>
            <span>{new Date(profile.collected_at).toLocaleString("de-DE")}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Pakete</span>
            <Badge variant="secondary">
              {Array.isArray(profile.installed_packages) ? profile.installed_packages.length : "-"}
            </Badge>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
