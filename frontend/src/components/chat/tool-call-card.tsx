"use client";

import { useState } from "react";
import {
  ChevronDown,
  ChevronRight,
  Wrench,
  CheckCircle2,
  XCircle,
  Loader2,
  Clock,
} from "lucide-react";
import type { AgentToolCall } from "@/types/api";
import { cn } from "@/lib/utils";

interface ToolCallCardProps {
  toolCall: AgentToolCall;
}

export function ToolCallCard({ toolCall }: ToolCallCardProps) {
  const [expanded, setExpanded] = useState(false);

  const statusConfig = {
    success: {
      icon: <CheckCircle2 className="h-3.5 w-3.5 text-green-500" />,
      bg: "border-green-500/20 bg-green-500/5",
    },
    error: {
      icon: <XCircle className="h-3.5 w-3.5 text-destructive" />,
      bg: "border-destructive/20 bg-destructive/5",
    },
    running: {
      icon: <Loader2 className="h-3.5 w-3.5 animate-spin text-blue-500" />,
      bg: "border-blue-500/20 bg-blue-500/5",
    },
    pending: {
      icon: <Clock className="h-3.5 w-3.5 text-muted-foreground" />,
      bg: "border-muted bg-muted/30",
    },
  }[toolCall.status] || {
    icon: null,
    bg: "border-muted bg-muted/30",
  };

  return (
    <div
      className={cn(
        "mx-4 md:mx-6 my-1.5 rounded-lg border text-xs transition-colors",
        statusConfig.bg
      )}
    >
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex w-full items-center gap-2 px-3 py-2 text-left hover:bg-muted/30 rounded-lg transition-colors"
      >
        {expanded ? (
          <ChevronDown className="h-3 w-3 shrink-0 text-muted-foreground" />
        ) : (
          <ChevronRight className="h-3 w-3 shrink-0 text-muted-foreground" />
        )}
        <Wrench className="h-3 w-3 shrink-0 text-muted-foreground" />
        <span className="font-medium font-mono">{toolCall.tool_name}</span>
        {statusConfig.icon}
        {toolCall.duration_ms > 0 && (
          <span className="ml-auto text-muted-foreground tabular-nums">
            {toolCall.duration_ms}ms
          </span>
        )}
      </button>
      {expanded && (
        <div className="space-y-2 border-t px-3 py-2">
          <div>
            <div className="font-medium text-muted-foreground mb-1">Argumente:</div>
            <pre
              className={cn(
                "overflow-auto rounded-md bg-muted/50 p-2 text-[11px]",
                "max-h-32 scrollbar-thin"
              )}
            >
              {typeof toolCall.arguments === "string"
                ? toolCall.arguments
                : JSON.stringify(toolCall.arguments, null, 2)}
            </pre>
          </div>
          {toolCall.result != null && (
            <div>
              <div className="font-medium text-muted-foreground mb-1">Ergebnis:</div>
              <pre className="max-h-48 overflow-auto rounded-md bg-muted/50 p-2 text-[11px] scrollbar-thin">
                {typeof toolCall.result === "string"
                  ? toolCall.result
                  : (JSON.stringify(toolCall.result, null, 2) as string)}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
