"use client";

import { memo } from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { Network } from "lucide-react";
import { cn } from "@/lib/utils";

interface NetworkNodeData {
  label: string;
  status: string;
  cidr?: string;
  [key: string]: unknown;
}

function NetworkNodeComponent({ data }: NodeProps) {
  const d = data as unknown as NetworkNodeData;

  return (
    <div className={cn("rounded-lg border bg-card px-3 py-2 shadow-sm min-w-[120px]", "border-cyan-500/40")}>
      <Handle type="target" position={Position.Top} className="!bg-muted-foreground !w-2 !h-2" />

      <div className="flex items-center gap-2">
        <Network className="h-3.5 w-3.5 text-cyan-500 shrink-0" />
        <span className="text-xs font-medium truncate">{d.label}</span>
      </div>

      {d.cidr && (
        <p className="text-[10px] text-muted-foreground mt-0.5">{d.cidr}</p>
      )}

      <Handle type="source" position={Position.Bottom} className="!bg-muted-foreground !w-2 !h-2" />
    </div>
  );
}

export const NetworkNode = memo(NetworkNodeComponent);
