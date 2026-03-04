"use client";

import { useEffect, useState } from "react";
import { RefreshCw, TrendingDown, TrendingUp, Shield, Settings2, Coins } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { rightsizingApi, toArray } from "@/lib/api";
import { useNodeStore } from "@/stores/node-store";
import { RecommendationList } from "@/components/recommendations/recommendation-list";
import { KpiCard } from "@/components/ui/kpi-card";
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
  const optimalCount = recommendations.filter((r) => r.recommendation_type === "optimal").length;

  // Categorize recommendations
  const performanceRecs = recommendations.filter(
    (r) => r.resource_type === "cpu" || r.resource_type === "memory"
  );
  const configRecs = recommendations.filter(
    (r) => r.resource_type === "disk" || r.resource_type === "balloon"
  );

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-bold">Empfehlungen</h2>
          <p className="text-sm text-muted-foreground">
            Optimierungs- und Sicherheitsvorschlaege fuer Ihre Infrastruktur.
          </p>
        </div>
        <Button variant="outline" onClick={fetchData} disabled={isLoading}>
          <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? "animate-spin" : ""}`} />
          Aktualisieren
        </Button>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <KpiCard
          title="Gesamt"
          value={recommendations.length}
          subtitle="Empfehlungen"
          icon={Settings2}
          color="blue"
        />
        <KpiCard
          title="Verkleinern"
          value={downsizeCount}
          subtitle="Ressourcen sparen"
          icon={TrendingDown}
          color="green"
        />
        <KpiCard
          title="Vergroessern"
          value={upsizeCount}
          subtitle="Mehr Leistung"
          icon={TrendingUp}
          color="orange"
        />
        <KpiCard
          title="Optimal"
          value={optimalCount}
          subtitle="Richtig konfiguriert"
          icon={Shield}
          color="green"
        />
      </div>

      <Tabs defaultValue="all">
        <TabsList>
          <TabsTrigger value="all">
            Alle ({recommendations.length})
          </TabsTrigger>
          <TabsTrigger value="performance">
            Performance ({performanceRecs.length})
          </TabsTrigger>
          <TabsTrigger value="config">
            Konfiguration ({configRecs.length})
          </TabsTrigger>
        </TabsList>

        <TabsContent value="all">
          <Card>
            <CardContent className="p-0">
              <RecommendationList recommendations={recommendations} getNodeName={getNodeName} />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="performance">
          <Card>
            <CardContent className="p-0">
              <RecommendationList recommendations={performanceRecs} getNodeName={getNodeName} />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="config">
          <Card>
            <CardContent className="p-0">
              <RecommendationList recommendations={configRecs} getNodeName={getNodeName} />
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
