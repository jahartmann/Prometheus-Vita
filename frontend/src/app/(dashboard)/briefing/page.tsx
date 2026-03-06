"use client";

import { useEffect, useState, useCallback } from "react";
import { briefingApi, anomalyApi, predictionApi } from "@/lib/api";
import { toArray } from "@/lib/api";
import type {
  LiveBriefingSummary,
  AnomalyRecord,
  MaintenancePrediction,
} from "@/types/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { KpiCard } from "@/components/ui/kpi-card";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  Server,
  ServerCog,
  Cpu,
  MemoryStick,
  HardDrive,
  Activity,
  AlertTriangle,
  TrendingUp,
  Clock,
  ChevronDown,
  CheckCircle2,
  XCircle,
  Zap,
  ArrowRight,
  Timer,
  Link as LinkIcon,
} from "lucide-react";
import Link from "next/link";

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  if (days > 0) return `${days}d ${hours}h`;
  const mins = Math.floor((seconds % 3600) / 60);
  return `${hours}h ${mins}m`;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
}

function getGreeting(): string {
  const hour = new Date().getHours();
  if (hour < 12) return "Guten Morgen";
  if (hour < 18) return "Guten Tag";
  return "Guten Abend";
}

function severityColor(severity: string) {
  switch (severity) {
    case "critical":
      return "text-red-500";
    case "warning":
      return "text-orange-500";
    default:
      return "text-blue-500";
  }
}

function severityBg(severity: string) {
  switch (severity) {
    case "critical":
      return "bg-red-500/10 border-red-500/30";
    case "warning":
      return "bg-orange-500/10 border-orange-500/30";
    default:
      return "bg-blue-500/10 border-blue-500/30";
  }
}

function severityBadgeVariant(severity: string) {
  switch (severity) {
    case "critical":
      return "destructive" as const;
    case "warning":
      return "warning" as const;
    default:
      return "secondary" as const;
  }
}

function metricLabel(metric: string): string {
  switch (metric) {
    case "cpu_usage":
    case "cpu":
      return "CPU";
    case "memory_usage":
    case "memory":
    case "ram":
    case "mem":
      return "RAM";
    case "disk_usage":
    case "disk":
      return "Disk";
    default:
      return metric.toUpperCase();
  }
}

function metricIcon(metric: string) {
  if (metric.includes("cpu")) return Cpu;
  if (metric.includes("mem") || metric.includes("ram")) return MemoryStick;
  if (metric.includes("disk")) return HardDrive;
  return Activity;
}

function trendText(direction?: string): string {
  switch (direction) {
    case "rising": return "steigend";
    case "falling": return "fallend";
    case "stable": return "stabil";
    default: return "";
  }
}

function trendColor(direction?: string): string {
  switch (direction) {
    case "rising": return "text-red-500";
    case "falling": return "text-green-500";
    case "stable": return "text-blue-500";
    default: return "text-muted-foreground";
  }
}

export default function BriefingPage() {
  const [data, setData] = useState<LiveBriefingSummary | null>(null);
  const [anomalies, setAnomalies] = useState<AnomalyRecord[]>([]);
  const [predictions, setPredictions] = useState<MaintenancePrediction[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [resolvingIds, setResolvingIds] = useState<Set<string>>(new Set());
  const [anomaliesOpen, setAnomaliesOpen] = useState(true);
  const [predictionsOpen, setPredictionsOpen] = useState(true);

  useEffect(() => {
    Promise.all([
      briefingApi.getLive(),
      anomalyApi.listUnresolved().catch(() => []),
      predictionApi.listCritical().catch(() => []),
    ])
      .then(([briefing, rawAnomalies, rawPredictions]) => {
        setData(briefing);
        setAnomalies(toArray<AnomalyRecord>(rawAnomalies) as AnomalyRecord[]);
        setPredictions(
          toArray<MaintenancePrediction>(rawPredictions) as MaintenancePrediction[]
        );
      })
      .catch(() => setError("Briefing konnte nicht geladen werden"))
      .finally(() => setIsLoading(false));
  }, []);

  const handleResolve = useCallback(async (id: string) => {
    setResolvingIds((prev) => new Set(prev).add(id));
    try {
      await anomalyApi.resolve(id);
      setAnomalies((prev) => prev.filter((a) => a.id !== id));
    } catch {
      // silent
    } finally {
      setResolvingIds((prev) => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
    }
  }, []);

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{getGreeting()}</h1>
          <p className="text-muted-foreground">Lade Briefing...</p>
        </div>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {[...Array(4)].map((_, i) => (
            <Card key={i}>
              <CardContent className="p-5">
                <div className="h-16 animate-pulse rounded bg-muted" />
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{getGreeting()}</h1>
          <p className="text-muted-foreground">Ihr Infrastruktur-Briefing</p>
        </div>
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">
            {error || "Keine Daten verfuegbar"}
          </CardContent>
        </Card>
      </div>
    );
  }

  const healthStatus =
    data.nodes_offline === 0 && anomalies.length === 0
      ? "healthy"
      : data.nodes_offline > 0
        ? "critical"
        : "warning";

  // Build recommended actions
  const actions: {
    icon: typeof AlertTriangle;
    title: string;
    description: string;
    href: string;
    severity: string;
  }[] = [];

  if (data.nodes_offline > 0) {
    actions.push({
      icon: XCircle,
      title: `${data.nodes_offline} Node${data.nodes_offline > 1 ? "s" : ""} offline`,
      description: "Offline-Nodes umgehend pruefen und wiederherstellen",
      href: "/monitoring",
      severity: "critical",
    });
  }
  if (anomalies.length > 0) {
    const critical = anomalies.filter((a) => a.severity === "critical").length;
    actions.push({
      icon: AlertTriangle,
      title: `${anomalies.length} Anomalie${anomalies.length > 1 ? "n" : ""} pruefen`,
      description: critical > 0
        ? `${critical} davon kritisch - sofortige Untersuchung empfohlen`
        : "Ungewoehnliche Metriken erkannt - Analyse empfohlen",
      href: "#anomalies",
      severity: critical > 0 ? "critical" : "warning",
    });
  }
  if (predictions.length > 0) {
    const minDays = Math.min(
      ...predictions.map((p) => p.days_until_threshold ?? 999)
    );
    actions.push({
      icon: Timer,
      title: `Ressourcen-Engpass in ${minDays} Tagen erwartet`,
      description: `${predictions.length} Vorhersage${predictions.length > 1 ? "n" : ""} zeigen bevorstehende Schwellwertuebrschreitungen`,
      href: "#predictions",
      severity: minDays <= 7 ? "critical" : "warning",
    });
  }
  if (data.avg_cpu > 80) {
    actions.push({
      icon: Cpu,
      title: "Cluster stark ausgelastet (CPU)",
      description: `Durchschnittliche CPU-Last bei ${data.avg_cpu.toFixed(1)}% - Entlastung pruefen`,
      href: "/monitoring",
      severity: "warning",
    });
  }
  if (data.avg_ram > 80) {
    actions.push({
      icon: MemoryStick,
      title: "Cluster stark ausgelastet (RAM)",
      description: `Durchschnittliche RAM-Nutzung bei ${data.avg_ram.toFixed(1)}% - Optimierung pruefen`,
      href: "/monitoring",
      severity: "warning",
    });
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{getGreeting()}</h1>
          <p className="text-muted-foreground">
            {new Date().toLocaleDateString("de-DE", {
              weekday: "long",
              year: "numeric",
              month: "long",
              day: "numeric",
            })}
          </p>
        </div>
        <Badge
          variant={
            healthStatus === "healthy"
              ? "success"
              : healthStatus === "critical"
                ? "destructive"
                : "warning"
          }
          className="text-sm px-3 py-1"
        >
          {healthStatus === "healthy"
            ? "Alle Systeme operativ"
            : healthStatus === "critical"
              ? "Achtung erforderlich"
              : "Warnungen vorhanden"}
        </Badge>
      </div>

      {/* KPI Cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard
          title="Server"
          value={`${data.nodes_online}/${data.nodes_total}`}
          subtitle={
            data.nodes_offline > 0
              ? `${data.nodes_offline} offline`
              : "Alle online"
          }
          icon={Server}
          color={data.nodes_offline > 0 ? "red" : "green"}
        />
        <KpiCard
          title="VMs / Container"
          value={data.vms_total}
          subtitle={`${data.vms_running} aktiv, ${data.vms_stopped} gestoppt`}
          icon={ServerCog}
          color="blue"
        />
        <KpiCard
          title="CPU-Durchschnitt"
          value={`${data.avg_cpu.toFixed(1)}%`}
          subtitle="Cluster-weit"
          icon={Cpu}
          color={data.avg_cpu > 80 ? "red" : data.avg_cpu > 60 ? "orange" : "green"}
        />
        <KpiCard
          title="RAM-Durchschnitt"
          value={`${data.avg_ram.toFixed(1)}%`}
          subtitle="Cluster-weit"
          icon={MemoryStick}
          color={data.avg_ram > 85 ? "red" : data.avg_ram > 70 ? "orange" : "green"}
        />
      </div>

      {/* Recommended Actions */}
      {actions.length > 0 && (
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-base">
              <Zap className="h-4 w-4 text-muted-foreground" />
              Empfohlene Aktionen
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid gap-3 sm:grid-cols-2">
              {actions.map((action, i) => (
                <Link key={i} href={action.href}>
                  <div
                    className={`flex items-start gap-3 rounded-lg border p-3 transition-colors hover:bg-muted/50 ${severityBg(action.severity)}`}
                  >
                    <div
                      className={`mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg ${
                        action.severity === "critical"
                          ? "bg-red-500/15"
                          : "bg-orange-500/15"
                      }`}
                    >
                      <action.icon
                        className={`h-4 w-4 ${severityColor(action.severity)}`}
                      />
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="text-sm font-medium">{action.title}</p>
                      <p className="text-xs text-muted-foreground">
                        {action.description}
                      </p>
                    </div>
                    <ArrowRight className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
                  </div>
                </Link>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Anomalies Section */}
      <Collapsible open={anomaliesOpen} onOpenChange={setAnomaliesOpen}>
        <Card id="anomalies">
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center gap-2 text-base">
                <AlertTriangle className="h-4 w-4 text-muted-foreground" />
                Anomalien
                {anomalies.length > 0 && (
                  <Badge variant="warning" className="ml-1 text-xs">
                    {anomalies.length}
                  </Badge>
                )}
              </CardTitle>
              <CollapsibleTrigger asChild>
                <Button variant="ghost" size="sm">
                  <ChevronDown
                    className={`h-4 w-4 transition-transform ${anomaliesOpen ? "rotate-180" : ""}`}
                  />
                </Button>
              </CollapsibleTrigger>
            </div>
          </CardHeader>
          <CollapsibleContent>
            <CardContent>
              {anomalies.length === 0 ? (
                <div className="flex items-center gap-2 text-sm text-muted-foreground py-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  Keine Anomalien erkannt — alle Metriken im Normalbereich
                </div>
              ) : (
                <div className="space-y-3">
                  {anomalies.map((a) => {
                    const MetricIcon = metricIcon(a.metric);
                    return (
                      <div
                        key={a.id}
                        className={`rounded-lg border p-4 ${severityBg(a.severity)}`}
                      >
                        {/* Header */}
                        <div className="flex items-start justify-between mb-2">
                          <div className="flex items-center gap-2">
                            <div className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-lg ${
                              a.severity === "critical" ? "bg-red-500/15" : "bg-orange-500/15"
                            }`}>
                              <MetricIcon className={`h-4 w-4 ${severityColor(a.severity)}`} />
                            </div>
                            <div>
                              <div className="flex items-center gap-2">
                                <span className="font-semibold text-sm">
                                  {metricLabel(a.metric)} — {a.node_name || a.node_id.slice(0, 8)}
                                </span>
                                <Badge variant={severityBadgeVariant(a.severity)} className="text-xs">
                                  {a.severity === "critical" ? "Kritisch" : "Warnung"}
                                </Badge>
                              </div>
                              <p className="text-xs text-muted-foreground">
                                {new Date(a.detected_at).toLocaleString("de-DE", {
                                  day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit",
                                })}
                                {" · "}Aktuell: <span className="font-mono font-medium">{a.value.toFixed(1)}%</span>
                                {" · "}Durchschnitt: <span className="font-mono">{a.mean.toFixed(1)}%</span>
                              </p>
                            </div>
                          </div>
                          <div className="flex items-center gap-2">
                            <Link
                              href={`/nodes/${a.node_id}/monitoring`}
                              className="text-xs text-primary hover:underline inline-flex items-center gap-1"
                            >
                              Monitoring <LinkIcon className="h-3 w-3" />
                            </Link>
                            <Button
                              variant="outline"
                              size="sm"
                              disabled={resolvingIds.has(a.id)}
                              onClick={() => handleResolve(a.id)}
                              className="h-7 text-xs"
                            >
                              {resolvingIds.has(a.id) ? "..." : "Aufloesen"}
                            </Button>
                          </div>
                        </div>

                        {/* Description */}
                        {a.description && (
                          <p className="text-sm mb-2">{a.description}</p>
                        )}

                        {/* Impact + Recommendation */}
                        <div className="grid gap-2 sm:grid-cols-2">
                          {a.impact && (
                            <div className="rounded bg-background/50 p-2.5 text-xs">
                              <p className="font-medium text-muted-foreground mb-0.5">Auswirkung</p>
                              <p>{a.impact}</p>
                            </div>
                          )}
                          {a.recommendation && (
                            <div className="rounded bg-background/50 p-2.5 text-xs">
                              <p className="font-medium text-muted-foreground mb-0.5">Empfehlung</p>
                              <p>{a.recommendation}</p>
                            </div>
                          )}
                        </div>

                        {/* Affected VMs */}
                        {a.affected_vms && a.affected_vms.length > 0 && (
                          <div className="mt-2 flex items-center gap-1.5 flex-wrap">
                            <span className="text-xs text-muted-foreground">Betroffene VMs:</span>
                            {a.affected_vms.slice(0, 8).map((vm) => (
                              <Badge key={vm} variant="secondary" className="text-xs py-0">
                                {vm}
                              </Badge>
                            ))}
                            {a.affected_vms.length > 8 && (
                              <span className="text-xs text-muted-foreground">
                                +{a.affected_vms.length - 8} weitere
                              </span>
                            )}
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              )}
            </CardContent>
          </CollapsibleContent>
        </Card>
      </Collapsible>

      {/* Predictions Section */}
      <Collapsible open={predictionsOpen} onOpenChange={setPredictionsOpen}>
        <Card id="predictions">
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center gap-2 text-base">
                <Activity className="h-4 w-4 text-muted-foreground" />
                Vorhersagen
                {predictions.length > 0 && (
                  <Badge variant="destructive" className="ml-1 text-xs">
                    {predictions.length}
                  </Badge>
                )}
              </CardTitle>
              <CollapsibleTrigger asChild>
                <Button variant="ghost" size="sm">
                  <ChevronDown
                    className={`h-4 w-4 transition-transform ${predictionsOpen ? "rotate-180" : ""}`}
                  />
                </Button>
              </CollapsibleTrigger>
            </div>
          </CardHeader>
          <CollapsibleContent>
            <CardContent>
              {predictions.length === 0 ? (
                <div className="flex items-center gap-2 text-sm text-muted-foreground py-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  Keine kritischen Vorhersagen — Ressourcen im gruenen Bereich
                </div>
              ) : (
                <div className="space-y-3">
                  {predictions.map((p) => {
                    const MetricIcon = metricIcon(p.metric);
                    const days = p.days_until_threshold != null ? Math.round(p.days_until_threshold) : null;
                    return (
                      <div
                        key={p.id}
                        className={`rounded-lg border p-4 ${severityBg(p.severity)}`}
                      >
                        {/* Header */}
                        <div className="flex items-start justify-between mb-2">
                          <div className="flex items-center gap-2">
                            <div className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-lg ${
                              p.severity === "critical" ? "bg-red-500/15" : "bg-orange-500/15"
                            }`}>
                              <MetricIcon className={`h-4 w-4 ${severityColor(p.severity)}`} />
                            </div>
                            <div>
                              <div className="flex items-center gap-2">
                                <span className="font-semibold text-sm">
                                  {metricLabel(p.metric)} — {p.node_name || p.node_id.slice(0, 8)}
                                </span>
                                <Badge variant={severityBadgeVariant(p.severity)} className="text-xs">
                                  {p.severity === "critical" ? "Kritisch" : "Warnung"}
                                </Badge>
                                {p.trend_direction && (
                                  <span className={`text-xs font-medium ${trendColor(p.trend_direction)}`}>
                                    Trend: {trendText(p.trend_direction)}
                                  </span>
                                )}
                              </div>
                              <p className="text-xs text-muted-foreground">
                                {p.vm_count != null && p.vm_count > 0
                                  ? `${p.vm_count} laufende VMs betroffen · `
                                  : ""}
                                Schwellwert: {p.threshold.toFixed(0)}%
                                {" · "}R² = {p.r_squared.toFixed(3)}
                              </p>
                            </div>
                          </div>
                          <Link
                            href={`/nodes/${p.node_id}/monitoring`}
                            className="text-xs text-primary hover:underline inline-flex items-center gap-1"
                          >
                            Monitoring <LinkIcon className="h-3 w-3" />
                          </Link>
                        </div>

                        {/* Key metrics row */}
                        <div className="flex items-center gap-4 mb-3">
                          <div className="flex items-center gap-2 rounded bg-background/50 px-3 py-2">
                            <span className="text-xs text-muted-foreground">Aktuell</span>
                            <span className="text-lg font-bold font-mono">{p.current_value.toFixed(1)}%</span>
                          </div>
                          <ArrowRight className="h-4 w-4 text-muted-foreground shrink-0" />
                          <div className="flex items-center gap-2 rounded bg-background/50 px-3 py-2">
                            <span className="text-xs text-muted-foreground">Prognose (30d)</span>
                            <span className={`text-lg font-bold font-mono ${severityColor(p.severity)}`}>
                              {p.predicted_value.toFixed(1)}%
                            </span>
                          </div>
                          {days != null && (
                            <div className={`ml-auto flex items-center gap-1.5 rounded px-3 py-2 ${
                              p.severity === "critical" ? "bg-red-500/10" : "bg-orange-500/10"
                            }`}>
                              <Timer className={`h-4 w-4 ${severityColor(p.severity)}`} />
                              <span className={`text-lg font-bold ${severityColor(p.severity)}`}>
                                {days}d
                              </span>
                              <span className="text-xs text-muted-foreground">bis Limit</span>
                            </div>
                          )}
                        </div>

                        {/* Description */}
                        {p.description && (
                          <p className="text-sm mb-2">{p.description}</p>
                        )}

                        {/* Recommendation */}
                        {p.recommendation && (
                          <div className="rounded bg-background/50 p-2.5 text-xs">
                            <p className="font-medium text-muted-foreground mb-0.5">Empfehlung</p>
                            <p>{p.recommendation}</p>
                          </div>
                        )}

                        {/* Affected VMs */}
                        {p.affected_vms && p.affected_vms.length > 0 && (
                          <div className="mt-2 flex items-center gap-1.5 flex-wrap">
                            <span className="text-xs text-muted-foreground">VMs auf diesem Node:</span>
                            {p.affected_vms.slice(0, 6).map((vm) => (
                              <Badge key={vm} variant="secondary" className="text-xs py-0">
                                {vm}
                              </Badge>
                            ))}
                            {p.affected_vms.length > 6 && (
                              <span className="text-xs text-muted-foreground">
                                +{p.affected_vms.length - 6} weitere
                              </span>
                            )}
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              )}
            </CardContent>
          </CollapsibleContent>
        </Card>
      </Collapsible>

      {/* Top Nodes & VMs */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* Top Nodes by CPU */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-base">
              <TrendingUp className="h-4 w-4 text-muted-foreground" />
              Top Nodes nach CPU-Last
            </CardTitle>
          </CardHeader>
          <CardContent>
            {data.top_nodes_by_cpu && data.top_nodes_by_cpu.length > 0 ? (
              <div className="space-y-3">
                {data.top_nodes_by_cpu.map((node, i) => (
                  <div key={node.node_id} className="flex items-center gap-3">
                    <span className="flex h-6 w-6 items-center justify-center rounded-full bg-muted text-xs font-bold">
                      {i + 1}
                    </span>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center justify-between">
                        <span className="font-medium truncate">{node.node_name}</span>
                        <span
                          className={`text-sm font-mono ${
                            node.cpu_usage > 80
                              ? "text-red-500"
                              : node.cpu_usage > 60
                                ? "text-orange-500"
                                : "text-green-500"
                          }`}
                        >
                          {node.cpu_usage.toFixed(1)}%
                        </span>
                      </div>
                      <div className="mt-1.5 h-1.5 w-full rounded-full bg-muted">
                        <div
                          className={`h-full rounded-full transition-all ${
                            node.cpu_usage > 80
                              ? "bg-red-500"
                              : node.cpu_usage > 60
                                ? "bg-orange-500"
                                : "bg-green-500"
                          }`}
                          style={{ width: `${Math.min(node.cpu_usage, 100)}%` }}
                        />
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">Keine Daten verfuegbar</p>
            )}
          </CardContent>
        </Card>

        {/* Top VMs by RAM */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-base">
              <MemoryStick className="h-4 w-4 text-muted-foreground" />
              Top VMs nach RAM-Nutzung
            </CardTitle>
          </CardHeader>
          <CardContent>
            {data.top_vms_by_ram && data.top_vms_by_ram.length > 0 ? (
              <div className="space-y-3">
                {data.top_vms_by_ram.map((vm, i) => (
                  <div
                    key={`${vm.node_id}-${vm.vmid}`}
                    className="flex items-center gap-3"
                  >
                    <span className="flex h-6 w-6 items-center justify-center rounded-full bg-muted text-xs font-bold">
                      {i + 1}
                    </span>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center justify-between">
                        <div className="truncate">
                          <span className="font-medium">
                            {vm.vm_name || `VM ${vm.vmid}`}
                          </span>
                          <span className="text-xs text-muted-foreground ml-2">
                            auf {vm.node_name}
                          </span>
                        </div>
                        <span className="text-sm font-mono text-muted-foreground">
                          {formatBytes(vm.mem_used)} / {formatBytes(vm.mem_total)}
                        </span>
                      </div>
                      <div className="mt-1.5 h-1.5 w-full rounded-full bg-muted">
                        <div
                          className={`h-full rounded-full transition-all ${
                            vm.mem_used_pct > 90
                              ? "bg-red-500"
                              : vm.mem_used_pct > 75
                                ? "bg-orange-500"
                                : "bg-blue-500"
                          }`}
                          style={{ width: `${Math.min(vm.mem_used_pct, 100)}%` }}
                        />
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">Keine laufenden VMs</p>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Node Details */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="flex items-center gap-2 text-base">
            <Server className="h-4 w-4 text-muted-foreground" />
            Node-Uebersicht
          </CardTitle>
        </CardHeader>
        <CardContent>
          {data.node_details && data.node_details.length > 0 ? (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b text-left text-muted-foreground">
                    <th className="pb-2 font-medium">Node</th>
                    <th className="pb-2 font-medium">Status</th>
                    <th className="pb-2 font-medium text-right">CPU</th>
                    <th className="pb-2 font-medium text-right">RAM</th>
                    <th className="pb-2 font-medium text-right">Disk</th>
                    <th className="pb-2 font-medium text-right">VMs</th>
                    <th className="pb-2 font-medium text-right">Uptime</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {data.node_details.map((node) => (
                    <tr key={node.node_id} className="hover:bg-muted/50">
                      <td className="py-2.5 font-medium">{node.node_name}</td>
                      <td className="py-2.5">
                        <Badge
                          variant={node.is_online ? "success" : "destructive"}
                          className="text-xs"
                        >
                          {node.is_online ? "Online" : "Offline"}
                        </Badge>
                      </td>
                      <td className="py-2.5 text-right font-mono">
                        {node.is_online
                          ? `${(node.cpu_usage ?? 0).toFixed(1)}%`
                          : "-"}
                      </td>
                      <td className="py-2.5 text-right font-mono">
                        {node.is_online
                          ? `${(node.mem_pct ?? 0).toFixed(1)}%`
                          : "-"}
                      </td>
                      <td className="py-2.5 text-right font-mono">
                        {node.is_online
                          ? `${(node.disk_pct ?? 0).toFixed(1)}%`
                          : "-"}
                      </td>
                      <td className="py-2.5 text-right">
                        {node.is_online
                          ? `${node.vm_running ?? 0}/${node.vm_count ?? 0}`
                          : "-"}
                      </td>
                      <td className="py-2.5 text-right text-muted-foreground">
                        <span className="flex items-center justify-end gap-1">
                          <Clock className="h-3 w-3" />
                          {node.is_online && node.uptime > 0
                            ? formatUptime(node.uptime)
                            : "-"}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">Keine Nodes konfiguriert</p>
          )}
        </CardContent>
      </Card>

      {/* Disk Usage */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="flex items-center gap-2 text-base">
            <HardDrive className="h-4 w-4 text-muted-foreground" />
            Speicher-Uebersicht
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-4">
            <div className="flex-1">
              <div className="flex items-center justify-between mb-1.5">
                <span className="text-sm text-muted-foreground">
                  Durchschnittliche Disk-Auslastung
                </span>
                <span className="text-sm font-mono font-medium">
                  {data.avg_disk.toFixed(1)}%
                </span>
              </div>
              <div className="h-2.5 w-full rounded-full bg-muted">
                <div
                  className={`h-full rounded-full transition-all ${
                    data.avg_disk > 90
                      ? "bg-red-500"
                      : data.avg_disk > 75
                        ? "bg-orange-500"
                        : "bg-green-500"
                  }`}
                  style={{ width: `${Math.min(data.avg_disk, 100)}%` }}
                />
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
