"use client";

import { useEffect, useMemo, useState, type ComponentType } from "react";
import { Activity, AlertTriangle, Gauge, Network, Radar, ShieldAlert } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { metricsApi } from "@/lib/api";
import { normalizeNetworkScanResults } from "@/lib/network-scan-normalizer";
import { cn, formatBandwidth, formatTraffic } from "@/lib/utils";
import { useNetworkStore } from "@/stores/network-store";
import type { NetworkSummary } from "@/types/api";

interface NetworkSecurityOverviewProps {
  nodeId: string;
}

interface MetricTileProps {
  icon: ComponentType<{ className?: string }>;
  label: string;
  value: string;
  detail?: string;
  tone?: "default" | "ok" | "warning" | "danger";
}

const toneClasses: Record<NonNullable<MetricTileProps["tone"]>, string> = {
  default: "border-border bg-card text-muted-foreground",
  ok: "border-green-500/20 bg-green-500/5 text-green-400",
  warning: "border-orange-500/25 bg-orange-500/10 text-orange-400",
  danger: "border-red-500/25 bg-red-500/10 text-red-400",
};

function formatScanTime(value?: string): string {
  if (!value) return "Nie";
  return new Date(value).toLocaleString("de-DE", {
    day: "2-digit",
    month: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function latestScanTimestamp(
  scans: Array<{ scan_type: string; started_at: string }>,
  scanType: "quick" | "full"
): string | undefined {
  return scans
    .filter((scan) => scan.scan_type === scanType)
    .sort((a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime())[0]
    ?.started_at;
}

function MetricTile({ icon: Icon, label, value, detail, tone = "default" }: MetricTileProps) {
  return (
    <div className={cn("flex min-w-0 items-center gap-3 rounded-lg border px-3 py-2.5", toneClasses[tone])}>
      <Icon className="h-4 w-4 shrink-0" />
      <div className="min-w-0">
        <p className="text-[10px] uppercase tracking-wide text-muted-foreground">{label}</p>
        <p className="truncate text-sm font-semibold text-foreground">{value}</p>
        {detail && <p className="truncate text-xs text-muted-foreground">{detail}</p>}
      </div>
    </div>
  );
}

export function NetworkSecurityOverview({ nodeId }: NetworkSecurityOverviewProps) {
  const scans = useNetworkStore((state) => state.scans);
  const devices = useNetworkStore((state) => state.devices);
  const anomalies = useNetworkStore((state) => state.anomalies);
  const [summary, setSummary] = useState<NetworkSummary | null>(null);

  useEffect(() => {
    if (!nodeId) {
      setSummary(null);
      return;
    }

    let cancelled = false;
    setSummary(null);

    metricsApi
      .getNodeNetworkSummary(nodeId, "24h")
      .then((res) => {
        const data = res.data;
        if (!cancelled) {
          setSummary(data && typeof data === "object" && "total_in" in data ? data : null);
        }
      })
      .catch(() => {
        if (!cancelled) setSummary(null);
      });

    return () => {
      cancelled = true;
    };
  }, [nodeId]);

  const nodeScans = useMemo(
    () => scans.filter((scan) => scan.node_id === nodeId),
    [scans, nodeId]
  );
  const nodeDevices = useMemo(
    () => devices.filter((device) => device.node_id === nodeId),
    [devices, nodeId]
  );
  const nodeAnomalies = useMemo(
    () => anomalies.filter((anomaly) => anomaly.node_id === nodeId),
    [anomalies, nodeId]
  );
  const scanSummary = useMemo(
    () => normalizeNetworkScanResults(nodeScans[0]?.results_json),
    [nodeScans]
  );

  const lastQuickScan = useMemo(() => latestScanTimestamp(nodeScans, "quick"), [nodeScans]);
  const lastFullScan = useMemo(() => latestScanTimestamp(nodeScans, "full"), [nodeScans]);
  const unacknowledgedAnomalies = nodeAnomalies.filter((anomaly) => !anomaly.is_acknowledged).length;
  const riskTone =
    scanSummary.highRiskCount > 0 ? "danger" : scanSummary.mediumRiskCount > 0 ? "warning" : "ok";
  const trafficRate = summary
    ? formatBandwidth(summary.avg_in_rate + summary.avg_out_rate)
    : "Keine Daten";
  const trafficTotal = summary
    ? `${formatTraffic(summary.total_in + summary.total_out)} in 24h`
    : "Warte auf Metriken";

  return (
    <Card className="border-border bg-card">
      <CardContent className="space-y-3 p-3">
        <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-6">
          <MetricTile
            icon={Radar}
            label="Quick Scan"
            value={formatScanTime(lastQuickScan)}
            detail="Letzter Schnellscan"
          />
          <MetricTile
            icon={ShieldAlert}
            label="Full Scan"
            value={formatScanTime(lastFullScan)}
            detail="Letzter Tiefenscan"
          />
          <MetricTile
            icon={Activity}
            label="Ports"
            value={`${scanSummary.listeningCount} offen`}
            detail={`${scanSummary.connectionCount} Verbindungen`}
          />
          <MetricTile
            icon={AlertTriangle}
            label="Risiko"
            value={`${scanSummary.highRiskCount} hoch / ${scanSummary.mediumRiskCount} mittel`}
            detail={scanSummary.highRiskCount > 0 ? "Sofort prüfen" : "Scan-Auswertung"}
            tone={riskTone}
          />
          <MetricTile
            icon={Network}
            label="Geräte"
            value={`${nodeDevices.length}`}
            detail="Erkannt im Netzwerk"
          />
          <MetricTile
            icon={Gauge}
            label="Traffic 24h"
            value={trafficRate}
            detail={trafficTotal}
          />
        </div>

        {unacknowledgedAnomalies > 0 && (
          <Badge variant="warning" className="gap-1.5">
            <AlertTriangle className="h-3 w-3" />
            {unacknowledgedAnomalies} unbestätigte Anomalie
            {unacknowledgedAnomalies !== 1 ? "n" : ""}
          </Badge>
        )}
      </CardContent>
    </Card>
  );
}
