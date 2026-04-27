import type { ReactNode } from "react";
import { AlertTriangle, CheckCircle2, Clock, Info, XCircle } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

export type StatusTone = "ok" | "warning" | "critical" | "info" | "muted";

const toneClasses: Record<StatusTone, string> = {
  ok: "border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900/60 dark:bg-emerald-950/25 dark:text-emerald-300",
  warning: "border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/25 dark:text-amber-300",
  critical: "border-red-200 bg-red-50 text-red-700 dark:border-red-900/60 dark:bg-red-950/25 dark:text-red-300",
  info: "border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-900/60 dark:bg-sky-950/25 dark:text-sky-300",
  muted: "border-border bg-muted text-muted-foreground",
};

const toneIcons = {
  ok: CheckCircle2,
  warning: AlertTriangle,
  critical: XCircle,
  info: Info,
  muted: Clock,
};

interface StatusBadgeProps {
  tone: StatusTone;
  children: ReactNode;
  className?: string;
  withIcon?: boolean;
}

export function StatusBadge({ tone, children, className, withIcon = true }: StatusBadgeProps) {
  const Icon = toneIcons[tone];
  return (
    <Badge variant="outline" className={cn("gap-1.5 rounded-full px-2.5 py-1 font-medium", toneClasses[tone], className)}>
      {withIcon && <Icon className="h-3.5 w-3.5" />}
      {children}
    </Badge>
  );
}
