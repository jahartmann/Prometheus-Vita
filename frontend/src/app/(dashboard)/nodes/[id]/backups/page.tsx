"use client";

import { useEffect } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useNodeStore } from "@/stores/node-store";
import { useBackupStore } from "@/stores/backup-store";
import { BackupList } from "@/components/backup/backup-list";
import { Skeleton } from "@/components/ui/skeleton";

export default function NodeBackupsPage() {
  const params = useParams<{ id: string }>();
  const nodeId = params.id;
  const { nodes, fetchNodes } = useNodeStore();
  const { fetchBackups, fetchSchedules } = useBackupStore();

  useEffect(() => {
    if (nodes.length === 0) fetchNodes();
  }, [nodes.length, fetchNodes]);

  useEffect(() => {
    if (nodeId) {
      fetchBackups(nodeId);
      fetchSchedules(nodeId);
    }
  }, [nodeId, fetchBackups, fetchSchedules]);

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
        <h1 className="text-2xl font-bold">Backups - {node.name}</h1>
      </div>
      <BackupList nodeId={nodeId} />
    </div>
  );
}
