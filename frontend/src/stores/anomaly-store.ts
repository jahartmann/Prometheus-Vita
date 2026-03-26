import { create } from "zustand";
import { toast } from "sonner";
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
      toast.error("Anomalien konnten nicht geladen werden");
      set({ error: "Anomalien konnten nicht geladen werden", isLoading: false });
    }
  },

  fetchByNode: async (nodeId: string) => {
    set({ isLoading: true, error: null });
    try {
      const data = await anomalyApi.listByNode(nodeId);
      set({ nodeAnomalies: toArray<AnomalyRecord>(data), isLoading: false });
    } catch {
      toast.error("Node-Anomalien konnten nicht geladen werden");
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
      toast.success("Anomalie aufgelöst");
    } catch {
      toast.error("Anomalie konnte nicht aufgelöst werden");
      set({ error: "Anomalie konnte nicht aufgelöst werden" });
    }
  },
}));
