"use client";

import { useNetworkStore } from "@/stores/network-store";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { ShieldAlert, CheckCircle2 } from "lucide-react";

interface AnomalyListProps {
  nodeId: string;
}

function riskBadge(score: number) {
  if (score >= 0.8)
    return <Badge className="bg-red-500/20 text-red-400 border-red-500/30">Kritisch ({(score * 100).toFixed(0)}%)</Badge>;
  if (score >= 0.6)
    return <Badge className="bg-orange-500/20 text-orange-400 border-orange-500/30">Hoch ({(score * 100).toFixed(0)}%)</Badge>;
  if (score >= 0.3)
    return <Badge className="bg-yellow-500/20 text-yellow-400 border-yellow-500/30">Mittel ({(score * 100).toFixed(0)}%)</Badge>;
  return <Badge className="bg-green-500/20 text-green-400 border-green-500/30">Niedrig ({(score * 100).toFixed(0)}%)</Badge>;
}

function formatAnomalyType(type: string): string {
  return type
    .split("_")
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(" ");
}

function renderDetails(details: unknown): string {
  if (!details) return "";
  if (typeof details === "string") return details;
  try {
    const obj = details as Record<string, unknown>;
    const parts: string[] = [];
    if (obj.message) parts.push(String(obj.message));
    if (obj.description) parts.push(String(obj.description));
    if (obj.new_ports && Array.isArray(obj.new_ports))
      parts.push(`Neue Ports: ${(obj.new_ports as number[]).join(", ")}`);
    if (obj.removed_ports && Array.isArray(obj.removed_ports))
      parts.push(`Entfernte Ports: ${(obj.removed_ports as number[]).join(", ")}`);
    if (obj.new_device)
      parts.push(`Neues Gerät: ${String(obj.new_device)}`);
    return parts.join(" · ") || JSON.stringify(details).slice(0, 120);
  } catch {
    return "";
  }
}

export function NetworkAnomalyList({ nodeId: _nodeId }: AnomalyListProps) {
  const anomalies = useNetworkStore((s) => s.anomalies);
  const acknowledgeAnomaly = useNetworkStore((s) => s.acknowledgeAnomaly);

  if (anomalies.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-zinc-500">
        <ShieldAlert className="h-8 w-8 mb-3 opacity-30" />
        <p className="text-sm">Keine Netzwerk-Anomalien erkannt.</p>
      </div>
    );
  }

  const unacked = anomalies.filter((a) => !a.is_acknowledged);
  const acked = anomalies.filter((a) => a.is_acknowledged);

  return (
    <div className="space-y-3">
      {unacked.length > 0 && (
        <div className="space-y-2">
          <p className="text-xs font-medium text-zinc-500 uppercase tracking-wide">
            Aktiv ({unacked.length})
          </p>
          {unacked.map((a) => (
            <Card key={a.id} className="border-zinc-800 bg-zinc-900/60">
              <CardContent className="py-3 px-4">
                <div className="flex items-start gap-3">
                  <ShieldAlert className="h-4 w-4 mt-0.5 text-orange-400 shrink-0" />
                  <div className="flex-1 min-w-0 space-y-1.5">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="text-sm font-medium text-zinc-200">
                        {formatAnomalyType(a.anomaly_type)}
                      </span>
                      {riskBadge(a.risk_score)}
                      <span className="text-xs text-zinc-600 ml-auto">
                        {new Date(a.created_at).toLocaleString("de-DE")}
                      </span>
                    </div>
                    {a.details_json != null && (
                      <p className="text-xs text-zinc-500 truncate">
                        {renderDetails(a.details_json)}
                      </p>
                    )}
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-7 text-xs mt-1 gap-1.5"
                      onClick={() => acknowledgeAnomaly(a.id)}
                    >
                      <CheckCircle2 className="h-3 w-3" />
                      Als normal markieren
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {acked.length > 0 && (
        <div className="space-y-2">
          <p className="text-xs font-medium text-zinc-500 uppercase tracking-wide">
            Bestätigt ({acked.length})
          </p>
          {acked.map((a) => (
            <div
              key={a.id}
              className="flex items-center gap-3 rounded-md border border-zinc-800/50 bg-zinc-900/30 px-4 py-2.5 opacity-50"
            >
              <CheckCircle2 className="h-3.5 w-3.5 text-zinc-600 shrink-0" />
              <span className="text-xs text-zinc-500 line-through">
                {formatAnomalyType(a.anomaly_type)}
              </span>
              <span className="text-[10px] text-zinc-700 ml-auto">
                {new Date(a.created_at).toLocaleString("de-DE")}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
