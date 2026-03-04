import { create } from "zustand";
import type { UpdateCheck } from "@/types/api";
import { updateApi, toArray } from "@/lib/api";

interface UpdateState {
  checks: UpdateCheck[];
  nodeChecks: Record<string, UpdateCheck[]>;
  isLoading: boolean;
  error: string | null;
  fetchAll: () => Promise<void>;
  fetchByNode: (nodeId: string) => Promise<void>;
  triggerCheck: (nodeId: string) => Promise<void>;
}

export const useUpdateStore = create<UpdateState>()((set) => ({
  checks: [],
  nodeChecks: {},
  isLoading: false,
  error: null,

  fetchAll: async () => {
    set({ isLoading: true, error: null });
    try {
      const resp = await updateApi.listAll();
      set({ checks: toArray<UpdateCheck>(resp.data), isLoading: false });
    } catch {
      set({ error: "Updates konnten nicht geladen werden", isLoading: false });
    }
  },

  fetchByNode: async (nodeId: string) => {
    try {
      const resp = await updateApi.listByNode(nodeId);
      const data = toArray<UpdateCheck>(resp.data);
      set((state) => ({
        nodeChecks: { ...state.nodeChecks, [nodeId]: data },
      }));
    } catch {
      // nicht verfuegbar
    }
  },

  triggerCheck: async (nodeId: string) => {
    await updateApi.triggerCheck(nodeId);
  },
}));
