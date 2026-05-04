"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import { Activity, AlertCircle, Archive, ArrowRightLeft, Bell, CalendarClock, CheckCircle2, Clock, ListChecks, RefreshCw, ShieldAlert } from "lucide-react";
import { getApiErrorMessage, operationsApi } from "@/lib/api";
import type { OperationTask } from "@/types/api";
import { PageShell } from "@/components/layout/page-shell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { Input } from "@/components/ui/input";
import { KpiCard } from "@/components/ui/kpi-card";
import { Progress } from "@/components/ui/progress";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";

const statusLabel: Record<string, string> = {
  running: "Aktiv",
  pending: "Wartet",
  failed: "Fehler",
  completed: "Fertig",
  warning: "Prüfen",
};

const statusClass: Record<string, string> = {
  running: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300",
  pending: "bg-slate-100 text-slate-800 dark:bg-slate-900/40 dark:text-slate-300",
  failed: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300",
  completed: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300",
  warning: "bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-300",
};

function formatDate(value?: string) {
  if (!value) return "-";
  return new Date(value).toLocaleString("de-DE", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" });
}

export default function TaskCenterPage() {
  const [tasks, setTasks] = useState<OperationTask[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [hasLoaded, setHasLoaded] = useState(false);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState("all");
  const [query, setQuery] = useState("");
  const [autoRefresh, setAutoRefresh] = useState(true);
  const requestSeqRef = useRef(0);

  const load = useCallback(async () => {
    const requestSeq = requestSeqRef.current + 1;
    requestSeqRef.current = requestSeq;
    setIsLoading(true);
    setError(null);
    try {
      const nextTasks = await operationsApi.listTasks({ limit: 80 }) as OperationTask[];
      if (requestSeqRef.current !== requestSeq) return;
      setTasks(nextTasks);
    } catch (e) {
      if (requestSeqRef.current !== requestSeq) return;
      setError(getApiErrorMessage(e, "Aufgaben konnten nicht geladen werden"));
      setTasks([]);
    } finally {
      if (requestSeqRef.current === requestSeq) {
        setLastUpdated(new Date());
        setHasLoaded(true);
        setIsLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    if (!autoRefresh) return;
    const interval = setInterval(load, 15000);
    return () => clearInterval(interval);
  }, [autoRefresh, load]);

  const totals = useMemo(() => {
    return {
      active: tasks.filter((task) => task.status === "running" || task.status === "pending").length,
      failed: tasks.filter((task) => task.status === "failed").length,
      warning: tasks.filter((task) => task.status === "warning").length,
      done: tasks.filter((task) => task.status === "completed").length,
    };
  }, [tasks]);

  const visibleTasks = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return tasks.filter((task) => {
      const matchesStatus = statusFilter === "all" || task.status === statusFilter;
      const matchesQuery =
        !needle ||
        task.title.toLowerCase().includes(needle) ||
        task.detail.toLowerCase().includes(needle) ||
        task.type.toLowerCase().includes(needle);
      return matchesStatus && matchesQuery;
    });
  }, [query, statusFilter, tasks]);

  const isInitialLoading = isLoading && !hasLoaded;
  const isRefreshing = isLoading && hasLoaded;

  const iconFor = (type: OperationTask["type"]) => {
    if (type === "migration") return <ArrowRightLeft className="h-4 w-4" />;
    if (type === "backup") return <Archive className="h-4 w-4" />;
    if (type === "incident") return <ShieldAlert className="h-4 w-4" />;
    if (type === "scheduled_job" || type === "scheduled_report" || type === "scheduled_action") return <CalendarClock className="h-4 w-4" />;
    return <Bell className="h-4 w-4" />;
  };

  const pageActions = (
    <div className="flex flex-wrap items-center gap-2">
      <div className="flex items-center gap-2 rounded-md border bg-card px-2.5 py-1.5">
        <Switch checked={autoRefresh} onCheckedChange={setAutoRefresh} />
        <span className="text-xs text-muted-foreground">Auto</span>
      </div>
      <Button variant="outline" size="sm" onClick={load} disabled={isLoading}>
        <RefreshCw className={cn("mr-2 h-4 w-4", isLoading && "animate-spin")} />
        Aktualisieren
      </Button>
    </div>
  );

  return (
    <PageShell
      title="Aufgaben"
      eyebrow="Operations"
      description="Lange Operationen, offene Incidents und fehlgeschlagene Benachrichtigungen in einer Arbeitsliste."
      actions={pageActions}
    >

      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <KpiCard title="Aktiv" value={totals.active} subtitle="Laufend oder wartend" icon={Activity} color="blue" />
        <KpiCard title="Fehler" value={totals.failed} subtitle="Fehlgeschlagen" icon={ShieldAlert} color="red" />
        <KpiCard title="Prüfen" value={totals.warning} subtitle="Benötigen Aufmerksamkeit" icon={AlertCircle} color="orange" />
        <KpiCard title="Fertig" value={totals.done} subtitle="Abgeschlossen" icon={CheckCircle2} color="green" />
      </div>

      {error && (
        <div className="flex items-start gap-2 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/25 dark:text-red-300">
          <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      <div className="rounded-md border bg-card">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3">
          <div className="flex items-center gap-2 text-sm font-medium">
            <ListChecks className="h-4 w-4" />
            Operations-Queue
            {isRefreshing && (
              <Badge variant="secondary" className="gap-1 text-[10px]">
                <RefreshCw className="h-3 w-3 animate-spin" />
                aktualisiert
              </Badge>
            )}
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Aufgabe suchen..."
              className="h-8 w-[180px]"
            />
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="h-8 w-[150px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Alle Status</SelectItem>
                <SelectItem value="running">Aktiv</SelectItem>
                <SelectItem value="pending">Wartet</SelectItem>
                <SelectItem value="warning">Prüfen</SelectItem>
                <SelectItem value="failed">Fehler</SelectItem>
                <SelectItem value="completed">Fertig</SelectItem>
              </SelectContent>
            </Select>
            <div className="flex items-center gap-1 text-xs text-muted-foreground">
              <Clock className="h-3.5 w-3.5" />
              {lastUpdated ? formatDate(lastUpdated.toISOString()) : "-"}
            </div>
          </div>
        </div>
        <div className="divide-y">
          {isInitialLoading ? (
            Array.from({ length: 6 }).map((_, index) => (
              <div key={index} className="grid gap-3 px-4 py-3 md:grid-cols-[1fr,180px,110px] md:items-center">
                <div className="space-y-2">
                  <Skeleton className="h-4 w-2/3" />
                  <Skeleton className="h-3 w-4/5" />
                </div>
                <Skeleton className="h-2 w-full" />
                <Skeleton className="h-3 w-20 md:ml-auto" />
              </div>
            ))
          ) : tasks.length === 0 ? (
            <EmptyState
              icon={ListChecks}
              title="Keine Operationen gefunden"
              description="Aktuell gibt es keine laufenden, fehlgeschlagenen oder auffälligen Aufgaben."
              action={<Button variant="outline" size="sm" onClick={load}>Jetzt pruefen</Button>}
            />
          ) : visibleTasks.length === 0 ? (
            <EmptyState
              icon={ListChecks}
              title="Keine Treffer"
              description="Passe Suche oder Statusfilter an, um weitere Aufgaben zu sehen."
              action={<Button variant="outline" size="sm" onClick={() => { setQuery(""); setStatusFilter("all"); }}>Filter zuruecksetzen</Button>}
            />
          ) : (
            visibleTasks.slice(0, 60).map((task) => (
              <Link key={task.id} href={task.href} className="grid gap-3 px-4 py-3 transition-colors hover:bg-muted/50 md:grid-cols-[1fr,180px,110px] md:items-center">
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    {iconFor(task.type)}
                    <span className="truncate text-sm font-medium">{task.title}</span>
                    <Badge variant="secondary" className={statusClass[task.status] ?? statusClass.pending}>{statusLabel[task.status] ?? task.status}</Badge>
                  </div>
                  <p className="mt-1 truncate text-xs text-muted-foreground">{task.detail}</p>
                </div>
                <div className="flex items-center gap-2">
                  <Progress value={Math.max(0, Math.min(100, task.progress ?? 0))} className="h-2" />
                  <span className="w-9 text-right text-[10px] text-muted-foreground">{Math.round(task.progress ?? 0)}%</span>
                </div>
                <span className="text-xs text-muted-foreground md:text-right">{formatDate(task.due_at ?? task.created_at)}</span>
              </Link>
            ))
          )}
        </div>
      </div>

      <div className="flex items-center gap-2 rounded-md border bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
        <Activity className="h-4 w-4" />
        Migrationen, Backups, Incidents und Notification-Fehler werden zusammengefuehrt. Schreibende Aktionen bleiben auf den jeweiligen Detailseiten.
      </div>
    </PageShell>
  );
}
