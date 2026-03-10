"use client";

import { useEffect, useState } from "react";
import { anomalyApi, predictionApi } from "@/lib/api";
import { useNodeStore } from "@/stores/node-store";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { AlertTriangle, Info, AlertCircle, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { AnomalyRecord, MaintenancePrediction } from "@/types/api";

interface TimelineEntry {
  id: string;
  type: "anomaly" | "prediction";
  nodeId: string;
  nodeName: string;
  metric: string;
  value: number;
  severity: string;
  timestamp: string;
  detail: string;
}

const severityConfig: Record<string, { color: string; icon: React.ComponentType<{ className?: string }> }> = {
  critical: { color: "destructive", icon: AlertCircle },
  warning: { color: "warning", icon: AlertTriangle },
  info: { color: "secondary", icon: Info },
};

export function AlertHistory() {
  const { nodes } = useNodeStore();
  const [entries, setEntries] = useState<TimelineEntry[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  const nodeMap = Object.fromEntries(nodes.map((n) => [n.id, n.name]));

  const fetchData = async () => {
    setIsLoading(true);
    try {
      const [anomaliesRaw, predictionsRaw] = await Promise.all([
        anomalyApi.listUnresolved(),
        predictionApi.listCritical(),
      ]);

      const anomalies: AnomalyRecord[] = Array.isArray(anomaliesRaw) ? (anomaliesRaw as AnomalyRecord[]) : [];
      const predictions: MaintenancePrediction[] = Array.isArray(predictionsRaw) ? (predictionsRaw as MaintenancePrediction[]) : [];

      const anomalyEntries: TimelineEntry[] = anomalies.map((a) => ({
        id: a.id,
        type: "anomaly",
        nodeId: a.node_id,
        nodeName: nodeMap[a.node_id] || a.node_id.slice(0, 8),
        metric: a.metric,
        value: a.value,
        severity: a.severity,
        timestamp: a.detected_at,
        detail: `Z-Score: ${a.z_score.toFixed(2)} | Mittelwert: ${a.mean.toFixed(2)} | Stddev: ${a.stddev.toFixed(2)}`,
      }));

      const predictionEntries: TimelineEntry[] = predictions.map((p) => ({
        id: p.id,
        type: "prediction",
        nodeId: p.node_id,
        nodeName: nodeMap[p.node_id] || p.node_id.slice(0, 8),
        metric: p.metric,
        value: p.current_value,
        severity: p.severity,
        timestamp: p.predicted_at,
        detail: `Prognose: ${p.predicted_value.toFixed(1)} | Schwellenwert: ${p.threshold.toFixed(1)}${p.days_until_threshold != null ? ` | Tage bis Schwellenwert: ${p.days_until_threshold}` : ""}`,
      }));

      const combined = [...anomalyEntries, ...predictionEntries].sort(
        (a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
      );

      setEntries(combined);
    } catch (error) {
      console.error("Failed to fetch alert history:", error);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [nodes]);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          {entries.length} aktive Meldungen
        </p>
        <Button variant="outline" size="sm" onClick={fetchData} disabled={isLoading}>
          <RefreshCw className={`mr-2 h-3 w-3 ${isLoading ? "animate-spin" : ""}`} />
          Aktualisieren
        </Button>
      </div>

      {entries.length === 0 && !isLoading ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Info className="mb-3 h-10 w-10 text-muted-foreground" />
            <p className="text-muted-foreground">Keine aktiven Anomalien oder Vorhersagen.</p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-2">
          {entries.map((entry) => {
            const config = severityConfig[entry.severity] || severityConfig.info;
            const SeverityIcon = config.icon;
            const badgeVariant = config.color as "destructive" | "secondary" | "outline" | "default";

            return (
              <Card key={`${entry.type}-${entry.id}`}>
                <CardContent className="flex items-start gap-3 py-3">
                  <SeverityIcon className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
                  <div className="flex-1 space-y-1">
                    <div className="flex items-center gap-2">
                      <Badge variant={badgeVariant} className="text-xs">
                        {entry.severity}
                      </Badge>
                      <Badge variant="outline" className="text-xs">
                        {entry.type === "anomaly" ? "Anomalie" : "Vorhersage"}
                      </Badge>
                      <span className="text-xs text-muted-foreground">{entry.nodeName}</span>
                    </div>
                    <p className="text-sm font-medium">
                      {entry.metric}: {entry.value.toFixed(2)}
                    </p>
                    <p className="text-xs text-muted-foreground">{entry.detail}</p>
                  </div>
                  <span className="shrink-0 text-xs text-muted-foreground">
                    {new Date(entry.timestamp).toLocaleString("de-DE", {
                      day: "2-digit",
                      month: "2-digit",
                      hour: "2-digit",
                      minute: "2-digit",
                    })}
                  </span>
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}
