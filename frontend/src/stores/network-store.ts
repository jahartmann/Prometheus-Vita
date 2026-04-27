import { create } from "zustand";
import { getApiErrorMessage, networkApi, nodeApi } from "@/lib/api";
import type { ToolPreflightResult } from "@/types/api";

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
  toolPreflightByNode: Record<string, ToolPreflightResult | undefined>;
  errorsByScope: Record<string, string | undefined>;
  activeTab: "ports" | "devices" | "anomalies" | "services" | "history";
  scanStatus: { lastQuick?: string; lastFull?: string; isScanning: boolean; scanningNodeId?: string };

  setActiveTab: (tab: NetworkState["activeTab"]) => void;
  fetchScans: (nodeId: string) => Promise<void>;
  fetchDevices: (nodeId: string) => Promise<void>;
  fetchAnomalies: (nodeId: string) => Promise<void>;
  fetchBaselines: (nodeId: string) => Promise<void>;
  fetchToolPreflight: (nodeId: string) => Promise<void>;
  triggerScan: (nodeId: string, scanType: "quick" | "full") => Promise<void>;
  acknowledgeAnomaly: (id: string) => Promise<void>;
  activateBaseline: (id: string) => Promise<void>;
}

let scansRequestSeq = 0;
let devicesRequestSeq = 0;
let anomaliesRequestSeq = 0;
let baselinesRequestSeq = 0;
let toolsRequestSeq = 0;

export const useNetworkStore = create<NetworkState>()((set) => ({
  scans: [],
  devices: [],
  anomalies: [],
  baselines: [],
  toolPreflightByNode: {},
  errorsByScope: {},
  activeTab: "ports",
  scanStatus: { isScanning: false },

  setActiveTab: (tab) => set({ activeTab: tab }),

  fetchScans: async (nodeId) => {
    const requestSeq = ++scansRequestSeq;
    try {
      const res = await networkApi.getScans(nodeId, { limit: 50 });
      if (requestSeq !== scansRequestSeq) return;
      const scans = Array.isArray(res.data) ? res.data : [];
      const lastQuick = scans.find((s: NetworkScan) => s.scan_type === "quick")?.started_at;
      const lastFull = scans.find((s: NetworkScan) => s.scan_type === "full")?.started_at;
      set((state) => ({
        scans,
        errorsByScope: { ...state.errorsByScope, scans: undefined },
        scanStatus: {
          lastQuick,
          lastFull,
          isScanning: state.scanStatus.scanningNodeId === nodeId ? false : state.scanStatus.isScanning,
          scanningNodeId: state.scanStatus.scanningNodeId === nodeId ? undefined : state.scanStatus.scanningNodeId,
        },
      }));
    } catch (e) {
      console.error('Failed to fetch scans:', e);
      if (requestSeq === scansRequestSeq) {
        set((state) => ({
          scans: [],
          errorsByScope: {
            ...state.errorsByScope,
            scans: getApiErrorMessage(e, "Scans konnten nicht geladen werden"),
          },
          scanStatus: {
            isScanning: state.scanStatus.scanningNodeId === nodeId ? false : state.scanStatus.isScanning,
            scanningNodeId: state.scanStatus.scanningNodeId === nodeId ? undefined : state.scanStatus.scanningNodeId,
          },
        }));
      }
    }
  },

  fetchDevices: async (nodeId) => {
    const requestSeq = ++devicesRequestSeq;
    try {
      const res = await networkApi.getDevices(nodeId);
      if (requestSeq !== devicesRequestSeq) return;
      set((state) => ({
        devices: Array.isArray(res.data) ? res.data : [],
        errorsByScope: { ...state.errorsByScope, devices: undefined },
      }));
    } catch (e) {
      console.error('Failed to fetch devices:', e);
      if (requestSeq === devicesRequestSeq) {
        set((state) => ({
          devices: [],
          errorsByScope: {
            ...state.errorsByScope,
            devices: getApiErrorMessage(e, "Netzwerk-Geräte konnten nicht geladen werden"),
          },
        }));
      }
    }
  },

  fetchAnomalies: async (nodeId) => {
    const requestSeq = ++anomaliesRequestSeq;
    try {
      const res = await networkApi.getAnomalies(nodeId, { limit: 100 });
      if (requestSeq !== anomaliesRequestSeq) return;
      set((state) => ({
        anomalies: Array.isArray(res.data) ? res.data : [],
        errorsByScope: { ...state.errorsByScope, anomalies: undefined },
      }));
    } catch (e) {
      console.error('Failed to fetch network anomalies:', e);
      if (requestSeq === anomaliesRequestSeq) {
        set((state) => ({
          anomalies: [],
          errorsByScope: {
            ...state.errorsByScope,
            anomalies: getApiErrorMessage(e, "Netzwerk-Anomalien konnten nicht geladen werden"),
          },
        }));
      }
    }
  },

  fetchBaselines: async (nodeId) => {
    const requestSeq = ++baselinesRequestSeq;
    try {
      const res = await networkApi.getBaselines(nodeId);
      if (requestSeq !== baselinesRequestSeq) return;
      set((state) => ({
        baselines: Array.isArray(res.data) ? res.data : [],
        errorsByScope: { ...state.errorsByScope, baselines: undefined },
      }));
    } catch (e) {
      console.error('Failed to fetch baselines:', e);
      if (requestSeq === baselinesRequestSeq) {
        set((state) => ({
          baselines: [],
          errorsByScope: {
            ...state.errorsByScope,
            baselines: getApiErrorMessage(e, "Scan-Baselines konnten nicht geladen werden"),
          },
        }));
      }
    }
  },

  fetchToolPreflight: async (nodeId) => {
    const requestSeq = ++toolsRequestSeq;
    try {
      const res = await nodeApi.getToolPreflight(nodeId);
      if (requestSeq !== toolsRequestSeq) return;
      const preflight = res.data as ToolPreflightResult;
      set((state) => ({
        toolPreflightByNode: {
          ...state.toolPreflightByNode,
          [nodeId]: preflight,
        },
        errorsByScope: { ...state.errorsByScope, tools: undefined },
      }));
    } catch (e) {
      console.error('Failed to fetch tool preflight:', e);
      if (requestSeq === toolsRequestSeq) {
        set((state) => ({
          toolPreflightByNode: {
            ...state.toolPreflightByNode,
            [nodeId]: undefined,
          },
          errorsByScope: {
            ...state.errorsByScope,
            tools: getApiErrorMessage(e, "Tool-Preflight konnte nicht geladen werden"),
          },
        }));
      }
    }
  },

  triggerScan: async (nodeId, scanType) => {
    set((state) => ({
      errorsByScope: { ...state.errorsByScope, trigger: undefined },
      scanStatus: { ...state.scanStatus, isScanning: true, scanningNodeId: nodeId },
    }));
    try {
      await networkApi.triggerScan(nodeId, { scan_type: scanType });
    } catch (e) {
      console.error('Failed to trigger scan:', e);
      set((state) => {
        if (state.scanStatus.scanningNodeId !== nodeId) return {};
        return {
          errorsByScope: {
            ...state.errorsByScope,
            trigger: getApiErrorMessage(e, "Scan konnte nicht gestartet werden"),
          },
          scanStatus: { ...state.scanStatus, isScanning: false, scanningNodeId: undefined },
        };
      });
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
      set((state) => ({
        errorsByScope: {
          ...state.errorsByScope,
          anomalies: getApiErrorMessage(e, "Netzwerk-Anomalie konnte nicht bestätigt werden"),
        },
      }));
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
      set((state) => ({
        errorsByScope: {
          ...state.errorsByScope,
          baselines: getApiErrorMessage(e, "Scan-Baseline konnte nicht aktiviert werden"),
        },
      }));
    }
  },
}));
