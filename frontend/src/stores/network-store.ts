import { create } from "zustand";
import { networkApi } from "@/lib/api";

interface NetworkScan {
  id: string;
  node_id: string;
  scan_type: string;
  results_json: unknown;
  started_at: string;
  completed_at?: string;
}

interface NetworkDevice {
  id: string;
  node_id: string;
  ip: string;
  mac?: string;
  hostname?: string;
  first_seen: string;
  last_seen: string;
  is_known: boolean;
}

interface NetworkAnomaly {
  id: string;
  node_id: string;
  anomaly_type: string;
  risk_score: number;
  details_json: unknown;
  scan_id?: string;
  is_acknowledged: boolean;
  created_at: string;
}

interface ScanBaseline {
  id: string;
  node_id: string;
  label?: string;
  is_active: boolean;
  baseline_json: unknown;
  whitelist_json?: unknown;
  created_at: string;
}

interface NetworkState {
  scans: NetworkScan[];
  devices: NetworkDevice[];
  anomalies: NetworkAnomaly[];
  baselines: ScanBaseline[];
  activeTab: "ports" | "devices" | "anomalies" | "history";
  scanStatus: { lastQuick?: string; lastFull?: string; isScanning: boolean };

  setActiveTab: (tab: NetworkState["activeTab"]) => void;
  fetchScans: (nodeId: string) => Promise<void>;
  fetchDevices: (nodeId: string) => Promise<void>;
  fetchAnomalies: (nodeId: string) => Promise<void>;
  fetchBaselines: (nodeId: string) => Promise<void>;
  triggerScan: (nodeId: string, scanType: "quick" | "full") => Promise<void>;
  acknowledgeAnomaly: (id: string) => Promise<void>;
  activateBaseline: (id: string) => Promise<void>;
}

export const useNetworkStore = create<NetworkState>()((set) => ({
  scans: [],
  devices: [],
  anomalies: [],
  baselines: [],
  activeTab: "ports",
  scanStatus: { isScanning: false },

  setActiveTab: (tab) => set({ activeTab: tab }),

  fetchScans: async (nodeId) => {
    try {
      const res = await networkApi.getScans(nodeId, { limit: 50 });
      const scans = Array.isArray(res.data) ? res.data : [];
      const lastQuick = scans.find((s: NetworkScan) => s.scan_type === "quick")?.started_at;
      const lastFull = scans.find((s: NetworkScan) => s.scan_type === "full")?.started_at;
      set({ scans, scanStatus: { lastQuick, lastFull, isScanning: false } });
    } catch (e) {
      console.error('Failed to fetch scans:', e);
      set({ scans: [], scanStatus: { isScanning: false } });
    }
  },

  fetchDevices: async (nodeId) => {
    try {
      const res = await networkApi.getDevices(nodeId);
      set({ devices: Array.isArray(res.data) ? res.data : [] });
    } catch (e) {
      console.error('Failed to fetch devices:', e);
      set({ devices: [] });
    }
  },

  fetchAnomalies: async (nodeId) => {
    try {
      const res = await networkApi.getAnomalies(nodeId, { limit: 100 });
      set({ anomalies: Array.isArray(res.data) ? res.data : [] });
    } catch (e) {
      console.error('Failed to fetch network anomalies:', e);
      set({ anomalies: [] });
    }
  },

  fetchBaselines: async (nodeId) => {
    try {
      const res = await networkApi.getBaselines(nodeId);
      set({ baselines: Array.isArray(res.data) ? res.data : [] });
    } catch (e) {
      console.error('Failed to fetch baselines:', e);
      set({ baselines: [] });
    }
  },

  triggerScan: async (nodeId, scanType) => {
    set((state) => ({ scanStatus: { ...state.scanStatus, isScanning: true } }));
    try {
      await networkApi.triggerScan(nodeId, { scan_type: scanType });
    } catch (e) {
      console.error('Failed to trigger scan:', e);
    }
    // Status will update via polling
  },

  acknowledgeAnomaly: async (id) => {
    try {
      await networkApi.acknowledgeAnomaly(id);
      set((state) => ({
        anomalies: state.anomalies.map((a) =>
          a.id === id ? { ...a, is_acknowledged: true } : a
        ),
      }));
    } catch (e) {
      console.error('Failed to acknowledge network anomaly:', e);
    }
  },

  activateBaseline: async (id) => {
    try {
      await networkApi.activateBaseline(id);
      set((state) => ({
        baselines: state.baselines.map((b) => ({
          ...b,
          is_active: b.id === id,
        })),
      }));
    } catch (e) {
      console.error('Failed to activate baseline:', e);
    }
  },
}));
