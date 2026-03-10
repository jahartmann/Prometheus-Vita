"use client";

import { useState } from "react";
import { useNodeStore } from "@/stores/node-store";
import { useNodeMetrics } from "@/hooks/use-node-metrics";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { ArrowDown, ArrowUp, Activity } from "lucide-react";
import {
  AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend,
} from "recharts";

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B/s";
  const k = 1024;
  const sizes = ["B/s", "KB/s", "MB/s", "GB/s"];
  const i = Math.floor(Math.log(Math.abs(bytes)) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
}

function formatTime(ts: string): string {
  return new Date(ts).toLocaleTimeString("de-DE", { hour: "2-digit", minute: "2-digit", second: "2-digit" });
}

export function LiveTraffic() {
  const { nodes } = useNodeStore();
  const [selectedNode, setSelectedNode] = useState<string>(nodes[0]?.id || "");
  const { metrics } = useNodeMetrics(selectedNode);

  // network_in/out are already per-second rates from the backend
  const trafficData = metrics.map((m) => ({
    time: formatTime(m.timestamp),
    inRate: m.network_in,
    outRate: m.network_out,
  }));

  const currentIn = trafficData.length > 0 ? trafficData[trafficData.length - 1].inRate : 0;
  const currentOut = trafficData.length > 0 ? trafficData[trafficData.length - 1].outRate : 0;

  return (
    <div className="space-y-4">
      {/* Node selector */}
      <div className="flex items-center gap-3">
        <Select value={selectedNode} onValueChange={setSelectedNode}>
          <SelectTrigger className="w-64">
            <SelectValue placeholder="Node waehlen" />
          </SelectTrigger>
          <SelectContent>
            {nodes.map((n) => (
              <SelectItem key={n.id} value={n.id}>{n.name}</SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Badge variant="outline" className="gap-1">
          <Activity className="h-3 w-3" />
          Live
        </Badge>
      </div>

      {/* Current bandwidth cards */}
      <div className="grid grid-cols-2 gap-4">
        <Card>
          <CardContent className="flex items-center gap-3 py-4">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-green-500/10">
              <ArrowDown className="h-5 w-5 text-green-500" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Eingehend</p>
              <p className="text-lg font-bold">{formatBytes(currentIn)}</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-3 py-4">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-500/10">
              <ArrowUp className="h-5 w-5 text-blue-500" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Ausgehend</p>
              <p className="text-lg font-bold">{formatBytes(currentOut)}</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Network traffic chart */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Netzwerk-Durchsatz</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="h-[300px]">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={trafficData}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                <XAxis dataKey="time" className="text-xs" tick={{ fill: "hsl(var(--muted-foreground))" }} />
                <YAxis tickFormatter={(v) => formatBytes(v)} className="text-xs" tick={{ fill: "hsl(var(--muted-foreground))" }} width={80} />
                <Tooltip
                  formatter={(value: number) => formatBytes(value)}
                  contentStyle={{ backgroundColor: "hsl(var(--popover))", border: "1px solid hsl(var(--border))", borderRadius: "8px" }}
                  labelStyle={{ color: "hsl(var(--popover-foreground))" }}
                />
                <Legend />
                <Area type="monotone" dataKey="inRate" name="Eingehend" stroke="hsl(142, 71%, 45%)" fill="hsl(142, 71%, 45%)" fillOpacity={0.15} strokeWidth={2} />
                <Area type="monotone" dataKey="outRate" name="Ausgehend" stroke="hsl(217, 91%, 60%)" fill="hsl(217, 91%, 60%)" fillOpacity={0.15} strokeWidth={2} />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
