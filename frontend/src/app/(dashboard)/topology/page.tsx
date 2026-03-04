"use client";

import { useEffect } from "react";
import { RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useTopologyStore } from "@/stores/topology-store";
import { TopologyMap } from "@/components/topology/topology-map";

export default function TopologyPage() {
  const { graph, isLoading, error, fetchTopology } = useTopologyStore();

  useEffect(() => {
    fetchTopology();
  }, [fetchTopology]);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">Cluster-Topologie</h2>
          <p className="text-sm text-muted-foreground">
            Visuelle Uebersicht der Cluster-Infrastruktur.
          </p>
        </div>
        <Button variant="outline" onClick={fetchTopology} disabled={isLoading}>
          <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? "animate-spin" : ""}`} />
          Aktualisieren
        </Button>
      </div>

      {error && (
        <div className="text-sm text-destructive">{error}</div>
      )}

      <div className="border rounded-lg bg-card" style={{ height: "calc(100vh - 200px)" }}>
        {graph ? (
          <TopologyMap graph={graph} />
        ) : isLoading ? (
          <div className="flex items-center justify-center h-full text-muted-foreground">
            Lade Topologie...
          </div>
        ) : (
          <div className="flex items-center justify-center h-full text-muted-foreground">
            Keine Topologie-Daten verfuegbar.
          </div>
        )}
      </div>
    </div>
  );
}
