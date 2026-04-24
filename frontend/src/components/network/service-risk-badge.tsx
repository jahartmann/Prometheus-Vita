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
        <span
          tabIndex={0}
          aria-label={`${item.label}: ${reason}`}
          className="inline-flex rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
        >
          <Badge variant={item.variant} className="gap-1 [&_svg]:size-3">
            <Icon aria-hidden="true" />
            {item.label}
          </Badge>
        </span>
      </TooltipTrigger>
      <TooltipContent side="top">
        <p>{reason}</p>
      </TooltipContent>
    </Tooltip>
  );
}
