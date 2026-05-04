# Prometheus SRE-Cockpit Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign the Prometheus frontend into a calm dark SRE cockpit that reduces visible clutter while preserving every existing function and route.

**Architecture:** Keep the current Next.js App Router, Zustand stores, shadcn-style UI primitives, and `AppLayout -> Sidebar + Header + Main` shell. Add a small ops UI layer for panels, status indicators, metric cells, dashboard summary logic, attention rows, and compact node fleet rows, then recompose the dashboard around attention-first information hierarchy.

**Tech Stack:** Next.js 15, React 19, TypeScript, Tailwind CSS 4, Zustand, Lucide React, existing local UI primitives in `src/components/ui`.

---

## Spec Reference

Implement against:

- `docs/superpowers/specs/2026-05-04-prometheus-sre-cockpit-redesign-design.md`

Design guardrails:

- Do not remove features, routes, workflows, nav entries, search, chat, settings, or node drill-downs.
- Reduce overload through grouping, hierarchy, density, tabs/collapsible affordances, compact link groups, and detail pages.
- Dark SRE cockpit is the primary visual direction; light mode stays readable.

## File Structure

Create:

- `frontend/src/components/ops/ops-panel.tsx` - shared low-noise panel primitive for operational surfaces.
- `frontend/src/components/ops/status-indicator.tsx` - consistent status dot/icon/label treatment.
- `frontend/src/components/ops/metric-cell.tsx` - compact metric display with tabular numbers.
- `frontend/src/components/dashboard/dashboard-summary.ts` - pure calculations for dashboard health and attention items.
- `frontend/src/components/dashboard/ops-status-bar.tsx` - top cluster status strip.
- `frontend/src/components/dashboard/attention-queue.tsx` - prioritized action/attention list.
- `frontend/src/components/dashboard/node-fleet-table.tsx` - compact fleet view preserving node drill-down.

Modify:

- `frontend/src/app/globals.css` - SRE cockpit tokens and surface utilities.
- `frontend/src/components/layout/app-layout.tsx` - calmer main canvas spacing.
- `frontend/src/components/layout/sidebar.tsx` - calmer dark nav, same groups and links.
- `frontend/src/components/layout/header.tsx` - quieter workbar.
- `frontend/src/components/dashboard/dashboard-overview.tsx` - replace equal-weight KPI/card grid with status, attention, fleet, and compact function links.
- `frontend/src/app/(dashboard)/page.tsx` - remove duplicated hero/KPI noise and let the dashboard composition own the first viewport.

Do not modify backend code in this phase.

---

### Task 1: SRE Cockpit Tokens And Surfaces

**Files:**

- Modify: `frontend/src/app/globals.css`

- [ ] **Step 1: Update dark theme tokens**

Replace the existing `.dark` token block in `frontend/src/app/globals.css` with this block:

```css
.dark {
  --background: oklch(0.118 0.009 255);
  --foreground: oklch(0.965 0.004 255);
  --card: oklch(0.17 0.011 255);
  --card-foreground: oklch(0.965 0.004 255);
  --popover: oklch(0.185 0.012 255);
  --popover-foreground: oklch(0.965 0.004 255);
  --primary: oklch(0.72 0.17 55);
  --primary-foreground: oklch(0.12 0.01 255);
  --secondary: oklch(0.225 0.013 255);
  --secondary-foreground: oklch(0.94 0.006 255);
  --muted: oklch(0.215 0.013 255);
  --muted-foreground: oklch(0.68 0.018 255);
  --accent: oklch(0.245 0.015 255);
  --accent-foreground: oklch(0.96 0.004 255);
  --destructive: oklch(0.61 0.22 27);
  --destructive-foreground: oklch(0.99 0 0);
  --border: oklch(0.285 0.016 255);
  --input: oklch(0.245 0.015 255);
  --ring: oklch(0.72 0.17 55);
  --sidebar: oklch(0.095 0.01 255);
  --sidebar-foreground: oklch(0.955 0.004 255);
  --sidebar-muted: oklch(0.63 0.018 255);
}
```

- [ ] **Step 2: Add ops utility classes**

Append these utilities inside the existing `@layer utilities` block:

```css
.ops-canvas {
  background:
    linear-gradient(180deg, oklch(1 0 0 / 0.02), transparent 220px),
    var(--background);
}

.ops-panel {
  @apply rounded-lg border bg-card text-card-foreground;
  border-color: oklch(0.4 0.014 255 / 0.45);
  box-shadow: 0 1px 0 oklch(1 0 0 / 0.04);
}

.dark .ops-panel {
  background: linear-gradient(180deg, oklch(0.19 0.012 255), oklch(0.162 0.011 255));
  border-color: oklch(0.34 0.016 255 / 0.8);
}

.ops-row {
  @apply rounded-md border bg-muted/35;
  border-color: oklch(0.4 0.014 255 / 0.35);
}

.ops-divider {
  @apply border-border/70;
}

.ops-focus-ring {
  @apply focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background;
}
```

- [ ] **Step 3: Run a syntax check through build tooling**

Run:

```bash
cd frontend
npm run build
```

Expected: the CSS parses. If unrelated API or environment failures appear, capture the exact error and continue after confirming no CSS parse error is present.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/app/globals.css
git commit -m "style: tune sre cockpit surfaces"
```

---

### Task 2: Shared Ops UI Components

**Files:**

- Create: `frontend/src/components/ops/ops-panel.tsx`
- Create: `frontend/src/components/ops/status-indicator.tsx`
- Create: `frontend/src/components/ops/metric-cell.tsx`

- [ ] **Step 1: Create `OpsPanel`**

Create `frontend/src/components/ops/ops-panel.tsx`:

```tsx
import * as React from "react";
import { cn } from "@/lib/utils";

interface OpsPanelProps extends React.HTMLAttributes<HTMLDivElement> {
  interactive?: boolean;
}

export const OpsPanel = React.forwardRef<HTMLDivElement, OpsPanelProps>(
  ({ className, interactive, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(
        "ops-panel",
        interactive && "card-hover",
        className
      )}
      {...props}
    />
  )
);
OpsPanel.displayName = "OpsPanel";

export function OpsPanelHeader({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        "flex flex-col gap-1 border-b ops-divider px-4 py-3",
        className
      )}
      {...props}
    />
  );
}

export function OpsPanelTitle({
  className,
  ...props
}: React.HTMLAttributes<HTMLHeadingElement>) {
  return (
    <h2
      className={cn("text-sm font-semibold tracking-tight", className)}
      {...props}
    />
  );
}

export function OpsPanelDescription({
  className,
  ...props
}: React.HTMLAttributes<HTMLParagraphElement>) {
  return (
    <p
      className={cn("text-xs text-muted-foreground", className)}
      {...props}
    />
  );
}

export function OpsPanelContent({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("p-4", className)} {...props} />;
}
```

- [ ] **Step 2: Create `StatusIndicator`**

Create `frontend/src/components/ops/status-indicator.tsx`:

```tsx
import type { ComponentType } from "react";
import { AlertTriangle, CheckCircle2, Circle, Info, XCircle } from "lucide-react";
import { cn } from "@/lib/utils";

export type StatusTone = "ok" | "warning" | "critical" | "info" | "muted";

const toneClasses: Record<StatusTone, string> = {
  ok: "text-emerald-500",
  warning: "text-amber-500",
  critical: "text-red-500",
  info: "text-sky-500",
  muted: "text-muted-foreground",
};

const dotClasses: Record<StatusTone, string> = {
  ok: "bg-emerald-500",
  warning: "bg-amber-500",
  critical: "bg-red-500",
  info: "bg-sky-500",
  muted: "bg-muted-foreground",
};

const icons = {
  ok: CheckCircle2,
  warning: AlertTriangle,
  critical: XCircle,
  info: Info,
  muted: Circle,
} satisfies Record<StatusTone, ComponentType<{ className?: string }>>;

interface StatusIndicatorProps {
  tone: StatusTone;
  label: string;
  description?: string;
  withIcon?: boolean;
  className?: string;
}

export function StatusIndicator({
  tone,
  label,
  description,
  withIcon = false,
  className,
}: StatusIndicatorProps) {
  const Icon = icons[tone];

  return (
    <span className={cn("inline-flex min-w-0 items-center gap-2", className)}>
      {withIcon ? (
        <Icon className={cn("h-4 w-4 shrink-0", toneClasses[tone])} />
      ) : (
        <span className={cn("h-2 w-2 shrink-0 rounded-full", dotClasses[tone])} />
      )}
      <span className="min-w-0">
        <span className="block truncate text-xs font-medium">{label}</span>
        {description && (
          <span className="block truncate text-[11px] text-muted-foreground">
            {description}
          </span>
        )}
      </span>
    </span>
  );
}
```

- [ ] **Step 3: Create `MetricCell`**

Create `frontend/src/components/ops/metric-cell.tsx`:

```tsx
import { cn } from "@/lib/utils";

interface MetricCellProps {
  label: string;
  value: string | number;
  helper?: string;
  tone?: "default" | "ok" | "warning" | "critical";
  className?: string;
}

const valueToneClasses = {
  default: "text-foreground",
  ok: "text-emerald-500",
  warning: "text-amber-500",
  critical: "text-red-500",
};

export function MetricCell({
  label,
  value,
  helper,
  tone = "default",
  className,
}: MetricCellProps) {
  return (
    <div className={cn("min-w-0", className)}>
      <p className="truncate text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
        {label}
      </p>
      <p className={cn("mt-1 truncate text-lg font-semibold tabular", valueToneClasses[tone])}>
        {value}
      </p>
      {helper && (
        <p className="mt-0.5 truncate text-xs text-muted-foreground">{helper}</p>
      )}
    </div>
  );
}
```

- [ ] **Step 4: Verify TypeScript imports**

Run:

```bash
cd frontend
npm run build
```

Expected: no TypeScript errors from the new `ops` components.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/ops
git commit -m "feat: add ops ui primitives"
```

---

### Task 3: Dashboard Summary And Attention Model

**Files:**

- Create: `frontend/src/components/dashboard/dashboard-summary.ts`

- [ ] **Step 1: Create pure summary helpers**

Create `frontend/src/components/dashboard/dashboard-summary.ts`:

```ts
import type { Node, NodeStatus } from "@/types/api";

export type AttentionSeverity = "critical" | "warning" | "info";

export interface AttentionItem {
  id: string;
  severity: AttentionSeverity;
  title: string;
  description: string;
  href: string;
}

export interface DashboardSummary {
  onlineNodes: number;
  offlineNodes: number;
  totalNodes: number;
  totalWorkloads: number;
  runningWorkloads: number;
  avgCpu: number;
  avgMemory: number;
  healthLabel: string;
  healthTone: "ok" | "warning" | "critical";
  attentionItems: AttentionItem[];
}

export function buildDashboardSummary(
  nodes: Node[],
  nodeStatus: Record<string, NodeStatus | undefined>
): DashboardSummary {
  const onlineNodes = nodes.filter((node) => node.is_online).length;
  const offlineNodes = nodes.length - onlineNodes;
  const statuses = Object.values(nodeStatus).filter(Boolean) as NodeStatus[];

  const totalWorkloads = statuses.reduce(
    (sum, status) => sum + status.vm_count + status.ct_count,
    0
  );
  const runningWorkloads = statuses.reduce(
    (sum, status) => sum + status.vm_running + status.ct_running,
    0
  );
  const avgCpu = average(statuses.map((status) => status.cpu_usage));
  const avgMemory = average(
    statuses.map((status) =>
      status.memory_total > 0 ? (status.memory_used / status.memory_total) * 100 : 0
    )
  );

  const attentionItems = buildAttentionItems(nodes, statuses, offlineNodes, avgCpu, avgMemory);
  const criticalCount = attentionItems.filter((item) => item.severity === "critical").length;
  const warningCount = attentionItems.filter((item) => item.severity === "warning").length;

  return {
    onlineNodes,
    offlineNodes,
    totalNodes: nodes.length,
    totalWorkloads,
    runningWorkloads,
    avgCpu,
    avgMemory,
    healthLabel:
      criticalCount > 0
        ? `${criticalCount} kritisch`
        : warningCount > 0
        ? `${warningCount} Hinweise`
        : "Cluster operativ",
    healthTone: criticalCount > 0 ? "critical" : warningCount > 0 ? "warning" : "ok",
    attentionItems,
  };
}

function buildAttentionItems(
  nodes: Node[],
  statuses: NodeStatus[],
  offlineNodes: number,
  avgCpu: number,
  avgMemory: number
): AttentionItem[] {
  const items: AttentionItem[] = [];

  if (offlineNodes > 0) {
    items.push({
      id: "offline-nodes",
      severity: "critical",
      title: `${offlineNodes} Node${offlineNodes === 1 ? "" : "s"} offline`,
      description: "Pruefen Sie Erreichbarkeit, Token und Netzwerkpfad.",
      href: "/nodes",
    });
  }

  const hotNodes = statuses.filter((status) => status.cpu_usage >= 80);
  if (hotNodes.length > 0) {
    items.push({
      id: "cpu-pressure",
      severity: "warning",
      title: `${hotNodes.length} Node${hotNodes.length === 1 ? "" : "s"} mit hoher CPU`,
      description: `Cluster-Durchschnitt ${avgCpu.toFixed(1)} Prozent.`,
      href: "/monitoring",
    });
  }

  if (avgMemory >= 80) {
    items.push({
      id: "memory-pressure",
      severity: "warning",
      title: "RAM-Auslastung erhoeht",
      description: `Durchschnittlich ${avgMemory.toFixed(1)} Prozent belegt.`,
      href: "/monitoring",
    });
  }

  if (items.length === 0 && nodes.length > 0) {
    items.push({
      id: "all-clear",
      severity: "info",
      title: "Keine akute Aufmerksamkeit",
      description: "Alle bekannten Nodes melden einen stabilen Grundzustand.",
      href: "/monitoring",
    });
  }

  if (nodes.length === 0) {
    items.push({
      id: "no-nodes",
      severity: "info",
      title: "Noch keine Nodes konfiguriert",
      description: "Fuegen Sie den ersten Proxmox Node in den Einstellungen hinzu.",
      href: "/settings/nodes",
    });
  }

  return items;
}

function average(values: number[]): number {
  if (values.length === 0) return 0;
  return values.reduce((sum, value) => sum + value, 0) / values.length;
}
```

- [ ] **Step 2: Verify helper compiles**

Run:

```bash
cd frontend
npm run build
```

Expected: no TypeScript errors from `dashboard-summary.ts`.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/dashboard/dashboard-summary.ts
git commit -m "feat: add dashboard summary model"
```

---

### Task 4: Dashboard Status Bar And Attention Queue

**Files:**

- Create: `frontend/src/components/dashboard/ops-status-bar.tsx`
- Create: `frontend/src/components/dashboard/attention-queue.tsx`

- [ ] **Step 1: Create `OpsStatusBar`**

Create `frontend/src/components/dashboard/ops-status-bar.tsx`:

```tsx
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
    <OpsPanel className="grid gap-4 p-4 lg:grid-cols-[1.2fr_repeat(4,minmax(0,1fr))]">
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
```

- [ ] **Step 2: Create `AttentionQueue`**

Create `frontend/src/components/dashboard/attention-queue.tsx`:

```tsx
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
```

- [ ] **Step 3: Verify components compile**

Run:

```bash
cd frontend
npm run build
```

Expected: no TypeScript errors from `ops-status-bar.tsx` or `attention-queue.tsx`.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/dashboard/ops-status-bar.tsx frontend/src/components/dashboard/attention-queue.tsx
git commit -m "feat: add dashboard attention surfaces"
```

---

### Task 5: Compact Node Fleet View

**Files:**

- Create: `frontend/src/components/dashboard/node-fleet-table.tsx`

- [ ] **Step 1: Create compact fleet component**

Create `frontend/src/components/dashboard/node-fleet-table.tsx`:

```tsx
"use client";

import Link from "next/link";
import { Server } from "lucide-react";
import { OpsPanel, OpsPanelContent, OpsPanelHeader, OpsPanelTitle } from "@/components/ops/ops-panel";
import { StatusIndicator } from "@/components/ops/status-indicator";
import { Skeleton } from "@/components/ui/skeleton";
import { formatBytes, formatPercentage } from "@/lib/utils";
import type { Node, NodeStatus } from "@/types/api";

interface NodeFleetTableProps {
  nodes: Node[];
  nodeStatus: Record<string, NodeStatus | undefined>;
  isLoading: boolean;
}

export function NodeFleetTable({ nodes, nodeStatus, isLoading }: NodeFleetTableProps) {
  if (isLoading) {
    return (
      <OpsPanel>
        <OpsPanelHeader>
          <OpsPanelTitle>Server-Flotte</OpsPanelTitle>
        </OpsPanelHeader>
        <OpsPanelContent className="space-y-2">
          {Array.from({ length: 4 }).map((_, index) => (
            <Skeleton key={index} className="h-12 rounded-md" />
          ))}
        </OpsPanelContent>
      </OpsPanel>
    );
  }

  return (
    <OpsPanel>
      <OpsPanelHeader className="flex-row items-center justify-between">
        <div>
          <OpsPanelTitle>Server-Flotte</OpsPanelTitle>
          <p className="mt-1 text-xs text-muted-foreground">
            Kompakte Uebersicht. Details bleiben per Klick auf die Node erreichbar.
          </p>
        </div>
        <Link href="/nodes" className="text-xs font-medium text-primary hover:underline">
          Alle Nodes
        </Link>
      </OpsPanelHeader>
      <OpsPanelContent className="space-y-2">
        {nodes.length === 0 ? (
          <Link
            href="/settings/nodes"
            className="ops-row ops-focus-ring flex items-center justify-between px-3 py-3 text-sm hover:bg-accent/60"
          >
            <span>Keine Nodes konfiguriert.</span>
            <span className="text-primary">Node hinzufuegen</span>
          </Link>
        ) : (
          nodes.map((node) => {
            const status = nodeStatus[node.id];
            const memUsage =
              status && status.memory_total > 0
                ? (status.memory_used / status.memory_total) * 100
                : 0;
            const diskUsage =
              status && status.disk_total > 0
                ? (status.disk_used / status.disk_total) * 100
                : 0;

            return (
              <Link
                key={node.id}
                href={`/nodes/${node.id}`}
                className="ops-row ops-focus-ring grid gap-3 px-3 py-2.5 transition-colors hover:bg-accent/60 md:grid-cols-[1.3fr_repeat(4,minmax(0,1fr))]"
              >
                <div className="flex min-w-0 items-center gap-2">
                  <Server className="h-4 w-4 shrink-0 text-muted-foreground" />
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium">{node.name}</p>
                    <p className="truncate text-xs text-muted-foreground">
                      {node.hostname}:{node.port}
                    </p>
                  </div>
                </div>
                <StatusIndicator
                  tone={node.is_online ? "ok" : "critical"}
                  label={node.is_online ? "Online" : "Offline"}
                />
                <FleetMetric label="CPU" value={status ? formatPercentage(status.cpu_usage) : "-"} />
                <FleetMetric label="RAM" value={status ? formatPercentage(memUsage) : "-"} />
                <FleetMetric
                  label="Disk"
                  value={status ? formatPercentage(diskUsage) : "-"}
                  helper={status ? formatBytes(status.disk_total) : undefined}
                />
              </Link>
            );
          })
        )}
      </OpsPanelContent>
    </OpsPanel>
  );
}

function FleetMetric({
  label,
  value,
  helper,
}: {
  label: string;
  value: string;
  helper?: string;
}) {
  return (
    <div className="min-w-0">
      <p className="text-[11px] uppercase tracking-wide text-muted-foreground">{label}</p>
      <p className="truncate text-sm font-medium tabular">{value}</p>
      {helper && <p className="truncate text-[11px] text-muted-foreground">{helper}</p>}
    </div>
  );
}
```

- [ ] **Step 2: Verify component compiles**

Run:

```bash
cd frontend
npm run build
```

Expected: no TypeScript errors from `node-fleet-table.tsx`.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/dashboard/node-fleet-table.tsx
git commit -m "feat: add compact node fleet"
```

---

### Task 6: Recompose Dashboard Overview

**Files:**

- Modify: `frontend/src/components/dashboard/dashboard-overview.tsx`

- [ ] **Step 1: Replace dashboard overview with attention-first composition**

Replace the full contents of `frontend/src/components/dashboard/dashboard-overview.tsx` with:

```tsx
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
```

- [ ] **Step 2: Verify no old imports remain**

Run:

```bash
cd frontend
npm run build
```

Expected: no unused import or TypeScript errors in `dashboard-overview.tsx`.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/dashboard/dashboard-overview.tsx
git commit -m "feat: recompose dashboard overview"
```

---

### Task 7: Calm The Dashboard Page

**Files:**

- Modify: `frontend/src/app/(dashboard)/page.tsx`

- [ ] **Step 1: Replace duplicate hero-heavy page layout**

Replace the full contents of `frontend/src/app/(dashboard)/page.tsx` with:

```tsx
"use client";

import { useEffect } from "react";
import { AgentActivityFeed } from "@/components/dashboard/agent-activity-feed";
import { AttentionBanner } from "@/components/dashboard/attention-banner";
import { BriefingWidget } from "@/components/dashboard/briefing-widget";
import { DashboardOverview } from "@/components/dashboard/dashboard-overview";
import { SecurityWidget } from "@/components/dashboard/security-widget";
import { StatusBadge } from "@/components/ui/status-badge";
import { useNodeStore } from "@/stores/node-store";

export default function DashboardPage() {
  const { nodes, fetchNodes } = useNodeStore();

  useEffect(() => {
    fetchNodes();
  }, [fetchNodes]);

  const onlineNodes = nodes.filter((node) => node.is_online).length;
  const offlineNodes = nodes.length - onlineNodes;
  const isHealthy = offlineNodes === 0;

  return (
    <div className="flex flex-col gap-4">
      <section className="flex flex-col gap-3 border-b ops-divider pb-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="eyebrow">{greetingFor(new Date())}</p>
          <h1 className="mt-1 text-2xl font-semibold tracking-tight">Lagezentrum</h1>
          <p className="mt-1 max-w-2xl text-sm text-muted-foreground">
            Prioritaeten, Clusterzustand und naechste Aktionen - ohne Funktionslaerm.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <StatusBadge tone={isHealthy ? "ok" : "warning"}>
            {isHealthy ? "Cluster operativ" : `${offlineNodes} offline`}
          </StatusBadge>
          <StatusBadge tone="muted" withIcon={false}>
            <span className="tabular">{onlineNodes}</span>/<span className="tabular">{nodes.length}</span> Nodes online
          </StatusBadge>
        </div>
      </section>

      <AttentionBanner />
      <DashboardOverview />

      <section className="grid gap-4 xl:grid-cols-2">
        <BriefingWidget />
        <SecurityWidget />
      </section>

      <AgentActivityFeed limit={12} pollInterval={15000} />
    </div>
  );
}

function greetingFor(d: Date): string {
  const hour = d.getHours();
  if (hour < 5) return "Gute Nacht";
  if (hour < 11) return "Guten Morgen";
  if (hour < 14) return "Mittag";
  if (hour < 18) return "Guten Tag";
  if (hour < 22) return "Guten Abend";
  return "Spaete Stunde";
}
```

- [ ] **Step 2: Confirm all previous dashboard functions remain reachable**

Check that these existing components/routes are still present:

- `AttentionBanner` remains on the page.
- `BriefingWidget` remains on the page.
- `SecurityWidget` remains on the page.
- `AgentActivityFeed` remains on the page.
- Quick links in `DashboardOverview` point to Backups, Netzwerk, Logs, Tasks, Benachrichtigungen, Sicherheit.
- Sidebar still contains the full route list.

Run:

```bash
cd frontend
npm run build
```

Expected: dashboard page compiles and the retained imports are used.

- [ ] **Step 3: Commit**

```bash
git add 'frontend/src/app/(dashboard)/page.tsx'
git commit -m "feat: calm dashboard first viewport"
```

---

### Task 8: Shell Visual Pass

**Files:**

- Modify: `frontend/src/components/layout/app-layout.tsx`
- Modify: `frontend/src/components/layout/header.tsx`
- Modify: `frontend/src/components/layout/sidebar.tsx`

- [ ] **Step 1: Update main canvas**

In `frontend/src/components/layout/app-layout.tsx`, change:

```tsx
<div className="flex h-screen overflow-hidden bg-background">
```

to:

```tsx
<div className="flex h-screen overflow-hidden bg-background ops-canvas">
```

and change:

```tsx
<main className="flex-1 overflow-auto p-4 md:p-6">{children}</main>
```

to:

```tsx
<main className="flex-1 overflow-auto px-4 py-4 md:px-6 md:py-5">{children}</main>
```

- [ ] **Step 2: Calm header surface**

In `frontend/src/components/layout/header.tsx`, change the `<header>` class from:

```tsx
className="sticky top-0 z-30 flex h-14 shrink-0 items-center gap-3 border-b border-border bg-background/80 px-4 backdrop-blur-md"
```

to:

```tsx
className="sticky top-0 z-30 flex h-14 shrink-0 items-center gap-3 border-b ops-divider bg-background/88 px-4 backdrop-blur-md"
```

Change the agent status button class from:

```tsx
className="flex items-center gap-1.5 rounded-full border border-border bg-card px-2.5 py-1 text-[11px] font-medium text-muted-foreground transition-colors hover:bg-accent"
```

to:

```tsx
className="ops-focus-ring flex items-center gap-1.5 rounded-md border border-border/70 bg-card/80 px-2.5 py-1 text-[11px] font-medium text-muted-foreground transition-colors hover:bg-accent"
```

- [ ] **Step 3: Calm sidebar surface without removing nav items**

In `frontend/src/components/layout/sidebar.tsx`, change the `<aside>` class from:

```tsx
className="flex h-screen w-64 flex-col border-r border-border bg-sidebar"
```

to:

```tsx
className="flex h-screen w-64 flex-col border-r ops-divider bg-sidebar"
```

Change the brand icon wrapper from:

```tsx
className="flex h-8 w-8 items-center justify-center rounded-md bg-gradient-to-br from-orange-500 to-rose-500 shadow-sm"
```

to:

```tsx
className="flex h-8 w-8 items-center justify-center rounded-md bg-primary text-primary-foreground shadow-sm"
```

Change active nav link classes inside `renderNavLink` from:

```tsx
? "bg-accent font-medium text-foreground"
```

to:

```tsx
? "bg-primary/12 font-medium text-foreground"
```

Change the active indicator span class from:

```tsx
className="absolute left-0 top-1.5 bottom-1.5 w-0.5 rounded-r-full bg-primary"
```

to:

```tsx
className="absolute left-0 top-1.5 bottom-1.5 w-0.5 rounded-r-full bg-primary"
```

The indicator class stays the same; the important change is removing the old loud active fill.

- [ ] **Step 4: Verify no navigation entries were removed**

Run:

```bash
cd frontend
npm run build
```

Expected: layout components compile. Manually check `sections` in `sidebar.tsx` still contains the same labels and hrefs as before this task.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/layout/app-layout.tsx frontend/src/components/layout/header.tsx frontend/src/components/layout/sidebar.tsx
git commit -m "style: calm app shell"
```

---

### Task 9: Browser Verification And Polish

**Files:**

- Modify files from earlier tasks only if browser verification reveals concrete visual issues.

- [ ] **Step 1: Run the development server**

Run:

```bash
cd frontend
npm run dev
```

Expected: Next.js starts and prints a local URL, usually `http://localhost:3000`.

- [ ] **Step 2: Open dashboard in the in-app browser**

Navigate to:

```text
http://localhost:3000
```

Expected:

- First viewport shows `Lagezentrum`, `AttentionBanner`, `OpsStatusBar`, `AttentionQueue`, compact direct entries, and part of the fleet or next section.
- It does not show a wall of equal-weight cards.
- Existing functions remain reachable through sidebar/search/quick links/detail links.

- [ ] **Step 3: Verify mobile layout**

Use a mobile-sized viewport in the browser.

Expected:

- Sidebar is accessible through the mobile menu.
- Status bar stacks cleanly.
- Attention rows and direct entries do not overflow.
- Node fleet rows stack without text collision.

- [ ] **Step 4: Fix concrete issues**

If text overflows in dashboard rows, tighten the relevant class by adding `min-w-0`, `truncate`, or responsive grid columns to the specific element. Example fix for a row child:

```tsx
<div className="min-w-0">
  <p className="truncate text-sm font-medium">{node.name}</p>
  <p className="truncate text-xs text-muted-foreground">{node.hostname}:{node.port}</p>
</div>
```

If a section feels too visually loud, remove hover lift from non-clickable surfaces by ensuring only clickable `OpsPanel` instances pass `interactive`.

- [ ] **Step 5: Run final checks**

Run:

```bash
cd frontend
npm run build
```

Then run:

```bash
cd frontend
npm run lint
```

Expected:

- `npm run build` passes, or only fails for an existing environment issue that is unrelated to this redesign and is documented in the final handoff.
- `npm run lint` passes, or reports the known Next.js lint command compatibility issue if present.

- [ ] **Step 6: Commit polish**

If Step 4 changed files:

```bash
git add frontend/src
git commit -m "fix: polish sre cockpit layout"
```

If Step 4 changed nothing, do not create an empty commit.

---

## Self-Review

Spec coverage:

- Dark SRE cockpit tokens: Task 1.
- Keep functions while reducing overload: Tasks 6, 7, 8, and Task 9 checks.
- App shell visual pass: Task 8.
- Attention-first dashboard: Tasks 3, 4, 6, 7.
- Reusable ops components: Task 2.
- Compact node fleet: Task 5.
- Testing and browser verification: Task 9.

Placeholder scan:

- This plan contains concrete file paths, code blocks, commands, expected outcomes, and no deferred implementation notes.

Type consistency:

- `StatusTone` is defined in `status-indicator.tsx` and reused in `attention-queue.tsx`.
- `DashboardSummary` and `AttentionItem` are defined in `dashboard-summary.ts` and reused by dashboard components.
- `Node` and `NodeStatus` come from existing `@/types/api`.
