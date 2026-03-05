"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useNodeStore } from "@/stores/node-store";
import { MetricsCharts } from "@/components/monitoring/metrics-charts";
import { NetworkTraffic } from "@/components/monitoring/network-traffic";
import { VMNetworkTraffic } from "@/components/monitoring/vm-network-traffic";
import { ErrorBoundary } from "@/components/error-boundary";
import { MetricsSummaryCards } from "@/components/monitoring/metrics-summary";
import { Skeleton } from "@/components/ui/skeleton";
import { metricsApi, toArray } from "@/lib/api";
import type { MetricsRecord, MetricsSummary } from "@/types/api";

const periods = [
  { label: "1h", value: "1h" },
  { label: "6h", value: "6h" },
  { label: "24h", value: "24h" },
  { label: "7d", value: "7d" },
];

export default function NodeMonitoringPage() {
  const params = useParams<{ id: string }>();
  const nodeId = params.id;
  const { nodes, fetchNodes } = useNodeStore();
  const [metrics, setMetrics] = useState<MetricsRecord[]>([]);
  const [summary, setSummary] = useState<MetricsSummary | null>(null);
  const [period, setPeriod] = useState("24h");

  useEffect(() => {
    if (nodes.length === 0) fetchNodes();
  }, [nodes.length, fetchNodes]);

  useEffect(() => {
    if (!nodeId) return;
    const since = new Date();
    const hours = period === "7d" ? 168 : period === "24h" ? 24 : period === "6h" ? 6 : 1;
    since.setHours(since.getHours() - hours);

    metricsApi
      .getHistory(nodeId, since.toISOString(), new Date().toISOString())
      .then((res) => setMetrics(toArray<MetricsRecord>(res.data)))
      .catch(() => {});

    metricsApi
      .getSummary(nodeId, period)
      .then((res) => setSummary(res.data?.data ?? res.data ?? null))
      .catch(() => {});
  }, [nodeId, period]);

  const node = nodes.find((n) => n.id === nodeId);
  if (!node) return <Skeleton className="h-64 w-full" />;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href={`/nodes/${nodeId}`}>
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <h1 className="text-2xl font-bold">Monitoring - {node.name}</h1>
        </div>
      </div>

      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">Uebersicht</TabsTrigger>
          <TabsTrigger value="network">Netzwerk</TabsTrigger>
          <TabsTrigger value="vm-traffic">VM Traffic</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-6">
          <div className="flex justify-end gap-1">
            {periods.map((p) => (
              <Button
                key={p.value}
                variant={period === p.value ? "default" : "outline"}
                size="sm"
                onClick={() => setPeriod(p.value)}
              >
                {p.label}
              </Button>
            ))}
          </div>
          {summary && <MetricsSummaryCards summary={summary} />}
          <ErrorBoundary>
            <MetricsCharts metrics={metrics} />
          </ErrorBoundary>
        </TabsContent>

        <TabsContent value="network">
          <ErrorBoundary>
            <NetworkTraffic nodeId={nodeId} />
          </ErrorBoundary>
        </TabsContent>

        <TabsContent value="vm-traffic">
          <ErrorBoundary>
            <VMNetworkTraffic nodeId={nodeId} />
          </ErrorBoundary>
        </TabsContent>
      </Tabs>
    </div>
  );
}
