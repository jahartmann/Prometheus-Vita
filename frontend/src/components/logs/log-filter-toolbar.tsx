"use client";

import { useLogStore } from "@/stores/log-store";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Brain, Download } from "lucide-react";

const SEVERITIES = ["debug", "info", "warning", "error", "critical"] as const;

const SEVERITY_COLORS: Record<string, string> = {
  debug: "bg-zinc-700 text-zinc-300 hover:bg-zinc-600",
  info: "bg-zinc-600 text-zinc-200 hover:bg-zinc-500",
  warning: "bg-yellow-600 text-yellow-100 hover:bg-yellow-500",
  error: "bg-red-700 text-red-100 hover:bg-red-600",
  critical: "bg-red-900 text-red-200 hover:bg-red-800",
};

const SEVERITY_INACTIVE: Record<string, string> = {
  debug: "bg-zinc-800/50 text-zinc-600 hover:bg-zinc-700/50",
  info: "bg-zinc-800/50 text-zinc-500 hover:bg-zinc-700/50",
  warning: "bg-zinc-800/50 text-zinc-500 hover:bg-zinc-700/50",
  error: "bg-zinc-800/50 text-zinc-500 hover:bg-zinc-700/50",
  critical: "bg-zinc-800/50 text-zinc-500 hover:bg-zinc-700/50",
};

interface LogFilterToolbarProps {
  nodeId: string;
  onAnalyze: () => void;
  onExport: () => void;
  autoScroll: boolean;
  onAutoScrollChange: (v: boolean) => void;
}

export function LogFilterToolbar({
  onAnalyze,
  onExport,
  autoScroll,
  onAutoScrollChange,
}: LogFilterToolbarProps) {
  const sources = useLogStore((s) => s.sources);
  const filters = useLogStore((s) => s.filters);
  const setFilters = useLogStore((s) => s.setFilters);
  const isAnalyzing = useLogStore((s) => s.isAnalyzing);

  const toggleSeverity = (sev: string) => {
    const current = filters.severities;
    const next = current.includes(sev)
      ? current.filter((s) => s !== sev)
      : [...current, sev];
    setFilters({ severities: next });
  };

  const activeSeverities =
    filters.severities.length > 0 ? filters.severities : [...SEVERITIES];

  return (
    <div className="flex flex-wrap items-center gap-2 rounded-lg border border-zinc-800 bg-zinc-900 p-3">
      {/* Source Select */}
      <Select
        value={filters.sources[0] ?? "all"}
        onValueChange={(val) =>
          setFilters({ sources: val === "all" ? [] : [val] })
        }
      >
        <SelectTrigger className="h-8 w-44 border-zinc-700 bg-zinc-800 text-sm">
          <SelectValue placeholder="Alle Quellen" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">Alle Quellen</SelectItem>
          {sources.map((src) => (
            <SelectItem key={src.id} value={src.path}>
              {src.path}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      {/* Severity Toggles */}
      <div className="flex items-center gap-1">
        {SEVERITIES.map((sev) => {
          const isActive = activeSeverities.includes(sev);
          return (
            <button
              key={sev}
              onClick={() => toggleSeverity(sev)}
              className={`rounded px-2 py-0.5 text-xs font-medium transition-colors cursor-pointer ${
                isActive ? SEVERITY_COLORS[sev] : SEVERITY_INACTIVE[sev]
              }`}
            >
              {sev}
            </button>
          );
        })}
      </div>

      {/* Search Input */}
      <Input
        className="h-8 w-52 border-zinc-700 bg-zinc-800 text-sm"
        placeholder="Regex suchen..."
        value={filters.searchRegex}
        onChange={(e) => setFilters({ searchRegex: e.target.value })}
      />

      <div className="ml-auto flex items-center gap-2">
        {/* Auto-scroll */}
        <label className="flex items-center gap-2 text-xs text-zinc-400 cursor-pointer select-none">
          <Switch checked={autoScroll} onCheckedChange={onAutoScrollChange} />
          Auto-scroll
        </label>

        {/* Analyze button */}
        <Button
          size="sm"
          variant="default"
          onClick={onAnalyze}
          disabled={isAnalyzing}
          className="h-8 gap-1.5"
        >
          <Brain className="h-3.5 w-3.5" />
          {isAnalyzing ? "Analysiere..." : "KI analysieren"}
        </Button>

        {/* Download button */}
        <Button
          size="sm"
          variant="outline"
          onClick={onExport}
          className="h-8 gap-1.5 border-zinc-700"
        >
          <Download className="h-3.5 w-3.5" />
          Download
        </Button>
      </div>
    </div>
  );
}
