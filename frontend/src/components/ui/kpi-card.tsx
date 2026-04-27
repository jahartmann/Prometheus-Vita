import { Card, CardContent } from "@/components/ui/card";
import { cn } from "@/lib/utils";

interface KpiCardProps {
  title: string;
  value: number | string;
  subtitle?: string;
  icon: React.ComponentType<{ className?: string }>;
  color?: "blue" | "green" | "orange" | "red" | "purple" | "neutral" | string;
  trend?: {
    direction: "up" | "down" | "flat";
    label: string;
  };
}

// Restrained, monochrome-by-default. The colored variant is reserved for
// genuine signal — don't paint every KPI just to make the dashboard busy.
const colorClasses: Record<string, { bg: string; text: string; ring: string }> = {
  blue: {
    bg: "bg-sky-500/10",
    text: "text-sky-600 dark:text-sky-400",
    ring: "ring-sky-500/20",
  },
  green: {
    bg: "bg-emerald-500/10",
    text: "text-emerald-600 dark:text-emerald-400",
    ring: "ring-emerald-500/20",
  },
  orange: {
    bg: "bg-amber-500/10",
    text: "text-amber-600 dark:text-amber-400",
    ring: "ring-amber-500/20",
  },
  red: {
    bg: "bg-red-500/10",
    text: "text-red-600 dark:text-red-400",
    ring: "ring-red-500/20",
  },
  purple: {
    bg: "bg-violet-500/10",
    text: "text-violet-600 dark:text-violet-400",
    ring: "ring-violet-500/20",
  },
  neutral: {
    bg: "bg-muted",
    text: "text-muted-foreground",
    ring: "ring-border",
  },
};

export function KpiCard({ title, value, subtitle, icon: Icon, color = "neutral", trend }: KpiCardProps) {
  const palette = colorClasses[color] ?? colorClasses.neutral;
  return (
    <Card hover className="border-border/70">
      <CardContent className="flex items-start gap-3.5 p-4">
        <div
          className={cn(
            "flex size-10 shrink-0 items-center justify-center rounded-lg ring-1 ring-inset",
            palette.bg,
            palette.ring
          )}
        >
          <Icon className={cn("h-5 w-5", palette.text)} />
        </div>
        <div className="min-w-0 flex-1">
          <p className="truncate text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
            {title}
          </p>
          <p className="mt-0.5 truncate text-2xl font-semibold tabular tracking-tight">
            {value}
          </p>
          <div className="mt-0.5 flex items-center gap-1.5">
            {subtitle && (
              <p className="truncate text-xs text-muted-foreground">{subtitle}</p>
            )}
            {trend && (
              <span
                className={cn(
                  "inline-flex items-center gap-0.5 rounded text-[10px] font-medium tabular",
                  trend.direction === "up" && "text-emerald-600 dark:text-emerald-400",
                  trend.direction === "down" && "text-red-600 dark:text-red-400",
                  trend.direction === "flat" && "text-muted-foreground"
                )}
              >
                {trend.direction === "up" && "↑"}
                {trend.direction === "down" && "↓"}
                {trend.direction === "flat" && "→"}
                {trend.label}
              </span>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
