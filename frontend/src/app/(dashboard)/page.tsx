"use client";

import { useEffect } from "react";
import { useNodeStore } from "@/stores/node-store";
import { DashboardOverview } from "@/components/dashboard/dashboard-overview";

export default function DashboardPage() {
  const { fetchNodes } = useNodeStore();

  useEffect(() => {
    fetchNodes();
  }, [fetchNodes]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Dashboard</h1>
        <p className="text-muted-foreground">
          Uebersicht ueber Ihre gesamte Infrastruktur.
        </p>
      </div>
      <DashboardOverview />
    </div>
  );
}
