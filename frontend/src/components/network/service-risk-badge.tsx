"use client";

import { AlertTriangle, Info, ShieldAlert, ShieldCheck } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import type { PortRisk } from "@/lib/network-scan-normalizer";

interface ServiceRiskBadgeProps {
  risk: PortRisk;
  reason: string;
}

const config = {
  high: { label: "Hoch", variant: "destructive" as const, icon: ShieldAlert },
  medium: { label: "Mittel", variant: "warning" as const, icon: AlertTriangle },
  low: { label: "Niedrig", variant: "success" as const, icon: ShieldCheck },
  info: { label: "Info", variant: "secondary" as const, icon: Info },
};

export function ServiceRiskBadge({ risk, reason }: ServiceRiskBadgeProps) {
  const item = config[risk];
  const Icon = item.icon;

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Badge variant={item.variant} className="gap-1 [&_svg]:size-3">
          <Icon />
          {item.label}
        </Badge>
      </TooltipTrigger>
      <TooltipContent side="top">
        <p>{reason}</p>
      </TooltipContent>
    </Tooltip>
  );
}
