"use client";

import { useState } from "react";
import { ChevronDown, ChevronRight, Wrench, CheckCircle2, XCircle, Loader2 } from "lucide-react";
import type { AgentToolCall } from "@/types/api";
import { cn } from "@/lib/utils";

interface ToolCallCardProps {
  toolCall: AgentToolCall;
}

export function ToolCallCard({ toolCall }: ToolCallCardProps) {
  const [expanded, setExpanded] = useState(false);

  const statusIcon = {
    success: <CheckCircle2 className="h-3.5 w-3.5 text-green-500" />,
    error: <XCircle className="h-3.5 w-3.5 text-destructive" />,
    running: <Loader2 className="h-3.5 w-3.5 animate-spin text-blue-500" />,
    pending: <Loader2 className="h-3.5 w-3.5 animate-spin text-muted-foreground" />,
  }[toolCall.status] || null;

  return (
    <div className="mx-4 my-1 rounded-md border bg-muted/30 text-xs">
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex w-full items-center gap-2 px-3 py-2 text-left hover:bg-muted/50"
      >
        {expanded ? (
          <ChevronDown className="h-3 w-3 shrink-0" />
        ) : (
          <ChevronRight className="h-3 w-3 shrink-0" />
        )}
        <Wrench className="h-3 w-3 shrink-0 text-muted-foreground" />
        <span className="font-medium">{toolCall.tool_name}</span>
        {statusIcon}
        {toolCall.duration_ms > 0 && (
          <span className="ml-auto text-muted-foreground">
            {toolCall.duration_ms}ms
          </span>
        )}
      </button>
      {expanded && (
        <div className="space-y-2 border-t px-3 py-2">
          <div>
            <div className="font-medium text-muted-foreground">Argumente:</div>
            <pre
              className={cn(
                "mt-1 overflow-auto rounded bg-muted p-2",
                "max-h-32"
              )}
            >
              {typeof toolCall.arguments === "string"
                ? toolCall.arguments
                : JSON.stringify(toolCall.arguments, null, 2)}
            </pre>
          </div>
          {toolCall.result != null && (
            <div>
              <div className="font-medium text-muted-foreground">Ergebnis:</div>
              <pre className="mt-1 max-h-48 overflow-auto rounded bg-muted p-2">
                {typeof toolCall.result === "string"
                  ? toolCall.result
                  : JSON.stringify(toolCall.result, null, 2) as string}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
