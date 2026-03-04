import { create } from "zustand";
import type { Node, NodeStatus, VM } from "@/types/api";
import api, { toArray } from "@/lib/api";
import { useAuthStore } from "@/stores/auth-store";

interface NodeState {
  nodes: Node[];
  selectedNode: Node | null;
  nodeStatus: Record<string, NodeStatus>;
  nodeVMs: Record<string, VM[]>;
  nodeErrors: Record<string, string>;
  isLoading: boolean;
  error: string | null;
  fetchNodes: () => Promise<void>;
  fetchNodeStatus: (nodeId: string) => Promise<void>;
  fetchNodeVMs: (nodeId: string) => Promise<void>;
  selectNode: (node: Node | null) => void;
  addNode: (node: Node) => void;
  removeNode: (nodeId: string) => void;
}

export const useNodeStore = create<NodeState>()((set) => ({
  nodes: [],
  selectedNode: null,
  nodeStatus: {},
  nodeVMs: {},
  nodeErrors: {},
  isLoading: false,
  error: null,

  fetchNodes: async () => {
    if (!useAuthStore.getState().accessToken) return;
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
    if (!useAuthStore.getState().accessToken) return;
    try {
      const response = await api.get<NodeStatus>(`/nodes/${nodeId}/status`);
      set((state) => ({
        nodeStatus: { ...state.nodeStatus, [nodeId]: response.data },
        nodeErrors: { ...state.nodeErrors, [nodeId]: "" },
      }));
    } catch (err: unknown) {
      const status = (err as { response?: { status?: number } })?.response?.status;
      const msg =
        status === 503
          ? "Node nicht erreichbar"
          : status === 401
          ? ""
          : "Statusabfrage fehlgeschlagen";
      if (msg) {
        set((state) => ({ nodeErrors: { ...state.nodeErrors, [nodeId]: msg } }));
      }
    }
  },

  fetchNodeVMs: async (nodeId: string) => {
    if (!useAuthStore.getState().accessToken) return;
    try {
      const response = await api.get<VM[]>(`/nodes/${nodeId}/vms`);
      set((state) => ({
        nodeVMs: { ...state.nodeVMs, [nodeId]: toArray<VM>(response.data) },
        nodeErrors: { ...state.nodeErrors, [nodeId]: "" },
      }));
    } catch (err: unknown) {
      const status = (err as { response?: { status?: number } })?.response?.status;
      const msg =
        status === 503
          ? "Node nicht erreichbar"
          : status === 401
          ? ""
          : "VM-Abfrage fehlgeschlagen";
      if (msg) {
        set((state) => ({ nodeErrors: { ...state.nodeErrors, [nodeId]: msg } }));
      }
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
      selectedNode: state.selectedNode?.id === nodeId ? null : state.selectedNode,
    }));
  },
}));
