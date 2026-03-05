"use client";

import { useEffect, useMemo } from "react";
import {
  GitCompare,
  FileWarning,
  AlertTriangle,
  CheckCircle,
  Brain,
  Sparkles,
  Shield,
} from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useDriftStore } from "@/stores/drift-store";
import { useNodeStore } from "@/stores/node-store";
import { DriftList } from "@/components/drift/drift-list";
import { NodeComparison } from "@/components/drift/node-comparison";

export default function DriftPage() {
  const { checks, isLoading, fetchAll } = useDriftStore();
  const { nodes, fetchNodes } = useNodeStore();

  useEffect(() => {
    fetchAll();
    fetchNodes();
  }, [fetchAll, fetchNodes]);

  const nodeNames: Record<string, string> = {};
  for (const node of nodes) {
    nodeNames[node.id] = node.name;
  }

  const completedChecks = checks.filter((c) => c.status === "completed");
  const withDrift = completedChecks.filter(
    (c) => c.changed_files + c.added_files + c.removed_files > 0
  );
  const failedChecks = checks.filter((c) => c.status === "failed");

  // AI severity distribution
  const severityDistribution = useMemo(() => {
    let critical = 0;
    let warning = 0;
    let ok = 0;
    let analyzed = 0;
    let latestAnalysisTime: string | null = null;

    for (const check of completedChecks) {
      if (check.ai_analysis) {
        analyzed++;
        const sev = check.ai_analysis.overall_severity;
        if (sev >= 7) critical++;
        else if (sev >= 4) warning++;
        else ok++;

        if (!latestAnalysisTime || check.ai_analysis.analyzed_at > latestAnalysisTime) {
          latestAnalysisTime = check.ai_analysis.analyzed_at;
        }
      }
    }

    // Top risk items
    const topRisks = completedChecks
      .filter((c) => c.ai_analysis && c.ai_analysis.overall_severity >= 7)
      .sort((a, b) => (b.ai_analysis?.overall_severity || 0) - (a.ai_analysis?.overall_severity || 0))
      .slice(0, 3);

    return { critical, warning, ok, analyzed, latestAnalysisTime, topRisks };
  }, [completedChecks]);

  const totalSeverityChecks = severityDistribution.critical + severityDistribution.warning + severityDistribution.ok;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Drift Detection</h1>
        <p className="text-muted-foreground">
          Konfigurationsabweichungen zwischen Backups und aktuellem Zustand.
        </p>
      </div>

      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardContent className="flex items-center gap-4 p-4">
            <GitCompare className="h-8 w-8 text-blue-500" />
            <div>
              <p className="text-2xl font-bold">{checks.length}</p>
              <p className="text-sm text-muted-foreground">Checks gesamt</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-4 p-4">
            <CheckCircle className="h-8 w-8 text-green-500" />
            <div>
              <p className="text-2xl font-bold">{completedChecks.length - withDrift.length}</p>
              <p className="text-sm text-muted-foreground">Ohne Drift</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-4 p-4">
            <AlertTriangle className="h-8 w-8 text-yellow-500" />
            <div>
              <p className="text-2xl font-bold">{withDrift.length}</p>
              <p className="text-sm text-muted-foreground">Mit Drift</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-4 p-4">
            <FileWarning className="h-8 w-8 text-red-500" />
            <div>
              <p className="text-2xl font-bold">{failedChecks.length}</p>
              <p className="text-sm text-muted-foreground">Fehlgeschlagen</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* AI Severity Distribution Card */}
      {severityDistribution.analyzed > 0 && (
        <Card className="border-violet-500/20">
          <CardContent className="p-4">
            <div className="flex items-center gap-2 mb-3">
              <Brain className="h-5 w-5 text-violet-500" />
              <h3 className="text-sm font-semibold text-violet-600">KI-Analyse Uebersicht</h3>
              <Sparkles className="h-4 w-4 text-violet-400" />
              {severityDistribution.latestAnalysisTime && (
                <span className="ml-auto text-xs text-muted-foreground">
                  Letzte KI-Analyse: {new Date(severityDistribution.latestAnalysisTime).toLocaleString("de-DE")}
                </span>
              )}
            </div>

            <div className="grid gap-4 md:grid-cols-2">
              {/* Severity Distribution Bar */}
              <div>
                <p className="text-xs text-muted-foreground mb-2">Severity-Verteilung ({severityDistribution.analyzed} analysiert)</p>
                <div className="flex h-6 w-full rounded-lg overflow-hidden">
                  {severityDistribution.critical > 0 && (
                    <div
                      className="bg-red-500 flex items-center justify-center text-white text-xs font-medium"
                      style={{ width: `${(severityDistribution.critical / totalSeverityChecks) * 100}%` }}
                    >
                      {severityDistribution.critical}
                    </div>
                  )}
                  {severityDistribution.warning > 0 && (
                    <div
                      className="bg-amber-500 flex items-center justify-center text-white text-xs font-medium"
                      style={{ width: `${(severityDistribution.warning / totalSeverityChecks) * 100}%` }}
                    >
                      {severityDistribution.warning}
                    </div>
                  )}
                  {severityDistribution.ok > 0 && (
                    <div
                      className="bg-green-500 flex items-center justify-center text-white text-xs font-medium"
                      style={{ width: `${(severityDistribution.ok / totalSeverityChecks) * 100}%` }}
                    >
                      {severityDistribution.ok}
                    </div>
                  )}
                </div>
                <div className="flex justify-between mt-1.5">
                  <span className="text-xs flex items-center gap-1">
                    <span className="w-2.5 h-2.5 rounded-full bg-red-500 inline-block" />
                    Kritisch ({severityDistribution.critical})
                  </span>
                  <span className="text-xs flex items-center gap-1">
                    <span className="w-2.5 h-2.5 rounded-full bg-amber-500 inline-block" />
                    Warnung ({severityDistribution.warning})
                  </span>
                  <span className="text-xs flex items-center gap-1">
                    <span className="w-2.5 h-2.5 rounded-full bg-green-500 inline-block" />
                    OK ({severityDistribution.ok})
                  </span>
                </div>
              </div>

              {/* Top Risks */}
              <div>
                <p className="text-xs text-muted-foreground mb-2">Top Risiken</p>
                {severityDistribution.topRisks.length === 0 ? (
                  <div className="flex items-center gap-2 text-sm text-green-600">
                    <Shield className="h-4 w-4" />
                    <span>Keine kritischen Risiken erkannt</span>
                  </div>
                ) : (
                  <div className="space-y-1.5">
                    {severityDistribution.topRisks.map((check) => (
                      <div
                        key={check.id}
                        className="flex items-center gap-2 text-xs rounded-md bg-red-500/5 border border-red-500/10 p-1.5"
                      >
                        <AlertTriangle className="h-3 w-3 text-red-500 shrink-0" />
                        <span className="font-medium">
                          {nodeNames[check.node_id] || check.node_id.slice(0, 8)}
                        </span>
                        <span className="text-muted-foreground truncate">
                          {check.ai_analysis?.overall_summary?.slice(0, 80)}...
                        </span>
                        <span className="ml-auto font-mono text-red-500 shrink-0">
                          {check.ai_analysis?.overall_severity}/10
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      <Tabs defaultValue="checks">
        <TabsList>
          <TabsTrigger value="checks">Drift-Checks</TabsTrigger>
          <TabsTrigger value="comparison">Node-Vergleich</TabsTrigger>
        </TabsList>

        <TabsContent value="checks">
          <Card>
            <CardContent className="p-0">
              <DriftList checks={checks} nodeNames={nodeNames} isLoading={isLoading} />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="comparison">
          <NodeComparison nodes={nodes} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
