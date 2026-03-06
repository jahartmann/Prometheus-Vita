"use client";

import { useEffect, useState, useMemo } from "react";
import {
  RefreshCw,
  TrendingDown,
  Shield,
  Settings2,
  Coins,
  Database,
  Globe,
  Mail,
  Wifi,
  HardDrive,
  Activity,
  Boxes,
  Server,
  ArrowDown,
  ArrowUp,
  Minus,
  Info,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import {
  TooltipProvider,
} from "@/components/ui/tooltip";
import { rightsizingApi, toArray } from "@/lib/api";
import { useNodeStore } from "@/stores/node-store";
import { KpiCard } from "@/components/ui/kpi-card";
import type { ResourceRecommendation } from "@/types/api";

const VM_CONTEXT_CONFIG: Record<
  string,
  { label: string; icon: typeof Database; color: string }
> = {
  Datenbank: { label: "Datenbanken", icon: Database, color: "text-blue-500" },
  Webserver: { label: "Webserver", icon: Globe, color: "text-green-500" },
  Mailserver: { label: "Mailserver", icon: Mail, color: "text-orange-500" },
  DNS: { label: "DNS", icon: Wifi, color: "text-purple-500" },
  Backup: { label: "Backup", icon: HardDrive, color: "text-yellow-500" },
  Monitoring: { label: "Monitoring", icon: Activity, color: "text-cyan-500" },
  "Container-Host": { label: "Container-Hosts", icon: Boxes, color: "text-pink-500" },
  Allgemein: { label: "Allgemein", icon: Server, color: "text-muted-foreground" },
};

const STATUS_CONFIG: Record<
  string,
  { label: string; variant: "default" | "secondary" | "outline" | "destructive"; color: string }
> = {
  downsize: { label: "Handlung empfohlen", variant: "default", color: "text-orange-500" },
  upsize: { label: "Handlung empfohlen", variant: "destructive", color: "text-red-500" },
  optimal: { label: "Optimal", variant: "outline", color: "text-green-500" },
};

function formatValue(value: number, type: string): string {
  if (type === "memory" || type === "disk") {
    if (value >= 1073741824) return `${(value / 1073741824).toFixed(1)} GB`;
    if (value >= 1048576) return `${(value / 1048576).toFixed(0)} MB`;
    return `${value} B`;
  }
  if (type === "cpu") return `${value} vCPU`;
  return String(value);
}

function estimateSavings(recs: ResourceRecommendation[]): number {
  return recs.filter((r) => r.recommendation_type === "downsize").length;
}

type FilterType = "all" | "action" | "optimal";

export default function RecommendationsPage() {
  const [recommendations, setRecommendations] = useState<ResourceRecommendation[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [filter, setFilter] = useState<FilterType>("all");
  const { nodes, fetchNodes } = useNodeStore();

  const fetchData = async () => {
    setIsLoading(true);
    try {
      const resp = await rightsizingApi.listAll();
      setRecommendations(toArray<ResourceRecommendation>(resp.data));
    } catch {
      // Fehler
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
    fetchNodes();
  }, [fetchNodes]);

  const getNodeName = (nodeId: string) => {
    const node = nodes.find((n) => n.id === nodeId);
    return node?.name || nodeId.slice(0, 8);
  };

  // Group recommendations by VM (vmid + node_id)
  const vmGroups = useMemo(() => {
    const filtered =
      filter === "all"
        ? recommendations
        : filter === "action"
          ? recommendations.filter((r) => r.recommendation_type !== "optimal")
          : recommendations.filter((r) => r.recommendation_type === "optimal");

    const groups = new Map<
      string,
      { vmid: number; vm_name: string; vm_type: string; node_id: string; vm_context: string; recs: ResourceRecommendation[] }
    >();

    for (const rec of filtered) {
      const key = `${rec.node_id}-${rec.vmid}`;
      if (!groups.has(key)) {
        groups.set(key, {
          vmid: rec.vmid,
          vm_name: rec.vm_name,
          vm_type: rec.vm_type,
          node_id: rec.node_id,
          vm_context: rec.vm_context || "Allgemein",
          recs: [],
        });
      }
      groups.get(key)!.recs.push(rec);
    }

    return Array.from(groups.values());
  }, [recommendations, filter]);

  // Group by VM context type
  const contextGroups = useMemo(() => {
    const groups = new Map<string, typeof vmGroups>();
    for (const vm of vmGroups) {
      const ctx = vm.vm_context || "Allgemein";
      if (!groups.has(ctx)) groups.set(ctx, []);
      groups.get(ctx)!.push(vm);
    }
    return groups;
  }, [vmGroups]);

  // Available context tabs
  const contextTabs = useMemo(() => {
    const tabs = Array.from(contextGroups.keys()).sort((a, b) => {
      if (a === "Allgemein") return 1;
      if (b === "Allgemein") return -1;
      return a.localeCompare(b);
    });
    return tabs;
  }, [contextGroups]);

  const downsizeCount = recommendations.filter((r) => r.recommendation_type === "downsize").length;
  const upsizeCount = recommendations.filter((r) => r.recommendation_type === "upsize").length;
  const optimalCount = recommendations.filter((r) => r.recommendation_type === "optimal").length;
  const actionCount = downsizeCount + upsizeCount;
  const savingsCount = estimateSavings(recommendations);

  return (
    <TooltipProvider>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-xl font-bold">Empfehlungen</h2>
            <p className="text-sm text-muted-foreground">
              Kontextbewusste Optimierungsvorschlaege fuer Ihre VMs.
            </p>
          </div>
          <Button variant="outline" onClick={fetchData} disabled={isLoading}>
            <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? "animate-spin" : ""}`} />
            Analyse aktualisieren
          </Button>
        </div>

        {/* KPI Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <KpiCard
            title="Analysierte VMs"
            value={vmGroups.length}
            subtitle="mit Empfehlungen"
            icon={Settings2}
            color="blue"
          />
          <KpiCard
            title="Handlung empfohlen"
            value={actionCount}
            subtitle={`${downsizeCount} verkleinern, ${upsizeCount} vergroessern`}
            icon={TrendingDown}
            color="orange"
          />
          <KpiCard
            title="Einsparpotenzial"
            value={savingsCount}
            subtitle="VMs ueberprovisioniert"
            icon={Coins}
            color="green"
          />
          <KpiCard
            title="Optimal"
            value={optimalCount}
            subtitle="richtig konfiguriert"
            icon={Shield}
            color="green"
          />
        </div>

        {/* Filter */}
        <div className="flex gap-2">
          <Button
            size="sm"
            variant={filter === "all" ? "default" : "outline"}
            onClick={() => setFilter("all")}
          >
            Alle ({recommendations.length})
          </Button>
          <Button
            size="sm"
            variant={filter === "action" ? "default" : "outline"}
            onClick={() => setFilter("action")}
          >
            Handlung noetig ({actionCount})
          </Button>
          <Button
            size="sm"
            variant={filter === "optimal" ? "default" : "outline"}
            onClick={() => setFilter("optimal")}
          >
            Optimal ({optimalCount})
          </Button>
        </div>

        {/* Context-grouped tabs */}
        {contextTabs.length > 0 ? (
          <Tabs defaultValue={contextTabs[0]}>
            <TabsList>
              {contextTabs.map((ctx) => {
                const cfg = VM_CONTEXT_CONFIG[ctx] || VM_CONTEXT_CONFIG.Allgemein;
                const Icon = cfg.icon;
                return (
                  <TabsTrigger key={ctx} value={ctx}>
                    <Icon className={`h-4 w-4 mr-1.5 ${cfg.color}`} />
                    {cfg.label} ({contextGroups.get(ctx)?.length || 0})
                  </TabsTrigger>
                );
              })}
            </TabsList>

            {contextTabs.map((ctx) => (
              <TabsContent key={ctx} value={ctx} className="space-y-4 mt-4">
                {(contextGroups.get(ctx) || [])
                  .sort((a, b) => {
                    // Sort by action-needed first
                    const aAction = a.recs.some((r) => r.recommendation_type !== "optimal") ? 0 : 1;
                    const bAction = b.recs.some((r) => r.recommendation_type !== "optimal") ? 0 : 1;
                    return aAction - bAction;
                  })
                  .map((vm) => (
                    <VMRecommendationCard
                      key={`${vm.node_id}-${vm.vmid}`}
                      vm={vm}
                      getNodeName={getNodeName}
                    />
                  ))}
              </TabsContent>
            ))}
          </Tabs>
        ) : (
          <Card>
            <CardContent className="py-12 text-center text-muted-foreground">
              {isLoading
                ? "Analyse laeuft..."
                : "Keine Empfehlungen vorhanden. Starten Sie eine Analyse."}
            </CardContent>
          </Card>
        )}
      </div>
    </TooltipProvider>
  );
}

function VMRecommendationCard({
  vm,
  getNodeName,
}: {
  vm: {
    vmid: number;
    vm_name: string;
    vm_type: string;
    node_id: string;
    vm_context: string;
    recs: ResourceRecommendation[];
  };
  getNodeName: (id: string) => string;
}) {
  const hasAction = vm.recs.some((r) => r.recommendation_type !== "optimal");
  const worstType = vm.recs.some((r) => r.recommendation_type === "upsize")
    ? "upsize"
    : vm.recs.some((r) => r.recommendation_type === "downsize")
      ? "downsize"
      : "optimal";

  const statusCfg = STATUS_CONFIG[worstType];
  const ctxCfg = VM_CONTEXT_CONFIG[vm.vm_context] || VM_CONTEXT_CONFIG.Allgemein;
  const CtxIcon = ctxCfg.icon;

  // Get context reason from first rec that has one
  const contextReason = vm.recs.find((r) => r.context_reason)?.context_reason;

  return (
    <Card className={hasAction ? "border-l-4 border-l-orange-500" : ""}>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-2">
              <CtxIcon className={`h-5 w-5 ${ctxCfg.color}`} />
              <CardTitle className="text-base">
                {vm.vm_name || `VM ${vm.vmid}`}
              </CardTitle>
              <Badge variant="outline" className="text-xs">
                {vm.vm_type}
              </Badge>
              <Badge variant="secondary" className="text-xs">
                {vm.vm_context}
              </Badge>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs text-muted-foreground">
              {getNodeName(vm.node_id)}
            </span>
            <Badge variant={statusCfg.variant}>
              {statusCfg.label}
            </Badge>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Context explanation */}
        {contextReason && (
          <div className="flex items-start gap-2 rounded-md bg-muted/50 p-3 text-sm">
            <Info className="h-4 w-4 mt-0.5 text-blue-500 shrink-0" />
            <span className="text-muted-foreground">{contextReason}</span>
          </div>
        )}

        {/* Resource recommendations */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {vm.recs.map((rec) => (
            <ResourceRow key={rec.id} rec={rec} />
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

function ResourceRow({ rec }: { rec: ResourceRecommendation }) {
  const isOptimal = rec.recommendation_type === "optimal";
  const isDownsize = rec.recommendation_type === "downsize";
  const Icon = isDownsize ? ArrowDown : isOptimal ? Minus : ArrowUp;
  const iconColor = isDownsize
    ? "text-green-500"
    : isOptimal
      ? "text-muted-foreground"
      : "text-red-500";

  const usagePercent = Math.min(rec.avg_usage, 100);
  const usageColor =
    rec.avg_usage > 80 ? "bg-red-500" : rec.avg_usage > 50 ? "bg-orange-500" : "bg-green-500";

  return (
    <div className="rounded-lg border p-3 space-y-2">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Badge variant="outline" className="font-mono text-xs uppercase">
            {rec.resource_type}
          </Badge>
          <Icon className={`h-4 w-4 ${iconColor}`} />
        </div>
        <span className="text-xs text-muted-foreground">
          avg {(rec.avg_usage ?? 0).toFixed(1)}% / max {(rec.max_usage ?? 0).toFixed(1)}%
        </span>
      </div>

      {/* Usage bar */}
      <div className="space-y-1">
        <div className="h-2 w-full rounded-full bg-muted overflow-hidden">
          <div
            className={`h-full rounded-full transition-all ${usageColor}`}
            style={{ width: `${usagePercent}%` }}
          />
        </div>
      </div>

      {/* Current vs recommended */}
      <div className="flex items-center justify-between text-sm">
        <div>
          <span className="text-muted-foreground">Aktuell: </span>
          <span className="font-mono font-medium">
            {formatValue(rec.current_value, rec.resource_type)}
          </span>
        </div>
        {!isOptimal && (
          <div>
            <span className="text-muted-foreground">Empfohlen: </span>
            <span className="font-mono font-medium">
              {formatValue(rec.recommended_value, rec.resource_type)}
            </span>
          </div>
        )}
      </div>

      {/* Reason */}
      {rec.reason && (
        <p className="text-xs text-muted-foreground">{rec.reason}</p>
      )}
    </div>
  );
}
