"use client";

import { useEffect, useState, useMemo } from "react";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  Legend,
  PieChart,
  Pie,
  Cell,
} from "recharts";
import {
  Server,
  Cpu,
  MemoryStick,
  HardDrive,
  CheckCircle2,
  XCircle,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { clusterApi } from "@/lib/api";
import { formatBytes, formatBandwidth } from "@/lib/utils";

interface NodeStatusData {
  node_id: string;
  node_name: string;
  is_online: boolean;
  status?: {
    cpu_usage: number;
    cpu_cores: number;
    mem_used: number;
    mem_total: number;
    disk_used: number;
    disk_total: number;
    uptime: number;
    vm_count: number;
  };
}

interface ClusterSummary {
  total_nodes: number;
  online_nodes: number;
  total_cpu_usage: number;
  total_mem_used: number;
  total_mem_total: number;
  total_disk_used: number;
  total_disk_total: number;
  nodes: NodeStatusData[];
}

interface HistoryPoint {
  time: string;
  cpu_avg: number;
  mem_pct: number;
  disk_pct: number;
  net_in: number;
  net_out: number;
}

const periods = [
  { label: "1h", value: "1h" },
  { label: "6h", value: "6h" },
  { label: "24h", value: "24h" },
  { label: "7d", value: "7d" },
  { label: "30d", value: "30d" },
];

const COLORS = [
  "hsl(210, 80%, 55%)",
  "hsl(25, 95%, 53%)",
  "hsl(45, 93%, 47%)",
  "hsl(280, 65%, 55%)",
  "hsl(142, 71%, 45%)",
  "hsl(0, 72%, 51%)",
  "hsl(180, 60%, 50%)",
  "hsl(330, 70%, 55%)",
];

export default function ClusterDashboardPage() {
  const [summary, setSummary] = useState<ClusterSummary | null>(null);
  const [history, setHistory] = useState<HistoryPoint[]>([]);
  const [period, setPeriod] = useState("24h");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    Promise.all([
      clusterApi.getSummary().then((r) => setSummary(r.data)).catch(() => null),
      clusterApi
        .getHistory(period)
        .then((r) => {
          const data = Array.isArray(r.data) ? r.data : [];
          setHistory(data);
        })
        .catch(() => setHistory([])),
    ]).finally(() => setLoading(false));
  }, [period]);

  const chartData = useMemo(
    () =>
      history.map((h) => ({
        time: new Date(h.time).toLocaleTimeString("de-DE", {
          hour: "2-digit",
          minute: "2-digit",
        }),
        cpu: h.cpu_avg,
        ram: h.mem_pct,
        disk: h.disk_pct,
        netIn: h.net_in,
        netOut: h.net_out,
      })),
    [history]
  );

  const memPieData = useMemo(() => {
    if (!summary) return [];
    const used = summary.total_mem_used;
    const free = summary.total_mem_total - summary.total_mem_used;
    return [
      { name: "Belegt", value: Math.max(0, used) },
      { name: "Frei", value: Math.max(0, free) },
    ];
  }, [summary]);

  const diskPieData = useMemo(() => {
    if (!summary) return [];
    const used = summary.total_disk_used;
    const free = summary.total_disk_total - summary.total_disk_used;
    return [
      { name: "Belegt", value: Math.max(0, used) },
      { name: "Frei", value: Math.max(0, free) },
    ];
  }, [summary]);

  if (loading && !summary) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-48" />
        <div className="grid gap-4 sm:grid-cols-4">
          {[1, 2, 3, 4].map((i) => (
            <Skeleton key={i} className="h-24" />
          ))}
        </div>
        <Skeleton className="h-[350px]" />
      </div>
    );
  }

  const avgCpu =
    summary && summary.online_nodes > 0
      ? summary.total_cpu_usage / summary.online_nodes
      : 0;
  const memPct =
    summary && summary.total_mem_total > 0
      ? (summary.total_mem_used / summary.total_mem_total) * 100
      : 0;
  const diskPct =
    summary && summary.total_disk_total > 0
      ? (summary.total_disk_used / summary.total_disk_total) * 100
      : 0;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Cluster-Dashboard</h1>
          <p className="text-sm text-muted-foreground">
            Aggregierte Metriken aller Nodes
          </p>
        </div>
        <div className="flex gap-1">
          {periods.map((p) => (
            <Button
              key={p.value}
              variant={period === p.value ? "default" : "outline"}
              size="sm"
              onClick={() => setPeriod(p.value)}
            >
              {p.label}
            </Button>
          ))}
        </div>
      </div>

      {/* KPI Cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardContent className="flex items-center gap-4 p-5">
            <div className="rounded-lg bg-blue-500/10 p-3">
              <Server className="h-5 w-5 text-blue-500" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Nodes</p>
              <p className="text-2xl font-bold">
                {summary?.online_nodes ?? 0}
                <span className="text-sm font-normal text-muted-foreground">
                  {" "}
                  / {summary?.total_nodes ?? 0}
                </span>
              </p>
              <p className="text-xs text-green-500">Online</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="flex items-center gap-4 p-5">
            <div className="rounded-lg bg-orange-500/10 p-3">
              <Cpu className="h-5 w-5 text-orange-500" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">CPU (Durchschnitt)</p>
              <p className="text-2xl font-bold">{avgCpu.toFixed(1)}%</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="flex items-center gap-4 p-5">
            <div className="rounded-lg bg-yellow-500/10 p-3">
              <MemoryStick className="h-5 w-5 text-yellow-500" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">RAM gesamt</p>
              <p className="text-2xl font-bold">{memPct.toFixed(1)}%</p>
              <p className="text-xs text-muted-foreground">
                {formatBytes(summary?.total_mem_used ?? 0)} /{" "}
                {formatBytes(summary?.total_mem_total ?? 0)}
              </p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="flex items-center gap-4 p-5">
            <div className="rounded-lg bg-purple-500/10 p-3">
              <HardDrive className="h-5 w-5 text-purple-500" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Disk gesamt</p>
              <p className="text-2xl font-bold">{diskPct.toFixed(1)}%</p>
              <p className="text-xs text-muted-foreground">
                {formatBytes(summary?.total_disk_used ?? 0)} /{" "}
                {formatBytes(summary?.total_disk_total ?? 0)}
              </p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Charts Row: Pie + Timeline */}
      <div className="grid gap-4 lg:grid-cols-3">
        {/* Pie Charts */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Speicher-Verteilung</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-6">
              <div>
                <p className="mb-2 text-center text-xs text-muted-foreground">
                  RAM
                </p>
                <ResponsiveContainer width="100%" height={140}>
                  <PieChart>
                    <Pie
                      data={memPieData}
                      cx="50%"
                      cy="50%"
                      innerRadius={40}
                      outerRadius={60}
                      dataKey="value"
                      strokeWidth={2}
                    >
                      <Cell fill="hsl(45, 93%, 47%)" />
                      <Cell fill="hsl(var(--muted))" />
                    </Pie>
                    <Tooltip
                      formatter={(v: number) => formatBytes(v)}
                      contentStyle={{
                        backgroundColor: "hsl(var(--card))",
                        border: "1px solid hsl(var(--border))",
                        borderRadius: "0.5rem",
                      }}
                    />
                  </PieChart>
                </ResponsiveContainer>
              </div>
              <div>
                <p className="mb-2 text-center text-xs text-muted-foreground">
                  Disk
                </p>
                <ResponsiveContainer width="100%" height={140}>
                  <PieChart>
                    <Pie
                      data={diskPieData}
                      cx="50%"
                      cy="50%"
                      innerRadius={40}
                      outerRadius={60}
                      dataKey="value"
                      strokeWidth={2}
                    >
                      <Cell fill="hsl(280, 65%, 55%)" />
                      <Cell fill="hsl(var(--muted))" />
                    </Pie>
                    <Tooltip
                      formatter={(v: number) => formatBytes(v)}
                      contentStyle={{
                        backgroundColor: "hsl(var(--card))",
                        border: "1px solid hsl(var(--border))",
                        borderRadius: "0.5rem",
                      }}
                    />
                  </PieChart>
                </ResponsiveContainer>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Aggregated Timeline */}
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle className="text-base">Cluster-Verlauf</CardTitle>
          </CardHeader>
          <CardContent>
            {chartData.length === 0 ? (
              <div className="flex h-[350px] items-center justify-center text-sm text-muted-foreground">
                Keine Daten im gewaehlten Zeitraum.
              </div>
            ) : (
              <ResponsiveContainer width="100%" height={350}>
                <AreaChart data={chartData}>
                  <CartesianGrid
                    strokeDasharray="3 3"
                    className="stroke-border"
                  />
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
                </AreaChart>
              </ResponsiveContainer>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Node Comparison Table */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Node-Vergleich</CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-muted-foreground">
                  <th className="p-3 text-left font-medium">Node</th>
                  <th className="p-3 text-center font-medium">Status</th>
                  <th className="p-3 text-right font-medium">CPU</th>
                  <th className="p-3 text-right font-medium">RAM</th>
                  <th className="p-3 text-right font-medium">Disk</th>
                  <th className="p-3 text-right font-medium">VMs</th>
                  <th className="p-3 text-right font-medium">Uptime</th>
                </tr>
              </thead>
              <tbody>
                {(summary?.nodes ?? []).map((node, idx) => {
                  const memNodePct =
                    node.status && node.status.mem_total > 0
                      ? (node.status.mem_used / node.status.mem_total) * 100
                      : 0;
                  const diskNodePct =
                    node.status && node.status.disk_total > 0
                      ? (node.status.disk_used / node.status.disk_total) * 100
                      : 0;
                  const uptimeDays = node.status
                    ? (node.status.uptime / 86400).toFixed(1)
                    : "-";
                  return (
                    <tr key={node.node_id} className="border-b last:border-0">
                      <td className="p-3">
                        <div className="flex items-center gap-2">
                          <span
                            className="inline-block h-2 w-2 rounded-full"
                            style={{
                              backgroundColor: COLORS[idx % COLORS.length],
                            }}
                          />
                          <span className="font-medium">{node.node_name}</span>
                        </div>
                      </td>
                      <td className="p-3 text-center">
                        {node.is_online ? (
                          <Badge
                            variant="outline"
                            className="bg-green-500/10 text-green-500 border-green-500/30"
                          >
                            <CheckCircle2 className="mr-1 h-3 w-3" />
                            Online
                          </Badge>
                        ) : (
                          <Badge
                            variant="outline"
                            className="bg-red-500/10 text-red-500 border-red-500/30"
                          >
                            <XCircle className="mr-1 h-3 w-3" />
                            Offline
                          </Badge>
                        )}
                      </td>
                      <td className="p-3 text-right">
                        {node.status
                          ? `${node.status.cpu_usage.toFixed(1)}%`
                          : "-"}
                      </td>
                      <td className="p-3 text-right">
                        {node.status ? (
                          <span>
                            {memNodePct.toFixed(1)}%
                            <span className="ml-1 text-xs text-muted-foreground">
                              ({formatBytes(node.status.mem_used)})
                            </span>
                          </span>
                        ) : (
                          "-"
                        )}
                      </td>
                      <td className="p-3 text-right">
                        {node.status ? (
                          <span>
                            {diskNodePct.toFixed(1)}%
                            <span className="ml-1 text-xs text-muted-foreground">
                              ({formatBytes(node.status.disk_used)})
                            </span>
                          </span>
                        ) : (
                          "-"
                        )}
                      </td>
                      <td className="p-3 text-right">
                        {node.status?.vm_count ?? "-"}
                      </td>
                      <td className="p-3 text-right">
                        {uptimeDays !== "-" ? `${uptimeDays}d` : "-"}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
