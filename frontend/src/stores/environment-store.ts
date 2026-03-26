import { create } from "zustand";
import { toast } from "sonner";
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

export const useEnvironmentStore = create<EnvironmentState>()((set, get) => ({
  environments: [],
  isLoading: false,
  error: null,

  fetchEnvironments: async () => {
    set({ isLoading: true, error: null });
    try {
      const resp = await environmentApi.list();
      set({ environments: toArray<Environment>(resp.data), isLoading: false });
    } catch {
      toast.error("Umgebungen konnten nicht geladen werden");
      set({ error: "Umgebungen konnten nicht geladen werden", isLoading: false });
    }
  },

  createEnvironment: async (data) => {
    try {
      const resp = await environmentApi.create(data);
      const env = resp.data;
      set((state) => ({ environments: [...state.environments, env] }));
      toast.success("Umgebung erstellt");
      // Refresh the list after creation
      get().fetchEnvironments();
    } catch {
      toast.error("Umgebung konnte nicht erstellt werden");
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
      toast.success("Umgebung aktualisiert");
      // Refresh the list after update
      get().fetchEnvironments();
    } catch {
      toast.error("Umgebung konnte nicht aktualisiert werden");
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
      toast.success("Umgebung gelöscht");
      // Refresh the list after deletion
      get().fetchEnvironments();
    } catch {
      toast.error("Umgebung konnte nicht gelöscht werden");
      set({ error: "Umgebung konnte nicht gelöscht werden" });
      throw new Error("Umgebung konnte nicht gelöscht werden");
    }
  },

  assignNode: async (nodeId, environmentId) => {
    try {
      await environmentApi.assignNode(nodeId, environmentId);
      toast.success("Node zugewiesen");
    } catch {
      toast.error("Node konnte nicht zugewiesen werden");
      set({ error: "Node konnte nicht zugewiesen werden" });
      throw new Error("Node konnte nicht zugewiesen werden");
    }
  },
}));
