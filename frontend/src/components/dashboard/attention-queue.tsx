import Link from "next/link";
import { ArrowRight } from "lucide-react";
import {
  OpsPanel,
  OpsPanelContent,
  OpsPanelDescription,
  OpsPanelHeader,
  OpsPanelTitle,
} from "@/components/ops/ops-panel";
import { StatusIndicator, type StatusTone } from "@/components/ops/status-indicator";
import type { AttentionItem } from "./dashboard-summary";

interface AttentionQueueProps {
  items: AttentionItem[];
}

const severityToTone: Record<AttentionItem["severity"], StatusTone> = {
  critical: "critical",
  warning: "warning",
  info: "info",
};

export function AttentionQueue({ items }: AttentionQueueProps) {
  return (
    <OpsPanel>
      <OpsPanelHeader>
        <OpsPanelTitle>Aufmerksamkeit</OpsPanelTitle>
        <OpsPanelDescription>
          Priorisierte Betriebsereignisse, ohne die ganze Funktionsliste nach oben zu ziehen.
        </OpsPanelDescription>
      </OpsPanelHeader>
      <OpsPanelContent className="space-y-2">
        {items.map((item) => (
          <Link
            key={item.id}
            href={item.href}
            className="ops-row ops-focus-ring group flex items-center justify-between gap-3 px-3 py-2.5 transition-colors hover:bg-accent/60"
          >
            <StatusIndicator
              tone={severityToTone[item.severity]}
              label={item.title}
              description={item.description}
            />
            <ArrowRight className="h-4 w-4 shrink-0 text-muted-foreground transition-transform group-hover:translate-x-0.5" />
          </Link>
        ))}
      </OpsPanelContent>
    </OpsPanel>
  );
}
