"use client";

import { useTheme } from "next-themes";
import { useRouter, usePathname } from "next/navigation";
import { Moon, Sun, LogOut, User, PanelLeftClose, PanelLeft, Bell, Menu } from "lucide-react";
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
import { useAuthStore } from "@/stores/auth-store";
import { SearchTrigger } from "@/components/search/search-command";
import { Breadcrumbs } from "@/components/layout/breadcrumbs";

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
};

interface HeaderProps {
  collapsed: boolean;
  onToggleCollapse: () => void;
  onMobileMenuToggle?: () => void;
}

export function Header({ collapsed, onToggleCollapse, onMobileMenuToggle }: HeaderProps) {
  const { theme, setTheme } = useTheme();
  const router = useRouter();
  const pathname = usePathname();
  const { user, logout } = useAuthStore();

  // Derive mobile page title from pathname
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

  const initials = user?.username
    ? user.username.slice(0, 2).toUpperCase()
    : "??";

  return (
    <header className="flex h-14 items-center justify-between border-b bg-card px-4">
      <div className="flex items-center gap-3">
        {/* Mobile menu button */}
        <Button variant="ghost" size="icon" className="md:hidden" onClick={onMobileMenuToggle} aria-label="Menü öffnen">
          <Menu className="h-5 w-5" />
        </Button>
        {/* Desktop collapse button */}
        <Button variant="ghost" size="icon" className="hidden md:inline-flex" onClick={onToggleCollapse} aria-label={collapsed ? "Seitenleiste einblenden" : "Seitenleiste ausblenden"}>
          {collapsed ? (
            <PanelLeft className="h-4 w-4" />
          ) : (
            <PanelLeftClose className="h-4 w-4" />
          )}
        </Button>
        <SearchTrigger />
        {/* Mobile: show current page title */}
        <span className="md:hidden text-sm font-medium truncate max-w-[200px]">
          {mobileTitle}
        </span>
        {/* Desktop: show breadcrumbs */}
        <div className="hidden md:block">
          <Breadcrumbs />
        </div>
      </div>

      <div className="flex items-center gap-1">
        <Button
          variant="ghost"
          size="icon"
          onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
        >
          <Sun className="h-4 w-4 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
          <Moon className="absolute h-4 w-4 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
          <span className="sr-only">Theme wechseln</span>
        </Button>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" className="relative h-8 w-8 rounded-full" aria-label="Benutzermenü">
              <Avatar className="h-8 w-8">
                <AvatarFallback className="bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400 text-xs">
                  {initials}
                </AvatarFallback>
              </Avatar>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent className="w-56" align="end" forceMount>
            <DropdownMenuLabel className="font-normal">
              <div className="flex flex-col space-y-1">
                <p className="text-sm font-medium leading-none">
                  {user?.username}
                </p>
                <p className="text-xs leading-none text-muted-foreground">
                  {user?.role}
                </p>
              </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem>
              <User className="mr-2 h-4 w-4" />
              <span>Profil</span>
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
