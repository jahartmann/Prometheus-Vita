import { create } from "zustand";
import type { Environment } from "@/types/api";
import { environmentApi, toArray } from "@/lib/api";

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
      set({ environments: toArray<Environment>(resp.data), isLoading: false });
    } catch {
      set({ error: "Umgebungen konnten nicht geladen werden", isLoading: false });
    }
  },

  createEnvironment: async (data) => {
    try {
      const resp = await environmentApi.create(data);
      const env = resp.data;
      set((state) => ({ environments: [...state.environments, env] }));
    } catch {
      set({ error: "Umgebung konnte nicht erstellt werden" });
      throw new Error("Umgebung konnte nicht erstellt werden");
    }
  },

  updateEnvironment: async (id, data) => {
    try {
      const resp = await environmentApi.update(id, data);
      const updated = resp.data;
      set((state) => ({
        environments: state.environments.map((e) => (e.id === id ? updated : e)),
      }));
    } catch {
      set({ error: "Umgebung konnte nicht aktualisiert werden" });
      throw new Error("Umgebung konnte nicht aktualisiert werden");
    }
  },

  deleteEnvironment: async (id) => {
    try {
      await environmentApi.delete(id);
      set((state) => ({
        environments: state.environments.filter((e) => e.id !== id),
      }));
    } catch {
      set({ error: "Umgebung konnte nicht geloescht werden" });
      throw new Error("Umgebung konnte nicht geloescht werden");
    }
  },

  assignNode: async (nodeId, environmentId) => {
    try {
      await environmentApi.assignNode(nodeId, environmentId);
    } catch {
      set({ error: "Node konnte nicht zugewiesen werden" });
      throw new Error("Node konnte nicht zugewiesen werden");
    }
  },
}));
