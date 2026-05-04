import { Activity } from "lucide-react";
import { MetricCell } from "@/components/ops/metric-cell";
import { OpsPanel } from "@/components/ops/ops-panel";
import { StatusIndicator } from "@/components/ops/status-indicator";
import type { DashboardSummary } from "./dashboard-summary";

interface OpsStatusBarProps {
  summary: DashboardSummary;
}

export function OpsStatusBar({ summary }: OpsStatusBarProps) {
  return (
    <OpsPanel className="grid gap-4 p-4 md:grid-cols-[1.6fr_repeat(4,minmax(0,1fr))]">
      <div className="flex min-w-0 items-center gap-3">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-md bg-primary/12 text-primary ring-1 ring-primary/25">
          <Activity className="h-5 w-5" />
        </div>
        <StatusIndicator
          tone={summary.healthTone}
          label={summary.healthLabel}
          description="Lage des Proxmox-Clusters"
          withIcon
        />
      </div>
      <MetricCell
        label="Nodes"
        value={`${summary.onlineNodes}/${summary.totalNodes}`}
        helper={summary.offlineNodes === 0 ? "online" : `${summary.offlineNodes} offline`}
        tone={summary.offlineNodes === 0 ? "ok" : "critical"}
      />
      <MetricCell
        label="Workloads"
        value={`${summary.runningWorkloads}/${summary.totalWorkloads}`}
        helper="VMs und Container"
      />
      <MetricCell
        label="CPU"
        value={`${summary.avgCpu.toFixed(1)}%`}
        helper="Durchschnitt"
        tone={summary.avgCpu >= 80 ? "warning" : "default"}
      />
      <MetricCell
        label="RAM"
        value={`${summary.avgMemory.toFixed(1)}%`}
        helper="Durchschnitt"
        tone={summary.avgMemory >= 80 ? "warning" : "default"}
      />
    </OpsPanel>
  );
}
