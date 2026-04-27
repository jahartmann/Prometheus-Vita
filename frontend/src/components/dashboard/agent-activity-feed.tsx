"use client";

import { useCallback, useEffect, useState } from "react";
import { Bot, CheckCircle2, Clock, RefreshCw, ShieldX, XCircle } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { chatApi } from "@/lib/api";

interface AgentToolCall {
  id: string;
  message_id: string;
  tool_name: string;
  arguments?: unknown;
  result?: unknown;
  status: "running" | "completed" | "error" | "denied" | string;
  duration_ms: number;
  created_at: string;
}

function statusIcon(status: string) {
  switch (status) {
    case "completed":
      return <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500" />;
    case "error":
      return <XCircle className="h-3.5 w-3.5 text-red-500" />;
    case "denied":
      return <ShieldX className="h-3.5 w-3.5 text-amber-500" />;
    case "running":
      return <Clock className="h-3.5 w-3.5 animate-pulse text-blue-500" />;
    default:
      return <Clock className="h-3.5 w-3.5 text-muted-foreground" />;
  }
}

function statusTone(status: string): "outline" | "success" | "destructive" | "warning" | "secondary" {
  switch (status) {
    case "completed":
      return "success";
    case "error":
      return "destructive";
    case "denied":
      return "warning";
    case "running":
      return "secondary";
    default:
      return "outline";
  }
}

function timeAgo(iso: string): string {
  const then = new Date(iso).getTime();
  const diff = Math.max(0, Date.now() - then);
  const sec = Math.floor(diff / 1000);
  if (sec < 60) return `vor ${sec}s`;
  const min = Math.floor(sec / 60);
  if (min < 60) return `vor ${min}m`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `vor ${hr}h`;
  const days = Math.floor(hr / 24);
  return `vor ${days}T`;
}

function summarizeArgs(args: unknown): string {
  if (!args) return "";
  try {
    const obj = typeof args === "string" ? JSON.parse(args) : args;
    if (!obj || typeof obj !== "object") return "";
    const entries = Object.entries(obj as Record<string, unknown>).slice(0, 3);
    return entries
      .map(([k, v]) => `${k}=${typeof v === "string" ? v : JSON.stringify(v)}`)
      .join(", ");
  } catch {
    return "";
  }
}

interface AgentActivityFeedProps {
  limit?: number;
  pollInterval?: number;
}

export function AgentActivityFeed({ limit = 25, pollInterval = 15000 }: AgentActivityFeedProps) {
  const [calls, setCalls] = useState<AgentToolCall[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const data = (await chatApi.recentActivity(limit)) as AgentToolCall[];
      setCalls(Array.isArray(data) ? data : []);
      setError(null);
    } catch {
      setError("Aktivitäten konnten nicht geladen werden");
    } finally {
      setIsLoading(false);
    }
  }, [limit]);

  useEffect(() => {
    load();
    if (!pollInterval) return;
    const handle = setInterval(load, pollInterval);
    return () => clearInterval(handle);
  }, [load, pollInterval]);

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-start gap-3">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-primary/10">
              <Bot className="h-4 w-4 text-primary" />
            </div>
            <div>
              <CardTitle className="text-base">Was der Agent gerade tut</CardTitle>
              <CardDescription>
                Live-Stream der Tool-Aufrufe — wie ein Build-Log für deinen Admin-Helfer.
              </CardDescription>
            </div>
          </div>
          <Button variant="ghost" size="sm" onClick={load} disabled={isLoading}>
            <RefreshCw className={`h-4 w-4 ${isLoading ? "animate-spin" : ""}`} />
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {error && (
          <div className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-xs text-destructive">
            {error}
          </div>
        )}
        {!error && calls.length === 0 && !isLoading && (
          <div className="py-8 text-center text-sm text-muted-foreground">
            Der Agent hat noch nichts getan. Stell ihm eine Frage im Chat oder warte auf den
            nächsten Briefing-Zyklus.
          </div>
        )}
        <ul className="divide-y divide-border">
          {calls.map((c) => {
            const argSummary = summarizeArgs(c.arguments);
            return (
              <li key={c.id} className="flex items-start gap-3 py-2.5">
                <div className="mt-0.5 shrink-0">{statusIcon(c.status)}</div>
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-baseline gap-2">
                    <span className="font-mono text-sm font-medium">{c.tool_name}</span>
                    <Badge variant={statusTone(c.status)} className="text-[10px]">
                      {c.status}
                    </Badge>
                    {c.duration_ms > 0 && (
                      <span className="text-[10px] text-muted-foreground">{c.duration_ms}ms</span>
                    )}
                    <span className="ml-auto text-[10px] text-muted-foreground">
                      {timeAgo(c.created_at)}
                    </span>
                  </div>
                  {argSummary && (
                    <p className="mt-0.5 truncate text-xs text-muted-foreground">{argSummary}</p>
                  )}
                </div>
              </li>
            );
          })}
        </ul>
      </CardContent>
    </Card>
  );
}
