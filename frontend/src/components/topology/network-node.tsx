"use client";

import { memo } from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { Network, Wifi } from "lucide-react";
import { cn } from "@/lib/utils";

interface NetworkNodeData {
  label: string;
  status: string;
  cidr?: string;
  bridgeType?: string;
  vmCount?: number;
  highlighted?: boolean;
  dimmed?: boolean;
  [key: string]: unknown;
}

function NetworkNodeComponent({ data }: NodeProps) {
  const d = data as unknown as NetworkNodeData;
  const isActive = d.status === "active" || d.status === "online";
  const Icon = d.bridgeType === "wifi" ? Wifi : Network;

  return (
    <div
      className={cn(
        "rounded-xl border bg-card px-3 py-2 shadow-sm min-w-[130px] transition-all duration-300",
        isActive ? "border-cyan-500/50" : "border-muted",
        d.highlighted && "ring-2 ring-blue-500 ring-offset-1 ring-offset-background shadow-md shadow-blue-500/20",
        d.dimmed && "opacity-30"
      )}
    >
      <Handle type="target" position={Position.Left} className="!bg-cyan-500 !w-2 !h-2 !border-2 !border-background" />

      <div className="flex items-center gap-2">
        <div className={cn("h-2 w-2 rounded-full shrink-0", isActive ? "bg-cyan-500" : "bg-zinc-400")} />
        <Icon className="h-3.5 w-3.5 text-cyan-500 shrink-0" />
        <span className="text-xs font-medium truncate">{d.label}</span>
      </div>

      {d.cidr && (
        <p className="text-[10px] text-muted-foreground mt-0.5 font-mono">{d.cidr}</p>
      )}

      {d.vmCount !== undefined && d.vmCount > 0 && (
        <p className="text-[10px] text-muted-foreground mt-0.5">{d.vmCount} Geräte</p>
      )}

      <Handle type="source" position={Position.Right} className="!bg-cyan-500 !w-2 !h-2 !border-2 !border-background" />
    </div>
  );
}

export const NetworkNode = memo(NetworkNodeComponent);
