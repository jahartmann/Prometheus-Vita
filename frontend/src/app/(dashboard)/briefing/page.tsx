"use client";

import { useEffect, useState } from "react";
import { briefingApi } from "@/lib/api";
import type { LiveBriefingSummary } from "@/types/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { KpiCard } from "@/components/ui/kpi-card";
import {
  Server,
  ServerCog,
  Cpu,
  MemoryStick,
  HardDrive,
  Activity,
  AlertTriangle,
  TrendingUp,
  Clock,
} from "lucide-react";

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  if (days > 0) return `${days}d ${hours}h`;
  const mins = Math.floor((seconds % 3600) / 60);
  return `${hours}h ${mins}m`;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
}

function getGreeting(): string {
  const hour = new Date().getHours();
  if (hour < 12) return "Guten Morgen";
  if (hour < 18) return "Guten Tag";
  return "Guten Abend";
}

export default function BriefingPage() {
  const [data, setData] = useState<LiveBriefingSummary | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    briefingApi
      .getLive()
      .then((d) => setData(d))
      .catch(() => setError("Briefing konnte nicht geladen werden"))
      .finally(() => setIsLoading(false));
  }, []);

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{getGreeting()}</h1>
          <p className="text-muted-foreground">Lade Briefing...</p>
        </div>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {[...Array(4)].map((_, i) => (
            <Card key={i}>
              <CardContent className="p-5">
                <div className="h-16 animate-pulse rounded bg-muted" />
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{getGreeting()}</h1>
          <p className="text-muted-foreground">Ihr Infrastruktur-Briefing</p>
        </div>
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">
            {error || "Keine Daten verfuegbar"}
          </CardContent>
        </Card>
      </div>
    );
  }

  const healthStatus =
    data.nodes_offline === 0 && data.unresolved_anomalies === 0
      ? "healthy"
      : data.nodes_offline > 0
        ? "critical"
        : "warning";

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{getGreeting()}</h1>
          <p className="text-muted-foreground">
            {new Date().toLocaleDateString("de-DE", {
              weekday: "long",
              year: "numeric",
              month: "long",
              day: "numeric",
            })}
          </p>
        </div>
        <Badge
          variant={
            healthStatus === "healthy"
              ? "success"
              : healthStatus === "critical"
                ? "destructive"
                : "warning"
          }
          className="text-sm px-3 py-1"
        >
          {healthStatus === "healthy"
            ? "Alle Systeme operativ"
            : healthStatus === "critical"
              ? "Achtung erforderlich"
              : "Warnungen vorhanden"}
        </Badge>
      </div>

      {/* KPI Cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard
          title="Server"
          value={`${data.nodes_online}/${data.nodes_total}`}
          subtitle={
            data.nodes_offline > 0
              ? `${data.nodes_offline} offline`
              : "Alle online"
          }
          icon={Server}
          color={data.nodes_offline > 0 ? "red" : "green"}
        />
        <KpiCard
          title="VMs / Container"
          value={data.vms_total}
          subtitle={`${data.vms_running} aktiv, ${data.vms_stopped} gestoppt`}
          icon={ServerCog}
          color="blue"
        />
        <KpiCard
          title="CPU-Durchschnitt"
          value={`${data.avg_cpu.toFixed(1)}%`}
          subtitle="Cluster-weit"
          icon={Cpu}
          color={data.avg_cpu > 80 ? "red" : data.avg_cpu > 60 ? "orange" : "green"}
        />
        <KpiCard
          title="RAM-Durchschnitt"
          value={`${data.avg_ram.toFixed(1)}%`}
          subtitle="Cluster-weit"
          icon={MemoryStick}
          color={data.avg_ram > 85 ? "red" : data.avg_ram > 70 ? "orange" : "green"}
        />
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Top Nodes by CPU */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-base">
              <TrendingUp className="h-4 w-4 text-muted-foreground" />
              Top Nodes nach CPU-Last
            </CardTitle>
          </CardHeader>
          <CardContent>
            {data.top_nodes_by_cpu && data.top_nodes_by_cpu.length > 0 ? (
              <div className="space-y-3">
                {data.top_nodes_by_cpu.map((node, i) => (
                  <div key={node.node_id} className="flex items-center gap-3">
                    <span className="flex h-6 w-6 items-center justify-center rounded-full bg-muted text-xs font-bold">
                      {i + 1}
                    </span>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center justify-between">
                        <span className="font-medium truncate">{node.node_name}</span>
                        <span
                          className={`text-sm font-mono ${
                            node.cpu_usage > 80
                              ? "text-red-500"
                              : node.cpu_usage > 60
                                ? "text-orange-500"
                                : "text-green-500"
                          }`}
                        >
                          {node.cpu_usage.toFixed(1)}%
                        </span>
                      </div>
                      <div className="mt-1.5 h-1.5 w-full rounded-full bg-muted">
                        <div
                          className={`h-full rounded-full transition-all ${
                            node.cpu_usage > 80
                              ? "bg-red-500"
                              : node.cpu_usage > 60
                                ? "bg-orange-500"
                                : "bg-green-500"
                          }`}
                          style={{ width: `${Math.min(node.cpu_usage, 100)}%` }}
                        />
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">Keine Daten verfuegbar</p>
            )}
          </CardContent>
        </Card>

        {/* Top VMs by RAM */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-base">
              <MemoryStick className="h-4 w-4 text-muted-foreground" />
              Top VMs nach RAM-Nutzung
            </CardTitle>
          </CardHeader>
          <CardContent>
            {data.top_vms_by_ram && data.top_vms_by_ram.length > 0 ? (
              <div className="space-y-3">
                {data.top_vms_by_ram.map((vm, i) => (
                  <div key={`${vm.node_id}-${vm.vmid}`} className="flex items-center gap-3">
                    <span className="flex h-6 w-6 items-center justify-center rounded-full bg-muted text-xs font-bold">
                      {i + 1}
                    </span>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center justify-between">
                        <div className="truncate">
                          <span className="font-medium">{vm.vm_name || `VM ${vm.vmid}`}</span>
                          <span className="text-xs text-muted-foreground ml-2">
                            auf {vm.node_name}
                          </span>
                        </div>
                        <span className="text-sm font-mono text-muted-foreground">
                          {formatBytes(vm.mem_used)} / {formatBytes(vm.mem_total)}
                        </span>
                      </div>
                      <div className="mt-1.5 h-1.5 w-full rounded-full bg-muted">
                        <div
                          className={`h-full rounded-full transition-all ${
                            vm.mem_used_pct > 90
                              ? "bg-red-500"
                              : vm.mem_used_pct > 75
                                ? "bg-orange-500"
                                : "bg-blue-500"
                          }`}
                          style={{ width: `${Math.min(vm.mem_used_pct, 100)}%` }}
                        />
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">Keine laufenden VMs</p>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Alerts & Warnings */}
      {(data.unresolved_anomalies > 0 || data.critical_predictions > 0) && (
        <div className="grid gap-4 sm:grid-cols-2">
          {data.unresolved_anomalies > 0 && (
            <Card className="border-orange-500/30">
              <CardContent className="flex items-center gap-4 p-5">
                <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-orange-500/15">
                  <AlertTriangle className="h-5 w-5 text-orange-500" />
                </div>
                <div>
                  <p className="text-2xl font-bold">{data.unresolved_anomalies}</p>
                  <p className="text-sm text-muted-foreground">Ungeloeste Anomalien</p>
                </div>
              </CardContent>
            </Card>
          )}
          {data.critical_predictions > 0 && (
            <Card className="border-red-500/30">
              <CardContent className="flex items-center gap-4 p-5">
                <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-red-500/15">
                  <Activity className="h-5 w-5 text-red-500" />
                </div>
                <div>
                  <p className="text-2xl font-bold">{data.critical_predictions}</p>
                  <p className="text-sm text-muted-foreground">Kritische Vorhersagen</p>
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      )}

      {/* Node Details */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="flex items-center gap-2 text-base">
            <Server className="h-4 w-4 text-muted-foreground" />
            Node-Uebersicht
          </CardTitle>
        </CardHeader>
        <CardContent>
          {data.node_details && data.node_details.length > 0 ? (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b text-left text-muted-foreground">
                    <th className="pb-2 font-medium">Node</th>
                    <th className="pb-2 font-medium">Status</th>
                    <th className="pb-2 font-medium text-right">CPU</th>
                    <th className="pb-2 font-medium text-right">RAM</th>
                    <th className="pb-2 font-medium text-right">Disk</th>
                    <th className="pb-2 font-medium text-right">VMs</th>
                    <th className="pb-2 font-medium text-right">Uptime</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {data.node_details.map((node) => (
                    <tr key={node.node_id} className="hover:bg-muted/50">
                      <td className="py-2.5 font-medium">{node.node_name}</td>
                      <td className="py-2.5">
                        <Badge
                          variant={node.is_online ? "success" : "destructive"}
                          className="text-xs"
                        >
                          {node.is_online ? "Online" : "Offline"}
                        </Badge>
                      </td>
                      <td className="py-2.5 text-right font-mono">
                        {node.is_online ? `${(node.cpu_usage ?? 0).toFixed(1)}%` : "-"}
                      </td>
                      <td className="py-2.5 text-right font-mono">
                        {node.is_online ? `${(node.mem_pct ?? 0).toFixed(1)}%` : "-"}
                      </td>
                      <td className="py-2.5 text-right font-mono">
                        {node.is_online ? `${(node.disk_pct ?? 0).toFixed(1)}%` : "-"}
                      </td>
                      <td className="py-2.5 text-right">
                        {node.is_online
                          ? `${node.vm_running ?? 0}/${node.vm_count ?? 0}`
                          : "-"}
                      </td>
                      <td className="py-2.5 text-right text-muted-foreground">
                        <span className="flex items-center justify-end gap-1">
                          <Clock className="h-3 w-3" />
                          {node.is_online && node.uptime > 0
                            ? formatUptime(node.uptime)
                            : "-"}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">Keine Nodes konfiguriert</p>
          )}
        </CardContent>
      </Card>

      {/* Disk Usage */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="flex items-center gap-2 text-base">
            <HardDrive className="h-4 w-4 text-muted-foreground" />
            Speicher-Uebersicht
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-4">
            <div className="flex-1">
              <div className="flex items-center justify-between mb-1.5">
                <span className="text-sm text-muted-foreground">Durchschnittliche Disk-Auslastung</span>
                <span className="text-sm font-mono font-medium">
                  {data.avg_disk.toFixed(1)}%
                </span>
              </div>
              <div className="h-2.5 w-full rounded-full bg-muted">
                <div
                  className={`h-full rounded-full transition-all ${
                    data.avg_disk > 90
                      ? "bg-red-500"
                      : data.avg_disk > 75
                        ? "bg-orange-500"
                        : "bg-green-500"
                  }`}
                  style={{ width: `${Math.min(data.avg_disk, 100)}%` }}
                />
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
