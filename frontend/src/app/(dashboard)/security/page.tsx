"use client";

import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { securityApi } from "@/lib/api";
import type { SecurityEvent, SecurityStats, SecurityCategory } from "@/types/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { KpiCard } from "@/components/ui/kpi-card";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Shield,
  ShieldCheck,
  ShieldAlert,
  AlertTriangle,
  Bell,
  Sparkles,
  Gauge,
  HardDrive,
  Server,
  Settings,
  ChevronDown,
  CheckCircle2,
  RefreshCw,
  Clock,
} from "lucide-react";

const categoryConfig: Record<
  SecurityCategory,
  { label: string; icon: typeof Shield; colorClass: string; badgeClass: string }
> = {
  security: {
    label: "Sicherheit",
    icon: Shield,
    colorClass: "text-violet-500",
    badgeClass: "bg-violet-500/10 text-violet-500 border-violet-500/30",
  },
  performance: {
    label: "Performance",
    icon: Gauge,
    colorClass: "text-blue-500",
    badgeClass: "bg-blue-500/10 text-blue-500 border-blue-500/30",
  },
  capacity: {
    label: "Kapazität",
    icon: HardDrive,
    colorClass: "text-orange-500",
    badgeClass: "bg-orange-500/10 text-orange-500 border-orange-500/30",
  },
  availability: {
    label: "Verfügbarkeit",
    icon: Server,
    colorClass: "text-green-500",
    badgeClass: "bg-green-500/10 text-green-500 border-green-500/30",
  },
  config: {
    label: "Konfiguration",
    icon: Settings,
    colorClass: "text-slate-500",
    badgeClass: "bg-slate-500/10 text-slate-500 border-slate-500/30",
  },
};

const severityBorder: Record<string, string> = {
  info: "border-l-green-500",
  warning: "border-l-yellow-500",
  critical: "border-l-orange-500",
  emergency: "border-l-red-500",
};

const severityBadge: Record<string, { label: string; variant: "secondary" | "warning" | "destructive" }> = {
  info: { label: "Info", variant: "secondary" },
  warning: { label: "Warnung", variant: "warning" },
  critical: { label: "Kritisch", variant: "destructive" },
  emergency: { label: "Notfall", variant: "destructive" },
};

function timeAgo(dateStr: string): string {
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const diff = Math.max(0, now - then);
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "gerade eben";
  if (mins < 60) return `vor ${mins} Min.`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `vor ${hours} Std.`;
  const days = Math.floor(hours / 24);
  return `vor ${days} Tag${days > 1 ? "en" : ""}`;
}

export default function SecurityPage() {
  const [events, setEvents] = useState<SecurityEvent[]>([]);
  const [stats, setStats] = useState<SecurityStats | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [categoryFilter, setCategoryFilter] = useState<string>("all");
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());
  const [acknowledgingIds, setAcknowledgingIds] = useState<Set<string>>(new Set());
  const [analysisMode, setAnalysisMode] = useState<string>("hybrid");

  const fetchData = useCallback(async () => {
    setIsLoading(true);
    try {
      const [evts, st, modeRes] = await Promise.all([
        securityApi.getRecent(100),
        securityApi.getStats().catch(() => null),
        securityApi.getMode().catch(() => ({ mode: "hybrid" })),
      ]);
      setEvents(evts as SecurityEvent[]);
      setStats(st as SecurityStats | null);
      if (modeRes && typeof modeRes === "object" && "mode" in modeRes) {
        setAnalysisMode((modeRes as { mode: string }).mode);
      }
    } catch {
      // silent
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleAcknowledge = useCallback(async (id: string) => {
    setAcknowledgingIds((prev) => new Set(prev).add(id));
    try {
      await securityApi.acknowledge(id);
      setEvents((prev) =>
        prev.map((e) => (e.id === id ? { ...e, is_acknowledged: true, acknowledged_at: new Date().toISOString() } : e))
      );
      setStats((prev) =>
        prev ? { ...prev, unacknowledged: Math.max(0, prev.unacknowledged - 1) } : prev
      );
    } catch {
      // silent
    } finally {
      setAcknowledgingIds((prev) => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
    }
  }, []);

  const toggleExpand = (id: string) => {
    setExpandedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const filtered =
    categoryFilter === "all"
      ? events
      : events.filter((e) => e.category === categoryFilter);

  const unacknowledgedCount = stats?.unacknowledged ?? events.filter((e) => !e.is_acknowledged).length;
  const criticalCount = (stats?.by_severity?.critical ?? 0) + (stats?.by_severity?.emergency ?? 0);
  const totalCount = stats?.total ?? events.length;

  // Status banner
  const hasCritical = criticalCount > 0;
  const hasWarnings = unacknowledgedCount > 0 && !hasCritical;
  const hasEmergency = (stats?.by_severity?.emergency ?? 0) > 0;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            Sicherheit & Intelligente Analyse
          </h1>
          <p className="text-muted-foreground">
            KI-gestützte Erkennung von Anomalien, Sicherheitsbedrohungen und
            Kapazitätsrisiken
          </p>
        </div>
        <div className="flex items-center gap-2">
          {/* Analysis Mode Switcher */}
          <div className="flex items-center rounded-lg border bg-muted/50 p-0.5">
            {([
              { value: "hybrid", label: "Hybrid" },
              { value: "full_llm", label: "Full LLM" },
              { value: "rule_only", label: "Nur Regeln" },
            ] as const).map((m) => (
              <button
                key={m.value}
                onClick={async () => {
                  try {
                    await securityApi.setMode(m.value);
                    setAnalysisMode(m.value);
                  } catch { /* silent */ }
                }}
                className={`rounded-md px-2.5 py-1 text-xs font-medium transition-colors ${
                  analysisMode === m.value
                    ? "bg-background text-foreground shadow-sm"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                {m.label}
              </button>
            ))}
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={fetchData}
            disabled={isLoading}
          >
            <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? "animate-spin" : ""}`} />
            Aktualisieren
          </Button>
        </div>
      </div>

      {/* Status Banner */}
      {!isLoading && (
        <div
          className={`flex items-center gap-3 rounded-lg border p-4 ${
            hasEmergency
              ? "border-red-700 bg-red-950/50 text-red-300"
              : hasCritical
                ? "border-red-500/50 bg-red-500/10 text-red-500"
                : hasWarnings
                  ? "border-yellow-500/50 bg-yellow-500/10 text-yellow-600 dark:text-yellow-400"
                  : "border-green-500/50 bg-green-500/10 text-green-600 dark:text-green-400"
          }`}
        >
          {hasEmergency ? (
            <ShieldAlert className="h-5 w-5 shrink-0" />
          ) : hasCritical ? (
            <AlertTriangle className="h-5 w-5 shrink-0" />
          ) : hasWarnings ? (
            <Bell className="h-5 w-5 shrink-0" />
          ) : (
            <ShieldCheck className="h-5 w-5 shrink-0" />
          )}
          <span className="font-medium">
            {hasEmergency
              ? `NOTFALL: ${stats?.by_severity?.emergency ?? 0} Sicherheitsvorfall${(stats?.by_severity?.emergency ?? 0) > 1 ? "e" : ""} erkannt`
              : hasCritical
                ? `${criticalCount} kritische Befunde -- sofortiges Handeln erforderlich`
                : hasWarnings
                  ? `${unacknowledgedCount} Befund${unacknowledgedCount > 1 ? "e" : ""} erfordern Aufmerksamkeit`
                  : "Alle Systeme sicher -- keine offenen Befunde"}
          </span>
        </div>
      )}

      {/* KPI Cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard
          title="Gesamt Events"
          value={totalCount}
          subtitle="Erkannte Befunde"
          icon={Shield}
          color="blue"
        />
        <KpiCard
          title="Kritisch / Notfall"
          value={criticalCount}
          subtitle={criticalCount > 0 ? "Sofortiges Handeln" : "Keine kritischen"}
          icon={AlertTriangle}
          color="red"
        />
        <KpiCard
          title="Unbestätigt"
          value={unacknowledgedCount}
          subtitle="Offene Befunde"
          icon={Bell}
          color="orange"
        />
        <KpiCard
          title="KI-analysiert"
          value={totalCount}
          subtitle="Automatische Analyse"
          icon={Sparkles}
          color="default"
        />
      </div>

      {/* Category Filter Tabs */}
      <Tabs value={categoryFilter} onValueChange={setCategoryFilter}>
        <TabsList>
          <TabsTrigger value="all">Alle</TabsTrigger>
          <TabsTrigger value="security" className="gap-1.5">
            <Shield className="h-3.5 w-3.5" />
            Sicherheit
          </TabsTrigger>
          <TabsTrigger value="performance" className="gap-1.5">
            <Gauge className="h-3.5 w-3.5" />
            Performance
          </TabsTrigger>
          <TabsTrigger value="capacity" className="gap-1.5">
            <HardDrive className="h-3.5 w-3.5" />
            Kapazität
          </TabsTrigger>
          <TabsTrigger value="availability" className="gap-1.5">
            <Server className="h-3.5 w-3.5" />
            Verfügbarkeit
          </TabsTrigger>
          <TabsTrigger value="config" className="gap-1.5">
            <Settings className="h-3.5 w-3.5" />
            Konfiguration
          </TabsTrigger>
        </TabsList>
      </Tabs>

      {/* Event Timeline */}
      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <Card key={i}>
              <CardContent className="p-5">
                <div className="h-16 animate-pulse rounded bg-muted" />
              </CardContent>
            </Card>
          ))}
        </div>
      ) : filtered.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-16">
            <ShieldCheck className="mb-4 h-12 w-12 text-muted-foreground" />
            <p className="text-lg font-medium">Keine Befunde</p>
            <p className="mt-1 text-sm text-muted-foreground">
              {categoryFilter === "all"
                ? "Es wurden keine Sicherheitsereignisse erkannt."
                : `Keine Ereignisse in der Kategorie "${categoryConfig[categoryFilter as SecurityCategory]?.label ?? categoryFilter}".`}
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {filtered.map((event) => {
            const catCfg = categoryConfig[event.category] ?? categoryConfig.config;
            const CatIcon = catCfg.icon;
            const sevCfg = severityBadge[event.severity] ?? severityBadge.info;
            const isExpanded = expandedIds.has(event.id);

            return (
              <Collapsible
                key={event.id}
                open={isExpanded}
                onOpenChange={() => toggleExpand(event.id)}
              >
                <Card
                  className={`border-l-4 ${severityBorder[event.severity] ?? "border-l-muted"} ${
                    event.is_acknowledged ? "opacity-60" : ""
                  }`}
                >
                  <CardHeader className="pb-2 pt-4 px-5">
                    <div className="flex items-start justify-between gap-4">
                      <div className="flex items-start gap-3 min-w-0">
                        <div
                          className={`mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg ${
                            catCfg.colorClass === "text-violet-500"
                              ? "bg-violet-500/10"
                              : catCfg.colorClass === "text-blue-500"
                                ? "bg-blue-500/10"
                                : catCfg.colorClass === "text-orange-500"
                                  ? "bg-orange-500/10"
                                  : catCfg.colorClass === "text-green-500"
                                    ? "bg-green-500/10"
                                    : "bg-slate-500/10"
                          }`}
                        >
                          <CatIcon className={`h-4 w-4 ${catCfg.colorClass}`} />
                        </div>
                        <div className="min-w-0">
                          <div className="flex flex-wrap items-center gap-2">
                            <span className="font-semibold text-sm">
                              {event.title}
                            </span>
                            <Badge
                              variant={sevCfg.variant}
                              className="text-xs"
                            >
                              {sevCfg.label}
                            </Badge>
                            <Badge
                              variant="outline"
                              className={`text-xs ${catCfg.badgeClass}`}
                            >
                              {catCfg.label}
                            </Badge>
                            {event.is_acknowledged && (
                              <Badge variant="secondary" className="text-xs gap-1">
                                <CheckCircle2 className="h-3 w-3" />
                                Bestätigt
                              </Badge>
                            )}
                          </div>
                          <div className="flex flex-wrap items-center gap-2 mt-1 text-xs text-muted-foreground">
                            {event.node_name && (
                              <Link
                                href={`/nodes/${event.node_id}/monitoring`}
                                className="text-primary hover:underline"
                              >
                                {event.node_name}
                              </Link>
                            )}
                            <span className="flex items-center gap-1">
                              <Clock className="h-3 w-3" />
                              {timeAgo(event.detected_at)}
                            </span>
                            {event.affected_vms && event.affected_vms.length > 0 && (
                              <span className="flex items-center gap-1">
                                {event.affected_vms.slice(0, 3).map((vm) => (
                                  <Badge
                                    key={vm}
                                    variant="secondary"
                                    className="text-[10px] py-0 px-1.5"
                                  >
                                    {vm}
                                  </Badge>
                                ))}
                                {event.affected_vms.length > 3 && (
                                  <span className="text-[10px]">
                                    +{event.affected_vms.length - 3}
                                  </span>
                                )}
                              </span>
                            )}
                          </div>
                        </div>
                      </div>
                      <CollapsibleTrigger asChild>
                        <Button variant="ghost" size="sm" className="shrink-0">
                          <ChevronDown
                            className={`h-4 w-4 transition-transform ${isExpanded ? "rotate-180" : ""}`}
                          />
                        </Button>
                      </CollapsibleTrigger>
                    </div>
                  </CardHeader>

                  <CollapsibleContent>
                    <CardContent className="px-5 pb-4 pt-0">
                      <div className="ml-11 space-y-3">
                        {/* Analyse / Description */}
                        {event.description && (
                          <div className="rounded bg-muted/50 p-3 text-sm">
                            <p className="text-xs text-foreground/70 font-semibold mb-1">
                              Analyse
                            </p>
                            <p>{event.description}</p>
                          </div>
                        )}

                        {/* Impact + Recommendation */}
                        <div className="grid gap-3 sm:grid-cols-2">
                          {event.impact && (
                            <div className="rounded bg-muted/50 p-3 text-sm">
                              <p className="text-xs text-foreground/70 font-semibold mb-1">
                                Auswirkung
                              </p>
                              <p>{event.impact}</p>
                            </div>
                          )}
                          {event.recommendation && (
                            <div className="rounded bg-muted/50 p-3 text-sm">
                              <p className="text-xs text-foreground/70 font-semibold mb-1">
                                Empfehlung
                              </p>
                              <p>{event.recommendation}</p>
                            </div>
                          )}
                        </div>

                        {/* Metrics Context */}
                        {event.metrics &&
                          Object.keys(event.metrics).length > 0 && (
                            <div className="rounded bg-muted/50 p-3 text-sm">
                              <p className="text-xs text-foreground/70 font-semibold mb-1">
                                Metrik-Kontext
                              </p>
                              <div className="flex flex-wrap gap-3">
                                {Object.entries(event.metrics).map(
                                  ([key, val]) => (
                                    <span key={key} className="font-mono text-xs">
                                      <span className="text-muted-foreground">
                                        {key}:
                                      </span>{" "}
                                      {typeof val === "number"
                                        ? val.toFixed(2)
                                        : String(val)}
                                    </span>
                                  )
                                )}
                              </div>
                            </div>
                          )}

                        {/* Footer: Model + Acknowledge */}
                        <div className="flex items-center justify-between pt-1">
                          {event.analysis_model && (
                            <span className="text-xs text-muted-foreground">
                              Analysiert durch: {event.analysis_model}
                            </span>
                          )}
                          {!event.is_acknowledged && (
                            <Button
                              variant="outline"
                              size="sm"
                              className="ml-auto"
                              disabled={acknowledgingIds.has(event.id)}
                              onClick={() => handleAcknowledge(event.id)}
                            >
                              {acknowledgingIds.has(event.id)
                                ? "..."
                                : "Als gelesen markieren"}
                            </Button>
                          )}
                        </div>
                      </div>
                    </CardContent>
                  </CollapsibleContent>
                </Card>
              </Collapsible>
            );
          })}
        </div>
      )}
    </div>
  );
}
