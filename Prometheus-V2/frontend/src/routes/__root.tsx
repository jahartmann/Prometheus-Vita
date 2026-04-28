import { createRootRoute, Outlet, useRouterState } from "@tanstack/react-router";
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools";
import { ThemeProvider } from "@/components/theme-provider";
import { AppShell } from "@/components/layout/app-shell";
import { AuthGate } from "@/lib/auth/guard";

export const Route = createRootRoute({
  component: RootRoute,
});

function RootRoute() {
  const isLogin = useRouterState({ select: (s) => s.location.pathname === "/login" });
  return (
    <ThemeProvider>
      <AuthGate>
        {isLogin ? (
          <Outlet />
        ) : (
          <AppShell>
            <Outlet />
          </AppShell>
        )}
      </AuthGate>
      {import.meta.env.DEV && <TanStackRouterDevtools position="bottom-right" />}
    </ThemeProvider>
  );
}
