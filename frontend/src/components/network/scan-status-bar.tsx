"use client";

import { useNetworkStore } from "@/stores/network-store";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Loader2, Zap, Search } from "lucide-react";

interface ScanStatusBarProps {
  nodeId: string;
}

function relativeTime(isoStr: string): string {
  const diff = Math.floor((Date.now() - new Date(isoStr).getTime()) / 1000);
  if (diff < 60) return `vor ${diff}s`;
  if (diff < 3600) return `vor ${Math.floor(diff / 60)} Min.`;
  if (diff < 86400) return `vor ${Math.floor(diff / 3600)}h`;
  return `vor ${Math.floor(diff / 86400)}d`;
}

export function ScanStatusBar({ nodeId }: ScanStatusBarProps) {
  const scans = useNetworkStore((s) => s.scans);
  const scanStatus = useNetworkStore((s) => s.scanStatus);
  const triggerScan = useNetworkStore((s) => s.triggerScan);

  const nodeScans = Array.isArray(scans)
    ? scans.filter((scan) => scan.node_id === nodeId)
    : [];
  const lastQuick = nodeScans.find((scan) => scan.scan_type === "quick")?.started_at;
  const lastFull = nodeScans.find((scan) => scan.scan_type === "full")?.started_at;
  const isScanning = scanStatus.isScanning && scanStatus.scanningNodeId === nodeId;

  return (
    <div className="flex flex-wrap items-center gap-4 rounded-lg border border-zinc-800 bg-zinc-900/60 px-4 py-3">
      {/* Scan info */}
      <div className="flex items-center gap-6 flex-1 min-w-0">
        <div className="flex flex-col">
          <div className="flex items-center gap-2">
            <span className="text-[10px] uppercase tracking-wide text-muted-foreground">Quick Scan</span>
            <Badge variant="secondary" className="text-[10px] px-1.5 py-0">
              Socket-Inventar
            </Badge>
          </div>
          {lastQuick ? (
            <span className="text-sm text-zinc-300 font-mono">
              {new Date(lastQuick).toLocaleTimeString("de-DE", { hour: "2-digit", minute: "2-digit" })}{" "}
              <span className="text-muted-foreground text-xs">({relativeTime(lastQuick)})</span>
            </span>
          ) : (
            <span className="text-xs text-muted-foreground">Noch nicht ausgeführt</span>
          )}
        </div>

        <div className="w-px h-8 bg-zinc-800" />

        <div className="flex flex-col">
          <div className="flex items-center gap-2">
            <span className="text-[10px] uppercase tracking-wide text-muted-foreground">Full Scan</span>
            <Badge variant="secondary" className="text-[10px] px-1.5 py-0">
              Dienstversionen
            </Badge>
          </div>
          {lastFull ? (
            <span className="text-sm text-zinc-300 font-mono">
              {new Date(lastFull).toLocaleTimeString("de-DE", { hour: "2-digit", minute: "2-digit" })}{" "}
              <span className="text-muted-foreground text-xs">({relativeTime(lastFull)})</span>
            </span>
          ) : (
            <Badge variant="warning" className="text-[10px] mt-0.5">
              Full Scan fehlt oder nmap nicht verfügbar
            </Badge>
          )}
        </div>

        {isScanning && (
          <>
            <div className="w-px h-8 bg-zinc-800" />
            <div className="flex items-center gap-2 text-sm text-blue-400">
              <Loader2 className="h-4 w-4 animate-spin" />
              Scan läuft...
            </div>
          </>
        )}
      </div>

      {/* Buttons */}
      <div className="flex items-center gap-2 shrink-0">
        <Button
          variant="outline"
          size="sm"
          disabled={isScanning}
          onClick={() => triggerScan(nodeId, "quick")}
          className="gap-1.5"
        >
          <Zap className="h-3 w-3" />
          Quick Scan
        </Button>
        <Button
          variant="outline"
          size="sm"
          disabled={isScanning}
          onClick={() => triggerScan(nodeId, "full")}
          className="gap-1.5"
        >
          <Search className="h-3 w-3" />
          Full Scan
        </Button>
      </div>
    </div>
  );
}
