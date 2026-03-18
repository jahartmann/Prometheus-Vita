"use client";

import { useMemo, useState } from "react";
import { useNetworkStore } from "@/stores/network-store";
import { networkApi } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer } from "recharts";
import { GitCompare, Clock } from "lucide-react";

interface ScanTimelineProps {
  nodeId: string;
}

interface DiffChange {
  type: "added" | "removed" | "changed";
  description: string;
}

function parseDiff(data: unknown): DiffChange[] {
  if (!data || typeof data !== "object") return [];
  const obj = data as Record<string, unknown>;
  const changes: DiffChange[] = [];

  if (Array.isArray(obj.new_ports)) {
    for (const p of obj.new_ports as number[]) {
      changes.push({ type: "added", description: `Port ${p} neu geöffnet` });
    }
  }
  if (Array.isArray(obj.removed_ports)) {
    for (const p of obj.removed_ports as number[]) {
      changes.push({ type: "removed", description: `Port ${p} geschlossen` });
    }
  }
  if (Array.isArray(obj.new_devices)) {
    for (const d of obj.new_devices as string[]) {
      changes.push({ type: "added", description: `Neues Gerät: ${d}` });
    }
  }
  if (Array.isArray(obj.removed_devices)) {
    for (const d of obj.removed_devices as string[]) {
      changes.push({ type: "removed", description: `Gerät nicht mehr sichtbar: ${d}` });
    }
  }
  if (Array.isArray(obj.changes)) {
    for (const c of obj.changes as Array<Record<string, unknown>>) {
      changes.push({
        type: "changed",
        description: String(c.description ?? c.message ?? JSON.stringify(c).slice(0, 80)),
      });
    }
  }
  return changes;
}

export function ScanTimeline({ nodeId: _nodeId }: ScanTimelineProps) {
  const scans = useNetworkStore((s) => s.scans);

  const [scanA, setScanA] = useState<string>("");
  const [scanB, setScanB] = useState<string>("");
  const [diff, setDiff] = useState<DiffChange[] | null>(null);
  const [diffLoading, setDiffLoading] = useState(false);
  const [diffError, setDiffError] = useState<string | null>(null);

  // Chart data: port count per scan
  const chartData = useMemo(() => {
    return scans
      .slice(0, 20)
      .reverse()
      .map((s) => {
        let portCount = 0;
        if (s.results_json && typeof s.results_json === "object") {
          const obj = s.results_json as Record<string, unknown>;
          if (Array.isArray(obj.ports)) portCount += obj.ports.length;
          if (Array.isArray(obj.nmap_results)) {
            for (const h of obj.nmap_results) {
              const host = h as Record<string, unknown>;
              if (Array.isArray(host.ports)) portCount += host.ports.length;
            }
          }
        }
        return {
          label: new Date(s.started_at).toLocaleString("de-DE", {
            day: "2-digit",
            month: "2-digit",
            hour: "2-digit",
            minute: "2-digit",
          }),
          ports: portCount,
          type: s.scan_type,
        };
      });
  }, [scans]);

  const handleCompare = async () => {
    if (!scanA || !scanB) return;
    setDiffLoading(true);
    setDiffError(null);
    setDiff(null);
    try {
      const res = await networkApi.diffScans(scanA, scanB);
      setDiff(parseDiff(res.data));
    } catch {
      setDiffError("Vergleich fehlgeschlagen.");
    } finally {
      setDiffLoading(false);
    }
  };

  if (scans.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-zinc-500">
        <Clock className="h-8 w-8 mb-3 opacity-30" />
        <p className="text-sm">Noch keine Scan-Historie vorhanden.</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Chart */}
      <Card className="border-zinc-800 bg-zinc-900/60">
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium text-zinc-300">Port-Anzahl über Zeit</CardTitle>
        </CardHeader>
        <CardContent>
          {chartData.length > 1 ? (
            <ResponsiveContainer width="100%" height={160}>
              <BarChart data={chartData} margin={{ top: 4, right: 8, bottom: 4, left: -16 }}>
                <XAxis
                  dataKey="label"
                  tick={{ fontSize: 10, fill: "#71717a" }}
                  interval="preserveStartEnd"
                />
                <YAxis tick={{ fontSize: 10, fill: "#71717a" }} />
                <Tooltip
                  contentStyle={{
                    background: "#18181b",
                    border: "1px solid #3f3f46",
                    borderRadius: "6px",
                    fontSize: "12px",
                  }}
                  labelStyle={{ color: "#a1a1aa" }}
                />
                <Bar dataKey="ports" fill="#3b82f6" radius={[2, 2, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <p className="text-xs text-zinc-600 text-center py-8">Nicht genug Daten für Chart</p>
          )}
        </CardContent>
      </Card>

      {/* Scan list */}
      <div className="space-y-1.5">
        <p className="text-xs font-medium text-zinc-500 uppercase tracking-wide">Letzte Scans</p>
        {scans.slice(0, 15).map((s) => (
          <div
            key={s.id}
            className="flex items-center gap-3 rounded-md border border-zinc-800/50 bg-zinc-900/40 px-3 py-2"
          >
            <Badge
              variant={s.scan_type === "full" ? "default" : "secondary"}
              className={`text-[10px] shrink-0 ${
                s.scan_type === "full"
                  ? "bg-blue-500/20 text-blue-400 border-blue-500/30"
                  : "bg-zinc-700/50 text-zinc-400"
              }`}
            >
              {s.scan_type === "full" ? "Full" : "Quick"}
            </Badge>
            <span className="text-xs text-zinc-400 font-mono">
              {new Date(s.started_at).toLocaleString("de-DE")}
            </span>
            {s.completed_at && (
              <span className="text-xs text-zinc-600">
                Dauer:{" "}
                {Math.round(
                  (new Date(s.completed_at).getTime() - new Date(s.started_at).getTime()) / 1000
                )}
                s
              </span>
            )}
            <span className="text-xs text-zinc-700 font-mono ml-auto">{s.id.slice(0, 8)}…</span>
          </div>
        ))}
      </div>

      {/* Diff comparison */}
      {scans.length >= 2 && (
        <Card className="border-zinc-800 bg-zinc-900/60">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-zinc-300 flex items-center gap-2">
              <GitCompare className="h-4 w-4" />
              Scans vergleichen
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex items-center gap-2 flex-wrap">
              <Select value={scanA} onValueChange={setScanA}>
                <SelectTrigger className="w-52 h-8 text-xs bg-zinc-900 border-zinc-700">
                  <SelectValue placeholder="Scan A wählen..." />
                </SelectTrigger>
                <SelectContent>
                  {scans.map((s) => (
                    <SelectItem key={s.id} value={s.id} className="text-xs">
                      {new Date(s.started_at).toLocaleString("de-DE")} ({s.scan_type})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

              <span className="text-zinc-600 text-xs">vs.</span>

              <Select value={scanB} onValueChange={setScanB}>
                <SelectTrigger className="w-52 h-8 text-xs bg-zinc-900 border-zinc-700">
                  <SelectValue placeholder="Scan B wählen..." />
                </SelectTrigger>
                <SelectContent>
                  {scans.map((s) => (
                    <SelectItem key={s.id} value={s.id} className="text-xs">
                      {new Date(s.started_at).toLocaleString("de-DE")} ({s.scan_type})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

              <Button
                size="sm"
                variant="outline"
                className="h-8 text-xs gap-1.5"
                disabled={!scanA || !scanB || scanA === scanB || diffLoading}
                onClick={handleCompare}
              >
                <GitCompare className="h-3.5 w-3.5" />
                {diffLoading ? "Vergleiche..." : "Vergleichen"}
              </Button>
            </div>

            {diffError && (
              <p className="text-xs text-red-400">{diffError}</p>
            )}

            {diff !== null && (
              <div className="space-y-1 mt-2">
                {diff.length === 0 ? (
                  <p className="text-xs text-zinc-500">Keine Unterschiede gefunden.</p>
                ) : (
                  diff.map((c, i) => (
                    <div key={i} className="flex items-start gap-2 text-xs py-1 border-b border-zinc-800/30 last:border-0">
                      <span
                        className={`shrink-0 font-mono font-bold ${
                          c.type === "added"
                            ? "text-green-400"
                            : c.type === "removed"
                            ? "text-red-400"
                            : "text-yellow-400"
                        }`}
                      >
                        {c.type === "added" ? "+" : c.type === "removed" ? "−" : "~"}
                      </span>
                      <span className="text-zinc-400">{c.description}</span>
                    </div>
                  ))
                )}
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
