import { create } from "zustand";
import { persist } from "zustand/middleware";
import axios from "axios";
import type { User, LoginRequest, LoginResponse } from "@/types/api";
import api from "@/lib/api";

interface AuthState {
  user: User | null;
  accessToken: string | null;
  refreshToken: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
  login: (credentials: LoginRequest) => Promise<void>;
  logout: () => void;
  setAccessToken: (token: string) => void;
  setRefreshToken: (token: string) => void;
  fetchUser: () => Promise<void>;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      accessToken: null,
      refreshToken: null,
      isAuthenticated: false,
      isLoading: false,
      error: null,

      login: async (credentials: LoginRequest) => {
        set({ isLoading: true, error: null });
        try {
          const response = await api.post<LoginResponse>(
            "/auth/login",
            credentials
          );
          const { access_token, refresh_token, user } = response.data;
          set({
            user,
            accessToken: access_token,
            refreshToken: refresh_token,
            isAuthenticated: true,
            isLoading: false,
          });
        } catch (err: unknown) {
          let message = "Login fehlgeschlagen";
          if (axios.isAxiosError(err) && err.response?.data?.error) {
            message = err.response.data.error;
          } else if (err instanceof Error) {
            message = err.message;
          }
          set({ error: message, isLoading: false });
          throw new Error(message);
        }
      },

      logout: () => {
        set({
          user: null,
          accessToken: null,
          refreshToken: null,
          isAuthenticated: false,
          error: null,
        });
      },

      setAccessToken: (token: string) => {
        set({ accessToken: token });
      },

      setRefreshToken: (token: string) => {
        set({ refreshToken: token });
      },

      fetchUser: async () => {
        try {
          const response = await api.get<User>("/auth/me");
          set({ user: response.data });
        } catch {
          if (!get().accessToken) {
            get().logout();
          }
        }
      },
    }),
    {
      name: "prometheus-auth",
      partialize: (state) => ({
        accessToken: state.accessToken,
        refreshToken: state.refreshToken,
        isAuthenticated: state.isAuthenticated,
        user: state.user,
      }),
    }
  )
);
