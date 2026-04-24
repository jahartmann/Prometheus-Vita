"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { Activity, Archive, Bell, Filter, History, RefreshCw, ShieldAlert, Terminal } from "lucide-react";
import { operationsApi } from "@/lib/api";
import type { TimelineEvent } from "@/types/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";

type RecorderKind = "audit" | "security" | "migration" | "backup" | "notification" | "alert";

function formatDate(value: string) {
  return new Date(value).toLocaleString("de-DE", { day: "2-digit", month: "2-digit", year: "2-digit", hour: "2-digit", minute: "2-digit", second: "2-digit" });
}

function severityClass(severity: TimelineEvent["severity"]) {
  if (severity === "critical") return "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300";
  if (severity === "warning") return "bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-300";
  return "bg-slate-100 text-slate-800 dark:bg-slate-900/40 dark:text-slate-300";
}

export default function FlightRecorderPage() {
  const [events, setEvents] = useState<TimelineEvent[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [filter, setFilter] = useState<"all" | RecorderKind>("all");
  const [range, setRange] = useState("24h");

  const load = useCallback(async () => {
    setIsLoading(true);
    try {
      const from = range === "24h"
        ? new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString()
        : range === "7d"
          ? new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString()
          : undefined;
      const nextEvents = await operationsApi.getTimeline({ limit: 120, from }) as TimelineEvent[];
      setEvents(nextEvents);
    } catch {
      setEvents([]);
    }
    setIsLoading(false);
  }, [range]);

  useEffect(() => {
    load();
  }, [load]);

  const filteredEvents = useMemo(() => events.filter((event) => filter === "all" || event.source === filter), [events, filter]);
  const criticalCount = events.filter((event) => event.severity === "critical").length;
  const warningCount = events.filter((event) => event.severity === "warning").length;

  const iconFor = (kind: RecorderKind) => {
    if (kind === "security") return <ShieldAlert className="h-4 w-4" />;
    if (kind === "migration") return <Activity className="h-4 w-4" />;
    if (kind === "backup") return <Archive className="h-4 w-4" />;
    if (kind === "notification") return <Bell className="h-4 w-4" />;
    return <Terminal className="h-4 w-4" />;
  };

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Infrastructure Flight Recorder</h1>
          <p className="text-sm text-muted-foreground">Zeitlinie aus Audits, Security, Backups, Migrationen und Notifications.</p>
        </div>
        <Button variant="outline" size="sm" onClick={load} disabled={isLoading}>
          <RefreshCw className={cn("mr-2 h-4 w-4", isLoading && "animate-spin")} />
          Aktualisieren
        </Button>
      </div>

      <div className="grid gap-3 sm:grid-cols-3">
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">Ereignisse</CardTitle></CardHeader><CardContent className="text-2xl font-semibold">{events.length}</CardContent></Card>
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">Kritisch</CardTitle></CardHeader><CardContent className="text-2xl font-semibold">{criticalCount}</CardContent></Card>
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">Warnungen</CardTitle></CardHeader><CardContent className="text-2xl font-semibold">{warningCount}</CardContent></Card>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Filter className="h-4 w-4 text-muted-foreground" />
        <Select value={filter} onValueChange={(value) => setFilter(value as "all" | RecorderKind)}>
          <SelectTrigger className="w-[220px]"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Alle Quellen</SelectItem>
            <SelectItem value="audit">Audit</SelectItem>
            <SelectItem value="security">Security</SelectItem>
            <SelectItem value="migration">Migrationen</SelectItem>
            <SelectItem value="backup">Backups</SelectItem>
            <SelectItem value="notification">Notifications</SelectItem>
          </SelectContent>
        </Select>
        <Select value={range} onValueChange={setRange}>
          <SelectTrigger className="w-[170px]"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="24h">Letzte 24h</SelectItem>
            <SelectItem value="7d">Letzte 7 Tage</SelectItem>
            <SelectItem value="all">Alle Zeiten</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="rounded-md border">
        <div className="flex items-center gap-2 border-b px-4 py-3 text-sm font-medium">
          <History className="h-4 w-4" />
          Chronologische Spur
        </div>
        <div className="divide-y">
          {isLoading ? (
            <div className="px-4 py-10 text-center text-sm text-muted-foreground">Recorder-Daten werden geladen...</div>
          ) : filteredEvents.length === 0 ? (
            <div className="px-4 py-10 text-center text-sm text-muted-foreground">Keine Ereignisse im gewaehlten Filter.</div>
          ) : (
            filteredEvents.slice(0, 120).map((event) => (
              <Link key={event.id} href={event.href} className="grid gap-2 px-4 py-3 transition-colors hover:bg-muted/50 md:grid-cols-[170px,1fr,150px] md:items-center">
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  {iconFor(event.source as RecorderKind)}
                  <span>{formatDate(event.created_at)}</span>
                </div>
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="truncate text-sm font-medium">{event.title}</span>
                    <Badge variant="secondary" className={severityClass(event.severity)}>{event.severity}</Badge>
                    <Badge variant="outline">{event.source}</Badge>
                  </div>
                  <p className="mt-1 truncate text-xs text-muted-foreground">{event.detail}</p>
                </div>
                <span className="truncate text-xs text-muted-foreground md:text-right">{event.actor}</span>
              </Link>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
