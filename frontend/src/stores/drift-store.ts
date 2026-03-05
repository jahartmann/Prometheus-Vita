import { create } from "zustand";
import { toast } from "sonner";
import type { DriftCheck } from "@/types/api";
import { driftApi, toArray } from "@/lib/api";

interface DriftState {
  checks: DriftCheck[];
  nodeChecks: Record<string, DriftCheck[]>;
  isLoading: boolean;
  error: string | null;
  fetchAll: () => Promise<void>;
  fetchByNode: (nodeId: string) => Promise<void>;
  triggerCheck: (nodeId: string) => Promise<void>;
}

export const useDriftStore = create<DriftState>()((set) => ({
  checks: [],
  nodeChecks: {},
  isLoading: false,
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
}));
