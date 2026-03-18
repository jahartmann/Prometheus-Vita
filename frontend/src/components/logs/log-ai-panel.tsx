"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { X, Brain, AlertCircle, ListChecks, Lightbulb, Wrench } from "lucide-react";

interface LogAnomaly {
  id: string;
  node_id: string;
  timestamp: string;
  source: string;
  severity: string;
  anomaly_score: number;
  category: string;
  summary: string;
  raw_log: string;
  is_acknowledged: boolean;
  created_at: string;
}

interface LogAnalysisReport {
  summary: string;
  anomalies: LogAnomaly[];
  patterns: Array<{
    pattern: string;
    occurrences: number;
    severity: string;
    description: string;
  }>;
  root_cause_hypotheses: string[];
  recommendations: string[];
  time_range: { from: string; to: string };
  nodes_analyzed: string[];
  model_used: string;
}

interface LogAiPanelProps {
  report: LogAnalysisReport | null;
  onClose: () => void;
}

export function LogAiPanel({ report, onClose }: LogAiPanelProps) {
  if (!report) return null;

  return (
    <div
      className="absolute bottom-0 left-0 right-0 z-20 transition-transform duration-300 ease-in-out"
      style={{ transform: report ? "translateY(0)" : "translateY(100%)" }}
    >
      <Card className="rounded-b-none border-b-0 border-zinc-700 bg-zinc-900 shadow-xl max-h-80 flex flex-col">
        <CardHeader className="shrink-0 flex-row items-center justify-between py-3 px-4">
          <CardTitle className="flex items-center gap-2 text-sm">
            <Brain className="h-4 w-4 text-blue-400" />
            KI-Analyse
            {report.model_used && (
              <span className="text-xs text-zinc-500">({report.model_used})</span>
            )}
          </CardTitle>
          <Button
            size="icon"
            variant="ghost"
            onClick={onClose}
            className="h-6 w-6 text-zinc-400 hover:text-zinc-100"
          >
            <X className="h-4 w-4" />
          </Button>
        </CardHeader>

        <CardContent className="overflow-auto px-4 pb-4 space-y-4 text-sm">
          {/* Summary */}
          {report.summary && (
            <div>
              <p className="text-zinc-300 leading-relaxed">{report.summary}</p>
            </div>
          )}

          {/* Anomalies count */}
          {report.anomalies.length > 0 && (
            <div className="flex items-center gap-2">
              <AlertCircle className="h-4 w-4 text-orange-400 shrink-0" />
              <span className="text-zinc-400">
                {report.anomalies.length} Anomalie
                {report.anomalies.length !== 1 ? "n" : ""} erkannt
              </span>
              <div className="flex flex-wrap gap-1">
                {report.anomalies.slice(0, 5).map((a) => (
                  <Badge
                    key={a.id}
                    className="bg-orange-500/20 text-orange-400 border-orange-500/30 text-[10px]"
                  >
                    {a.category}
                  </Badge>
                ))}
                {report.anomalies.length > 5 && (
                  <span className="text-xs text-zinc-500">
                    +{report.anomalies.length - 5} weitere
                  </span>
                )}
              </div>
            </div>
          )}

          {/* Patterns */}
          {report.patterns.length > 0 && (
            <div>
              <div className="flex items-center gap-1.5 mb-2 text-zinc-400 font-medium">
                <ListChecks className="h-3.5 w-3.5" />
                Muster
              </div>
              <div className="space-y-1">
                {report.patterns.slice(0, 5).map((p, i) => (
                  <div key={i} className="flex items-start gap-2 text-xs">
                    <Badge
                      className={`shrink-0 text-[10px] ${
                        p.severity === "critical" || p.severity === "error"
                          ? "bg-red-500/20 text-red-400 border-red-500/30"
                          : p.severity === "warning"
                          ? "bg-yellow-500/20 text-yellow-400 border-yellow-500/30"
                          : "bg-zinc-700/50 text-zinc-400 border-zinc-600"
                      }`}
                    >
                      {p.occurrences}x
                    </Badge>
                    <span className="text-zinc-400">{p.description || p.pattern}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Root Cause Hypotheses */}
          {report.root_cause_hypotheses.length > 0 && (
            <div>
              <div className="flex items-center gap-1.5 mb-2 text-zinc-400 font-medium">
                <Lightbulb className="h-3.5 w-3.5 text-yellow-400" />
                Ursachen-Hypothesen
              </div>
              <ul className="space-y-1">
                {report.root_cause_hypotheses.map((h, i) => (
                  <li key={i} className="flex items-start gap-2 text-xs text-zinc-400">
                    <span className="text-zinc-600 shrink-0">{i + 1}.</span>
                    {h}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {/* Recommendations */}
          {report.recommendations.length > 0 && (
            <div>
              <div className="flex items-center gap-1.5 mb-2 text-zinc-400 font-medium">
                <Wrench className="h-3.5 w-3.5 text-blue-400" />
                Empfehlungen
              </div>
              <ul className="space-y-1">
                {report.recommendations.map((r, i) => (
                  <li key={i} className="flex items-start gap-2 text-xs text-zinc-400">
                    <span className="text-zinc-600 shrink-0">•</span>
                    {r}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
