import { create } from "zustand";
import type { User } from "./client";

type AuthState = {
  accessToken: string | null;
  user: User | null;
  status: "anonymous" | "authenticated" | "refreshing";
  setSession: (token: string, user: User) => void;
  clearSession: () => void;
  setStatus: (status: AuthState["status"]) => void;
};

export const useAuthStore = create<AuthState>((set) => ({
  accessToken: null,
  user: null,
  status: "anonymous",
  setSession: (token, user) => set({ accessToken: token, user, status: "authenticated" }),
  clearSession: () => set({ accessToken: null, user: null, status: "anonymous" }),
  setStatus: (status) => set({ status }),
}));

// Read-only access for non-React code (api interceptor).
export function getAccessToken(): string | null {
  return useAuthStore.getState().accessToken;
}
