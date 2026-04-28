import { useEffect, type ReactNode } from "react";
import { useNavigate, useRouterState } from "@tanstack/react-router";
import { useAuthStore } from "./store";
import { getMeRequest, refreshRequest } from "./client";
import { ApiError } from "@/lib/api/client";

// AuthGate ensures a session exists before rendering children. On mount it
// tries /auth/refresh (HttpOnly cookie hits the backend) and falls back to
// /login if no session can be established. While refreshing, it renders
// nothing to avoid flashing protected UI.
export function AuthGate({ children }: { children: ReactNode }) {
  const status = useAuthStore((s) => s.status);
  const setSession = useAuthStore((s) => s.setSession);
  const setStatus = useAuthStore((s) => s.setStatus);
  const clearSession = useAuthStore((s) => s.clearSession);
  const navigate = useNavigate();
  const routerLocation = useRouterState({ select: (s) => s.location.pathname });

  useEffect(() => {
    if (status !== "anonymous") return;
    if (routerLocation === "/login") return;
    setStatus("refreshing");
    refreshRequest()
      .then(async (r) => {
        const me = await getMeRequest();
        setSession(r.access_token, me);
      })
      .catch((err) => {
        clearSession();
        if (!(err instanceof ApiError) || err.status !== 401) {
          // network error vs unauth: swallow either; user lands on /login
        }
        navigate({ to: "/login" });
      });
  }, [status, routerLocation, setStatus, setSession, clearSession, navigate]);

  if (status === "refreshing") return null;
  if (status === "anonymous" && routerLocation !== "/login") return null;
  return <>{children}</>;
}
