"use client";

import { memo } from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { Server, Cpu, MemoryStick, HardDrive, ChevronDown, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";

interface HostNodeData {
  label: string;
  status: string;
  hostname?: string;
  nodeType?: string;
  cpuUsage?: number;
  memoryPercent?: number;
  vmCount?: number;
  ctCount?: number;
  vmRunning?: number;
  ctRunning?: number;
  expanded?: boolean;
  onToggleExpand?: () => void;
  storageItems?: { label: string; type?: string; usagePercent?: number }[];
  [key: string]: unknown;
}

function getStatusBorderClass(status: string): string {
  switch (status) {
    case "online":
      return "border-green-500/60";
    case "warning":
      return "border-amber-500/60";
    case "critical":
      return "border-red-500/60";
    default:
      return "border-zinc-400/40 dark:border-zinc-600/40";
  }
}

function getStatusDotClass(status: string): string {
  switch (status) {
    case "online":
      return "bg-green-500";
    case "warning":
      return "bg-amber-500";
    case "critical":
      return "bg-red-500";
    default:
      return "bg-zinc-400";
  }
}

function HostNodeComponent({ data }: NodeProps) {
  const d = data as unknown as HostNodeData;
  const online = d.status === "online";
  const expanded = d.expanded !== false;

  return (
    <div
      className={cn(
        "rounded-2xl border-2 bg-card shadow-lg min-w-[200px] transition-all duration-300",
        getStatusBorderClass(d.status),
        online && "shadow-green-500/5 dark:shadow-green-500/10"
      )}
    >
      <Handle type="target" position={Position.Top} className="!bg-muted-foreground !w-2.5 !h-2.5 !border-2 !border-background" />

      <div className="px-4 py-3">
        <div className="flex items-center gap-2.5 mb-1">
          <div className={cn("h-2.5 w-2.5 rounded-full shrink-0 ring-2 ring-offset-1 ring-offset-card", getStatusDotClass(d.status), online ? "ring-green-500/30" : "ring-transparent")} />
          <Server className="h-4 w-4 text-muted-foreground shrink-0" />
          <span className="font-semibold text-sm truncate flex-1">{d.label}</span>
          {d.onToggleExpand && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                d.onToggleExpand?.();
              }}
              className="p-0.5 rounded hover:bg-muted/80 transition-colors"
            >
              {expanded ? (
                <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
              ) : (
                <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
              )}
            </button>
          )}
        </div>

        {d.hostname && (
          <p className="text-xs text-muted-foreground mb-2 truncate pl-5">{d.hostname}</p>
        )}

        {online && (d.cpuUsage !== undefined || d.memoryPercent !== undefined) && (
          <div className="space-y-1.5 mt-2">
            {d.cpuUsage !== undefined && (
              <div className="flex items-center gap-1.5 text-xs">
                <Cpu className="h-3 w-3 text-blue-500 shrink-0" />
                <div className="flex-1 h-1.5 bg-muted rounded-full overflow-hidden">
                  <div
                    className={cn(
                      "h-full rounded-full transition-all duration-500",
                      d.cpuUsage >= 90 ? "bg-red-500" : d.cpuUsage >= 75 ? "bg-amber-500" : "bg-blue-500"
                    )}
                    style={{ width: `${Math.min(d.cpuUsage, 100)}%` }}
                  />
                </div>
                <span className="text-muted-foreground w-10 text-right tabular-nums">{d.cpuUsage.toFixed(1)}%</span>
              </div>
            )}
            {d.memoryPercent !== undefined && (
              <div className="flex items-center gap-1.5 text-xs">
                <MemoryStick className="h-3 w-3 text-purple-500 shrink-0" />
                <div className="flex-1 h-1.5 bg-muted rounded-full overflow-hidden">
                  <div
                    className={cn(
                      "h-full rounded-full transition-all duration-500",
                      d.memoryPercent >= 90 ? "bg-red-500" : d.memoryPercent >= 75 ? "bg-amber-500" : "bg-purple-500"
                    )}
                    style={{ width: `${Math.min(d.memoryPercent, 100)}%` }}
                  />
                </div>
                <span className="text-muted-foreground w-10 text-right tabular-nums">{d.memoryPercent.toFixed(1)}%</span>
              </div>
            )}
          </div>
        )}

        <div className="flex items-center gap-3 mt-2.5 text-xs text-muted-foreground">
          {d.vmCount !== undefined && (
            <span className="flex items-center gap-1">
              <span className="font-medium text-foreground">{d.vmRunning ?? d.vmCount}</span>/{d.vmCount} VMs
            </span>
          )}
          {d.ctCount !== undefined && (
            <span className="flex items-center gap-1">
              <span className="font-medium text-foreground">{d.ctRunning ?? d.ctCount}</span>/{d.ctCount} CTs
            </span>
          )}
        </div>

        {d.storageItems && d.storageItems.length > 0 && (
          <div className="flex items-center gap-2 mt-2 pt-2 border-t border-border/50">
            <HardDrive className="h-3 w-3 text-amber-500 shrink-0" />
            <div className="flex gap-1 flex-wrap">
              {d.storageItems.map((s, i) => (
                <span key={i} className="text-[10px] text-muted-foreground bg-muted/60 rounded px-1.5 py-0.5">
                  {s.label}
                </span>
              ))}
            </div>
          </div>
        )}
      </div>

      <Handle type="source" position={Position.Bottom} className="!bg-muted-foreground !w-2.5 !h-2.5 !border-2 !border-background" />
    </div>
  );
}

export const HostNode = memo(HostNodeComponent);
