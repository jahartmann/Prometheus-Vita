import type { LucideIcon } from "lucide-react";
import { Card, CardContent } from "./card";
import { cn } from "@/lib/utils";

export type KpiTone = "neutral" | "primary" | "ok" | "warning" | "critical" | "info";

const toneIconBg: Record<KpiTone, string> = {
  neutral: "bg-muted text-muted-foreground",
  primary: "bg-primary/12 text-primary",
  ok: "bg-[oklch(from_var(--color-status-ok)_l_c_h_/_0.12)] text-[var(--color-status-ok)]",
  warning: "bg-[oklch(from_var(--color-status-warning)_l_c_h_/_0.12)] text-[var(--color-status-warning)]",
  critical: "bg-[oklch(from_var(--color-status-critical)_l_c_h_/_0.12)] text-[var(--color-status-critical)]",
  info: "bg-[oklch(from_var(--color-status-info)_l_c_h_/_0.12)] text-[var(--color-status-info)]",
};

export interface KpiCardProps {
  title: string;
  value: string | number;
  delta?: string;
  icon?: LucideIcon;
  tone?: KpiTone;
  className?: string;
}

export function KpiCard({ title, value, delta, icon: Icon, tone = "neutral", className }: KpiCardProps) {
  return (
    <Card className={cn("h-full", className)}>
      <CardContent className="flex items-start justify-between gap-3 p-4">
        <div className="min-w-0">
          <p className="truncate text-xs font-medium uppercase tracking-wide text-muted-foreground">{title}</p>
          <p className="mt-1 text-2xl font-semibold tabular-nums">{value}</p>
          {delta && <p className="mt-1 text-xs text-muted-foreground">{delta}</p>}
        </div>
        {Icon && (
          <div className={cn("flex h-10 w-10 shrink-0 items-center justify-center rounded-md", toneIconBg[tone])}>
            <Icon className="h-5 w-5" />
          </div>
        )}
      </CardContent>
    </Card>
  );
}
