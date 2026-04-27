"use client";

import { useEffect, useState, useCallback, useRef } from "react";
import { useNodeStore } from "@/stores/node-store";
import { getApiErrorMessage, logApi } from "@/lib/api";
import { PageShell } from "@/components/layout/page-shell";
import { KpiCard } from "@/components/ui/kpi-card";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { EmptyState } from "@/components/ui/empty-state";
import { Skeleton } from "@/components/ui/skeleton";
import { Activity, AlertCircle, AlertTriangle, Loader2, RefreshCw, Server, Terminal, Zap } from "lucide-react";

const LOG_FILES = [
  { value: "syslog", label: "/var/log/syslog" },
  { value: "auth.log", label: "/var/log/auth.log" },
  { value: "pveproxy", label: "/var/log/pveproxy/access.log" },
  { value: "pvedaemon", label: "/var/log/pvedaemon.log" },
  { value: "pve-firewall", label: "/var/log/pve-firewall.log" },
  { value: "corosync", label: "/var/log/corosync/corosync.log" },
];

function inferSeverity(line: string): string {
  const lower = line.toLowerCase();
  if (lower.includes("emerg") || lower.includes("panic") || lower.includes("fatal")) return "critical";
  if (lower.includes("error") || lower.includes("err]") || lower.includes("fail")) return "error";
  if (lower.includes("warn")) return "warning";
  if (lower.includes("debug")) return "debug";
  return "info";
}

const SEVERITY_COLORS: Record<string, string> = {
  critical: "text-red-500 animate-pulse font-bold",
  error: "text-red-400",
  warning: "text-yellow-400",
  info: "text-zinc-300",
  debug: "text-zinc-500",
};

interface NodeLogData {
  nodeId: string;
  nodeName: string;
  lines: string[];
}

function normalizeLogLines(data: unknown): string[] {
  if (Array.isArray(data)) return data.map(String).filter((line) => line.trim());
  if (data && typeof data === "object" && "lines" in data) {
    return normalizeLogLines((data as { lines?: unknown }).lines);
  }
  return String(data ?? "")
    .split("\n")
    .filter((line) => line.trim());
}

export default function ClusterLogsPage() {
  const { nodes, fetchNodes, isLoading: isLoadingNodes, error: nodeError } = useNodeStore();
  const [selectedNodeIds, setSelectedNodeIds] = useState<string[]>([]);
  const [logFile, setLogFile] = useState("syslog");
  const [lineCount] = useState(100);
  const [nodeLogs, setNodeLogs] = useState<NodeLogData[]>([]);
  const [loadErrors, setLoadErrors] = useState<Record<string, string>>({});
  const [isLoading, setIsLoading] = useState(false);
  const [hasLoaded, setHasLoaded] = useState(false);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [filter, setFilter] = useState("");
  const containerRef = useRef<HTMLDivElement>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const requestSeqRef = useRef(0);
  const initializedSelectionRef = useRef(false);

  useEffect(() => { fetchNodes(); }, [fetchNodes]);

  // Default to all nodes
  useEffect(() => {
    if (nodes.length > 0 && !initializedSelectionRef.current) {
      initializedSelectionRef.current = true;
      setSelectedNodeIds(nodes.map((n) => n.id));
    }
  }, [nodes]);

  const fetchAllLogs = useCallback(async () => {
    if (selectedNodeIds.length === 0) {
      setNodeLogs([]);
      setLoadErrors({});
      setHasLoaded(true);
      return;
    }
    const requestSeq = requestSeqRef.current + 1;
    requestSeqRef.current = requestSeq;
    setIsLoading(true);
    try {
      const results = await Promise.allSettled(
        selectedNodeIds.map(async (nodeId) => {
          const node = nodes.find((n) => n.id === nodeId);
          const res = await logApi.getLogs(nodeId, logFile, lineCount);
          const lines = normalizeLogLines(res.data);
          return { nodeId, nodeName: node?.name || nodeId, lines };
        })
      );
      if (requestSeqRef.current !== requestSeq) return;
      const data: NodeLogData[] = results
        .filter((r): r is PromiseFulfilledResult<NodeLogData> => r.status === "fulfilled")
        .map((r) => r.value);
      const errors: Record<string, string> = {};
      results.forEach((result, index) => {
        if (result.status === "rejected") {
          const nodeId = selectedNodeIds[index];
          errors[nodeId] = getApiErrorMessage(result.reason, "Logs konnten nicht geladen werden");
        }
      });
      setNodeLogs(data);
      setLoadErrors(errors);
    } finally {
      if (requestSeqRef.current === requestSeq) {
        setLastUpdated(new Date());
        setHasLoaded(true);
        setIsLoading(false);
      }
    }
  }, [selectedNodeIds, nodes, logFile, lineCount]);

  useEffect(() => { fetchAllLogs(); }, [fetchAllLogs]);

  useEffect(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
    if (autoRefresh) {
      intervalRef.current = setInterval(fetchAllLogs, 5000);
    }
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  }, [autoRefresh, fetchAllLogs]);

  useEffect(() => {
    if (containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [nodeLogs]);

  const toggleNode = (id: string) => {
    setSelectedNodeIds((prev) =>
      prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]
    );
  };

  // Merge and sort all log lines
  const allLines = nodeLogs.flatMap((nl) =>
    nl.lines.map((line) => ({ nodeName: nl.nodeName, line }))
  );

  const filteredLines = allLines.filter(({ line }) => {
    if (!filter) return true;
    try { return new RegExp(filter, "i").test(line); }
    catch { return line.toLowerCase().includes(filter.toLowerCase()); }
  });

  const counts = { errors: 0, warnings: 0, critical: 0 };
  for (const { line } of filteredLines) {
    const sev = inferSeverity(line);
    if (sev === "error") counts.errors++;
    else if (sev === "warning") counts.warnings++;
    else if (sev === "critical") counts.critical++;
  }

  const loadErrorEntries = Object.entries(loadErrors);
  const isInitialLoading = isLoading && !hasLoaded;
  const isRefreshing = isLoading && hasLoaded;

  return (
    <PageShell
      title="Logs"
      eyebrow="Operations"
      description="Clusterweite Log-Sicht mit Filter, Auto-Refresh und sichtbaren Ladefehlern."
      className="h-full min-h-0"
    >

      {/* Node Selector */}
      <div className="flex flex-wrap items-center gap-2 rounded-lg border bg-card px-3 py-2">
        <span className="mr-1 flex items-center gap-1 text-xs font-medium text-muted-foreground">
          <Server className="h-3.5 w-3.5" />
          Nodes:
        </span>
        {isLoadingNodes && nodes.length === 0 ? (
          <>
            <Skeleton className="h-6 w-14" />
            <Skeleton className="h-6 w-14" />
            <div className="h-4 w-px bg-border" />
            {Array.from({ length: 8 }).map((_, index) => (
              <Skeleton key={index} className="h-6 w-24" />
            ))}
          </>
        ) : nodes.length > 0 ? (
          <>
          <Button size="sm" variant="ghost" className="h-6 px-2 text-xs text-muted-foreground"
            onClick={() => setSelectedNodeIds(nodes.map((n) => n.id))}>Alle</Button>
          <Button size="sm" variant="ghost" className="h-6 px-2 text-xs text-muted-foreground"
            onClick={() => setSelectedNodeIds([])}>Keine</Button>
          <div className="h-4 w-px bg-border" />
          {nodes.map((node) => (
            <button key={node.id} onClick={() => toggleNode(node.id)}
              className={`cursor-pointer rounded-md px-2 py-0.5 text-xs font-medium transition-colors ${
                selectedNodeIds.includes(node.id)
                  ? "bg-primary text-primary-foreground shadow-sm"
                  : "bg-muted text-muted-foreground hover:bg-accent hover:text-foreground"
              }`}>{node.name}</button>
          ))}
          <Badge variant="secondary" className="ml-auto text-[10px]">
            {selectedNodeIds.length} / {nodes.length}
          </Badge>
          </>
        ) : (
          <span className="text-xs text-muted-foreground">Keine Nodes verfuegbar.</span>
        )}
      </div>

      {nodeError && (
        <div className="flex items-start gap-2 rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
          <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
          <span>{nodeError}</span>
        </div>
      )}

      {/* KPIs */}
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <KpiCard title="Errors" value={counts.errors} subtitle="Gefilterte Zeilen" icon={AlertCircle} color={counts.errors > 0 ? "red" : "neutral"} />
        <KpiCard title="Warnings" value={counts.warnings} subtitle="Gefilterte Zeilen" icon={AlertTriangle} color="orange" />
        <KpiCard title="Critical" value={counts.critical} subtitle="Gefilterte Zeilen" icon={Activity} color={counts.critical > 0 ? "red" : "neutral"} />
        <KpiCard title="Sichtbar" value={filteredLines.length} subtitle="Log-Zeilen" icon={Zap} color="blue" />
      </div>

      {/* Controls */}
      <div className="flex flex-wrap items-center gap-3 shrink-0 rounded-lg border bg-card px-3 py-2">
        <Select value={logFile} onValueChange={setLogFile}>
          <SelectTrigger className="w-[220px]"><SelectValue /></SelectTrigger>
          <SelectContent>
            {LOG_FILES.map((f) => (
              <SelectItem key={f.value} value={f.value}>{f.label}</SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Input placeholder="Filter (Regex)..." value={filter}
          onChange={(e) => setFilter(e.target.value)} className="w-[200px]" />
        <div className="flex items-center gap-2">
          <Switch checked={autoRefresh} onCheckedChange={setAutoRefresh} />
          <span className="text-xs text-muted-foreground">Auto-Refresh</span>
          {autoRefresh && <Badge variant="secondary" className="text-[10px]">5s</Badge>}
        </div>
        <Button variant="outline" size="sm" onClick={fetchAllLogs} disabled={isLoading}>
          <RefreshCw className={`h-4 w-4 mr-1 ${isLoading ? "animate-spin" : ""}`} />
          Aktualisieren
        </Button>
        <div className="ml-auto flex items-center gap-2 text-xs text-muted-foreground">
          {isRefreshing && (
            <span className="flex items-center gap-1">
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
              aktualisiere
            </span>
          )}
          <span>
            {lastUpdated
              ? `Stand ${lastUpdated.toLocaleTimeString("de-DE", { hour: "2-digit", minute: "2-digit", second: "2-digit" })}`
              : "Noch nicht geladen"}
          </span>
        </div>
      </div>

      {loadErrorEntries.length > 0 && (
        <div className="flex flex-col gap-2 rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {loadErrorEntries.map(([nodeId, message]) => {
            const nodeName = nodes.find((node) => node.id === nodeId)?.name ?? nodeId;
            return (
              <div key={nodeId} className="flex items-start gap-2">
                <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
                <span><span className="font-medium">{nodeName}:</span> {message}</span>
              </div>
            );
          })}
        </div>
      )}

      {/* Log Output */}
      <div ref={containerRef}
        className="relative min-h-[360px] flex-1 overflow-auto rounded-lg border bg-zinc-950 p-3 font-mono text-sm shadow-inner">
        {isRefreshing && filteredLines.length > 0 && (
          <div className="sticky top-0 z-10 mb-2 inline-flex items-center gap-2 rounded-full border border-zinc-700 bg-zinc-900/95 px-3 py-1 text-xs text-zinc-300 shadow">
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
            Neue Logs werden geladen
          </div>
        )}
        {isInitialLoading && (
          <div className="space-y-2">
            {Array.from({ length: 14 }).map((_, index) => (
              <Skeleton key={index} className="h-4 w-full bg-zinc-800/80" />
            ))}
          </div>
        )}
        {!isInitialLoading && selectedNodeIds.length === 0 && (
          <EmptyState
            icon={Server}
            title="Keine Node ausgewaehlt"
            description="Waehle mindestens eine Node, damit Logs geladen werden."
          />
        )}
        {!isInitialLoading && selectedNodeIds.length > 0 && filteredLines.length === 0 && (
          <div className="flex h-48 items-center justify-center text-zinc-500">
            <div className="text-center">
              <Terminal className="mx-auto mb-2 h-6 w-6" />
              <p className="text-sm">Keine Log-Eintraege gefunden</p>
              <p className="mt-1 text-xs">Filter, Log-Datei oder Node-Auswahl pruefen.</p>
            </div>
          </div>
        )}
        {filteredLines.map(({ nodeName, line }, i) => {
          const severity = inferSeverity(line);
          const colorClass = SEVERITY_COLORS[severity] ?? "text-zinc-300";
          return (
            <div key={i} className={`px-1 py-0.5 leading-relaxed break-all ${colorClass}`}>
              <span className="text-zinc-600 mr-2">[{nodeName}]</span>
              {line}
            </div>
          );
        })}
      </div>
    </PageShell>
  );
}
