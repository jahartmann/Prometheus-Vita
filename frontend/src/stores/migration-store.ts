"use client";

import { create } from "zustand";
import { migrationApi, toArray } from "@/lib/api";
import type { VMMigration } from "@/types/api";

interface MigrationState {
  migrations: VMMigration[];
  activeMigrations: VMMigration[];
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
  updateMigrationProgress: (migration: VMMigration) => void;
}

export const useMigrationStore = create<MigrationState>()((set, get) => ({
  migrations: [],
  activeMigrations: [],
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
      set({ error: "Fehler beim Laden der Migrationen", isLoading: false });
    }
  },

  startMigration: async (data) => {
    const res = await migrationApi.start(data);
    const migration = res.data?.data || res.data;
    set((state) => ({
      migrations: [migration, ...state.migrations],
      activeMigrations: [migration, ...state.activeMigrations],
    }));
    return migration;
  },

  cancelMigration: async (id: string) => {
    await migrationApi.cancel(id);
    set((state) => ({
      migrations: state.migrations.map((m) =>
        m.id === id ? { ...m, status: "cancelled" as const } : m
      ),
      activeMigrations: state.activeMigrations.filter((m) => m.id !== id),
    }));
  },

  deleteMigration: async (id: string) => {
    await migrationApi.delete(id);
    set((state) => ({
      migrations: state.migrations.filter((m) => m.id !== id),
      activeMigrations: state.activeMigrations.filter((m) => m.id !== id),
    }));
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
}));
