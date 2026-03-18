"use client";

import { useEffect, useRef, useState, useMemo } from "react";
import { useLogStore } from "@/stores/log-store";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Bookmark } from "lucide-react";
import { logAnalysisApi } from "@/lib/api";

interface LogStreamProps {
  autoScroll: boolean;
}

const SEVERITY_CLASS: Record<string, string> = {
  debug: "text-zinc-500",
  info: "text-zinc-300",
  warning: "text-yellow-400",
  error: "text-red-500",
  critical: "text-red-500 animate-pulse font-bold",
};

function formatTimestamp(ts: string): string {
  try {
    return new Date(ts).toLocaleTimeString("de-DE", {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  } catch {
    return ts;
  }
}

export function LogStream({ autoScroll }: LogStreamProps) {
  const entries = useLogStore((s) => s.entries);
  const filters = useLogStore((s) => s.filters);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const userScrolledRef = useRef(false);

  // Filter entries
  const visibleEntries = useMemo(() => {
    let filtered = entries;

    // Source filter
    if (filters.sources.length > 0) {
      filtered = filtered.filter((e) => filters.sources.includes(e.source));
    }

    // Severity filter
    if (filters.severities.length > 0) {
      filtered = filtered.filter((e) => filters.severities.includes(e.severity));
    }

    // Regex search
    if (filters.searchRegex) {
      try {
        const re = new RegExp(filters.searchRegex, "i");
        filtered = filtered.filter(
          (e) => re.test(e.message) || re.test(e.raw)
        );
      } catch {
        // invalid regex — skip filter
      }
    }

    // Limit to last 1000
    if (filtered.length > 1000) {
      filtered = filtered.slice(filtered.length - 1000);
    }

    return filtered;
  }, [entries, filters]);

  // Handle auto-scroll
  useEffect(() => {
    if (!autoScroll || userScrolledRef.current) return;
    const el = containerRef.current;
    if (el) {
      el.scrollTop = el.scrollHeight;
    }
  }, [visibleEntries, autoScroll]);

  const handleScroll = () => {
    const el = containerRef.current;
    if (!el) return;
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 40;
    userScrolledRef.current = !atBottom;
  };

  const handleBookmark = async (entry: (typeof entries)[0]) => {
    try {
      await logAnalysisApi.createBookmark({
        node_id: entry.node_id,
        log_entry_json: entry,
      });
    } catch {
      // ignore
    }
  };

  return (
    <div
      ref={containerRef}
      onScroll={handleScroll}
      className="flex-1 overflow-auto bg-zinc-950 rounded-lg border border-zinc-800 p-2 font-mono text-sm min-h-0"
      style={{ minHeight: "300px" }}
    >
      {visibleEntries.length === 0 && (
        <div className="flex items-center justify-center h-24 text-zinc-600 text-sm">
          Keine Log-Eintraege — warte auf Stream...
        </div>
      )}

      {visibleEntries.map((entry) => {
        const isExpanded = expandedId === entry.id;
        const hasAnomaly =
          entry.assessment && entry.assessment.anomaly_score > 0;
        const severityClass =
          SEVERITY_CLASS[entry.severity] ?? "text-zinc-300";

        return (
          <div key={entry.id} className="group">
            {/* Main log line */}
            <div
              className={`flex items-start gap-2 px-1 py-0.5 rounded cursor-pointer hover:bg-zinc-900 transition-colors ${
                isExpanded ? "bg-zinc-900" : ""
              }`}
              onClick={() => setExpandedId(isExpanded ? null : entry.id)}
            >
              {/* Anomaly indicator */}
              {hasAnomaly ? (
                <Badge className="shrink-0 self-center bg-orange-500/20 text-orange-400 border-orange-500/30 text-[10px] px-1 py-0">
                  {entry.assessment!.anomaly_score.toFixed(2)}
                </Badge>
              ) : (
                <span className="w-12 shrink-0" />
              )}

              {/* Log line text */}
              <span className={`flex-1 break-all leading-relaxed ${severityClass}`}>
                <span className="text-zinc-600 mr-1">
                  [{formatTimestamp(entry.timestamp)}]
                </span>
                <span className="text-zinc-500 mr-1">
                  [{entry.severity.toUpperCase()}]
                </span>
                <span className="text-zinc-500 mr-1">[{entry.process}]</span>
                {entry.message}
              </span>
            </div>

            {/* Expanded details */}
            {isExpanded && entry.assessment && (
              <div className="ml-14 mb-1 rounded border border-zinc-800 bg-zinc-900/80 p-3 text-xs space-y-1.5">
                <div className="flex items-center justify-between">
                  <span className="text-zinc-400 font-medium">KI-Analyse</span>
                  <Button
                    size="sm"
                    variant="outline"
                    className="h-6 border-zinc-700 text-xs gap-1"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleBookmark(entry);
                    }}
                  >
                    <Bookmark className="h-3 w-3" />
                    Bookmark
                  </Button>
                </div>
                <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-zinc-400">
                  <span>Score:</span>
                  <span className="text-orange-400">
                    {entry.assessment.anomaly_score.toFixed(3)}
                  </span>
                  <span>Kategorie:</span>
                  <span className="text-zinc-300">
                    {entry.assessment.category}
                  </span>
                </div>
                {entry.assessment.summary && (
                  <p className="text-zinc-400 leading-relaxed">
                    {entry.assessment.summary}
                  </p>
                )}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
