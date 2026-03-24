"use client";

import { useMemo } from "react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  AreaChart,
  Area,
  Legend,
} from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatBandwidth } from "@/lib/utils";
import type { MetricsRecord } from "@/types/api";

const tooltipStyle = {
  backgroundColor: "hsl(var(--card))",
  border: "1px solid hsl(var(--border))",
  borderRadius: "8px",
  fontSize: "12px",
};

interface MetricsChartsProps {
  metrics: MetricsRecord[];
}

export function MetricsCharts({ metrics }: MetricsChartsProps) {
  const chartData = useMemo(
    () =>
      metrics.map((m) => ({
        time: new Date(m.recorded_at).toLocaleTimeString("de-DE", {
          hour: "2-digit",
          minute: "2-digit",
        }),
        cpu: m.cpu_usage,
        memPercent:
          m.memory_total > 0 ? (m.memory_used / m.memory_total) * 100 : 0,
        diskPercent:
          m.disk_total > 0 ? (m.disk_used / m.disk_total) * 100 : 0,
        netIn: m.net_in,
        netOut: m.net_out,
      })),
    [metrics]
  );

  if (chartData.length === 0) {
    return (
      <Card>
        <CardContent className="flex h-[300px] items-center justify-center">
          <p className="text-sm text-muted-foreground">
            Keine Metriken-Daten verfuegbar.
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="grid gap-4 lg:grid-cols-2">
      <Card>
        <CardHeader>
          <CardTitle className="text-base">CPU-Auslastung</CardTitle>
        </CardHeader>
        <CardContent>
          <ResponsiveContainer width="100%" height={200}>
            <AreaChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
              <XAxis dataKey="time" tick={{ fontSize: 10 }} />
              <YAxis
                domain={[0, 100]}
                tick={{ fontSize: 10 }}
                tickFormatter={(v) => `${v}%`}
              />
              <Tooltip
                formatter={(v: number) => `${v.toFixed(1)}%`}
                contentStyle={tooltipStyle}
              />
              <Legend
                verticalAlign="top"
                height={24}
                iconSize={10}
                wrapperStyle={{ fontSize: '12px' }}
              />
              <Area
                type="monotone"
                dataKey="cpu"
                stroke="hsl(25, 95%, 53%)"
                fill="hsl(25, 95%, 53%)"
                fillOpacity={0.1}
                strokeWidth={2}
                name="CPU"
              />
            </AreaChart>
          </ResponsiveContainer>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">RAM-Auslastung</CardTitle>
        </CardHeader>
        <CardContent>
          <ResponsiveContainer width="100%" height={200}>
            <AreaChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
              <XAxis dataKey="time" tick={{ fontSize: 10 }} />
              <YAxis
                domain={[0, 100]}
                tick={{ fontSize: 10 }}
                tickFormatter={(v) => `${v}%`}
              />
              <Tooltip
                formatter={(v: number) => `${v.toFixed(1)}%`}
                contentStyle={tooltipStyle}
              />
              <Legend
                verticalAlign="top"
                height={24}
                iconSize={10}
                wrapperStyle={{ fontSize: '12px' }}
              />
              <Area
                type="monotone"
                dataKey="memPercent"
                stroke="hsl(45, 93%, 47%)"
                fill="hsl(45, 93%, 47%)"
                fillOpacity={0.1}
                strokeWidth={2}
                name="RAM"
              />
            </AreaChart>
          </ResponsiveContainer>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Disk-Auslastung</CardTitle>
        </CardHeader>
        <CardContent>
          <ResponsiveContainer width="100%" height={200}>
            <AreaChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
              <XAxis dataKey="time" tick={{ fontSize: 10 }} />
              <YAxis
                domain={[0, 100]}
                tick={{ fontSize: 10 }}
                tickFormatter={(v) => `${v}%`}
              />
              <Tooltip
                formatter={(v: number) => `${v.toFixed(1)}%`}
                contentStyle={tooltipStyle}
              />
              <Legend
                verticalAlign="top"
                height={24}
                iconSize={10}
                wrapperStyle={{ fontSize: '12px' }}
              />
              <Area
                type="monotone"
                dataKey="diskPercent"
                stroke="hsl(200, 80%, 50%)"
                fill="hsl(200, 80%, 50%)"
                fillOpacity={0.1}
                strokeWidth={2}
                name="Disk"
              />
            </AreaChart>
          </ResponsiveContainer>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Netzwerk I/O (Rate)</CardTitle>
        </CardHeader>
        <CardContent>
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
              <XAxis dataKey="time" tick={{ fontSize: 10 }} />
              <YAxis tick={{ fontSize: 10 }} tickFormatter={(v) => formatBandwidth(v)} />
              <Tooltip
                formatter={(v: number, name: string) => {
                  const label = name === "Eingehend" ? "Eingehend" : "Ausgehend";
                  return [formatBandwidth(v), label];
                }}
                contentStyle={tooltipStyle}
              />
              <Legend
                verticalAlign="top"
                height={24}
                iconSize={10}
                wrapperStyle={{ fontSize: '12px' }}
              />
              <Line
                type="monotone"
                dataKey="netIn"
                stroke="hsl(210, 80%, 55%)"
                strokeWidth={2}
                dot={false}
                name="Eingehend"
              />
              <Line
                type="monotone"
                dataKey="netOut"
                stroke="hsl(142, 71%, 45%)"
                strokeWidth={2}
                dot={false}
                name="Ausgehend"
              />
            </LineChart>
          </ResponsiveContainer>
        </CardContent>
      </Card>
    </div>
  );
}
