import { create } from "zustand";
import type { Environment } from "@/types/api";
import { environmentApi } from "@/lib/api";

interface EnvironmentState {
  environments: Environment[];
  isLoading: boolean;
  error: string | null;
  fetchEnvironments: () => Promise<void>;
  createEnvironment: (data: { name: string; description?: string; color?: string }) => Promise<void>;
  updateEnvironment: (id: string, data: { name?: string; description?: string; color?: string }) => Promise<void>;
  deleteEnvironment: (id: string) => Promise<void>;
  assignNode: (nodeId: string, environmentId: string) => Promise<void>;
}

export const useEnvironmentStore = create<EnvironmentState>()((set) => ({
  environments: [],
  isLoading: false,
  error: null,

  fetchEnvironments: async () => {
    set({ isLoading: true, error: null });
    try {
      const resp = await environmentApi.list();
      set({ environments: resp.data?.data || resp.data || [], isLoading: false });
    } catch {
      set({ error: "Umgebungen konnten nicht geladen werden", isLoading: false });
    }
  },

  createEnvironment: async (data) => {
    const resp = await environmentApi.create(data);
    const env = resp.data?.data || resp.data;
    set((state) => ({ environments: [...state.environments, env] }));
  },

  updateEnvironment: async (id, data) => {
    const resp = await environmentApi.update(id, data);
    const updated = resp.data?.data || resp.data;
    set((state) => ({
      environments: state.environments.map((e) => (e.id === id ? updated : e)),
    }));
  },

  deleteEnvironment: async (id) => {
    await environmentApi.delete(id);
    set((state) => ({
      environments: state.environments.filter((e) => e.id !== id),
    }));
  },

  assignNode: async (nodeId, environmentId) => {
    await environmentApi.assignNode(nodeId, environmentId);
  },
}));
