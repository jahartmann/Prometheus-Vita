"use client";

import { useEffect } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useNodeStore } from "@/stores/node-store";
import { VmList } from "@/components/nodes/vm-list";
import { Skeleton } from "@/components/ui/skeleton";

export default function NodeVmsPage() {
  const params = useParams<{ id: string }>();
  const nodeId = params.id;
  const { nodes, nodeVMs, fetchNodes, fetchNodeVMs } = useNodeStore();

  useEffect(() => {
    if (nodes.length === 0) {
      fetchNodes();
    }
  }, [nodes.length, fetchNodes]);

  useEffect(() => {
    if (nodeId) {
      fetchNodeVMs(nodeId);
    }
  }, [nodeId, fetchNodeVMs]);

  const node = nodes.find((n) => n.id === nodeId);
  const vms = nodeVMs[nodeId] || [];

  if (!node) {
    return <Skeleton className="h-64 w-full" />;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link href={`/nodes/${nodeId}`}>
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <h1 className="text-2xl font-bold">
            VMs & Container - {node.name}
          </h1>
          <p className="text-sm text-muted-foreground">{node.hostname}:{node.port}</p>
        </div>
      </div>

      <VmList vms={vms} nodeId={nodeId} onRefresh={() => fetchNodeVMs(nodeId)} />
    </div>
  );
}
