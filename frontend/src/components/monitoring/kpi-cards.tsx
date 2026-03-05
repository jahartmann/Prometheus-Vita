"use client";

import { useMemo } from "react";
import { Cpu, MemoryStick, HardDrive, Network } from "lucide-react";
import {
  AreaChart,
  Area,
  ResponsiveContainer,
} from "recharts";
import { Card, CardContent } from "@/components/ui/card";
import { cn, formatPercentage, formatBandwidth } from "@/lib/utils";

interface KPICardsProps {
  cpuUsage: number;
  memoryUsage: number;
  diskUsage: number;
  netIn: number;
  netOut: number;
  history: Array<{
    cpu: number;
    mem: number;
    disk: number;
    net: number;
  }>;
}

interface SingleKPIProps {
  label: string;
  value: string;
  icon: React.ElementType;
  color: string;
  sparkColor: string;
  sparkData: Array<{ v: number }>;
  trend: "up" | "down" | "neutral";
}

function SingleKPI({ label, value, icon: Icon, color, sparkColor, sparkData, trend }: SingleKPIProps) {
  const trendIcon = trend === "up" ? "\u2191" : trend === "down" ? "\u2193" : "\u2192";
  const trendColorClass =
    trend === "up" ? "text-red-500" : trend === "down" ? "text-green-500" : "text-muted-foreground";

  return (
    <Card hover>
      <CardContent className="p-5">
        <div className="flex items-center justify-between">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <Icon className={cn("h-4 w-4", color)} />
              <p className="text-sm font-medium text-muted-foreground">{label}</p>
            </div>
            <p className="text-2xl font-bold tracking-tight">{value}</p>
            <span className={cn("text-xs font-medium", trendColorClass)}>
              {trendIcon} Trend
            </span>
          </div>
          <div className="h-12 w-24">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={sparkData}>
                <Area
                  type="monotone"
                  dataKey="v"
                  stroke={sparkColor}
                  fill={sparkColor}
                  fillOpacity={0.15}
                  strokeWidth={1.5}
                  dot={false}
                  isAnimationActive={false}
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function calculateTrend(data: number[]): "up" | "down" | "neutral" {
  if (data.length < 4) return "neutral";
  const half = Math.floor(data.length / 2);
  const firstHalf = data.slice(0, half);
  const secondHalf = data.slice(half);
  const avgFirst = firstHalf.reduce((a, b) => a + b, 0) / firstHalf.length;
  const avgSecond = secondHalf.reduce((a, b) => a + b, 0) / secondHalf.length;
  const diff = avgSecond - avgFirst;
  if (Math.abs(diff) < 2) return "neutral";
  return diff > 0 ? "up" : "down";
}

export function KPICards({ cpuUsage, memoryUsage, diskUsage, netIn, netOut, history }: KPICardsProps) {
  const cpuSpark = useMemo(() => history.map((h) => ({ v: h.cpu })), [history]);
  const memSpark = useMemo(() => history.map((h) => ({ v: h.mem })), [history]);
  const diskSpark = useMemo(() => history.map((h) => ({ v: h.disk })), [history]);
  const netSpark = useMemo(() => history.map((h) => ({ v: h.net })), [history]);

  const cpuTrend = useMemo(() => calculateTrend(history.map((h) => h.cpu)), [history]);
  const memTrend = useMemo(() => calculateTrend(history.map((h) => h.mem)), [history]);
  const diskTrend = useMemo(() => calculateTrend(history.map((h) => h.disk)), [history]);
  const netTrend = useMemo(() => calculateTrend(history.map((h) => h.net)), [history]);

  const netRate = netIn + netOut;

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <SingleKPI
        label="CPU"
        value={formatPercentage(cpuUsage)}
        icon={Cpu}
        color="text-blue-500"
        sparkColor="hsl(210, 80%, 55%)"
        sparkData={cpuSpark}
        trend={cpuTrend}
      />
      <SingleKPI
        label="RAM"
        value={formatPercentage(memoryUsage)}
        icon={MemoryStick}
        color="text-amber-500"
        sparkColor="hsl(45, 93%, 47%)"
        sparkData={memSpark}
        trend={memTrend}
      />
      <SingleKPI
        label="Disk"
        value={formatPercentage(diskUsage)}
        icon={HardDrive}
        color="text-purple-500"
        sparkColor="hsl(280, 65%, 55%)"
        sparkData={diskSpark}
        trend={diskTrend}
      />
      <SingleKPI
        label="Netzwerk"
        value={formatBandwidth(netRate)}
        icon={Network}
        color="text-green-500"
        sparkColor="hsl(142, 71%, 45%)"
        sparkData={netSpark}
        trend={netTrend}
      />
    </div>
  );
}
