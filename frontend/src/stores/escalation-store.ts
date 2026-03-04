import { create } from "zustand";
import type { EscalationPolicy, AlertIncident } from "@/types/api";
import { escalationApi } from "@/lib/api";

interface EscalationState {
  policies: EscalationPolicy[];
  incidents: AlertIncident[];
  isLoading: boolean;
  error: string | null;

  fetchPolicies: () => Promise<void>;
  fetchIncidents: (limit?: number, offset?: number) => Promise<void>;
}

export const useEscalationStore = create<EscalationState>()((set) => ({
  policies: [],
  incidents: [],
  isLoading: false,
  error: null,

  fetchPolicies: async () => {
    set({ isLoading: true, error: null });
    try {
      const response = await escalationApi.listPolicies();
      set({ policies: response.data?.data || response.data || [], isLoading: false });
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : "Eskalationsrichtlinien konnten nicht geladen werden";
      set({ error: message, isLoading: false });
    }
  },

  fetchIncidents: async (limit = 50, offset = 0) => {
    set({ isLoading: true, error: null });
    try {
      const response = await escalationApi.listIncidents(limit, offset);
      set({ incidents: response.data?.data || response.data || [], isLoading: false });
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : "Vorfaelle konnten nicht geladen werden";
      set({ error: message, isLoading: false });
    }
  },
}));
