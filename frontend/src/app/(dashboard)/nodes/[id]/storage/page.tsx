"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useNodeStore } from "@/stores/node-store";
import { StorageOverview } from "@/components/nodes/storage-overview";
import { DiskList } from "@/components/nodes/disk-list";
import { Skeleton } from "@/components/ui/skeleton";
import { diskApi, toArray } from "@/lib/api";
import type { DiskInfo } from "@/types/api";

export default function NodeStoragePage() {
  const params = useParams<{ id: string }>();
  const nodeId = params.id;
  const { nodes, nodeStatus, fetchNodes, fetchNodeStatus } = useNodeStore();
  const [disks, setDisks] = useState<DiskInfo[]>([]);

  useEffect(() => {
    if (nodes.length === 0) fetchNodes();
  }, [nodes.length, fetchNodes]);

  useEffect(() => {
    if (nodeId) {
      fetchNodeStatus(nodeId);
      diskApi
        .getDisks(nodeId)
        .then((res) => {
          setDisks(toArray<DiskInfo>(res.data));
        })
        .catch(() => {});
    }
  }, [nodeId, fetchNodeStatus]);

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
        <h1 className="text-2xl font-bold">Storage - {node.name}</h1>
      </div>
      <StorageOverview nodeId={nodeId} status={nodeStatus[nodeId]} />
      <DiskList disks={disks} />
    </div>
  );
}
