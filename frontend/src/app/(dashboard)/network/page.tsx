"use client";

import { useEffect, useState } from "react";
import { BookMarked, ChevronDown, Gauge, Network, ShieldAlert, Wrench } from "lucide-react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { useNodeStore } from "@/stores/node-store";
import { useNetworkStore } from "@/stores/network-store";
import { useNetworkScan } from "@/hooks/use-network-scan";
import { ScanStatusBar } from "@/components/network/scan-status-bar";
import { NetworkSecurityOverview } from "@/components/network/network-security-overview";
import { PortTable } from "@/components/network/port-table";
import { DeviceTable } from "@/components/network/device-table";
import { NetworkAnomalyList } from "@/components/network/anomaly-list";
import { ScanTimeline } from "@/components/network/scan-timeline";
import { BaselineManager } from "@/components/network/baseline-manager";
import { VMServiceAnalysis } from "@/components/network/vm-service-analysis";
import { BandwidthTest } from "@/components/network/bandwidth-test";
import { FeatureStatusCard } from "@/components/ui/feature-status-card";

export default function ClusterNetworkPage() {
  const { nodes, fetchNodes } = useNodeStore();
  const {
    anomalies,
    errorsByScope,
    fetchBaselines,
    fetchToolPreflight,
    toolPreflightByNode,
    activeTab,
    setActiveTab,
  } = useNetworkStore();

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

  // Load node-specific readiness when node changes
  useEffect(() => {
    if (selectedNodeId) {
      fetchBaselines(selectedNodeId);
      fetchToolPreflight(selectedNodeId);
    }
  }, [selectedNodeId, fetchBaselines, fetchToolPreflight]);

  // Poll for scan data
  useNetworkScan({ nodeId: selectedNodeId, enabled: !!selectedNodeId });

  const selectedNodeAnomalies = anomalies.filter((a) => a.node_id === selectedNodeId);
  const unacknowledgedCount = selectedNodeAnomalies.filter((a) => !a.is_acknowledged).length;
  const selectedNode = nodes.find((n) => n.id === selectedNodeId);
  const toolPreflight = selectedNodeId ? toolPreflightByNode[selectedNodeId] : undefined;
  const scopedError = (scope: string) => errorsByScope[`${selectedNodeId}:${scope}`];
  const nmapCheck = toolPreflight?.tools.find((tool) => tool.name === "nmap");
  const fullScanAvailable = !toolPreflight || !!nmapCheck?.available;
  const fullScanUnavailableReason = fullScanAvailable ? undefined : "nmap fehlt auf der ausgewählten Node";
  const toolStatus = toolPreflight
    ? fullScanAvailable
      ? "Full-Scan bereit"
      : "nmap fehlt"
    : "Preflight lädt";
  const toolTone = toolPreflight ? (fullScanAvailable ? "ok" : "warning") : "muted";
  const toolDetails = (
    <div className="flex flex-wrap gap-2">
      {toolPreflight?.tools.length ? (
        toolPreflight.tools.map((tool) => (
          <Badge
            key={tool.name}
            variant={tool.available ? "success" : "warning"}
            title={tool.path ?? undefined}
          >
            {tool.name}
          </Badge>
        ))
      ) : (
        <span className="text-sm text-muted-foreground">Tool-Status wird geladen.</span>
      )}
    </div>
  );

  return (
    <div className="space-y-5">
      {/* Page header */}
      <div className="flex items-center gap-3">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-blue-500/10">
          <Network className="h-5 w-5 text-blue-500" />
        </div>
        <div className="flex-1 min-w-0">
          <h1 className="text-2xl font-semibold tracking-tight">Netzwerk-Analyse</h1>
          <p className="text-sm text-muted-foreground">Cluster-weite Port- und Geräteerkennung</p>
        </div>
      </div>

      {/* Node selector */}
      {nodes.length > 0 && (
        <div className="flex flex-wrap items-center gap-2 rounded-lg border bg-card px-3 py-2">
          <span className="text-xs text-muted-foreground mr-1">Node:</span>
          {nodes.map((node) => {
            const isActive = selectedNodeId === node.id;
            return (
              <button
                key={node.id}
                onClick={() => setSelectedNodeId(node.id)}
                className={`rounded px-2.5 py-1 text-xs font-medium transition-colors cursor-pointer ${
                  isActive
                    ? "bg-primary text-primary-foreground"
                    : "bg-muted text-muted-foreground hover:bg-accent hover:text-foreground"
                }`}
              >
                {node.name}
              </button>
            );
          })}
          {selectedNode && (
            <span className="ml-auto text-xs text-muted-foreground font-mono">{selectedNode.id}</span>
          )}
        </div>
      )}

      {!selectedNodeId ? (
        <div className="rounded-lg border border-dashed p-10 text-center">
          <p className="text-sm font-medium">Kein Node ausgewählt</p>
          <p className="mt-1 text-sm text-muted-foreground">
            Wähle eine Node aus, um Ports, Geräte, Anomalien und Bandbreite zu prüfen.
          </p>
        </div>
      ) : (
        <>
          {/* Scan status */}
          <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
            <ScanStatusBar
              nodeId={selectedNodeId}
              fullScanAvailable={fullScanAvailable}
              fullScanUnavailableReason={fullScanUnavailableReason}
            />
            <FeatureStatusCard
              title="Tool-Preflight"
              description="Node-Werkzeuge für Netzwerk-Erkennung und Scan-Tiefe."
              icon={Wrench}
              tone={toolTone}
              status={toolStatus}
              details={toolDetails}
              error={scopedError("tools")}
            />
          </div>
          <NetworkSecurityOverview nodeId={selectedNodeId} />

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
              <TabsTrigger value="services" className="text-sm">VM-/Service-Analyse</TabsTrigger>
              <TabsTrigger value="bandwidth" className="text-sm gap-1.5">
                <Gauge className="h-3.5 w-3.5" />
                Bandbreite
              </TabsTrigger>
              <TabsTrigger value="history" className="text-sm">Scan-Historie</TabsTrigger>
            </TabsList>

            <TabsContent value="ports" className="mt-4">
              <PortTable nodeId={selectedNodeId} />
            </TabsContent>

            <TabsContent value="devices" className="mt-4">
              <NetworkSectionError message={scopedError("devices")} />
              <DeviceTable nodeId={selectedNodeId} />
            </TabsContent>

            <TabsContent value="anomalies" className="mt-4">
              <NetworkSectionError message={scopedError("anomalies")} />
              <NetworkAnomalyList nodeId={selectedNodeId} />
            </TabsContent>

            <TabsContent value="services" className="mt-4">
              <VMServiceAnalysis nodeId={selectedNodeId} />
            </TabsContent>

            <TabsContent value="bandwidth" className="mt-4">
              <BandwidthTest sourceNodeId={selectedNodeId} />
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
            <NetworkSectionError message={scopedError("baselines")} className="mt-2" />
            <CollapsibleContent className="mt-2 rounded-lg border border-zinc-800 bg-zinc-900/40 p-4">
              <BaselineManager nodeId={selectedNodeId} />
            </CollapsibleContent>
          </Collapsible>
        </>
      )}
    </div>
  );
}

function NetworkSectionError({
  message,
  className = "mb-3",
}: {
  message?: string;
  className?: string;
}) {
  if (!message) return null;

  return (
    <p className={`rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/25 dark:text-red-300 ${className}`}>
      {message}
    </p>
  );
}
