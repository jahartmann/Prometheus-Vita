"use client";

import { useEffect, useState } from "react";
import { AlertTriangle, CheckCircle, RefreshCw, FileWarning, Brain, Sparkles } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useDriftStore } from "@/stores/drift-store";
import { DriftDetailDialog } from "./drift-detail-dialog";
import type { DriftCheck } from "@/types/api";

interface DriftSummaryCardProps {
  nodeId: string;
}

export function DriftSummaryCard({ nodeId }: DriftSummaryCardProps) {
  const { nodeChecks, fetchByNode, triggerCheck } = useDriftStore();
  const [checking, setChecking] = useState(false);
  const [selectedCheck, setSelectedCheck] = useState<DriftCheck | null>(null);

  const checks = nodeChecks[nodeId] || [];
  const latest = checks.length > 0 ? checks[0] : null;

  useEffect(() => {
    fetchByNode(nodeId);
  }, [nodeId, fetchByNode]);

  const handleCheck = async () => {
    setChecking(true);
    try {
      await triggerCheck(nodeId);
      setTimeout(() => fetchByNode(nodeId), 2000);
    } finally {
      setChecking(false);
    }
  };

  const hasDrift = latest && (latest.changed_files > 0 || latest.added_files > 0 || latest.removed_files > 0);
  const aiAnalysis = latest?.ai_analysis;

  return (
    <>
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center gap-2">
              <FileWarning className="h-4 w-4" />
              <span className="font-medium text-sm">Konfigurationsdrift</span>
            </div>
            <Button variant="outline" size="sm" onClick={handleCheck} disabled={checking}>
              <RefreshCw className={`h-3 w-3 mr-1 ${checking ? "animate-spin" : ""}`} />
              Prüfen
            </Button>
          </div>

          {!latest ? (
            <p className="text-sm text-muted-foreground">Noch kein Drift-Check durchgeführt.</p>
          ) : latest.status === "running" || latest.status === "pending" ? (
            <p className="text-sm text-muted-foreground">Check läuft...</p>
          ) : latest.status === "failed" ? (
            <div className="flex items-center gap-2 text-destructive text-sm">
              <AlertTriangle className="h-4 w-4" />
              <span>Fehler: {latest.error_message || "Unbekannt"}</span>
            </div>
          ) : hasDrift ? (
            <div
              className="space-y-2 cursor-pointer"
              onClick={() => setSelectedCheck(latest)}
            >
              <div className="flex items-center gap-2">
                <AlertTriangle className="h-4 w-4 text-yellow-500" />
                <span className="text-sm font-medium text-yellow-600">Drift erkannt</span>
              </div>
              <div className="flex gap-2 flex-wrap">
                {latest.changed_files > 0 && (
                  <Badge variant="secondary">{latest.changed_files} geändert</Badge>
                )}
                {latest.added_files > 0 && (
                  <Badge variant="secondary">{latest.added_files} hinzugefügt</Badge>
                )}
                {latest.removed_files > 0 && (
                  <Badge variant="secondary">{latest.removed_files} entfernt</Badge>
                )}
              </div>

              {/* AI Analysis Summary */}
              {aiAnalysis && (
                <div className="mt-2 rounded-md border border-violet-500/20 bg-gradient-to-r from-violet-500/5 to-purple-500/5 p-2">
                  <div className="flex items-center gap-1.5 mb-1">
                    <Brain className="h-3 w-3 text-violet-500" />
                    <span className="text-xs font-medium text-violet-600">KI-Analyse</span>
                    <Sparkles className="h-2.5 w-2.5 text-violet-400" />
                    <Badge
                      variant="outline"
                      className={`ml-auto text-xs font-mono ${
                        aiAnalysis.overall_severity >= 7
                          ? "bg-red-500/10 text-red-600 border-red-500/20"
                          : aiAnalysis.overall_severity >= 4
                          ? "bg-amber-500/10 text-amber-600 border-amber-500/20"
                          : "bg-green-500/10 text-green-600 border-green-500/20"
                      }`}
                    >
                      {aiAnalysis.overall_severity}/10
                    </Badge>
                  </div>
                  <p className="text-xs text-violet-700 dark:text-violet-300 line-clamp-2">
                    {aiAnalysis.overall_summary}
                  </p>
                </div>
              )}

              <p className="text-xs text-muted-foreground">
                {new Date(latest.checked_at).toLocaleString("de-DE")} - {latest.total_files} Dateien geprüft
              </p>
            </div>
          ) : (
            <div className="flex items-center gap-2 text-sm">
              <CheckCircle className="h-4 w-4 text-green-500" />
              <span className="text-green-600">Kein Drift erkannt</span>
              <span className="text-muted-foreground text-xs ml-auto">
                {new Date(latest.checked_at).toLocaleString("de-DE")}
              </span>
            </div>
          )}
        </CardContent>
      </Card>

      <DriftDetailDialog
        check={selectedCheck}
        open={!!selectedCheck}
        onOpenChange={(open) => !open && setSelectedCheck(null)}
      />
    </>
  );
}
