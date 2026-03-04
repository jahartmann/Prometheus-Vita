import type { LucideIcon } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import { TrendingUp, TrendingDown, Minus } from "lucide-react";

type KpiColor = "blue" | "green" | "orange" | "red" | "default";

interface KpiCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  icon: LucideIcon;
  color?: KpiColor;
  trend?: "up" | "down" | "neutral";
  trendValue?: string;
  className?: string;
}

const colorMap: Record<KpiColor, { gradient: string; iconBg: string; iconColor: string }> = {
  blue: {
    gradient: "gradient-blue",
    iconBg: "bg-kpi-blue/15",
    iconColor: "text-kpi-blue",
  },
  green: {
    gradient: "gradient-green",
    iconBg: "bg-kpi-green/15",
    iconColor: "text-kpi-green",
  },
  orange: {
    gradient: "gradient-orange",
    iconBg: "bg-kpi-orange/15",
    iconColor: "text-kpi-orange",
  },
  red: {
    gradient: "gradient-red",
    iconBg: "bg-kpi-red/15",
    iconColor: "text-kpi-red",
  },
  default: {
    gradient: "",
    iconBg: "bg-primary/10",
    iconColor: "text-primary",
  },
};

const TrendIcon = { up: TrendingUp, down: TrendingDown, neutral: Minus };
const trendColor = { up: "text-kpi-green", down: "text-kpi-red", neutral: "text-muted-foreground" };

export function KpiCard({
  title,
  value,
  subtitle,
  icon: Icon,
  color = "default",
  trend,
  trendValue,
  className,
}: KpiCardProps) {
  const colors = colorMap[color];

  return (
    <Card hover className={cn(colors.gradient, className)}>
      <CardContent className="p-5">
        <div className="flex items-center justify-between">
          <div className="space-y-1">
            <p className="text-sm font-medium text-muted-foreground">{title}</p>
            <p className="text-2xl font-bold tracking-tight">{value}</p>
            <div className="flex items-center gap-1.5">
              {trend && (
                <>
                  {(() => {
                    const TIcon = TrendIcon[trend];
                    return <TIcon className={cn("h-3 w-3", trendColor[trend])} />;
                  })()}
                  {trendValue && (
                    <span className={cn("text-xs font-medium", trendColor[trend])}>
                      {trendValue}
                    </span>
                  )}
                </>
              )}
              {subtitle && (
                <p className="text-xs text-muted-foreground">{subtitle}</p>
              )}
            </div>
          </div>
          <div className={cn("flex h-11 w-11 items-center justify-center rounded-xl", colors.iconBg)}>
            <Icon className={cn("h-5 w-5", colors.iconColor)} />
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
