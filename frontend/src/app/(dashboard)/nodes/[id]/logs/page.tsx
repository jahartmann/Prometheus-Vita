"use client";

import { useEffect, useState, useCallback, useRef } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { ArrowLeft, RefreshCw, Search, Play, Pause } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { useNodeStore } from "@/stores/node-store";
import { logApi } from "@/lib/api";

const LOG_FILES = [
  { value: "syslog", label: "Syslog" },
  { value: "auth", label: "Auth" },
  { value: "pveproxy", label: "PVE Proxy" },
  { value: "pvedaemon", label: "PVE Daemon" },
  { value: "pve-firewall", label: "PVE Firewall" },
  { value: "corosync", label: "Corosync" },
  { value: "tasks", label: "Tasks" },
];

const LINE_OPTIONS = [
  { value: "50", label: "50 Zeilen" },
  { value: "100", label: "100 Zeilen" },
  { value: "200", label: "200 Zeilen" },
  { value: "500", label: "500 Zeilen" },
];

export default function NodeLogsPage() {
  const params = useParams<{ id: string }>();
  const nodeId = params.id;
  const { nodes, fetchNodes } = useNodeStore();
  const [logFile, setLogFile] = useState("syslog");
  const [lines, setLines] = useState("100");
  const [logContent, setLogContent] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [filter, setFilter] = useState("");
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const logRef = useRef<HTMLPreElement>(null);

  const node = nodes.find((n) => n.id === nodeId);

  useEffect(() => {
    fetchNodes();
  }, [fetchNodes]);

  const fetchLogs = useCallback(async () => {
    setIsLoading(true);
    try {
      const response = await logApi.getLogs(nodeId, logFile, Number(lines));
      const data = response.data;
      setLogContent(typeof data?.lines === "string" ? data.lines : "");
    } catch (error) {
      console.error("Failed to fetch logs:", error);
      setLogContent("Fehler beim Laden der Logs.");
    } finally {
      setIsLoading(false);
    }
  }, [nodeId, logFile, lines]);

  // Initial fetch and when params change
  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  // Auto-refresh
  useEffect(() => {
    if (autoRefresh) {
      intervalRef.current = setInterval(fetchLogs, 5000);
    } else if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [autoRefresh, fetchLogs]);

  // Auto-scroll to bottom when content changes
  useEffect(() => {
    if (logRef.current) {
      logRef.current.scrollTop = logRef.current.scrollHeight;
    }
  }, [logContent]);

  const logLines = logContent.split("\n");
  const filteredLines = filter
    ? logLines.map((line, idx) => ({ line, idx, match: line.toLowerCase().includes(filter.toLowerCase()) }))
    : logLines.map((line, idx) => ({ line, idx, match: false }));

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Link href={`/nodes/${nodeId}`}>
          <Button variant="ghost" size="icon">
            <ArrowLeft className="h-4 w-4" />
          </Button>
        </Link>
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            Logs {node ? `- ${node.name}` : ""}
          </h1>
          <p className="text-muted-foreground">System-Logs anzeigen und durchsuchen.</p>
        </div>
      </div>

      {/* Controls */}
      <div className="flex flex-wrap items-center gap-3">
        <Select value={logFile} onValueChange={setLogFile}>
          <SelectTrigger className="w-48">
            <SelectValue placeholder="Log-Datei" />
          </SelectTrigger>
          <SelectContent>
            {LOG_FILES.map((f) => (
              <SelectItem key={f.value} value={f.value}>{f.label}</SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select value={lines} onValueChange={setLines}>
          <SelectTrigger className="w-36">
            <SelectValue placeholder="Zeilen" />
          </SelectTrigger>
          <SelectContent>
            {LINE_OPTIONS.map((o) => (
              <SelectItem key={o.value} value={o.value}>{o.label}</SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Button variant="outline" size="sm" onClick={fetchLogs} disabled={isLoading}>
          <RefreshCw className={`mr-2 h-3 w-3 ${isLoading ? "animate-spin" : ""}`} />
          Aktualisieren
        </Button>

        <Button
          variant={autoRefresh ? "default" : "outline"}
          size="sm"
          onClick={() => setAutoRefresh(!autoRefresh)}
        >
          {autoRefresh ? (
            <>
              <Pause className="mr-2 h-3 w-3" />
              Auto-Refresh an
            </>
          ) : (
            <>
              <Play className="mr-2 h-3 w-3" />
              Auto-Refresh
            </>
          )}
        </Button>

        {autoRefresh && (
          <Badge variant="outline" className="gap-1">
            Alle 5s
          </Badge>
        )}
      </div>

      {/* Filter */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          className="pl-9"
          placeholder="Logs filtern..."
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
        />
      </div>

      {/* Log Output */}
      <Card>
        <CardHeader className="py-3">
          <CardTitle className="flex items-center gap-2 text-sm">
            <span>{LOG_FILES.find((f) => f.value === logFile)?.label || logFile}</span>
            <Badge variant="outline" className="text-xs font-normal">
              {filteredLines.length} Zeilen
            </Badge>
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          <pre
            ref={logRef}
            className="max-h-[600px] overflow-auto bg-zinc-950 p-4 font-mono text-xs leading-relaxed text-zinc-300"
          >
            {filteredLines.map(({ line, idx, match }) => (
              <div
                key={idx}
                className={`flex ${filter && match ? "bg-yellow-500/20" : ""} ${filter && !match && filter.length > 0 ? "opacity-30" : ""}`}
              >
                <span className="mr-4 inline-block w-10 select-none text-right text-zinc-600">
                  {idx + 1}
                </span>
                <span className="flex-1 whitespace-pre-wrap break-all">{line}</span>
              </div>
            ))}
          </pre>
        </CardContent>
      </Card>
    </div>
  );
}
