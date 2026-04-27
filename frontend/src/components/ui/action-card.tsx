import Link from "next/link";
import type { LucideIcon } from "lucide-react";
import { ArrowRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import type { StatusTone } from "@/components/ui/status-badge";
import { StatusBadge } from "@/components/ui/status-badge";

interface ActionCardProps {
  tone: StatusTone;
  icon: LucideIcon;
  title: string;
  description: string;
  badge: string;
  href: string;
  actionLabel: string;
}

const accentClass: Record<StatusTone, string> = {
  ok: "status-accent-ok",
  warning: "status-accent-warning",
  critical: "status-accent-critical",
  info: "status-accent-info",
  muted: "border-l-4 border-l-border",
};

export function ActionCard({ tone, icon: Icon, title, description, badge, href, actionLabel }: ActionCardProps) {
  return (
    <Card hover className={cn("overflow-hidden", accentClass[tone])}>
      <CardContent className="flex h-full flex-col gap-4 p-4">
        <div className="flex items-start justify-between gap-3">
          <div className="flex min-w-0 items-start gap-3">
            <div className="flex size-9 shrink-0 items-center justify-center rounded-md bg-muted">
              <Icon className="h-4 w-4" />
            </div>
            <div className="min-w-0">
              <h3 className="text-sm font-semibold">{title}</h3>
              <p className="mt-1 text-sm text-muted-foreground">{description}</p>
            </div>
          </div>
          <StatusBadge tone={tone}>{badge}</StatusBadge>
        </div>
        <Button asChild variant="outline" size="sm" className="mt-auto w-fit">
          <Link href={href}>
            {actionLabel}
            <ArrowRight className="ml-2 h-4 w-4" />
          </Link>
        </Button>
      </CardContent>
    </Card>
  );
}
