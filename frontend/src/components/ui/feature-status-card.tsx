import type { ReactNode } from "react";
import type { LucideIcon } from "lucide-react";
import { RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { StatusBadge, type StatusTone } from "@/components/ui/status-badge";

interface FeatureStatusCardProps {
  title: string;
  description: string;
  icon: LucideIcon;
  tone: StatusTone;
  status: string;
  details?: ReactNode;
  actionLabel?: string;
  onAction?: () => void;
  isActionPending?: boolean;
  error?: string | null;
}

export function FeatureStatusCard({
  title,
  description,
  icon: Icon,
  tone,
  status,
  details,
  actionLabel,
  onAction,
  isActionPending,
  error,
}: FeatureStatusCardProps) {
  return (
    <Card>
      <CardHeader className="flex-row items-start justify-between gap-4 pb-3">
        <div className="flex items-start gap-3">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-md bg-muted">
            <Icon className="h-5 w-5" />
          </div>
          <div>
            <CardTitle className="text-base">{title}</CardTitle>
            <p className="mt-1 text-sm text-muted-foreground">{description}</p>
          </div>
        </div>
        <StatusBadge tone={tone}>{status}</StatusBadge>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        {details}
        {error && <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/25 dark:text-red-300">{error}</p>}
        {actionLabel && onAction && (
          <Button variant="outline" size="sm" className="w-fit" onClick={onAction} disabled={isActionPending}>
            {isActionPending && <RefreshCw className="mr-2 h-4 w-4 animate-spin" />}
            {actionLabel}
          </Button>
        )}
      </CardContent>
    </Card>
  );
}
