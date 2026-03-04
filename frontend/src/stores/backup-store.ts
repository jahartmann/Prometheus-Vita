"use client";

import { create } from "zustand";
import type { ConfigBackup, BackupSchedule } from "@/types/api";
import { backupApi, scheduleApi } from "@/lib/api";

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
      set({ backups: response.data?.data || response.data || [], isLoading: false });
    } catch {
      set({ error: "Backups konnten nicht geladen werden", isLoading: false });
    }
  },

  createBackup: async (nodeId: string, notes?: string) => {
    const response = await backupApi.createBackup(nodeId, {
      backup_type: "manual",
      notes,
    });
    const backup = response.data?.data || response.data;
    set((state) => ({ backups: [backup, ...state.backups] }));
    return backup;
  },

  deleteBackup: async (backupId: string) => {
    await backupApi.deleteBackup(backupId);
    set((state) => ({
      backups: state.backups.filter((b) => b.id !== backupId),
    }));
  },

  fetchSchedules: async (nodeId: string) => {
    try {
      const response = await scheduleApi.listSchedules(nodeId);
      set({ schedules: response.data?.data || response.data || [] });
    } catch {
      // Schedules nicht verfuegbar
    }
  },
}));
