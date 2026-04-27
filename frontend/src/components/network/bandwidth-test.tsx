"use client";

import { useMemo, useState } from "react";
import { ArrowLeftRight, Gauge, Loader2, Play } from "lucide-react";
import { toast } from "sonner";
import { bandwidthApi, getApiErrorMessage } from "@/lib/api";
import { useNodeStore } from "@/stores/node-store";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";

interface BandwidthResult {
  source_node_id: string;
  target_node_id: string;
  target_host: string;
  duration_sec: number;
  protocol: "tcp" | "udp" | string;
  direction: "send" | "reverse" | string;
  bits_per_second: number;
  bytes_total: number;
  retransmits?: number;
  started_at: string;
  completed_at: string;
  warnings?: string[];
}

function formatBitrate(bps: number): string {
  if (!Number.isFinite(bps) || bps <= 0) return "—";
  const units = ["bit/s", "kbit/s", "Mbit/s", "Gbit/s", "Tbit/s"];
  let i = 0;
  let v = bps;
  while (v >= 1000 && i < units.length - 1) {
    v /= 1000;
    i++;
  }
  return `${v.toFixed(v >= 100 ? 0 : v >= 10 ? 1 : 2)} ${units[i]}`;
}

function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return "—";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let i = 0;
  let v = bytes;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i++;
  }
  return `${v.toFixed(v >= 100 ? 0 : 1)} ${units[i]}`;
}

interface BandwidthTestProps {
  sourceNodeId: string;
}

export function BandwidthTest({ sourceNodeId }: BandwidthTestProps) {
  const { nodes } = useNodeStore();
  const [targetId, setTargetId] = useState<string>("");
  const [targetHost, setTargetHost] = useState<string>("");
  const [duration, setDuration] = useState<number>(5);
  const [protocol, setProtocol] = useState<"tcp" | "udp">("tcp");
  const [reverse, setReverse] = useState<boolean>(false);
  const [running, setRunning] = useState<boolean>(false);
  const [result, setResult] = useState<BandwidthResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  const sourceNode = useMemo(
    () => nodes.find((n) => n.id === sourceNodeId),
    [nodes, sourceNodeId]
  );
  const targetNode = useMemo(
    () => nodes.find((n) => n.id === targetId),
    [nodes, targetId]
  );

  const runTest = async () => {
    if (!targetId) {
      toast.error("Bitte einen Ziel-Node auswählen");
      return;
    }
    setRunning(true);
    setError(null);
    setResult(null);
    try {
      const res = (await bandwidthApi.run(sourceNodeId, {
        target_node_id: targetId,
        target_host: targetHost.trim() || undefined,
        duration_sec: duration,
        protocol,
        reverse,
      })) as BandwidthResult;
      setResult(res);
      toast.success(`Bandbreite gemessen: ${formatBitrate(res.bits_per_second)}`);
    } catch (e: unknown) {
      const msg = getApiErrorMessage(e, "Bandbreitentest fehlgeschlagen");
      setError(msg);
      toast.error(msg);
    } finally {
      setRunning(false);
    }
  };

  const otherNodes = nodes.filter((n) => n.id !== sourceNodeId);

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start gap-3">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-emerald-500/10">
            <Gauge className="h-4 w-4 text-emerald-500" />
          </div>
          <div>
            <CardTitle className="text-base">Bandbreiten-Messung (iperf3)</CardTitle>
            <CardDescription>
              Misst aktiv den Durchsatz zwischen zwei Nodes via iperf3 über SSH. iperf3 muss auf
              beiden Seiten installiert sein.
            </CardDescription>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-3 lg:grid-cols-[1fr_auto_1fr]">
          <div className="space-y-1.5">
            <Label className="text-xs uppercase tracking-wide text-muted-foreground">Quelle</Label>
            <div className="flex h-9 items-center rounded-md border bg-muted/40 px-3 text-sm font-medium">
              {sourceNode?.name ?? sourceNodeId}
            </div>
          </div>
          <div className="flex items-end justify-center pb-1">
            <ArrowLeftRight
              className={`h-5 w-5 ${reverse ? "text-blue-400" : "text-muted-foreground"}`}
            />
          </div>
          <div className="space-y-1.5">
            <Label className="text-xs uppercase tracking-wide text-muted-foreground">Ziel</Label>
            <Select value={targetId} onValueChange={setTargetId}>
              <SelectTrigger>
                <SelectValue placeholder="Ziel-Node auswählen" />
              </SelectTrigger>
              <SelectContent>
                {otherNodes.length === 0 ? (
                  <SelectItem value="__none__" disabled>
                    Mindestens 2 Nodes nötig
                  </SelectItem>
                ) : (
                  otherNodes.map((n) => (
                    <SelectItem key={n.id} value={n.id}>
                      {n.name}
                    </SelectItem>
                  ))
                )}
              </SelectContent>
            </Select>
          </div>
        </div>

        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <div className="space-y-1.5">
            <Label className="text-xs uppercase tracking-wide text-muted-foreground">Dauer (s)</Label>
            <Input
              type="number"
              min={1}
              max={60}
              value={duration}
              onChange={(e) => setDuration(Math.max(1, Math.min(60, Number(e.target.value) || 5)))}
            />
          </div>
          <div className="space-y-1.5">
            <Label className="text-xs uppercase tracking-wide text-muted-foreground">Protokoll</Label>
            <Select value={protocol} onValueChange={(v: "tcp" | "udp") => setProtocol(v)}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="tcp">TCP</SelectItem>
                <SelectItem value="udp">UDP</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <Label className="text-xs uppercase tracking-wide text-muted-foreground">
              Ziel-Host (optional)
            </Label>
            <Input
              placeholder={targetNode?.hostname || "wird aus Ziel-Node gezogen"}
              value={targetHost}
              onChange={(e) => setTargetHost(e.target.value)}
            />
          </div>
          <div className="space-y-1.5">
            <Label className="text-xs uppercase tracking-wide text-muted-foreground">Richtung</Label>
            <div className="flex h-9 items-center gap-2 rounded-md border px-3 text-sm">
              <Switch checked={reverse} onCheckedChange={setReverse} />
              <span className="text-muted-foreground">
                {reverse ? "Ziel → Quelle" : "Quelle → Ziel"}
              </span>
            </div>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <Button onClick={runTest} disabled={running || !targetId}>
            {running ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                Misst…
              </>
            ) : (
              <>
                <Play className="h-4 w-4" />
                Test starten
              </>
            )}
          </Button>
          {running && (
            <span className="text-xs text-muted-foreground">
              Läuft ~{duration}s — Ergebnis erscheint danach.
            </span>
          )}
        </div>

        {error && (
          <div className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
            {error}
          </div>
        )}

        {result && (
          <div className="space-y-3 rounded-lg border bg-muted/30 p-4">
            <div className="flex items-baseline gap-3">
              <span className="text-3xl font-bold tracking-tight">
                {formatBitrate(result.bits_per_second)}
              </span>
              <Badge variant="outline">{result.protocol.toUpperCase()}</Badge>
              <Badge variant="secondary">{result.direction === "reverse" ? "↩︎" : "→"}</Badge>
            </div>
            <div className="grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
              <ResultStat label="Übertragen" value={formatBytes(result.bytes_total)} />
              <ResultStat label="Dauer" value={`${result.duration_sec}s`} />
              <ResultStat label="Ziel-Host" value={result.target_host} mono />
              <ResultStat
                label="Retransmits"
                value={result.retransmits != null ? String(result.retransmits) : "—"}
              />
            </div>
            {result.warnings && result.warnings.length > 0 && (
              <div className="rounded-md border border-amber-300/40 bg-amber-300/10 px-3 py-2 text-xs text-amber-200">
                {result.warnings.map((w, i) => (
                  <div key={i}>• {w}</div>
                ))}
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function ResultStat({
  label,
  value,
  mono,
}: {
  label: string;
  value: string;
  mono?: boolean;
}) {
  return (
    <div>
      <div className="text-[10px] font-medium uppercase tracking-wide text-muted-foreground">
        {label}
      </div>
      <div className={`text-sm font-medium ${mono ? "font-mono" : ""}`}>{value}</div>
    </div>
  );
}
