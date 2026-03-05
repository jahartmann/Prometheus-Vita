"use client";

import { create } from "zustand";
import { toast } from "sonner";
import type { NodeProfile, DRReadinessScore, RecoveryRunbook, DRSimulationResult } from "@/types/api";
import { drApi, toArray } from "@/lib/api";

interface DRState {
  profile: NodeProfile | null;
  scores: DRReadinessScore[];
  currentScore: DRReadinessScore | null;
  runbooks: RecoveryRunbook[];
  simulationResult: DRSimulationResult | null;
  isLoading: boolean;
  error: string | null;
  fetchProfile: (nodeId: string) => Promise<void>;
  collectProfile: (nodeId: string) => Promise<NodeProfile>;
  fetchReadiness: (nodeId: string) => Promise<void>;
  calculateReadiness: (nodeId: string) => Promise<DRReadinessScore>;
  fetchAllScores: () => Promise<void>;
  fetchRunbooks: (nodeId: string) => Promise<void>;
  generateRunbook: (nodeId: string, scenario: string) => Promise<RecoveryRunbook>;
  deleteRunbook: (runbookId: string) => Promise<void>;
  simulate: (nodeId: string, scenario: string) => Promise<DRSimulationResult>;
}

export const useDRStore = create<DRState>()((set) => ({
  profile: null,
  scores: [],
  currentScore: null,
  runbooks: [],
  simulationResult: null,
  isLoading: false,
  error: null,

  fetchProfile: async (nodeId: string) => {
    set({ isLoading: true, error: null });
    try {
      const response = await drApi.getProfile(nodeId);
      set({ profile: response.data, isLoading: false });
    } catch {
      set({ profile: null, error: null, isLoading: false });
    }
  },

  collectProfile: async (nodeId: string) => {
    const response = await drApi.collectProfile(nodeId);
    set({ profile: response.data });
    toast.success("Profil erfolgreich erfasst");
    return response.data;
  },

  fetchReadiness: async (nodeId: string) => {
    set({ isLoading: true, error: null });
    try {
      const response = await drApi.getReadiness(nodeId);
      set({ currentScore: response.data, isLoading: false });
    } catch {
      set({ currentScore: null, error: null, isLoading: false });
    }
  },

  calculateReadiness: async (nodeId: string) => {
    const response = await drApi.calculateReadiness(nodeId);
    set({ currentScore: response.data });
    return response.data;
  },

  fetchAllScores: async () => {
    set({ isLoading: true, error: null });
    try {
      const response = await drApi.listAllScores();
      set({ scores: toArray<DRReadinessScore>(response.data), isLoading: false });
    } catch {
      toast.error("DR Scores konnten nicht geladen werden");
      set({ error: "DR Scores konnten nicht geladen werden", isLoading: false });
    }
  },

  fetchRunbooks: async (nodeId: string) => {
    set({ isLoading: true, error: null });
    try {
      const response = await drApi.listRunbooks(nodeId);
      set({ runbooks: toArray<RecoveryRunbook>(response.data), isLoading: false });
    } catch {
      toast.error("Runbooks konnten nicht geladen werden");
      set({ error: "Runbooks konnten nicht geladen werden", isLoading: false });
    }
  },

  generateRunbook: async (nodeId: string, scenario: string) => {
    const response = await drApi.generateRunbook(nodeId, scenario);
    set((state) => ({ runbooks: [response.data, ...state.runbooks] }));
    toast.success("Runbook generiert");
    return response.data;
  },

  deleteRunbook: async (runbookId: string) => {
    await drApi.deleteRunbook(runbookId);
    set((state) => ({
      runbooks: state.runbooks.filter((r) => r.id !== runbookId),
    }));
    toast.success("Runbook geloescht");
  },

  simulate: async (nodeId: string, scenario: string) => {
    const response = await drApi.simulate(nodeId, scenario);
    set({ simulationResult: response.data });
    return response.data;
  },
}));
