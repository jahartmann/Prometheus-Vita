import { create } from "zustand";
import type { UserResponse } from "@/types/api";
import { userApi } from "@/lib/api";

interface UserState {
  users: UserResponse[];
  isLoading: boolean;
  error: string | null;
  fetchUsers: () => Promise<void>;
}

export const useUserStore = create<UserState>()((set) => ({
  users: [],
  isLoading: false,
  error: null,

  fetchUsers: async () => {
    set({ isLoading: true, error: null });
    try {
      const response = await userApi.list();
      set({ users: response.data?.data || response.data || [], isLoading: false });
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : "Benutzer konnten nicht geladen werden";
      set({ error: message, isLoading: false });
    }
  },
}));
