"use client";

import { useMemo } from "react";
import {
  AreaChart,
  Area,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  Legend,
} from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { MetricsRecord, RRDDataPoint } from "@/types/api";

interface CPUDetailChartProps {
  metrics: MetricsRecord[];
  rrdData: RRDDataPoint[];
}

function computeStats(values: number[]) {
  if (values.length === 0) return { min: 0, avg: 0, max: 0, p95: 0 };
  const sorted = [...values].sort((a, b) => a - b);
  const sum = sorted.reduce((a, b) => a + b, 0);
  const avg = sum / sorted.length;
  const p95Idx = Math.floor(sorted.length * 0.95);
  return {
    min: sorted[0],
    avg,
    max: sorted[sorted.length - 1],
    p95: sorted[Math.min(p95Idx, sorted.length - 1)],
  };
}

export function CPUDetailChart({ metrics, rrdData }: CPUDetailChartProps) {
  const cpuChartData = useMemo(
    () =>
      metrics.map((m) => ({
        time: new Date(m.recorded_at).toLocaleTimeString("de-DE", {
          hour: "2-digit",
          minute: "2-digit",
        }),
        cpu: m.cpu_usage,
      })),
    [metrics]
  );

  const ioWaitData = useMemo(
    () =>
      rrdData.map((d) => ({
        time: new Date(d.time * 1000).toLocaleTimeString("de-DE", {
          hour: "2-digit",
          minute: "2-digit",
        }),
        ioWait: (d.io_wait ?? 0) * 100,
      })),
    [rrdData]
  );

  const loadData = useMemo(
    () =>
      metrics.map((m) => {
        const la = Array.isArray(m.load_avg) ? m.load_avg : [];
        return {
          time: new Date(m.recorded_at).toLocaleTimeString("de-DE", {
            hour: "2-digit",
            minute: "2-digit",
          }),
          load1: la[0] ?? 0,
          load5: la[1] ?? 0,
          load15: la[2] ?? 0,
        };
      }),
    [metrics]
  );

  const cpuStats = useMemo(() => computeStats(metrics.map((m) => m.cpu_usage)), [metrics]);

  return (
    <div className="space-y-4">
      {/* CPU Usage Area Chart */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">CPU-Auslastung</CardTitle>
        </CardHeader>
        <CardContent>
          {cpuChartData.length === 0 ? (
            <div className="flex h-[300px] items-center justify-center text-sm text-muted-foreground">
              Warte auf CPU-Daten... (Sammlung alle 60s)
            </div>
          ) : (
            <ResponsiveContainer width="100%" height={300}>
              <AreaChart data={cpuChartData}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                <XAxis dataKey="time" tick={{ fontSize: 10 }} />
                <YAxis
                  domain={[0, 100]}
                  tick={{ fontSize: 10 }}
                  tickFormatter={(v) => `${v}%`}
                />
                <Tooltip
                  formatter={(v: number) => [`${v.toFixed(1)}%`, "CPU"]}
                  contentStyle={{
                    backgroundColor: "hsl(var(--card))",
                    border: "1px solid hsl(var(--border))",
                    borderRadius: "0.5rem",
                  }}
                />
                <Area
                  type="monotone"
                  dataKey="cpu"
                  stroke="hsl(210, 80%, 55%)"
                  fill="hsl(210, 80%, 55%)"
                  fillOpacity={0.15}
                  strokeWidth={2}
                  name="CPU"
                />
              </AreaChart>
            </ResponsiveContainer>
          )}
        </CardContent>
      </Card>

      {/* CPU Statistics */}
      <div className="grid gap-4 sm:grid-cols-4">
        {[
          { label: "Minimum", value: cpuStats.min, color: "text-green-500" },
          { label: "Durchschnitt", value: cpuStats.avg, color: "text-blue-500" },
          { label: "Maximum", value: cpuStats.max, color: "text-red-500" },
          { label: "P95", value: cpuStats.p95, color: "text-amber-500" },
        ].map((s) => (
          <Card key={s.label}>
            <CardContent className="p-4">
              <p className="text-xs text-muted-foreground">{s.label}</p>
              <p className={`text-xl font-bold ${s.color}`}>{s.value.toFixed(1)}%</p>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* IO Wait Chart */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">IO-Wait</CardTitle>
        </CardHeader>
        <CardContent>
          {ioWaitData.length === 0 ? (
            <div className="flex h-[250px] items-center justify-center text-sm text-muted-foreground">
              Keine IO-Wait-Daten verfügbar (RRD-Daten erforderlich).
            </div>
          ) : (
            <ResponsiveContainer width="100%" height={250}>
              <AreaChart data={ioWaitData}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                <XAxis dataKey="time" tick={{ fontSize: 10 }} />
                <YAxis
                  domain={[0, "auto"]}
                  tick={{ fontSize: 10 }}
                  tickFormatter={(v) => `${v.toFixed(1)}%`}
                />
                <Tooltip
                  formatter={(v: number) => [`${v.toFixed(2)}%`, "IO-Wait"]}
                  contentStyle={{
                    backgroundColor: "hsl(var(--card))",
                    border: "1px solid hsl(var(--border))",
                    borderRadius: "0.5rem",
                  }}
                />
                <Area
                  type="monotone"
                  dataKey="ioWait"
                  stroke="hsl(25, 95%, 53%)"
                  fill="hsl(25, 95%, 53%)"
                  fillOpacity={0.15}
                  strokeWidth={2}
                  name="IO-Wait"
                />
              </AreaChart>
            </ResponsiveContainer>
          )}
        </CardContent>
      </Card>

      {/* Load Average Chart */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Load Average</CardTitle>
        </CardHeader>
        <CardContent>
          {loadData.length === 0 ? (
            <div className="flex h-[250px] items-center justify-center text-sm text-muted-foreground">
              Warte auf Load-Daten... (Sammlung alle 60s)
            </div>
          ) : (
            <ResponsiveContainer width="100%" height={250}>
              <LineChart data={loadData}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                <XAxis dataKey="time" tick={{ fontSize: 10 }} />
                <YAxis tick={{ fontSize: 10 }} />
                <Tooltip
                  formatter={(v: number, name: string) => {
                    const labels: Record<string, string> = {
                      load1: "1 Min",
                      load5: "5 Min",
                      load15: "15 Min",
                    };
                    return [v.toFixed(2), labels[name] || name];
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
                      load1: "1 Min",
                      load5: "5 Min",
                      load15: "15 Min",
                    };
                    return labels[value] || value;
                  }}
                />
                <Line
                  type="monotone"
                  dataKey="load1"
                  stroke="hsl(0, 84%, 60%)"
                  strokeWidth={2}
                  dot={false}
                  name="load1"
                />
                <Line
                  type="monotone"
                  dataKey="load5"
                  stroke="hsl(25, 95%, 53%)"
                  strokeWidth={2}
                  dot={false}
                  name="load5"
                />
                <Line
                  type="monotone"
                  dataKey="load15"
                  stroke="hsl(45, 93%, 47%)"
                  strokeWidth={2}
                  dot={false}
                  name="load15"
                />
              </LineChart>
            </ResponsiveContainer>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
