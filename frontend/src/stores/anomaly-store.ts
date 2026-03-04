import { create } from "zustand";
import type { AnomalyRecord } from "@/types/api";
import { anomalyApi, toArray } from "@/lib/api";

interface AnomalyState {
  anomalies: AnomalyRecord[];
  nodeAnomalies: AnomalyRecord[];
  isLoading: boolean;
  error: string | null;

  fetchUnresolved: () => Promise<void>;
  fetchByNode: (nodeId: string) => Promise<void>;
  resolve: (id: string) => Promise<void>;
}

export const useAnomalyStore = create<AnomalyState>()((set, get) => ({
  anomalies: [],
  nodeAnomalies: [],
  isLoading: false,
  error: null,

  fetchUnresolved: async () => {
    set({ isLoading: true, error: null });
    try {
      const data = await anomalyApi.listUnresolved();
      set({ anomalies: toArray<AnomalyRecord>(data), isLoading: false });
    } catch {
      set({ error: "Anomalien konnten nicht geladen werden", isLoading: false });
    }
  },

  fetchByNode: async (nodeId: string) => {
    set({ isLoading: true, error: null });
    try {
      const data = await anomalyApi.listByNode(nodeId);
      set({ nodeAnomalies: toArray<AnomalyRecord>(data), isLoading: false });
    } catch {
      set({ error: "Node-Anomalien konnten nicht geladen werden", isLoading: false });
    }
  },

  resolve: async (id: string) => {
    try {
      await anomalyApi.resolve(id);
      set((s) => ({
        anomalies: s.anomalies.filter((a) => a.id !== id),
        nodeAnomalies: s.nodeAnomalies.filter((a) => a.id !== id),
      }));
    } catch {
      set({ error: "Anomalie konnte nicht aufgeloest werden" });
    }
  },
}));
