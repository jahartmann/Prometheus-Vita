"use client";

import { useEffect, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import {
  LogOut,
  Menu,
  User,
  Bell,
  Bot,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useAuthStore } from "@/stores/auth-store";
import { SearchTrigger } from "@/components/search/search-command";
import { Breadcrumbs } from "@/components/layout/breadcrumbs";
import { chatApi } from "@/lib/api";

const segmentLabels: Record<string, string> = {
  nodes: "Nodes",
  settings: "Einstellungen",
  backups: "Backups",
  monitoring: "Monitoring",
  migrations: "Migrationen",
  topology: "Topologie",
  recommendations: "Empfehlungen",
  "disaster-recovery": "Disaster Recovery",
  storage: "Speicher",
  security: "Sicherheit",
  alerts: "Alerts",
  drift: "Drift-Erkennung",
  health: "VM-Gesundheit",
  tags: "Tags",
  isos: "ISOs & Vorlagen",
  reflex: "Reflex-Regeln",
  dependencies: "Abhängigkeiten",
  search: "Suche",
  chat: "KI-Chat",
  network: "Netzwerk",
  "task-center": "Tasks",
};

interface HeaderProps {
  // Kept in the interface for backwards-compatibility with AppLayout, but the
  // collapsed/expand toggle is no longer rendered. The sidebar is always 256px
  // on desktop; users who want more space should use the mobile drawer.
  collapsed?: boolean;
  onToggleCollapse?: () => void;
  onMobileMenuToggle?: () => void;
}

interface AgentToolCall {
  status: string;
  created_at: string;
}

export function Header({ onMobileMenuToggle }: HeaderProps) {
  const router = useRouter();
  const pathname = usePathname();
  const { user, logout } = useAuthStore();
  const [agentLive, setAgentLive] = useState<"idle" | "active" | "unknown">("unknown");

  // Mini live-indicator for the agent. We poll the recent activity feed
  // every 30s and surface a green dot if any tool call ran in the last
  // 2 minutes — the dashboard already shows the full feed.
  useEffect(() => {
    let cancelled = false;
    const tick = async () => {
      try {
        const data = (await chatApi.recentActivity(5)) as AgentToolCall[];
        if (cancelled) return;
        const recent = (data || []).find((c) => {
          const age = Date.now() - new Date(c.created_at).getTime();
          return age < 2 * 60 * 1000;
        });
        setAgentLive(recent ? "active" : "idle");
      } catch {
        if (!cancelled) setAgentLive("unknown");
      }
    };
    tick();
    const handle = setInterval(tick, 30_000);
    return () => {
      cancelled = true;
      clearInterval(handle);
    };
  }, []);

  const mobileTitle = (() => {
    if (pathname === "/") return "Dashboard";
    const segments = pathname.split("/").filter(Boolean);
    const last = segments[segments.length - 1];
    return segmentLabels[last] || last;
  })();

  const handleLogout = () => {
    logout();
    router.push("/login");
  };

  const initials = user?.username ? user.username.slice(0, 2).toUpperCase() : "??";

  return (
    <header className="sticky top-0 z-30 flex h-14 shrink-0 items-center gap-3 border-b ops-divider bg-background/88 px-4 backdrop-blur-md">
      {/* Mobile menu trigger */}
      <Button
        variant="ghost"
        size="icon"
        className="h-8 w-8 md:hidden"
        onClick={onMobileMenuToggle}
        aria-label="Menü öffnen"
      >
        <Menu className="h-4.5 w-4.5" />
      </Button>

      {/* Search — visually anchored on the left for fast access */}
      <SearchTrigger />

      {/* Mobile page title */}
      <span className="truncate text-sm font-medium md:hidden">{mobileTitle}</span>

      {/* Desktop breadcrumbs */}
      <div className="hidden flex-1 md:flex">
        <Breadcrumbs />
      </div>

      <div className="ml-auto flex items-center gap-1">
        {/* Agent live-status pill — small but constant signal that the
            admin-agent is paying attention. */}
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              type="button"
              className="ops-focus-ring flex items-center gap-1.5 rounded-md border border-border/70 bg-card/80 px-2.5 py-1 text-[11px] font-medium text-muted-foreground transition-colors hover:bg-accent"
              onClick={() => router.push("/chat")}
            >
              <span
                className={`relative flex h-1.5 w-1.5 rounded-full ${
                  agentLive === "active"
                    ? "bg-emerald-500"
                    : agentLive === "idle"
                    ? "bg-emerald-400/60"
                    : "bg-zinc-500"
                }`}
              >
                {agentLive === "active" && (
                  <span className="absolute inset-0 animate-ping rounded-full bg-emerald-500/60" />
                )}
              </span>
              <Bot className="h-3 w-3" />
              <span className="hidden lg:inline">
                {agentLive === "active" ? "Agent aktiv" : agentLive === "idle" ? "Agent bereit" : "Agent"}
              </span>
            </button>
          </TooltipTrigger>
          <TooltipContent side="bottom">
            <p className="text-xs">
              {agentLive === "active"
                ? "Der Agent hat in den letzten 2 Min Tools aufgerufen."
                : agentLive === "idle"
                ? "Agent verfügbar — klicken für Chat."
                : "Agent-Status unbekannt"}
            </p>
          </TooltipContent>
        </Tooltip>

        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={() => router.push("/settings/notifications")}
              aria-label="Benachrichtigungen"
            >
              <Bell className="h-4 w-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="bottom">
            <p className="text-xs">Benachrichtigungen</p>
          </TooltipContent>
        </Tooltip>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" className="relative h-8 w-8 rounded-full" aria-label="Benutzermenü">
              <Avatar className="h-8 w-8">
                <AvatarFallback className="bg-muted text-[11px] font-semibold">
                  {initials}
                </AvatarFallback>
              </Avatar>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent className="w-56" align="end">
            <DropdownMenuLabel className="font-normal">
              <div className="flex flex-col space-y-1">
                <p className="text-sm font-medium leading-none">{user?.username}</p>
                <p className="text-xs capitalize leading-none text-muted-foreground">
                  {user?.role}
                </p>
              </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={() => router.push("/change-password")}>
              <User className="mr-2 h-4 w-4" />
              <span>Passwort ändern</span>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={handleLogout}>
              <LogOut className="mr-2 h-4 w-4" />
              <span>Abmelden</span>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  );
}
