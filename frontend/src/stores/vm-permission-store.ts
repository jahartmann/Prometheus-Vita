import { create } from "zustand";
import { toast } from "sonner";
import type { VMPermission } from "@/types/api";
import { vmPermissionApi, toArray } from "@/lib/api";

interface VMPermissionState {
  permissions: VMPermission[];
  allPermissionTypes: string[];
  isLoading: boolean;
  error: string | null;
  fetchPermissions: () => Promise<void>;
  fetchAllPermissionTypes: () => Promise<void>;
  upsertPermission: (data: {
    user_id: string;
    target_type: string;
    target_id: string;
    node_id: string;
    permissions: string[];
  }) => Promise<void>;
  deletePermission: (id: string) => Promise<void>;
}

export const useVMPermissionStore = create<VMPermissionState>()((set, get) => ({
  permissions: [],
  allPermissionTypes: [],
  isLoading: false,
  error: null,

  fetchPermissions: async () => {
    set({ isLoading: true, error: null });
    try {
      const response = await vmPermissionApi.list();
      set({ permissions: toArray<VMPermission>(response.data), isLoading: false });
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : "VM-Berechtigungen konnten nicht geladen werden";
      toast.error(message);
      set({ error: message, isLoading: false });
    }
  },

  fetchAllPermissionTypes: async () => {
    try {
      const response = await vmPermissionApi.listAllPermissions();
      const data = response.data;
      set({ allPermissionTypes: Array.isArray(data) ? data : [] });
    } catch {
      // silent
    }
  },

  upsertPermission: async (data) => {
    try {
      await vmPermissionApi.upsert(data);
      toast.success("Berechtigung gespeichert");
      await get().fetchPermissions();
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : "Berechtigung konnte nicht gespeichert werden";
      toast.error(message);
    }
  },

  deletePermission: async (id) => {
    try {
      await vmPermissionApi.delete(id);
      toast.success("Berechtigung entfernt");
      await get().fetchPermissions();
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : "Berechtigung konnte nicht entfernt werden";
      toast.error(message);
    }
  },
}));
