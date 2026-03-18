"use client";

import { useEffect } from "react";
import { useParams } from "next/navigation";
import { useNodeStore } from "@/stores/node-store";
import { useAuthStore } from "@/stores/auth-store";
import { NodeDetail } from "@/components/nodes/node-detail";
import { Skeleton } from "@/components/ui/skeleton";

export default function NodeDetailPage() {
  const params = useParams<{ id: string }>();
  const nodeId = params.id;
  const { nodes, fetchNodes, fetchNodeStatus, fetchNodeVMs } = useNodeStore();

  useEffect(() => {
    const token = useAuthStore.getState().accessToken;
    if (token && nodes.length === 0) {
      fetchNodes();
      return;
    }
    const unsub = useAuthStore.subscribe((state) => {
      if (state.accessToken && nodes.length === 0) {
        fetchNodes();
        unsub();
      }
    });
    return () => unsub();
  }, [nodes.length, fetchNodes]);

  useEffect(() => {
    if (!nodeId) return;
    const load = () => {
      fetchNodeStatus(nodeId);
      fetchNodeVMs(nodeId);
    };
    const token = useAuthStore.getState().accessToken;
    if (token) {
      load();
      return;
    }
    const unsub = useAuthStore.subscribe((state) => {
      if (state.accessToken) {
        load();
        unsub();
      }
    });
    return () => unsub();
  }, [nodeId, fetchNodeStatus, fetchNodeVMs]);

  const node = nodes.find((n) => n.id === nodeId);

  if (!node) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  return <NodeDetail node={node} />;
}
