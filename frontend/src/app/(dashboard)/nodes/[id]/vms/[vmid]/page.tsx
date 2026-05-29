"use client";

import { useEffect } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { ArrowLeft, ServerOff } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
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

  // Distinguish "still loading" from "does not exist": once nodes are loaded
  // and the node's VM list has been fetched (the key exists), a missing VM is
  // genuinely not found — show a clear state instead of skeletons forever.
  const loading = nodes.length === 0 || !(nodeId in nodeVMs);

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild aria-label="Zurück zur VM-Liste">
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

  if (!node || !vm) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild aria-label="Zurück zur VM-Liste">
            <Link href={`/nodes/${nodeId}/vms`}>
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
        </div>
        <EmptyState
          icon={ServerOff}
          title={!node ? "Node nicht gefunden" : `VM ${vmid} nicht gefunden`}
          description={
            !node
              ? "Dieser Node existiert nicht mehr oder ist nicht erreichbar."
              : "Diese VM/dieser Container existiert auf diesem Node nicht (mehr)."
          }
          variant="error"
          action={
            <Button asChild variant="outline" size="sm">
              <Link href={`/nodes/${nodeId}/vms`}>Zurück zur VM-Liste</Link>
            </Button>
          }
        />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild aria-label="Zurück zur VM-Liste">
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
