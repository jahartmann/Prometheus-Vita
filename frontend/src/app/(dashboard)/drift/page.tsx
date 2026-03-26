"use client";

import { useEffect, useMemo } from "react";
import {
  CheckCircle2,
  AlertTriangle,
  ShieldAlert,
  ChevronDown,
  RefreshCw,
  FileText,
} from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Skeleton } from "@/components/ui/skeleton";
import { useDriftStore } from "@/stores/drift-store";
import { useNodeStore } from "@/stores/node-store";
import type { DriftCheck } from "@/types/api";

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 60) return `vor ${mins} Min.`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `vor ${hours} Std.`;
  const days = Math.floor(hours / 24);
  return `vor ${days} Tag${days > 1 ? "en" : ""}`;
}

function severityLabel(severity: number): { text: string; className: string } {
  if (severity >= 7) return { text: "Kritisch", className: "text-red-600 bg-red-50 border-red-200" };
  if (severity >= 4) return { text: "Prüfen empfohlen", className: "text-amber-600 bg-amber-50 border-amber-200" };
  return { text: "Unkritisch", className: "text-green-600 bg-green-50 border-green-200" };
}

function DriftCheckRow({
  check,
  nodeName,
  onAcceptBaseline,
}: {
  check: DriftCheck;
  nodeName: string;
  onAcceptBaseline: (id: string) => void;
}) {
  const totalChanges = check.changed_files + check.added_files + check.removed_files;
  const aiSeverity = check.ai_analysis?.overall_severity;
  const sev = aiSeverity != null ? severityLabel(aiSeverity) : null;

  return (
    <Collapsible>
      <CollapsibleTrigger asChild>
        <button className="w-full flex items-center gap-3 px-4 py-3 hover:bg-muted/50 transition-colors text-left border-b last:border-b-0">
          {/* Severity Dot */}
          <span
            className={`h-2.5 w-2.5 rounded-full shrink-0 ${
              aiSeverity != null && aiSeverity >= 7
                ? "bg-red-500"
                : aiSeverity != null && aiSeverity >= 4
                ? "bg-amber-500"
                : "bg-yellow-400"
            }`}
          />

          {/* Node Name */}
          <span className="font-medium text-sm min-w-0 truncate">{nodeName}</span>

          {/* Change Summary */}
          <span className="text-xs text-muted-foreground shrink-0">
            {totalChanges} {totalChanges === 1 ? "Datei" : "Dateien"} geändert
          </span>

          {/* AI Badge */}
          {sev && (
            <Badge variant="outline" className={`text-[10px] shrink-0 ${sev.className}`}>
              {sev.text}
            </Badge>
          )}

          {/* Time */}
          <span className="text-xs text-muted-foreground ml-auto shrink-0">
            {timeAgo(check.checked_at)}
          </span>

          <ChevronDown className="h-4 w-4 text-muted-foreground shrink-0 transition-transform [[data-state=open]>&]:rotate-180" />
        </button>
      </CollapsibleTrigger>

      <CollapsibleContent>
        <div className="px-4 py-3 bg-muted/30 border-b space-y-3">
          {/* File Details */}
          {check.details && check.details.length > 0 && (
            <div className="space-y-1">
              {check.details
                .filter((d) => d.status !== "unchanged")
                .map((detail) => (
                  <div
                    key={detail.file_path}
                    className="flex items-center gap-2 text-xs"
                  >
                    <FileText className="h-3 w-3 text-muted-foreground shrink-0" />
                    <span className="font-mono truncate">{detail.file_path}</span>
                    <Badge
                      variant="outline"
                      className={`text-[10px] px-1.5 py-0 shrink-0 ${
                        detail.status === "added"
                          ? "text-green-600 border-green-300"
                          : detail.status === "removed"
                          ? "text-red-600 border-red-300"
                          : "text-amber-600 border-amber-300"
                      }`}
                    >
                      {detail.status === "added"
                        ? "Neu"
                        : detail.status === "removed"
                        ? "Entfernt"
                        : "Geändert"}
                    </Badge>
                    {detail.ai_file_analysis && (
                      <span className="text-muted-foreground ml-auto">
                        {detail.ai_file_analysis.summary}
                      </span>
                    )}
                  </div>
                ))}
            </div>
          )}

          {/* AI Summary */}
          {check.ai_analysis && (
            <div className="text-xs text-muted-foreground bg-background rounded-md p-2 border">
              <span className="font-medium">KI-Bewertung:</span>{" "}
              {check.ai_analysis.overall_summary}
            </div>
          )}

          {/* Actions */}
          <div className="flex gap-2">
            <Button
              size="sm"
              variant="outline"
              onClick={(e) => {
                e.stopPropagation();
                onAcceptBaseline(check.id);
              }}
            >
              <RefreshCw className="h-3.5 w-3.5 mr-1.5" />
              Baseline aktualisieren
            </Button>
          </div>
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}

export default function DriftPage() {
  const { checks, isLoading, fetchAll, acceptBaseline } = useDriftStore();
  const { nodes, fetchNodes } = useNodeStore();

  useEffect(() => {
    fetchAll();
    fetchNodes();
  }, [fetchAll, fetchNodes]);

  const nodeNames: Record<string, string> = {};
  for (const node of nodes) {
    nodeNames[node.id] = node.name;
  }

  const stats = useMemo(() => {
    const completed = checks.filter((c) => c.status === "completed");
    const withDrift = completed.filter(
      (c) => c.changed_files + c.added_files + c.removed_files > 0
    );
    const critical = withDrift.filter(
      (c) => c.ai_analysis && c.ai_analysis.overall_severity >= 7
    );
    return { completed, withDrift, critical };
  }, [checks]);

  // Determine overall status
  const bannerStatus: "green" | "yellow" | "red" = useMemo(() => {
    if (stats.critical.length > 0) return "red";
    if (stats.withDrift.length > 0) return "yellow";
    return "green";
  }, [stats]);

  const bannerConfig = {
    green: {
      icon: CheckCircle2,
      text: "Alle Konfigurationen stimmen überein",
      className: "bg-green-50 border-green-200 text-green-800 dark:bg-green-950/30 dark:border-green-800 dark:text-green-300",
      iconClass: "text-green-600 dark:text-green-400",
    },
    yellow: {
      icon: AlertTriangle,
      text: `${stats.withDrift.length} Abweichung${stats.withDrift.length !== 1 ? "en" : ""} erkannt`,
      className: "bg-amber-50 border-amber-200 text-amber-800 dark:bg-amber-950/30 dark:border-amber-800 dark:text-amber-300",
      iconClass: "text-amber-600 dark:text-amber-400",
    },
    red: {
      icon: ShieldAlert,
      text: `${stats.critical.length} kritische Abweichung${stats.critical.length !== 1 ? "en" : ""}`,
      className: "bg-red-50 border-red-200 text-red-800 dark:bg-red-950/30 dark:border-red-800 dark:text-red-300",
      iconClass: "text-red-600 dark:text-red-400",
    },
  };

  const banner = bannerConfig[bannerStatus];
  const BannerIcon = banner.icon;

  if (isLoading && checks.length === 0) {
    return (
      <div className="space-y-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Konfigurations-Drift</h1>
          <p className="text-sm text-muted-foreground">
            Vergleicht aktuelle VM-Konfigurationen mit der letzten gesicherten Baseline.
          </p>
        </div>
        <Skeleton className="h-20 w-full rounded-xl" />
        <Skeleton className="h-64 w-full rounded-xl" />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Konfigurations-Drift</h1>
        <p className="text-sm text-muted-foreground">
          Vergleicht aktuelle VM-Konfigurationen mit der letzten gesicherten Baseline.
        </p>
      </div>

      {/* Status Banner */}
      <div className={`flex items-center gap-3 rounded-lg border p-4 ${banner.className}`}>
        <BannerIcon className={`h-6 w-6 shrink-0 ${banner.iconClass}`} />
        <div>
          <p className="font-semibold text-base">{banner.text}</p>
          <p className="text-xs opacity-75">
            {stats.completed.length} Checks abgeschlossen
            {stats.withDrift.length > 0 &&
              ` | ${stats.withDrift.length} mit Änderungen`}
          </p>
        </div>
        <Button
          size="sm"
          variant="ghost"
          className="ml-auto shrink-0"
          onClick={() => fetchAll()}
        >
          <RefreshCw className="h-4 w-4" />
        </Button>
      </div>

      {/* Drift List - only items with drift */}
      {stats.withDrift.length > 0 && (
        <Card>
          <CardContent className="p-0">
            {stats.withDrift
              .sort((a, b) => {
                const sevA = a.ai_analysis?.overall_severity ?? 0;
                const sevB = b.ai_analysis?.overall_severity ?? 0;
                return sevB - sevA;
              })
              .map((check) => (
                <DriftCheckRow
                  key={check.id}
                  check={check}
                  nodeName={nodeNames[check.node_id] || check.node_id.slice(0, 8)}
                  onAcceptBaseline={acceptBaseline}
                />
              ))}
          </CardContent>
        </Card>
      )}

      {stats.withDrift.length === 0 && !isLoading && (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <CheckCircle2 className="h-12 w-12 text-green-500 mb-3" />
            <p className="text-muted-foreground">Keine Abweichungen gefunden.</p>
            <p className="text-sm text-muted-foreground">
              Alle Konfigurationen entsprechen der Baseline.
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
