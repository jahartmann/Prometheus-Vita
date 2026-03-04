"use client";

import { useEffect, useState } from "react";
import { Plus, Server, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { useNodeStore } from "@/stores/node-store";
import { AddNodeDialog } from "@/components/nodes/add-node-dialog";
import { DeleteNodeDialog } from "@/components/nodes/delete-node-dialog";
import type { Node } from "@/types/api";
import { Skeleton } from "@/components/ui/skeleton";

export default function SettingsNodesPage() {
  const { nodes, isLoading, fetchNodes } = useNodeStore();
  const [addOpen, setAddOpen] = useState(false);
  const [deleteNode, setDeleteNode] = useState<Node | null>(null);

  useEffect(() => {
    fetchNodes();
  }, [fetchNodes]);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">Node-Verwaltung</h2>
          <p className="text-sm text-muted-foreground">
            Proxmox Nodes hinzufuegen und verwalten.
          </p>
        </div>
        <Button onClick={() => setAddOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Node hinzufuegen
        </Button>
      </div>

      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-20 w-full" />
          ))}
        </div>
      ) : nodes.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Server className="mb-3 h-10 w-10 text-muted-foreground" />
            <p className="text-muted-foreground">
              Noch keine Nodes konfiguriert.
            </p>
            <Button
              variant="outline"
              className="mt-4"
              onClick={() => setAddOpen(true)}
            >
              <Plus className="mr-2 h-4 w-4" />
              Ersten Node hinzufuegen
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {nodes.map((node) => (
            <Card key={node.id}>
              <CardContent className="flex items-center justify-between p-4">
                <div className="flex items-center gap-4">
                  <Server className="h-5 w-5 text-muted-foreground" />
                  <div>
                    <div className="flex items-center gap-2">
                      <p className="font-medium">{node.name}</p>
                      <Badge
                        variant={node.is_online ? "success" : "destructive"}
                      >
                        {node.is_online ? "Online" : "Offline"}
                      </Badge>
                      <Badge variant="outline">{node.type.toUpperCase()}</Badge>
                    </div>
                    <p className="text-sm text-muted-foreground">
                      {node.hostname}:{node.port}
                    </p>
                  </div>
                </div>
                <Button
                  variant="ghost"
                  size="icon"
                  className="text-destructive hover:text-destructive"
                  onClick={() => setDeleteNode(node)}
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <AddNodeDialog open={addOpen} onOpenChange={setAddOpen} />
      <DeleteNodeDialog
        node={deleteNode}
        open={!!deleteNode}
        onOpenChange={(open) => {
          if (!open) setDeleteNode(null);
        }}
      />
    </div>
  );
}
