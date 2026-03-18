import { create } from "zustand";
import { logAnalysisApi } from "@/lib/api";

interface LogEntry {
  id: string;
  timestamp: string;
  node_id: string;
  source: string;
  severity: string;
  process: string;
  pid: number;
  message: string;
  raw: string;
  assessment?: {
    severity: string;
    anomaly_score: number;
    category: string;
    summary: string;
  };
}

interface LogAnomaly {
  id: string;
  node_id: string;
  timestamp: string;
  source: string;
  severity: string;
  anomaly_score: number;
  category: string;
  summary: string;
  raw_log: string;
  is_acknowledged: boolean;
  created_at: string;
}

interface LogSource {
  id: string;
  node_id: string;
  path: string;
  enabled: boolean;
  is_builtin: boolean;
  parser_type: string;
}

interface LogKpis {
  errorsPerMin: number;
  warningsPerMin: number;
  activeAnomalies: number;
  throughput: number;
}

interface LogAnalysisReport {
  summary: string;
  anomalies: LogAnomaly[];
  patterns: Array<{ pattern: string; occurrences: number; severity: string; description: string }>;
  root_cause_hypotheses: string[];
  recommendations: string[];
  time_range: { from: string; to: string };
  nodes_analyzed: string[];
  model_used: string;
}

interface LogState {
  entries: LogEntry[];
  anomalies: LogAnomaly[];
  bookmarks: unknown[];
  sources: LogSource[];
  kpis: LogKpis;
  filters: {
    nodeIds: string[];
    sources: string[];
    severities: string[];
    searchRegex: string;
    timeRange: { from: string; to: string } | null;
  };
  isAnalyzing: boolean;
  analysisReport: LogAnalysisReport | null;
  maxEntries: number;

  addEntry: (entry: LogEntry) => void;
  updateKpis: (kpis: LogKpis) => void;
  clearEntries: () => void;
  setFilters: (filters: Partial<LogState["filters"]>) => void;

  fetchAnomalies: (nodeId: string) => Promise<void>;
  fetchBookmarks: (nodeId: string) => Promise<void>;
  fetchSources: (nodeId: string) => Promise<void>;
  acknowledgeAnomaly: (id: string) => Promise<void>;
  analyze: (nodeIds: string[], timeFrom: string, timeTo: string, context?: string) => Promise<void>;
  setAnalysisReport: (report: LogAnalysisReport | null) => void;
}

export const useLogStore = create<LogState>()((set, get) => ({
  entries: [],
  anomalies: [],
  bookmarks: [],
  sources: [],
  kpis: { errorsPerMin: 0, warningsPerMin: 0, activeAnomalies: 0, throughput: 0 },
  filters: { nodeIds: [], sources: [], severities: [], searchRegex: "", timeRange: null },
  isAnalyzing: false,
  analysisReport: null,
  maxEntries: 10000,

  addEntry: (entry) => set((state) => {
    const entries = [...state.entries, entry];
    if (entries.length > state.maxEntries) {
      entries.splice(0, entries.length - state.maxEntries);
    }
    return { entries };
  }),

  updateKpis: (kpis) => set({ kpis }),
  clearEntries: () => set({ entries: [] }),
  setFilters: (filters) => set((state) => ({ filters: { ...state.filters, ...filters } })),

  fetchAnomalies: async (nodeId) => {
    try {
      const res = await logAnalysisApi.getAnomalies(nodeId, { limit: 100 });
      set({ anomalies: res.data });
    } catch { /* ignore */ }
  },

  fetchBookmarks: async (nodeId) => {
    try {
      const res = await logAnalysisApi.getBookmarks(nodeId);
      set({ bookmarks: res.data });
    } catch { /* ignore */ }
  },

  fetchSources: async (nodeId) => {
    try {
      const res = await logAnalysisApi.getSources(nodeId);
      set({ sources: res.data });
    } catch { /* ignore */ }
  },

  acknowledgeAnomaly: async (id) => {
    try {
      await logAnalysisApi.acknowledgeAnomaly(id);
      set((state) => ({
        anomalies: state.anomalies.map((a) =>
          a.id === id ? { ...a, is_acknowledged: true } : a
        ),
      }));
    } catch { /* ignore */ }
  },

  analyze: async (nodeIds, timeFrom, timeTo, context) => {
    set({ isAnalyzing: true });
    try {
      const res = await logAnalysisApi.analyze({ node_ids: nodeIds, time_from: timeFrom, time_to: timeTo, context });
      set({ analysisReport: res.data?.report_json ? JSON.parse(res.data.report_json) : res.data, isAnalyzing: false });
    } catch {
      set({ isAnalyzing: false });
    }
  },

  setAnalysisReport: (report) => set({ analysisReport: report }),
}));
