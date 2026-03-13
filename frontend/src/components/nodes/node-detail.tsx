"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { ArrowLeft, AlertTriangle, Cpu, MemoryStick, HardDrive, Clock, Monitor, Disc, Network, FileText, ExternalLink, RefreshCw } from "lucide-react";
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
import { NetworkTraffic } from "@/components/monitoring/network-traffic";
import { ErrorBoundary } from "@/components/error-boundary";
import { NetworkInterfaces } from "@/components/nodes/network-interfaces";
import { NodePorts } from "@/components/nodes/node-ports";
import { StorageOverview } from "@/components/nodes/storage-overview";
import { DiskList } from "@/components/nodes/disk-list";
import { PBSOverview } from "@/components/nodes/pbs-overview";
import { TagAssignDialog } from "@/components/tags/tag-assign-dialog";
import { metricsApi, networkApi, diskApi, tagApi, isoApi, nodeApi, logApi, toArray } from "@/lib/api";
import type { Node, MetricsRecord, NetworkInterface, DiskInfo, Tag, StorageContent } from "@/types/api";
import {
  formatBytes,
  formatUptime,
  formatPercentage,
  formatBandwidth,
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
  const { nodeStatus, nodeVMs, nodeErrors, fetchNodeVMs } = useNodeStore();
  const status = nodeStatus[node.id];
  const vms = nodeVMs[node.id] || [];
  const { metrics, latestMetrics } = useNodeMetrics(node.id, node.is_online);
  const { fetchBackups, fetchSchedules } = useBackupStore();
  const [metricsHistory, setMetricsHistory] = useState<MetricsRecord[]>([]);
  const [networkIfaces, setNetworkIfaces] = useState<NetworkInterface[]>([]);
  const [disks, setDisks] = useState<DiskInfo[]>([]);
  const [nodeTags, setNodeTags] = useState<Tag[]>([]);
  const [tagDialogOpen, setTagDialogOpen] = useState(false);
  const [isos, setIsos] = useState<StorageContent[]>([]);
  const [templates, setTemplates] = useState<StorageContent[]>([]);
  const [allNodes, setAllNodes] = useState<Node[]>([]);
  const [syncDialogOpen, setSyncDialogOpen] = useState(false);
  const [syncType, setSyncType] = useState<"iso" | "template">("iso");
  const [syncSourceNode, setSyncSourceNode] = useState("");
  const [syncSourceContent, setSyncSourceContent] = useState<StorageContent[]>([]);
  const [syncSelectedVolid, setSyncSelectedVolid] = useState("");
  const [syncTargetStorage, setSyncTargetStorage] = useState("local");
  const [syncLoading, setSyncLoading] = useState(false);
  const [logContent, setLogContent] = useState("");
  const [logLoading, setLogLoading] = useState(false);

  useEffect(() => {
    fetchBackups(node.id);
    fetchSchedules(node.id);
    fetchNodeVMs(node.id);

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

    isoApi
      .listISOs(node.id)
      .then((res) => setIsos(toArray(res.data)))
      .catch(() => {});

    isoApi
      .listTemplates(node.id)
      .then((res) => setTemplates(toArray(res.data)))
      .catch(() => {});

    nodeApi
      .list()
      .then((res) => setAllNodes(toArray(res.data)))
      .catch(() => {});

    logApi
      .getLogs(node.id, "syslog", 100)
      .then((res) => {
        const data = res.data;
        setLogContent(typeof data?.lines === "string" ? data.lines : "");
      })
      .catch(() => {});
  }, [node.id, fetchBackups, fetchSchedules, fetchNodeVMs]);

  const cpuUsage = status?.cpu_usage ?? 0;
  const memUsage = status && status.memory_total > 0
    ? (status.memory_used / status.memory_total) * 100
    : 0;
  const diskUsage = status && status.disk_total > 0
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

      {nodeErrors[node.id] && (
        <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 p-4 text-sm text-amber-600 dark:text-amber-400">
          <div className="flex items-center gap-2">
            <AlertTriangle className="h-4 w-4" />
            <span>{nodeErrors[node.id]} — Einige Daten sind moeglicherweise nicht verfuegbar.</span>
          </div>
        </div>
      )}

      {status && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
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
              <Network className="h-5 w-5 text-primary" />
              <div>
                <p className="text-xs text-muted-foreground">Netzwerk</p>
                <p className="text-sm font-bold text-blue-500">
                  &darr; {formatBandwidth(latestMetrics?.network_in ?? 0)}
                </p>
                <p className="text-sm font-bold text-green-500">
                  &uarr; {formatBandwidth(latestMetrics?.network_out ?? 0)}
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
                  {status.vm_running ?? 0}/{status.vm_count ?? 0}
                </p>
                <p className="text-xs text-muted-foreground">
                  {status.ct_running ?? 0}/{status.ct_count ?? 0} CTs
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
          <TabsTrigger value="monitoring">Monitoring</TabsTrigger>
          <TabsTrigger value="network">Netzwerk</TabsTrigger>
          <TabsTrigger value="storage">Storage</TabsTrigger>
          <TabsTrigger value="backups">Backups</TabsTrigger>
          <TabsTrigger value="iso-templates">ISOs & Vorlagen</TabsTrigger>
          <TabsTrigger value="logs">Logs</TabsTrigger>
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

        <TabsContent value="monitoring" className="space-y-6">
          <ErrorBoundary>
            <MetricsCharts metrics={metricsHistory} />
          </ErrorBoundary>
          <ErrorBoundary>
            <NetworkTraffic nodeId={node.id} />
          </ErrorBoundary>
        </TabsContent>

        <TabsContent value="network" className="space-y-6">
          <NetworkInterfaces
            nodeId={node.id}
            interfaces={networkIfaces}
            onRefresh={() => {
              networkApi
                .getInterfaces(node.id)
                .then((res) => setNetworkIfaces(toArray(res.data)));
            }}
          />
          <NodePorts nodeId={node.id} />
        </TabsContent>

        <TabsContent value="storage">
          <StorageOverview nodeId={node.id} status={status} />
          <DiskList disks={disks} />
        </TabsContent>

        <TabsContent value="backups">
          <BackupList nodeId={node.id} />
        </TabsContent>

        <TabsContent value="iso-templates" className="space-y-6">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold">ISO Images</h2>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setSyncType("iso");
                setSyncDialogOpen(true);
                setSyncSourceNode("");
                setSyncSourceContent([]);
                setSyncSelectedVolid("");
              }}
            >
              <Disc className="mr-2 h-4 w-4" />
              Von Node synchronisieren
            </Button>
          </div>
          <Card>
            <CardContent className="p-0">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b">
                    <th className="p-3 text-left font-medium">Name</th>
                    <th className="p-3 text-left font-medium">Format</th>
                    <th className="p-3 text-right font-medium">Groesse</th>
                    <th className="p-3 text-right font-medium">Datum</th>
                  </tr>
                </thead>
                <tbody>
                  {isos.length === 0 ? (
                    <tr>
                      <td colSpan={4} className="p-6 text-center text-muted-foreground">
                        Keine ISO Images gefunden
                      </td>
                    </tr>
                  ) : (
                    isos.map((iso) => (
                      <tr key={iso.volid} className="border-b last:border-0">
                        <td className="p-3 font-mono text-xs">{iso.volid}</td>
                        <td className="p-3">{iso.format}</td>
                        <td className="p-3 text-right">{formatBytes(iso.size)}</td>
                        <td className="p-3 text-right">
                          {new Date(iso.ctime * 1000).toLocaleDateString("de-DE")}
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </CardContent>
          </Card>

          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold">Container Templates</h2>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setSyncType("template");
                setSyncDialogOpen(true);
                setSyncSourceNode("");
                setSyncSourceContent([]);
                setSyncSelectedVolid("");
              }}
            >
              <Disc className="mr-2 h-4 w-4" />
              Von Node synchronisieren
            </Button>
          </div>
          <Card>
            <CardContent className="p-0">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b">
                    <th className="p-3 text-left font-medium">Name</th>
                    <th className="p-3 text-left font-medium">Format</th>
                    <th className="p-3 text-right font-medium">Groesse</th>
                    <th className="p-3 text-right font-medium">Datum</th>
                  </tr>
                </thead>
                <tbody>
                  {templates.length === 0 ? (
                    <tr>
                      <td colSpan={4} className="p-6 text-center text-muted-foreground">
                        Keine Container Templates gefunden
                      </td>
                    </tr>
                  ) : (
                    templates.map((tpl) => (
                      <tr key={tpl.volid} className="border-b last:border-0">
                        <td className="p-3 font-mono text-xs">{tpl.volid}</td>
                        <td className="p-3">{tpl.format}</td>
                        <td className="p-3 text-right">{formatBytes(tpl.size)}</td>
                        <td className="p-3 text-right">
                          {new Date(tpl.ctime * 1000).toLocaleDateString("de-DE")}
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="logs" className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold flex items-center gap-2">
              <FileText className="h-5 w-5" />
              System-Logs
            </h2>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  setLogLoading(true);
                  logApi
                    .getLogs(node.id, "syslog", 100)
                    .then((res) => {
                      const data = res.data;
                      setLogContent(typeof data?.lines === "string" ? data.lines : "");
                    })
                    .catch(() => {})
                    .finally(() => setLogLoading(false));
                }}
              >
                <RefreshCw className={`mr-2 h-3 w-3 ${logLoading ? "animate-spin" : ""}`} />
                Aktualisieren
              </Button>
              <Button variant="outline" size="sm" asChild>
                <Link href={`/nodes/${node.id}/logs`}>
                  <ExternalLink className="mr-2 h-3 w-3" />
                  Erweiterte Ansicht
                </Link>
              </Button>
            </div>
          </div>
          <Card>
            <CardContent className="p-0">
              <pre className="max-h-[500px] overflow-auto bg-zinc-950 p-4 font-mono text-xs leading-relaxed text-zinc-300">
                {logContent ? (
                  logContent.split("\n").map((line, idx) => (
                    <div key={idx} className="flex">
                      <span className="mr-4 inline-block w-10 select-none text-right text-zinc-600">
                        {idx + 1}
                      </span>
                      <span className="flex-1 whitespace-pre-wrap break-all">{line}</span>
                    </div>
                  ))
                ) : (
                  <div className="py-8 text-center text-zinc-500">
                    Keine Logs verfuegbar.
                  </div>
                )}
              </pre>
            </CardContent>
          </Card>
        </TabsContent>

        {node.type === "pbs" && (
          <TabsContent value="pbs">
            <PBSOverview nodeId={node.id} />
          </TabsContent>
        )}
      </Tabs>

      {syncDialogOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <Card className="w-full max-w-lg">
            <CardHeader>
              <CardTitle>
                {syncType === "iso" ? "ISO" : "Template"} von anderem Node synchronisieren
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <label className="text-sm font-medium">Quell-Node</label>
                <select
                  className="mt-1 w-full rounded-md border bg-background px-3 py-2 text-sm"
                  value={syncSourceNode}
                  onChange={(e) => {
                    const sourceId = e.target.value;
                    setSyncSourceNode(sourceId);
                    setSyncSelectedVolid("");
                    if (sourceId) {
                      const fetchFn = syncType === "iso" ? isoApi.listISOs : isoApi.listTemplates;
                      fetchFn(sourceId)
                        .then((res) => setSyncSourceContent(toArray(res.data)))
                        .catch(() => setSyncSourceContent([]));
                    } else {
                      setSyncSourceContent([]);
                    }
                  }}
                >
                  <option value="">Node auswaehlen...</option>
                  {allNodes
                    .filter((n) => n.id !== node.id)
                    .map((n) => (
                      <option key={n.id} value={n.id}>
                        {n.name} ({n.hostname})
                      </option>
                    ))}
                </select>
              </div>

              {syncSourceContent.length > 0 && (
                <div>
                  <label className="text-sm font-medium">Inhalt</label>
                  <select
                    className="mt-1 w-full rounded-md border bg-background px-3 py-2 text-sm"
                    value={syncSelectedVolid}
                    onChange={(e) => setSyncSelectedVolid(e.target.value)}
                  >
                    <option value="">Datei auswaehlen...</option>
                    {syncSourceContent.map((c) => (
                      <option key={c.volid} value={c.volid}>
                        {c.volid} ({formatBytes(c.size)})
                      </option>
                    ))}
                  </select>
                </div>
              )}

              <div>
                <label className="text-sm font-medium">Ziel-Storage</label>
                <input
                  className="mt-1 w-full rounded-md border bg-background px-3 py-2 text-sm"
                  value={syncTargetStorage}
                  onChange={(e) => setSyncTargetStorage(e.target.value)}
                  placeholder="local"
                />
              </div>

              <div className="flex justify-end gap-2">
                <Button variant="outline" onClick={() => setSyncDialogOpen(false)}>
                  Abbrechen
                </Button>
                <Button
                  disabled={!syncSourceNode || !syncSelectedVolid || syncLoading}
                  onClick={async () => {
                    setSyncLoading(true);
                    try {
                      await isoApi.syncContent(node.id, {
                        source_node_id: syncSourceNode,
                        volid: syncSelectedVolid,
                        target_storage: syncTargetStorage || "local",
                      });
                      setSyncDialogOpen(false);
                      // Refresh lists
                      isoApi.listISOs(node.id).then((res) => setIsos(toArray(res.data))).catch(() => {});
                      isoApi.listTemplates(node.id).then((res) => setTemplates(toArray(res.data))).catch(() => {});
                    } catch {
                      // Error handling handled by interceptor
                    } finally {
                      setSyncLoading(false);
                    }
                  }}
                >
                  {syncLoading ? "Synchronisiere..." : "Synchronisieren"}
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      <TagAssignDialog nodeId={node.id} open={tagDialogOpen} onOpenChange={setTagDialogOpen} />
    </div>
  );
}
