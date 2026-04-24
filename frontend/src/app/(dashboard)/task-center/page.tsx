"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { Activity, Archive, ArrowRightLeft, Bell, CalendarClock, Clock, ListChecks, RefreshCw, ShieldAlert } from "lucide-react";
import { operationsApi } from "@/lib/api";
import type { OperationTask } from "@/types/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { cn } from "@/lib/utils";

const statusLabel: Record<string, string> = {
  running: "Aktiv",
  pending: "Wartet",
  failed: "Fehler",
  completed: "Fertig",
  warning: "Pruefen",
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
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  const load = useCallback(async () => {
    setIsLoading(true);
    try {
      const nextTasks = await operationsApi.listTasks({ limit: 80 }) as OperationTask[];
      setTasks(nextTasks);
    } catch {
      setTasks([]);
    }
    setLastUpdated(new Date());
    setIsLoading(false);
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const totals = useMemo(() => {
    return {
      active: tasks.filter((task) => task.status === "running" || task.status === "pending").length,
      failed: tasks.filter((task) => task.status === "failed").length,
      warning: tasks.filter((task) => task.status === "warning").length,
      done: tasks.filter((task) => task.status === "completed").length,
    };
  }, [tasks]);

  const iconFor = (type: OperationTask["type"]) => {
    if (type === "migration") return <ArrowRightLeft className="h-4 w-4" />;
    if (type === "backup") return <Archive className="h-4 w-4" />;
    if (type === "incident") return <ShieldAlert className="h-4 w-4" />;
    if (type === "scheduled_job" || type === "scheduled_report" || type === "scheduled_action") return <CalendarClock className="h-4 w-4" />;
    return <Bell className="h-4 w-4" />;
  };

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Task-Center</h1>
          <p className="text-sm text-muted-foreground">Lange Operationen, offene Incidents und fehlgeschlagene Benachrichtigungen in einer Arbeitsliste.</p>
        </div>
        <Button variant="outline" size="sm" onClick={load} disabled={isLoading}>
          <RefreshCw className={cn("mr-2 h-4 w-4", isLoading && "animate-spin")} />
          Aktualisieren
        </Button>
      </div>

      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">Aktiv</CardTitle></CardHeader><CardContent className="text-2xl font-semibold">{totals.active}</CardContent></Card>
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">Fehler</CardTitle></CardHeader><CardContent className="text-2xl font-semibold">{totals.failed}</CardContent></Card>
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">Pruefen</CardTitle></CardHeader><CardContent className="text-2xl font-semibold">{totals.warning}</CardContent></Card>
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">Fertig</CardTitle></CardHeader><CardContent className="text-2xl font-semibold">{totals.done}</CardContent></Card>
      </div>

      <div className="rounded-md border">
        <div className="flex items-center justify-between border-b px-4 py-3">
          <div className="flex items-center gap-2 text-sm font-medium">
            <ListChecks className="h-4 w-4" />
            Operations-Queue
          </div>
          <div className="flex items-center gap-1 text-xs text-muted-foreground">
            <Clock className="h-3.5 w-3.5" />
            {lastUpdated ? formatDate(lastUpdated.toISOString()) : "-"}
          </div>
        </div>
        <div className="divide-y">
          {isLoading ? (
            <div className="px-4 py-10 text-center text-sm text-muted-foreground">Tasks werden geladen...</div>
          ) : tasks.length === 0 ? (
            <div className="px-4 py-10 text-center text-sm text-muted-foreground">Keine laufenden oder auffaelligen Operationen.</div>
          ) : (
            tasks.slice(0, 40).map((task) => (
              <Link key={task.id} href={task.href} className="grid gap-3 px-4 py-3 transition-colors hover:bg-muted/50 md:grid-cols-[1fr,180px,110px] md:items-center">
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    {iconFor(task.type)}
                    <span className="truncate text-sm font-medium">{task.title}</span>
                    <Badge variant="secondary" className={statusClass[task.status]}>{statusLabel[task.status]}</Badge>
                  </div>
                  <p className="mt-1 truncate text-xs text-muted-foreground">{task.detail}</p>
                </div>
                <Progress value={task.progress} className="h-2" />
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
    </div>
  );
}
