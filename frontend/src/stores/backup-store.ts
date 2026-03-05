import { create } from "zustand";
import { toast } from "sonner";
import type { ConfigBackup, BackupSchedule } from "@/types/api";
import { backupApi, scheduleApi, toArray } from "@/lib/api";

interface BackupState {
  backups: ConfigBackup[];
  schedules: BackupSchedule[];
  isLoading: boolean;
  error: string | null;
  fetchBackups: (nodeId: string) => Promise<void>;
  createBackup: (nodeId: string, notes?: string) => Promise<ConfigBackup>;
  deleteBackup: (backupId: string) => Promise<void>;
  fetchSchedules: (nodeId: string) => Promise<void>;
}

export const useBackupStore = create<BackupState>()((set) => ({
  backups: [],
  schedules: [],
  isLoading: false,
  error: null,

  fetchBackups: async (nodeId: string) => {
    set({ isLoading: true, error: null });
    try {
      const response = await backupApi.listBackups(nodeId);
      set({ backups: toArray<ConfigBackup>(response.data), isLoading: false });
    } catch {
      toast.error("Backups konnten nicht geladen werden");
      set({ error: "Backups konnten nicht geladen werden", isLoading: false });
    }
  },

  createBackup: async (nodeId: string, notes?: string) => {
    try {
      const response = await backupApi.createBackup(nodeId, {
        backup_type: "manual",
        notes,
      });
      const backup = response.data;
      set((state) => ({ backups: [backup, ...state.backups] }));
      toast.success("Backup erfolgreich erstellt");
      return backup;
    } catch {
      toast.error("Backup konnte nicht erstellt werden");
      set({ error: "Backup konnte nicht erstellt werden" });
      throw new Error("Backup konnte nicht erstellt werden");
    }
  },

  deleteBackup: async (backupId: string) => {
    try {
      await backupApi.deleteBackup(backupId);
      set((state) => ({
        backups: state.backups.filter((b) => b.id !== backupId),
      }));
      toast.success("Backup geloescht");
    } catch {
      toast.error("Backup konnte nicht geloescht werden");
      set({ error: "Backup konnte nicht geloescht werden" });
      throw new Error("Backup konnte nicht geloescht werden");
    }
  },

  fetchSchedules: async (nodeId: string) => {
    try {
      const response = await scheduleApi.listSchedules(nodeId);
      set({ schedules: toArray<BackupSchedule>(response.data) });
    } catch {
      toast.error("Zeitplaene konnten nicht geladen werden");
      set({ error: "Zeitplaene konnten nicht geladen werden" });
    }
  },
}));
