"use client";

import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { securityApi } from "@/lib/api";
import type { SecurityEvent, SecurityStats, SecurityCategory } from "@/types/api";
import { PageShell } from "@/components/layout/page-shell";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { FeatureStatusCard } from "@/components/ui/feature-status-card";
import { KpiCard } from "@/components/ui/kpi-card";
import { StatusBadge, type StatusTone } from "@/components/ui/status-badge";
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
  { label: string; icon: typeof Shield; colorClass: string }
> = {
  security: {
    label: "Sicherheit",
    icon: Shield,
    colorClass: "text-violet-500",
  },
  performance: {
    label: "Performance",
    icon: Gauge,
    colorClass: "text-blue-500",
  },
  capacity: {
    label: "Kapazität",
    icon: HardDrive,
    colorClass: "text-orange-500",
  },
  availability: {
    label: "Verfügbarkeit",
    icon: Server,
    colorClass: "text-green-500",
  },
  config: {
    label: "Konfiguration",
    icon: Settings,
    colorClass: "text-slate-500",
  },
};

const severityBorder: Record<string, string> = {
  info: "border-l-green-500",
  warning: "border-l-yellow-500",
  critical: "border-l-orange-500",
  emergency: "border-l-red-500",
};

const severityBadge: Record<string, { label: string }> = {
  info: { label: "Info" },
  warning: { label: "Warnung" },
  critical: { label: "Kritisch" },
  emergency: { label: "Notfall" },
};

const severityTone: Record<string, StatusTone> = {
  info: "info",
  warning: "warning",
  critical: "critical",
  emergency: "critical",
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
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setIsLoading(true);
    setError(null);
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
      setError("Sicherheitsdaten konnten nicht geladen werden");
      setEvents([]);
      setStats(null);
      setExpandedIds(new Set());
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

  const eventEmergencyCount = events.filter((e) => e.severity === "emergency").length;
  const eventCriticalCount = events.filter((e) => e.severity === "critical").length;
  const unacknowledgedCount = stats?.unacknowledged ?? events.filter((e) => !e.is_acknowledged).length;
  const emergencyCount = stats?.by_severity?.emergency ?? eventEmergencyCount;
  const criticalCount = (stats?.by_severity?.critical ?? eventCriticalCount) + emergencyCount;
  const warningCount = stats?.by_severity?.warning ?? events.filter((e) => e.severity === "warning").length;
  const totalCount = stats?.total ?? events.length;
  const canShowData = !error;

  const hasCritical = criticalCount > 0;
  const hasEmergency = emergencyCount > 0;
  const hasWarnings = !hasCritical && (warningCount > 0 || unacknowledgedCount > 0);
  const statusTone: StatusTone = hasCritical ? "critical" : hasWarnings ? "warning" : "ok";
  const statusIcon = hasCritical ? ShieldAlert : hasWarnings ? Bell : ShieldCheck;
  const statusLabel = hasEmergency
    ? "Notfall"
    : hasCritical
      ? "Kritisch"
      : hasWarnings
        ? "Aufmerksamkeit"
        : "Stabil";
  const statusDescription = hasEmergency
    ? `${emergencyCount} Notfall-Befund${emergencyCount > 1 ? "e" : ""} erkannt. Sofort pruefen und Massnahmen einleiten.`
    : hasCritical
      ? `${criticalCount} kritische Befunde erkannt. Bitte priorisiert bearbeiten und betroffene Nodes absichern.`
      : hasWarnings
        ? `${warningCount || unacknowledgedCount} Befund${(warningCount || unacknowledgedCount) > 1 ? "e" : ""} benoetigen Aufmerksamkeit oder Bestaetigung.`
        : totalCount > 0
          ? "Aktuelle Analyse ohne kritische oder warnende Befunde."
          : "Alle Systeme sicher, keine offenen Befunde.";
  const pageActions = (
    <>
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
        <RefreshCw className={`mr-2 h-4 w-4 ${isLoading ? "animate-spin" : ""}`} />
        Aktualisieren
      </Button>
    </>
  );

  return (
    <PageShell
      title="Sicherheit"
      eyebrow="Security"
      description="Befunde, Analysemodus und Bestaetigungen in einem klaren Bewertungsflow."
      actions={pageActions}
    >
      {error && (
        <div className="flex items-start gap-2 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/25 dark:text-red-300">
          <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      {canShowData && (
        <>
      {!isLoading && (
        <FeatureStatusCard
          title="Sicherheitslage"
          description={statusDescription}
          icon={statusIcon}
          tone={statusTone}
          status={statusLabel}
          details={
            <div className="flex flex-wrap gap-2 text-sm text-muted-foreground">
              <span>{totalCount} Befunde gesamt</span>
              <span>{criticalCount} kritisch/notfall</span>
              <span>{unacknowledgedCount} unbestaetigt</span>
            </div>
          }
        />
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
                  className={`border-l-2 ${severityBorder[event.severity] ?? "border-l-muted"} ${
                    event.is_acknowledged ? "opacity-60" : ""
                  }`}
                >
                  <CardHeader className="pb-2 pt-4 px-5">
                    <div className="flex items-start justify-between gap-4">
                      <div className="flex items-start gap-3 min-w-0">
                        <div className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-muted">
                          <CatIcon className={`h-4 w-4 ${catCfg.colorClass}`} />
                        </div>
                        <div className="min-w-0">
                          <div className="flex flex-wrap items-center gap-2">
                            <span className="font-semibold text-sm">
                              {event.title}
                            </span>
                            <StatusBadge tone={severityTone[event.severity] ?? "muted"} className="text-xs">
                              {sevCfg.label}
                            </StatusBadge>
                            <StatusBadge tone="muted" className="text-xs" withIcon={false}>
                              {catCfg.label}
                            </StatusBadge>
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
        </>
      )}
    </PageShell>
  );
}
