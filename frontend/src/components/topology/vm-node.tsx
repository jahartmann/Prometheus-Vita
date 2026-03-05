"use client";

import { memo } from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { Monitor, Box } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

interface VMNodeData {
  label: string;
  status: string;
  vmType: string;
  vmid?: number;
  [key: string]: unknown;
}

function VMNodeComponent({ data }: NodeProps) {
  const d = data as unknown as VMNodeData;
  const running = d.status === "running";
  const isContainer = d.vmType === "ct" || d.vmType === "lxc";
  const Icon = isContainer ? Box : Monitor;

  return (
    <div
      className={cn(
        "rounded-lg border bg-card px-3 py-2 shadow-sm min-w-[140px]",
        running ? "border-green-500/40" : "border-muted"
      )}
    >
      <Handle type="target" position={Position.Top} className="!bg-muted-foreground !w-2 !h-2" />

      <div className="flex items-center gap-2">
        <div className={cn("h-2 w-2 rounded-full shrink-0", running ? "bg-green-500" : "bg-zinc-400")} />
        <Icon className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
        <span className="text-xs font-medium truncate flex-1">{d.label}</span>
      </div>

      <div className="flex items-center gap-1.5 mt-1.5">
        <Badge variant="outline" className="text-[10px] px-1.5 py-0">
          {isContainer ? "CT" : "VM"}
        </Badge>
        {d.vmid !== undefined && (
          <span className="text-[10px] text-muted-foreground">ID: {d.vmid}</span>
        )}
        <Badge
          variant={running ? "success" : "secondary"}
          className="text-[10px] px-1.5 py-0 ml-auto"
        >
          {running ? "Running" : "Stopped"}
        </Badge>
      </div>

      <Handle type="source" position={Position.Bottom} className="!bg-muted-foreground !w-2 !h-2" />
    </div>
  );
}

export const VMNode = memo(VMNodeComponent);
