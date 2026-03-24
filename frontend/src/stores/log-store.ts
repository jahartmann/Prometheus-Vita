import { create } from "zustand";
import { logAnalysisApi, logApi } from "@/lib/api";

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
  error: string | null;
  maxEntries: number;

  addEntry: (entry: LogEntry) => void;
  updateKpis: (kpis: LogKpis) => void;
  clearEntries: () => void;
  setFilters: (filters: Partial<LogState["filters"]>) => void;

  fetchAnomalies: (nodeId: string) => Promise<void>;
  fetchBookmarks: (nodeId: string) => Promise<void>;
  fetchSources: (nodeId: string) => Promise<void>;
  fetchLogs: (nodeId: string, file?: string, lines?: number) => Promise<void>;
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
  error: null,
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
      set({ anomalies: Array.isArray(res.data) ? res.data : [] });
    } catch (e) {
      console.error('Failed to fetch anomalies:', e);
      set({ error: 'Failed to fetch anomalies' });
    }
  },

  fetchBookmarks: async (nodeId) => {
    try {
      const res = await logAnalysisApi.getBookmarks(nodeId);
      set({ bookmarks: Array.isArray(res.data) ? res.data : [] });
    } catch (e) {
      console.error('Failed to fetch bookmarks:', e);
      set({ error: 'Failed to fetch bookmarks' });
    }
  },

  fetchSources: async (nodeId) => {
    try {
      const res = await logAnalysisApi.getSources(nodeId);
      set({ sources: Array.isArray(res.data) ? res.data : [] });
    } catch (e) {
      console.error('Failed to fetch sources:', e);
      set({ error: 'Failed to fetch sources' });
    }
  },

  fetchLogs: async (nodeId, file = "syslog", lines = 200) => {
    try {
      const res = await logApi.getLogs(nodeId, file, lines);
      const raw = typeof res.data === "string" ? res.data : (res.data?.lines || "");
      if (!raw) return;
      const logLines = raw.split("\n").filter((l: string) => l.trim());
      const now = new Date().toISOString();
      const newEntries: LogEntry[] = logLines.map((line: string, i: number) => {
        // Try to infer severity from content
        const lower = line.toLowerCase();
        let severity = "info";
        if (lower.includes("error") || lower.includes("fail")) severity = "error";
        else if (lower.includes("warn")) severity = "warning";
        else if (lower.includes("debug")) severity = "debug";
        else if (lower.includes("crit") || lower.includes("emerg") || lower.includes("panic")) severity = "critical";

        return {
          id: `rest-${nodeId}-${Date.now()}-${i}`,
          timestamp: now,
          node_id: nodeId,
          source: file,
          severity,
          process: "",
          pid: 0,
          message: line,
          raw: line,
        };
      });
      set((state) => {
        const entries = [...state.entries, ...newEntries];
        if (entries.length > state.maxEntries) {
          entries.splice(0, entries.length - state.maxEntries);
        }
        return { entries };
      });
    } catch (e) {
      console.error('Failed to fetch logs:', e);
      set({ error: 'Failed to fetch logs' });
    }
  },

  acknowledgeAnomaly: async (id) => {
    try {
      await logAnalysisApi.acknowledgeAnomaly(id);
      set((state) => ({
        anomalies: state.anomalies.map((a) =>
          a.id === id ? { ...a, is_acknowledged: true } : a
        ),
      }));
    } catch (e) {
      console.error('Failed to acknowledge anomaly:', e);
      set({ error: 'Failed to acknowledge anomaly' });
    }
  },

  analyze: async (nodeIds, timeFrom, timeTo, context) => {
    set({ isAnalyzing: true });
    try {
      const res = await logAnalysisApi.analyze({ node_ids: nodeIds, time_from: timeFrom, time_to: timeTo, context });
      set({ analysisReport: res.data?.report_json ? JSON.parse(res.data.report_json) : res.data, isAnalyzing: false });
    } catch (e) {
      console.error('Failed to analyze logs:', e);
      set({ isAnalyzing: false, error: 'Failed to analyze logs' });
    }
  },

  setAnalysisReport: (report) => set({ analysisReport: report }),
}));
