"use client";

import { useEffect, useState, useCallback } from "react";
import {
  RefreshCw,
  HeartPulse,
  ShieldCheck,
  AlertTriangle,
  AlertCircle,
  Cpu,
  MemoryStick,
  HardDrive,
  Activity,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { KpiCard } from "@/components/ui/kpi-card";
import { vmHealthApi, toArray } from "@/lib/api";
import { useNodeStore } from "@/stores/node-store";
import type { VMHealthScore } from "@/types/api";

function getScoreColor(score: number): string {
  if (score > 80) return "text-green-500";
  if (score > 50) return "text-yellow-500";
  return "text-red-500";
}

function getScoreBg(score: number): string {
  if (score > 80) return "bg-green-500";
  if (score > 50) return "bg-yellow-500";
  return "bg-red-500";
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

export default function HealthPage() {
  const [allScores, setAllScores] = useState<VMHealthScore[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const { nodes, fetchNodes } = useNodeStore();

  const fetchAllHealth = useCallback(async () => {
    setIsLoading(true);
    try {
      const scores: VMHealthScore[] = [];
      for (const node of nodes) {
        try {
          const res = await vmHealthApi.getAllHealth(node.id);
          const nodeScores = toArray<VMHealthScore>(res.data);
          scores.push(...nodeScores);
        } catch {
          // skip offline nodes
        }
      }
      // Sort by score ascending (worst first)
      scores.sort((a, b) => a.score - b.score);
      setAllScores(scores);
    } catch {
      // ignore
    } finally {
      setIsLoading(false);
    }
  }, [nodes]);

  useEffect(() => {
    fetchNodes();
  }, [fetchNodes]);

  useEffect(() => {
    if (nodes.length > 0) {
      fetchAllHealth();
      // Auto-refresh every 60 seconds
      const interval = setInterval(fetchAllHealth, 60000);
      return () => clearInterval(interval);
    }
  }, [nodes, fetchAllHealth]);

  const getNodeName = (nodeId: string) => {
    const node = nodes.find((n) => n.id === nodeId);
    return node?.name || nodeId.slice(0, 8);
  };

  const totalVMs = allScores.length;
  const healthyCount = allScores.filter((s) => s.status === "healthy").length;
  const warningCount = allScores.filter((s) => s.status === "warning").length;
  const criticalCount = allScores.filter((s) => s.status === "critical").length;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-bold">VM-Gesundheit</h2>
          <p className="text-sm text-muted-foreground">
            Übersicht aller VMs sortiert nach Gesundheitsbewertung.
          </p>
        </div>
        <Button variant="outline" onClick={fetchAllHealth} disabled={isLoading}>
          <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? "animate-spin" : ""}`} />
          Aktualisieren
        </Button>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <KpiCard
          title="VMs gesamt"
          value={totalVMs}
          subtitle="analysierte VMs"
          icon={HeartPulse}
          color="blue"
        />
        <KpiCard
          title="Gesund"
          value={healthyCount}
          subtitle="Score > 80"
          icon={ShieldCheck}
          color="green"
        />
        <KpiCard
          title="Warnung"
          value={warningCount}
          subtitle="Score 51-80"
          icon={AlertTriangle}
          color="orange"
        />
        <KpiCard
          title="Kritisch"
          value={criticalCount}
          subtitle="Score <= 50"
          icon={AlertCircle}
          color="red"
        />
      </div>

      {/* VM Health List */}
      {isLoading && allScores.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center text-muted-foreground">
            Gesundheitsbewertungen werden berechnet...
          </CardContent>
        </Card>
      ) : allScores.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center text-muted-foreground">
            Keine VMs gefunden. Stellen Sie sicher, dass Server verbunden und VMs aktiv sind.
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {allScores.map((score) => (
            <Card
              key={`${score.node_id}-${score.vmid}`}
              className={
                score.status === "critical"
                  ? "border-l-4 border-l-red-500"
                  : score.status === "warning"
                    ? "border-l-4 border-l-yellow-500"
                    : ""
              }
            >
              <CardContent className="p-4">
                <div className="flex items-center gap-4">
                  {/* Score badge */}
                  <div
                    className={`flex h-12 w-12 items-center justify-center rounded-full ${getScoreBg(score.score)} text-white font-bold text-lg`}
                  >
                    {score.score}
                  </div>

                  {/* VM Info */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="font-medium">
                        {score.vm_name || `VM ${score.vmid}`}
                      </span>
                      <Badge variant="outline" className="text-xs">
                        {score.vm_type?.toUpperCase()} {score.vmid}
                      </Badge>
                      {getStatusBadge(score.status)}
                    </div>
                    <p className="text-xs text-muted-foreground mt-0.5">
                      {getNodeName(score.node_id)}
                    </p>
                  </div>

                  {/* Breakdown bars */}
                  <div className="hidden md:flex items-center gap-4">
                    <BreakdownItem
                      icon={Cpu}
                      label="CPU"
                      score={score.breakdown.cpu_score}
                      max={25}
                      detail={`${score.breakdown.cpu_avg.toFixed(0)}%`}
                    />
                    <BreakdownItem
                      icon={MemoryStick}
                      label="RAM"
                      score={score.breakdown.ram_score}
                      max={25}
                      detail={`${score.breakdown.ram_avg.toFixed(0)}%`}
                    />
                    <BreakdownItem
                      icon={HardDrive}
                      label="Disk"
                      score={score.breakdown.disk_score}
                      max={25}
                      detail={`${score.breakdown.disk_usage.toFixed(0)}%`}
                    />
                    <BreakdownItem
                      icon={Activity}
                      label="Stabil"
                      score={score.breakdown.stability_score}
                      max={25}
                      detail={`${score.breakdown.uptime_days.toFixed(0)}d`}
                    />
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

function BreakdownItem({
  icon: Icon,
  label,
  score,
  max,
  detail,
}: {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  score: number;
  max: number;
  detail: string;
}) {
  const pct = (score / max) * 100;
  return (
    <div className="w-20 space-y-0.5">
      <div className="flex items-center gap-1 text-[10px] text-muted-foreground">
        <Icon className="h-3 w-3" />
        <span>{label}</span>
        <span className="ml-auto font-medium text-foreground">{detail}</span>
      </div>
      <div className="h-1 w-full rounded-full bg-muted overflow-hidden">
        <div
          className={`h-full rounded-full ${
            pct >= 80 ? "bg-green-500" : pct >= 50 ? "bg-yellow-500" : "bg-red-500"
          }`}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
}
