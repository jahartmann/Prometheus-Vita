"use client";

import { useEffect } from "react";
import Link from "next/link";
import { Activity, Cpu, MemoryStick, HardDrive, Server } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Progress } from "@/components/ui/progress";
import { useNodeStore } from "@/stores/node-store";
import { formatBytes } from "@/lib/utils";

export default function MonitoringPage() {
  const { nodes, nodeStatus, isLoading, fetchNodes, fetchNodeStatus } = useNodeStore();

  useEffect(() => {
    fetchNodes();
  }, [fetchNodes]);

  useEffect(() => {
    nodes.forEach((node) => {
      if (node.is_online) {
        fetchNodeStatus(node.id);
      }
    });
  }, [nodes, fetchNodeStatus]);

  const onlineNodes = nodes.filter((n) => n.is_online);
  const statuses = Object.values(nodeStatus);

  const avgCpu = statuses.length > 0
    ? statuses.reduce((sum, s) => sum + s.cpu_usage, 0) / statuses.length
    : 0;

  const avgMemory = statuses.length > 0
    ? statuses.reduce((sum, s) => sum + (s.memory_used / s.memory_total) * 100, 0) / statuses.length
    : 0;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Monitoring</h1>
        <p className="text-muted-foreground">
          Echtzeit-Status aller Nodes.
        </p>
      </div>

      {/* Summary Cards */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardContent className="flex items-center gap-4 p-4">
            <Activity className="h-8 w-8 text-green-500" />
            <div>
              <p className="text-2xl font-bold">
                {onlineNodes.length} / {nodes.length}
              </p>
              <p className="text-sm text-muted-foreground">Nodes online</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-4 p-4">
            <Cpu className="h-8 w-8 text-blue-500" />
            <div>
              <p className="text-2xl font-bold">{avgCpu.toFixed(1)}%</p>
              <p className="text-sm text-muted-foreground">CPU (Durchschnitt)</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-4 p-4">
            <MemoryStick className="h-8 w-8 text-purple-500" />
            <div>
              <p className="text-2xl font-bold">{avgMemory.toFixed(1)}%</p>
              <p className="text-sm text-muted-foreground">RAM (Durchschnitt)</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Node Status Grid */}
      {isLoading ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-48 w-full" />
          ))}
        </div>
      ) : nodes.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Server className="mb-3 h-10 w-10 text-muted-foreground" />
            <p className="text-muted-foreground">Noch keine Nodes konfiguriert.</p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {nodes.map((node) => {
            const status = nodeStatus[node.id];
            const cpuPercent = status?.cpu_usage ?? 0;
            const memPercent = status ? (status.memory_used / status.memory_total) * 100 : 0;
            const diskPercent = status ? (status.disk_used / status.disk_total) * 100 : 0;

            return (
              <Link key={node.id} href={`/nodes/${node.id}`}>
                <Card className="transition-colors hover:bg-muted/50">
                  <CardContent className="space-y-4 p-4">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <Server className="h-4 w-4 text-muted-foreground" />
                        <p className="font-medium">{node.name}</p>
                      </div>
                      <Badge variant={node.is_online ? "success" : "destructive"}>
                        {node.is_online ? "Online" : "Offline"}
                      </Badge>
                    </div>

                    {node.is_online && status ? (
                      <div className="space-y-3">
                        <div className="space-y-1">
                          <div className="flex items-center justify-between text-sm">
                            <span className="flex items-center gap-1">
                              <Cpu className="h-3 w-3" /> CPU
                            </span>
                            <span>{cpuPercent.toFixed(1)}%</span>
                          </div>
                          <Progress value={cpuPercent} className="h-2" />
                        </div>

                        <div className="space-y-1">
                          <div className="flex items-center justify-between text-sm">
                            <span className="flex items-center gap-1">
                              <MemoryStick className="h-3 w-3" /> RAM
                            </span>
                            <span>
                              {formatBytes(status.memory_used)} / {formatBytes(status.memory_total)}
                            </span>
                          </div>
                          <Progress value={memPercent} className="h-2" />
                        </div>

                        <div className="space-y-1">
                          <div className="flex items-center justify-between text-sm">
                            <span className="flex items-center gap-1">
                              <HardDrive className="h-3 w-3" /> Disk
                            </span>
                            <span>
                              {formatBytes(status.disk_used)} / {formatBytes(status.disk_total)}
                            </span>
                          </div>
                          <Progress value={diskPercent} className="h-2" />
                        </div>

                        <p className="text-xs text-muted-foreground">
                          Uptime: {Math.floor(status.uptime / 86400)}d {Math.floor((status.uptime % 86400) / 3600)}h
                        </p>
                      </div>
                    ) : (
                      <p className="text-sm text-muted-foreground">
                        Keine Statusdaten verfuegbar.
                      </p>
                    )}
                  </CardContent>
                </Card>
              </Link>
            );
          })}
        </div>
      )}
    </div>
  );
}
