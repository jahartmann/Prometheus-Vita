"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useTheme } from "next-themes";
import {
  LayoutDashboard,
  Server,
  Settings,
  Flame,
  ChevronDown,
  ChevronRight,
  Monitor,
  Network,
  HardDrive,
  FolderArchive,
  BarChart3,
  FileText,
  Plus,
  Bell,
  Archive,
  Shield,
  ShieldCheck,
  Disc,
  ArrowRightLeft,
  GitCompare,
  Zap,
  GitBranch,
  Tag,
  AlertTriangle,
  Lightbulb,
  Search,
  Moon,
  Sun,
  LogOut,
  ChevronUp,
  HeartPulse,
  Link2,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useNodeStore } from "@/stores/node-store";
import { useAuthStore } from "@/stores/auth-store";
import { OnboardNodeDialog } from "@/components/nodes/onboard-node-dialog";

interface NavItem {
  label: string;
  href: string;
  icon: React.ComponentType<{ className?: string }>;
  matchPrefix?: string;
  excludePrefix?: string[];
}

interface NavSection {
  label: string;
  items: NavItem[];
}

const topNavItems: NavItem[] = [
  { label: "Suche", href: "/search", icon: Search, matchPrefix: "/search" },
  { label: "Benachrichtigungen", href: "/settings/notifications", icon: Bell, matchPrefix: "/settings/notifications" },
];

const mainNavItems: NavItem[] = [
  { label: "Dashboard", href: "/", icon: LayoutDashboard },
  { label: "Monitoring", href: "/monitoring", icon: BarChart3, matchPrefix: "/monitoring" },
];

const sections: NavSection[] = [
  {
    label: "Infrastruktur",
    items: [
      { label: "Speicher", href: "/storage", icon: HardDrive, matchPrefix: "/storage" },
      { label: "Backups", href: "/backups", icon: Archive, matchPrefix: "/backups" },
      { label: "Migration", href: "/migrations", icon: ArrowRightLeft, matchPrefix: "/migrations" },
      { label: "Disaster Recovery", href: "/disaster-recovery", icon: Shield, matchPrefix: "/disaster-recovery" },
    ],
  },
  {
    label: "Sicherheit & KI",
    items: [
      { label: "Sicherheit", href: "/security", icon: ShieldCheck, matchPrefix: "/security" },
      { label: "Alerts", href: "/alerts", icon: AlertTriangle, matchPrefix: "/alerts" },
      { label: "Drift-Erkennung", href: "/drift", icon: GitCompare, matchPrefix: "/drift" },
      { label: "Empfehlungen", href: "/recommendations", icon: Lightbulb, matchPrefix: "/recommendations" },
      { label: "VM-Gesundheit", href: "/health", icon: HeartPulse, matchPrefix: "/health" },
    ],
  },
  {
    label: "System",
    items: [
      { label: "Topologie", href: "/topology", icon: GitBranch, matchPrefix: "/topology" },
      { label: "Abhängigkeiten", href: "/dependencies", icon: Link2, matchPrefix: "/dependencies" },
      { label: "Reflex-Regeln", href: "/reflex", icon: Zap, matchPrefix: "/reflex" },
      { label: "Tags", href: "/tags", icon: Tag, matchPrefix: "/tags" },
      { label: "ISOs & Vorlagen", href: "/isos", icon: Disc, matchPrefix: "/isos" },
      { label: "Einstellungen", href: "/settings/nodes", icon: Settings, matchPrefix: "/settings", excludePrefix: ["/settings/notifications"] },
    ],
  },
];

const nodeSubItems = [
  { label: "Übersicht", path: "", icon: LayoutDashboard },
  { label: "VMs & Container", path: "vms", icon: Monitor },
  { label: "Monitoring", path: "monitoring", icon: BarChart3 },
  { label: "Netzwerk", path: "network", icon: Network },
  { label: "Storage", path: "storage", icon: HardDrive },
  { label: "Backups", path: "backups", icon: FolderArchive },
  { label: "ISOs & Vorlagen", path: "iso-templates", icon: Disc },
];

interface SidebarProps {
  mobileOpen?: boolean;
  onMobileClose?: () => void;
}

export function Sidebar({ mobileOpen = false, onMobileClose }: SidebarProps) {
  const pathname = usePathname();
  const router = useRouter();
  const { theme, setTheme } = useTheme();
  const { nodes, fetchNodes } = useNodeStore();
  const { user, logout } = useAuthStore();
  const [serversOpen, setServersOpen] = useState(pathname.startsWith("/nodes"));
  const [openNodes, setOpenNodes] = useState<Record<string, boolean>>({});
  const [onboardOpen, setOnboardOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);

  useEffect(() => {
    const token = useAuthStore.getState().accessToken;
    if (token) fetchNodes();
    const unsub = useAuthStore.subscribe((state) => {
      if (state.accessToken) fetchNodes();
    });
    return () => unsub();
  }, [fetchNodes]);

  useEffect(() => {
    if (pathname.startsWith("/nodes/")) {
      const activeNodeId = pathname.split("/")[2];
      if (activeNodeId) {
        setServersOpen(true);
        setOpenNodes((prev) => ({ ...prev, [activeNodeId]: true }));
      }
    }
  }, [pathname]);

  const isActive = (href: string, matchPrefix?: string, excludePrefix?: string[]) => {
    if (matchPrefix) {
      if (excludePrefix?.some((ex) => pathname.startsWith(ex))) return false;
      return pathname.startsWith(matchPrefix);
    }
    return pathname === href;
  };

  const handleLogout = () => {
    logout();
    router.push("/login");
  };

  const initials = user?.username ? user.username.slice(0, 2).toUpperCase() : "??";

  const renderNavLink = (item: NavItem) => {
    const active = isActive(item.href, item.matchPrefix, item.excludePrefix);
    const Icon = item.icon;
    return (
      <Link
        key={item.href}
        href={item.href}
        className={cn(
          "flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors",
          active
            ? "bg-primary/10 text-primary font-semibold"
            : "text-sidebar-muted hover:bg-accent hover:text-foreground"
        )}
      >
        <Icon className="h-4 w-4 shrink-0" />
        <span>{item.label}</span>
      </Link>
    );
  };

  const sidebarContent = (
    <aside className="flex h-screen w-60 flex-col bg-sidebar border-r border-border" role="navigation" aria-label="Hauptnavigation">
      {/* Logo */}
      <div className="flex h-14 items-center gap-2 px-4">
        <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-zinc-900 dark:bg-white">
          <Flame className="h-4 w-4 text-white dark:text-zinc-900" />
        </div>
        <span className="text-sm font-semibold">Prometheus</span>
      </div>

      {/* Top Nav */}
      <nav className="space-y-0.5 px-3">
        {topNavItems.map(renderNavLink)}
      </nav>

      <div className="mx-3 my-2 border-t border-border" />

      {/* Main Nav + Sections */}
      <nav className="flex-1 space-y-0.5 overflow-y-auto px-3">
        {mainNavItems.map(renderNavLink)}

        {/* Server Collapsible */}
        <Collapsible open={serversOpen} onOpenChange={setServersOpen}>
          <CollapsibleTrigger
            className={cn(
              "flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors",
              pathname.startsWith("/nodes")
                ? "bg-primary/10 text-primary font-semibold"
                : "text-sidebar-muted hover:bg-accent hover:text-foreground"
            )}
          >
            <Server className="h-4 w-4 shrink-0" />
            <span className="flex-1 text-left">Server</span>
            {serversOpen ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
          </CollapsibleTrigger>
          <CollapsibleContent className="ml-4 space-y-0.5 border-l border-border pl-3">
            {nodes.map((node) => (
              <Collapsible
                key={node.id}
                open={openNodes[node.id] ?? false}
                onOpenChange={() => setOpenNodes((prev) => ({ ...prev, [node.id]: !prev[node.id] }))}
              >
                <CollapsibleTrigger
                  className={cn(
                    "flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-sm transition-colors",
                    pathname.startsWith(`/nodes/${node.id}`)
                      ? "bg-primary/10 text-primary font-medium"
                      : "text-sidebar-muted hover:bg-accent hover:text-foreground"
                  )}
                >
                  <span className={cn("h-2 w-2 shrink-0 rounded-full", node.is_online ? "bg-green-500" : "bg-red-500")} />
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="flex-1 truncate text-left">{node.name}</span>
                    </TooltipTrigger>
                    <TooltipContent side="right">
                      <p>{node.name}</p>
                      <p className="text-xs text-muted-foreground">{node.hostname}</p>
                    </TooltipContent>
                  </Tooltip>
                  {openNodes[node.id] ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
                </CollapsibleTrigger>
                <CollapsibleContent className="ml-3 space-y-0.5 border-l border-border pl-2">
                  {nodeSubItems.map((sub) => {
                    const SubIcon = sub.icon;
                    const subHref = sub.path ? `/nodes/${node.id}/${sub.path}` : `/nodes/${node.id}`;
                    const subActive = pathname === subHref;
                    return (
                      <Link
                        key={sub.path || "overview"}
                        href={subHref}
                        className={cn(
                          "flex items-center gap-2 rounded-lg px-2 py-1 text-xs transition-colors",
                          subActive
                            ? "bg-primary/10 text-primary font-medium"
                            : "text-sidebar-muted hover:bg-accent hover:text-foreground"
                        )}
                      >
                        <SubIcon className="h-3.5 w-3.5 shrink-0" />
                        <span>{sub.label}</span>
                      </Link>
                    );
                  })}
                </CollapsibleContent>
              </Collapsible>
            ))}
            <button
              onClick={() => setOnboardOpen(true)}
              className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-sm text-sidebar-muted transition-colors hover:bg-accent hover:text-foreground"
            >
              <Plus className="h-3.5 w-3.5 shrink-0" />
              <span>Server hinzufügen</span>
            </button>
          </CollapsibleContent>
        </Collapsible>

        {/* Grouped Sections */}
        {sections.map((section) => (
          <div key={section.label}>
            <div className="mb-1 mt-4 px-3 text-xs font-semibold uppercase tracking-wider text-sidebar-muted">
              {section.label}
            </div>
            {section.items.map(renderNavLink)}
          </div>
        ))}
      </nav>

      {/* User Area */}
      <div className="border-t border-border p-3">
        <button
          onClick={() => setUserMenuOpen(!userMenuOpen)}
          className="flex w-full items-center gap-3 rounded-lg px-2 py-2 text-sm transition-colors hover:bg-accent"
        >
          <div className="flex h-7 w-7 items-center justify-center rounded-full bg-zinc-200 text-xs font-medium text-zinc-700 dark:bg-zinc-700 dark:text-zinc-200">
            {initials}
          </div>
          <span className="flex-1 text-left font-medium text-sm">{user?.username ?? "User"}</span>
          <ChevronUp className={cn("h-3.5 w-3.5 text-sidebar-muted transition-transform", !userMenuOpen && "rotate-180")} />
        </button>
        {userMenuOpen && (
          <div className="mt-1 space-y-0.5">
            <button
              onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
              className="flex w-full items-center gap-3 rounded-lg px-2 py-1.5 text-sm text-sidebar-muted transition-colors hover:bg-accent hover:text-foreground"
            >
              {theme === "dark" ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
              <span>{theme === "dark" ? "Light Mode" : "Dark Mode"}</span>
            </button>
            <button
              onClick={handleLogout}
              className="flex w-full items-center gap-3 rounded-lg px-2 py-1.5 text-sm text-sidebar-muted transition-colors hover:bg-accent hover:text-foreground"
            >
              <LogOut className="h-4 w-4" />
              <span>Abmelden</span>
            </button>
          </div>
        )}
      </div>
    </aside>
  );

  return (
    <>
      {/* Mobile drawer */}
      <div className="md:hidden">
        {mobileOpen && (
          <>
            <div className="fixed inset-0 z-40 bg-black/50" onClick={onMobileClose} aria-hidden="true" />
            <div className="fixed inset-y-0 left-0 z-50 w-60">
              {sidebarContent}
            </div>
          </>
        )}
      </div>

      {/* Desktop sidebar */}
      <div className="hidden md:block">
        {sidebarContent}
      </div>

      <OnboardNodeDialog open={onboardOpen} onOpenChange={setOnboardOpen} />
    </>
  );
}
