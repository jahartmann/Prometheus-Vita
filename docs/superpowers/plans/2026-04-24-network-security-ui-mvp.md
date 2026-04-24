# Network Security UI MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first polished Network Security MVP by decluttering navigation, strengthening card visuals, and making existing network scan, port, device, anomaly, bandwidth, and VM/service data useful in the UI.

**Architecture:** Keep the backend mostly unchanged. Add a focused frontend normalization layer for scan results, then compose new network cockpit components around the existing Zustand store and API clients. Apply small design-system improvements through existing card/KPI/sidebar components instead of introducing a new UI framework.

**Tech Stack:** Next.js 15, React 19, TypeScript, Zustand, shadcn/Radix-style local components, Tailwind CSS v4 tokens, Go/Echo backend for verification only.

---

## File Structure

- Create `frontend/src/lib/network-scan-normalizer.ts`: Pure TypeScript utility that converts backend scan result variants into one `NormalizedPortEntry[]` model and calculates risk counts.
- Create `frontend/src/components/network/service-risk-badge.tsx`: Small reusable risk badge for ports and services.
- Create `frontend/src/components/network/network-security-overview.tsx`: KPI row for scan recency, port counts, device counts, anomalies, and bandwidth.
- Create `frontend/src/components/network/vm-service-analysis.tsx`: MVP VM traffic and service/port analysis tab using existing node VM APIs and metrics APIs.
- Modify `frontend/src/components/network/port-table.tsx`: Replace local parsing with the normalizer, add risk display, and support quick/full scan result shapes.
- Modify `frontend/src/app/(dashboard)/network/page.tsx`: Add cockpit overview, add the VM/service tab, and improve page header/tabs.
- Modify `frontend/src/stores/network-store.ts`: Add typed scan/device/anomaly interfaces needed by new components and expose derived-safe data.
- Modify `frontend/src/components/network/scan-status-bar.tsx`: Improve scan buttons/status wording and use semantic styling.
- Modify `frontend/src/components/layout/sidebar.tsx`: Reduce visible top-level noise and make active sections clearer.
- Modify `frontend/src/components/ui/card.tsx`: Adjust radius and base border/shadow treatment globally.
- Modify `frontend/src/components/ui/kpi-card.tsx`: Make KPI cards more expressive using semantic color variants.
- Modify `frontend/src/app/globals.css`: Tune color tokens and hover utilities without changing the application framework.
- Verify backend files under `backend/internal/service/netscan` and `backend/internal/api/handler/network_scan_handler.go`; only change them if frontend reveals a concrete contract bug.

---

### Task 1: Add Network Scan Normalizer

**Files:**
- Create: `frontend/src/lib/network-scan-normalizer.ts`
- Modify: `frontend/src/components/network/port-table.tsx`

- [ ] **Step 1: Create the normalizer with explicit types**

Add `frontend/src/lib/network-scan-normalizer.ts`:

```ts
export type PortRisk = "high" | "medium" | "low" | "info";

export interface NormalizedPortEntry {
  id: string;
  port: number;
  protocol: string;
  state: string;
  service?: string;
  version?: string;
  process?: string;
  source: string;
  sourceType: "node" | "device" | "vm" | "connection";
  localAddr?: string;
  peerAddr?: string;
  peerPort?: number;
  risk: PortRisk;
  riskReason: string;
}

export interface NormalizedScanSummary {
  ports: NormalizedPortEntry[];
  listeningCount: number;
  connectionCount: number;
  highRiskCount: number;
  mediumRiskCount: number;
  unknownServiceCount: number;
}

const HIGH_RISK_PORTS = new Set([
  21, 22, 23, 25, 53, 111, 135, 139, 389, 445, 3389, 5432, 5900, 6379,
  8006, 8086, 9200, 9300, 11211, 27017,
]);

const WELL_KNOWN_SERVICES: Record<number, string> = {
  22: "ssh",
  25: "smtp",
  53: "dns",
  80: "http",
  443: "https",
  445: "smb",
  3389: "rdp",
  5432: "postgres",
  6379: "redis",
  8006: "proxmox",
  8080: "http-alt",
  8443: "https-alt",
  27017: "mongodb",
};

function asRecord(value: unknown): Record<string, unknown> {
  return value && typeof value === "object" ? (value as Record<string, unknown>) : {};
}

function asArray(value: unknown): Record<string, unknown>[] {
  return Array.isArray(value) ? value.map(asRecord) : [];
}

function text(value: unknown): string | undefined {
  return typeof value === "string" && value.trim() ? value : undefined;
}

function numberValue(value: unknown): number {
  const n = Number(value);
  return Number.isFinite(n) ? n : 0;
}

export function getPortRisk(entry: {
  port: number;
  state?: string;
  service?: string;
  version?: string;
  sourceType?: string;
}): { risk: PortRisk; reason: string } {
  const state = (entry.state ?? "").toLowerCase();
  if (entry.sourceType === "connection") {
    return { risk: "info", reason: "Bestehende Verbindung, kein Listening-Port" };
  }
  if (state && state !== "open" && state !== "listen" && state !== "listening" && state !== "unconn") {
    return { risk: "info", reason: "Port ist nicht offen" };
  }
  if (HIGH_RISK_PORTS.has(entry.port)) {
    return { risk: "high", reason: "Administrations-, Datenbank- oder Infrastruktur-Port" };
  }
  if (!entry.service && !WELL_KNOWN_SERVICES[entry.port]) {
    return { risk: "medium", reason: "Offener Port ohne bekannte Dienstzuordnung" };
  }
  if (entry.service && !entry.version && entry.port !== 80 && entry.port !== 443) {
    return { risk: "medium", reason: "Dienst erkannt, aber ohne Versionsinformation" };
  }
  return { risk: "low", reason: "Bekannter Dienst mit niedriger MVP-Risikoeinstufung" };
}

function entryFromSocket(raw: Record<string, unknown>, source: string, sourceType: NormalizedPortEntry["sourceType"], index: number): NormalizedPortEntry {
  const port = numberValue(raw.port ?? raw.local_port);
  const protocol = (text(raw.protocol ?? raw.proto) ?? "tcp").toLowerCase();
  const state = text(raw.state) ?? (sourceType === "connection" ? "established" : "open");
  const service = text(raw.service) ?? WELL_KNOWN_SERVICES[port];
  const version = text(raw.version);
  const risk = getPortRisk({ port, state, service, version, sourceType });
  return {
    id: `${sourceType}-${source}-${protocol}-${port}-${index}`,
    port,
    protocol,
    state,
    service,
    version,
    process: text(raw.process),
    source,
    sourceType,
    localAddr: text(raw.local_addr),
    peerAddr: text(raw.peer_addr),
    peerPort: numberValue(raw.peer_port),
    risk: risk.risk,
    riskReason: risk.reason,
  };
}

function entryFromFullScan(raw: Record<string, unknown>, source: string, index: number): NormalizedPortEntry {
  const port = numberValue(raw.port ?? raw.portid);
  const protocol = (text(raw.protocol) ?? "tcp").toLowerCase();
  const state = text(raw.state) ?? "open";
  const service = text(raw.service ?? raw.service_name) ?? WELL_KNOWN_SERVICES[port];
  const version = text(raw.version ?? raw.service_version);
  const risk = getPortRisk({ port, state, service, version, sourceType: "node" });
  return {
    id: `full-${source}-${protocol}-${port}-${index}`,
    port,
    protocol,
    state,
    service,
    version,
    source,
    sourceType: source.startsWith("VM ") ? "vm" : "node",
    risk: risk.risk,
    riskReason: risk.reason,
  };
}

export function normalizeNetworkScanResults(results: unknown): NormalizedScanSummary {
  const obj = asRecord(results);
  const ports: NormalizedPortEntry[] = [];

  asArray(obj.listening_tcp).forEach((raw, index) => {
    ports.push(entryFromSocket(raw, "Node TCP", "node", index));
  });
  asArray(obj.listening_udp).forEach((raw, index) => {
    ports.push(entryFromSocket(raw, "Node UDP", "node", index));
  });
  asArray(obj.established).forEach((raw, index) => {
    ports.push(entryFromSocket(raw, "Established", "connection", index));
  });
  asArray(obj.ports).forEach((raw, index) => {
    ports.push(entryFromFullScan(raw, "Full Scan", index));
  });
  asArray(obj.vm_ports).forEach((raw, index) => {
    const vmid = text(raw.vmid) ?? String(raw.vmid ?? "");
    ports.push(entryFromFullScan(raw, `VM ${vmid}`.trim(), index));
  });
  asArray(obj.nmap_results).forEach((host, hostIndex) => {
    const source = text(host.ip ?? host.address) ?? `Host ${hostIndex + 1}`;
    asArray(host.ports).forEach((raw, index) => {
      ports.push(entryFromFullScan(raw, source, index));
    });
  });

  return {
    ports,
    listeningCount: ports.filter((p) => p.sourceType !== "connection").length,
    connectionCount: ports.filter((p) => p.sourceType === "connection").length,
    highRiskCount: ports.filter((p) => p.risk === "high").length,
    mediumRiskCount: ports.filter((p) => p.risk === "medium").length,
    unknownServiceCount: ports.filter((p) => !p.service && p.sourceType !== "connection").length,
  };
}
```

- [ ] **Step 2: Type-check the new utility**

Run:

```bash
cd frontend
npx tsc --noEmit
```

Expected: TypeScript may report pre-existing project errors. If it reports errors in `src/lib/network-scan-normalizer.ts`, fix them before continuing.

- [ ] **Step 3: Replace `PortTable` parsing with the normalizer**

In `frontend/src/components/network/port-table.tsx`, remove the local `PortEntry`, `WELL_KNOWN`, `getPortSeverity`, and `parseResultsJson` definitions. Import:

```ts
import { normalizeNetworkScanResults, type NormalizedPortEntry, type PortRisk } from "@/lib/network-scan-normalizer";
import { ServiceRiskBadge } from "@/components/network/service-risk-badge";
```

Change the `PortGroup` props to use `NormalizedPortEntry[]` and add `risk` to sorting:

```ts
type SortKey = "port" | "protocol" | "state" | "service" | "risk";

interface PortGroupProps {
  label: string;
  ports: NormalizedPortEntry[];
  filter: string;
  sortKey: SortKey;
  sortDir: SortDir;
}
```

Change the latest scan mapping:

```ts
const latestScan = scans[0];
const normalized = useMemo(
  () => normalizeNetworkScanResults(latestScan?.results_json),
  [latestScan]
);
const allPorts = normalized.ports;
```

In the row, render the risk badge:

```tsx
<TableCell className="w-28">
  <ServiceRiskBadge risk={p.risk} reason={p.riskReason} />
</TableCell>
```

Update the desktop header grid to include `Risiko`.

- [ ] **Step 4: Run frontend lint/type verification**

Run:

```bash
cd frontend
npm run lint
npx tsc --noEmit
```

Expected: No new errors from `port-table.tsx` or `network-scan-normalizer.ts`.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/network-scan-normalizer.ts frontend/src/components/network/port-table.tsx
git commit -m "feat: normalize network scan results"
```

---

### Task 2: Add Service Risk Badge

**Files:**
- Create: `frontend/src/components/network/service-risk-badge.tsx`
- Modify: `frontend/src/components/network/port-table.tsx`

- [ ] **Step 1: Create the badge component**

Add `frontend/src/components/network/service-risk-badge.tsx`:

```tsx
"use client";

import { ShieldAlert, ShieldCheck, AlertTriangle, Info } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { PortRisk } from "@/lib/network-scan-normalizer";

interface ServiceRiskBadgeProps {
  risk: PortRisk;
  reason: string;
}

const config = {
  high: { label: "Hoch", variant: "destructive" as const, icon: ShieldAlert },
  medium: { label: "Mittel", variant: "warning" as const, icon: AlertTriangle },
  low: { label: "Niedrig", variant: "success" as const, icon: ShieldCheck },
  info: { label: "Info", variant: "secondary" as const, icon: Info },
};

export function ServiceRiskBadge({ risk, reason }: ServiceRiskBadgeProps) {
  const item = config[risk];
  const Icon = item.icon;

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Badge variant={item.variant} className="gap-1">
          <Icon />
          {item.label}
        </Badge>
      </TooltipTrigger>
      <TooltipContent side="top">
        <p>{reason}</p>
      </TooltipContent>
    </Tooltip>
  );
}
```

- [ ] **Step 2: Ensure icons follow local button-free sizing**

If lint or visual review shows icon sizing is too large inside badges, add this class to the `Badge` call:

```tsx
className="gap-1 [&_svg]:size-3"
```

- [ ] **Step 3: Verify component import works**

Run:

```bash
cd frontend
npx tsc --noEmit
```

Expected: `ServiceRiskBadge` imports `PortRisk` and badge variants without type errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/network/service-risk-badge.tsx frontend/src/components/network/port-table.tsx
git commit -m "feat: show network service risk badges"
```

---

### Task 3: Add Network Security Overview

**Files:**
- Create: `frontend/src/components/network/network-security-overview.tsx`
- Modify: `frontend/src/app/(dashboard)/network/page.tsx`

- [ ] **Step 1: Create the overview component**

Add `frontend/src/components/network/network-security-overview.tsx`:

```tsx
"use client";

import { useEffect, useMemo, useState } from "react";
import { Activity, AlertTriangle, Gauge, Network, Radar, ShieldAlert } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { metricsApi } from "@/lib/api";
import { formatBandwidth, formatTraffic } from "@/lib/utils";
import { normalizeNetworkScanResults } from "@/lib/network-scan-normalizer";
import { useNetworkStore } from "@/stores/network-store";
import type { NetworkSummary } from "@/types/api";

interface NetworkSecurityOverviewProps {
  nodeId: string;
}

function compactDate(iso?: string) {
  if (!iso) return "Nie";
  return new Date(iso).toLocaleString("de-DE", {
    day: "2-digit",
    month: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function MetricTile({
  label,
  value,
  detail,
  icon: Icon,
  tone = "neutral",
}: {
  label: string;
  value: string | number;
  detail: string;
  icon: React.ComponentType;
  tone?: "neutral" | "good" | "warning" | "danger";
}) {
  const toneClass =
    tone === "danger"
      ? "border-red-500/30 bg-red-500/10 text-red-300"
      : tone === "warning"
        ? "border-orange-500/30 bg-orange-500/10 text-orange-300"
        : tone === "good"
          ? "border-green-500/30 bg-green-500/10 text-green-300"
          : "border-border bg-card text-card-foreground";

  return (
    <Card className={toneClass}>
      <CardContent className="flex items-center gap-3 p-4">
        <div className="flex size-9 items-center justify-center rounded-md bg-background/60">
          <Icon />
        </div>
        <div className="min-w-0">
          <p className="text-xs font-medium text-muted-foreground">{label}</p>
          <p className="truncate text-xl font-semibold">{value}</p>
          <p className="truncate text-xs text-muted-foreground">{detail}</p>
        </div>
      </CardContent>
    </Card>
  );
}

export function NetworkSecurityOverview({ nodeId }: NetworkSecurityOverviewProps) {
  const { scans, devices, anomalies } = useNetworkStore();
  const [summary, setSummary] = useState<NetworkSummary | null>(null);

  useEffect(() => {
    if (!nodeId) return;
    metricsApi
      .getNodeNetworkSummary(nodeId, "24h")
      .then((res) => setSummary(res.data as NetworkSummary))
      .catch(() => setSummary(null));
  }, [nodeId]);

  const quickScan = scans.find((scan) => scan.scan_type === "quick");
  const fullScan = scans.find((scan) => scan.scan_type === "full");
  const normalized = useMemo(
    () => normalizeNetworkScanResults(scans[0]?.results_json),
    [scans]
  );
  const openAnomalies = anomalies.filter((a) => !a.is_acknowledged).length;
  const riskTone = normalized.highRiskCount > 0 ? "danger" : normalized.mediumRiskCount > 0 ? "warning" : "good";

  return (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-6">
      <MetricTile label="Quick Scan" value={compactDate(quickScan?.started_at)} detail="Socket-Bestand" icon={Radar} />
      <MetricTile label="Full Scan" value={compactDate(fullScan?.started_at)} detail="Dienstversionen" icon={Network} />
      <MetricTile label="Offene Ports" value={normalized.listeningCount} detail={`${normalized.connectionCount} Verbindungen`} icon={Activity} />
      <MetricTile label="Risiko" value={normalized.highRiskCount} detail={`${normalized.mediumRiskCount} mittlere Treffer`} icon={ShieldAlert} tone={riskTone} />
      <MetricTile label="Geräte" value={devices.length} detail="Erkannt auf Node" icon={Gauge} />
      <MetricTile
        label="Traffic 24h"
        value={summary ? formatBandwidth(summary.avg_in_rate + summary.avg_out_rate) : "Keine Daten"}
        detail={summary ? formatTraffic(summary.total_in + summary.total_out) : "Node-Metriken fehlen"}
        icon={AlertTriangle}
        tone={openAnomalies > 0 ? "warning" : "neutral"}
      />
      {openAnomalies > 0 && (
        <div className="sm:col-span-2 xl:col-span-6">
          <Badge variant="warning">{openAnomalies} unbestätigte Netzwerk-Anomalien</Badge>
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Add overview to `/network`**

In `frontend/src/app/(dashboard)/network/page.tsx`, import:

```ts
import { NetworkSecurityOverview } from "@/components/network/network-security-overview";
```

Render it directly after `ScanStatusBar`:

```tsx
<ScanStatusBar nodeId={selectedNodeId} />
<NetworkSecurityOverview nodeId={selectedNodeId} />
```

- [ ] **Step 3: Verify**

Run:

```bash
cd frontend
npm run lint
npx tsc --noEmit
```

Expected: no new errors from `network-security-overview.tsx` or the network page.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/network/network-security-overview.tsx frontend/src/app/(dashboard)/network/page.tsx
git commit -m "feat: add network security overview"
```

---

### Task 4: Add VM Service Analysis MVP Tab

**Files:**
- Create: `frontend/src/components/network/vm-service-analysis.tsx`
- Modify: `frontend/src/app/(dashboard)/network/page.tsx`

- [ ] **Step 1: Create VM/service component**

Add `frontend/src/components/network/vm-service-analysis.tsx`:

```tsx
"use client";

import { useEffect, useMemo, useState } from "react";
import { ServerCog, ShieldAlert } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { metricsApi, nodeApi, toArray } from "@/lib/api";
import { vmApi } from "@/lib/vm-api";
import { formatBandwidth, formatTraffic } from "@/lib/utils";
import { getPortRisk } from "@/lib/network-scan-normalizer";
import { ServiceRiskBadge } from "@/components/network/service-risk-badge";
import type { NetworkSummary, VM, VMPort } from "@/types/api";

interface VMServiceAnalysisProps {
  nodeId: string;
}

interface VMServiceRow {
  vm: VM;
  summary: NetworkSummary | null;
  ports: VMPort[];
}

export function VMServiceAnalysis({ nodeId }: VMServiceAnalysisProps) {
  const [rows, setRows] = useState<VMServiceRow[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!nodeId) return;
    setLoading(true);
    nodeApi
      .getVMs(nodeId)
      .then(async (res) => {
        const vms = toArray<VM>(res.data).slice(0, 12);
        const resolved = await Promise.all(
          vms.map(async (vm) => {
            const [summaryResult, portsResult] = await Promise.allSettled([
              metricsApi.getVMNetworkSummary(nodeId, vm.vmid, "24h"),
              vmApi.getPorts(nodeId, vm.vmid),
            ]);
            return {
              vm,
              summary:
                summaryResult.status === "fulfilled"
                  ? (summaryResult.value.data as NetworkSummary)
                  : null,
              ports:
                portsResult.status === "fulfilled"
                  ? toArray<VMPort>(portsResult.value.data)
                  : [],
            };
          })
        );
        setRows(resolved);
      })
      .catch(() => setRows([]))
      .finally(() => setLoading(false));
  }, [nodeId]);

  const sorted = useMemo(() => {
    return [...rows].sort((a, b) => {
      const at = (a.summary?.total_in ?? 0) + (a.summary?.total_out ?? 0);
      const bt = (b.summary?.total_in ?? 0) + (b.summary?.total_out ?? 0);
      return bt - at;
    });
  }, [rows]);

  if (loading) {
    return <div className="py-10 text-center text-sm text-muted-foreground">Lade VM-/Service-Analyse...</div>;
  }

  if (sorted.length === 0) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center gap-2 py-12 text-muted-foreground">
          <ServerCog />
          <p className="text-sm">Keine VM-Daten für diese Node verfügbar.</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <ServerCog />
          VM-/Service-Analyse
        </CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>VM</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="text-right">Traffic 24h</TableHead>
              <TableHead className="text-right">Ø Bandbreite</TableHead>
              <TableHead>Dienste</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {sorted.map((row) => {
              const total = (row.summary?.total_in ?? 0) + (row.summary?.total_out ?? 0);
              const avg = (row.summary?.avg_in_rate ?? 0) + (row.summary?.avg_out_rate ?? 0);
              const riskyPorts = row.ports.filter((port) => getPortRisk({ port: port.port, state: port.state, service: port.process }).risk !== "low");
              return (
                <TableRow key={`${row.vm.vmid}-${row.vm.type}`}>
                  <TableCell>
                    <div className="font-medium">{row.vm.name || `VM ${row.vm.vmid}`}</div>
                    <div className="text-xs text-muted-foreground">{row.vm.type} · {row.vm.vmid}</div>
                  </TableCell>
                  <TableCell>
                    <Badge variant={row.vm.status === "running" ? "success" : "secondary"}>{row.vm.status}</Badge>
                  </TableCell>
                  <TableCell className="text-right font-mono text-sm">{formatTraffic(total)}</TableCell>
                  <TableCell className="text-right font-mono text-sm">{formatBandwidth(avg)}</TableCell>
                  <TableCell>
                    {row.ports.length === 0 ? (
                      <span className="text-xs text-muted-foreground">Keine Portdaten</span>
                    ) : (
                      <div className="flex flex-wrap gap-1">
                        {row.ports.slice(0, 5).map((port, index) => {
                          const risk = getPortRisk({ port: port.port, state: port.state, service: port.process });
                          return (
                            <ServiceRiskBadge
                              key={`${port.protocol}-${port.port}-${index}`}
                              risk={risk.risk}
                              reason={`${port.protocol.toUpperCase()} ${port.port}: ${risk.reason}`}
                            />
                          );
                        })}
                        {riskyPorts.length > 0 && (
                          <Badge variant="warning" className="gap-1">
                            <ShieldAlert />
                            {riskyPorts.length} prüfen
                          </Badge>
                        )}
                      </div>
                    )}
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
```

- [ ] **Step 2: Add the tab to `/network`**

In `frontend/src/app/(dashboard)/network/page.tsx`, import:

```ts
import { VMServiceAnalysis } from "@/components/network/vm-service-analysis";
```

Add a tab trigger:

```tsx
<TabsTrigger value="services" className="text-sm">VM-/Service-Analyse</TabsTrigger>
```

Add tab content:

```tsx
<TabsContent value="services" className="mt-4">
  <VMServiceAnalysis nodeId={selectedNodeId} />
</TabsContent>
```

Update the `activeTab` type in `frontend/src/stores/network-store.ts` to include `"services"`.

- [ ] **Step 3: Verify API names**

Use the existing `vmApi.getPorts` from `frontend/src/lib/vm-api.ts`. It calls the VM cockpit route:

```ts
getPorts: (nodeId: string, vmid: number) =>
  api.get<{ data: VMPort[] }>(`/nodes/${nodeId}/vms/${vmid}/cockpit/ports?type=lxc`),
```

Run:

```bash
cd frontend
npx tsc --noEmit
```

Expected: `VMServiceAnalysis` compiles with `vmApi` imported from `@/lib/vm-api`.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/network/vm-service-analysis.tsx frontend/src/app/(dashboard)/network/page.tsx frontend/src/stores/network-store.ts frontend/src/lib/vm-api.ts
git commit -m "feat: add vm service network analysis"
```

---

### Task 5: Polish Network Page Empty and Scan States

**Files:**
- Modify: `frontend/src/components/network/scan-status-bar.tsx`
- Modify: `frontend/src/components/network/device-table.tsx`
- Modify: `frontend/src/app/(dashboard)/network/page.tsx`

- [ ] **Step 1: Improve scan status copy**

In `scan-status-bar.tsx`, keep the existing trigger behavior but update labels:

```tsx
<span className="text-[10px] uppercase tracking-wide text-muted-foreground">Quick Scan</span>
```

Use:

```tsx
<Badge variant="secondary" className="text-[10px]">Socket-Inventar</Badge>
```

near the Quick Scan timestamp, and:

```tsx
<Badge variant="secondary" className="text-[10px]">Dienstversionen</Badge>
```

near the Full Scan timestamp.

- [ ] **Step 2: Make missing full scan explicit**

When `lastFull` is missing, render:

```tsx
<Badge variant="warning" className="text-[10px]">
  Full Scan fehlt oder nmap nicht verfügbar
</Badge>
```

- [ ] **Step 3: Add network page no-data guidance**

In `/network/page.tsx`, replace the plain “Kein Node ausgewählt” block with:

```tsx
<div className="rounded-lg border border-dashed p-10 text-center">
  <p className="text-sm font-medium">Kein Node ausgewählt</p>
  <p className="mt-1 text-sm text-muted-foreground">
    Wähle eine Node aus, um Ports, Geräte, Anomalien und Bandbreite zu prüfen.
  </p>
</div>
```

- [ ] **Step 4: Verify**

Run:

```bash
cd frontend
npm run lint
```

Expected: No lint errors in modified network components.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/network/scan-status-bar.tsx frontend/src/components/network/device-table.tsx frontend/src/app/(dashboard)/network/page.tsx
git commit -m "feat: clarify network scan states"
```

---

### Task 6: Declutter Sidebar

**Files:**
- Modify: `frontend/src/components/layout/sidebar.tsx`

- [ ] **Step 1: Reduce visible operations items**

In the `sections` constant, change the Cockpit/Operations items to:

```ts
items: [
  { label: "Dashboard", href: "/", icon: LayoutDashboard },
  { label: "Monitoring", href: "/monitoring", icon: BarChart3, matchPrefix: "/monitoring" },
  { label: "Alerts", href: "/alerts", icon: AlertTriangle, matchPrefix: "/alerts" },
  { label: "Task-Center", href: "/task-center", icon: ListChecks, matchPrefix: "/task-center" },
  { label: "Logs", href: "/logs", icon: FileText, matchPrefix: "/logs" },
],
```

- [ ] **Step 2: Move analysis-heavy items into Sicherheit & KI**

Change Sicherheit & KI items to:

```ts
items: [
  { label: "Sicherheit", href: "/security", icon: ShieldCheck, matchPrefix: "/security" },
  { label: "Netzwerk-Security", href: "/network", icon: Network, matchPrefix: "/network" },
  { label: "Root Cause", href: "/root-cause", icon: SearchCheck, matchPrefix: "/root-cause" },
  { label: "Drift-Erkennung", href: "/drift", icon: GitCompare, matchPrefix: "/drift" },
  { label: "VM-Gesundheit", href: "/health", icon: HeartPulse, matchPrefix: "/health" },
  { label: "KI-Chat", href: "/chat", icon: Bot, matchPrefix: "/chat" },
],
```

Remove the duplicate Netzwerk item from Infrastruktur so `/network` has one clear home.

- [ ] **Step 3: Keep secondary knowledge items compact**

Change System/Automatisierung items to:

```ts
items: [
  { label: "Topologie", href: "/topology", icon: GitBranch, matchPrefix: "/topology" },
  { label: "Reflex-Regeln", href: "/reflex", icon: Zap, matchPrefix: "/reflex" },
  { label: "Abhängigkeiten", href: "/dependencies", icon: Link2, matchPrefix: "/dependencies" },
  { label: "Knowledge Graph", href: "/knowledge-graph", icon: NetworkIcon, matchPrefix: "/knowledge-graph" },
  { label: "Reports", href: "/reports", icon: FileBarChart, matchPrefix: "/reports" },
  { label: "Tags", href: "/tags", icon: Tag, matchPrefix: "/tags" },
],
```

- [ ] **Step 4: Make link styling denser and active state clearer**

Change nav link classes from:

```ts
"flex items-center gap-3 rounded-md px-2.5 py-1.5 text-sm transition-colors"
```

to:

```ts
"flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-colors"
```

Change active class to:

```ts
"bg-primary text-primary-foreground font-semibold shadow-sm"
```

- [ ] **Step 5: Verify**

Run:

```bash
cd frontend
npm run lint
npx tsc --noEmit
```

Expected: no unused icon imports remain. Remove imports for icons no longer used.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/layout/sidebar.tsx
git commit -m "feat: declutter dashboard sidebar"
```

---

### Task 7: Strengthen Cards and KPI Visuals

**Files:**
- Modify: `frontend/src/components/ui/card.tsx`
- Modify: `frontend/src/components/ui/kpi-card.tsx`
- Modify: `frontend/src/app/globals.css`

- [ ] **Step 1: Adjust global card radius and base treatment**

In `card.tsx`, change:

```ts
"rounded-xl border bg-card text-card-foreground shadow-xs"
```

to:

```ts
"rounded-lg border bg-card text-card-foreground shadow-sm"
```

In `CardHeader`, replace `space-y-1.5` with `gap-1.5`:

```ts
className={cn("flex flex-col gap-1.5 p-5", className)}
```

In `CardContent`, reduce default padding:

```ts
className={cn("p-5 pt-0", className)}
```

- [ ] **Step 2: Add KPI color variants**

Replace `frontend/src/components/ui/kpi-card.tsx` with:

```tsx
import { Card, CardContent } from "@/components/ui/card";
import { cn } from "@/lib/utils";

interface KpiCardProps {
  title: string;
  value: number | string;
  subtitle?: string;
  icon: React.ComponentType<{ className?: string }>;
  color?: "blue" | "green" | "orange" | "red" | "purple" | "neutral" | string;
}

const colorClasses: Record<string, string> = {
  blue: "bg-blue-500/15 text-blue-600 dark:text-blue-300",
  green: "bg-green-500/15 text-green-600 dark:text-green-300",
  orange: "bg-orange-500/15 text-orange-600 dark:text-orange-300",
  red: "bg-red-500/15 text-red-600 dark:text-red-300",
  purple: "bg-violet-500/15 text-violet-600 dark:text-violet-300",
  neutral: "bg-muted text-muted-foreground",
};

export function KpiCard({ title, value, subtitle, icon: Icon, color = "neutral" }: KpiCardProps) {
  return (
    <Card hover>
      <CardContent className="flex items-center gap-4 p-4">
        <div className={cn("flex size-10 shrink-0 items-center justify-center rounded-md", colorClasses[color] ?? colorClasses.neutral)}>
          <Icon className="h-5 w-5" />
        </div>
        <div className="min-w-0">
          <p className="text-xs font-medium text-muted-foreground">{title}</p>
          <p className="truncate text-2xl font-semibold tracking-normal">{value}</p>
          {subtitle && <p className="truncate text-xs text-muted-foreground">{subtitle}</p>}
        </div>
      </CardContent>
    </Card>
  );
}
```

- [ ] **Step 3: Tune CSS tokens without one-note palette**

In `globals.css`, adjust only hover utility and dark card/border tokens:

```css
.dark {
  --card: oklch(0.19 0.012 286);
  --border: oklch(0.31 0.018 286);
  --sidebar: oklch(0.125 0.012 286);
}

@layer utilities {
  .card-hover {
    transition: transform 0.15s ease, box-shadow 0.15s ease, border-color 0.15s ease;
  }
  .card-hover:hover {
    transform: translateY(-1px);
    box-shadow: 0 8px 22px oklch(0 0 0 / 0.08);
  }
  .dark .card-hover:hover {
    box-shadow: 0 8px 22px oklch(0 0 0 / 0.35);
  }
}
```

- [ ] **Step 4: Verify**

Run:

```bash
cd frontend
npm run lint
npx tsc --noEmit
```

Expected: no component typing errors and no broken imports.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/ui/card.tsx frontend/src/components/ui/kpi-card.tsx frontend/src/app/globals.css
git commit -m "style: strengthen enterprise card visuals"
```

---

### Task 8: Final Verification

**Files:**
- Review all changed files

- [ ] **Step 1: Run frontend verification**

Run:

```bash
cd frontend
npm run lint
npx tsc --noEmit
```

Expected: no new frontend errors. If `npm run lint` fails because `next lint` is unsupported in Next.js 15, record the exact message and rely on `npx tsc --noEmit` plus manual review.

- [ ] **Step 2: Run backend formatting/tests**

Run:

```bash
cd backend
go fmt ./...
go test ./...
```

Expected: Go formatting produces no relevant diff unless backend files were touched. Tests pass, or any unrelated existing failures are documented with exact package and failure.

- [ ] **Step 3: Run local frontend**

Run:

```bash
cd frontend
npm run dev
```

Expected: Next.js starts and prints a local URL, normally `http://localhost:3000`.

- [ ] **Step 4: Manual browser acceptance**

Open `/network` and verify:

- Sidebar has one prominent Netzwerk-Security entry.
- Network page header, scan status, overview KPIs, tabs, and tables render without overlap.
- Quick scan JSON with `listening_tcp`, `listening_udp`, and `established` displays rows.
- Full scan JSON with `ports`, `service_name`, and `service_version` displays rows.
- Risk badges appear for administrative/database ports.
- VM-/Service-Analyse tab handles missing VM port data without crashing.
- Mobile width keeps cards and tabs readable.

- [ ] **Step 5: Final commit if verification changes were needed**

```bash
git status --short
git add frontend backend
git commit -m "chore: verify network security ui mvp"
```

Only make this commit if Step 1-4 required additional code fixes.
