"use client";

import { useEffect, useState } from "react";
import { RefreshCw, TrendingDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { rightsizingApi, toArray } from "@/lib/api";
import { useNodeStore } from "@/stores/node-store";
import { RecommendationList } from "@/components/recommendations/recommendation-list";
import type { ResourceRecommendation } from "@/types/api";

export default function RecommendationsPage() {
  const [recommendations, setRecommendations] = useState<ResourceRecommendation[]>([]);
  const [isLoading, setIsLoading] = useState(false);
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

  const downsizeCount = recommendations.filter((r) => r.recommendation_type === "downsize").length;
  const upsizeCount = recommendations.filter((r) => r.recommendation_type === "upsize").length;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">Ressourcen-Empfehlungen</h2>
          <p className="text-sm text-muted-foreground">
            Right-Sizing Vorschlaege fuer VMs und Container.
          </p>
        </div>
        <Button variant="outline" onClick={fetchData} disabled={isLoading}>
          <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? "animate-spin" : ""}`} />
          Aktualisieren
        </Button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <CardContent className="p-4 flex items-center gap-3">
            <TrendingDown className="h-8 w-8 text-blue-500" />
            <div>
              <p className="text-2xl font-bold">{recommendations.length}</p>
              <p className="text-sm text-muted-foreground">Empfehlungen gesamt</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-2xl font-bold text-green-600">{downsizeCount}</p>
            <p className="text-sm text-muted-foreground">Verkleinerung moeglich</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-2xl font-bold text-yellow-600">{upsizeCount}</p>
            <p className="text-sm text-muted-foreground">Vergroesserung empfohlen</p>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardContent className="p-0">
          <RecommendationList recommendations={recommendations} getNodeName={getNodeName} />
        </CardContent>
      </Card>
    </div>
  );
}
