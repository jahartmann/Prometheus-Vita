"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth-store";
import { Sidebar } from "./sidebar";
import { Header } from "./header";
import { TooltipProvider } from "@/components/ui/tooltip";
import { ChatPanel } from "@/components/chat/chat-panel";
import { ChatToggle } from "@/components/chat/chat-toggle";

interface AppLayoutProps {
  children: React.ReactNode;
}

export function AppLayout({ children }: AppLayoutProps) {
  const [collapsed, setCollapsed] = useState(false);
  const [hydrated, setHydrated] = useState(false);
  const router = useRouter();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);

  useEffect(() => {
    // Check if already hydrated synchronously (persist hydrates sync from localStorage)
    if (useAuthStore.persist.hasHydrated()) {
      setHydrated(true);
    } else {
      // If not yet hydrated, wait for it
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

  // Show nothing until hydration is complete (prevents flash)
  if (!hydrated || !isAuthenticated) {
    return null;
  }

  return (
    <TooltipProvider>
      <div className="flex h-screen overflow-hidden">
        <Sidebar collapsed={collapsed} />
        <div className="flex flex-1 flex-col overflow-hidden">
          <Header
            collapsed={collapsed}
            onToggleCollapse={() => setCollapsed(!collapsed)}
          />
          <main className="flex-1 overflow-auto p-6">{children}</main>
        </div>
        <ChatPanel />
        <ChatToggle />
      </div>
    </TooltipProvider>
  );
}
