"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { ArrowLeft, Cpu, MemoryStick, HardDrive, Clock, Monitor } from "lucide-react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip as RechartsTooltip,
} from "recharts";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useNodeStore } from "@/stores/node-store";
import { useNodeMetrics } from "@/hooks/use-node-metrics";
import { useBackupStore } from "@/stores/backup-store";
import { VmList } from "./vm-list";
import { BackupList } from "@/components/backup/backup-list";
import { MetricsCharts } from "@/components/monitoring/metrics-charts";
import { NetworkInterfaces } from "@/components/nodes/network-interfaces";
import { StorageOverview } from "@/components/nodes/storage-overview";
import { DiskList } from "@/components/nodes/disk-list";
import { PBSOverview } from "@/components/nodes/pbs-overview";
import { TagAssignDialog } from "@/components/tags/tag-assign-dialog";
import { metricsApi, networkApi, diskApi, tagApi, toArray } from "@/lib/api";
import type { Node, MetricsRecord, NetworkInterface, DiskInfo, Tag } from "@/types/api";
import {
  formatBytes,
  formatUptime,
  formatPercentage,
  getUsageBgColor,
} from "@/lib/utils";

function GaugeBar({
  label,
  value,
  detail,
}: {
  label: string;
  value: number;
  detail: string;
}) {
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium">{label}</span>
        <span className="text-sm text-muted-foreground">{detail}</span>
      </div>
      <div className="h-2 w-full rounded-full bg-secondary">
        <div
          className={`h-2 rounded-full transition-all ${getUsageBgColor(value)}`}
          style={{ width: `${Math.min(value, 100)}%` }}
        />
      </div>
      <p className="text-right text-xs font-medium">{formatPercentage(value)}</p>
    </div>
  );
}

interface NodeDetailProps {
  node: Node;
}

export function NodeDetail({ node }: NodeDetailProps) {
  const { nodeStatus, nodeVMs, fetchNodeVMs } = useNodeStore();
  const status = nodeStatus[node.id];
  const vms = nodeVMs[node.id] || [];
  const { metrics } = useNodeMetrics(node.id, node.is_online);
  const { fetchBackups, fetchSchedules } = useBackupStore();
  const [metricsHistory, setMetricsHistory] = useState<MetricsRecord[]>([]);
  const [networkIfaces, setNetworkIfaces] = useState<NetworkInterface[]>([]);
  const [disks, setDisks] = useState<DiskInfo[]>([]);
  const [nodeTags, setNodeTags] = useState<Tag[]>([]);
  const [tagDialogOpen, setTagDialogOpen] = useState(false);

  useEffect(() => {
    fetchBackups(node.id);
    fetchSchedules(node.id);

    metricsApi
      .getHistory(node.id, new Date(Date.now() - 3600000).toISOString(), new Date().toISOString())
      .then((res) => setMetricsHistory(toArray(res.data)))
      .catch(() => {});

    networkApi
      .getInterfaces(node.id)
      .then((res) => setNetworkIfaces(toArray(res.data)))
      .catch(() => {});

    diskApi
      .getDisks(node.id)
      .then((res) => setDisks(toArray(res.data)))
      .catch(() => {});

    tagApi
      .getNodeTags(node.id)
      .then((res) => setNodeTags(toArray(res.data)))
      .catch(() => {});
  }, [node.id, fetchBackups, fetchSchedules]);

  const cpuUsage = status ? status.cpu_usage : 0;
  const memUsage = status
    ? (status.memory_used / status.memory_total) * 100
    : 0;
  const diskUsage = status
    ? (status.disk_used / status.disk_total) * 100
    : 0;

  const chartData = useMemo(
    () =>
      metrics.map((m) => ({
        time: new Date(m.timestamp).toLocaleTimeString("de-DE", {
          hour: "2-digit",
          minute: "2-digit",
          second: "2-digit",
        }),
        cpu: m.cpu_usage,
        memory: m.memory_usage,
      })),
    [metrics]
  );

  const formatTooltipValue = useCallback(
    (value: number) => `${(value ?? 0).toFixed(1)}%`,
    []
  );

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link href="/">
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-2xl font-bold">{node.name}</h1>
            <Badge variant={node.is_online ? "success" : "destructive"}>
              {node.is_online ? "Online" : "Offline"}
            </Badge>
            {nodeTags.map((tag) => (
              <Badge key={tag.id} style={{ backgroundColor: tag.color, color: "white" }}>
                {tag.name}
              </Badge>
            ))}
            <Button variant="ghost" size="sm" onClick={() => setTagDialogOpen(true)}>
              Tags
            </Button>
          </div>
          <p className="text-sm text-muted-foreground">
            {node.hostname}:{node.port}
          </p>
        </div>
      </div>

      {status && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
          <Card>
            <CardContent className="flex items-center gap-3 p-4">
              <Cpu className="h-5 w-5 text-primary" />
              <div>
                <p className="text-xs text-muted-foreground">CPU</p>
                <p className="text-lg font-bold">
                  {formatPercentage(cpuUsage)}
                </p>
                <p className="text-xs text-muted-foreground">
                  {status.cpu_cores} Cores
                </p>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="flex items-center gap-3 p-4">
              <MemoryStick className="h-5 w-5 text-primary" />
              <div>
                <p className="text-xs text-muted-foreground">RAM</p>
                <p className="text-lg font-bold">
                  {formatPercentage(memUsage)}
                </p>
                <p className="text-xs text-muted-foreground">
                  {formatBytes(status.memory_used)} / {formatBytes(status.memory_total)}
                </p>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="flex items-center gap-3 p-4">
              <HardDrive className="h-5 w-5 text-primary" />
              <div>
                <p className="text-xs text-muted-foreground">Disk</p>
                <p className="text-lg font-bold">
                  {formatPercentage(diskUsage)}
                </p>
                <p className="text-xs text-muted-foreground">
                  {formatBytes(status.disk_used)} / {formatBytes(status.disk_total)}
                </p>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="flex items-center gap-3 p-4">
              <Clock className="h-5 w-5 text-primary" />
              <div>
                <p className="text-xs text-muted-foreground">Uptime</p>
                <p className="text-lg font-bold">
                  {formatUptime(status.uptime)}
                </p>
                <p className="text-xs text-muted-foreground">
                  Load: {(status.load_average || []).map((l) => (l ?? 0).toFixed(2)).join(" ")}
                </p>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="flex items-center gap-3 p-4">
              <Monitor className="h-5 w-5 text-primary" />
              <div>
                <p className="text-xs text-muted-foreground">VMs</p>
                <p className="text-lg font-bold">
                  {status.vm_running}/{status.vm_count}
                </p>
                <p className="text-xs text-muted-foreground">
                  {status.ct_running}/{status.ct_count} CTs
                </p>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">Uebersicht</TabsTrigger>
          <TabsTrigger value="vms">VMs & Container</TabsTrigger>
          <TabsTrigger value="backups">Backups</TabsTrigger>
          <TabsTrigger value="monitoring">Monitoring</TabsTrigger>
          <TabsTrigger value="network">Netzwerk</TabsTrigger>
          <TabsTrigger value="storage">Storage</TabsTrigger>
          {node.type === "pbs" && <TabsTrigger value="pbs">PBS</TabsTrigger>}
        </TabsList>

        <TabsContent value="overview" className="space-y-4">
          {status && (
            <div className="grid gap-4 lg:grid-cols-2">
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Ressourcen</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <GaugeBar
                    label="CPU"
                    value={cpuUsage}
                    detail={`${status.cpu_model} (${status.cpu_cores} Cores)`}
                  />
                  <GaugeBar
                    label="RAM"
                    value={memUsage}
                    detail={`${formatBytes(status.memory_used)} / ${formatBytes(status.memory_total)}`}
                  />
                  <GaugeBar
                    label="Swap"
                    value={
                      status.swap_total > 0
                        ? (status.swap_used / status.swap_total) * 100
                        : 0
                    }
                    detail={`${formatBytes(status.swap_used)} / ${formatBytes(status.swap_total)}`}
                  />
                  <GaugeBar
                    label="Disk"
                    value={diskUsage}
                    detail={`${formatBytes(status.disk_used)} / ${formatBytes(status.disk_total)}`}
                  />
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="text-base">CPU & RAM Verlauf</CardTitle>
                </CardHeader>
                <CardContent>
                  {chartData.length > 0 ? (
                    <ResponsiveContainer width="100%" height={250}>
                      <LineChart data={chartData}>
                        <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                        <XAxis
                          dataKey="time"
                          className="text-xs"
                          tick={{ fontSize: 10 }}
                        />
                        <YAxis
                          domain={[0, 100]}
                          className="text-xs"
                          tick={{ fontSize: 10 }}
                          tickFormatter={(v) => `${v}%`}
                        />
                        <RechartsTooltip
                          formatter={formatTooltipValue}
                          contentStyle={{
                            backgroundColor: "hsl(var(--card))",
                            border: "1px solid hsl(var(--border))",
                            borderRadius: "0.5rem",
                          }}
                        />
                        <Line
                          type="monotone"
                          dataKey="cpu"
                          stroke="hsl(25, 95%, 53%)"
                          strokeWidth={2}
                          dot={false}
                          name="CPU"
                        />
                        <Line
                          type="monotone"
                          dataKey="memory"
                          stroke="hsl(45, 93%, 47%)"
                          strokeWidth={2}
                          dot={false}
                          name="RAM"
                        />
                      </LineChart>
                    </ResponsiveContainer>
                  ) : (
                    <div className="flex h-[250px] items-center justify-center text-sm text-muted-foreground">
                      Warte auf Live-Daten...
                    </div>
                  )}
                </CardContent>
              </Card>
            </div>
          )}

          {status && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base">System-Info</CardTitle>
              </CardHeader>
              <CardContent>
                <dl className="grid grid-cols-2 gap-4 text-sm md:grid-cols-3">
                  <div>
                    <dt className="text-muted-foreground">Kernel</dt>
                    <dd className="font-medium">{status.kernel_version}</dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">PVE Version</dt>
                    <dd className="font-medium">{status.pve_version}</dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">CPU Modell</dt>
                    <dd className="font-medium">{status.cpu_model}</dd>
                  </div>
                </dl>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="vms">
          <VmList vms={vms} nodeId={node.id} onRefresh={() => fetchNodeVMs(node.id)} />
        </TabsContent>

        <TabsContent value="backups">
          <BackupList nodeId={node.id} />
        </TabsContent>

        <TabsContent value="monitoring">
          <MetricsCharts metrics={metricsHistory} />
        </TabsContent>

        <TabsContent value="network">
          <NetworkInterfaces
            nodeId={node.id}
            interfaces={networkIfaces}
            onRefresh={() => {
              networkApi
                .getInterfaces(node.id)
                .then((res) => setNetworkIfaces(toArray(res.data)));
            }}
          />
        </TabsContent>

        <TabsContent value="storage">
          <StorageOverview nodeId={node.id} status={status} />
          <DiskList disks={disks} />
        </TabsContent>

        {node.type === "pbs" && (
          <TabsContent value="pbs">
            <PBSOverview nodeId={node.id} />
          </TabsContent>
        )}
      </Tabs>

      <TagAssignDialog nodeId={node.id} open={tagDialogOpen} onOpenChange={setTagDialogOpen} />
    </div>
  );
}
