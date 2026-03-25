"use client";

import { useEffect, useState, useCallback, useRef } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ArrowDown, ArrowUp, Activity, RefreshCw } from "lucide-react";
import {
  AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
} from "recharts";
import { metricsApi } from "@/lib/api";
import type { RRDDataPoint } from "@/types/api";

function formatBandwidth(bytesPerSec: number): string {
  if (bytesPerSec === 0) return "0 B/s";
  const abs = Math.abs(bytesPerSec);
  const k = 1024;
  const sizes = ["B/s", "KB/s", "MB/s", "GB/s"];
  const i = Math.min(Math.floor(Math.log(abs) / Math.log(k)), sizes.length - 1);
  return `${(bytesPerSec / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
}

function formatTime(epoch: number): string {
  return new Date(epoch * 1000).toLocaleTimeString("de-DE", {
    hour: "2-digit", minute: "2-digit",
  });
}

interface RRDBandwidthProps {
  nodeId: string;
}

export function RRDBandwidth({ nodeId }: RRDBandwidthProps) {
  const [data, setData] = useState<RRDDataPoint[]>([]);
  const [timeframe, setTimeframe] = useState<string>("hour");
  const [loading, setLoading] = useState(false);
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null);
  const lastFetchRef = useRef(0);

  const fetchData = useCallback(async () => {
    if (!nodeId) return;
    const now = Date.now();
    if (now - lastFetchRef.current < 5000) return; // 5s cooldown
    lastFetchRef.current = now;
    setLoading(true);
    try {
      const res = await metricsApi.getNodeRRD(nodeId, timeframe);
      const points = Array.isArray(res.data) ? res.data : [];
      // Filter out points with no data
      setData(points.filter((p: RRDDataPoint) => p.time > 0));
      setLastUpdate(new Date());
    } catch {
      setData([]);
    } finally {
      setLoading(false);
    }
  }, [nodeId, timeframe]);

  // Initial fetch + auto-refresh every 30s
  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 30_000);
    return () => clearInterval(interval);
  }, [fetchData]);

  const chartData = data.map((p) => ({
    time: formatTime(p.time),
    netIn: p.net_in,
    netOut: p.net_out,
  }));

  const currentIn = data.length > 0 ? data[data.length - 1].net_in : 0;
  const currentOut = data.length > 0 ? data[data.length - 1].net_out : 0;
  const peakIn = data.length > 0 ? Math.max(...data.map(d => d.net_in)) : 0;
  const peakOut = data.length > 0 ? Math.max(...data.map(d => d.net_out)) : 0;

  const timeframes = [
    { label: "1h", value: "hour" },
    { label: "24h", value: "day" },
    { label: "7d", value: "week" },
    { label: "30d", value: "month" },
  ];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold">Bandbreite (Proxmox RRD)</h3>
        <div className="flex items-center gap-2">
          <div className="flex flex-wrap gap-1">
            {timeframes.map((tf) => (
              <Button
                key={tf.value}
                variant={timeframe === tf.value ? "default" : "outline"}
                size="sm"
                onClick={() => setTimeframe(tf.value)}
              >
                {tf.label}
              </Button>
            ))}
          </div>
          <Button variant="ghost" size="icon" onClick={fetchData} disabled={loading}>
            <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          </Button>
          {lastUpdate && (
            <Badge variant="outline" className="text-xs">
              <Activity className="mr-1 h-3 w-3" />
              {lastUpdate.toLocaleTimeString("de-DE", { hour: "2-digit", minute: "2-digit", second: "2-digit" })}
            </Badge>
          )}
        </div>
      </div>

      {/* Current + Peak Stats */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardContent className="flex items-center gap-3 p-4">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-green-500/10">
              <ArrowDown className="h-5 w-5 text-green-500" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Eingehend (aktuell)</p>
              <p className="text-lg font-bold">{formatBandwidth(currentIn)}</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-3 p-4">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-500/10">
              <ArrowUp className="h-5 w-5 text-blue-500" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Ausgehend (aktuell)</p>
              <p className="text-lg font-bold">{formatBandwidth(currentOut)}</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-3 p-4">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-green-500/5">
              <ArrowDown className="h-5 w-5 text-green-400" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Spitze Eingehend</p>
              <p className="text-lg font-bold">{formatBandwidth(peakIn)}</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-3 p-4">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-500/5">
              <ArrowUp className="h-5 w-5 text-blue-400" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Spitze Ausgehend</p>
              <p className="text-lg font-bold">{formatBandwidth(peakOut)}</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Chart */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Netzwerk-Durchsatz (RRD)</CardTitle>
        </CardHeader>
        <CardContent>
          {loading && data.length === 0 ? (
            <div className="flex h-[300px] items-center justify-center text-sm text-muted-foreground">
              Lade RRD-Daten...
            </div>
          ) : chartData.length === 0 ? (
            <div className="flex h-[300px] items-center justify-center text-sm text-muted-foreground">
              Keine RRD-Daten verfügbar
            </div>
          ) : (
            <ResponsiveContainer width="100%" height={250}>
              <AreaChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                <XAxis dataKey="time" tick={{ fontSize: 10 }} />
                <YAxis tick={{ fontSize: 10 }} tickFormatter={(v) => formatBandwidth(v)} width={80} />
                <Tooltip
                  formatter={(value: number, name: string) => {
                    const label = name === "netIn" ? "Eingehend" : "Ausgehend";
                    return [formatBandwidth(value), label];
                  }}
                  contentStyle={{
                    backgroundColor: "hsl(var(--card))",
                    border: "1px solid hsl(var(--border))",
                    borderRadius: "0.5rem",
                  }}
                />
                <Area type="monotone" dataKey="netIn" stroke="hsl(142, 71%, 45%)" fill="hsl(142, 71%, 45%)" fillOpacity={0.15} strokeWidth={2} name="netIn" />
                <Area type="monotone" dataKey="netOut" stroke="hsl(217, 91%, 60%)" fill="hsl(217, 91%, 60%)" fillOpacity={0.15} strokeWidth={2} name="netOut" />
              </AreaChart>
            </ResponsiveContainer>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
