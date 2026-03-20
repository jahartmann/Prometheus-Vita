"use client";

import { create } from "zustand";
import { toast } from "sonner";
import { migrationApi, toArray } from "@/lib/api";
import type { VMMigration } from "@/types/api";

interface MigrationLogEntry {
  line: string;
  timestamp: string;
}

interface MigrationState {
  migrations: VMMigration[];
  activeMigrations: VMMigration[];
  migrationLogs: Record<string, MigrationLogEntry[]>;
  isLoading: boolean;
  error: string | null;

  fetchMigrations: () => Promise<void>;
  fetchByNode: (nodeId: string) => Promise<void>;
  startMigration: (data: {
    source_node_id: string;
    target_node_id: string;
    vmid: number;
    target_storage: string;
    mode?: string;
    new_vmid?: number;
    cleanup_source?: boolean;
    cleanup_target?: boolean;
  }) => Promise<VMMigration>;
  cancelMigration: (id: string) => Promise<void>;
  deleteMigration: (id: string) => Promise<void>;
  retryMigration: (migration: VMMigration) => Promise<VMMigration>;
  updateMigrationProgress: (migration: VMMigration) => void;
  addMigrationLog: (migrationId: string, line: string, timestamp: string) => void;
  loadMigrationLogs: (migrationId: string) => Promise<void>;
}

export const useMigrationStore = create<MigrationState>()((set, get) => ({
  migrations: [],
  activeMigrations: [],
  migrationLogs: {},
  isLoading: false,
  error: null,

  fetchMigrations: async () => {
    set({ isLoading: true, error: null });
    try {
      const res = await migrationApi.list();
      const migrations = toArray<VMMigration>(res.data);
      set({
        migrations,
        activeMigrations: migrations.filter(
          (m) => !["completed", "failed", "cancelled"].includes(m.status)
        ),
        isLoading: false,
      });
    } catch {
      toast.error("Fehler beim Laden der Migrationen");
      set({ error: "Fehler beim Laden der Migrationen", isLoading: false });
    }
  },

  fetchByNode: async (nodeId: string) => {
    set({ isLoading: true, error: null });
    try {
      const res = await migrationApi.listByNode(nodeId);
      const migrations = toArray<VMMigration>(res.data);
      set({
        migrations,
        activeMigrations: migrations.filter(
          (m) => !["completed", "failed", "cancelled"].includes(m.status)
        ),
        isLoading: false,
      });
    } catch {
      toast.error("Fehler beim Laden der Migrationen");
      set({ error: "Fehler beim Laden der Migrationen", isLoading: false });
    }
  },

  startMigration: async (data) => {
    const res = await migrationApi.start(data);
    set((state) => ({
      migrations: [res.data, ...state.migrations],
      activeMigrations: [res.data, ...state.activeMigrations],
      migrationLogs: { ...state.migrationLogs, [res.data.id]: [] },
    }));
    toast.success("Migration gestartet");
    return res.data;
  },

  cancelMigration: async (id: string) => {
    await migrationApi.cancel(id);
    set((state) => ({
      migrations: state.migrations.map((m) =>
        m.id === id ? { ...m, status: "cancelled" as const } : m
      ),
      activeMigrations: state.activeMigrations.filter((m) => m.id !== id),
    }));
    toast.success("Migration abgebrochen");
  },

  deleteMigration: async (id: string) => {
    await migrationApi.delete(id);
    set((state) => {
      const newLogs = { ...state.migrationLogs };
      delete newLogs[id];
      return {
        migrations: state.migrations.filter((m) => m.id !== id),
        activeMigrations: state.activeMigrations.filter((m) => m.id !== id),
        migrationLogs: newLogs,
      };
    });
    toast.success("Migration geloescht");
  },

  retryMigration: async (migration: VMMigration) => {
    // Delete old failed migration, then start a new one with same parameters
    try {
      await migrationApi.delete(migration.id);
    } catch {
      // ignore if delete fails
    }
    const res = await migrationApi.start({
      source_node_id: migration.source_node_id,
      target_node_id: migration.target_node_id,
      vmid: migration.vmid,
      target_storage: migration.target_storage,
      mode: migration.mode,
    });
    set((state) => {
      const newLogs = { ...state.migrationLogs };
      delete newLogs[migration.id];
      return {
        migrations: [res.data, ...state.migrations.filter((m) => m.id !== migration.id)],
        activeMigrations: [res.data, ...state.activeMigrations],
        migrationLogs: { ...newLogs, [res.data.id]: [] },
      };
    });
    toast.success("Migration wird wiederholt");
    return res.data;
  },

  updateMigrationProgress: (migration: VMMigration) => {
    set((state) => {
      const updated = state.migrations.map((m) =>
        m.id === migration.id ? migration : m
      );

      // If not in list, add it
      if (!updated.find((m) => m.id === migration.id)) {
        updated.unshift(migration);
      }

      return {
        migrations: updated,
        activeMigrations: updated.filter(
          (m) => !["completed", "failed", "cancelled"].includes(m.status)
        ),
      };
    });
  },

  addMigrationLog: (migrationId: string, line: string, timestamp: string) => {
    set((state) => {
      const existing = state.migrationLogs[migrationId] || [];
      // Keep last 200 log entries to prevent memory issues
      const updated = [...existing, { line, timestamp }];
      if (updated.length > 200) {
        updated.splice(0, updated.length - 200);
      }
      return {
        migrationLogs: { ...state.migrationLogs, [migrationId]: updated },
      };
    });
  },

  loadMigrationLogs: async (migrationId: string) => {
    try {
      const res = await migrationApi.getLogs(migrationId);
      const logs = toArray<{ line: string; created_at: string; level: string }>(res.data);
      set((state) => ({
        migrationLogs: {
          ...state.migrationLogs,
          [migrationId]: logs.map(l => ({
            line: l.line,
            timestamp: new Date(l.created_at).toLocaleTimeString("de-DE", { hour: "2-digit", minute: "2-digit", second: "2-digit" }),
          })),
        },
      }));
    } catch {
      // silently fail - logs are optional
    }
  },
}));
