import type { LucideIcon } from "lucide-react";
import { RefreshCw } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "./card";
import { Button } from "./button";
import { StatusBadge, type StatusTone } from "./status-badge";

export interface FeatureStatusCardProps {
  title: string;
  description: string;
  icon: LucideIcon;
  tone: StatusTone;
  status: string;
  details?: React.ReactNode;
  actionLabel?: string;
  onAction?: () => void;
  pending?: boolean;
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
  pending,
  error,
}: FeatureStatusCardProps) {
  return (
    <Card>
      <CardHeader className="flex-row items-start justify-between gap-4 pb-3">
        <div className="flex items-start gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-md bg-muted">
            <Icon className="h-5 w-5" />
          </div>
          <div>
            <CardTitle>{title}</CardTitle>
            <p className="mt-1 text-sm text-muted-foreground">{description}</p>
          </div>
        </div>
        <StatusBadge tone={tone}>{status}</StatusBadge>
      </CardHeader>
      <CardContent>
        {details}
        {error && (
          <p className="rounded-md border border-[oklch(from_var(--color-status-critical)_l_c_h_/_0.4)] bg-[oklch(from_var(--color-status-critical)_l_c_h_/_0.08)] px-3 py-2 text-sm" style={{ color: "var(--color-status-critical)" }}>
            {error}
          </p>
        )}
        {actionLabel && onAction && (
          <Button variant="outline" size="sm" className="w-fit" onClick={onAction} disabled={pending}>
            {pending && <RefreshCw className="mr-2 h-4 w-4 animate-spin" />}
            {actionLabel}
          </Button>
        )}
      </CardContent>
    </Card>
  );
}
