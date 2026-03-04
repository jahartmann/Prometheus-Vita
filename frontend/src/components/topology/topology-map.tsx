"use client";

import { useMemo } from "react";
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  type Node as RFNode,
  type Edge as RFEdge,
} from "@xyflow/react";
import dagre from "@dagrejs/dagre";
import "@xyflow/react/dist/style.css";

import type { TopologyGraph } from "@/types/api";
import { NodeNode } from "./custom-nodes/node-node";
import { VMNode } from "./custom-nodes/vm-node";
import { StorageNode } from "./custom-nodes/storage-node";
import { NetworkNode } from "./custom-nodes/network-node";

const nodeTypes = {
  host: NodeNode,
  vm: VMNode,
  ct: VMNode,
  storage: StorageNode,
  network: NetworkNode,
};

const nodeWidth = 200;
const nodeHeight = 80;

function getLayoutedElements(
  nodes: RFNode[],
  edges: RFEdge[],
  direction: "TB" | "LR" = "TB"
) {
  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({ rankdir: direction, nodesep: 60, ranksep: 100 });

  nodes.forEach((node) => {
    g.setNode(node.id, { width: nodeWidth, height: nodeHeight });
  });

  edges.forEach((edge) => {
    g.setEdge(edge.source, edge.target);
  });

  dagre.layout(g);

  const layoutedNodes = nodes.map((node) => {
    const nodeWithPosition = g.node(node.id);
    return {
      ...node,
      position: {
        x: nodeWithPosition.x - nodeWidth / 2,
        y: nodeWithPosition.y - nodeHeight / 2,
      },
    };
  });

  return { nodes: layoutedNodes, edges };
}

interface TopologyMapProps {
  graph: TopologyGraph;
}

export function TopologyMap({ graph }: TopologyMapProps) {
  const layouted = useMemo(() => {
    const rfNodes: RFNode[] = graph.nodes.map((n) => ({
      id: n.id,
      type: n.type,
      data: {
        label: n.label,
        status: n.status,
        metadata: n.metadata,
        nodeType: n.type,
      },
      position: { x: 0, y: 0 },
    }));

    const rfEdges: RFEdge[] = graph.edges.map((e, i) => ({
      id: `edge-${i}`,
      source: e.source,
      target: e.target,
      label: e.label,
      type: "smoothstep",
      animated: e.label === "runs",
      style: { stroke: "#64748b" },
    }));

    return getLayoutedElements(rfNodes, rfEdges);
  }, [graph]);

  const [nodes, , onNodesChange] = useNodesState(layouted.nodes);
  const [edges, , onEdgesChange] = useEdgesState(layouted.edges);

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      nodeTypes={nodeTypes}
      fitView
      fitViewOptions={{ padding: 0.2 }}
      minZoom={0.1}
      maxZoom={2}
    >
      <Background />
      <Controls />
      <MiniMap
        nodeStrokeWidth={3}
        pannable
        zoomable
      />
    </ReactFlow>
  );
}
