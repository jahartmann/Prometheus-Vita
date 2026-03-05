"use client";

import { useEffect, useMemo, useState } from "react";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  PieChart,
  Pie,
  Cell,
} from "recharts";
import { ArrowDownToLine, ArrowUpFromLine } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { metricsApi, nodeApi, toArray } from "@/lib/api";
import { formatBandwidth, formatTraffic, bytesToMbits } from "@/lib/utils";
import type { VM, VMMetricsRecord, NetworkSummary } from "@/types/api";

const periods = [
  { label: "1h", value: "1h" },
  { label: "24h", value: "24h" },
  { label: "7d", value: "7d" },
  { label: "30d", value: "30d" },
];

const PIE_COLORS = [
  "hsl(210, 80%, 55%)",
  "hsl(142, 71%, 45%)",
  "hsl(25, 95%, 53%)",
  "hsl(45, 93%, 47%)",
  "hsl(280, 65%, 55%)",
  "hsl(0, 84%, 60%)",
  "hsl(180, 60%, 50%)",
  "hsl(330, 80%, 55%)",
];

interface VMTrafficEntry {
  vmid: number;
  name: string;
  vmType: string;
  totalIn: number;
  totalOut: number;
  total: number;
  avgIn: number;
  avgOut: number;
  peakIn: number;
  peakOut: number;
}

interface VMNetworkTrafficProps {
  nodeId: string;
}

export function VMNetworkTraffic({ nodeId }: VMNetworkTrafficProps) {
  const [period, setPeriod] = useState("24h");
  const [vms, setVMs] = useState<VM[]>([]);
  const [vmTraffic, setVMTraffic] = useState<VMTrafficEntry[]>([]);
  const [selectedVM, setSelectedVM] = useState<number | null>(null);
  const [vmMetrics, setVMMetrics] = useState<VMMetricsRecord[]>([]);
  const [loading, setLoading] = useState(false);

  // Load VMs
  useEffect(() => {
    if (!nodeId) return;
    nodeApi
      .getVMs(nodeId)
      .then((res) => setVMs(toArray<VM>(res.data)))
      .catch(() => setVMs([]));
  }, [nodeId]);

  // Load traffic summaries for all VMs
  useEffect(() => {
    if (!nodeId || vms.length === 0) return;
    setLoading(true);

    const fetchPromises = vms.map((vm) =>
      metricsApi
        .getVMNetworkSummary(nodeId, vm.vmid, period)
        .then((res) => {
          const d = res.data as NetworkSummary | null;
          if (!d) return null;
          return {
            vmid: vm.vmid,
            name: vm.name || `VM ${vm.vmid}`,
            vmType: vm.type,
            totalIn: d.total_in,
            totalOut: d.total_out,
            total: d.total_in + d.total_out,
            avgIn: d.avg_in_rate,
            avgOut: d.avg_out_rate,
            peakIn: d.peak_in_rate,
            peakOut: d.peak_out_rate,
          } as VMTrafficEntry;
        })
        .catch(() => null)
    );

    Promise.all(fetchPromises).then((results) => {
      const entries = results.filter((r): r is VMTrafficEntry => r !== null && r.total > 0);
      entries.sort((a, b) => b.total - a.total);
      setVMTraffic(entries);
      setLoading(false);
    });
  }, [nodeId, vms, period]);

  // Load selected VM metrics
  useEffect(() => {
    if (!nodeId || selectedVM === null) {
      setVMMetrics([]);
      return;
    }

    const hours =
      period === "30d" ? 720 :
      period === "7d" ? 168 :
      period === "24h" ? 24 : 1;
    const start = new Date();
    start.setHours(start.getHours() - hours);

    metricsApi
      .getVMMetrics(nodeId, selectedVM, start.toISOString(), new Date().toISOString())
      .then((res) => setVMMetrics(toArray<VMMetricsRecord>(res.data)))
      .catch(() => setVMMetrics([]));
  }, [nodeId, selectedVM, period]);

  const selectedVMChartData = useMemo(() => {
    const isLongPeriod = period === "7d" || period === "30d";
    return vmMetrics.map((m) => ({
      time: new Date(m.recorded_at).toLocaleString("de-DE", {
        ...(isLongPeriod
          ? { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" }
          : { hour: "2-digit", minute: "2-digit" }),
      }),
      netIn: bytesToMbits(m.net_in),
      netOut: bytesToMbits(m.net_out),
    }));
  }, [vmMetrics, period]);

  const pieData = useMemo(() => {
    return vmTraffic.slice(0, 8).map((entry) => ({
      name: entry.name,
      value: entry.total,
    }));
  }, [vmTraffic]);

  const totalTraffic = useMemo(() => {
    return vmTraffic.reduce((sum, v) => sum + v.total, 0);
  }, [vmTraffic]);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold">VM Netzwerk-Traffic</h3>
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

      <div className="grid gap-4 lg:grid-cols-3">
        {/* VM Ranking Table */}
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle className="text-base">VM Traffic-Ranking</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            {loading ? (
              <div className="p-6 text-center text-sm text-muted-foreground">Lade...</div>
            ) : vmTraffic.length === 0 ? (
              <div className="p-6 text-center text-sm text-muted-foreground">
                Keine VM-Traffic-Daten verfuegbar.
              </div>
            ) : (
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b">
                    <th className="p-3 text-left font-medium">VM</th>
                    <th className="p-3 text-left font-medium">VMID</th>
                    <th className="p-3 text-right font-medium">Eingehend</th>
                    <th className="p-3 text-right font-medium">Ausgehend</th>
                    <th className="p-3 text-right font-medium">Gesamt</th>
                    <th className="p-3 text-right font-medium">Anteil</th>
                  </tr>
                </thead>
                <tbody>
                  {vmTraffic.map((entry) => (
                    <tr
                      key={entry.vmid}
                      className={`border-b last:border-0 cursor-pointer transition-colors hover:bg-muted/50 ${
                        selectedVM === entry.vmid ? "bg-muted" : ""
                      }`}
                      onClick={() => setSelectedVM(selectedVM === entry.vmid ? null : entry.vmid)}
                    >
                      <td className="p-3">
                        <div className="flex items-center gap-2">
                          <span className="font-medium">{entry.name}</span>
                          <Badge variant="outline" className="text-xs">
                            {entry.vmType}
                          </Badge>
                        </div>
                      </td>
                      <td className="p-3 font-mono text-xs">{entry.vmid}</td>
                      <td className="p-3 text-right text-blue-500">
                        <ArrowDownToLine className="mr-1 inline h-3 w-3" />
                        {formatTraffic(entry.totalIn)}
                      </td>
                      <td className="p-3 text-right text-green-500">
                        <ArrowUpFromLine className="mr-1 inline h-3 w-3" />
                        {formatTraffic(entry.totalOut)}
                      </td>
                      <td className="p-3 text-right font-bold">
                        {formatTraffic(entry.total)}
                      </td>
                      <td className="p-3 text-right text-muted-foreground">
                        {totalTraffic > 0
                          ? `${((entry.total / totalTraffic) * 100).toFixed(1)}%`
                          : "0%"}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </CardContent>
        </Card>

        {/* Pie Chart */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Verteilung</CardTitle>
          </CardHeader>
          <CardContent>
            {pieData.length === 0 ? (
              <div className="flex h-[250px] items-center justify-center text-sm text-muted-foreground">
                Keine Daten
              </div>
            ) : (
              <ResponsiveContainer width="100%" height={250}>
                <PieChart>
                  <Pie
                    data={pieData}
                    cx="50%"
                    cy="50%"
                    innerRadius={50}
                    outerRadius={90}
                    dataKey="value"
                    label={({ name, percent }) =>
                      `${name} (${(percent * 100).toFixed(0)}%)`
                    }
                    labelLine={false}
                  >
                    {pieData.map((_, idx) => (
                      <Cell
                        key={idx}
                        fill={PIE_COLORS[idx % PIE_COLORS.length]}
                      />
                    ))}
                  </Pie>
                  <Tooltip
                    formatter={(value: number) => formatTraffic(value)}
                    contentStyle={{
                      backgroundColor: "hsl(var(--card))",
                      border: "1px solid hsl(var(--border))",
                      borderRadius: "0.5rem",
                    }}
                  />
                </PieChart>
              </ResponsiveContainer>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Selected VM Detail Chart */}
      {selectedVM !== null && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">
              Netzwerk-Verlauf:{" "}
              {vmTraffic.find((v) => v.vmid === selectedVM)?.name ?? `VM ${selectedVM}`}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {selectedVMChartData.length === 0 ? (
              <div className="flex h-[250px] items-center justify-center text-sm text-muted-foreground">
                Keine Daten fuer diese VM.
              </div>
            ) : (
              <ResponsiveContainer width="100%" height={250}>
                <AreaChart data={selectedVMChartData}>
                  <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                  <XAxis dataKey="time" tick={{ fontSize: 10 }} />
                  <YAxis
                    tick={{ fontSize: 10 }}
                    tickFormatter={(v) => `${v.toFixed(1)}`}
                    label={{
                      value: "Mbit/s",
                      angle: -90,
                      position: "insideLeft",
                      style: { fontSize: 10, fill: "hsl(var(--muted-foreground))" },
                    }}
                  />
                  <Tooltip
                    formatter={(value: number, name: string) => {
                      const label = name === "netIn" ? "Eingehend" : "Ausgehend";
                      return [`${value.toFixed(2)} Mbit/s`, label];
                    }}
                    contentStyle={{
                      backgroundColor: "hsl(var(--card))",
                      border: "1px solid hsl(var(--border))",
                      borderRadius: "0.5rem",
                    }}
                  />
                  <Area
                    type="monotone"
                    dataKey="netIn"
                    stroke="hsl(210, 80%, 55%)"
                    fill="hsl(210, 80%, 55%)"
                    fillOpacity={0.15}
                    strokeWidth={2}
                    name="netIn"
                  />
                  <Area
                    type="monotone"
                    dataKey="netOut"
                    stroke="hsl(142, 71%, 45%)"
                    fill="hsl(142, 71%, 45%)"
                    fillOpacity={0.15}
                    strokeWidth={2}
                    name="netOut"
                  />
                </AreaChart>
              </ResponsiveContainer>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
