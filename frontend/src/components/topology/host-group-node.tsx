"use client";

import { memo } from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { Server, Cpu, MemoryStick, ChevronDown, ChevronRight, HardDrive } from "lucide-react";
import { cn } from "@/lib/utils";

interface HostGroupNodeData {
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
  onToggleExpand?: (nodeId: string) => void;
  nodeId?: string;
  containerWidth?: number;
  containerHeight?: number;
  storageItems?: { label: string; type?: string; usagePercent?: number }[];
  [key: string]: unknown;
}

function getStatusBorderClass(status: string): string {
  switch (status) {
    case "online":
      return "border-green-500/40 dark:border-green-500/30";
    case "warning":
      return "border-amber-500/40 dark:border-amber-500/30";
    case "critical":
      return "border-red-500/40 dark:border-red-500/30";
    default:
      return "border-zinc-300/60 dark:border-zinc-700/60";
  }
}

function getStatusGlow(status: string): string {
  switch (status) {
    case "online":
      return "shadow-green-500/5 dark:shadow-green-500/8";
    case "warning":
      return "shadow-amber-500/5 dark:shadow-amber-500/8";
    case "critical":
      return "shadow-red-500/5 dark:shadow-red-500/8";
    default:
      return "";
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

function HostGroupNodeComponent({ data, id }: NodeProps) {
  const d = data as unknown as HostGroupNodeData;
  const online = d.status === "online";
  const expanded = d.expanded !== false;

  return (
    <div
      className={cn(
        "rounded-2xl border-2 bg-muted/20 dark:bg-muted/10 shadow-lg transition-all duration-300 relative",
        getStatusBorderClass(d.status),
        getStatusGlow(d.status)
      )}
      style={{
        width: d.containerWidth ?? 400,
        height: d.containerHeight ?? 300,
      }}
    >
      <Handle type="target" position={Position.Top} className="!bg-muted-foreground !w-3 !h-3 !border-2 !border-background !-top-1.5" />

      <div className="px-4 py-3 bg-card rounded-t-2xl border-b border-border/30">
        <div className="flex items-center gap-2.5">
          <div className={cn(
            "h-3 w-3 rounded-full shrink-0 ring-2 ring-offset-1 ring-offset-card",
            getStatusDotClass(d.status),
            online ? "ring-green-500/30 animate-pulse" : "ring-transparent"
          )} />
          <Server className="h-4.5 w-4.5 text-muted-foreground shrink-0" />
          <span className="font-bold text-sm truncate flex-1">{d.label}</span>
          {d.onToggleExpand && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                d.onToggleExpand?.(id);
              }}
              className="p-1 rounded-lg hover:bg-muted/80 transition-colors"
              title={expanded ? "Einklappen" : "Ausklappen"}
            >
              {expanded ? (
                <ChevronDown className="h-4 w-4 text-muted-foreground" />
              ) : (
                <ChevronRight className="h-4 w-4 text-muted-foreground" />
              )}
            </button>
          )}
        </div>

        <div className="flex items-center gap-4 mt-2">
          {d.hostname && (
            <span className="text-xs text-muted-foreground truncate">{d.hostname}</span>
          )}
          <div className="flex items-center gap-3 ml-auto">
            {d.vmCount !== undefined && (
              <span className="text-xs text-muted-foreground">
                <span className="font-semibold text-foreground">{d.vmRunning ?? d.vmCount}</span>/{d.vmCount} VMs
              </span>
            )}
            {d.ctCount !== undefined && (
              <span className="text-xs text-muted-foreground">
                <span className="font-semibold text-foreground">{d.ctRunning ?? d.ctCount}</span>/{d.ctCount} CTs
              </span>
            )}
          </div>
        </div>

        {online && (d.cpuUsage !== undefined || d.memoryPercent !== undefined) && (
          <div className="flex items-center gap-4 mt-2">
            {d.cpuUsage !== undefined && (
              <div className="flex items-center gap-1.5 text-xs flex-1">
                <Cpu className="h-3 w-3 text-blue-500 shrink-0" />
                <div className="flex-1 h-1.5 bg-muted rounded-full overflow-hidden max-w-[120px]">
                  <div
                    className={cn(
                      "h-full rounded-full transition-all duration-500",
                      d.cpuUsage >= 90 ? "bg-red-500" : d.cpuUsage >= 75 ? "bg-amber-500" : "bg-blue-500"
                    )}
                    style={{ width: `${Math.min(d.cpuUsage, 100)}%` }}
                  />
                </div>
                <span className="text-muted-foreground tabular-nums">{d.cpuUsage.toFixed(1)}%</span>
              </div>
            )}
            {d.memoryPercent !== undefined && (
              <div className="flex items-center gap-1.5 text-xs flex-1">
                <MemoryStick className="h-3 w-3 text-purple-500 shrink-0" />
                <div className="flex-1 h-1.5 bg-muted rounded-full overflow-hidden max-w-[120px]">
                  <div
                    className={cn(
                      "h-full rounded-full transition-all duration-500",
                      d.memoryPercent >= 90 ? "bg-red-500" : d.memoryPercent >= 75 ? "bg-amber-500" : "bg-purple-500"
                    )}
                    style={{ width: `${Math.min(d.memoryPercent, 100)}%` }}
                  />
                </div>
                <span className="text-muted-foreground tabular-nums">{d.memoryPercent.toFixed(1)}%</span>
              </div>
            )}
          </div>
        )}

        {d.storageItems && d.storageItems.length > 0 && (
          <div className="flex items-center gap-2 mt-2 pt-2 border-t border-border/30">
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

      <Handle type="source" position={Position.Bottom} className="!bg-muted-foreground !w-3 !h-3 !border-2 !border-background !-bottom-1.5" />
    </div>
  );
}

export const HostGroupNode = memo(HostGroupNodeComponent);
