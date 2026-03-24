"use client";

import { useState, useEffect } from "react";
import { useRouter, usePathname } from "next/navigation";
import { useAuthStore } from "@/stores/auth-store";
import { Sidebar } from "./sidebar";
import { Header } from "./header";
import { TooltipProvider } from "@/components/ui/tooltip";
import { ChatPanel } from "@/components/chat/chat-panel";
import { ChatToggle } from "@/components/chat/chat-toggle";
import { SearchCommand } from "@/components/search/search-command";

interface AppLayoutProps {
  children: React.ReactNode;
}

export function AppLayout({ children }: AppLayoutProps) {
  const [hydrated, setHydrated] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const router = useRouter();
  const pathname = usePathname();
  const { isAuthenticated, user } = useAuthStore();

  useEffect(() => {
    if (useAuthStore.persist.hasHydrated()) {
      setHydrated(true);
    } else {
      const unsub = useAuthStore.persist.onFinishHydration(() => {
        setHydrated(true);
      });
      return unsub;
    }
  }, []);

  useEffect(() => {
    if (hydrated && !isAuthenticated) {
      router.push("/login");
    }
  }, [isAuthenticated, hydrated, router]);

  // Redirect users who must change password
  useEffect(() => {
    if (
      isAuthenticated &&
      user?.must_change_password &&
      pathname !== "/change-password"
    ) {
      router.push("/change-password?forced=true");
    }
  }, [isAuthenticated, user, pathname, router]);

  if (!hydrated || !isAuthenticated) {
    return (
      <div className="flex h-screen items-center justify-center bg-background">
        <div className="flex flex-col items-center gap-3">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
          <span className="text-sm text-muted-foreground">Lade Prometheus...</span>
        </div>
      </div>
    );
  }

  return (
    <TooltipProvider>
      <div className="flex h-screen overflow-hidden bg-background">
        <Sidebar mobileOpen={mobileMenuOpen} onMobileClose={() => setMobileMenuOpen(false)} />
        <div className="flex flex-1 flex-col overflow-hidden">
          <Header
            collapsed={sidebarCollapsed}
            onToggleCollapse={() => setSidebarCollapsed(!sidebarCollapsed)}
            onMobileMenuToggle={() => setMobileMenuOpen(!mobileMenuOpen)}
          />
          <main className="flex-1 overflow-auto p-4 md:p-6">{children}</main>
        </div>
        <ChatPanel />
        <ChatToggle />
        <SearchCommand />
      </div>
    </TooltipProvider>
  );
}
