import { create } from "zustand";
import { toast } from "sonner";
import type { VMGroup, VMGroupMember } from "@/types/api";
import { vmGroupApi, toArray } from "@/lib/api";

interface VMGroupState {
  groups: VMGroup[];
  members: Record<string, VMGroupMember[]>;
  isLoading: boolean;
  error: string | null;
  fetchGroups: () => Promise<void>;
  fetchMembers: (groupId: string) => Promise<void>;
  createGroup: (data: { name: string; description?: string; tag_filter?: string }) => Promise<void>;
  updateGroup: (id: string, data: { name?: string; description?: string; tag_filter?: string }) => Promise<void>;
  deleteGroup: (id: string) => Promise<void>;
  addMember: (groupId: string, nodeId: string, vmid: number) => Promise<void>;
  removeMember: (groupId: string, nodeId: string, vmid: number) => Promise<void>;
}

export const useVMGroupStore = create<VMGroupState>()((set, get) => ({
  groups: [],
  members: {},
  isLoading: false,
  error: null,

  fetchGroups: async () => {
    set({ isLoading: true, error: null });
    try {
      const response = await vmGroupApi.list();
      set({ groups: toArray<VMGroup>(response.data), isLoading: false });
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : "VM-Gruppen konnten nicht geladen werden";
      toast.error(message);
      set({ error: message, isLoading: false });
    }
  },

  fetchMembers: async (groupId) => {
    try {
      const response = await vmGroupApi.listMembers(groupId);
      const data = toArray<VMGroupMember>(response.data);
      set((state) => ({
        members: { ...state.members, [groupId]: data },
      }));
    } catch {
      // silent
    }
  },

  createGroup: async (data) => {
    try {
      await vmGroupApi.create(data);
      toast.success("VM-Gruppe erstellt");
      await get().fetchGroups();
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : "VM-Gruppe konnte nicht erstellt werden";
      toast.error(message);
    }
  },

  updateGroup: async (id, data) => {
    try {
      await vmGroupApi.update(id, data);
      toast.success("VM-Gruppe aktualisiert");
      await get().fetchGroups();
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : "VM-Gruppe konnte nicht aktualisiert werden";
      toast.error(message);
    }
  },

  deleteGroup: async (id) => {
    try {
      await vmGroupApi.delete(id);
      toast.success("VM-Gruppe gelöscht");
      await get().fetchGroups();
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : "VM-Gruppe konnte nicht gelöscht werden";
      toast.error(message);
    }
  },

  addMember: async (groupId, nodeId, vmid) => {
    try {
      await vmGroupApi.addMember(groupId, { node_id: nodeId, vmid });
      toast.success("VM zur Gruppe hinzugefügt");
      await get().fetchMembers(groupId);
      await get().fetchGroups();
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : "VM konnte nicht zur Gruppe hinzugefügt werden";
      toast.error(message);
    }
  },

  removeMember: async (groupId, nodeId, vmid) => {
    try {
      await vmGroupApi.removeMember(groupId, { node_id: nodeId, vmid });
      toast.success("VM aus Gruppe entfernt");
      await get().fetchMembers(groupId);
      await get().fetchGroups();
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : "VM konnte nicht aus Gruppe entfernt werden";
      toast.error(message);
    }
  },
}));
