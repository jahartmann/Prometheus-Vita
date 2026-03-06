import { Card, CardContent } from "@/components/ui/card";
import { cn } from "@/lib/utils";

const colorMap: Record<string, string> = {
  blue: "text-kpi-blue",
  green: "text-kpi-green",
  orange: "text-kpi-orange",
  red: "text-kpi-red",
  default: "text-muted-foreground",
};

interface KpiCardProps {
  title: string;
  value: number | string;
  subtitle?: string;
  icon: React.ComponentType<{ className?: string }>;
  color?: string;
}

export function KpiCard({ title, value, subtitle, icon: Icon, color = "default" }: KpiCardProps) {
  return (
    <Card>
      <CardContent className="flex items-center gap-4 p-4">
        <div className={cn("flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-muted", colorMap[color] ?? colorMap.default)}>
          <Icon className="h-5 w-5" />
        </div>
        <div className="min-w-0">
          <p className="text-xs font-medium text-muted-foreground">{title}</p>
          <p className="text-2xl font-bold">{value}</p>
          {subtitle && <p className="text-xs text-muted-foreground">{subtitle}</p>}
        </div>
      </CardContent>
    </Card>
  );
}
