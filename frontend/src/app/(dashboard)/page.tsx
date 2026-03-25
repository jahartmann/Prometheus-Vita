"use client";

import { useEffect } from "react";
import { useNodeStore } from "@/stores/node-store";
import { DashboardOverview } from "@/components/dashboard/dashboard-overview";
import { BriefingWidget } from "@/components/dashboard/briefing-widget";
import { SecurityWidget } from "@/components/dashboard/security-widget";
import { AttentionBanner } from "@/components/dashboard/attention-banner";

export default function DashboardPage() {
  const { fetchNodes } = useNodeStore();

  useEffect(() => {
    fetchNodes();
  }, [fetchNodes]);

  return (
    <div className="space-y-4 md:space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Dashboard</h1>
        <p className="text-muted-foreground">
          Übersicht über Ihre gesamte Infrastruktur.
        </p>
      </div>
      <AttentionBanner />
      <BriefingWidget />
      <SecurityWidget />
      <DashboardOverview />
    </div>
  );
}
