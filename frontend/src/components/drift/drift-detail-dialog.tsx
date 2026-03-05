"use client";

import { Brain, Sparkles, Shield, AlertTriangle, CheckCircle2, Eye, EyeOff } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { DriftCheck, DriftFileDetail, AIFileAnalysis } from "@/types/api";
import { useDriftStore } from "@/stores/drift-store";

interface DriftDetailDialogProps {
  check: DriftCheck | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const statusLabel: Record<string, string> = {
  added: "Hinzugefuegt",
  removed: "Entfernt",
  modified: "Geaendert",
  unchanged: "Unveraendert",
};

const statusColor: Record<string, "default" | "secondary" | "outline"> = {
  added: "default",
  removed: "secondary",
  modified: "outline",
};

const categoryColors: Record<string, string> = {
  Security: "bg-red-500/10 text-red-600 border-red-500/20",
  Performance: "bg-purple-500/10 text-purple-600 border-purple-500/20",
  Network: "bg-blue-500/10 text-blue-600 border-blue-500/20",
  Configuration: "bg-orange-500/10 text-orange-600 border-orange-500/20",
  Cosmetic: "bg-gray-500/10 text-gray-600 border-gray-500/20",
};

const recommendationConfig: Record<string, { label: string; className: string; icon: React.ReactNode }> = {
  fix: {
    label: "Beheben",
    className: "bg-red-500/10 text-red-600 border-red-500/20 hover:bg-red-500/20",
    icon: <AlertTriangle className="h-3 w-3" />,
  },
  accept: {
    label: "Akzeptieren",
    className: "bg-green-500/10 text-green-600 border-green-500/20 hover:bg-green-500/20",
    icon: <CheckCircle2 className="h-3 w-3" />,
  },
  monitor: {
    label: "Beobachten",
    className: "bg-yellow-500/10 text-yellow-600 border-yellow-500/20 hover:bg-yellow-500/20",
    icon: <Eye className="h-3 w-3" />,
  },
};

function SeverityBadge({ severity }: { severity: number }) {
  let color = "bg-green-500/10 text-green-600 border-green-500/20";
  if (severity >= 7) {
    color = "bg-red-500/10 text-red-600 border-red-500/20";
  } else if (severity >= 4) {
    color = "bg-amber-500/10 text-amber-600 border-amber-500/20";
  }

  return (
    <Badge variant="outline" className={`${color} font-mono`}>
      {severity}/10
    </Badge>
  );
}

function AIAnalysisCard({ analysis }: { analysis: AIFileAnalysis }) {
  const rec = recommendationConfig[analysis.recommendation] || recommendationConfig.monitor;
  const catColor = categoryColors[analysis.category] || categoryColors.Configuration;

  return (
    <div className="mt-3 rounded-lg border border-violet-500/20 bg-gradient-to-r from-violet-500/5 via-purple-500/5 to-fuchsia-500/5 p-3">
      <div className="flex items-center gap-1.5 mb-2">
        <Brain className="h-3.5 w-3.5 text-violet-500" />
        <span className="text-xs font-medium text-violet-600">KI-Analyse</span>
        <Sparkles className="h-3 w-3 text-violet-400" />
      </div>

      <div className="flex flex-wrap gap-2 mb-2">
        <SeverityBadge severity={analysis.severity} />
        <Badge variant="outline" className={catColor}>
          {analysis.category}
        </Badge>
        <Badge variant="outline" className={rec.className}>
          <span className="flex items-center gap-1">
            {rec.icon}
            {rec.label}
          </span>
        </Badge>
      </div>

      <p className="text-xs text-muted-foreground mb-1">
        <strong>Risiko:</strong> {analysis.risk_assessment}
      </p>

      <p className="text-xs text-violet-700 dark:text-violet-300 bg-violet-500/10 rounded px-2 py-1">
        {analysis.summary}
      </p>

      {analysis.severity_reason && (
        <p className="text-xs text-muted-foreground mt-1">
          <strong>Begruendung:</strong> {analysis.severity_reason}
        </p>
      )}
    </div>
  );
}

export function DriftDetailDialog({ check, open, onOpenChange }: DriftDetailDialogProps) {
  const { acceptBaseline, ignoreDrift } = useDriftStore();

  if (!check) return null;

  const details: DriftFileDetail[] = Array.isArray(check.details)
    ? check.details
    : [];

  const filtered = details.filter((d) => d.status !== "unchanged");
  const aiAnalysis = check.ai_analysis;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            Drift-Details
            {aiAnalysis && (
              <Badge variant="outline" className="bg-violet-500/10 text-violet-600 border-violet-500/20">
                <Brain className="h-3 w-3 mr-1" />
                KI-analysiert
              </Badge>
            )}
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-2 text-sm mb-4">
          <p className="text-muted-foreground">
            Geprueft: {new Date(check.checked_at).toLocaleString("de-DE")} |{" "}
            {check.total_files} Dateien total
          </p>
          <div className="flex gap-2">
            {check.changed_files > 0 && <Badge variant="outline">{check.changed_files} geaendert</Badge>}
            {check.added_files > 0 && <Badge variant="default">{check.added_files} hinzugefuegt</Badge>}
            {check.removed_files > 0 && <Badge variant="secondary">{check.removed_files} entfernt</Badge>}
          </div>
          {check.baseline_updated_at && (
            <p className="text-xs text-muted-foreground">
              Baseline aktualisiert: {new Date(check.baseline_updated_at).toLocaleString("de-DE")}
            </p>
          )}
        </div>

        {/* Overall AI Analysis Summary */}
        {aiAnalysis && (
          <div className="rounded-lg border border-violet-500/20 bg-gradient-to-r from-violet-500/5 via-purple-500/5 to-fuchsia-500/5 p-4 mb-4">
            <div className="flex items-center gap-2 mb-2">
              <div className="flex items-center gap-1.5">
                <Brain className="h-4 w-4 text-violet-500" />
                <span className="text-sm font-semibold text-violet-600">KI-Gesamtanalyse</span>
                <Sparkles className="h-3.5 w-3.5 text-violet-400" />
              </div>
              <SeverityBadge severity={aiAnalysis.overall_severity} />
            </div>

            <p className="text-sm text-violet-700 dark:text-violet-300">
              {aiAnalysis.overall_summary}
            </p>

            <p className="text-xs text-muted-foreground mt-2">
              Analysiert: {new Date(aiAnalysis.analyzed_at).toLocaleString("de-DE")} | Modell: {aiAnalysis.model}
            </p>
          </div>
        )}

        {/* Accept Baseline Button */}
        {filtered.length > 0 && (
          <div className="flex gap-2 mb-4">
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                acceptBaseline(check.id);
                onOpenChange(false);
              }}
              className="bg-green-500/10 text-green-600 border-green-500/20 hover:bg-green-500/20"
            >
              <CheckCircle2 className="h-3.5 w-3.5 mr-1" />
              Alle als Baseline akzeptieren
            </Button>
          </div>
        )}

        {filtered.length === 0 ? (
          <p className="text-muted-foreground text-sm">Keine geaenderten Dateien.</p>
        ) : (
          <div className="space-y-3">
            {filtered.map((file) => (
              <div
                key={file.file_path}
                className={`border rounded-lg p-3 ${file.acknowledged ? "opacity-60" : ""}`}
              >
                <div className="flex items-center justify-between mb-1">
                  <code className="text-xs font-mono">{file.file_path}</code>
                  <div className="flex items-center gap-2">
                    {file.acknowledged && (
                      <Badge variant="outline" className="bg-gray-500/10 text-gray-500 border-gray-500/20">
                        <EyeOff className="h-3 w-3 mr-1" />
                        Ignoriert
                      </Badge>
                    )}
                    <Badge variant={statusColor[file.status] || "outline"}>
                      {statusLabel[file.status] || file.status}
                    </Badge>
                  </div>
                </div>

                {/* Per-file action buttons */}
                {!file.acknowledged && (
                  <div className="flex gap-2 mt-2 mb-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-6 text-xs text-muted-foreground hover:text-foreground"
                      onClick={() => ignoreDrift(check.id, file.file_path)}
                    >
                      <EyeOff className="h-3 w-3 mr-1" />
                      Ignorieren
                    </Button>
                  </div>
                )}

                {file.diff && (
                  <pre className="mt-2 text-xs bg-muted p-2 rounded overflow-x-auto whitespace-pre-wrap">
                    {file.diff}
                  </pre>
                )}

                {/* AI Analysis for this file */}
                {file.ai_file_analysis && (
                  <AIAnalysisCard analysis={file.ai_file_analysis} />
                )}
              </div>
            ))}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
