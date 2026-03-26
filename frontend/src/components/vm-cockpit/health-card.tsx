"use client";

import { useEffect, useState, useCallback } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Cpu, MemoryStick, HardDrive, ShieldCheck, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { vmHealthApi } from "@/lib/api";
import type { VMHealthScore } from "@/types/api";

interface HealthCardProps {
  nodeId: string;
  vmid: number;
}

function getScoreColor(score: number): string {
  if (score > 80) return "text-green-500";
  if (score > 50) return "text-yellow-500";
  return "text-red-500";
}

function getScoreRingColor(score: number): string {
  if (score > 80) return "stroke-green-500";
  if (score > 50) return "stroke-yellow-500";
  return "stroke-red-500";
}

function getStatusBadge(status: string) {
  switch (status) {
    case "healthy":
      return <Badge variant="success">Gesund</Badge>;
    case "warning":
      return <Badge variant="warning">Warnung</Badge>;
    case "critical":
      return <Badge variant="destructive">Kritisch</Badge>;
    default:
      return <Badge variant="secondary">{status}</Badge>;
  }
}

function CircularProgress({ score }: { score: number }) {
  const radius = 40;
  const circumference = 2 * Math.PI * radius;
  const offset = circumference - (score / 100) * circumference;

  return (
    <div className="relative flex items-center justify-center">
      <svg width="100" height="100" viewBox="0 0 100 100" className="-rotate-90">
        <circle
          cx="50"
          cy="50"
          r={radius}
          fill="none"
          strokeWidth="8"
          className="stroke-muted"
        />
        <circle
          cx="50"
          cy="50"
          r={radius}
          fill="none"
          strokeWidth="8"
          strokeLinecap="round"
          strokeDasharray={circumference}
          strokeDashoffset={offset}
          className={`transition-all duration-700 ${getScoreRingColor(score)}`}
        />
      </svg>
      <span className={`absolute text-2xl font-bold ${getScoreColor(score)}`}>
        {score}
      </span>
    </div>
  );
}

export function HealthCard({ nodeId, vmid }: HealthCardProps) {
  const [health, setHealth] = useState<VMHealthScore | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchHealth = useCallback(async () => {
    if (!nodeId || !vmid) return;
    setLoading(true);
    try {
      const res = await vmHealthApi.getHealth(nodeId, vmid);
      setHealth(res.data as VMHealthScore);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [nodeId, vmid]);

  useEffect(() => {
    fetchHealth();
  }, [fetchHealth]);

  if (!health && !loading) {
    return null;
  }

  const breakdownItems = health
    ? [
        {
          label: "CPU",
          icon: Cpu,
          score: health.breakdown.cpu_score,
          detail: `Durchschnitt: ${health.breakdown.cpu_avg.toFixed(1)}%`,
          max: 25,
        },
        {
          label: "RAM",
          icon: MemoryStick,
          score: health.breakdown.ram_score,
          detail: `Durchschnitt: ${health.breakdown.ram_avg.toFixed(1)}%`,
          max: 25,
        },
        {
          label: "Disk",
          icon: HardDrive,
          score: health.breakdown.disk_score,
          detail: `Auslastung: ${health.breakdown.disk_usage.toFixed(1)}%`,
          max: 25,
        },
        {
          label: "Stabilität",
          icon: ShieldCheck,
          score: health.breakdown.stability_score,
          detail: `Uptime: ${health.breakdown.uptime_days.toFixed(1)} Tage, Ausfälle: ${health.breakdown.crash_count}`,
          max: 25,
        },
      ]
    : [];

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-base">Gesundheitsbewertung</CardTitle>
        <div className="flex items-center gap-2">
          {health && getStatusBadge(health.status)}
          <Button variant="ghost" size="icon" onClick={fetchHealth} disabled={loading}>
            <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {loading && !health ? (
          <div className="flex h-[120px] items-center justify-center text-sm text-muted-foreground">
            Berechne Gesundheitsbewertung...
          </div>
        ) : health ? (
          <div className="flex items-center gap-6">
            <CircularProgress score={health.score} />
            <TooltipProvider>
              <div className="flex-1 grid grid-cols-2 gap-3">
                {breakdownItems.map((item) => {
                  const Icon = item.icon;
                  const pct = (item.score / item.max) * 100;
                  return (
                    <Tooltip key={item.label}>
                      <TooltipTrigger asChild>
                        <div className="space-y-1 cursor-default">
                          <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                            <Icon className="h-3.5 w-3.5" />
                            <span>{item.label}</span>
                            <span className="ml-auto font-medium text-foreground">
                              {item.score}/{item.max}
                            </span>
                          </div>
                          <div className="h-1.5 w-full rounded-full bg-muted overflow-hidden">
                            <div
                              className={`h-full rounded-full transition-all ${
                                pct >= 80
                                  ? "bg-green-500"
                                  : pct >= 50
                                    ? "bg-yellow-500"
                                    : "bg-red-500"
                              }`}
                              style={{ width: `${pct}%` }}
                            />
                          </div>
                        </div>
                      </TooltipTrigger>
                      <TooltipContent>
                        <p>{item.detail}</p>
                      </TooltipContent>
                    </Tooltip>
                  );
                })}
              </div>
            </TooltipProvider>
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}
