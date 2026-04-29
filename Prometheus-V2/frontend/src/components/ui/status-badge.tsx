import { AlertTriangle, CheckCircle2, Clock, Info, XCircle, type LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";

export type StatusTone = "ok" | "warning" | "critical" | "info" | "muted";

const toneToken: Record<StatusTone, string> = {
  ok: "var(--color-status-ok)",
  warning: "var(--color-status-warning)",
  critical: "var(--color-status-critical)",
  info: "var(--color-status-info)",
  muted: "var(--color-status-muted)",
};

const toneIcon: Record<StatusTone, LucideIcon> = {
  ok: CheckCircle2,
  warning: AlertTriangle,
  critical: XCircle,
  info: Info,
  muted: Clock,
};

export interface StatusBadgeProps {
  tone: StatusTone;
  children: React.ReactNode;
  className?: string;
  withIcon?: boolean;
}

export function StatusBadge({ tone, children, className, withIcon = true }: StatusBadgeProps) {
  const Icon = toneIcon[tone];
  return (
    <span
      data-tone={tone}
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium",
        className
      )}
      style={{
        borderColor: `oklch(from ${toneToken[tone]} l c h / 0.4)`,
        backgroundColor: `oklch(from ${toneToken[tone]} l c h / 0.1)`,
        color: toneToken[tone],
      }}
    >
      {withIcon && <Icon className="h-3.5 w-3.5" />}
      {children}
    </span>
  );
}
