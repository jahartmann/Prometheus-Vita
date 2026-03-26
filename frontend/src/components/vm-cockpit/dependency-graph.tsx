"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ReactFlow,
  useNodesState,
  useEdgesState,
  Background,
  Controls,
  MiniMap,
  type Node as RFNode,
  type Edge as RFEdge,
  BackgroundVariant,
  Panel,
  MarkerType,
  Handle,
  Position,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import {
  Loader2,
  RefreshCw,
  Plus,
  Trash2,
  Monitor,
  Box,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { vmDependencyApi, toArray } from "@/lib/api";
import type { VMDependency } from "@/types/api";
import { useRouter } from "next/navigation";

// --- Custom VM Node ---

function VMGraphNode({ data }: { data: Record<string, unknown> }) {
  const router = useRouter();
  const status = data.status as string;
  const vmType = data.vm_type as string;
  const vmName = data.vm_name as string;
  const vmid = data.vmid as number;
  const nodeId = data.node_id as string;
  const Icon = vmType === "lxc" ? Box : Monitor;

  return (
    <div
      className="rounded-lg border bg-card p-3 shadow-sm min-w-[150px] cursor-pointer hover:border-primary/50 transition-colors"
      onClick={() => router.push(`/nodes/${nodeId}/vms`)}
    >
      <Handle type="target" position={Position.Left} className="!bg-muted-foreground !w-2 !h-2" />
      <Handle type="source" position={Position.Right} className="!bg-muted-foreground !w-2 !h-2" />
      <div className="flex items-center gap-2">
        <Icon className="h-4 w-4 text-muted-foreground" />
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium truncate">{vmName || `VM ${vmid}`}</p>
          <div className="flex items-center gap-1.5 mt-0.5">
            <Badge variant="outline" className="text-[10px] px-1 py-0 h-4">
              {vmType?.toUpperCase()} {vmid}
            </Badge>
            <span
              className={`h-2 w-2 rounded-full ${
                status === "running"
                  ? "bg-green-500"
                  : status === "stopped"
                    ? "bg-red-500"
                    : "bg-gray-400"
              }`}
            />
          </div>
        </div>
      </div>
    </div>
  );
}

const nodeTypes = {
  vmDep: VMGraphNode,
};

// --- Main Component ---

interface DependencyGraphProps {
  nodeId?: string;
  vmid?: number;
  fullPage?: boolean;
}

export function DependencyGraph({ nodeId, vmid, fullPage }: DependencyGraphProps) {
  const [nodes, setNodes, onNodesChange] = useNodesState<RFNode>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<RFEdge>([]);
  const [loading, setLoading] = useState(true);
  const [dependencies, setDependencies] = useState<VMDependency[]>([]);
  const [dialogOpen, setDialogOpen] = useState(false);

  const fetchDeps = useCallback(async () => {
    setLoading(true);
    try {
      let res;
      if (nodeId && vmid) {
        res = await vmDependencyApi.listByVM(nodeId, vmid);
      } else {
        res = await vmDependencyApi.listAll();
      }
      setDependencies(toArray<VMDependency>(res.data));
    } catch {
      setDependencies([]);
    } finally {
      setLoading(false);
    }
  }, [nodeId, vmid]);

  useEffect(() => {
    fetchDeps();
  }, [fetchDeps]);

  // Build graph from dependencies
  useEffect(() => {
    if (dependencies.length === 0) {
      setNodes([]);
      setEdges([]);
      return;
    }

    const nodeMap = new Map<string, RFNode>();
    const rfEdges: RFEdge[] = [];

    for (const dep of dependencies) {
      const sourceKey = `${dep.source_node_id}-${dep.source_vmid}`;
      const targetKey = `${dep.target_node_id}-${dep.target_vmid}`;

      if (!nodeMap.has(sourceKey)) {
        nodeMap.set(sourceKey, {
          id: sourceKey,
          type: "vmDep",
          data: {
            vm_name: dep.source_vm_name || `VM ${dep.source_vmid}`,
            vmid: dep.source_vmid,
            vm_type: dep.source_vm_type || "qemu",
            status: dep.source_status || "unknown",
            node_id: dep.source_node_id,
          },
          position: { x: 0, y: 0 },
        });
      }

      if (!nodeMap.has(targetKey)) {
        nodeMap.set(targetKey, {
          id: targetKey,
          type: "vmDep",
          data: {
            vm_name: dep.target_vm_name || `VM ${dep.target_vmid}`,
            vmid: dep.target_vmid,
            vm_type: dep.target_vm_type || "qemu",
            status: dep.target_status || "unknown",
            node_id: dep.target_node_id,
          },
          position: { x: 0, y: 0 },
        });
      }

      rfEdges.push({
        id: dep.id,
        source: sourceKey,
        target: targetKey,
        label: dep.dependency_type === "depends_on" ? "hängt ab von" : dep.dependency_type,
        animated: true,
        style: { stroke: "hsl(var(--muted-foreground))", strokeWidth: 2 },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          width: 14,
          height: 14,
          color: "hsl(var(--muted-foreground))",
        },
      });
    }

    // Layout: simple left-to-right layered
    const allNodes = Array.from(nodeMap.values());
    const sourceIds = new Set(rfEdges.map((e) => e.source));
    const targetIds = new Set(rfEdges.map((e) => e.target));

    // Root nodes: nodes that are only sources (not targets)
    const rootIds = new Set<string>();
    for (const id of sourceIds) {
      if (!targetIds.has(id)) rootIds.add(id);
    }
    // If all nodes are also targets, pick first sources
    if (rootIds.size === 0 && sourceIds.size > 0) {
      rootIds.add(sourceIds.values().next().value!);
    }

    // Simple BFS layout
    const positioned = new Set<string>();
    const queue: { id: string; col: number }[] = [];
    const colMap = new Map<string, number>();

    for (const id of rootIds) {
      queue.push({ id, col: 0 });
    }

    while (queue.length > 0) {
      const { id, col } = queue.shift()!;
      if (positioned.has(id)) continue;
      positioned.add(id);
      colMap.set(id, col);

      for (const edge of rfEdges) {
        if (edge.source === id && !positioned.has(edge.target)) {
          queue.push({ id: edge.target, col: col + 1 });
        }
      }
    }

    // Position unpositioned nodes
    for (const node of allNodes) {
      if (!positioned.has(node.id)) {
        colMap.set(node.id, 0);
      }
    }

    // Group by column
    const columns = new Map<number, string[]>();
    for (const [id, col] of colMap) {
      if (!columns.has(col)) columns.set(col, []);
      columns.get(col)!.push(id);
    }

    for (const [col, ids] of columns) {
      ids.forEach((id, row) => {
        const node = nodeMap.get(id);
        if (node) {
          node.position = { x: col * 280, y: row * 100 };
        }
      });
    }

    setNodes(allNodes);
    setEdges(rfEdges);
  }, [dependencies, setNodes, setEdges]);

  const handleDelete = async (depId: string) => {
    try {
      await vmDependencyApi.delete(depId);
      fetchDeps();
    } catch {
      // ignore
    }
  };

  const height = fullPage ? "h-[calc(100vh-12rem)]" : "h-[400px]";

  if (loading) {
    return (
      <div className={`flex ${height} items-center justify-center`}>
        <div className="flex flex-col items-center gap-3">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          <p className="text-sm text-muted-foreground">Abhängigkeiten werden geladen...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className={`${height} w-full rounded-xl border bg-background overflow-hidden`}>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          nodeTypes={nodeTypes}
          fitView
          fitViewOptions={{ padding: 0.3 }}
          minZoom={0.3}
          maxZoom={2}
          proOptions={{ hideAttribution: true }}
        >
          <Background variant={BackgroundVariant.Dots} gap={24} size={1} className="opacity-40" />
          <Controls showInteractive={false} className="!border-border !bg-card !rounded-xl !shadow-lg" />
          <MiniMap
            pannable
            zoomable
            className="!bg-card !border-border !rounded-xl !shadow-lg"
            nodeColor={() => "#3b82f6"}
          />
          <Panel position="top-right">
            <div className="flex gap-2">
              <Button variant="outline" size="sm" onClick={fetchDeps}>
                <RefreshCw className="h-3.5 w-3.5 mr-1" />
                Aktualisieren
              </Button>
              <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
                <DialogTrigger asChild>
                  <Button size="sm">
                    <Plus className="h-3.5 w-3.5 mr-1" />
                    Abhängigkeit
                  </Button>
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>Abhängigkeit erstellen</DialogTitle>
                  </DialogHeader>
                  <CreateDependencyForm
                    defaultNodeId={nodeId}
                    defaultVmid={vmid}
                    onCreated={() => {
                      setDialogOpen(false);
                      fetchDeps();
                    }}
                  />
                </DialogContent>
              </Dialog>
            </div>
          </Panel>
        </ReactFlow>
      </div>

      {/* Dependency list */}
      {dependencies.length > 0 && (
        <div className="space-y-2">
          <h4 className="text-sm font-medium">Abhängigkeiten ({dependencies.length})</h4>
          {dependencies.map((dep) => (
            <div
              key={dep.id}
              className="flex items-center justify-between rounded-lg border p-2 text-sm"
            >
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="text-xs">
                  {dep.source_vm_name || `VM ${dep.source_vmid}`}
                </Badge>
                <span className="text-muted-foreground">
                  {dep.dependency_type === "depends_on" ? "hängt ab von" : dep.dependency_type}
                </span>
                <Badge variant="outline" className="text-xs">
                  {dep.target_vm_name || `VM ${dep.target_vmid}`}
                </Badge>
                {dep.description && (
                  <span className="text-xs text-muted-foreground ml-2">({dep.description})</span>
                )}
              </div>
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7"
                onClick={() => handleDelete(dep.id)}
              >
                <Trash2 className="h-3.5 w-3.5 text-destructive" />
              </Button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// --- Create Dependency Form ---

function CreateDependencyForm({
  defaultNodeId,
  defaultVmid,
  onCreated,
}: {
  defaultNodeId?: string;
  defaultVmid?: number;
  onCreated: () => void;
}) {
  const [sourceNodeId, setSourceNodeId] = useState(defaultNodeId || "");
  const [sourceVmid, setSourceVmid] = useState(defaultVmid?.toString() || "");
  const [targetNodeId, setTargetNodeId] = useState(defaultNodeId || "");
  const [targetVmid, setTargetVmid] = useState("");
  const [depType, setDepType] = useState("depends_on");
  const [description, setDescription] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      await vmDependencyApi.create({
        source_node_id: sourceNodeId,
        source_vmid: Number(sourceVmid),
        target_node_id: targetNodeId,
        target_vmid: Number(targetVmid),
        dependency_type: depType,
        description: description || undefined,
      });
      onCreated();
    } catch {
      // ignore
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-2">
          <Label>Quell-Node-ID</Label>
          <Input
            value={sourceNodeId}
            onChange={(e) => setSourceNodeId(e.target.value)}
            placeholder="Node UUID"
            required
          />
        </div>
        <div className="space-y-2">
          <Label>Quell-VMID</Label>
          <Input
            type="number"
            value={sourceVmid}
            onChange={(e) => setSourceVmid(e.target.value)}
            placeholder="z.B. 100"
            required
          />
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-2">
          <Label>Ziel-Node-ID</Label>
          <Input
            value={targetNodeId}
            onChange={(e) => setTargetNodeId(e.target.value)}
            placeholder="Node UUID"
            required
          />
        </div>
        <div className="space-y-2">
          <Label>Ziel-VMID</Label>
          <Input
            type="number"
            value={targetVmid}
            onChange={(e) => setTargetVmid(e.target.value)}
            placeholder="z.B. 101"
            required
          />
        </div>
      </div>

      <div className="space-y-2">
        <Label>Typ</Label>
        <Input
          value={depType}
          onChange={(e) => setDepType(e.target.value)}
          placeholder="depends_on"
        />
      </div>

      <div className="space-y-2">
        <Label>Beschreibung (optional)</Label>
        <Input
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="z.B. App benötigt Datenbank"
        />
      </div>

      <Button type="submit" className="w-full" disabled={submitting || !sourceVmid || !targetVmid}>
        {submitting ? "Erstelle..." : "Abhängigkeit erstellen"}
      </Button>
    </form>
  );
}
