"use client";

import Link from "next/link";
import { Archive, Bell, ListChecks, Network, Plus, ScrollText, ShieldCheck } from "lucide-react";
import { AttentionQueue } from "@/components/dashboard/attention-queue";
import { buildDashboardSummary } from "@/components/dashboard/dashboard-summary";
import { NodeFleetTable } from "@/components/dashboard/node-fleet-table";
import { OpsStatusBar } from "@/components/dashboard/ops-status-bar";
import {
  OpsPanel,
  OpsPanelContent,
  OpsPanelDescription,
  OpsPanelHeader,
  OpsPanelTitle,
} from "@/components/ops/ops-panel";
import { Button } from "@/components/ui/button";
import { useNodeStore } from "@/stores/node-store";

const quickLinks = [
  { label: "Backups", href: "/backups", icon: Archive },
  { label: "Netzwerk", href: "/network", icon: Network },
  { label: "Logs", href: "/logs", icon: ScrollText },
  { label: "Tasks", href: "/task-center", icon: ListChecks },
  { label: "Benachrichtigungen", href: "/settings/notifications", icon: Bell },
  { label: "Sicherheit", href: "/security", icon: ShieldCheck },
];

export function DashboardOverview() {
  const { nodes, nodeStatus, isLoading } = useNodeStore();
  const summary = buildDashboardSummary(nodes, nodeStatus);

  return (
    <div className="grid gap-4">
      <OpsStatusBar summary={summary} />

      <section className="grid gap-4 xl:grid-cols-[minmax(0,1.4fr)_minmax(320px,0.6fr)]">
        <AttentionQueue items={summary.attentionItems} />
        <OpsPanel>
          <OpsPanelHeader>
            <OpsPanelTitle>Direkteinstiege</OpsPanelTitle>
            <OpsPanelDescription>
              Funktionen bleiben erreichbar, ohne den ersten Blick zu ueberladen.
            </OpsPanelDescription>
          </OpsPanelHeader>
          <OpsPanelContent className="grid gap-2 sm:grid-cols-2 xl:grid-cols-1">
            {quickLinks.map((item) => {
              const Icon = item.icon;
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  className="ops-row ops-focus-ring flex items-center gap-2 px-3 py-2 text-sm transition-colors hover:bg-accent/60"
                >
                  <Icon className="h-4 w-4 text-muted-foreground" />
                  <span className="font-medium">{item.label}</span>
                </Link>
              );
            })}
            <Button variant="outline" size="sm" asChild className="justify-start">
              <Link href="/settings/nodes">
                <Plus className="h-4 w-4" />
                Server hinzufuegen
              </Link>
            </Button>
          </OpsPanelContent>
        </OpsPanel>
      </section>

      <NodeFleetTable nodes={nodes} nodeStatus={nodeStatus} isLoading={isLoading} />
    </div>
  );
}
