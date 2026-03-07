"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth-store";
import { Sidebar } from "./sidebar";
import { TooltipProvider } from "@/components/ui/tooltip";
import { ChatPanel } from "@/components/chat/chat-panel";
import { ChatToggle } from "@/components/chat/chat-toggle";

interface AppLayoutProps {
  children: React.ReactNode;
}

export function AppLayout({ children }: AppLayoutProps) {
  const [hydrated, setHydrated] = useState(false);
  const router = useRouter();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);

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

  if (!hydrated || !isAuthenticated) {
    return null;
  }

  return (
    <TooltipProvider>
      <div className="flex h-screen overflow-hidden bg-background">
        <Sidebar />
        <main className="flex-1 overflow-auto p-6">{children}</main>
        <ChatPanel />
        <ChatToggle />
      </div>
    </TooltipProvider>
  );
}
