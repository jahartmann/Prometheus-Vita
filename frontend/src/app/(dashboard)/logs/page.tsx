"use client";

import { useEffect, useState, useCallback, useRef } from "react";
import { useNodeStore } from "@/stores/node-store";
import { logApi } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Activity, AlertCircle, AlertTriangle, RefreshCw, Zap } from "lucide-react";

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

export default function ClusterLogsPage() {
  const { nodes, fetchNodes } = useNodeStore();
  const [selectedNodeIds, setSelectedNodeIds] = useState<string[]>([]);
  const [logFile, setLogFile] = useState("syslog");
  const [lineCount] = useState(100);
  const [nodeLogs, setNodeLogs] = useState<NodeLogData[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [filter, setFilter] = useState("");
  const containerRef = useRef<HTMLDivElement>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => { fetchNodes(); }, [fetchNodes]);

  // Default to all nodes
  useEffect(() => {
    if (nodes.length > 0 && selectedNodeIds.length === 0) {
      setSelectedNodeIds(nodes.map((n) => n.id));
    }
  }, [nodes, selectedNodeIds.length]);

  const fetchAllLogs = useCallback(async () => {
    if (selectedNodeIds.length === 0) return;
    setIsLoading(true);
    try {
      const results = await Promise.allSettled(
        selectedNodeIds.map(async (nodeId) => {
          const node = nodes.find((n) => n.id === nodeId);
          const res = await logApi.getLogs(nodeId, logFile, lineCount);
          const raw = typeof res.data === "string" ? res.data : (res.data?.lines || res.data || "");
          const lines = String(raw).split("\n").filter((l: string) => l.trim());
          return { nodeId, nodeName: node?.name || nodeId, lines };
        })
      );
      const data: NodeLogData[] = results
        .filter((r): r is PromiseFulfilledResult<NodeLogData> => r.status === "fulfilled")
        .map((r) => r.value);
      setNodeLogs(data);
    } finally {
      setIsLoading(false);
    }
  }, [selectedNodeIds, nodes, logFile, lineCount]);

  useEffect(() => { fetchAllLogs(); }, [fetchAllLogs]);

  useEffect(() => {
    if (autoRefresh) {
      intervalRef.current = setInterval(fetchAllLogs, 5000);
    }
    return () => { if (intervalRef.current) clearInterval(intervalRef.current); };
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

  return (
    <div className="flex flex-col gap-4 h-full">
      {/* Header */}
      <div className="flex items-center gap-3 shrink-0">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-blue-500/10">
          <Activity className="h-5 w-5 text-blue-500" />
        </div>
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Cluster Log Viewer</h1>
          <p className="text-sm text-zinc-400">
            {filteredLines.length} Zeilen von {selectedNodeIds.length} Nodes
          </p>
        </div>
      </div>

      {/* Node Selector */}
      {nodes.length > 0 && (
        <div className="shrink-0 flex flex-wrap items-center gap-2 rounded-lg border border-zinc-800 bg-zinc-900/50 px-3 py-2">
          <span className="text-xs text-zinc-500 mr-1">Nodes:</span>
          <Button size="sm" variant="ghost" className="h-6 text-xs text-zinc-400 px-2"
            onClick={() => setSelectedNodeIds(nodes.map((n) => n.id))}>Alle</Button>
          <Button size="sm" variant="ghost" className="h-6 text-xs text-zinc-400 px-2"
            onClick={() => setSelectedNodeIds([])}>Keine</Button>
          <div className="w-px h-4 bg-zinc-700" />
          {nodes.map((node) => (
            <button key={node.id} onClick={() => toggleNode(node.id)}
              className={`rounded px-2 py-0.5 text-xs font-medium transition-colors cursor-pointer ${
                selectedNodeIds.includes(node.id)
                  ? "bg-blue-600 text-blue-100"
                  : "bg-zinc-800/50 text-zinc-500 hover:bg-zinc-700/50"
              }`}>{node.name}</button>
          ))}
          <Badge className="ml-auto bg-zinc-800 text-zinc-400 border-zinc-700 text-[10px]">
            {selectedNodeIds.length} / {nodes.length}
          </Badge>
        </div>
      )}

      {/* KPIs */}
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4 shrink-0">
        <Card><CardContent className="flex items-center gap-3 p-4">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-red-500/10">
            <AlertCircle className="h-4 w-4 text-red-500" /></div>
          <div><p className="text-2xl font-bold text-red-500">{counts.errors}</p>
          <p className="text-xs text-zinc-400">Errors</p></div>
        </CardContent></Card>
        <Card><CardContent className="flex items-center gap-3 p-4">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-yellow-500/10">
            <AlertTriangle className="h-4 w-4 text-yellow-500" /></div>
          <div><p className="text-2xl font-bold text-yellow-500">{counts.warnings}</p>
          <p className="text-xs text-zinc-400">Warnings</p></div>
        </CardContent></Card>
        <Card><CardContent className="flex items-center gap-3 p-4">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-orange-500/10">
            <Activity className="h-4 w-4 text-orange-500" /></div>
          <div><p className="text-2xl font-bold text-orange-400">{counts.critical}</p>
          <p className="text-xs text-zinc-400">Critical</p></div>
        </CardContent></Card>
        <Card><CardContent className="flex items-center gap-3 p-4">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-blue-500/10">
            <Zap className="h-4 w-4 text-blue-500" /></div>
          <div><p className="text-2xl font-bold text-blue-400">{filteredLines.length}</p>
          <p className="text-xs text-zinc-400">Sichtbar</p></div>
        </CardContent></Card>
      </div>

      {/* Controls */}
      <div className="flex flex-wrap items-center gap-3 shrink-0">
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
          <span className="text-xs text-zinc-400">Auto-Refresh</span>
        </div>
        <Button variant="outline" size="sm" onClick={fetchAllLogs} disabled={isLoading}>
          <RefreshCw className={`h-4 w-4 mr-1 ${isLoading ? "animate-spin" : ""}`} />
          Aktualisieren
        </Button>
      </div>

      {/* Log Output */}
      <div ref={containerRef}
        className="flex-1 overflow-auto bg-zinc-950 rounded-lg border border-zinc-800 p-3 font-mono text-sm min-h-0"
        style={{ minHeight: "300px" }}>
        {filteredLines.length === 0 && !isLoading && (
          <div className="flex items-center justify-center h-24 text-zinc-600 text-sm">
            Keine Log-Einträge gefunden
          </div>
        )}
        {isLoading && filteredLines.length === 0 && (
          <div className="flex items-center justify-center h-24 text-zinc-600 text-sm">
            Lade Logs...
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
    </div>
  );
}
