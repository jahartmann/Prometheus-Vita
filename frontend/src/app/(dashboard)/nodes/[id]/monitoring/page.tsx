"use client";

import { useEffect, useState, useMemo } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import {
  ArrowLeft,
  Activity,
  Cpu,
  MemoryStick,
  Network,
  Server,
  Wifi,
  WifiOff,
  RefreshCw,
} from "lucide-react";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  Legend,
} from "recharts";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useNodeStore } from "@/stores/node-store";
import { metricsApi, anomalyApi, predictionApi, nodeApi, toArray } from "@/lib/api";
import { formatBandwidth, formatBytes, formatPercentage } from "@/lib/utils";
import { ErrorBoundary } from "@/components/error-boundary";
import { KPICards } from "@/components/monitoring/kpi-cards";
import { CPUDetailChart } from "@/components/monitoring/cpu-detail-chart";
import { RRDBandwidth } from "@/components/monitoring/rrd-bandwidth";
import { VMMetricsTable } from "@/components/monitoring/vm-metrics-table";
import { VMNetworkTraffic } from "@/components/monitoring/vm-network-traffic";
import { AnomalyOverlay } from "@/components/monitoring/anomaly-overlay";
import { useNodeMetrics } from "@/hooks/use-node-metrics";
import type { MetricsRecord, MetricsSummary, RRDDataPoint, VM, VMMetricsRecord, AnomalyRecord, MaintenancePrediction } from "@/types/api";

const periods = [
  { label: "1h", value: "1h", hours: 1, rrdTimeframe: "hour" },
  { label: "6h", value: "6h", hours: 6, rrdTimeframe: "hour" },
  { label: "24h", value: "24h", hours: 24, rrdTimeframe: "day" },
  { label: "7d", value: "7d", hours: 168, rrdTimeframe: "week" },
  { label: "30d", value: "30d", hours: 720, rrdTimeframe: "month" },
];

function PeriodSelector({
  period,
  onPeriodChange,
}: {
  period: string;
  onPeriodChange: (p: string) => void;
}) {
  return (
    <div className="flex gap-1">
      {periods.map((p) => (
        <Button
          key={p.value}
          variant={period === p.value ? "default" : "outline"}
          size="sm"
          onClick={() => onPeriodChange(p.value)}
        >
          {p.label}
        </Button>
      ))}
    </div>
  );
}

export default function NodeMonitoringPage() {
  const params = useParams<{ id: string }>();
  const nodeId = params.id;
  const { nodes, fetchNodes } = useNodeStore();
  const [metrics, setMetrics] = useState<MetricsRecord[]>([]);
  const [summary, setSummary] = useState<MetricsSummary | null>(null);
  const [rrdData, setRrdData] = useState<RRDDataPoint[]>([]);
  const [vms, setVMs] = useState<VM[]>([]);
  const [vmHistory, setVMHistory] = useState<Record<number, Array<{ cpu: number; mem: number }>>>({});
  const [period, setPeriod] = useState("24h");
  const [anomalies, setAnomalies] = useState<AnomalyRecord[]>([]);
  const [predictions, setPredictions] = useState<MaintenancePrediction[]>([]);
  const [activeTab, setActiveTab] = useState("overview");
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  // Live WebSocket metrics
  const { latestMetrics, metrics: wsMetrics } = useNodeMetrics(nodeId, true);

  useEffect(() => {
    if (nodes.length === 0) fetchNodes();
  }, [nodes.length, fetchNodes]);

  // Fetch VMs
  useEffect(() => {
    if (!nodeId) return;
    const controller = new AbortController();
    nodeApi
      .getVMs(nodeId)
      .then((res) => setVMs(toArray<VM>(res.data)))
      .catch((e) => { if (e?.name !== 'CanceledError') setVMs([]); });
    return () => controller.abort();
  }, [nodeId]);

  // Fetch metrics + summary + RRD based on period
  useEffect(() => {
    if (!nodeId) return;
    const controller = new AbortController();
    const signal = controller.signal;
    const periodConfig = periods.find((p) => p.value === period) || periods[2];
    const since = new Date();
    since.setHours(since.getHours() - periodConfig.hours);

    const ignoreCanceled = (e: unknown) => { if ((e as { name?: string })?.name !== 'CanceledError') throw e; };

    Promise.all([
      metricsApi
        .getHistory(nodeId, since.toISOString(), new Date().toISOString(), { signal })
        .then((res) => setMetrics(toArray<MetricsRecord>(res.data)))
        .catch((e) => { if ((e as { name?: string })?.name !== 'CanceledError') setMetrics([]); }),
      metricsApi
        .getSummary(nodeId, period, { signal })
        .then((res) => setSummary(res.data ?? null))
        .catch((e) => { if ((e as { name?: string })?.name !== 'CanceledError') setSummary(null); }),
      metricsApi
        .getNodeRRD(nodeId, periodConfig.rrdTimeframe, { signal })
        .then((res) => setRrdData(toArray<RRDDataPoint>(res.data)))
        .catch((e) => { if ((e as { name?: string })?.name !== 'CanceledError') setRrdData([]); }),
    ]).then(() => { if (!signal.aborted) setLastUpdated(new Date()); });

    // Fetch anomalies and predictions
    anomalyApi.listByNode(nodeId, { signal }).then((d) => setAnomalies(d as AnomalyRecord[])).catch((e) => { if ((e as { name?: string })?.name !== 'CanceledError') setAnomalies([]); });
    predictionApi.listByNode(nodeId, { signal }).then((d) => setPredictions(d as MaintenancePrediction[])).catch((e) => { if ((e as { name?: string })?.name !== 'CanceledError') setPredictions([]); });

    return () => controller.abort();
  }, [nodeId, period]);

  // Fetch VM metrics history for sparklines
  useEffect(() => {
    if (!nodeId || vms.length === 0) return;
    const controller = new AbortController();
    const since = new Date();
    since.setHours(since.getHours() - 1); // Last hour for sparklines

    const runningVMs = vms.filter((v) => v.status === "running");
    if (runningVMs.length === 0) return;

    // Fetch for up to 10 VMs to avoid excessive requests
    const fetchVMs = runningVMs.slice(0, 10);
    Promise.all(
      fetchVMs.map((vm) =>
        metricsApi
          .getVMMetrics(nodeId, vm.vmid, since.toISOString(), new Date().toISOString(), { signal: controller.signal })
          .then((res) => ({
            vmid: vm.vmid,
            records: toArray<VMMetricsRecord>(res.data),
          }))
          .catch(() => ({ vmid: vm.vmid, records: [] as VMMetricsRecord[] }))
      )
    ).then((results) => {
      if (controller.signal.aborted) return;
      const history: Record<number, Array<{ cpu: number; mem: number }>> = {};
      for (const { vmid, records } of results) {
        if (records.length > 0) {
          history[vmid] = records.map((r) => ({
            cpu: r.cpu_usage,
            mem: r.memory_total > 0 ? (r.memory_used / r.memory_total) * 100 : 0,
          }));
        }
      }
      setVMHistory(history);
    });
    return () => controller.abort();
  }, [nodeId, vms]);

  // Computed data for KPI cards
  const kpiData = useMemo(() => {
    const latest = latestMetrics;
    const s = summary;
    const cpuUsage = latest?.cpu_usage ?? s?.cpu_current ?? 0;
    const memoryUsage = latest?.memory_usage ?? s?.memory_current_percent ?? s?.memory_avg_percent ?? 0;
    const diskUsage = s?.disk_current_percent ?? s?.disk_avg_percent ?? 0;
    // Network: prefer live WS data, then latest REST record with actual data
    // Skip records with net_in=0 (first delta after scheduler restart is always 0)
    const lastNetRecord = [...metrics].reverse().find((m) => (m.net_in ?? 0) > 0 || (m.net_out ?? 0) > 0) ?? null;
    const netIn = latest?.network_in ?? lastNetRecord?.net_in ?? 0;
    const netOut = latest?.network_out ?? lastNetRecord?.net_out ?? 0;

    // Build sparkline history from WS metrics (last 20 points) or from regular metrics
    const historySource = wsMetrics.length >= 5 ? wsMetrics : [];
    const history = historySource.slice(-20).map((m) => ({
      cpu: m.cpu_usage,
      mem: m.memory_usage,
      disk: m.disk_usage,
      net: m.network_in + m.network_out,
    }));

    // If no WS data, use regular metrics
    if (history.length < 3 && metrics.length > 0) {
      const sampled = metrics.slice(-20);
      sampled.forEach((m) => {
        history.push({
          cpu: m.cpu_usage,
          mem: m.memory_total > 0 ? (m.memory_used / m.memory_total) * 100 : 0,
          disk: m.disk_total > 0 ? (m.disk_used / m.disk_total) * 100 : 0,
          net: m.net_in + m.net_out,
        });
      });
    }

    return { cpuUsage, memoryUsage, diskUsage, netIn, netOut, history };
  }, [latestMetrics, summary, wsMetrics, metrics]);

  // Overview multi-metric chart data
  const overviewChartData = useMemo(
    () =>
      metrics.map((m) => ({
        time: new Date(m.recorded_at).toLocaleTimeString("de-DE", {
          hour: "2-digit",
          minute: "2-digit",
        }),
        cpu: m.cpu_usage,
        ram: m.memory_total > 0 ? (m.memory_used / m.memory_total) * 100 : 0,
        disk: m.disk_total > 0 ? (m.disk_used / m.disk_total) * 100 : 0,
      })),
    [metrics]
  );

  const chartDataTimes = useMemo(
    () => overviewChartData.map((d) => d.time),
    [overviewChartData]
  );

  // Memory & Disk chart data
  const memDiskChartData = useMemo(
    () =>
      metrics.map((m) => ({
        time: new Date(m.recorded_at).toLocaleTimeString("de-DE", {
          hour: "2-digit",
          minute: "2-digit",
        }),
        ram: m.memory_total > 0 ? (m.memory_used / m.memory_total) * 100 : 0,
        ramUsedGB: m.memory_used / (1024 * 1024 * 1024),
        ramTotalGB: m.memory_total / (1024 * 1024 * 1024),
        disk: m.disk_total > 0 ? (m.disk_used / m.disk_total) * 100 : 0,
        diskUsedGB: m.disk_used / (1024 * 1024 * 1024),
        diskTotalGB: m.disk_total / (1024 * 1024 * 1024),
      })),
    [metrics]
  );

  // RRD memory chart data
  const rrdMemChartData = useMemo(
    () =>
      rrdData.map((d) => ({
        time: new Date(d.time * 1000).toLocaleTimeString("de-DE", {
          hour: "2-digit",
          minute: "2-digit",
        }),
        memPct: d.mem_total > 0 ? (d.mem_used / d.mem_total) * 100 : 0,
        rootPct: d.root_total > 0 ? (d.root_used / d.root_total) * 100 : 0,
      })),
    [rrdData]
  );

  // Load average from latest metrics
  const loadAvg = useMemo(() => {
    if (metrics.length === 0) return null;
    const latest = metrics[metrics.length - 1];
    const la = Array.isArray(latest.load_avg) ? latest.load_avg : [];
    return { load1: la[0] ?? 0, load5: la[1] ?? 0, load15: la[2] ?? 0 };
  }, [metrics]);

  // Top talker VMs
  const topTalkerVMs = useMemo(() => {
    return [...vms]
      .filter((v) => v.status === "running")
      .sort((a, b) => (b.net_in + b.net_out) - (a.net_in + a.net_out))
      .slice(0, 5);
  }, [vms]);

  const node = nodes.find((n) => n.id === nodeId);
  if (!node) return <Skeleton className="h-64 w-full" />;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href={`/nodes/${nodeId}`}>
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold">Monitoring</h1>
            <p className="text-sm text-muted-foreground">{node.name}</p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          {latestMetrics ? (
            <Badge variant="outline" className="bg-green-500/10 text-green-500 border-green-500/30">
              <Wifi className="mr-1 h-3 w-3" /> Live
            </Badge>
          ) : lastUpdated ? (
            <Badge variant="outline" className="text-muted-foreground">
              <WifiOff className="mr-1 h-3 w-3" /> Polling
            </Badge>
          ) : null}
          {lastUpdated && (
            <span className="text-xs text-muted-foreground">
              Daten von {lastUpdated.toLocaleTimeString("de-DE", { hour: "2-digit", minute: "2-digit", second: "2-digit" })}
            </span>
          )}
          <PeriodSelector period={period} onPeriodChange={setPeriod} />
        </div>
      </div>

      {/* Tabs */}
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="overview">
            <Activity className="mr-1.5 h-4 w-4" />
            Übersicht
          </TabsTrigger>
          <TabsTrigger value="cpu">
            <Cpu className="mr-1.5 h-4 w-4" />
            CPU & Load
          </TabsTrigger>
          <TabsTrigger value="memory">
            <MemoryStick className="mr-1.5 h-4 w-4" />
            Speicher & Disk
          </TabsTrigger>
          <TabsTrigger value="network">
            <Network className="mr-1.5 h-4 w-4" />
            Netzwerk
          </TabsTrigger>
          <TabsTrigger value="vms">
            <Server className="mr-1.5 h-4 w-4" />
            VMs
          </TabsTrigger>
        </TabsList>

        {/* ====== Tab 1: Übersicht ====== */}
        <TabsContent value="overview" className="space-y-6">
          <ErrorBoundary>
            <KPICards
              cpuUsage={kpiData.cpuUsage}
              memoryUsage={kpiData.memoryUsage}
              diskUsage={kpiData.diskUsage}
              netIn={kpiData.netIn}
              netOut={kpiData.netOut}
              history={kpiData.history}
            />
          </ErrorBoundary>

          {/* Multi-Metric Chart */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Ressourcen-Verlauf</CardTitle>
            </CardHeader>
            <CardContent>
              {overviewChartData.length === 0 ? (
                <div className="flex h-[350px] flex-col items-center justify-center gap-2 text-sm text-muted-foreground">
                  <RefreshCw className="h-5 w-5 animate-spin" />
                  <span>Warte auf Metriken-Daten...</span>
                  <span className="text-xs">Daten werden alle 60 Sekunden gesammelt.</span>
                </div>
              ) : (
                <ResponsiveContainer width="100%" height={350}>
                  <AreaChart data={overviewChartData}>
                    <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                    <XAxis dataKey="time" tick={{ fontSize: 10 }} />
                    <YAxis
                      domain={[0, 100]}
                      tick={{ fontSize: 10 }}
                      tickFormatter={(v) => `${v}%`}
                    />
                    <Tooltip
                      formatter={(v: number, name: string) => {
                        const labels: Record<string, string> = {
                          cpu: "CPU",
                          ram: "RAM",
                          disk: "Disk",
                        };
                        return [`${v.toFixed(1)}%`, labels[name] || name];
                      }}
                      contentStyle={{
                        backgroundColor: "hsl(var(--card))",
                        border: "1px solid hsl(var(--border))",
                        borderRadius: "0.5rem",
                      }}
                    />
                    <Legend
                      formatter={(value: string) => {
                        const labels: Record<string, string> = {
                          cpu: "CPU",
                          ram: "RAM",
                          disk: "Disk",
                        };
                        return labels[value] || value;
                      }}
                    />
                    <Area
                      type="monotone"
                      dataKey="cpu"
                      stroke="hsl(210, 80%, 55%)"
                      fill="hsl(210, 80%, 55%)"
                      fillOpacity={0.08}
                      strokeWidth={2}
                      name="cpu"
                    />
                    <Area
                      type="monotone"
                      dataKey="ram"
                      stroke="hsl(45, 93%, 47%)"
                      fill="hsl(45, 93%, 47%)"
                      fillOpacity={0.08}
                      strokeWidth={2}
                      name="ram"
                    />
                    <Area
                      type="monotone"
                      dataKey="disk"
                      stroke="hsl(280, 65%, 55%)"
                      fill="hsl(280, 65%, 55%)"
                      fillOpacity={0.08}
                      strokeWidth={2}
                      name="disk"
                    />
                    {/* Anomaly overlay dots */}
                    <AnomalyOverlay
                      anomalies={anomalies}
                      metric="cpu"
                      chartDataTimes={chartDataTimes}
                    />
                    <AnomalyOverlay
                      anomalies={anomalies}
                      metric="memory"
                      chartDataTimes={chartDataTimes}
                    />
                    <AnomalyOverlay
                      anomalies={anomalies}
                      metric="disk"
                      chartDataTimes={chartDataTimes}
                    />
                  </AreaChart>
                </ResponsiveContainer>
              )}
            </CardContent>
          </Card>

          {/* Predictions Info */}
          {predictions.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Trend-Prognosen</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid gap-3 sm:grid-cols-3">
                  {predictions.map((p) => {
                    const label = p.metric.includes("cpu") ? "CPU" : p.metric.includes("mem") ? "RAM" : "Disk";
                    const severityColor = p.severity === "critical" ? "text-red-500" : p.severity === "warning" ? "text-amber-500" : "text-blue-500";
                    return (
                      <div key={p.id} className="rounded-lg border p-3">
                        <div className="flex items-center justify-between">
                          <span className="text-sm font-medium">{label}</span>
                          <Badge variant="outline" className={severityColor}>
                            {p.severity}
                          </Badge>
                        </div>
                        <div className="mt-2 space-y-1 text-xs text-muted-foreground">
                          <div>Aktuell: {p.current_value.toFixed(1)}%</div>
                          <div>Prognose: {p.predicted_value.toFixed(1)}%</div>
                          {p.days_until_threshold != null && (
                            <div className={severityColor}>
                              Schwellwert ({p.threshold.toFixed(0)}%) in ~{p.days_until_threshold.toFixed(1)} Tagen
                            </div>
                          )}
                          <div>R²: {p.r_squared.toFixed(2)}</div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Anomalies Table */}
          {anomalies.filter((a) => !a.is_resolved).length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Aktive Anomalien</CardTitle>
              </CardHeader>
              <CardContent className="p-0">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b text-muted-foreground">
                      <th className="p-3 text-left font-medium">Metrik</th>
                      <th className="p-3 text-right font-medium">Wert</th>
                      <th className="p-3 text-right font-medium">Z-Score</th>
                      <th className="p-3 text-right font-medium">Durchschnitt</th>
                      <th className="p-3 text-left font-medium">Schwere</th>
                      <th className="p-3 text-left font-medium">Erkannt</th>
                    </tr>
                  </thead>
                  <tbody>
                    {anomalies
                      .filter((a) => !a.is_resolved)
                      .map((a) => {
                        const severityColor = a.severity === "critical" ? "text-red-500" : a.severity === "warning" ? "text-amber-500" : "text-blue-500";
                        return (
                          <tr key={a.id} className="border-b last:border-0">
                            <td className="p-3 font-medium">{a.metric}</td>
                            <td className="p-3 text-right">{a.value.toFixed(1)}%</td>
                            <td className="p-3 text-right">{a.z_score.toFixed(2)}</td>
                            <td className="p-3 text-right">{a.mean.toFixed(1)}%</td>
                            <td className={`p-3 ${severityColor}`}>{a.severity}</td>
                            <td className="p-3 text-muted-foreground">
                              {new Date(a.detected_at).toLocaleString("de-DE", {
                                hour: "2-digit",
                                minute: "2-digit",
                                day: "2-digit",
                                month: "2-digit",
                              })}
                            </td>
                          </tr>
                        );
                      })}
                  </tbody>
                </table>
              </CardContent>
            </Card>
          )}

          {/* Load Average Card */}
          {loadAvg && (
            <Card>
              <CardContent className="flex items-center gap-6 p-5">
                <Activity className="h-6 w-6 text-amber-500" />
                <div>
                  <p className="text-xs text-muted-foreground">Load Average</p>
                  <div className="flex gap-4 mt-1">
                    <div>
                      <span className="text-lg font-bold">{loadAvg.load1.toFixed(2)}</span>
                      <span className="ml-1 text-xs text-muted-foreground">1m</span>
                    </div>
                    <div>
                      <span className="text-lg font-bold">{loadAvg.load5.toFixed(2)}</span>
                      <span className="ml-1 text-xs text-muted-foreground">5m</span>
                    </div>
                    <div>
                      <span className="text-lg font-bold">{loadAvg.load15.toFixed(2)}</span>
                      <span className="ml-1 text-xs text-muted-foreground">15m</span>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        {/* ====== Tab 2: CPU & Load ====== */}
        <TabsContent value="cpu" className="space-y-6">
          <ErrorBoundary>
            <CPUDetailChart metrics={metrics} rrdData={rrdData} />
          </ErrorBoundary>
        </TabsContent>

        {/* ====== Tab 3: Speicher & Disk ====== */}
        <TabsContent value="memory" className="space-y-6">
          <ErrorBoundary>
            {/* RAM Usage Over Time */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">RAM-Auslastung</CardTitle>
              </CardHeader>
              <CardContent>
                {memDiskChartData.length === 0 ? (
                  <div className="flex h-[300px] items-center justify-center text-sm text-muted-foreground">
                    Warte auf Speicher-Daten... (Sammlung alle 60s)
                  </div>
                ) : (
                  <ResponsiveContainer width="100%" height={300}>
                    <AreaChart data={memDiskChartData}>
                      <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                      <XAxis dataKey="time" tick={{ fontSize: 10 }} />
                      <YAxis
                        domain={[0, 100]}
                        tick={{ fontSize: 10 }}
                        tickFormatter={(v) => `${v}%`}
                      />
                      <Tooltip
                        formatter={(v: number) => [`${v.toFixed(1)}%`, "RAM"]}
                        contentStyle={{
                          backgroundColor: "hsl(var(--card))",
                          border: "1px solid hsl(var(--border))",
                          borderRadius: "0.5rem",
                        }}
                      />
                      <Area
                        type="monotone"
                        dataKey="ram"
                        stroke="hsl(45, 93%, 47%)"
                        fill="hsl(45, 93%, 47%)"
                        fillOpacity={0.15}
                        strokeWidth={2}
                        name="RAM"
                      />
                    </AreaChart>
                  </ResponsiveContainer>
                )}
              </CardContent>
            </Card>

            {/* Current RAM info */}
            {metrics.length > 0 && (
              <div className="grid gap-4 sm:grid-cols-2">
                <Card>
                  <CardContent className="p-4">
                    <p className="text-xs text-muted-foreground">RAM belegt</p>
                    <p className="text-xl font-bold">
                      {formatBytes(metrics[metrics.length - 1].memory_used)} /{" "}
                      {formatBytes(metrics[metrics.length - 1].memory_total)}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {formatPercentage(
                        metrics[metrics.length - 1].memory_total > 0
                          ? (metrics[metrics.length - 1].memory_used /
                              metrics[metrics.length - 1].memory_total) *
                              100
                          : 0
                      )}
                    </p>
                  </CardContent>
                </Card>
                <Card>
                  <CardContent className="p-4">
                    <p className="text-xs text-muted-foreground">Disk belegt</p>
                    <p className="text-xl font-bold">
                      {formatBytes(metrics[metrics.length - 1].disk_used)} /{" "}
                      {formatBytes(metrics[metrics.length - 1].disk_total)}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {formatPercentage(
                        metrics[metrics.length - 1].disk_total > 0
                          ? (metrics[metrics.length - 1].disk_used /
                              metrics[metrics.length - 1].disk_total) *
                              100
                          : 0
                      )}
                    </p>
                  </CardContent>
                </Card>
              </div>
            )}

            {/* Disk Usage Over Time */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Disk-Auslastung</CardTitle>
              </CardHeader>
              <CardContent>
                {memDiskChartData.length === 0 ? (
                  <div className="flex h-[300px] items-center justify-center text-sm text-muted-foreground">
                    Warte auf Disk-Daten... (Sammlung alle 60s)
                  </div>
                ) : (
                  <ResponsiveContainer width="100%" height={300}>
                    <AreaChart data={memDiskChartData}>
                      <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                      <XAxis dataKey="time" tick={{ fontSize: 10 }} />
                      <YAxis
                        domain={[0, 100]}
                        tick={{ fontSize: 10 }}
                        tickFormatter={(v) => `${v}%`}
                      />
                      <Tooltip
                        formatter={(v: number) => [`${v.toFixed(1)}%`, "Disk"]}
                        contentStyle={{
                          backgroundColor: "hsl(var(--card))",
                          border: "1px solid hsl(var(--border))",
                          borderRadius: "0.5rem",
                        }}
                      />
                      <Area
                        type="monotone"
                        dataKey="disk"
                        stroke="hsl(280, 65%, 55%)"
                        fill="hsl(280, 65%, 55%)"
                        fillOpacity={0.15}
                        strokeWidth={2}
                        name="Disk"
                      />
                    </AreaChart>
                  </ResponsiveContainer>
                )}
              </CardContent>
            </Card>

            {/* RRD-based root/mem if available */}
            {rrdMemChartData.length > 0 && (
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">RRD: Speicher & Root (Proxmox)</CardTitle>
                </CardHeader>
                <CardContent>
                  <ResponsiveContainer width="100%" height={300}>
                    <AreaChart data={rrdMemChartData}>
                      <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                      <XAxis dataKey="time" tick={{ fontSize: 10 }} />
                      <YAxis
                        domain={[0, 100]}
                        tick={{ fontSize: 10 }}
                        tickFormatter={(v) => `${v}%`}
                      />
                      <Tooltip
                        formatter={(v: number, name: string) => {
                          const labels: Record<string, string> = {
                            memPct: "RAM",
                            rootPct: "Root-FS",
                          };
                          return [`${v.toFixed(1)}%`, labels[name] || name];
                        }}
                        contentStyle={{
                          backgroundColor: "hsl(var(--card))",
                          border: "1px solid hsl(var(--border))",
                          borderRadius: "0.5rem",
                        }}
                      />
                      <Legend
                        formatter={(value: string) => {
                          const labels: Record<string, string> = {
                            memPct: "RAM",
                            rootPct: "Root-FS",
                          };
                          return labels[value] || value;
                        }}
                      />
                      <Area
                        type="monotone"
                        dataKey="memPct"
                        stroke="hsl(45, 93%, 47%)"
                        fill="hsl(45, 93%, 47%)"
                        fillOpacity={0.1}
                        strokeWidth={2}
                        name="memPct"
                      />
                      <Area
                        type="monotone"
                        dataKey="rootPct"
                        stroke="hsl(200, 80%, 50%)"
                        fill="hsl(200, 80%, 50%)"
                        fillOpacity={0.1}
                        strokeWidth={2}
                        name="rootPct"
                      />
                    </AreaChart>
                  </ResponsiveContainer>
                </CardContent>
              </Card>
            )}
          </ErrorBoundary>
        </TabsContent>

        {/* ====== Tab 4: Netzwerk ====== */}
        <TabsContent value="network" className="space-y-6">
          <ErrorBoundary>
            {/* RRD-basierte Bandbreite - zuverlässige Quelle */}
            <RRDBandwidth nodeId={nodeId} />

            {/* Top-Talker VMs */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Top-Talker VMs</CardTitle>
              </CardHeader>
              <CardContent className="p-0">
                {topTalkerVMs.length === 0 ? (
                  <div className="p-6 text-center text-sm text-muted-foreground">
                    Keine laufenden VMs.
                  </div>
                ) : (
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b text-muted-foreground">
                        <th className="p-3 text-left font-medium">VM</th>
                        <th className="p-3 text-right font-medium">In</th>
                        <th className="p-3 text-right font-medium">Out</th>
                        <th className="p-3 text-right font-medium">Gesamt</th>
                      </tr>
                    </thead>
                    <tbody>
                      {topTalkerVMs.map((vm) => (
                        <tr key={vm.vmid} className="border-b last:border-0">
                          <td className="p-3">
                            <span className="font-medium">{vm.name || `VM ${vm.vmid}`}</span>
                            <span className="ml-2 text-xs text-muted-foreground">#{vm.vmid}</span>
                          </td>
                          <td className="p-3 text-right text-blue-500">
                            {formatBandwidth(vm.net_in)}
                          </td>
                          <td className="p-3 text-right text-green-500">
                            {formatBandwidth(vm.net_out)}
                          </td>
                          <td className="p-3 text-right font-bold">
                            {formatBandwidth(vm.net_in + vm.net_out)}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </CardContent>
            </Card>

            {/* VM Network Traffic */}
            <VMNetworkTraffic nodeId={nodeId} />
          </ErrorBoundary>
        </TabsContent>

        {/* ====== Tab 5: VMs ====== */}
        <TabsContent value="vms" className="space-y-6">
          <ErrorBoundary>
            <VMMetricsTable vms={vms} nodeId={nodeId} vmHistory={vmHistory} />
          </ErrorBoundary>
        </TabsContent>
      </Tabs>
    </div>
  );
}
