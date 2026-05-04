import type { ComponentType } from "react";
import { AlertTriangle, CheckCircle2, Circle, Info, XCircle } from "lucide-react";
import { cn } from "@/lib/utils";

export type StatusTone = "ok" | "warning" | "critical" | "info" | "muted";

const toneClasses: Record<StatusTone, string> = {
  ok: "text-emerald-500",
  warning: "text-amber-500",
  critical: "text-red-500",
  info: "text-sky-500",
  muted: "text-muted-foreground",
};

const dotClasses: Record<StatusTone, string> = {
  ok: "bg-emerald-500",
  warning: "bg-amber-500",
  critical: "bg-red-500",
  info: "bg-sky-500",
  muted: "bg-muted-foreground",
};

const icons = {
  ok: CheckCircle2,
  warning: AlertTriangle,
  critical: XCircle,
  info: Info,
  muted: Circle,
} satisfies Record<StatusTone, ComponentType<{ className?: string }>>;

interface StatusIndicatorProps {
  tone: StatusTone;
  label: string;
  description?: string;
  withIcon?: boolean;
  className?: string;
}

export function StatusIndicator({
  tone,
  label,
  description,
  withIcon = false,
  className,
}: StatusIndicatorProps) {
  const Icon = icons[tone];

  return (
    <span className={cn("inline-flex min-w-0 items-center gap-2", className)}>
      {withIcon ? (
        <Icon className={cn("h-4 w-4 shrink-0", toneClasses[tone])} />
      ) : (
        <span className={cn("h-2 w-2 shrink-0 rounded-full", dotClasses[tone])} />
      )}
      <span className="min-w-0">
        <span className="block truncate text-xs font-medium">{label}</span>
        {description && (
          <span className="block truncate text-[11px] text-muted-foreground">
            {description}
          </span>
        )}
      </span>
    </span>
  );
}
