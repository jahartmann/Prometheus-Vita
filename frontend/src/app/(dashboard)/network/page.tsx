"use client";

import { useEffect, useState } from "react";
import { Network, ShieldAlert, BookMarked } from "lucide-react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { ChevronDown } from "lucide-react";
import { useNodeStore } from "@/stores/node-store";
import { useNetworkStore } from "@/stores/network-store";
import { useNetworkScan } from "@/hooks/use-network-scan";
import { ScanStatusBar } from "@/components/network/scan-status-bar";
import { PortTable } from "@/components/network/port-table";
import { DeviceTable } from "@/components/network/device-table";
import { NetworkAnomalyList } from "@/components/network/anomaly-list";
import { ScanTimeline } from "@/components/network/scan-timeline";
import { BaselineManager } from "@/components/network/baseline-manager";

export default function ClusterNetworkPage() {
  const { nodes, fetchNodes } = useNodeStore();
  const { anomalies, fetchBaselines, activeTab, setActiveTab } = useNetworkStore();

  const [selectedNodeId, setSelectedNodeId] = useState<string>("");
  const [baselineOpen, setBaselineOpen] = useState(false);

  // Load nodes
  useEffect(() => {
    fetchNodes();
  }, [fetchNodes]);

  // Default to first node
  useEffect(() => {
    if (nodes.length > 0 && !selectedNodeId) {
      setSelectedNodeId(nodes[0].id);
    }
  }, [nodes, selectedNodeId]);

  // Load baselines when node changes
  useEffect(() => {
    if (selectedNodeId) fetchBaselines(selectedNodeId);
  }, [selectedNodeId, fetchBaselines]);

  // Poll for scan data
  useNetworkScan({ nodeId: selectedNodeId, enabled: !!selectedNodeId });

  const unacknowledgedCount = anomalies.filter((a) => !a.is_acknowledged).length;
  const selectedNode = nodes.find((n) => n.id === selectedNodeId);

  return (
    <div className="space-y-5">
      {/* Page header */}
      <div className="flex items-center gap-3">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-blue-500/10">
          <Network className="h-5 w-5 text-blue-500" />
        </div>
        <div className="flex-1 min-w-0">
          <h1 className="text-2xl font-bold tracking-tight">Netzwerk-Analyse</h1>
          <p className="text-sm text-muted-foreground">Cluster-weite Port- und Geräteerkennung</p>
        </div>
      </div>

      {/* Node selector */}
      {nodes.length > 0 && (
        <div className="flex flex-wrap items-center gap-2 rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-2">
          <span className="text-xs text-zinc-500 mr-1">Node:</span>
          {nodes.map((node) => {
            const isActive = selectedNodeId === node.id;
            return (
              <button
                key={node.id}
                onClick={() => setSelectedNodeId(node.id)}
                className={`rounded px-2.5 py-1 text-xs font-medium transition-colors cursor-pointer ${
                  isActive
                    ? "bg-blue-600 text-blue-100"
                    : "bg-zinc-800/50 text-zinc-500 hover:bg-zinc-700/50"
                }`}
              >
                {node.name}
              </button>
            );
          })}
          {selectedNode && (
            <span className="ml-auto text-xs text-zinc-600 font-mono">{selectedNode.id}</span>
          )}
        </div>
      )}

      {!selectedNodeId ? (
        <div className="flex items-center justify-center py-16 text-zinc-600">
          <p className="text-sm">Kein Node ausgewählt.</p>
        </div>
      ) : (
        <>
          {/* Scan status */}
          <ScanStatusBar nodeId={selectedNodeId} />

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

          {/* Tabs */}
          <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as typeof activeTab)}>
            <TabsList className="bg-zinc-900 border border-zinc-800">
              <TabsTrigger value="ports" className="text-sm">Ports</TabsTrigger>
              <TabsTrigger value="devices" className="text-sm">Netzwerk-Geräte</TabsTrigger>
              <TabsTrigger value="anomalies" className="text-sm gap-1.5">
                Anomalien
                {unacknowledgedCount > 0 && (
                  <Badge className="bg-orange-500/20 text-orange-400 border-orange-500/30 text-[10px] px-1.5 py-0 h-4">
                    {unacknowledgedCount}
                  </Badge>
                )}
              </TabsTrigger>
              <TabsTrigger value="history" className="text-sm">Scan-Historie</TabsTrigger>
            </TabsList>

            <TabsContent value="ports" className="mt-4">
              <PortTable nodeId={selectedNodeId} />
            </TabsContent>

            <TabsContent value="devices" className="mt-4">
              <DeviceTable nodeId={selectedNodeId} />
            </TabsContent>

            <TabsContent value="anomalies" className="mt-4">
              <NetworkAnomalyList nodeId={selectedNodeId} />
            </TabsContent>

            <TabsContent value="history" className="mt-4">
              <ScanTimeline nodeId={selectedNodeId} />
            </TabsContent>
          </Tabs>

          {/* Baseline manager */}
          <Collapsible open={baselineOpen} onOpenChange={setBaselineOpen}>
            <CollapsibleTrigger className="flex items-center gap-2 w-full rounded-lg border border-zinc-800 bg-zinc-900/60 px-4 py-2.5 hover:bg-zinc-800/60 transition-colors text-sm font-medium text-zinc-300">
              <BookMarked className="h-4 w-4 text-zinc-500" />
              Baseline-Verwaltung
              <ChevronDown
                className={`h-4 w-4 ml-auto text-zinc-500 transition-transform ${baselineOpen ? "rotate-180" : ""}`}
              />
            </CollapsibleTrigger>
            <CollapsibleContent className="mt-2 rounded-lg border border-zinc-800 bg-zinc-900/40 p-4">
              <BaselineManager nodeId={selectedNodeId} />
            </CollapsibleContent>
          </Collapsible>
        </>
      )}
    </div>
  );
}
