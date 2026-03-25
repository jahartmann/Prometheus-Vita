import { create } from "zustand";
import { toast } from "sonner";
import type {
  NotificationChannel,
  NotificationHistoryEntry,
  AlertRule,
} from "@/types/api";
import { notificationApi, alertApi, toArray } from "@/lib/api";

interface NotificationState {
  channels: NotificationChannel[];
  history: NotificationHistoryEntry[];
  alertRules: AlertRule[];
  isLoading: boolean;
  error: string | null;

  fetchChannels: () => Promise<void>;
  fetchHistory: (limit?: number, offset?: number) => Promise<void>;
  fetchAlertRules: () => Promise<void>;
}

export const useNotificationStore = create<NotificationState>()((set) => ({
  channels: [],
  history: [],
  alertRules: [],
  isLoading: false,
  error: null,

  fetchChannels: async () => {
    set({ isLoading: true, error: null });
    try {
      const response = await notificationApi.listChannels();
      set({ channels: toArray<NotificationChannel>(response.data), isLoading: false });
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : "Kanäle konnten nicht geladen werden";
      toast.error(message);
      set({ error: message, isLoading: false });
    }
  },

  fetchHistory: async (limit = 50, offset = 0) => {
    set({ isLoading: true, error: null });
    try {
      const response = await notificationApi.listHistory(limit, offset);
      set({ history: toArray<NotificationHistoryEntry>(response.data), isLoading: false });
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : "Verlauf konnte nicht geladen werden";
      toast.error(message);
      set({ error: message, isLoading: false });
    }
  },

  fetchAlertRules: async () => {
    set({ isLoading: true, error: null });
    try {
      const response = await alertApi.listRules();
      set({ alertRules: toArray<AlertRule>(response.data), isLoading: false });
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : "Alert-Regeln konnten nicht geladen werden";
      toast.error(message);
      set({ error: message, isLoading: false });
    }
  },
}));
