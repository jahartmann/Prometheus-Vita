"use client";

import { useEffect, useState, useMemo, useCallback } from "react";
import { useRouter } from "next/navigation";
import {
  Database,
  HardDrive,
  TrendingUp,
  AlertTriangle,
  Sparkles,
  ArrowUpDown,
  Filter,
  ChevronDown,
  ChevronRight,
  Layers,
  Server,
  Info,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Skeleton } from "@/components/ui/skeleton";
import { formatBytes } from "@/lib/utils";
import { storageApi, toArray } from "@/lib/api";
import type { ClusterStorageItem } from "@/types/api";
import { SnapshotAnalysis } from "@/components/storage/snapshot-analysis";

// --- Types ---

type SortField = "storage" | "type" | "node_name" | "total" | "used" | "available" | "usage_percent" | "content";
type SortDir = "asc" | "desc";

interface Recommendation {
  id: string;
  severity: "critical" | "warning" | "info";
  title: string;
  description: string;
  action: string;
}

// --- Constants ---

const storageTypeLabels: Record<string, string> = {
  zfspool: "ZFS Pool",
  dir: "Verzeichnis",
  lvm: "LVM",
  lvmthin: "LVM-Thin",
  nfs: "NFS",
  cifs: "CIFS/SMB",
  cephfs: "CephFS",
  rbd: "Ceph RBD",
  btrfs: "BTRFS",
  zfs: "ZFS (Dataset)",
  iscsi: "iSCSI",
  iscsidirect: "iSCSI Direct",
  glusterfs: "GlusterFS",
  pbs: "PBS",
};

const ALL_TYPES = ["zfspool", "dir", "lvm", "lvmthin", "nfs", "cifs", "cephfs", "rbd", "btrfs", "zfs", "iscsi", "glusterfs", "pbs"];

function getTypeLabel(type: string): string {
  return storageTypeLabels[type] || type.toUpperCase();
}

function getUsageColor(pct: number): string {
  if (pct >= 85) return "#ef4444";
  if (pct >= 60) return "#f59e0b";
  return "#22c55e";
}

function getUsageBg(pct: number): string {
  if (pct >= 85) return "bg-red-500";
  if (pct >= 60) return "bg-amber-500";
  return "bg-green-500";
}

function getUsageBgLight(pct: number): string {
  if (pct >= 85) return "bg-red-500/15 hover:bg-red-500/25 border-red-500/20";
  if (pct >= 60) return "bg-amber-500/15 hover:bg-amber-500/25 border-amber-500/20";
  return "bg-green-500/15 hover:bg-green-500/25 border-green-500/20";
}

// --- Recommendations Engine ---

function computeRecommendations(items: ClusterStorageItem[]): Recommendation[] {
  const recs: Recommendation[] = [];

  // Near capacity (>80%)
  const nearCapacity = items.filter((s) => s.usage_percent >= 80 && s.active);
  for (const s of nearCapacity) {
    const severity = s.usage_percent >= 90 ? "critical" : "warning";
    recs.push({
      id: `cap-${s.node_name}-${s.storage}`,
      severity,
      title: `${s.storage} auf ${s.node_name} fast voll (${s.usage_percent.toFixed(1)}%)`,
      description: `Der Speicher "${s.storage}" hat nur noch ${formatBytes(s.available)} frei von insgesamt ${formatBytes(s.total)}.`,
      action: severity === "critical"
        ? "Dringend: Daten bereinigen oder Speicher erweitern"
        : "Speichernutzung pruefen und ggf. bereinigen",
    });
  }

  // Thin provisioning overcommit (LVM-Thin check)
  const thinPools = items.filter((s) => s.type === "lvmthin");
  for (const tp of thinPools) {
    if (tp.usage_percent >= 70) {
      recs.push({
        id: `thin-${tp.node_name}-${tp.storage}`,
        severity: "warning",
        title: `LVM-Thin Overcommit: ${tp.storage} auf ${tp.node_name}`,
        description: `Thin-Provisioned Pool ist zu ${tp.usage_percent.toFixed(1)}% belegt. Overcommit-Risiko besteht.`,
        action: "Thin-Pool-Nutzung pruefen, VMs mit hohem Verbrauch identifizieren",
      });
    }
  }

  // Unbalanced distribution
  const nodeUsageMap = new Map<string, number[]>();
  for (const s of items) {
    if (!s.active || s.total === 0) continue;
    const existing = nodeUsageMap.get(s.node_name) || [];
    existing.push(s.usage_percent);
    nodeUsageMap.set(s.node_name, existing);
  }
  const nodeAvgs = Array.from(nodeUsageMap.entries()).map(([name, pcts]) => ({
    name,
    avg: pcts.reduce((a, b) => a + b, 0) / pcts.length,
  }));
  if (nodeAvgs.length >= 2) {
    const sorted = [...nodeAvgs].sort((a, b) => b.avg - a.avg);
    const diff = sorted[0].avg - sorted[sorted.length - 1].avg;
    if (diff > 40) {
      recs.push({
        id: "unbalanced",
        severity: "warning",
        title: "Ungleichmaessige Speicherverteilung",
        description: `${sorted[0].name} ist bei ${sorted[0].avg.toFixed(0)}% Auslastung, waehrend ${sorted[sorted.length - 1].name} nur bei ${sorted[sorted.length - 1].avg.toFixed(0)}% liegt (Differenz: ${diff.toFixed(0)}%).`,
        action: "VMs oder Daten zwischen Nodes migrieren fuer bessere Verteilung",
      });
    }
  }

  // Shared vs local ratio
  const shared = items.filter((s) => s.shared);
  const local = items.filter((s) => !s.shared);
  if (local.length > 0 && shared.length === 0 && items.length > 2) {
    recs.push({
      id: "no-shared",
      severity: "info",
      title: "Kein gemeinsamer Speicher konfiguriert",
      description: `Alle ${items.length} Storages sind lokal. Shared Storage erleichtert Live-Migration und HA.`,
      action: "NFS, Ceph oder anderes Shared Storage einrichten",
    });
  }

  // If all good
  if (recs.length === 0) {
    recs.push({
      id: "all-good",
      severity: "info",
      title: "Speicher-Zustand optimal",
      description: "Alle Storages befinden sich in einem gesunden Zustand.",
      action: "Keine Massnahmen erforderlich",
    });
  }

  return recs.sort((a, b) => {
    const order = { critical: 0, warning: 1, info: 2 };
    return order[a.severity] - order[b.severity];
  });
}

// --- Treemap Component ---

function StorageTreemap({
  items,
  onItemClick,
}: {
  items: ClusterStorageItem[];
  onItemClick: (item: ClusterStorageItem) => void;
}) {
  const [hoveredItem, setHoveredItem] = useState<string | null>(null);

  // Group by node
  const grouped = useMemo(() => {
    const map = new Map<string, ClusterStorageItem[]>();
    for (const item of items) {
      if (item.total <= 0) continue;
      const existing = map.get(item.node_name) || [];
      existing.push(item);
      map.set(item.node_name, existing);
    }
    return Array.from(map.entries()).sort(
      (a, b) => b[1].reduce((s, i) => s + i.total, 0) - a[1].reduce((s, i) => s + i.total, 0)
    );
  }, [items]);

  const totalCapacity = useMemo(() => items.reduce((s, i) => s + i.total, 0), [items]);

  if (totalCapacity === 0) return null;

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center gap-2">
          <Layers className="h-4 w-4 text-primary" />
          <CardTitle className="text-base">Speicher-Treemap</CardTitle>
          <div className="ml-auto flex items-center gap-3 text-xs text-muted-foreground">
            <span className="flex items-center gap-1"><span className="h-2.5 w-2.5 rounded-sm bg-green-500" /> &lt;60%</span>
            <span className="flex items-center gap-1"><span className="h-2.5 w-2.5 rounded-sm bg-amber-500" /> 60-85%</span>
            <span className="flex items-center gap-1"><span className="h-2.5 w-2.5 rounded-sm bg-red-500" /> &gt;85%</span>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-2">
          {grouped.map(([nodeName, nodeItems]) => {
            const nodeTotal = nodeItems.reduce((s, i) => s + i.total, 0);
            return (
              <div key={nodeName}>
                <div className="flex items-center gap-1.5 mb-1">
                  <Server className="h-3 w-3 text-muted-foreground" />
                  <span className="text-xs font-medium text-muted-foreground">{nodeName}</span>
                  <span className="text-[10px] text-muted-foreground">({formatBytes(nodeTotal)})</span>
                </div>
                <div className="flex gap-1 rounded-lg overflow-hidden" style={{ height: "64px" }}>
                  {nodeItems
                    .sort((a, b) => b.total - a.total)
                    .map((item) => {
                      const widthPct = Math.max((item.total / totalCapacity) * 100, 3);
                      const itemKey = `${item.node_name}-${item.storage}`;
                      const isHovered = hoveredItem === itemKey;
                      return (
                        <Tooltip key={itemKey}>
                          <TooltipTrigger asChild>
                            <button
                              className={`relative rounded-md border transition-all cursor-pointer ${getUsageBgLight(item.usage_percent)} ${
                                isHovered ? "ring-2 ring-primary/50 scale-[1.02] z-10" : ""
                              }`}
                              style={{
                                width: `${widthPct}%`,
                                minWidth: "32px",
                              }}
                              onClick={() => onItemClick(item)}
                              onMouseEnter={() => setHoveredItem(itemKey)}
                              onMouseLeave={() => setHoveredItem(null)}
                            >
                              <div className="absolute inset-0 flex flex-col items-center justify-center p-1 overflow-hidden">
                                <span className="text-[10px] font-medium truncate max-w-full">
                                  {item.storage}
                                </span>
                                <span
                                  className="text-[10px] font-bold"
                                  style={{ color: getUsageColor(item.usage_percent) }}
                                >
                                  {item.usage_percent.toFixed(0)}%
                                </span>
                                <span className="text-[9px] text-muted-foreground truncate max-w-full">
                                  {formatBytes(item.total)}
                                </span>
                              </div>
                            </button>
                          </TooltipTrigger>
                          <TooltipContent side="top" className="text-xs">
                            <div className="space-y-1">
                              <p className="font-medium">{item.storage} ({getTypeLabel(item.type)})</p>
                              <p>Node: {item.node_name}</p>
                              <p>Gesamt: {formatBytes(item.total)}</p>
                              <p>Belegt: {formatBytes(item.used)} ({item.usage_percent.toFixed(1)}%)</p>
                              <p>Frei: {formatBytes(item.available)}</p>
                              <p>Inhalt: {item.content || "k.A."}</p>
                            </div>
                          </TooltipContent>
                        </Tooltip>
                      );
                    })}
                </div>
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
}

// --- Recommendations Component ---

function RecommendationCards({ recommendations }: { recommendations: Recommendation[] }) {
  const [expanded, setExpanded] = useState(true);

  const severityConfig = {
    critical: { icon: AlertTriangle, color: "text-red-500", bg: "border-red-500/30 bg-red-500/5", badge: "destructive" as const },
    warning: { icon: AlertTriangle, color: "text-amber-500", bg: "border-amber-500/30 bg-amber-500/5", badge: "outline" as const },
    info: { icon: Info, color: "text-blue-500", bg: "border-blue-500/30 bg-blue-500/5", badge: "outline" as const },
  };

  return (
    <Card>
      <CardHeader className="pb-3">
        <Button
          variant="ghost"
          className="flex w-full items-center justify-between p-0 h-auto hover:bg-transparent"
          onClick={() => setExpanded(!expanded)}
        >
          <div className="flex items-center gap-2">
            <Sparkles className="h-4 w-4 text-primary" />
            <CardTitle className="text-base">Empfehlungen</CardTitle>
            <Badge variant="outline" className="text-xs">{recommendations.length}</Badge>
          </div>
          {expanded ? (
            <ChevronDown className="h-4 w-4 text-muted-foreground" />
          ) : (
            <ChevronRight className="h-4 w-4 text-muted-foreground" />
          )}
        </Button>
      </CardHeader>
      {expanded && (
        <CardContent className="space-y-2">
          {recommendations.map((rec) => {
            const cfg = severityConfig[rec.severity];
            const Icon = cfg.icon;
            return (
              <div key={rec.id} className={`flex items-start gap-3 rounded-lg border p-3 ${cfg.bg}`}>
                <Icon className={`h-4 w-4 mt-0.5 shrink-0 ${cfg.color}`} />
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <p className="text-sm font-medium">{rec.title}</p>
                    <Badge variant={cfg.badge} className="text-[10px] shrink-0">
                      {rec.severity === "critical" ? "Kritisch" : rec.severity === "warning" ? "Warnung" : "Info"}
                    </Badge>
                  </div>
                  <p className="text-xs text-muted-foreground mt-0.5">{rec.description}</p>
                  <p className="text-xs font-medium mt-1">{rec.action}</p>
                </div>
              </div>
            );
          })}
        </CardContent>
      )}
    </Card>
  );
}

// --- Main Page ---

export default function ClusterStoragePage() {
  const router = useRouter();
  const [items, setItems] = useState<ClusterStorageItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [sortField, setSortField] = useState<SortField>("usage_percent");
  const [sortDir, setSortDir] = useState<SortDir>("desc");
  const [typeFilter, setTypeFilter] = useState<string>("");
  const [showTable, setShowTable] = useState(true);
  const [showSnapshots, setShowSnapshots] = useState(false);

  useEffect(() => {
    setLoading(true);
    storageApi
      .getClusterStorage()
      .then((res) => {
        setItems(toArray<ClusterStorageItem>(res.data));
      })
      .catch((err) => {
        setError(err?.response?.data?.error || err?.message || "Cluster-Speicher konnte nicht geladen werden");
      })
      .finally(() => setLoading(false));
  }, []);

  // Aggregate KPIs
  const kpis = useMemo(() => {
    const totalStorage = items.reduce((s, i) => s + i.total, 0);
    const totalUsed = items.reduce((s, i) => s + i.used, 0);
    const totalAvailable = totalStorage - totalUsed;
    const avgUtilization = items.length > 0
      ? items.reduce((s, i) => s + i.usage_percent, 0) / items.length
      : 0;
    const poolCount = items.length;
    const nodeCount = new Set(items.map((i) => i.node_name)).size;
    return { totalStorage, totalUsed, totalAvailable, avgUtilization, poolCount, nodeCount };
  }, [items]);

  // Available types for filter
  const availableTypes = useMemo(() => {
    const types = new Set(items.map((i) => i.type));
    return ALL_TYPES.filter((t) => types.has(t));
  }, [items]);

  // Filtered and sorted items
  const filteredItems = useMemo(() => {
    let result = [...items];
    if (typeFilter) {
      result = result.filter((i) => i.type === typeFilter);
    }
    result.sort((a, b) => {
      let aVal: string | number;
      let bVal: string | number;
      switch (sortField) {
        case "storage": aVal = a.storage.toLowerCase(); bVal = b.storage.toLowerCase(); break;
        case "type": aVal = a.type; bVal = b.type; break;
        case "node_name": aVal = a.node_name.toLowerCase(); bVal = b.node_name.toLowerCase(); break;
        case "total": aVal = a.total; bVal = b.total; break;
        case "used": aVal = a.used; bVal = b.used; break;
        case "available": aVal = a.available; bVal = b.available; break;
        case "usage_percent": aVal = a.usage_percent; bVal = b.usage_percent; break;
        case "content": aVal = a.content; bVal = b.content; break;
        default: aVal = 0; bVal = 0;
      }
      if (typeof aVal === "string") {
        return sortDir === "asc" ? aVal.localeCompare(bVal as string) : (bVal as string).localeCompare(aVal);
      }
      return sortDir === "asc" ? (aVal as number) - (bVal as number) : (bVal as number) - (aVal as number);
    });
    return result;
  }, [items, typeFilter, sortField, sortDir]);

  const recommendations = useMemo(() => computeRecommendations(items), [items]);

  const handleSort = useCallback((field: SortField) => {
    setSortField((prev) => {
      if (prev === field) {
        setSortDir((d) => (d === "asc" ? "desc" : "asc"));
        return field;
      }
      setSortDir("desc");
      return field;
    });
  }, []);

  const handleTreemapClick = useCallback(
    (item: ClusterStorageItem) => {
      router.push(`/nodes/${item.node_id}/storage`);
    },
    [router]
  );

  if (loading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-24" />
          ))}
        </div>
        <Skeleton className="h-64" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-bold">Speicheranalyse</h1>
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center gap-3 text-destructive">
              <Database className="h-5 w-5" />
              <div>
                <p className="font-medium">Fehler beim Laden der Speicherdaten</p>
                <p className="text-sm text-muted-foreground">{error}</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Speicheranalyse</h1>
          <p className="text-sm text-muted-foreground">
            Cluster-weite Uebersicht aller Storage-Pools
          </p>
        </div>
      </div>

      {/* KPI Cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
        <Card hover>
          <CardContent className="p-5">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-zinc-100 dark:bg-zinc-800">
                <Database className="h-5 w-5 text-zinc-600 dark:text-zinc-400" />
              </div>
              <div>
                <p className="text-xs text-muted-foreground">Gesamt-Speicher</p>
                <p className="text-xl font-bold">{formatBytes(kpis.totalStorage)}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card hover>
          <CardContent className="p-5">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-zinc-100 dark:bg-zinc-800">
                <HardDrive className="h-5 w-5 text-zinc-600 dark:text-zinc-400" />
              </div>
              <div>
                <p className="text-xs text-muted-foreground">Belegt</p>
                <p className="text-xl font-bold">{formatBytes(kpis.totalUsed)}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card hover>
          <CardContent className="p-5">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-zinc-100 dark:bg-zinc-800">
                <HardDrive className="h-5 w-5 text-zinc-600 dark:text-zinc-400" />
              </div>
              <div>
                <p className="text-xs text-muted-foreground">Verfuegbar</p>
                <p className="text-xl font-bold">{formatBytes(kpis.totalAvailable)}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card hover>
          <CardContent className="p-5">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-zinc-100 dark:bg-zinc-800">
                <TrendingUp className="h-5 w-5 text-zinc-600 dark:text-zinc-400" />
              </div>
              <div>
                <p className="text-xs text-muted-foreground">Auslastung (Avg)</p>
                <p className="text-xl font-bold" style={{ color: getUsageColor(kpis.avgUtilization) }}>
                  {kpis.avgUtilization.toFixed(1)}%
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card hover>
          <CardContent className="p-5">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-zinc-100 dark:bg-zinc-800">
                <Server className="h-5 w-5 text-zinc-600 dark:text-zinc-400" />
              </div>
              <div>
                <p className="text-xs text-muted-foreground">Pools / Nodes</p>
                <p className="text-xl font-bold">
                  {kpis.poolCount} / {kpis.nodeCount}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Treemap */}
      {items.length > 0 && (
        <StorageTreemap items={items} onItemClick={handleTreemapClick} />
      )}

      {/* Recommendations */}
      <RecommendationCards recommendations={recommendations} />

      {/* Cluster Storage Table */}
      <Card>
        <CardHeader className="py-3 px-4">
          <div className="flex items-center justify-between">
            <Button
              variant="ghost"
              className="flex items-center gap-2 p-0 h-auto hover:bg-transparent"
              onClick={() => setShowTable(!showTable)}
            >
              <CardTitle className="text-base">
                Cluster-Speicher ({filteredItems.length})
              </CardTitle>
              {showTable ? (
                <ChevronDown className="h-4 w-4 text-muted-foreground" />
              ) : (
                <ChevronRight className="h-4 w-4 text-muted-foreground" />
              )}
            </Button>
            <div className="flex items-center gap-2">
              <Filter className="h-3.5 w-3.5 text-muted-foreground" />
              <select
                className="text-xs border rounded-md px-2 py-1 bg-background"
                value={typeFilter}
                onChange={(e) => setTypeFilter(e.target.value)}
              >
                <option value="">Alle Typen</option>
                {availableTypes.map((t) => (
                  <option key={t} value={t}>
                    {getTypeLabel(t)}
                  </option>
                ))}
              </select>
            </div>
          </div>
        </CardHeader>
        {showTable && (
          <CardContent className="p-0">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/50">
                    {[
                      { field: "storage" as SortField, label: "Name", align: "left" },
                      { field: "type" as SortField, label: "Typ", align: "left" },
                      { field: "node_name" as SortField, label: "Node", align: "left" },
                      { field: "total" as SortField, label: "Gesamt", align: "right" },
                      { field: "used" as SortField, label: "Belegt", align: "right" },
                      { field: "available" as SortField, label: "Frei", align: "right" },
                      { field: "usage_percent" as SortField, label: "Auslastung", align: "right" },
                      { field: "content" as SortField, label: "Inhalt", align: "left" },
                    ].map((col) => (
                      <th
                        key={col.field}
                        className={`p-3 font-medium cursor-pointer hover:bg-muted/80 select-none ${
                          col.align === "right" ? "text-right" : "text-left"
                        }`}
                        onClick={() => handleSort(col.field)}
                      >
                        <span className="inline-flex items-center gap-1">
                          {col.label}
                          {sortField === col.field && (
                            <ArrowUpDown className="h-3 w-3" />
                          )}
                        </span>
                      </th>
                    ))}
                    <th className="p-3 text-center font-medium">Status</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredItems.map((item) => {
                    const isCritical = item.usage_percent >= 90;
                    const isWarning = item.usage_percent >= 80 && item.usage_percent < 90;
                    return (
                      <tr
                        key={`${item.node_id}-${item.storage}`}
                        className={`border-b last:border-0 cursor-pointer hover:bg-muted/50 transition-colors ${
                          isCritical ? "bg-red-500/5" : isWarning ? "bg-amber-500/5" : ""
                        }`}
                        onClick={() => router.push(`/nodes/${item.node_id}/storage`)}
                      >
                        <td className="p-3">
                          <div className="flex items-center gap-2">
                            {isCritical && <AlertTriangle className="h-3.5 w-3.5 text-red-500 shrink-0" />}
                            {isWarning && <AlertTriangle className="h-3.5 w-3.5 text-amber-500 shrink-0" />}
                            <span className="font-medium">{item.storage}</span>
                          </div>
                        </td>
                        <td className="p-3">
                          <Badge variant="outline" className="text-xs">
                            {getTypeLabel(item.type)}
                          </Badge>
                        </td>
                        <td className="p-3 text-muted-foreground">{item.node_name}</td>
                        <td className="p-3 text-right">{formatBytes(item.total)}</td>
                        <td className="p-3 text-right">{formatBytes(item.used)}</td>
                        <td className="p-3 text-right">{formatBytes(item.available)}</td>
                        <td className="p-3 text-right">
                          <div className="flex items-center justify-end gap-2">
                            <div className="h-1.5 w-16 rounded-full bg-secondary">
                              <div
                                className={`h-1.5 rounded-full ${getUsageBg(item.usage_percent)}`}
                                style={{ width: `${Math.min(item.usage_percent, 100)}%` }}
                              />
                            </div>
                            <span className="w-12 text-right font-mono text-xs">
                              {item.usage_percent.toFixed(1)}%
                            </span>
                          </div>
                        </td>
                        <td className="p-3 text-xs text-muted-foreground">
                          {item.content || "-"}
                        </td>
                        <td className="p-3 text-center">
                          <Badge variant={item.active ? "success" : "secondary"} className="text-xs">
                            {item.active ? "Aktiv" : "Inaktiv"}
                          </Badge>
                        </td>
                      </tr>
                    );
                  })}
                  {filteredItems.length === 0 && (
                    <tr>
                      <td colSpan={9} className="p-8 text-center text-muted-foreground">
                        Keine Storages gefunden
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </CardContent>
        )}
      </Card>

      {/* Snapshot Analysis */}
      <div>
        <Button
          variant="outline"
          className="mb-3 gap-2"
          onClick={() => setShowSnapshots(!showSnapshots)}
        >
          {showSnapshots ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
          Snapshot-Analyse
        </Button>
        {showSnapshots && <SnapshotAnalysis />}
      </div>
    </div>
  );
}
