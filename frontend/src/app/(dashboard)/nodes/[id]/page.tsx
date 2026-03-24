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
    const controller = new AbortController();
    const load = () => {
      fetchNodeStatus(nodeId, { signal: controller.signal }).catch((e: unknown) => {
        if (e instanceof Error && e.name === 'CanceledError') return;
      });
      fetchNodeVMs(nodeId, { signal: controller.signal }).catch((e: unknown) => {
        if (e instanceof Error && e.name === 'CanceledError') return;
      });
    };
    const token = useAuthStore.getState().accessToken;
    if (token) {
      load();
      return () => controller.abort();
    }
    const unsub = useAuthStore.subscribe((state) => {
      if (state.accessToken) {
        load();
        unsub();
      }
    });
    return () => { controller.abort(); unsub(); };
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
