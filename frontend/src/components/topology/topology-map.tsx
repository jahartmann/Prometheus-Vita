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
  MarkerType,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import {
  Loader2,
  RefreshCw,
  Search,
  LayoutGrid,
  Maximize2,
  Network,
  Server,
  Monitor,
  Box,
  HardDrive,
  Filter,
  X,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import api from "@/lib/api";
import { useNodeStore } from "@/stores/node-store";
import { HostNode } from "./host-node";
import { HostGroupNode } from "./host-group-node";
import { VMNode } from "./vm-node";
import { StorageNode } from "./storage-node";
import { NetworkNode } from "./network-node";

// --- Types ---

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

type ViewMode = "cluster" | "detailed" | "network";
type StatusFilter = "all" | "running" | "stopped";
type TypeFilter = "all" | "vm" | "ct";

// --- Constants ---

const nodeTypes = {
  host: HostNode,
  hostGroup: HostGroupNode,
  vm: VMNode,
  ct: VMNode,
  storage: StorageNode,
  network: NetworkNode,
};

const VM_CARD_WIDTH = 170;
const VM_CARD_HEIGHT = 90;
const VM_GAP = 12;
const VM_COLS = 3;
const HOST_HEADER_HEIGHT = 140;
const HOST_PADDING_X = 16;
const HOST_PADDING_BOTTOM = 20;

// --- Helper: compute host group dimensions based on children ---

function computeHostDimensions(childCount: number): { width: number; height: number } {
  if (childCount === 0) {
    return { width: 380, height: HOST_HEADER_HEIGHT + 40 };
  }
  const cols = Math.min(childCount, VM_COLS);
  const rows = Math.ceil(childCount / VM_COLS);
  const width = HOST_PADDING_X * 2 + cols * VM_CARD_WIDTH + (cols - 1) * VM_GAP + 20;
  const height = HOST_HEADER_HEIGHT + rows * VM_CARD_HEIGHT + (rows - 1) * VM_GAP + HOST_PADDING_BOTTOM;
  return { width: Math.max(width, 380), height };
}

function getChildPosition(index: number): { x: number; y: number } {
  const col = index % VM_COLS;
  const row = Math.floor(index / VM_COLS);
  return {
    x: HOST_PADDING_X + col * (VM_CARD_WIDTH + VM_GAP),
    y: HOST_HEADER_HEIGHT + row * (VM_CARD_HEIGHT + VM_GAP),
  };
}

// --- Build nodes/edges for each view mode ---

function buildDetailedView(
  graph: TopologyGraph,
  nodeStatusMap: Record<string, { cpu_usage: number; memory_used: number; memory_total: number; vm_count: number; ct_count: number; vm_running?: number; ct_running?: number }>,
  expandedHosts: Set<string>,
  onToggleExpand: (nodeId: string) => void,
  searchQuery: string,
  statusFilter: StatusFilter,
  typeFilter: TypeFilter,
  tagFilter: string,
): { nodes: RFNode[]; edges: RFEdge[] } {
  const rfNodes: RFNode[] = [];
  const rfEdges: RFEdge[] = [];

  // Find host nodes and their children
  const hostNodes = graph.nodes.filter((n) => n.type === "host");
  const childEdges = graph.edges.filter((e) => e.label === "runs" || e.label === "has_storage" || e.label === "has_network");
  const hostChildMap = new Map<string, TopologyNode[]>();
  const storageByHost = new Map<string, TopologyNode[]>();
  const networkByHost = new Map<string, TopologyNode[]>();

  // Map children to hosts
  for (const edge of childEdges) {
    const child = graph.nodes.find((n) => n.id === edge.target);
    if (!child) continue;
    if (child.type === "vm" || child.type === "ct") {
      if (!hostChildMap.has(edge.source)) hostChildMap.set(edge.source, []);
      hostChildMap.get(edge.source)!.push(child);
    } else if (child.type === "storage") {
      if (!storageByHost.has(edge.source)) storageByHost.set(edge.source, []);
      storageByHost.get(edge.source)!.push(child);
    } else if (child.type === "network") {
      if (!networkByHost.has(edge.source)) networkByHost.set(edge.source, []);
      networkByHost.get(edge.source)!.push(child);
    }
  }

  // Also check edges by source being a host for "runs" type
  for (const edge of graph.edges) {
    const sourceNode = graph.nodes.find((n) => n.id === edge.source);
    const targetNode = graph.nodes.find((n) => n.id === edge.target);
    if (!sourceNode || !targetNode) continue;
    if (sourceNode.type === "host" && (targetNode.type === "vm" || targetNode.type === "ct")) {
      if (!hostChildMap.has(edge.source)) hostChildMap.set(edge.source, []);
      const existing = hostChildMap.get(edge.source)!;
      if (!existing.find((c) => c.id === targetNode.id)) {
        existing.push(targetNode);
      }
    }
    if (sourceNode.type === "host" && targetNode.type === "storage") {
      if (!storageByHost.has(edge.source)) storageByHost.set(edge.source, []);
      const existing = storageByHost.get(edge.source)!;
      if (!existing.find((c) => c.id === targetNode.id)) {
        existing.push(targetNode);
      }
    }
    if (sourceNode.type === "host" && targetNode.type === "network") {
      if (!networkByHost.has(edge.source)) networkByHost.set(edge.source, []);
      const existing = networkByHost.get(edge.source)!;
      if (!existing.find((c) => c.id === targetNode.id)) {
        existing.push(targetNode);
      }
    }
  }

  const searchLower = searchQuery.toLowerCase();
  const isSearching = searchQuery.length > 0;

  // Place hosts side by side
  let hostX = 0;
  const HOST_GAP = 60;

  for (const host of hostNodes) {
    const isExpanded = expandedHosts.has(host.id);
    const statusData = nodeStatusMap[host.id];
    let children = hostChildMap.get(host.id) ?? [];
    const storageItems = (storageByHost.get(host.id) ?? []).map((s) => ({
      label: s.label,
      type: (s.metadata?.type as string) ?? "",
      usagePercent: s.metadata?.total && s.metadata?.used
        ? ((s.metadata.used as number) / (s.metadata.total as number)) * 100
        : 0,
    }));

    // Apply filters to children
    children = children.filter((child) => {
      if (statusFilter !== "all") {
        if (statusFilter === "running" && child.status !== "running") return false;
        if (statusFilter === "stopped" && child.status !== "stopped") return false;
      }
      if (typeFilter !== "all") {
        if (typeFilter === "vm" && child.type !== "vm") return false;
        if (typeFilter === "ct" && child.type !== "ct") return false;
      }
      if (tagFilter) {
        const tags = (child.metadata?.tags as string[]) ?? [];
        if (!tags.some((t) => t.toLowerCase().includes(tagFilter.toLowerCase()))) return false;
      }
      return true;
    });

    const visibleChildren = isExpanded ? children : [];
    const { width, height } = computeHostDimensions(visibleChildren.length);

    const hostData: Record<string, unknown> = {
      label: host.label,
      status: host.status,
      hostname: host.metadata?.hostname,
      nodeType: host.metadata?.type,
      expanded: isExpanded,
      onToggleExpand,
      nodeId: host.id,
      containerWidth: width,
      containerHeight: height,
      storageItems,
    };

    if (statusData) {
      hostData.cpuUsage = statusData.cpu_usage;
      hostData.memoryPercent =
        statusData.memory_total > 0
          ? (statusData.memory_used / statusData.memory_total) * 100
          : 0;
      hostData.vmCount = statusData.vm_count;
      hostData.ctCount = statusData.ct_count;
      hostData.vmRunning = statusData.vm_running;
      hostData.ctRunning = statusData.ct_running;
    }

    rfNodes.push({
      id: host.id,
      type: "hostGroup",
      data: hostData,
      position: { x: hostX, y: 0 },
      style: { width, height },
    });

    // Add child VM/CT nodes inside the host
    visibleChildren.forEach((child, i) => {
      const pos = getChildPosition(i);
      const isMatch = isSearching && child.label.toLowerCase().includes(searchLower);
      const isDimmed = isSearching && !isMatch;

      const childData: Record<string, unknown> = {
        label: child.label,
        status: child.status,
        vmType: child.type,
        vmid: child.metadata?.vmid,
        tags: child.metadata?.tags,
        highlighted: isMatch,
        dimmed: isDimmed,
      };

      rfNodes.push({
        id: child.id,
        type: child.type === "ct" ? "ct" : "vm",
        data: childData,
        position: pos,
        parentId: host.id,
        extent: "parent" as const,
      });
    });

    hostX += width + HOST_GAP;
  }

  // Place storage nodes that are not hosted under a specific host
  const assignedStorageIds = new Set<string>();
  for (const storages of storageByHost.values()) {
    for (const s of storages) assignedStorageIds.add(s.id);
  }
  const unassignedStorage = graph.nodes.filter((n) => n.type === "storage" && !assignedStorageIds.has(n.id));
  let storageX = 0;
  for (const storage of unassignedStorage) {
    const isMatch = isSearching && storage.label.toLowerCase().includes(searchLower);
    const isDimmed = isSearching && !isMatch;
    rfNodes.push({
      id: storage.id,
      type: "storage",
      data: {
        label: storage.label,
        status: storage.status,
        storageType: storage.metadata?.type,
        total: storage.metadata?.total,
        used: storage.metadata?.used,
        highlighted: isMatch,
        dimmed: isDimmed,
      },
      position: { x: storageX, y: hostNodes.length > 0 ? 500 : 0 },
    });
    storageX += 200;
  }

  // Place network nodes on the right side
  const assignedNetworkIds = new Set<string>();
  for (const networks of networkByHost.values()) {
    for (const n of networks) assignedNetworkIds.add(n.id);
  }
  const unassignedNetworks = graph.nodes.filter((n) => n.type === "network" && !assignedNetworkIds.has(n.id));
  let networkY = 0;
  for (const net of unassignedNetworks) {
    const isMatch = isSearching && net.label.toLowerCase().includes(searchLower);
    const isDimmed = isSearching && !isMatch;
    rfNodes.push({
      id: net.id,
      type: "network",
      data: {
        label: net.label,
        status: net.status,
        cidr: net.metadata?.cidr,
        highlighted: isMatch,
        dimmed: isDimmed,
      },
      position: { x: hostX + 80, y: networkY },
    });
    networkY += 100;
  }

  // Edges between hosts (cluster connectivity)
  const hostIds = new Set(hostNodes.map((h) => h.id));
  for (const edge of graph.edges) {
    if (hostIds.has(edge.source) && hostIds.has(edge.target)) {
      rfEdges.push({
        id: `e-host-${edge.source}-${edge.target}`,
        source: edge.source,
        target: edge.target,
        animated: true,
        style: { stroke: "hsl(var(--muted-foreground))", strokeWidth: 1.5, strokeDasharray: "6 3" },
        markerEnd: { type: MarkerType.ArrowClosed, width: 12, height: 12, color: "hsl(var(--muted-foreground))" },
      });
    }
  }

  // Edges from hosts to unassigned network/storage
  for (const edge of graph.edges) {
    const isHostSource = hostIds.has(edge.source);
    const targetNode = graph.nodes.find((n) => n.id === edge.target);
    if (isHostSource && targetNode && (targetNode.type === "network" || targetNode.type === "storage")) {
      if (!assignedStorageIds.has(targetNode.id) && !assignedNetworkIds.has(targetNode.id)) {
        continue; // skip, these are standalone
      }
      // Only draw if not assigned inside the host group
    }
  }

  return { nodes: rfNodes, edges: rfEdges };
}

function buildClusterView(
  graph: TopologyGraph,
  nodeStatusMap: Record<string, { cpu_usage: number; memory_used: number; memory_total: number; vm_count: number; ct_count: number; vm_running?: number; ct_running?: number }>,
  searchQuery: string,
): { nodes: RFNode[]; edges: RFEdge[] } {
  const rfNodes: RFNode[] = [];
  const rfEdges: RFEdge[] = [];

  const hostNodes = graph.nodes.filter((n) => n.type === "host");
  const searchLower = searchQuery.toLowerCase();
  const isSearching = searchQuery.length > 0;

  // Collect storage per host for summary
  const storageByHost = new Map<string, TopologyNode[]>();
  for (const edge of graph.edges) {
    const sourceNode = graph.nodes.find((n) => n.id === edge.source);
    const targetNode = graph.nodes.find((n) => n.id === edge.target);
    if (sourceNode?.type === "host" && targetNode?.type === "storage") {
      if (!storageByHost.has(edge.source)) storageByHost.set(edge.source, []);
      storageByHost.get(edge.source)!.push(targetNode);
    }
  }

  const CLUSTER_GAP = 60;
  let x = 0;

  for (const host of hostNodes) {
    const statusData = nodeStatusMap[host.id];
    const isMatch = isSearching && host.label.toLowerCase().includes(searchLower);
    const isDimmed = isSearching && !isMatch;

    const storageItems = (storageByHost.get(host.id) ?? []).map((s) => ({
      label: s.label,
      type: (s.metadata?.type as string) ?? "",
      usagePercent: s.metadata?.total && s.metadata?.used
        ? ((s.metadata.used as number) / (s.metadata.total as number)) * 100
        : 0,
    }));

    const hostData: Record<string, unknown> = {
      label: host.label,
      status: host.status,
      hostname: host.metadata?.hostname,
      nodeType: host.metadata?.type,
      storageItems,
    };

    if (statusData) {
      hostData.cpuUsage = statusData.cpu_usage;
      hostData.memoryPercent =
        statusData.memory_total > 0
          ? (statusData.memory_used / statusData.memory_total) * 100
          : 0;
      hostData.vmCount = statusData.vm_count;
      hostData.ctCount = statusData.ct_count;
      hostData.vmRunning = statusData.vm_running;
      hostData.ctRunning = statusData.ct_running;
    }

    rfNodes.push({
      id: host.id,
      type: "host",
      data: hostData,
      position: { x, y: 0 },
      className: isDimmed ? "opacity-30" : "",
    });

    x += 300 + CLUSTER_GAP;
  }

  // Edges between hosts
  const hostIds = new Set(hostNodes.map((h) => h.id));
  for (const edge of graph.edges) {
    if (hostIds.has(edge.source) && hostIds.has(edge.target)) {
      rfEdges.push({
        id: `e-cluster-${edge.source}-${edge.target}`,
        source: edge.source,
        target: edge.target,
        animated: true,
        style: { stroke: "hsl(var(--muted-foreground))", strokeWidth: 2, strokeDasharray: "8 4" },
        markerEnd: { type: MarkerType.ArrowClosed, width: 14, height: 14, color: "hsl(var(--muted-foreground))" },
      });
    }
  }

  return { nodes: rfNodes, edges: rfEdges };
}

function buildNetworkView(
  graph: TopologyGraph,
  searchQuery: string,
): { nodes: RFNode[]; edges: RFEdge[] } {
  const rfNodes: RFNode[] = [];
  const rfEdges: RFEdge[] = [];

  const networkNodes = graph.nodes.filter((n) => n.type === "network");
  const vmNodes = graph.nodes.filter((n) => n.type === "vm" || n.type === "ct");
  const hostNodes = graph.nodes.filter((n) => n.type === "host");
  const searchLower = searchQuery.toLowerCase();
  const isSearching = searchQuery.length > 0;

  // Find which VMs connect to which networks
  const vmToNetwork = new Map<string, string[]>();
  const networkToVMs = new Map<string, string[]>();

  for (const edge of graph.edges) {
    const sourceNode = graph.nodes.find((n) => n.id === edge.source);
    const targetNode = graph.nodes.find((n) => n.id === edge.target);

    if (sourceNode?.type === "network" && (targetNode?.type === "vm" || targetNode?.type === "ct")) {
      if (!networkToVMs.has(edge.source)) networkToVMs.set(edge.source, []);
      networkToVMs.get(edge.source)!.push(edge.target);
      if (!vmToNetwork.has(edge.target)) vmToNetwork.set(edge.target, []);
      vmToNetwork.get(edge.target)!.push(edge.source);
    }
    if ((sourceNode?.type === "vm" || sourceNode?.type === "ct") && targetNode?.type === "network") {
      if (!networkToVMs.has(edge.target)) networkToVMs.set(edge.target, []);
      networkToVMs.get(edge.target)!.push(edge.source);
      if (!vmToNetwork.has(edge.source)) vmToNetwork.set(edge.source, []);
      vmToNetwork.get(edge.source)!.push(edge.target);
    }
  }

  // Find which VMs belong to which host
  const vmToHost = new Map<string, string>();
  for (const edge of graph.edges) {
    const sourceNode = graph.nodes.find((n) => n.id === edge.source);
    const targetNode = graph.nodes.find((n) => n.id === edge.target);
    if (sourceNode?.type === "host" && (targetNode?.type === "vm" || targetNode?.type === "ct")) {
      vmToHost.set(edge.target, edge.source);
    }
  }

  // Place networks in a column on the left
  let netY = 0;
  const NET_GAP = 120;

  for (const net of networkNodes) {
    const connectedVMs = networkToVMs.get(net.id) ?? [];
    const isMatch = isSearching && net.label.toLowerCase().includes(searchLower);
    const isDimmed = isSearching && !isMatch;

    rfNodes.push({
      id: net.id,
      type: "network",
      data: {
        label: net.label,
        status: net.status,
        cidr: net.metadata?.cidr,
        vmCount: connectedVMs.length,
        highlighted: isMatch,
        dimmed: isDimmed,
      },
      position: { x: 0, y: netY },
    });

    netY += NET_GAP;
  }

  // Place hosts next to the networks
  let hostY = 0;
  const HOST_X = 350;

  for (const host of hostNodes) {
    const isMatch = isSearching && host.label.toLowerCase().includes(searchLower);
    const isDimmed = isSearching && !isMatch;

    rfNodes.push({
      id: host.id,
      type: "host",
      data: {
        label: host.label,
        status: host.status,
        hostname: host.metadata?.hostname,
      },
      position: { x: HOST_X, y: hostY },
      className: isDimmed ? "opacity-30" : "",
    });

    hostY += 180;
  }

  // Place VMs to the right of hosts, grouped by their host
  const VM_X = 650;
  let vmY = 0;
  const placedVMs = new Set<string>();

  for (const host of hostNodes) {
    const hostVMs = vmNodes.filter((vm) => vmToHost.get(vm.id) === host.id);
    for (const vm of hostVMs) {
      if (placedVMs.has(vm.id)) continue;
      placedVMs.add(vm.id);

      const isMatch = isSearching && vm.label.toLowerCase().includes(searchLower);
      const isDimmed = isSearching && !isMatch;

      rfNodes.push({
        id: vm.id,
        type: vm.type === "ct" ? "ct" : "vm",
        data: {
          label: vm.label,
          status: vm.status,
          vmType: vm.type,
          vmid: vm.metadata?.vmid,
          highlighted: isMatch,
          dimmed: isDimmed,
        },
        position: { x: VM_X, y: vmY },
      });
      vmY += 80;
    }
  }

  // Place any VMs not assigned to a host
  for (const vm of vmNodes) {
    if (placedVMs.has(vm.id)) continue;
    placedVMs.add(vm.id);

    const isMatch = isSearching && vm.label.toLowerCase().includes(searchLower);
    const isDimmed = isSearching && !isMatch;

    rfNodes.push({
      id: vm.id,
      type: vm.type === "ct" ? "ct" : "vm",
      data: {
        label: vm.label,
        status: vm.status,
        vmType: vm.type,
        vmid: vm.metadata?.vmid,
        highlighted: isMatch,
        dimmed: isDimmed,
      },
      position: { x: VM_X, y: vmY },
    });
    vmY += 80;
  }

  // Edges: host -> VM
  for (const [vmId, hostId] of vmToHost.entries()) {
    rfEdges.push({
      id: `e-net-host-vm-${hostId}-${vmId}`,
      source: hostId,
      target: vmId,
      style: { stroke: "hsl(var(--muted-foreground))", strokeWidth: 1, opacity: 0.4 },
    });
  }

  // Edges: network -> VM
  for (const [netId, vmIds] of networkToVMs.entries()) {
    for (const vmId of vmIds) {
      rfEdges.push({
        id: `e-net-${netId}-${vmId}`,
        source: netId,
        target: vmId,
        animated: true,
        style: { stroke: "#22d3ee", strokeWidth: 1.5 },
        markerEnd: { type: MarkerType.ArrowClosed, width: 10, height: 10, color: "#22d3ee" },
      });
    }
  }

  // Edges: network -> host (if direct)
  for (const edge of graph.edges) {
    const sourceNode = graph.nodes.find((n) => n.id === edge.source);
    const targetNode = graph.nodes.find((n) => n.id === edge.target);
    if (sourceNode?.type === "host" && targetNode?.type === "network") {
      rfEdges.push({
        id: `e-net-host-net-${edge.source}-${edge.target}`,
        source: edge.target,
        target: edge.source,
        style: { stroke: "#22d3ee", strokeWidth: 1, strokeDasharray: "4 4", opacity: 0.6 },
      });
    }
  }

  return { nodes: rfNodes, edges: rfEdges };
}

// --- Main Component ---

export function TopologyMap() {
  const [nodes, setNodes, onNodesChange] = useNodesState<RFNode>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<RFEdge>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [graphData, setGraphData] = useState<TopologyGraph | null>(null);

  const [viewMode, setViewMode] = useState<ViewMode>("detailed");
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");
  const [typeFilter, setTypeFilter] = useState<TypeFilter>("all");
  const [tagFilter, setTagFilter] = useState("");
  const [expandedHosts, setExpandedHosts] = useState<Set<string>>(new Set());
  const [showFilters, setShowFilters] = useState(false);

  const { nodeStatus, fetchNodes, fetchNodeStatus } = useNodeStore();
  const { fitView } = useReactFlow();

  const nodeStatusMap = useMemo(() => {
    const map: Record<string, { cpu_usage: number; memory_used: number; memory_total: number; vm_count: number; ct_count: number; vm_running?: number; ct_running?: number }> = {};
    for (const [id, s] of Object.entries(nodeStatus)) {
      map[id] = {
        cpu_usage: s.cpu_usage,
        memory_used: s.memory_used,
        memory_total: s.memory_total,
        vm_count: s.vm_count,
        ct_count: s.ct_count,
        vm_running: s.vm_running,
        ct_running: s.ct_running,
      };
    }
    return map;
  }, [nodeStatus]);

  const handleToggleExpand = useCallback((nodeId: string) => {
    setExpandedHosts((prev) => {
      const next = new Set(prev);
      if (next.has(nodeId)) {
        next.delete(nodeId);
      } else {
        next.add(nodeId);
      }
      return next;
    });
  }, []);

  // Load topology data
  const loadTopology = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.get<TopologyGraph>("/topology");
      const graph = res.data as unknown as TopologyGraph;
      if (!graph || !graph.nodes) {
        setGraphData({ nodes: [], edges: [] });
        return;
      }
      setGraphData(graph);

      // Auto-expand all hosts on first load
      const hostIds = graph.nodes.filter((n) => n.type === "host").map((n) => n.id);
      setExpandedHosts(new Set(hostIds));
    } catch {
      setError("Topologie konnte nicht geladen werden.");
    } finally {
      setLoading(false);
    }
  }, []);

  // Fetch node statuses
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

  // Rebuild graph whenever view/filters/data change
  useEffect(() => {
    if (!graphData) return;

    let result: { nodes: RFNode[]; edges: RFEdge[] };

    switch (viewMode) {
      case "cluster":
        result = buildClusterView(graphData, nodeStatusMap, searchQuery);
        break;
      case "network":
        result = buildNetworkView(graphData, searchQuery);
        break;
      case "detailed":
      default:
        result = buildDetailedView(
          graphData,
          nodeStatusMap,
          expandedHosts,
          handleToggleExpand,
          searchQuery,
          statusFilter,
          typeFilter,
          tagFilter,
        );
        break;
    }

    setNodes(result.nodes);
    setEdges(result.edges);
    setTimeout(() => fitView({ padding: 0.15, duration: 300 }), 150);
  }, [graphData, viewMode, nodeStatusMap, expandedHosts, handleToggleExpand, searchQuery, statusFilter, typeFilter, tagFilter, setNodes, setEdges, fitView]);

  // Compute stats
  const stats = useMemo(() => {
    if (!graphData) return { hosts: 0, vms: 0, cts: 0, storages: 0, networks: 0, totalCpu: 0, totalRam: 0 };
    const hosts = graphData.nodes.filter((n) => n.type === "host").length;
    const vms = graphData.nodes.filter((n) => n.type === "vm").length;
    const cts = graphData.nodes.filter((n) => n.type === "ct").length;
    const storages = graphData.nodes.filter((n) => n.type === "storage").length;
    const networks = graphData.nodes.filter((n) => n.type === "network").length;

    let totalCpu = 0;
    let totalRam = 0;
    let hostCount = 0;
    for (const s of Object.values(nodeStatusMap)) {
      totalCpu += s.cpu_usage;
      totalRam += s.memory_total > 0 ? (s.memory_used / s.memory_total) * 100 : 0;
      hostCount++;
    }
    if (hostCount > 0) {
      totalCpu = totalCpu / hostCount;
      totalRam = totalRam / hostCount;
    }

    return { hosts, vms, cts, storages, networks, totalCpu, totalRam };
  }, [graphData, nodeStatusMap]);

  const hasActiveFilters = statusFilter !== "all" || typeFilter !== "all" || tagFilter !== "";

  if (loading) {
    return (
      <div className="flex h-[600px] items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          <p className="text-sm text-muted-foreground">Topologie wird geladen...</p>
        </div>
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
    <div className="h-[calc(100vh-12rem)] w-full rounded-xl border bg-background overflow-hidden relative">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        nodeTypes={nodeTypes}
        fitView
        fitViewOptions={{ padding: 0.15 }}
        minZoom={0.1}
        maxZoom={2.5}
        proOptions={{ hideAttribution: true }}
        defaultEdgeOptions={{
          style: { strokeWidth: 1.5 },
        }}
      >
        <Background variant={BackgroundVariant.Dots} gap={24} size={1} className="opacity-40" />
        <Controls showInteractive={false} className="!border-border !bg-card/90 !backdrop-blur-sm !rounded-xl !shadow-lg" />
        <MiniMap
          nodeStrokeWidth={2}
          pannable
          zoomable
          className="!bg-card/90 !backdrop-blur-sm !border-border !rounded-xl !shadow-lg"
          nodeColor={(node) => {
            switch (node.type) {
              case "host":
              case "hostGroup":
                return "#22c55e";
              case "vm":
                return "#3b82f6";
              case "ct":
                return "#8b5cf6";
              case "storage":
                return "#f59e0b";
              case "network":
                return "#06b6d4";
              default:
                return "#6b7280";
            }
          }}
        />

        {/* Toolbar - Top Left */}
        <Panel position="top-left">
          <div className="flex flex-col gap-2">
            {/* View Mode Toggle */}
            <div className="glass rounded-xl p-1.5 shadow-lg flex gap-1">
              <button
                onClick={() => setViewMode("cluster")}
                className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                  viewMode === "cluster"
                    ? "bg-primary text-primary-foreground shadow-sm"
                    : "text-muted-foreground hover:text-foreground hover:bg-muted/50"
                }`}
              >
                <LayoutGrid className="h-3.5 w-3.5" />
                Cluster-Uebersicht
              </button>
              <button
                onClick={() => setViewMode("detailed")}
                className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                  viewMode === "detailed"
                    ? "bg-primary text-primary-foreground shadow-sm"
                    : "text-muted-foreground hover:text-foreground hover:bg-muted/50"
                }`}
              >
                <Maximize2 className="h-3.5 w-3.5" />
                Detailansicht
              </button>
              <button
                onClick={() => setViewMode("network")}
                className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                  viewMode === "network"
                    ? "bg-primary text-primary-foreground shadow-sm"
                    : "text-muted-foreground hover:text-foreground hover:bg-muted/50"
                }`}
              >
                <Network className="h-3.5 w-3.5" />
                Netzwerk-Ansicht
              </button>
            </div>

            {/* Search & Filter Bar */}
            <div className="glass rounded-xl p-2 shadow-lg flex items-center gap-2">
              <div className="relative flex-1 min-w-[200px]">
                <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
                <Input
                  placeholder="Suche nach Name..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-8 h-8 text-xs bg-transparent border-border/50"
                />
                {searchQuery && (
                  <button
                    onClick={() => setSearchQuery("")}
                    className="absolute right-2 top-1/2 -translate-y-1/2 p-0.5 rounded hover:bg-muted/80"
                  >
                    <X className="h-3 w-3 text-muted-foreground" />
                  </button>
                )}
              </div>

              <Button
                variant={showFilters || hasActiveFilters ? "default" : "outline"}
                size="sm"
                className="h-8 text-xs gap-1.5"
                onClick={() => setShowFilters(!showFilters)}
              >
                <Filter className="h-3.5 w-3.5" />
                Filter
                {hasActiveFilters && (
                  <span className="bg-primary-foreground/20 text-[10px] rounded-full px-1.5">!</span>
                )}
              </Button>

              <Button variant="outline" size="sm" className="h-8 w-8 p-0" onClick={loadTopology}>
                <RefreshCw className="h-3.5 w-3.5" />
              </Button>
            </div>

            {/* Filter Panel */}
            {showFilters && (
              <div className="glass rounded-xl p-3 shadow-lg space-y-3">
                <div>
                  <label className="text-[10px] uppercase tracking-wider text-muted-foreground font-semibold mb-1.5 block">Status</label>
                  <div className="flex gap-1">
                    {(["all", "running", "stopped"] as StatusFilter[]).map((s) => (
                      <button
                        key={s}
                        onClick={() => setStatusFilter(s)}
                        className={`px-2.5 py-1 rounded-md text-xs transition-all ${
                          statusFilter === s
                            ? "bg-primary text-primary-foreground"
                            : "bg-muted/50 text-muted-foreground hover:bg-muted"
                        }`}
                      >
                        {s === "all" ? "Alle" : s === "running" ? "Aktiv" : "Gestoppt"}
                      </button>
                    ))}
                  </div>
                </div>

                <div>
                  <label className="text-[10px] uppercase tracking-wider text-muted-foreground font-semibold mb-1.5 block">Typ</label>
                  <div className="flex gap-1">
                    {(["all", "vm", "ct"] as TypeFilter[]).map((t) => (
                      <button
                        key={t}
                        onClick={() => setTypeFilter(t)}
                        className={`px-2.5 py-1 rounded-md text-xs transition-all ${
                          typeFilter === t
                            ? "bg-primary text-primary-foreground"
                            : "bg-muted/50 text-muted-foreground hover:bg-muted"
                        }`}
                      >
                        {t === "all" ? "Alle" : t === "vm" ? "VM" : "CT"}
                      </button>
                    ))}
                  </div>
                </div>

                <div>
                  <label className="text-[10px] uppercase tracking-wider text-muted-foreground font-semibold mb-1.5 block">Tag</label>
                  <Input
                    placeholder="Tag-Filter..."
                    value={tagFilter}
                    onChange={(e) => setTagFilter(e.target.value)}
                    className="h-7 text-xs bg-transparent border-border/50"
                  />
                </div>

                {hasActiveFilters && (
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-7 text-xs w-full"
                    onClick={() => {
                      setStatusFilter("all");
                      setTypeFilter("all");
                      setTagFilter("");
                    }}
                  >
                    <X className="h-3 w-3 mr-1" />
                    Filter zuruecksetzen
                  </Button>
                )}
              </div>
            )}
          </div>
        </Panel>

        {/* Stats Panel - Top Right */}
        <Panel position="top-right">
          <div className="glass rounded-xl p-3 shadow-lg space-y-2 min-w-[180px]">
            <h3 className="text-[10px] uppercase tracking-wider text-muted-foreground font-semibold">Infrastruktur</h3>
            <div className="grid grid-cols-2 gap-x-4 gap-y-1.5">
              <div className="flex items-center gap-1.5">
                <Server className="h-3 w-3 text-green-500" />
                <span className="text-xs text-muted-foreground">Hosts</span>
                <span className="text-xs font-semibold ml-auto">{stats.hosts}</span>
              </div>
              <div className="flex items-center gap-1.5">
                <Monitor className="h-3 w-3 text-blue-500" />
                <span className="text-xs text-muted-foreground">VMs</span>
                <span className="text-xs font-semibold ml-auto">{stats.vms}</span>
              </div>
              <div className="flex items-center gap-1.5">
                <Box className="h-3 w-3 text-purple-500" />
                <span className="text-xs text-muted-foreground">CTs</span>
                <span className="text-xs font-semibold ml-auto">{stats.cts}</span>
              </div>
              <div className="flex items-center gap-1.5">
                <HardDrive className="h-3 w-3 text-amber-500" />
                <span className="text-xs text-muted-foreground">Storage</span>
                <span className="text-xs font-semibold ml-auto">{stats.storages}</span>
              </div>
              <div className="flex items-center gap-1.5">
                <Network className="h-3 w-3 text-cyan-500" />
                <span className="text-xs text-muted-foreground">Netzwerk</span>
                <span className="text-xs font-semibold ml-auto">{stats.networks}</span>
              </div>
            </div>

            {stats.totalCpu > 0 && (
              <div className="pt-2 border-t border-border/30 space-y-1.5">
                <div className="flex items-center gap-1.5 text-xs">
                  <span className="text-muted-foreground w-8">CPU</span>
                  <div className="flex-1 h-1.5 bg-muted rounded-full overflow-hidden">
                    <div
                      className={`h-full rounded-full transition-all duration-500 ${
                        stats.totalCpu >= 90 ? "bg-red-500" : stats.totalCpu >= 75 ? "bg-amber-500" : "bg-blue-500"
                      }`}
                      style={{ width: `${Math.min(stats.totalCpu, 100)}%` }}
                    />
                  </div>
                  <span className="text-muted-foreground tabular-nums w-10 text-right">{stats.totalCpu.toFixed(1)}%</span>
                </div>
                <div className="flex items-center gap-1.5 text-xs">
                  <span className="text-muted-foreground w-8">RAM</span>
                  <div className="flex-1 h-1.5 bg-muted rounded-full overflow-hidden">
                    <div
                      className={`h-full rounded-full transition-all duration-500 ${
                        stats.totalRam >= 90 ? "bg-red-500" : stats.totalRam >= 75 ? "bg-amber-500" : "bg-purple-500"
                      }`}
                      style={{ width: `${Math.min(stats.totalRam, 100)}%` }}
                    />
                  </div>
                  <span className="text-muted-foreground tabular-nums w-10 text-right">{stats.totalRam.toFixed(1)}%</span>
                </div>
              </div>
            )}

            <div className="pt-2 border-t border-border/30">
              <div className="flex items-center gap-2 flex-wrap">
                <Badge variant="success" className="text-[10px] px-1.5 py-0 h-4">Aktiv</Badge>
                <Badge variant="warning" className="text-[10px] px-1.5 py-0 h-4">Warnung</Badge>
                <Badge variant="destructive" className="text-[10px] px-1.5 py-0 h-4">Kritisch</Badge>
                <Badge variant="secondary" className="text-[10px] px-1.5 py-0 h-4">Offline</Badge>
              </div>
            </div>
          </div>
        </Panel>
      </ReactFlow>
    </div>
  );
}
