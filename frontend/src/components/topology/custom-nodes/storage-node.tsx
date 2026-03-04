"use client";

import { Handle, Position, type NodeProps } from "@xyflow/react";
import { HardDrive } from "lucide-react";

export function StorageNode({ data }: NodeProps) {
  const d = data as { label: string; status: string; metadata?: Record<string, unknown> };

  const total = (d.metadata?.total as number) || 0;
  const used = (d.metadata?.used as number) || 0;
  const usagePercent = total > 0 ? (used / total) * 100 : 0;

  return (
    <div className="px-3 py-2 rounded-lg border border-purple-400 bg-card shadow-sm min-w-[160px]">
      <Handle type="target" position={Position.Top} className="!bg-slate-400" />
      <div className="flex items-center gap-2">
        <HardDrive className="h-4 w-4 text-purple-500" />
        <div>
          <div className="text-sm font-medium">{d.label}</div>
          <div className="text-xs text-muted-foreground">
            {(d.metadata?.type as string) || "Storage"}
          </div>
        </div>
      </div>
      {total > 0 && (
        <div className="mt-1">
          <div className="w-full bg-gray-200 rounded-full h-1.5">
            <div
              className={`h-1.5 rounded-full ${usagePercent > 80 ? "bg-red-500" : "bg-purple-500"}`}
              style={{ width: `${Math.min(usagePercent, 100)}%` }}
            />
          </div>
          <div className="text-xs text-muted-foreground mt-0.5">
            {(usagePercent ?? 0).toFixed(1)}% belegt
          </div>
        </div>
      )}
      <Handle type="source" position={Position.Bottom} className="!bg-slate-400" />
    </div>
  );
}
