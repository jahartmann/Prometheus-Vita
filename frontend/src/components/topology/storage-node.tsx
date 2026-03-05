"use client";

import { memo } from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { HardDrive, Database } from "lucide-react";
import { cn } from "@/lib/utils";

interface StorageNodeData {
  label: string;
  status: string;
  storageType?: string;
  total?: number;
  used?: number;
  highlighted?: boolean;
  dimmed?: boolean;
  [key: string]: unknown;
}

function formatSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
}

function StorageNodeComponent({ data }: NodeProps) {
  const d = data as unknown as StorageNodeData;
  const usagePercent = d.total && d.used ? (d.used / d.total) * 100 : 0;
  const Icon = d.storageType === "zfspool" || d.storageType === "lvm" ? Database : HardDrive;

  return (
    <div
      className={cn(
        "rounded-xl border bg-card/95 dark:bg-card/90 px-3 py-2 shadow-sm min-w-[140px] transition-all duration-300",
        usagePercent >= 90 ? "border-red-500/50" : usagePercent >= 75 ? "border-amber-500/50" : "border-amber-500/40",
        d.highlighted && "ring-2 ring-blue-500 ring-offset-1 ring-offset-background shadow-md shadow-blue-500/20",
        d.dimmed && "opacity-30"
      )}
    >
      <Handle type="target" position={Position.Top} className="!bg-amber-500 !w-2 !h-2 !border-2 !border-background" />

      <div className="flex items-center gap-2">
        <Icon className={cn(
          "h-3.5 w-3.5 shrink-0",
          usagePercent >= 90 ? "text-red-500" : usagePercent >= 75 ? "text-amber-500" : "text-amber-500"
        )} />
        <span className="text-xs font-medium truncate">{d.label}</span>
      </div>

      {d.storageType && (
        <span className="text-[10px] text-muted-foreground uppercase tracking-wider">{d.storageType}</span>
      )}

      {d.total !== undefined && d.total > 0 && (
        <div className="mt-1.5">
          <div className="h-1.5 bg-muted rounded-full overflow-hidden">
            <div
              className={cn(
                "h-full rounded-full transition-all duration-500",
                usagePercent >= 90 ? "bg-red-500" : usagePercent >= 75 ? "bg-amber-500" : "bg-amber-500"
              )}
              style={{ width: `${Math.min(usagePercent, 100)}%` }}
            />
          </div>
          <p className="text-[10px] text-muted-foreground mt-0.5 tabular-nums">
            {formatSize(d.used ?? 0)} / {formatSize(d.total)}
          </p>
        </div>
      )}

      <Handle type="source" position={Position.Bottom} className="!bg-amber-500 !w-2 !h-2 !border-2 !border-background" />
    </div>
  );
}

export const StorageNode = memo(StorageNodeComponent);
