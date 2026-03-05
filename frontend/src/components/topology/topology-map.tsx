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
  useReactFlow,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import dagre from "@dagrejs/dagre";
import { Loader2, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import api from "@/lib/api";
import { useNodeStore } from "@/stores/node-store";
import { HostNode } from "./host-node";
import { VMNode } from "./vm-node";
import { StorageNode } from "./storage-node";
import { NetworkNode } from "./network-node";

interface TopologyNode {
  id: string;
  type: string;
  label: string;
  status: string;
  metadata?: Record<string, unknown>;
}

interface TopologyEdge {
  source: string;
  target: string;
  label?: string;
}

interface TopologyGraph {
  nodes: TopologyNode[];
  edges: TopologyEdge[];
}

const nodeTypes = {
  host: HostNode,
  vm: VMNode,
  ct: VMNode,
  storage: StorageNode,
  network: NetworkNode,
};

const NODE_WIDTH: Record<string, number> = {
  host: 200,
  vm: 160,
  ct: 160,
  storage: 150,
  network: 140,
};

const NODE_HEIGHT: Record<string, number> = {
  host: 120,
  vm: 60,
  ct: 60,
  storage: 70,
  network: 50,
};

function layoutGraph(rfNodes: RFNode[], rfEdges: RFEdge[]): RFNode[] {
  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({ rankdir: "TB", nodesep: 40, ranksep: 80, edgesep: 20 });

  rfNodes.forEach((node) => {
    const t = node.type ?? "host";
    g.setNode(node.id, {
      width: NODE_WIDTH[t] ?? 160,
      height: NODE_HEIGHT[t] ?? 60,
    });
  });

  rfEdges.forEach((edge) => {
    g.setEdge(edge.source, edge.target);
  });

  dagre.layout(g);

  return rfNodes.map((node) => {
    const pos = g.node(node.id);
    const t = node.type ?? "host";
    return {
      ...node,
      position: {
        x: pos.x - (NODE_WIDTH[t] ?? 160) / 2,
        y: pos.y - (NODE_HEIGHT[t] ?? 60) / 2,
      },
    };
  });
}

function buildReactFlowGraph(
  graph: TopologyGraph,
  nodeStatusMap: Record<string, { cpu_usage: number; memory_used: number; memory_total: number; vm_count: number; ct_count: number }>
): { nodes: RFNode[]; edges: RFEdge[] } {
  const rfNodes: RFNode[] = graph.nodes.map((n) => {
    const baseData: Record<string, unknown> = {
      label: n.label,
      status: n.status,
    };

    if (n.type === "host") {
      baseData.hostname = n.metadata?.hostname;
      baseData.nodeType = n.metadata?.type;
      const statusData = nodeStatusMap[n.id];
      if (statusData) {
        baseData.cpuUsage = statusData.cpu_usage;
        baseData.memoryPercent =
          statusData.memory_total > 0
            ? (statusData.memory_used / statusData.memory_total) * 100
            : 0;
        baseData.vmCount = statusData.vm_count;
        baseData.ctCount = statusData.ct_count;
      }
    } else if (n.type === "vm" || n.type === "ct") {
      baseData.vmType = n.type;
      baseData.vmid = n.metadata?.vmid;
    } else if (n.type === "storage") {
      baseData.storageType = n.metadata?.type;
      baseData.total = n.metadata?.total;
      baseData.used = n.metadata?.used;
    } else if (n.type === "network") {
      baseData.cidr = n.metadata?.cidr;
    }

    return {
      id: n.id,
      type: n.type,
      data: baseData,
      position: { x: 0, y: 0 },
    };
  });

  const rfEdges: RFEdge[] = graph.edges.map((e, i) => ({
    id: `e-${i}-${e.source}-${e.target}`,
    source: e.source,
    target: e.target,
    animated: e.label === "runs",
    style: { stroke: "hsl(var(--muted-foreground))", strokeWidth: 1.5 },
    label: undefined,
  }));

  const laidOut = layoutGraph(rfNodes, rfEdges);
  return { nodes: laidOut, edges: rfEdges };
}

export function TopologyMap() {
  const [nodes, setNodes, onNodesChange] = useNodesState<RFNode>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<RFEdge>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const { nodeStatus, fetchNodes, fetchNodeStatus } = useNodeStore();
  const { fitView } = useReactFlow();

  const nodeStatusMap = useMemo(() => {
    const map: Record<string, { cpu_usage: number; memory_used: number; memory_total: number; vm_count: number; ct_count: number }> = {};
    for (const [id, s] of Object.entries(nodeStatus)) {
      map[id] = {
        cpu_usage: s.cpu_usage,
        memory_used: s.memory_used,
        memory_total: s.memory_total,
        vm_count: s.vm_count,
        ct_count: s.ct_count,
      };
    }
    return map;
  }, [nodeStatus]);

  const loadTopology = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.get<TopologyGraph>("/topology");
      const graph = res.data as unknown as TopologyGraph;
      if (!graph || !graph.nodes) {
        setNodes([]);
        setEdges([]);
        return;
      }
      const { nodes: rfNodes, edges: rfEdges } = buildReactFlowGraph(graph, nodeStatusMap);
      setNodes(rfNodes);
      setEdges(rfEdges);
      setTimeout(() => fitView({ padding: 0.2 }), 100);
    } catch {
      setError("Topologie konnte nicht geladen werden.");
    } finally {
      setLoading(false);
    }
  }, [nodeStatusMap, setNodes, setEdges, fitView]);

  useEffect(() => {
    fetchNodes().then(() => {
      const storeNodes = useNodeStore.getState().nodes;
      storeNodes.forEach((n) => {
        if (n.is_online) fetchNodeStatus(n.id);
      });
    });
  }, [fetchNodes, fetchNodeStatus]);

  useEffect(() => {
    loadTopology();
  }, [loadTopology]);

  const stats = useMemo(() => {
    const hosts = nodes.filter((n) => n.type === "host").length;
    const vms = nodes.filter((n) => n.type === "vm").length;
    const cts = nodes.filter((n) => n.type === "ct").length;
    const storages = nodes.filter((n) => n.type === "storage").length;
    const networks = nodes.filter((n) => n.type === "network").length;
    return { hosts, vms, cts, storages, networks };
  }, [nodes]);

  if (loading) {
    return (
      <div className="flex h-[600px] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex h-[600px] flex-col items-center justify-center gap-4">
        <p className="text-muted-foreground">{error}</p>
        <Button variant="outline" onClick={loadTopology}>
          <RefreshCw className="mr-2 h-4 w-4" />
          Erneut versuchen
        </Button>
      </div>
    );
  }

  return (
    <div className="h-[calc(100vh-12rem)] w-full rounded-lg border bg-background">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        nodeTypes={nodeTypes}
        fitView
        fitViewOptions={{ padding: 0.2 }}
        minZoom={0.2}
        maxZoom={2}
        proOptions={{ hideAttribution: true }}
      >
        <Background variant={BackgroundVariant.Dots} gap={20} size={1} />
        <Controls showInteractive={false} />
        <MiniMap
          nodeStrokeWidth={2}
          pannable
          zoomable
          className="!bg-card !border-border"
        />
        <Panel position="top-right" className="flex gap-2">
          <Badge variant="outline" className="text-xs">
            {stats.hosts} Hosts
          </Badge>
          <Badge variant="outline" className="text-xs">
            {stats.vms} VMs
          </Badge>
          <Badge variant="outline" className="text-xs">
            {stats.cts} CTs
          </Badge>
          <Badge variant="outline" className="text-xs">
            {stats.storages} Storage
          </Badge>
          <Badge variant="outline" className="text-xs">
            {stats.networks} Netzwerk
          </Badge>
          <Button variant="outline" size="sm" onClick={loadTopology}>
            <RefreshCw className="h-3.5 w-3.5" />
          </Button>
        </Panel>
      </ReactFlow>
    </div>
  );
}
