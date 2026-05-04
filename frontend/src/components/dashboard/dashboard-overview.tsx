"use client";

import Link from "next/link";
import {
  Archive,
  Bell,
  ChevronDown,
  ListChecks,
  Network,
  Plus,
  ScrollText,
  ShieldCheck,
} from "lucide-react";
import { AttentionQueue } from "@/components/dashboard/attention-queue";
import { buildDashboardSummary } from "@/components/dashboard/dashboard-summary";
import { NodeFleetTable } from "@/components/dashboard/node-fleet-table";
import { OpsStatusBar } from "@/components/dashboard/ops-status-bar";
import { OpsPanel, OpsPanelContent, OpsPanelTitle } from "@/components/ops/ops-panel";
import { Button } from "@/components/ui/button";
import { useNodeStore } from "@/stores/node-store";

const quickLinks = [
  { label: "Backups", href: "/backups", icon: Archive },
  { label: "Netzwerk", href: "/network", icon: Network },
  { label: "Logs", href: "/logs", icon: ScrollText },
  { label: "Aufgaben", href: "/task-center", icon: ListChecks },
  { label: "Benachrichtigungen", href: "/settings/notifications", icon: Bell },
  { label: "Sicherheit", href: "/security", icon: ShieldCheck },
];

export function DashboardOverview() {
  const { nodes, nodeStatus, isLoading } = useNodeStore();
  const summary = buildDashboardSummary(nodes, nodeStatus);

  return (
    <div className="grid gap-4">
      <OpsStatusBar summary={summary} />

      <section className="grid gap-4">
        <AttentionQueue items={summary.attentionItems} />
        <OpsPanel className="overflow-hidden">
          <details className="group">
            <summary className="ops-focus-ring flex cursor-pointer list-none items-center gap-3 px-4 py-3 transition-colors hover:bg-accent/35 [&::-webkit-details-marker]:hidden">
              <OpsPanelTitle className="flex-1">Direkteinstiege</OpsPanelTitle>
              <span className="text-xs text-muted-foreground">{quickLinks.length + 1}</span>
              <ChevronDown className="h-4 w-4 text-muted-foreground transition-transform group-open:rotate-180" />
            </summary>
            <OpsPanelContent className="grid gap-2 border-t ops-divider sm:grid-cols-2 xl:grid-cols-4">
              {quickLinks.map((item) => {
                const Icon = item.icon;
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    className="ops-row ops-focus-ring flex items-center gap-2 px-3 py-2 text-sm transition-colors hover:bg-accent/45"
                  >
                    <Icon className="h-4 w-4 text-muted-foreground" />
                    <span className="font-medium">{item.label}</span>
                  </Link>
                );
              })}
              <Button variant="outline" size="sm" asChild className="justify-start">
                <Link href="/settings/nodes">
                  <Plus className="h-4 w-4" />
                  Server hinzufügen
                </Link>
              </Button>
            </OpsPanelContent>
          </details>
        </OpsPanel>
      </section>

      <NodeFleetTable nodes={nodes} nodeStatus={nodeStatus} isLoading={isLoading} />
    </div>
  );
}
