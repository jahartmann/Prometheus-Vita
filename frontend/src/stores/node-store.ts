import { create } from "zustand";
import type { Node, NodeStatus, VM } from "@/types/api";
import api, { toArray } from "@/lib/api";

interface NodeState {
  nodes: Node[];
  selectedNode: Node | null;
  nodeStatus: Record<string, NodeStatus>;
  nodeVMs: Record<string, VM[]>;
  isLoading: boolean;
  error: string | null;
  fetchNodes: () => Promise<void>;
  fetchNodeStatus: (nodeId: string) => Promise<void>;
  fetchNodeVMs: (nodeId: string) => Promise<void>;
  selectNode: (node: Node | null) => void;
  addNode: (node: Node) => void;
  removeNode: (nodeId: string) => void;
}

export const useNodeStore = create<NodeState>()((set, get) => ({
  nodes: [],
  selectedNode: null,
  nodeStatus: {},
  nodeVMs: {},
  isLoading: false,
  error: null,

  fetchNodes: async () => {
    set({ isLoading: true, error: null });
    try {
      const response = await api.get<Node[]>("/nodes");
      set({ nodes: toArray<Node>(response.data), isLoading: false });
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : "Nodes konnten nicht geladen werden";
      set({ error: message, isLoading: false });
    }
  },

  fetchNodeStatus: async (nodeId: string) => {
    try {
      const response = await api.get<NodeStatus>(`/nodes/${nodeId}/status`);
      set((state) => ({
        nodeStatus: { ...state.nodeStatus, [nodeId]: response.data },
      }));
    } catch {
      // Status nicht verfuegbar
    }
  },

  fetchNodeVMs: async (nodeId: string) => {
    try {
      const response = await api.get<VM[]>(`/nodes/${nodeId}/vms`);
      set((state) => ({
        nodeVMs: { ...state.nodeVMs, [nodeId]: toArray<VM>(response.data) },
      }));
    } catch {
      // VMs nicht verfuegbar
    }
  },

  selectNode: (node: Node | null) => {
    set({ selectedNode: node });
  },

  addNode: (node: Node) => {
    set((state) => ({ nodes: [...state.nodes, node] }));
  },

  removeNode: (nodeId: string) => {
    set((state) => ({
      nodes: state.nodes.filter((n) => n.id !== nodeId),
      selectedNode:
        get().selectedNode?.id === nodeId ? null : get().selectedNode,
    }));
  },
}));
