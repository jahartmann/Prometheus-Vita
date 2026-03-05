"use client";

import Link from "next/link";
import { Server, Cpu, MemoryStick, HardDrive, AlertTriangle } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useNodeStore } from "@/stores/node-store";
import type { Node, NodeStatus } from "@/types/api";
import {
  formatBytes,
  formatUptime,
  formatPercentage,
  getUsageBgColor,
} from "@/lib/utils";

interface NodeCardProps {
  node: Node;
  status?: NodeStatus;
}

function UsageBar({ value, label }: { value: number; label: string }) {
  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between text-xs">
        <span className="text-muted-foreground">{label}</span>
        <span className="font-medium">{formatPercentage(value)}</span>
      </div>
      <div className="h-1.5 w-full rounded-full bg-secondary">
        <div
          className={`h-1.5 rounded-full transition-all ${getUsageBgColor(value)}`}
          style={{ width: `${Math.min(value, 100)}%` }}
        />
      </div>
    </div>
  );
}

export function NodeCard({ node, status }: NodeCardProps) {
  const { nodeErrors } = useNodeStore();
  const nodeError = nodeErrors[node.id];
  const cpuUsage = status?.cpu_usage ?? 0;
  const memUsage = status && status.memory_total > 0
    ? (status.memory_used / status.memory_total) * 100
    : 0;
  const diskUsage = status && status.disk_total > 0
    ? (status.disk_used / status.disk_total) * 100
    : 0;

  return (
    <Link href={`/nodes/${node.id}`}>
      <Card hover className="cursor-pointer">
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10">
                <Server className="h-4 w-4 text-primary" />
              </div>
              <CardTitle className="text-base">{node.name}</CardTitle>
            </div>
            <Badge variant={node.is_online ? "success" : "destructive"}>
              {node.is_online ? "Online" : "Offline"}
            </Badge>
          </div>
          <div className="flex items-center gap-2 text-xs text-muted-foreground pl-10">
            <span>{node.hostname}:{node.port}</span>
          </div>
        </CardHeader>
        <CardContent className="space-y-3">
          {nodeError ? (
            <div className="flex items-center gap-2 py-3 text-xs text-amber-600 dark:text-amber-400">
              <AlertTriangle className="h-3.5 w-3.5 shrink-0" />
              <span>Nicht erreichbar</span>
            </div>
          ) : status ? (
            <>
              <div className="flex items-center gap-4 text-xs text-muted-foreground">
                <span className="flex items-center gap-1">
                  <Cpu className="h-3 w-3" />
                  {status.cpu_cores} Cores
                </span>
                <span className="flex items-center gap-1">
                  <MemoryStick className="h-3 w-3" />
                  {formatBytes(status.memory_total)}
                </span>
                <span className="flex items-center gap-1">
                  <HardDrive className="h-3 w-3" />
                  {formatBytes(status.disk_total)}
                </span>
              </div>
              <UsageBar value={cpuUsage} label="CPU" />
              <UsageBar value={memUsage} label="RAM" />
              <UsageBar value={diskUsage} label="Disk" />
              <div className="flex items-center justify-between text-xs text-muted-foreground">
                <span>Uptime: {formatUptime(status.uptime)}</span>
                <div className="flex gap-2">
                  <span>{status.vm_running ?? 0}/{status.vm_count ?? 0} VMs</span>
                  <span>{status.ct_running ?? 0}/{status.ct_count ?? 0} CTs</span>
                </div>
              </div>
            </>
          ) : (
            <div className="py-4 text-center text-sm text-muted-foreground">
              {node.is_online
                ? "Lade Status..."
                : "Server ist offline"}
            </div>
          )}
        </CardContent>
      </Card>
    </Link>
  );
}
