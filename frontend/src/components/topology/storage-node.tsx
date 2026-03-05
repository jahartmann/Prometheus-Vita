"use client";

import { memo } from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { HardDrive } from "lucide-react";
import { cn } from "@/lib/utils";

interface StorageNodeData {
  label: string;
  status: string;
  storageType?: string;
  total?: number;
  used?: number;
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

  return (
    <div className={cn("rounded-lg border bg-card px-3 py-2 shadow-sm min-w-[130px]", "border-amber-500/40")}>
      <Handle type="target" position={Position.Top} className="!bg-muted-foreground !w-2 !h-2" />

      <div className="flex items-center gap-2">
        <HardDrive className="h-3.5 w-3.5 text-amber-500 shrink-0" />
        <span className="text-xs font-medium truncate">{d.label}</span>
      </div>

      {d.storageType && (
        <span className="text-[10px] text-muted-foreground">{d.storageType}</span>
      )}

      {d.total !== undefined && d.total > 0 && (
        <div className="mt-1.5">
          <div className="h-1.5 bg-muted rounded-full overflow-hidden">
            <div
              className="h-full bg-amber-500 rounded-full"
              style={{ width: `${Math.min(usagePercent, 100)}%` }}
            />
          </div>
          <p className="text-[10px] text-muted-foreground mt-0.5">
            {formatSize(d.used ?? 0)} / {formatSize(d.total)}
          </p>
        </div>
      )}

      <Handle type="source" position={Position.Bottom} className="!bg-muted-foreground !w-2 !h-2" />
    </div>
  );
}

export const StorageNode = memo(StorageNodeComponent);
