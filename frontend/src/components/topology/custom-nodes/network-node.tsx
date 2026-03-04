"use client";

import { Handle, Position, type NodeProps } from "@xyflow/react";
import { Network } from "lucide-react";

export function NetworkNode({ data }: NodeProps) {
  const d = data as { label: string; status: string; metadata?: Record<string, unknown> };

  return (
    <div className="px-3 py-2 rounded-lg border border-orange-400 bg-card shadow-sm min-w-[160px]">
      <Handle type="target" position={Position.Top} className="!bg-slate-400" />
      <div className="flex items-center gap-2">
        <Network className="h-4 w-4 text-orange-500" />
        <div>
          <div className="text-sm font-medium">{d.label}</div>
          <div className="text-xs text-muted-foreground">
            {(d.metadata?.cidr as string) || "Bridge"}
          </div>
        </div>
      </div>
      <Handle type="source" position={Position.Bottom} className="!bg-slate-400" />
    </div>
  );
}
