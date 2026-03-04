import { create } from "zustand";
import type { TopologyGraph } from "@/types/api";
import { topologyApi } from "@/lib/api";

interface TopologyState {
  graph: TopologyGraph | null;
  isLoading: boolean;
  error: string | null;
  fetchTopology: () => Promise<void>;
}

export const useTopologyStore = create<TopologyState>()((set) => ({
  graph: null,
  isLoading: false,
  error: null,

  fetchTopology: async () => {
    set({ isLoading: true, error: null });
    try {
      const resp = await topologyApi.get();
      set({ graph: resp.data?.data || resp.data || null, isLoading: false });
    } catch {
      set({ error: "Topologie konnte nicht geladen werden", isLoading: false });
    }
  },
}));
