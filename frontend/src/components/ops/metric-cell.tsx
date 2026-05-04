import { cn } from "@/lib/utils";

interface MetricCellProps {
  label: string;
  value: string | number;
  helper?: string;
  tone?: "default" | "ok" | "warning" | "critical";
  className?: string;
}

const valueToneClasses = {
  default: "text-foreground",
  ok: "text-emerald-500",
  warning: "text-amber-500",
  critical: "text-red-500",
};

export function MetricCell({
  label,
  value,
  helper,
  tone = "default",
  className,
}: MetricCellProps) {
  return (
    <div className={cn("min-w-0", className)}>
      <p className="truncate text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
        {label}
      </p>
      <p className={cn("mt-1 truncate text-lg font-semibold tabular", valueToneClasses[tone])}>
        {value}
      </p>
      {helper && (
        <p className="mt-0.5 truncate text-xs text-muted-foreground">{helper}</p>
      )}
    </div>
  );
}
