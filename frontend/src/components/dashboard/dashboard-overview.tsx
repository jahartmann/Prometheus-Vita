"use client";

import { Activity, AlertTriangle, ArrowRight, Bell, Cpu, ListChecks, Network, Plus, ScrollText, Server, ServerCog, ShieldCheck } from "lucide-react";
import Link from "next/link";
import { useNodeStore } from "@/stores/node-store";
import { KpiCard } from "@/components/ui/kpi-card";
import { Card, CardContent } from "@/components/ui/card";
import { NodeGrid } from "./node-grid";
import { Button } from "@/components/ui/button";

export function DashboardOverview() {
  const { nodes, nodeStatus } = useNodeStore();

  const onlineNodes = nodes.filter((n) => n.is_online).length;
  const totalVMs = Object.values(nodeStatus).reduce(
    (acc, s) => acc + (s?.vm_count ?? 0) + (s?.ct_count ?? 0),
    0
  );
  const runningVMs = Object.values(nodeStatus).reduce(
    (acc, s) => acc + (s?.vm_running ?? 0) + (s?.ct_running ?? 0),
    0
  );
  const offlineNodes = nodes.length - onlineNodes;

  const statusValues = Object.values(nodeStatus).filter(Boolean);
  const avgCpu = statusValues.length > 0
    ? statusValues.reduce((acc, s) => acc + (s?.cpu_usage ?? 0), 0) / statusValues.length
    : 0;

  const featureSummaries = [
    {
      title: "Notifications",
      description: "Telegram, SMTP und Verlauf",
      icon: Bell,
      href: "/settings/notifications",
    },
    {
      title: "Netzwerk",
      description: "Ports, Devices, Anomalien",
      icon: Network,
      href: "/network",
    },
    {
      title: "Logs",
      description: "Filter, Export, Analyse",
      icon: ScrollText,
      href: "/logs",
    },
    {
      title: "Tasks",
      description: "Migrationen, Backups, Incidents",
      icon: ListChecks,
      href: "/task-center",
    },
  ];

  return (
    <div className="flex flex-col gap-5">
      <div className="grid gap-3 grid-cols-2 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard
          title="Server"
          value={nodes.length}
          subtitle={`${onlineNodes} online`}
          icon={Server}
          color="blue"
        />
        <KpiCard
          title="VMs / Container"
          value={totalVMs}
          subtitle={`${runningVMs} aktiv`}
          icon={ServerCog}
          color="green"
        />
        <KpiCard
          title="CPU-Durchschnitt"
          value={`${avgCpu.toFixed(1)}%`}
          subtitle="Alle Server"
          icon={Cpu}
          color={avgCpu > 80 ? "red" : avgCpu > 60 ? "orange" : "blue"}
        />
        <KpiCard
          title="Status"
          value={offlineNodes === 0 ? "Gesund" : `${offlineNodes} offline`}
          subtitle={offlineNodes === 0 ? "Alle Systeme operativ" : "Achtung erforderlich"}
          icon={offlineNodes === 0 ? Activity : AlertTriangle}
          color={offlineNodes === 0 ? "green" : "red"}
        />
      </div>

      <section className="flex flex-col gap-3">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 className="text-base font-semibold">Server-Flotte</h2>
            <p className="text-sm text-muted-foreground">Nodes und Workload-Zustand fuer den schnellen Drill-down.</p>
          </div>
          <Button variant="ghost" size="sm" asChild>
            <Link href="/health">
              <ShieldCheck className="mr-2 h-4 w-4" />
              VM-Gesundheit
            </Link>
          </Button>
        </div>
        <NodeGrid />
      </section>

      <section className="flex flex-col gap-3">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 className="text-base font-semibold">Funktionsbereiche</h2>
            <p className="text-sm text-muted-foreground">Direkte Einstiege in zentrale Betriebsfunktionen.</p>
          </div>
          <Button variant="outline" size="sm" asChild>
            <Link href="/settings/nodes">
              <Plus className="mr-2 h-4 w-4" />
              Server hinzufuegen
            </Link>
          </Button>
        </div>
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          {featureSummaries.map((feature) => {
            const Icon = feature.icon;

            return (
              <Link key={feature.href} href={feature.href} className="block h-full">
                <Card hover className="h-full">
                  <CardContent className="flex h-full flex-col gap-4 p-4">
                    <div className="flex items-start gap-3">
                      <div className="flex size-10 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
                        <Icon className="h-5 w-5" />
                      </div>
                      <div>
                        <h3 className="text-base font-semibold">{feature.title}</h3>
                        <p className="mt-1 text-sm text-muted-foreground">{feature.description}</p>
                      </div>
                    </div>
                    <span className="mt-auto inline-flex items-center text-sm font-medium text-muted-foreground">
                      Oeffnen
                      <ArrowRight className="ml-2 h-4 w-4" />
                    </span>
                  </CardContent>
                </Card>
              </Link>
            );
          })}
        </div>
      </section>
    </div>
  );
}
