"use client";

import { Handle, Position, type NodeProps } from "@xyflow/react";
import { Monitor } from "lucide-react";

export function VMNode({ data }: NodeProps) {
  const d = data as { label: string; status: string; nodeType: string; metadata?: Record<string, unknown> };
  const isRunning = d.status === "running";
  const isCT = d.nodeType === "ct";

  return (
    <div className={`px-3 py-2 rounded-lg border bg-card shadow-sm min-w-[160px] ${
      isRunning ? "border-blue-400" : "border-gray-300"
    }`}>
      <Handle type="target" position={Position.Top} className="!bg-slate-400" />
      <div className="flex items-center gap-2">
        <Monitor className={`h-4 w-4 ${isRunning ? "text-blue-500" : "text-gray-400"}`} />
        <div>
          <div className="text-sm font-medium">{d.label || `VM ${d.metadata?.vmid}`}</div>
          <div className="text-xs text-muted-foreground">
            {isCT ? "Container" : "VM"} {d.metadata?.vmid ? `#${d.metadata.vmid}` : ""}
          </div>
        </div>
      </div>
      <div className={`mt-1 text-xs ${isRunning ? "text-blue-600" : "text-gray-500"}`}>
        {isRunning ? "Laeuft" : "Gestoppt"}
      </div>
      <Handle type="source" position={Position.Bottom} className="!bg-slate-400" />
    </div>
  );
}
