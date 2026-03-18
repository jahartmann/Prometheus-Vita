"use client";

import { useEffect, useState } from "react";
import { useLogStore } from "@/stores/log-store";
import { useNodeStore } from "@/stores/node-store";
import { useLogStream } from "@/hooks/use-log-stream";
import { LogKpiBar } from "@/components/logs/log-kpi-bar";
import { LogFilterToolbar } from "@/components/logs/log-filter-toolbar";
import { LogStream } from "@/components/logs/log-stream";
import { LogAiPanel } from "@/components/logs/log-ai-panel";
import { LogExportDialog } from "@/components/logs/log-export-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Activity } from "lucide-react";

export default function ClusterLogsPage() {
  const { nodes, fetchNodes } = useNodeStore();

  const fetchSources = useLogStore((s) => s.fetchSources);
  const fetchAnomalies = useLogStore((s) => s.fetchAnomalies);
  const setFilters = useLogStore((s) => s.setFilters);
  const analyze = useLogStore((s) => s.analyze);
  const analysisReport = useLogStore((s) => s.analysisReport);
  const setAnalysisReport = useLogStore((s) => s.setAnalysisReport);

  const [selectedNodeIds, setSelectedNodeIds] = useState<string[]>([]);
  const [autoScroll, setAutoScroll] = useState(true);
  const [exportOpen, setExportOpen] = useState(false);

  // After nodes load, default to all
  useEffect(() => {
    fetchNodes();
  }, [fetchNodes]);

  useEffect(() => {
    if (nodes.length > 0 && selectedNodeIds.length === 0) {
      const allIds = nodes.map((n) => n.id);
      setSelectedNodeIds(allIds);
      setFilters({ nodeIds: allIds });
      allIds.forEach((id) => {
        fetchSources(id);
        fetchAnomalies(id);
      });
    }
  }, [nodes, selectedNodeIds.length, setFilters, fetchSources, fetchAnomalies]);

  const { isConnected } = useLogStream({
    nodeIds: selectedNodeIds,
    enabled: selectedNodeIds.length > 0,
  });

  const toggleNode = (nodeId: string) => {
    setSelectedNodeIds((prev) => {
      const next = prev.includes(nodeId)
        ? prev.filter((id) => id !== nodeId)
        : [...prev, nodeId];
      setFilters({ nodeIds: next });
      return next;
    });
  };

  const handleAnalyze = () => {
    if (selectedNodeIds.length === 0) return;
    const now = new Date();
    const oneHourAgo = new Date(now.getTime() - 60 * 60 * 1000);
    analyze(selectedNodeIds, oneHourAgo.toISOString(), now.toISOString());
  };

  const exportNodeId = selectedNodeIds[0] ?? "";

  return (
    <div className="flex flex-col gap-4 h-full">
      {/* Page Title */}
      <div className="flex items-center gap-3 shrink-0">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-blue-500/10">
          <Activity className="h-5 w-5 text-blue-500" />
        </div>
        <div className="flex-1 min-w-0">
          <h1 className="text-2xl font-bold tracking-tight">
            Cluster Log Viewer
          </h1>
          <p className="text-sm text-muted-foreground flex items-center gap-2">
            Cluster-weiter Echtzeit-Log-Stream
            <span
              className={`inline-block h-2 w-2 rounded-full ${
                isConnected ? "bg-green-500" : "bg-red-500"
              }`}
            />
            {isConnected ? "Verbunden" : "Getrennt"}
          </p>
        </div>
      </div>

      {/* Node multi-select */}
      {nodes.length > 0 && (
        <div className="shrink-0 flex flex-wrap items-center gap-2 rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-2">
          <span className="text-xs text-zinc-500 mr-1">Nodes:</span>
          <Button
            size="sm"
            variant="ghost"
            className="h-6 text-xs text-zinc-400 px-2"
            onClick={() => {
              const allIds = nodes.map((n) => n.id);
              setSelectedNodeIds(allIds);
              setFilters({ nodeIds: allIds });
            }}
          >
            Alle
          </Button>
          <Button
            size="sm"
            variant="ghost"
            className="h-6 text-xs text-zinc-400 px-2"
            onClick={() => {
              setSelectedNodeIds([]);
              setFilters({ nodeIds: [] });
            }}
          >
            Keine
          </Button>
          <div className="w-px h-4 bg-zinc-700" />
          {nodes.map((node) => {
            const isActive = selectedNodeIds.includes(node.id);
            return (
              <button
                key={node.id}
                onClick={() => toggleNode(node.id)}
                className={`rounded px-2 py-0.5 text-xs font-medium transition-colors cursor-pointer ${
                  isActive
                    ? "bg-blue-600 text-blue-100"
                    : "bg-zinc-800/50 text-zinc-500 hover:bg-zinc-700/50"
                }`}
              >
                {node.name}
              </button>
            );
          })}
          {selectedNodeIds.length > 0 && (
            <Badge className="ml-auto bg-zinc-800 text-zinc-400 border-zinc-700 text-[10px]">
              {selectedNodeIds.length} / {nodes.length}
            </Badge>
          )}
        </div>
      )}

      {/* KPI Bar */}
      <div className="shrink-0">
        <LogKpiBar />
      </div>

      {/* Filter Toolbar */}
      <div className="shrink-0">
        <LogFilterToolbar
          nodeId={exportNodeId}
          onAnalyze={handleAnalyze}
          onExport={() => setExportOpen(true)}
          autoScroll={autoScroll}
          onAutoScrollChange={setAutoScroll}
        />
      </div>

      {/* Log Stream + AI Panel */}
      <div className="relative flex flex-col flex-1 min-h-0">
        <LogStream autoScroll={autoScroll} />

        {analysisReport && (
          <LogAiPanel
            report={analysisReport}
            onClose={() => setAnalysisReport(null)}
          />
        )}
      </div>

      {/* Export Dialog */}
      <LogExportDialog
        nodeId={exportNodeId}
        open={exportOpen}
        onOpenChange={setExportOpen}
      />
    </div>
  );
}
