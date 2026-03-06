"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { securityApi } from "@/lib/api";
import type { SecurityEvent, SecurityStats } from "@/types/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ShieldCheck, ShieldAlert, ArrowRight, Clock } from "lucide-react";

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
  return `vor ${days}d`;
}

const severityBadgeVariant: Record<string, "secondary" | "warning" | "destructive"> = {
  info: "secondary",
  warning: "warning",
  critical: "destructive",
  emergency: "destructive",
};

export function SecurityWidget() {
  const [events, setEvents] = useState<SecurityEvent[]>([]);
  const [stats, setStats] = useState<SecurityStats | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      securityApi.getRecent(3).catch(() => []),
      securityApi.getStats().catch(() => null),
    ])
      .then(([evts, st]) => {
        setEvents(evts as SecurityEvent[]);
        setStats(st as SecurityStats | null);
      })
      .finally(() => setIsLoading(false));
  }, []);

  if (isLoading) {
    return (
      <Card>
        <CardContent className="p-5">
          <div className="h-20 animate-pulse rounded bg-muted" />
        </CardContent>
      </Card>
    );
  }

  const unack = stats?.unacknowledged ?? 0;
  const critical = (stats?.by_severity?.critical ?? 0) + (stats?.by_severity?.emergency ?? 0);
  const hasIssues = unack > 0;

  return (
    <Card hover>
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2 text-base">
            {hasIssues ? (
              <ShieldAlert className="h-4 w-4 text-orange-500" />
            ) : (
              <ShieldCheck className="h-4 w-4 text-green-500" />
            )}
            Sicherheit
          </CardTitle>
          <Button variant="ghost" size="sm" asChild>
            <Link href="/security" className="flex items-center gap-1 text-xs">
              Details
              <ArrowRight className="h-3 w-3" />
            </Link>
          </Button>
        </div>
      </CardHeader>
      <CardContent className="pt-0">
        {/* Severity badges */}
        <div className="flex flex-wrap gap-2 mb-3">
          {hasIssues ? (
            <>
              {critical > 0 && (
                <Badge variant="destructive" className="text-xs">
                  {critical} kritisch
                </Badge>
              )}
              {(stats?.by_severity?.warning ?? 0) > 0 && (
                <Badge variant="warning" className="text-xs">
                  {stats?.by_severity?.warning} Warnungen
                </Badge>
              )}
              <Badge variant="outline" className="text-xs">
                {unack} unbestaetigt
              </Badge>
            </>
          ) : (
            <Badge variant="secondary" className="text-xs gap-1">
              <ShieldCheck className="h-3 w-3 text-green-500" />
              Alle Systeme sicher
            </Badge>
          )}
        </div>

        {/* Latest events */}
        {events.length > 0 && (
          <div className="space-y-1.5">
            {events.map((e) => (
              <div
                key={e.id}
                className="flex items-center gap-2 text-sm"
              >
                <Badge
                  variant={severityBadgeVariant[e.severity] ?? "secondary"}
                  className="text-[10px] py-0 px-1.5 shrink-0"
                >
                  {e.severity === "critical" || e.severity === "emergency"
                    ? "!"
                    : e.severity === "warning"
                      ? "W"
                      : "i"}
                </Badge>
                <span className="truncate flex-1 text-xs">{e.title}</span>
                <span className="text-[10px] text-muted-foreground shrink-0 flex items-center gap-0.5">
                  <Clock className="h-2.5 w-2.5" />
                  {timeAgo(e.detected_at)}
                </span>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
