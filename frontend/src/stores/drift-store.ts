import { create } from "zustand";
import { toast } from "sonner";
import type { DriftCheck, CompareNodesResponse } from "@/types/api";
import { driftApi, toArray } from "@/lib/api";

interface DriftState {
  checks: DriftCheck[];
  nodeChecks: Record<string, DriftCheck[]>;
  comparisonResult: CompareNodesResponse | null;
  isLoading: boolean;
  isComparing: boolean;
  error: string | null;
  fetchAll: () => Promise<void>;
  fetchByNode: (nodeId: string) => Promise<void>;
  triggerCheck: (nodeId: string) => Promise<void>;
  acceptBaseline: (checkId: string) => Promise<void>;
  ignoreDrift: (checkId: string, filePath: string) => Promise<void>;
  compareNodes: (filePaths: string[], nodeIds: string[]) => Promise<void>;
}

export const useDriftStore = create<DriftState>()((set) => ({
  checks: [],
  nodeChecks: {},
  comparisonResult: null,
  isLoading: false,
  isComparing: false,
  error: null,

  fetchAll: async () => {
    set({ isLoading: true, error: null });
    try {
      const resp = await driftApi.listAll();
      set({ checks: toArray<DriftCheck>(resp.data), isLoading: false });
    } catch {
      toast.error("Drift-Checks konnten nicht geladen werden");
      set({ error: "Drift-Checks konnten nicht geladen werden", isLoading: false });
    }
  },

  fetchByNode: async (nodeId: string) => {
    try {
      const resp = await driftApi.listByNode(nodeId);
      const data = toArray<DriftCheck>(resp.data);
      set((state) => ({
        nodeChecks: { ...state.nodeChecks, [nodeId]: data },
      }));
    } catch {
      // nicht verfuegbar
    }
  },

  triggerCheck: async (nodeId: string) => {
    await driftApi.triggerCheck(nodeId);
    toast.success("Drift-Check gestartet");
  },

  acceptBaseline: async (checkId: string) => {
    try {
      await driftApi.acceptBaseline(checkId);
      toast.success("Baseline aktualisiert");
      // Refresh the list
      const resp = await driftApi.listAll();
      set({ checks: toArray<DriftCheck>(resp.data) });
    } catch {
      toast.error("Baseline konnte nicht aktualisiert werden");
    }
  },

  ignoreDrift: async (checkId: string, filePath: string) => {
    try {
      await driftApi.ignoreDrift(checkId, filePath);
      toast.success("Aenderung ignoriert");
      // Refresh the list
      const resp = await driftApi.listAll();
      set({ checks: toArray<DriftCheck>(resp.data) });
    } catch {
      toast.error("Aenderung konnte nicht ignoriert werden");
    }
  },

  compareNodes: async (filePaths: string[], nodeIds: string[]) => {
    set({ isComparing: true });
    try {
      const resp = await driftApi.compareNodes({ file_paths: filePaths, node_ids: nodeIds });
      const data = resp.data as CompareNodesResponse;
      set({ comparisonResult: data, isComparing: false });
    } catch {
      toast.error("Node-Vergleich fehlgeschlagen");
      set({ isComparing: false });
    }
  },
}));
