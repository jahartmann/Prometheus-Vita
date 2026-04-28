import { AlertTriangle, RefreshCw } from "lucide-react";
import { Button } from "./button";
import { cn } from "@/lib/utils";

export interface ErrorStateProps {
  title?: string;
  message: string;
  onRetry?: () => void;
  retryLabel?: string;
  className?: string;
}

export function ErrorState({
  title = "Etwas ist schiefgelaufen",
  message,
  onRetry,
  retryLabel = "Erneut versuchen",
  className,
}: ErrorStateProps) {
  return (
    <div
      className={cn(
        "flex flex-col items-start gap-3 rounded-lg border px-4 py-3 text-sm",
        className
      )}
      style={{
        borderColor: "oklch(from var(--color-status-critical) l c h / 0.4)",
        backgroundColor: "oklch(from var(--color-status-critical) l c h / 0.08)",
        color: "oklch(var(--color-status-critical))",
      }}
    >
      <div className="flex items-start gap-2">
        <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
        <div>
          <p className="font-semibold">{title}</p>
          <p className="mt-0.5 opacity-90">{message}</p>
        </div>
      </div>
      {onRetry && (
        <Button variant="outline" size="sm" onClick={onRetry}>
          <RefreshCw className="mr-2 h-3.5 w-3.5" />
          {retryLabel}
        </Button>
      )}
    </div>
  );
}
