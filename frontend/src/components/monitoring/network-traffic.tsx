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
} from "recharts";
import { ArrowDownToLine, ArrowUpFromLine, Activity, Clock } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { metricsApi, toArray } from "@/lib/api";
import { formatBandwidth, formatTraffic, bytesToMbits } from "@/lib/utils";
import type { MetricsRecord, NetworkSummary } from "@/types/api";

const periods = [
  { label: "1h", value: "1h" },
  { label: "24h", value: "24h" },
  { label: "7d", value: "7d" },
  { label: "30d", value: "30d" },
  { label: "Alle", value: "all" },
];

interface NetworkTrafficProps {
  nodeId: string;
}

export function NetworkTraffic({ nodeId }: NetworkTrafficProps) {
  const [period, setPeriod] = useState("24h");
  const [metrics, setMetrics] = useState<MetricsRecord[]>([]);
  const [summary, setSummary] = useState<NetworkSummary | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!nodeId) return;
    setLoading(true);

    const hours =
      period === "all" ? 8760 :
      period === "30d" ? 720 :
      period === "7d" ? 168 :
      period === "24h" ? 24 : 1;
    const since = new Date();
    since.setHours(since.getHours() - hours);

    Promise.all([
      metricsApi
        .getHistory(nodeId, since.toISOString(), new Date().toISOString())
        .then((res) => setMetrics(toArray<MetricsRecord>(res.data)))
        .catch(() => setMetrics([])),
      metricsApi
        .getNodeNetworkSummary(nodeId, period)
        .then((res) => {
          const d = res.data;
          setSummary(d && typeof d === "object" && "total_in" in d ? d : null);
        })
        .catch(() => setSummary(null)),
    ]).finally(() => setLoading(false));
  }, [nodeId, period]);

  const chartData = useMemo(() => {
    // Choose appropriate time format based on period
    const isLongPeriod = period === "7d" || period === "30d" || period === "all";
    return metrics.map((m) => ({
      time: new Date(m.recorded_at).toLocaleString("de-DE", {
        ...(isLongPeriod
          ? { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" }
          : { hour: "2-digit", minute: "2-digit" }),
      }),
      netIn: bytesToMbits(m.net_in),
      netOut: bytesToMbits(m.net_out),
      rawNetIn: m.net_in,
      rawNetOut: m.net_out,
    }));
  }, [metrics, period]);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold">Netzwerk-Traffic</h3>
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

      {summary && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <Card>
            <CardContent className="flex items-center gap-3 p-4">
              <ArrowDownToLine className="h-5 w-5 text-blue-500" />
              <div>
                <p className="text-xs text-muted-foreground">Eingehend (Gesamt)</p>
                <p className="text-lg font-bold">{formatTraffic(summary.total_in)}</p>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="flex items-center gap-3 p-4">
              <ArrowUpFromLine className="h-5 w-5 text-green-500" />
              <div>
                <p className="text-xs text-muted-foreground">Ausgehend (Gesamt)</p>
                <p className="text-lg font-bold">{formatTraffic(summary.total_out)}</p>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="flex items-center gap-3 p-4">
              <Activity className="h-5 w-5 text-amber-500" />
              <div>
                <p className="text-xs text-muted-foreground">Durchschnitt</p>
                <p className="text-sm font-bold text-blue-500">
                  {formatBandwidth(summary.avg_in_rate)}
                </p>
                <p className="text-sm font-bold text-green-500">
                  {formatBandwidth(summary.avg_out_rate)}
                </p>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="flex items-center gap-3 p-4">
              <Clock className="h-5 w-5 text-red-500" />
              <div>
                <p className="text-xs text-muted-foreground">Spitze</p>
                <p className="text-sm font-bold text-blue-500">
                  {formatBandwidth(summary.peak_in_rate)}
                </p>
                <p className="text-sm font-bold text-green-500">
                  {formatBandwidth(summary.peak_out_rate)}
                </p>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Netzwerk-Durchsatz</CardTitle>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="flex h-[300px] items-center justify-center text-sm text-muted-foreground">
              Lade Daten...
            </div>
          ) : chartData.length === 0 ? (
            <div className="flex h-[300px] items-center justify-center text-sm text-muted-foreground">
              Keine Netzwerk-Daten verfuegbar.
            </div>
          ) : (
            <ResponsiveContainer width="100%" height={300}>
              <AreaChart data={chartData}>
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
    </div>
  );
}
