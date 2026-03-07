"use client";

import { memo } from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { Monitor, Box } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

interface VMNodeData {
  label: string;
  status: string;
  vmType: string;
  vmid?: number;
  cpuUsage?: number;
  memoryPercent?: number;
  tags?: string[];
  highlighted?: boolean;
  dimmed?: boolean;
  [key: string]: unknown;
}

function getHeatmapBg(cpu?: number, mem?: number): string {
  const val = Math.max(cpu ?? 0, mem ?? 0);
  if (val >= 90) return "bg-red-500/10 dark:bg-red-500/15";
  if (val >= 75) return "bg-amber-500/8 dark:bg-amber-500/12";
  if (val >= 50) return "bg-yellow-500/6 dark:bg-yellow-500/10";
  if (val > 0) return "bg-green-500/5 dark:bg-green-500/8";
  return "";
}

function getStatusBorderClass(status: string): string {
  switch (status) {
    case "running":
      return "border-green-500/50";
    case "paused":
    case "suspended":
      return "border-amber-500/50";
    case "stopped":
      return "border-red-500/40";
    default:
      return "border-muted";
  }
}

function getStatusDotClass(status: string): string {
  switch (status) {
    case "running":
      return "bg-green-500";
    case "paused":
    case "suspended":
      return "bg-amber-500";
    case "stopped":
      return "bg-red-500";
    default:
      return "bg-zinc-400";
  }
}

function getStatusLabel(status: string): string {
  switch (status) {
    case "running":
      return "Aktiv";
    case "paused":
      return "Pausiert";
    case "suspended":
      return "Suspendiert";
    case "stopped":
      return "Gestoppt";
    default:
      return status;
  }
}

function VMNodeComponent({ data }: NodeProps) {
  const d = data as unknown as VMNodeData;
  const running = d.status === "running";
  const isContainer = d.vmType === "ct" || d.vmType === "lxc";
  const Icon = isContainer ? Box : Monitor;

  return (
    <div
      className={cn(
        "rounded-xl border bg-card px-3 py-2 shadow-sm min-w-[150px] max-w-[180px] transition-all duration-300",
        getStatusBorderClass(d.status),
        getHeatmapBg(d.cpuUsage, d.memoryPercent),
        d.highlighted && "ring-2 ring-blue-500 ring-offset-1 ring-offset-background shadow-md shadow-blue-500/20",
        d.dimmed && "opacity-30"
      )}
    >
      <Handle type="target" position={Position.Top} className="!bg-muted-foreground !w-1.5 !h-1.5" />

      <div className="flex items-center gap-2">
        <div className={cn("h-2 w-2 rounded-full shrink-0", getStatusDotClass(d.status))} />
        <Icon className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
        <span className="text-xs font-medium truncate flex-1">{d.label}</span>
      </div>

      <div className="flex items-center gap-1.5 mt-1.5 flex-wrap">
        <Badge variant="outline" className="text-[10px] px-1.5 py-0 h-4">
          {isContainer ? "CT" : "VM"}
        </Badge>
        {d.vmid !== undefined && (
          <span className="text-[10px] text-muted-foreground tabular-nums">#{d.vmid}</span>
        )}
        <Badge
          variant={running ? "success" : d.status === "paused" ? "warning" : "secondary"}
          className="text-[10px] px-1.5 py-0 h-4 ml-auto"
        >
          {getStatusLabel(d.status)}
        </Badge>
      </div>

      {running && (d.cpuUsage !== undefined || d.memoryPercent !== undefined) && (
        <div className="flex items-center gap-2 mt-1.5 text-[10px] text-muted-foreground">
          {d.cpuUsage !== undefined && (
            <span className="tabular-nums">CPU {d.cpuUsage.toFixed(0)}%</span>
          )}
          {d.memoryPercent !== undefined && (
            <span className="tabular-nums">RAM {d.memoryPercent.toFixed(0)}%</span>
          )}
        </div>
      )}

      {d.tags && d.tags.length > 0 && (
        <div className="flex gap-1 mt-1 flex-wrap">
          {d.tags.slice(0, 3).map((tag) => (
            <span key={tag} className="text-[9px] bg-muted/80 text-muted-foreground rounded px-1 py-0.5">
              {tag}
            </span>
          ))}
        </div>
      )}

      <Handle type="source" position={Position.Bottom} className="!bg-muted-foreground !w-1.5 !h-1.5" />
    </div>
  );
}

export const VMNode = memo(VMNodeComponent);
