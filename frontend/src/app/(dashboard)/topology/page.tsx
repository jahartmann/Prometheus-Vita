"use client";

import { useEffect } from "react";
import Link from "next/link";
import {
  Server,
  Monitor,
  Cpu,
  MemoryStick,
  HardDrive,
  Circle,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useNodeStore } from "@/stores/node-store";
import type { VM } from "@/types/api";

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
}

function statusColor(status: VM["status"]): string {
  switch (status) {
    case "running":
      return "text-green-500";
    case "stopped":
      return "text-muted-foreground";
    case "paused":
      return "text-yellow-500";
    case "suspended":
      return "text-blue-500";
    default:
      return "text-muted-foreground";
  }
}

function statusLabel(status: VM["status"]): string {
  switch (status) {
    case "running":
      return "Laufend";
    case "stopped":
      return "Gestoppt";
    case "paused":
      return "Pausiert";
    case "suspended":
      return "Suspendiert";
    default:
      return status;
  }
}

export default function TopologyPage() {
  const { nodes, nodeStatus, nodeVMs, isLoading, fetchNodes, fetchNodeStatus, fetchNodeVMs } =
    useNodeStore();

  useEffect(() => {
    fetchNodes();
  }, [fetchNodes]);

  useEffect(() => {
    for (const node of nodes) {
      fetchNodeStatus(node.id);
      fetchNodeVMs(node.id);
    }
  }, [nodes, fetchNodeStatus, fetchNodeVMs]);

  if (isLoading && nodes.length === 0) {
    return (
      <div className="space-y-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Cluster-Topologie</h1>
          <p className="text-sm text-muted-foreground">
            Nodes und VMs im Cluster auf einen Blick.
          </p>
        </div>
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-64 w-full rounded-xl" />
          ))}
        </div>
      </div>
    );
  }

  const totalVMs = Object.values(nodeVMs).flat().length;
  const runningVMs = Object.values(nodeVMs)
    .flat()
    .filter((vm) => vm.status === "running").length;
  const onlineNodes = nodes.filter((n) => n.is_online).length;

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Cluster-Topologie</h1>
        <p className="text-sm text-muted-foreground">
          Nodes und VMs im Cluster auf einen Blick.
        </p>
      </div>

      {/* Summary */}
      <div className="flex flex-wrap gap-3">
        <Badge variant="outline" className="text-sm py-1 px-3">
          <Server className="h-3.5 w-3.5 mr-1.5" />
          {onlineNodes}/{nodes.length} Nodes online
        </Badge>
        <Badge variant="outline" className="text-sm py-1 px-3">
          <Monitor className="h-3.5 w-3.5 mr-1.5" />
          {runningVMs}/{totalVMs} VMs laufend
        </Badge>
      </div>

      {/* Node Grid */}
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {nodes.map((node) => {
          const status = nodeStatus[node.id];
          const vms = nodeVMs[node.id] || [];
          const cpuPercent = status ? Math.round(status.cpu_usage) : 0;
          const memPercent =
            status && status.memory_total > 0
              ? Math.round((status.memory_used / status.memory_total) * 100)
              : 0;
          const diskPercent =
            status && status.disk_total > 0
              ? Math.round((status.disk_used / status.disk_total) * 100)
              : 0;

          return (
            <Card key={node.id} className="overflow-hidden">
              {/* Node Header */}
              <CardHeader className="pb-3">
                <Link
                  href={`/nodes/${node.id}`}
                  className="hover:underline"
                >
                  <CardTitle className="flex items-center gap-2 text-base">
                    <Circle
                      className={`h-2.5 w-2.5 fill-current ${
                        node.is_online ? "text-green-500" : "text-red-500"
                      }`}
                    />
                    <Server className="h-4 w-4" />
                    {node.name}
                    <Badge variant="secondary" className="ml-auto text-xs">
                      {node.type.toUpperCase()}
                    </Badge>
                  </CardTitle>
                </Link>
              </CardHeader>

              <CardContent className="space-y-3 pt-0">
                {/* Resource Bars */}
                {status ? (
                  <div className="space-y-2">
                    <div className="flex items-center gap-2 text-xs">
                      <Cpu className="h-3.5 w-3.5 text-muted-foreground" />
                      <span className="w-8 text-right font-mono">{cpuPercent}%</span>
                      <Progress value={cpuPercent} className="flex-1 h-2" />
                    </div>
                    <div className="flex items-center gap-2 text-xs">
                      <MemoryStick className="h-3.5 w-3.5 text-muted-foreground" />
                      <span className="w-8 text-right font-mono">{memPercent}%</span>
                      <Progress value={memPercent} className="flex-1 h-2" />
                    </div>
                    <div className="flex items-center gap-2 text-xs">
                      <HardDrive className="h-3.5 w-3.5 text-muted-foreground" />
                      <span className="w-8 text-right font-mono">{diskPercent}%</span>
                      <Progress value={diskPercent} className="flex-1 h-2" />
                    </div>
                    <p className="text-[10px] text-muted-foreground">
                      {formatBytes(status.memory_used)} / {formatBytes(status.memory_total)} RAM
                      {" | "}
                      {formatBytes(status.disk_used)} / {formatBytes(status.disk_total)} Disk
                    </p>
                  </div>
                ) : (
                  <div className="text-xs text-muted-foreground">Status wird geladen...</div>
                )}

                {/* VM List */}
                {vms.length > 0 && (
                  <div className="border-t pt-2">
                    <p className="text-xs font-medium text-muted-foreground mb-1.5">
                      {vms.filter((v) => v.status === "running").length}/{vms.length} VMs laufend
                    </p>
                    <div className="space-y-1 max-h-48 overflow-y-auto">
                      {vms
                        .sort((a, b) => {
                          const order = { running: 0, paused: 1, suspended: 2, stopped: 3 };
                          return order[a.status] - order[b.status];
                        })
                        .map((vm) => {
                          const vmMemPct =
                            vm.memory_total > 0
                              ? Math.round((vm.memory_used / vm.memory_total) * 100)
                              : 0;
                          return (
                            <Link
                              key={vm.vmid}
                              href={`/nodes/${node.id}?vm=${vm.vmid}`}
                              className="flex items-center gap-2 text-xs rounded-md px-2 py-1 hover:bg-muted/50 transition-colors"
                            >
                              <Circle
                                className={`h-2 w-2 fill-current shrink-0 ${statusColor(vm.status)}`}
                              />
                              <span className="font-medium truncate flex-1">
                                {vm.name || `VM ${vm.vmid}`}
                              </span>
                              <Badge variant="outline" className="text-[10px] px-1 py-0 h-4">
                                {vm.type}
                              </Badge>
                              <span className="text-muted-foreground w-12 text-right font-mono">
                                {Math.round(vm.cpu_usage)}% C
                              </span>
                              <span className="text-muted-foreground w-12 text-right font-mono">
                                {vmMemPct}% M
                              </span>
                            </Link>
                          );
                        })}
                    </div>
                  </div>
                )}

                {vms.length === 0 && node.type === "pve" && (
                  <div className="border-t pt-2">
                    <p className="text-xs text-muted-foreground">Keine VMs</p>
                  </div>
                )}
              </CardContent>
            </Card>
          );
        })}
      </div>

      {nodes.length === 0 && !isLoading && (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Server className="h-12 w-12 text-muted-foreground mb-3" />
            <p className="text-muted-foreground">Keine Nodes konfiguriert.</p>
            <p className="text-sm text-muted-foreground">
              Fuege Nodes unter Einstellungen hinzu.
            </p>
          </CardContent>
        </Card>
      )}

      {/* Legend */}
      <div className="flex flex-wrap gap-4 text-xs text-muted-foreground">
        <span className="flex items-center gap-1">
          <Circle className="h-2 w-2 fill-current text-green-500" /> {statusLabel("running")}
        </span>
        <span className="flex items-center gap-1">
          <Circle className="h-2 w-2 fill-current text-yellow-500" /> {statusLabel("paused")}
        </span>
        <span className="flex items-center gap-1">
          <Circle className="h-2 w-2 fill-current text-muted-foreground" /> {statusLabel("stopped")}
        </span>
      </div>
    </div>
  );
}
