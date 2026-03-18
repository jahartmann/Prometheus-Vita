"use client";

import { useEffect, useState, useCallback, useRef } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { ArrowLeft, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { useNodeStore } from "@/stores/node-store";
import { logApi } from "@/lib/api";
import { AlertCircle, AlertTriangle, Activity, Zap } from "lucide-react";

const LOG_FILES = [
  { value: "syslog", label: "/var/log/syslog" },
  { value: "auth.log", label: "/var/log/auth.log" },
  { value: "pveproxy", label: "/var/log/pveproxy/access.log" },
  { value: "pvedaemon", label: "/var/log/pvedaemon.log" },
  { value: "pve-firewall", label: "/var/log/pve-firewall.log" },
  { value: "corosync", label: "/var/log/corosync/corosync.log" },
];

const LINE_COUNTS = [50, 100, 200, 500, 1000];

const SEVERITY_COLORS: Record<string, string> = {
  critical: "text-red-500 animate-pulse font-bold",
  error: "text-red-400",
  warning: "text-yellow-400",
  info: "text-zinc-300",
  debug: "text-zinc-500",
};

function inferSeverity(line: string): string {
  const lower = line.toLowerCase();
  if (lower.includes("emerg") || lower.includes("panic") || lower.includes("fatal")) return "critical";
  if (lower.includes("error") || lower.includes("err]") || lower.includes("fail")) return "error";
  if (lower.includes("warn")) return "warning";
  if (lower.includes("debug")) return "debug";
  return "info";
}

function countSeverities(lines: string[]) {
  let errors = 0, warnings = 0, critical = 0;
  for (const line of lines) {
    const sev = inferSeverity(line);
    if (sev === "error") errors++;
    else if (sev === "warning") warnings++;
    else if (sev === "critical") critical++;
  }
  return { errors, warnings, critical };
}

export default function NodeLogsPage() {
  const params = useParams<{ id: string }>();
  const nodeId = params.id;

  const { nodes, fetchNodes } = useNodeStore();
  const node = nodes.find((n) => n.id === nodeId);

  const [logFile, setLogFile] = useState("syslog");
  const [lineCount, setLineCount] = useState(200);
  const [logLines, setLogLines] = useState<string[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [filter, setFilter] = useState("");
  const [error, setError] = useState<string | null>(null);

  const containerRef = useRef<HTMLDivElement>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => { fetchNodes(); }, [fetchNodes]);

  const fetchLogs = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const res = await logApi.getLogs(nodeId, logFile, lineCount);
      const raw = typeof res.data === "string" ? res.data : (res.data?.lines || res.data || "");
      const lines = String(raw).split("\n").filter((l: string) => l.trim());
      setLogLines(lines);
    } catch (err) {
      setError("Fehler beim Laden der Logs");
      setLogLines([]);
    } finally {
      setIsLoading(false);
    }
  }, [nodeId, logFile, lineCount]);

  // Fetch on mount and when params change
  useEffect(() => { fetchLogs(); }, [fetchLogs]);

  // Auto-refresh
  useEffect(() => {
    if (autoRefresh) {
      intervalRef.current = setInterval(fetchLogs, 5000);
    }
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [autoRefresh, fetchLogs]);

  // Auto-scroll to bottom
  useEffect(() => {
    if (containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [logLines]);

  // Filter lines
  const filteredLines = logLines.filter((line) => {
    if (!filter) return true;
    try {
      return new RegExp(filter, "i").test(line);
    } catch {
      return line.toLowerCase().includes(filter.toLowerCase());
    }
  });

  const counts = countSeverities(filteredLines);

  return (
    <div className="flex flex-col gap-4 h-full">
      {/* Header */}
      <div className="flex items-center gap-3 shrink-0">
        <Link href={`/nodes/${nodeId}`}>
          <Button variant="ghost" size="icon">
            <ArrowLeft className="h-4 w-4" />
          </Button>
        </Link>
        <div className="flex-1 min-w-0">
          <h1 className="text-2xl font-bold tracking-tight truncate">
            Log Viewer{node ? ` — ${node.name}` : ""}
          </h1>
          <p className="text-sm text-zinc-400">
            {logLines.length} Zeilen geladen{autoRefresh ? " (Auto-Refresh aktiv)" : ""}
          </p>
        </div>
      </div>

      {/* KPI Bar */}
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4 shrink-0">
        <Card>
          <CardContent className="flex items-center gap-3 p-4">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-red-500/10">
              <AlertCircle className="h-4 w-4 text-red-500" />
            </div>
            <div>
              <p className="text-2xl font-bold text-red-500">{counts.errors}</p>
              <p className="text-xs text-zinc-400">Errors</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-3 p-4">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-yellow-500/10">
              <AlertTriangle className="h-4 w-4 text-yellow-500" />
            </div>
            <div>
              <p className="text-2xl font-bold text-yellow-500">{counts.warnings}</p>
              <p className="text-xs text-zinc-400">Warnings</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-3 p-4">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-orange-500/10">
              <Activity className="h-4 w-4 text-orange-500" />
            </div>
            <div>
              <p className="text-2xl font-bold text-orange-400">{counts.critical}</p>
              <p className="text-xs text-zinc-400">Critical</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-3 p-4">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-blue-500/10">
              <Zap className="h-4 w-4 text-blue-500" />
            </div>
            <div>
              <p className="text-2xl font-bold text-blue-400">{filteredLines.length}</p>
              <p className="text-xs text-zinc-400">Sichtbar</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Controls */}
      <div className="flex flex-wrap items-center gap-3 shrink-0">
        <Select value={logFile} onValueChange={setLogFile}>
          <SelectTrigger className="w-[220px]">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {LOG_FILES.map((f) => (
              <SelectItem key={f.value} value={f.value}>{f.label}</SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select value={String(lineCount)} onValueChange={(v) => setLineCount(Number(v))}>
          <SelectTrigger className="w-[100px]">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {LINE_COUNTS.map((n) => (
              <SelectItem key={n} value={String(n)}>{n} Zeilen</SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Input
          placeholder="Filter (Regex)..."
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          className="w-[200px]"
        />

        <div className="flex items-center gap-2">
          <Switch checked={autoRefresh} onCheckedChange={setAutoRefresh} />
          <span className="text-xs text-zinc-400">Auto-Refresh</span>
        </div>

        <Button variant="outline" size="sm" onClick={fetchLogs} disabled={isLoading}>
          <RefreshCw className={`h-4 w-4 mr-1 ${isLoading ? "animate-spin" : ""}`} />
          Aktualisieren
        </Button>
      </div>

      {/* Error */}
      {error && (
        <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-4 py-2 text-sm text-red-400">
          {error}
        </div>
      )}

      {/* Log Output */}
      <div
        ref={containerRef}
        className="flex-1 overflow-auto bg-zinc-950 rounded-lg border border-zinc-800 p-3 font-mono text-sm min-h-0"
        style={{ minHeight: "300px" }}
      >
        {filteredLines.length === 0 && !isLoading && (
          <div className="flex items-center justify-center h-24 text-zinc-600 text-sm">
            Keine Log-Eintraege gefunden
          </div>
        )}
        {isLoading && filteredLines.length === 0 && (
          <div className="flex items-center justify-center h-24 text-zinc-600 text-sm">
            Lade Logs...
          </div>
        )}
        {filteredLines.map((line, i) => {
          const severity = inferSeverity(line);
          const colorClass = SEVERITY_COLORS[severity] ?? "text-zinc-300";
          const isHighlighted = filter && (() => {
            try { return new RegExp(filter, "i").test(line); }
            catch { return false; }
          })();

          return (
            <div
              key={i}
              className={`px-1 py-0.5 leading-relaxed break-all ${colorClass} ${
                isHighlighted && filter ? "bg-yellow-500/10" : ""
              }`}
            >
              {line}
            </div>
          );
        })}
      </div>
    </div>
  );
}
