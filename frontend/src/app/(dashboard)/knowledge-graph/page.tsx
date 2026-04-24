"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { GitBranch, HardDrive, Link2, Monitor, Network, RefreshCw, Server } from "lucide-react";
import { operationsApi } from "@/lib/api";
import type { KnowledgeGraphEdge, KnowledgeGraphNode, KnowledgeGraphResponse } from "@/types/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";

function nodeIcon(type: string) {
  if (type === "node") return <Server className="h-4 w-4" />;
  if (type === "vm") return <Monitor className="h-4 w-4" />;
  if (type === "service") return <Network className="h-4 w-4" />;
  return <HardDrive className="h-4 w-4" />;
}

export default function KnowledgeGraphPage() {
  const [graph, setGraph] = useState<KnowledgeGraphResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const load = useCallback(async () => {
    setIsLoading(true);
    try {
      const result = await operationsApi.getKnowledgeGraph() as KnowledgeGraphResponse;
      setGraph(result);
    } catch {
      setGraph(null);
    }
    setIsLoading(false);
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const nodes = graph?.nodes ?? [];
  const edges = graph?.edges ?? [];
  const nodesByType = useMemo(() => {
    return nodes.reduce<Record<string, KnowledgeGraphNode[]>>((acc, node) => {
      acc[node.type] = [...(acc[node.type] || []), node];
      return acc;
    }, {});
  }, [nodes]);
  const dependencyEdges = edges.filter((edge) => edge.type === "depends_on");

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Knowledge Graph</h1>
          <p className="text-sm text-muted-foreground">Serverseitig aggregierte Beziehungen zwischen Nodes, VMs, Diensten, Ports und Abhaengigkeiten.</p>
        </div>
        <Button variant="outline" size="sm" onClick={load} disabled={isLoading}>
          <RefreshCw className={cn("mr-2 h-4 w-4", isLoading && "animate-spin")} />
          Aktualisieren
        </Button>
      </div>

      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">Nodes</CardTitle></CardHeader><CardContent className="text-2xl font-semibold">{graph?.stats.nodes ?? 0}</CardContent></Card>
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">VMs</CardTitle></CardHeader><CardContent className="text-2xl font-semibold">{graph?.stats.vms ?? 0}</CardContent></Card>
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">Devices</CardTitle></CardHeader><CardContent className="text-2xl font-semibold">{graph?.stats.devices ?? 0}</CardContent></Card>
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">Services</CardTitle></CardHeader><CardContent className="text-2xl font-semibold">{graph?.stats.services ?? 0}</CardContent></Card>
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">Dependencies</CardTitle></CardHeader><CardContent className="text-2xl font-semibold">{graph?.stats.dependencies ?? 0}</CardContent></Card>
      </div>

      <div className="grid gap-4 xl:grid-cols-[1fr,360px]">
        <div className="space-y-4">
          {isLoading ? (
            <div className="rounded-md border px-4 py-10 text-center text-sm text-muted-foreground">Knowledge Graph wird geladen...</div>
          ) : nodes.length === 0 ? (
            <div className="rounded-md border px-4 py-10 text-center text-sm text-muted-foreground">Keine Graph-Daten verfuegbar.</div>
          ) : (
            Object.entries(nodesByType).map(([type, group]) => (
              <Card key={type}>
                <CardHeader className="pb-2">
                  <CardTitle className="flex items-center gap-2 text-sm">
                    {nodeIcon(type)}
                    {type}
                    <Badge variant="outline">{group.length}</Badge>
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="grid gap-2 md:grid-cols-2">
                    {group.slice(0, 24).map((node) => (
                      <div key={node.id} className="rounded-md border p-3">
                        <div className="flex items-center gap-2">
                          {nodeIcon(node.type)}
                          <span className="truncate text-sm font-medium">{node.label}</span>
                          {node.status && <Badge variant="secondary">{node.status}</Badge>}
                        </div>
                        <div className="mt-2 flex flex-wrap gap-2">
                          {Object.entries(node.metadata || {}).slice(0, 4).map(([key, value]) => (
                            value ? <span key={`${node.id}-${key}`} className="rounded-md bg-muted px-2 py-1 text-xs text-muted-foreground">{key}: {value}</span> : null
                          ))}
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            ))
          )}
        </div>

        <div className="space-y-4">
          <Card>
            <CardHeader className="pb-2"><CardTitle className="flex items-center gap-2 text-sm"><Link2 className="h-4 w-4" /> Beziehungen</CardTitle></CardHeader>
            <CardContent className="space-y-2">
              {edges.length === 0 ? (
                <p className="text-sm text-muted-foreground">Keine Beziehungen erfasst.</p>
              ) : edges.slice(0, 40).map((edge: KnowledgeGraphEdge) => (
                <div key={edge.id} className="rounded-md border px-3 py-2 text-sm">
                  <div className="flex items-center gap-2">
                    <GitBranch className="h-4 w-4 text-muted-foreground" />
                    <span className="font-medium">{edge.type}</span>
                    {edge.status && <Badge variant="outline">{edge.status}</Badge>}
                  </div>
                  <p className="mt-1 break-all text-xs text-muted-foreground">{edge.from} -&gt; {edge.to}</p>
                </div>
              ))}
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2"><CardTitle className="flex items-center gap-2 text-sm"><GitBranch className="h-4 w-4" /> VM-Abhaengigkeiten</CardTitle></CardHeader>
            <CardContent className="space-y-2">
              {dependencyEdges.length === 0 ? (
                <p className="text-sm text-muted-foreground">Keine VM-Abhaengigkeiten im Graph.</p>
              ) : dependencyEdges.slice(0, 12).map((edge) => (
                <Link key={edge.id} href="/dependencies" className="block rounded-md border px-3 py-2 text-sm hover:bg-muted/50">
                  <span className="font-medium">{edge.label || edge.type}</span>
                  <p className="mt-1 break-all text-xs text-muted-foreground">{edge.from} -&gt; {edge.to}</p>
                </Link>
              ))}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
