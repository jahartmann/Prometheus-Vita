"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { ArrowLeft, ShieldAlert, BookMarked } from "lucide-react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { ChevronDown } from "lucide-react";
import { useNetworkStore } from "@/stores/network-store";
import { useNodeStore } from "@/stores/node-store";
import { useNetworkScan } from "@/hooks/use-network-scan";
import { ScanStatusBar } from "@/components/network/scan-status-bar";
import { PortTable } from "@/components/network/port-table";
import { DeviceTable } from "@/components/network/device-table";
import { NetworkAnomalyList } from "@/components/network/anomaly-list";
import { ScanTimeline } from "@/components/network/scan-timeline";
import { BaselineManager } from "@/components/network/baseline-manager";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";

export default function NodePortsPage() {
  const params = useParams<{ id: string }>();
  const nodeId = params.id;

  const { nodes, fetchNodes } = useNodeStore();
  const { anomalies, fetchBaselines, activeTab, setActiveTab } = useNetworkStore();

  const [baselineOpen, setBaselineOpen] = useState(false);

  // Load nodes list
  useEffect(() => {
    if (nodes.length === 0) fetchNodes();
  }, [nodes.length, fetchNodes]);

  // Load baselines on mount
  useEffect(() => {
    if (nodeId) fetchBaselines(nodeId);
  }, [nodeId, fetchBaselines]);

  // Poll for scan data
  useNetworkScan({ nodeId });

  const node = nodes.find((n) => n.id === nodeId);

  const unacknowledgedCount = anomalies.filter((a) => !a.is_acknowledged).length;

  if (!node) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-14 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  return (
    <div className="space-y-5">
      {/* Page header */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link href={`/nodes/${nodeId}`}>
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <h1 className="text-2xl font-bold">Port &amp; Netzwerk-Analyse</h1>
          <p className="text-sm text-zinc-500">{node.name}</p>
        </div>
      </div>

      {/* Scan status bar */}
      <ScanStatusBar nodeId={nodeId} />

      {/* Anomaly banner */}
      {unacknowledgedCount > 0 && (
        <button
          className="w-full flex items-center gap-3 rounded-lg border border-orange-500/30 bg-orange-500/10 px-4 py-3 text-left hover:bg-orange-500/15 transition-colors cursor-pointer"
          onClick={() => setActiveTab("anomalies")}
        >
          <ShieldAlert className="h-5 w-5 text-orange-400 shrink-0" />
          <div>
            <p className="text-sm font-medium text-orange-300">
              {unacknowledgedCount} unbestätigte Netzwerk-Anomalie{unacknowledgedCount !== 1 ? "n" : ""}
            </p>
            <p className="text-xs text-orange-400/70">Klicken zum Anzeigen</p>
          </div>
          <Badge className="ml-auto bg-orange-500/20 text-orange-300 border-orange-500/30">
            {unacknowledgedCount}
          </Badge>
        </button>
      )}

      {/* Main tabs */}
      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as typeof activeTab)}>
        <TabsList className="bg-zinc-900 border border-zinc-800">
          <TabsTrigger value="ports" className="text-sm">
            Ports
          </TabsTrigger>
          <TabsTrigger value="devices" className="text-sm">
            Netzwerk-Geräte
          </TabsTrigger>
          <TabsTrigger value="anomalies" className="text-sm gap-1.5">
            Anomalien
            {unacknowledgedCount > 0 && (
              <Badge className="bg-orange-500/20 text-orange-400 border-orange-500/30 text-[10px] px-1.5 py-0 h-4">
                {unacknowledgedCount}
              </Badge>
            )}
          </TabsTrigger>
          <TabsTrigger value="history" className="text-sm">
            Scan-Historie
          </TabsTrigger>
        </TabsList>

        <TabsContent value="ports" className="mt-4">
          <PortTable nodeId={nodeId} />
        </TabsContent>

        <TabsContent value="devices" className="mt-4">
          <DeviceTable nodeId={nodeId} />
        </TabsContent>

        <TabsContent value="anomalies" className="mt-4">
          <NetworkAnomalyList nodeId={nodeId} />
        </TabsContent>

        <TabsContent value="history" className="mt-4">
          <ScanTimeline nodeId={nodeId} />
        </TabsContent>
      </Tabs>

      {/* Baseline manager (collapsible) */}
      <Collapsible open={baselineOpen} onOpenChange={setBaselineOpen}>
        <CollapsibleTrigger className="flex items-center gap-2 w-full rounded-lg border border-zinc-800 bg-zinc-900/60 px-4 py-2.5 hover:bg-zinc-800/60 transition-colors text-sm font-medium text-zinc-300">
          <BookMarked className="h-4 w-4 text-zinc-500" />
          Baseline-Verwaltung
          <ChevronDown
            className={`h-4 w-4 ml-auto text-zinc-500 transition-transform ${baselineOpen ? "rotate-180" : ""}`}
          />
        </CollapsibleTrigger>
        <CollapsibleContent className="mt-2 rounded-lg border border-zinc-800 bg-zinc-900/40 p-4">
          <BaselineManager nodeId={nodeId} />
        </CollapsibleContent>
      </Collapsible>
    </div>
  );
}
