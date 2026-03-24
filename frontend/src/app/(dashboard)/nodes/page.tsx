"use client";

import { useEffect } from "react";
import Link from "next/link";
import { Server, Plus, Wifi, WifiOff, Circle, CircleOff } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useNodeStore } from "@/stores/node-store";

export default function NodesPage() {
  const { nodes, isLoading, fetchNodes } = useNodeStore();

  useEffect(() => {
    fetchNodes();
  }, [fetchNodes]);

  const onlineCount = nodes.filter((n) => n.is_online).length;
  const offlineCount = nodes.length - onlineCount;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Nodes</h1>
          <p className="text-muted-foreground">
            Uebersicht aller verbundenen Proxmox Nodes.
          </p>
        </div>
        <Button asChild>
          <Link href="/settings/nodes">
            <Plus className="mr-2 h-4 w-4" />
            Node hinzufuegen
          </Link>
        </Button>
      </div>

      {/* Summary Cards */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardContent className="flex items-center gap-4 p-4">
            <Server className="h-8 w-8 text-muted-foreground" />
            <div>
              <p className="text-2xl font-bold">{nodes.length}</p>
              <p className="text-sm text-muted-foreground">Nodes gesamt</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-4 p-4">
            <Wifi className="h-8 w-8 text-green-500" />
            <div>
              <p className="text-2xl font-bold">{onlineCount}</p>
              <p className="text-sm text-muted-foreground">Online</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-4 p-4">
            <WifiOff className="h-8 w-8 text-red-500" />
            <div>
              <p className="text-2xl font-bold">{offlineCount}</p>
              <p className="text-sm text-muted-foreground">Offline</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Node List */}
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
            <p className="text-muted-foreground">Noch keine Nodes konfiguriert.</p>
            <Button variant="outline" className="mt-4" asChild>
              <Link href="/settings/nodes">
                <Plus className="mr-2 h-4 w-4" />
                Ersten Node hinzufuegen
              </Link>
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {nodes.map((node) => (
            <Link key={node.id} href={`/nodes/${node.id}`}>
              <Card className="transition-colors hover:bg-muted/50">
                <CardContent className="p-4">
                  <div className="flex items-center gap-3">
                    <Server className="h-5 w-5 text-muted-foreground" />
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <p className="font-medium truncate">{node.name}</p>
                        {node.is_online ? (
                          <Badge variant="success" className="gap-1">
                            <Circle className="h-2 w-2 fill-current" />
                            Online
                          </Badge>
                        ) : (
                          <Badge variant="destructive" className="gap-1">
                            <CircleOff className="h-2 w-2" />
                            Offline
                          </Badge>
                        )}
                      </div>
                      <p className="text-sm text-muted-foreground truncate">
                        {node.hostname}:{node.port}
                      </p>
                      <Badge variant="outline" className="mt-1">
                        {node.type.toUpperCase()}
                      </Badge>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
