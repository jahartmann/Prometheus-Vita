import { Card, CardContent } from "@/components/ui/card";
import { cn } from "@/lib/utils";

interface KpiCardProps {
  title: string;
  value: number | string;
  subtitle?: string;
  icon: React.ComponentType<{ className?: string }>;
  color?: "blue" | "green" | "orange" | "red" | "purple" | "neutral" | string;
}

const colorClasses: Record<string, string> = {
  blue: "bg-blue-500/15 text-blue-600 dark:text-blue-300",
  green: "bg-green-500/15 text-green-600 dark:text-green-300",
  orange: "bg-orange-500/15 text-orange-600 dark:text-orange-300",
  red: "bg-red-500/15 text-red-600 dark:text-red-300",
  purple: "bg-violet-500/15 text-violet-600 dark:text-violet-300",
  neutral: "bg-muted text-muted-foreground",
};

export function KpiCard({ title, value, subtitle, icon: Icon, color = "neutral" }: KpiCardProps) {
  return (
    <Card hover>
      <CardContent className="flex items-center gap-4 p-4">
        <div
          className={cn(
            "flex size-10 shrink-0 items-center justify-center rounded-md",
            colorClasses[color] ?? colorClasses.neutral
          )}
        >
          <Icon className="h-5 w-5" />
        </div>
        <div className="min-w-0">
          <p className="text-xs font-medium text-muted-foreground">{title}</p>
          <p className="truncate text-2xl font-semibold tracking-normal">{value}</p>
          {subtitle && <p className="truncate text-xs text-muted-foreground">{subtitle}</p>}
        </div>
      </CardContent>
    </Card>
  );
}
