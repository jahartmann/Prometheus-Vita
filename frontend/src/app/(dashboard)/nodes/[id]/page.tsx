"use client";

import { useEffect } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { AlertCircle, ArrowLeft } from "lucide-react";
import { useNodeStore } from "@/stores/node-store";
import { useAuthStore } from "@/stores/auth-store";
import { NodeDetail } from "@/components/nodes/node-detail";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";

export default function NodeDetailPage() {
  const params = useParams<{ id: string }>();
  const nodeId = params.id;
  const { nodes, isLoading, fetchNodes, fetchNodeStatus, fetchNodeVMs } = useNodeStore();

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

  if (!node && isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (!node) {
    return (
      <div className="flex flex-col items-center justify-center py-20">
        <AlertCircle className="mb-4 h-12 w-12 text-muted-foreground" />
        <h2 className="text-xl font-semibold mb-2">Node nicht gefunden</h2>
        <p className="text-muted-foreground mb-4">Der angeforderte Node existiert nicht oder wurde entfernt.</p>
        <Button variant="outline" asChild>
          <Link href="/nodes">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Zurück zur Übersicht
          </Link>
        </Button>
      </div>
    );
  }

  return <NodeDetail node={node} />;
}
