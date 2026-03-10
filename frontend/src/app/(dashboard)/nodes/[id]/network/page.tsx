"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useNodeStore } from "@/stores/node-store";
import { NetworkInterfaces } from "@/components/nodes/network-interfaces";
import { LiveBandwidthGauge } from "@/components/monitoring/live-bandwidth-gauge";
import { NetworkTraffic } from "@/components/monitoring/network-traffic";
import { useNodeMetrics } from "@/hooks/use-node-metrics";
import { Skeleton } from "@/components/ui/skeleton";
import { networkApi, toArray } from "@/lib/api";
import type { NetworkInterface } from "@/types/api";

export default function NodeNetworkPage() {
  const params = useParams<{ id: string }>();
  const nodeId = params.id;
  const { nodes, fetchNodes } = useNodeStore();
  const [interfaces, setInterfaces] = useState<NetworkInterface[]>([]);
  const { latestMetrics } = useNodeMetrics(nodeId);

  useEffect(() => {
    if (nodes.length === 0) fetchNodes();
  }, [nodes.length, fetchNodes]);

  useEffect(() => {
    if (nodeId) {
      networkApi
        .getInterfaces(nodeId)
        .then((res) => {
          setInterfaces(toArray<NetworkInterface>(res.data));
        })
        .catch(() => {});
    }
  }, [nodeId]);

  const node = nodes.find((n) => n.id === nodeId);
  if (!node) return <Skeleton className="h-64 w-full" />;

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link href={`/nodes/${nodeId}`}>
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <h1 className="text-2xl font-bold">Netzwerk - {node.name}</h1>
      </div>

      {/* Live Bandwidth Gauge */}
      <div className="grid gap-6 md:grid-cols-2">
        <LiveBandwidthGauge
          netIn={latestMetrics?.network_in ?? 0}
          netOut={latestMetrics?.network_out ?? 0}
        />
        <div className="flex flex-col justify-center">
          <NetworkInterfaces
            nodeId={nodeId}
            interfaces={interfaces}
            onRefresh={() => {
              networkApi.getInterfaces(nodeId).then((res) => {
                setInterfaces(toArray<NetworkInterface>(res.data));
              });
            }}
          />
        </div>
      </div>

      {/* Historical Traffic */}
      <NetworkTraffic nodeId={nodeId} />
    </div>
  );
}
