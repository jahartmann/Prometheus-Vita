"use client";

import { useEffect, useMemo, useState } from "react";
import { ServerCog, ShieldAlert } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ServiceRiskBadge } from "@/components/network/service-risk-badge";
import { metricsApi, nodeApi, toArray } from "@/lib/api";
import { getPortRisk, type PortRisk } from "@/lib/network-scan-normalizer";
import { formatBandwidth, formatTraffic } from "@/lib/utils";
import { vmCockpitApi } from "@/lib/vm-api";
import type { NetworkSummary, VM, VMPort } from "@/types/api";

interface VMServiceAnalysisProps {
  nodeId: string;
}

interface PortRiskEntry {
  key: string;
  label: string;
  risk: PortRisk;
  reason: string;
}

interface VMServiceRow {
  vm: VM;
  summary: NetworkSummary | null;
  ports: VMPort[];
  portsAvailable: boolean;
}

function isNetworkSummary(value: unknown): value is NetworkSummary {
  return Boolean(value && typeof value === "object" && "total_in" in value && "total_out" in value);
}

function statusVariant(status: VM["status"]) {
  if (status === "running") return "success" as const;
  if (status === "stopped") return "maintenance" as const;
  return "warning" as const;
}

function portRisks(ports: VMPort[]): PortRiskEntry[] {
  return ports.map((port, index) => {
    const risk = getPortRisk({
      port: port.port,
      state: "open",
      service: port.process || undefined,
      sourceType: "vm",
    });

    return {
      key: `${port.protocol}-${port.address}-${port.port}-${index}`,
      label: `${port.port}/${port.protocol || "tcp"}`,
      risk: risk.risk,
      reason: risk.reason,
    };
  });
}

export function VMServiceAnalysis({ nodeId }: VMServiceAnalysisProps) {
  const [rows, setRows] = useState<VMServiceRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [vmLoadFailed, setVMLoadFailed] = useState(false);

  useEffect(() => {
    let cancelled = false;

    setRows([]);
    setVMLoadFailed(false);

    if (!nodeId) {
      setLoading(false);
      return () => {
        cancelled = true;
      };
    }

    setLoading(true);

    nodeApi
      .getVMs(nodeId)
      .then(async (res) => {
        const vms = toArray<VM>(res.data).slice(0, 12);
        if (cancelled) return;

        const vmRows = await Promise.all(
          vms.map(async (vm) => {
            const [summaryResult, portsResult] = await Promise.allSettled([
              metricsApi.getVMNetworkSummary(nodeId, vm.vmid, "24h"),
              vmCockpitApi.getPorts(nodeId, vm.vmid, vm.type),
            ]);

            const summary =
              summaryResult.status === "fulfilled" && isNetworkSummary(summaryResult.value.data)
                ? summaryResult.value.data
                : null;
            const ports =
              portsResult.status === "fulfilled"
                ? toArray<VMPort>(portsResult.value.data)
                : [];

            return {
              vm,
              summary,
              ports,
              portsAvailable: portsResult.status === "fulfilled",
            };
          })
        );

        vmRows.sort((a, b) => {
          const aTotal = (a.summary?.total_in ?? 0) + (a.summary?.total_out ?? 0);
          const bTotal = (b.summary?.total_in ?? 0) + (b.summary?.total_out ?? 0);
          return bTotal - aTotal;
        });

        if (!cancelled) {
          setRows(vmRows);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setVMLoadFailed(true);
          setRows([]);
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [nodeId]);

  const vmCount = rows.length;
  const riskyVMCount = useMemo(
    () =>
      rows.filter((row) =>
        portRisks(row.ports).some((port) => port.risk === "high" || port.risk === "medium")
      ).length,
    [rows]
  );

  return (
    <Card className="border-border bg-card">
      <CardHeader className="flex-row items-center gap-3 space-y-0">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-blue-500/10">
          <ServerCog className="h-5 w-5 text-blue-400" />
        </div>
        <div className="min-w-0 flex-1">
          <CardTitle className="text-base">VM-/Service-Analyse</CardTitle>
          <p className="mt-1 text-xs text-muted-foreground">
            24h-Traffic und Cockpit-Portdaten der ersten 12 VMs
          </p>
        </div>
        {riskyVMCount > 0 && (
          <Badge className="gap-1 border-orange-500/30 bg-orange-500/20 text-orange-300">
            <ShieldAlert className="h-3 w-3" />
            {riskyVMCount} prüfen
          </Badge>
        )}
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="flex items-center justify-center py-12 text-sm text-muted-foreground">
            VM-Services werden geladen...
          </div>
        ) : vmLoadFailed ? (
          <div className="flex items-center justify-center py-12 text-sm text-muted-foreground">
            VM-Daten konnten nicht geladen werden.
          </div>
        ) : vmCount === 0 ? (
          <div className="flex items-center justify-center py-12 text-sm text-muted-foreground">
            Keine VM-Daten für diesen Node.
          </div>
        ) : (
          <Table aria-label="VM-Service-Analyse mit Traffic, Bandbreite und Port-Risiken">
            <TableHeader>
              <TableRow className="border-border">
                <TableHead>VM</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Traffic 24h</TableHead>
                <TableHead className="text-right">Ø Bandbreite</TableHead>
                <TableHead>Service-/Port-Risiko</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row) => {
                const totalTraffic = (row.summary?.total_in ?? 0) + (row.summary?.total_out ?? 0);
                const avgBandwidth = (row.summary?.avg_in_rate ?? 0) + (row.summary?.avg_out_rate ?? 0);
                const risks = portRisks(row.ports);
                const visibleRisks = risks.slice(0, 5);
                const hasRisk = risks.some((port) => port.risk === "high" || port.risk === "medium");

                return (
                  <TableRow key={`${row.vm.type}-${row.vm.vmid}`} className="border-border">
                    <TableCell>
                      <div className="min-w-0">
                        <p className="truncate text-sm font-medium text-foreground">
                          {row.vm.name || `VM ${row.vm.vmid}`}
                        </p>
                        <p className="text-xs text-muted-foreground">
                          {row.vm.type.toUpperCase()} · {row.vm.vmid}
                        </p>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={statusVariant(row.vm.status)} className="text-xs">
                        {row.vm.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right font-mono text-sm">
                      <div>{formatTraffic(totalTraffic)}</div>
                      {!row.summary && (
                        <span className="text-[11px] text-muted-foreground">Keine Metriken</span>
                      )}
                    </TableCell>
                    <TableCell className="text-right font-mono text-sm">
                      {formatBandwidth(avgBandwidth)}
                    </TableCell>
                    <TableCell>
                      {visibleRisks.length > 0 ? (
                        <div className="flex flex-wrap items-center gap-1.5">
                          {visibleRisks.map((port) => (
                            <span key={port.key} className="inline-flex items-center gap-1">
                              <span className="font-mono text-xs text-muted-foreground">{port.label}</span>
                              <ServiceRiskBadge risk={port.risk} reason={port.reason} />
                            </span>
                          ))}
                          {risks.length > visibleRisks.length && (
                            <Badge variant="secondary" className="text-xs">
                              +{risks.length - visibleRisks.length}
                            </Badge>
                          )}
                          {hasRisk && (
                            <Badge className="border-orange-500/30 bg-orange-500/20 text-orange-300 text-xs">
                              prüfen
                            </Badge>
                          )}
                        </div>
                      ) : (
                        <span className="text-xs text-muted-foreground">
                          {row.portsAvailable ? "Keine Portdaten" : "Portdaten nicht verfügbar"}
                        </span>
                      )}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}
