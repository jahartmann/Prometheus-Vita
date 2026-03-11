"use client";

import { useEffect } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useNodeStore } from "@/stores/node-store";
import { VMCockpit } from "@/components/vm-cockpit/vm-cockpit";

export default function VMDetailPage() {
  const params = useParams<{ id: string; vmid: string }>();
  const nodeId = params.id;
  const vmid = parseInt(params.vmid, 10);
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
  const vm = vms.find((v) => v.vmid === vmid);

  if (!node || !vm) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href={`/nodes/${nodeId}/vms`}>
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <Skeleton className="h-8 w-48" />
        </div>
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-[600px] w-full" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link href={`/nodes/${nodeId}/vms`}>
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <p className="text-sm text-muted-foreground">
            {node.name} &rsaquo; VMs & Container
          </p>
        </div>
      </div>

      <VMCockpit vm={vm} nodeId={nodeId} />
    </div>
  );
}
