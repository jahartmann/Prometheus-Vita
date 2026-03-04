"use client";

import { Handle, Position, type NodeProps } from "@xyflow/react";
import { Server } from "lucide-react";

export function NodeNode({ data }: NodeProps) {
  const d = data as { label: string; status: string; metadata?: Record<string, unknown> };
  const isOnline = d.status === "online";

  return (
    <div className={`px-4 py-3 rounded-lg border-2 bg-card shadow-md min-w-[180px] ${
      isOnline ? "border-green-500" : "border-red-500"
    }`}>
      <Handle type="target" position={Position.Top} className="!bg-slate-400" />
      <div className="flex items-center gap-2">
        <Server className={`h-5 w-5 ${isOnline ? "text-green-500" : "text-red-500"}`} />
        <div>
          <div className="font-medium text-sm">{d.label}</div>
          <div className="text-xs text-muted-foreground">
            {(d.metadata?.hostname as string) || "Host"}
          </div>
        </div>
      </div>
      <div className={`mt-1 text-xs ${isOnline ? "text-green-600" : "text-red-600"}`}>
        {isOnline ? "Online" : "Offline"}
      </div>
      <Handle type="source" position={Position.Bottom} className="!bg-slate-400" />
    </div>
  );
}
