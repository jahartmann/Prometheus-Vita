import { create } from "zustand";
import type { ReflexRule } from "@/types/api";
import { reflexApi, toArray } from "@/lib/api";

interface ReflexState {
  rules: ReflexRule[];
  isLoading: boolean;
  error: string | null;
  fetchRules: () => Promise<void>;
  toggleRule: (id: string, isActive: boolean) => Promise<void>;
  deleteRule: (id: string) => Promise<void>;
}

export const useReflexStore = create<ReflexState>()((set, get) => ({
  rules: [],
  isLoading: false,
  error: null,

  fetchRules: async () => {
    set({ isLoading: true, error: null });
    try {
      const resp = await reflexApi.list();
      set({ rules: toArray<ReflexRule>(resp.data), isLoading: false });
    } catch {
      set({ error: "Reflex-Regeln konnten nicht geladen werden", isLoading: false });
    }
  },

  toggleRule: async (id: string, isActive: boolean) => {
    try {
      await reflexApi.update(id, { is_active: isActive });
      set({
        rules: get().rules.map((r) =>
          r.id === id ? { ...r, is_active: isActive } : r
        ),
      });
    } catch {
      set({ error: "Status konnte nicht geändert werden" });
    }
  },

  deleteRule: async (id: string) => {
    try {
      await reflexApi.delete(id);
      set({ rules: get().rules.filter((r) => r.id !== id) });
    } catch {
      set({ error: "Regel konnte nicht gelöscht werden" });
    }
  },
}));
