"use client";

import { memo } from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { Server, Cpu, MemoryStick } from "lucide-react";
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
  [key: string]: unknown;
}

function HostNodeComponent({ data }: NodeProps) {
  const d = data as unknown as HostNodeData;
  const online = d.status === "online";

  return (
    <div
      className={cn(
        "rounded-xl border-2 bg-card px-4 py-3 shadow-md min-w-[180px]",
        online ? "border-green-500/50" : "border-red-500/50"
      )}
    >
      <Handle type="target" position={Position.Top} className="!bg-muted-foreground !w-2 !h-2" />

      <div className="flex items-center gap-2 mb-2">
        <div className={cn("h-2.5 w-2.5 rounded-full shrink-0", online ? "bg-green-500" : "bg-red-500")} />
        <Server className="h-4 w-4 text-muted-foreground shrink-0" />
        <span className="font-semibold text-sm truncate">{d.label}</span>
      </div>

      {d.hostname && (
        <p className="text-xs text-muted-foreground mb-1 truncate">{d.hostname}</p>
      )}

      {online && (d.cpuUsage !== undefined || d.memoryPercent !== undefined) && (
        <div className="space-y-1 mt-2">
          {d.cpuUsage !== undefined && (
            <div className="flex items-center gap-1.5 text-xs">
              <Cpu className="h-3 w-3 text-blue-500" />
              <div className="flex-1 h-1.5 bg-muted rounded-full overflow-hidden">
                <div
                  className="h-full bg-blue-500 rounded-full transition-all"
                  style={{ width: `${Math.min(d.cpuUsage, 100)}%` }}
                />
              </div>
              <span className="text-muted-foreground w-10 text-right">{d.cpuUsage.toFixed(1)}%</span>
            </div>
          )}
          {d.memoryPercent !== undefined && (
            <div className="flex items-center gap-1.5 text-xs">
              <MemoryStick className="h-3 w-3 text-purple-500" />
              <div className="flex-1 h-1.5 bg-muted rounded-full overflow-hidden">
                <div
                  className="h-full bg-purple-500 rounded-full transition-all"
                  style={{ width: `${Math.min(d.memoryPercent, 100)}%` }}
                />
              </div>
              <span className="text-muted-foreground w-10 text-right">{d.memoryPercent.toFixed(1)}%</span>
            </div>
          )}
        </div>
      )}

      {(d.vmCount !== undefined || d.ctCount !== undefined) && (
        <div className="flex gap-2 mt-2 text-xs text-muted-foreground">
          {d.vmCount !== undefined && <span>{d.vmCount} VMs</span>}
          {d.ctCount !== undefined && <span>{d.ctCount} CTs</span>}
        </div>
      )}

      <Handle type="source" position={Position.Bottom} className="!bg-muted-foreground !w-2 !h-2" />
    </div>
  );
}

export const HostNode = memo(HostNodeComponent);
