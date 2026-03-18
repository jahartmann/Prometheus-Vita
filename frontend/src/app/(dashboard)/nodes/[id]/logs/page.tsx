"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useLogStore } from "@/stores/log-store";
import { useNodeStore } from "@/stores/node-store";
import { useLogStream } from "@/hooks/use-log-stream";
import { LogKpiBar } from "@/components/logs/log-kpi-bar";
import { LogFilterToolbar } from "@/components/logs/log-filter-toolbar";
import { LogStream } from "@/components/logs/log-stream";
import { LogAiPanel } from "@/components/logs/log-ai-panel";
import { LogExportDialog } from "@/components/logs/log-export-dialog";

export default function NodeLogsPage() {
  const params = useParams<{ id: string }>();
  const nodeId = params.id;

  const { nodes, fetchNodes } = useNodeStore();
  const node = nodes.find((n) => n.id === nodeId);

  const fetchSources = useLogStore((s) => s.fetchSources);
  const fetchAnomalies = useLogStore((s) => s.fetchAnomalies);
  const analyze = useLogStore((s) => s.analyze);
  const analysisReport = useLogStore((s) => s.analysisReport);
  const setAnalysisReport = useLogStore((s) => s.setAnalysisReport);

  const [autoScroll, setAutoScroll] = useState(true);
  const [exportOpen, setExportOpen] = useState(false);

  const { isConnected } = useLogStream({ nodeIds: [nodeId] });

  useEffect(() => {
    fetchNodes();
    fetchSources(nodeId);
    fetchAnomalies(nodeId);
  }, [nodeId, fetchNodes, fetchSources, fetchAnomalies]);

  const handleAnalyze = () => {
    const now = new Date();
    const oneHourAgo = new Date(now.getTime() - 60 * 60 * 1000);
    analyze([nodeId], oneHourAgo.toISOString(), now.toISOString());
  };

  return (
    <div className="flex flex-col gap-4 h-full">
      {/* Page Title */}
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
          <p className="text-sm text-muted-foreground flex items-center gap-2">
            Echtzeit-Log-Stream
            <span
              className={`inline-block h-2 w-2 rounded-full ${
                isConnected ? "bg-green-500" : "bg-red-500"
              }`}
            />
            {isConnected ? "Verbunden" : "Getrennt"}
          </p>
        </div>
      </div>

      {/* KPI Bar */}
      <div className="shrink-0">
        <LogKpiBar />
      </div>

      {/* Filter Toolbar */}
      <div className="shrink-0">
        <LogFilterToolbar
          nodeId={nodeId}
          onAnalyze={handleAnalyze}
          onExport={() => setExportOpen(true)}
          autoScroll={autoScroll}
          onAutoScrollChange={setAutoScroll}
        />
      </div>

      {/* Log Stream + AI Panel */}
      <div className="relative flex flex-col flex-1 min-h-0">
        <LogStream autoScroll={autoScroll} />

        {/* AI Panel — slides up over log stream */}
        {analysisReport && (
          <LogAiPanel
            report={analysisReport}
            onClose={() => setAnalysisReport(null)}
          />
        )}
      </div>

      {/* Export Dialog */}
      <LogExportDialog
        nodeId={nodeId}
        open={exportOpen}
        onOpenChange={setExportOpen}
      />
    </div>
  );
}
