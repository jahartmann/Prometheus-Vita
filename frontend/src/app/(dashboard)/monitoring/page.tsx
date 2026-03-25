"use client";

import { useEffect, useState, useCallback, useRef } from "react";
import Link from "next/link";
import { Activity, Cpu, MemoryStick, HardDrive, Server, Monitor, Box, RefreshCw, AlertCircle } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Progress } from "@/components/ui/progress";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useNodeStore } from "@/stores/node-store";
import { formatBytes, cn } from "@/lib/utils";
import { useWebSocket } from "@/hooks/use-websocket";
import { LiveTraffic } from "@/components/monitoring/live-traffic";
import { AlertHistory } from "@/components/monitoring/alert-history";

const POLL_INTERVAL = 30_000; // 30 seconds

function timeSince(date: Date): string {
  const seconds = Math.floor((Date.now() - date.getTime()) / 1000);
  if (seconds < 5) return "gerade eben";
  if (seconds < 60) return `vor ${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  return `vor ${minutes}m`;
}

export default function MonitoringPage() {
  const { nodes, nodeStatus, isLoading, error, fetchNodes, fetchNodeStatus } = useNodeStore();
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [displayTime, setDisplayTime] = useState("");
  const pollRef = useRef<ReturnType<typeof setInterval>>(undefined);
  const isMountedRef = useRef(true);

  const { isConnected } = useWebSocket({
    url: "/api/v1/ws",
    enabled: true,
  });

  const refreshAll = useCallback(async () => {
    await fetchNodes();
  }, [fetchNodes]);

  // Initial load
  useEffect(() => {
    isMountedRef.current = true;
    refreshAll();
    return () => { isMountedRef.current = false; };
  }, [refreshAll]);

  // Fetch status for each online node when nodes change
  useEffect(() => {
    if (nodes.length === 0) return;
    const promises = nodes
      .filter((n) => n.is_online)
      .map((n) => fetchNodeStatus(n.id));
    Promise.all(promises).then(() => {
      if (isMountedRef.current) setLastUpdated(new Date());
    });
  }, [nodes, fetchNodeStatus]);

  // Auto-refresh polling
  useEffect(() => {
    pollRef.current = setInterval(() => {
      refreshAll();
    }, POLL_INTERVAL);
    return () => clearInterval(pollRef.current);
  }, [refreshAll]);

  // Update displayed time every 5s
  useEffect(() => {
    if (!lastUpdated) return;
    setDisplayTime(timeSince(lastUpdated));
    const timer = setInterval(() => {
      if (lastUpdated) setDisplayTime(timeSince(lastUpdated));
    }, 5000);
    return () => clearInterval(timer);
  }, [lastUpdated]);

  const onlineNodes = nodes.filter((n) => n.is_online);
  const statuses = Object.values(nodeStatus);

  const avgCpu = statuses.length > 0
    ? statuses.reduce((sum, s) => sum + s.cpu_usage, 0) / statuses.length
    : 0;

  const avgMemory = statuses.length > 0
    ? statuses.reduce((sum, s) => sum + (s.memory_total > 0 ? (s.memory_used / s.memory_total) * 100 : 0), 0) / statuses.length
    : 0;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Monitoring</h1>
          <p className="text-muted-foreground">
            Echtzeit-Status aller Nodes.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2 text-sm text-muted-foreground" role="status" aria-live="polite">
            <span className={cn(
              "h-2 w-2 rounded-full",
              isConnected ? "bg-green-500" : "bg-red-500"
            )} aria-hidden="true" />
            <span>{isConnected ? "Live" : "Polling"}</span>
            {lastUpdated && (
              <span>&middot; Aktualisiert {displayTime}</span>
            )}
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={refreshAll}
            disabled={isLoading}
          >
            <RefreshCw className={`mr-1.5 h-3.5 w-3.5 ${isLoading ? "animate-spin" : ""}`} />
            Aktualisieren
          </Button>
        </div>
      </div>

      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">Übersicht</TabsTrigger>
          <TabsTrigger value="traffic">Live Traffic</TabsTrigger>
          <TabsTrigger value="alerts">Alerts</TabsTrigger>
        </TabsList>

        <TabsContent value="overview">
          {!isLoading && error && (
            <div className="mb-4 rounded-lg border border-destructive/50 bg-destructive/10 p-4 flex items-center gap-3">
              <AlertCircle className="h-5 w-5 text-destructive shrink-0" />
              <div>
                <p className="font-medium text-destructive">Verbindungsfehler</p>
                <p className="text-sm text-muted-foreground">Daten konnten nicht geladen werden. Automatischer Retry in 30s.</p>
              </div>
            </div>
          )}

          {/* Summary Cards */}
          <div className="grid gap-4 md:grid-cols-3 lg:grid-cols-5">
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
            <Card>
              <CardContent className="flex items-center gap-4 p-4">
                <Monitor className="h-8 w-8 text-orange-500" />
                <div>
                  <p className="text-2xl font-bold">
                    {statuses.reduce((s, st) => s + (st.vm_running ?? 0), 0)} / {statuses.reduce((s, st) => s + (st.vm_count ?? 0), 0)}
                  </p>
                  <p className="text-sm text-muted-foreground">VMs aktiv</p>
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="flex items-center gap-4 p-4">
                <Box className="h-8 w-8 text-teal-500" />
                <div>
                  <p className="text-2xl font-bold">
                    {statuses.reduce((s, st) => s + (st.ct_running ?? 0), 0)} / {statuses.reduce((s, st) => s + (st.ct_count ?? 0), 0)}
                  </p>
                  <p className="text-sm text-muted-foreground">CTs aktiv</p>
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Node Status Grid */}
          {isLoading ? (
            <div className="mt-4 grid gap-4 md:grid-cols-2 lg:grid-cols-3" aria-busy="true" aria-label="Nodes werden geladen">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-48 w-full" />
              ))}
            </div>
          ) : nodes.length === 0 ? (
            <Card className="mt-4">
              <CardContent className="flex flex-col items-center justify-center py-12">
                <Server className="mb-3 h-10 w-10 text-muted-foreground" />
                <p className="text-muted-foreground">Noch keine Nodes konfiguriert.</p>
              </CardContent>
            </Card>
          ) : (
            <div className="mt-4 grid gap-4 md:grid-cols-2 lg:grid-cols-3">
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
                              <p className="text-xs text-muted-foreground">
                                {status.cpu_cores} Cores &middot; {status.cpu_model}
                              </p>
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

                            {status.swap_total > 0 && (
                              <div className="space-y-1">
                                <div className="flex items-center justify-between text-sm">
                                  <span>Swap</span>
                                  <span>
                                    {formatBytes(status.swap_used)} / {formatBytes(status.swap_total)}
                                  </span>
                                </div>
                                <Progress value={(status.swap_used / status.swap_total) * 100} className="h-2" />
                              </div>
                            )}

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

                            <div className="flex items-center justify-between text-xs text-muted-foreground">
                              <span>Load: {status.load_average?.map((l) => l.toFixed(2)).join(", ")}</span>
                              <span>VMs: {status.vm_running}/{status.vm_count} &middot; CTs: {status.ct_running}/{status.ct_count}</span>
                            </div>

                            <div className="flex items-center justify-between text-xs text-muted-foreground">
                              <span>Uptime: {Math.floor(status.uptime / 86400)}d {Math.floor((status.uptime % 86400) / 3600)}h</span>
                              <span>PVE {status.pve_version}</span>
                            </div>
                          </div>
                        ) : (
                          <p className="text-sm text-muted-foreground">
                            Keine Statusdaten verfügbar.
                          </p>
                        )}
                      </CardContent>
                    </Card>
                  </Link>
                );
              })}
            </div>
          )}
        </TabsContent>

        <TabsContent value="traffic">
          <LiveTraffic />
        </TabsContent>

        <TabsContent value="alerts">
          <AlertHistory />
        </TabsContent>
      </Tabs>
    </div>
  );
}
