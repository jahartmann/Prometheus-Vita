"use client";

import { useEffect, useMemo, useState } from "react";
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
  Network,
  HardDrive,
  Archive,
  ShieldCheck,
  Disc,
  ArrowRightLeft,
  GitCompare,
  Zap,
  GitBranch,
  Tag,
  AlertTriangle,
  Search,
  Moon,
  Sun,
  LogOut,
  HeartPulse,
  Link2,
  Bot,
  KeyRound,
  Users,
  RadioTower,
  UserCog,
  ListChecks,
  SearchCheck,
  FileBarChart,
  FileText,
  Plus,
  Sparkles,
  Bell,
  Shield,
  Activity,
  Workflow,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useNodeStore } from "@/stores/node-store";
import { useAuthStore } from "@/stores/auth-store";
import { OnboardNodeDialog } from "@/components/nodes/onboard-node-dialog";

// Keep the operational core visible and put deeper functions behind
// predictable collapsed groups. No navigation target is removed.

interface NavItem {
  label: string;
  href: string;
  icon: React.ComponentType<{ className?: string }>;
  matchPrefix?: string;
  excludePrefix?: string[];
  keywords?: string;
}

interface NavSection {
  label: string;
  defaultOpen?: boolean;
  includeNodes?: boolean;
  items: NavItem[];
}

const sections: NavSection[] = [
  {
    label: "Übersicht",
    defaultOpen: true,
    items: [
      { label: "Dashboard", href: "/", icon: LayoutDashboard, keywords: "lagezentrum start home" },
      { label: "Monitoring", href: "/monitoring", icon: Activity, matchPrefix: "/monitoring", keywords: "metriken graphs" },
      { label: "Task-Center", href: "/task-center", icon: ListChecks, matchPrefix: "/task-center", keywords: "aufgaben todo timeline" },
      { label: "Alerts", href: "/alerts", icon: AlertTriangle, matchPrefix: "/alerts" },
    ],
  },
  {
    label: "Infrastruktur",
    defaultOpen: false,
    includeNodes: true,
    items: [
      { label: "Cluster", href: "/cluster", icon: RadioTower, matchPrefix: "/cluster" },
      { label: "Speicher", href: "/storage", icon: HardDrive, matchPrefix: "/storage" },
      { label: "Backups", href: "/backups", icon: Archive, matchPrefix: "/backups" },
      { label: "Migrationen", href: "/migrations", icon: ArrowRightLeft, matchPrefix: "/migrations" },
      { label: "Disaster Recovery", href: "/disaster-recovery", icon: Shield, matchPrefix: "/disaster-recovery", keywords: "dr runbook" },
      { label: "ISOs & Vorlagen", href: "/isos", icon: Disc, matchPrefix: "/isos" },
      { label: "Topologie", href: "/topology", icon: Workflow, matchPrefix: "/topology" },
    ],
  },
  {
    label: "Intelligenz",
    defaultOpen: false,
    items: [
      { label: "KI-Chat", href: "/chat", icon: Bot, matchPrefix: "/chat" },
      { label: "Sicherheit", href: "/security", icon: ShieldCheck, matchPrefix: "/security" },
      { label: "Netzwerk-Analyse", href: "/network", icon: Network, matchPrefix: "/network", keywords: "ports scan bandbreite" },
      { label: "VM-Gesundheit", href: "/health", icon: HeartPulse, matchPrefix: "/health" },
      { label: "Drift-Erkennung", href: "/drift", icon: GitCompare, matchPrefix: "/drift" },
      { label: "Root Cause", href: "/root-cause", icon: SearchCheck, matchPrefix: "/root-cause" },
      { label: "Reflex-Regeln", href: "/reflex", icon: Zap, matchPrefix: "/reflex" },
      { label: "Knowledge Graph", href: "/knowledge-graph", icon: GitBranch, matchPrefix: "/knowledge-graph", keywords: "wissensbasis brain" },
    ],
  },
  {
    label: "Verwaltung",
    defaultOpen: false,
    items: [
      { label: "Logs", href: "/logs", icon: FileText, matchPrefix: "/logs" },
      { label: "Reports", href: "/reports", icon: FileBarChart, matchPrefix: "/reports" },
      { label: "Abhängigkeiten", href: "/dependencies", icon: Link2, matchPrefix: "/dependencies" },
      { label: "Tags", href: "/tags", icon: Tag, matchPrefix: "/tags" },
    ],
  },
  {
    label: "Einstellungen",
    defaultOpen: false,
    items: [
      { label: "Übersicht", href: "/settings", icon: Settings },
      { label: "Nodes", href: "/settings/nodes", icon: Server, matchPrefix: "/settings/nodes" },
      { label: "Agent & KI", href: "/settings/agent", icon: Sparkles, matchPrefix: "/settings/agent" },
      { label: "Benutzer", href: "/settings/users", icon: Users, matchPrefix: "/settings/users" },
      { label: "Rollen & Rechte", href: "/settings/roles", icon: UserCog, matchPrefix: "/settings/roles" },
      { label: "API-Tokens", href: "/settings/api-tokens", icon: KeyRound, matchPrefix: "/settings/api-tokens" },
      { label: "Audit-Log", href: "/settings/audit-log", icon: FileText, matchPrefix: "/settings/audit-log" },
      { label: "Benachrichtigungen", href: "/settings/notifications", icon: Bell, matchPrefix: "/settings/notifications" },
    ],
  },
];

function matchesNavItem(pathname: string, item: NavItem): boolean {
  if (item.matchPrefix) {
    if (item.excludePrefix?.some((p) => pathname.startsWith(p))) return false;
    return pathname.startsWith(item.matchPrefix);
  }
  return pathname === item.href;
}

function fuzzyMatch(query: string, item: NavItem): boolean {
  const q = query.trim().toLowerCase();
  if (!q) return true;
  const haystack = `${item.label} ${item.keywords ?? ""} ${item.href}`.toLowerCase();
  return haystack.includes(q);
}

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
  const [onboardOpen, setOnboardOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [openSections, setOpenSections] = useState<Record<string, boolean>>(() =>
    sections.reduce<Record<string, boolean>>((acc, section) => {
      acc[section.label] =
        Boolean(section.defaultOpen) ||
        (section.includeNodes && pathname.startsWith("/nodes")) ||
        section.items.some((item) => matchesNavItem(pathname, item));
      return acc;
    }, {})
  );
  const [serversOpen, setServersOpen] = useState(pathname.startsWith("/nodes"));
  const [openNodes, setOpenNodes] = useState<Record<string, boolean>>({});

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
    const activeSection = sections.find(
      (s) =>
        (s.includeNodes && pathname.startsWith("/nodes")) ||
        s.items.some((item) => matchesNavItem(pathname, item))
    );
    if (activeSection) {
      setOpenSections((prev) => ({ ...prev, [activeSection.label]: true }));
    }
  }, [pathname]);

  const filteredSections = useMemo(() => {
    if (!search.trim()) return sections;
    return sections
      .map((section) => ({
        ...section,
        items: section.items.filter((item) => fuzzyMatch(search, item)),
      }))
      .filter((section) => section.items.length > 0 || (section.includeNodes && search.trim() === ""));
  }, [search]);

  // While searching, force-open every section that has results.
  useEffect(() => {
    if (!search.trim()) return;
    setOpenSections((prev) => {
      const next = { ...prev };
      for (const section of filteredSections) {
        next[section.label] = true;
      }
      return next;
    });
  }, [search, filteredSections]);

  const handleLogout = () => {
    logout();
    router.push("/login");
  };

  const initials = user?.username ? user.username.slice(0, 2).toUpperCase() : "??";

  // ────────────── Render helpers ──────────────

  const renderNavLink = (item: NavItem) => {
    const active = matchesNavItem(pathname, item);
    const Icon = item.icon;
    return (
      <Link
        key={item.href}
        href={item.href}
        className={cn(
          "group relative flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-[13px] transition-colors",
          active
            ? "bg-primary/10 font-medium text-foreground"
            : "text-sidebar-muted hover:bg-accent/45 hover:text-foreground"
        )}
        onClick={() => onMobileClose?.()}
      >
        {active && (
          <span
            aria-hidden
            className="absolute left-0 top-1.5 bottom-1.5 w-0.5 rounded-r-full bg-primary"
          />
        )}
        <Icon
          className={cn(
            "h-4 w-4 shrink-0 transition-colors",
            active ? "text-foreground" : "text-sidebar-muted group-hover:text-foreground"
          )}
        />
        <span className="truncate">{item.label}</span>
      </Link>
    );
  };

  const renderNodeTree = () => (
    <div className="space-y-0.5">
      <button
        type="button"
        onClick={() => setServersOpen((v) => !v)}
        className={cn(
          "flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-[13px] transition-colors",
          pathname.startsWith("/nodes")
            ? "bg-accent/70 font-medium text-foreground"
            : "text-sidebar-muted hover:bg-accent/45 hover:text-foreground"
        )}
      >
        <Server className="h-4 w-4 shrink-0" />
        <span className="flex-1 truncate text-left">Server</span>
        <span className="text-[10px] tabular-nums text-sidebar-muted">{nodes.length}</span>
        {serversOpen ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
      </button>
      {serversOpen && (
        <div className="ml-3.5 flex flex-col gap-0.5 border-l border-border/60 pl-2">
          {nodes.map((node) => {
            const open = openNodes[node.id] ?? false;
            const onPath = pathname.startsWith(`/nodes/${node.id}`);
            return (
              <div key={node.id}>
                <button
                  type="button"
                  onClick={() => setOpenNodes((p) => ({ ...p, [node.id]: !p[node.id] }))}
                  className={cn(
                    "flex w-full items-center gap-2 rounded-md px-2 py-1 text-[12.5px] transition-colors",
                    onPath
                      ? "bg-primary/10 text-primary"
                      : "text-sidebar-muted hover:bg-accent/45 hover:text-foreground"
                  )}
                >
                  <span
                    className={cn(
                      "h-1.5 w-1.5 shrink-0 rounded-full",
                      node.is_online ? "bg-emerald-500" : "bg-zinc-500"
                    )}
                  />
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="flex-1 truncate text-left">{node.name}</span>
                    </TooltipTrigger>
                    <TooltipContent side="right">
                      <p className="font-medium">{node.name}</p>
                      <p className="text-[11px] text-muted-foreground">{node.hostname}</p>
                    </TooltipContent>
                  </Tooltip>
                  {open ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
                </button>
                {open && (
                  <div className="mb-1 ml-3 flex flex-col gap-0.5 border-l border-border/60 pl-2">
                    {[
                      { label: "Übersicht", path: "" },
                      { label: "VMs & Container", path: "vms" },
                      { label: "Monitoring", path: "monitoring" },
                      { label: "Netzwerk", path: "network" },
                      { label: "Storage", path: "storage" },
                      { label: "Backups", path: "backups" },
                      { label: "ISOs & Vorlagen", path: "iso-templates" },
                    ].map((sub) => {
                      const subHref = sub.path ? `/nodes/${node.id}/${sub.path}` : `/nodes/${node.id}`;
                      const subActive = pathname === subHref;
                      return (
                        <Link
                          key={sub.path || "overview"}
                          href={subHref}
                          className={cn(
                            "rounded-md px-2 py-0.5 text-[11.5px] transition-colors",
                            subActive
                              ? "bg-primary/10 text-primary"
                              : "text-sidebar-muted hover:bg-accent/45 hover:text-foreground"
                          )}
                        >
                          {sub.label}
                        </Link>
                      );
                    })}
                  </div>
                )}
              </div>
            );
          })}
          <button
            type="button"
            onClick={() => setOnboardOpen(true)}
            className="flex w-full items-center gap-2 rounded-md px-2 py-1 text-[12.5px] text-sidebar-muted transition-colors hover:bg-accent/45 hover:text-foreground"
          >
            <Plus className="h-3 w-3 shrink-0" />
            <span>Server hinzufügen</span>
          </button>
        </div>
      )}
    </div>
  );

  const renderSection = (section: NavSection) => {
    const open = openSections[section.label] ?? false;
    return (
      <div key={section.label} className="mb-1">
        <button
          type="button"
          onClick={() => setOpenSections((prev) => ({ ...prev, [section.label]: !open }))}
          className={cn(
            "flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-[12px] font-medium transition-colors hover:bg-accent/35 hover:text-foreground",
            open ? "text-foreground" : "text-sidebar-muted"
          )}
        >
          <span className="flex-1 text-left">{section.label}</span>
          {!open && (
            <span className="text-[10px] tabular-nums text-sidebar-muted/70">
              {section.items.length + (section.includeNodes ? 1 : 0)}
            </span>
          )}
          <ChevronDown
            className={cn(
              "h-3 w-3 transition-transform",
              open ? "rotate-0" : "-rotate-90"
            )}
          />
        </button>
        {open && (
          <div className="mt-0.5 space-y-0.5">
            {section.includeNodes && renderNodeTree()}
            {section.items.map(renderNavLink)}
          </div>
        )}
      </div>
    );
  };

  // ────────────── Layout ──────────────

  const sidebarContent = (
    <aside
      className="flex h-screen w-60 flex-col border-r ops-divider bg-sidebar"
      role="navigation"
      aria-label="Hauptnavigation"
    >
      {/* Brand */}
      <div className="flex h-12 items-center gap-2.5 border-b border-border px-3.5">
        <div className="flex h-7 w-7 items-center justify-center rounded-md bg-primary text-primary-foreground shadow-sm">
          <Flame className="h-4 w-4 text-white" />
        </div>
        <div className="leading-tight">
          <div className="text-sm font-semibold tracking-tight">Prometheus</div>
          <div className="text-[10px] text-sidebar-muted">Operations</div>
        </div>
      </div>

      {/* Search */}
      <div className="border-b border-border/70 px-2.5 py-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-sidebar-muted" />
          <input
            type="text"
            placeholder="Suchen…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="h-7 w-full rounded-md border border-border/80 bg-background/65 pl-8 pr-2.5 text-[12px] outline-none transition-colors focus:border-ring placeholder:text-sidebar-muted/70"
          />
        </div>
      </div>

      {/* Sections */}
      <nav className="flex-1 overflow-y-auto px-2 py-2.5">
        {filteredSections.map(renderSection)}
        {filteredSections.length === 0 && (
          <p className="px-3 py-6 text-center text-[12px] text-sidebar-muted">
            Keine Treffer für „{search}".
          </p>
        )}
      </nav>

      {/* User Area */}
      <div className="border-t border-border p-2">
        <button
          type="button"
          onClick={() => setUserMenuOpen(!userMenuOpen)}
          className="flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-sm transition-colors hover:bg-accent/60"
        >
          <div className="flex h-7 w-7 items-center justify-center rounded-full bg-zinc-200 text-[10px] font-semibold text-zinc-700 dark:bg-zinc-800 dark:text-zinc-200">
            {initials}
          </div>
          <span className="flex-1 truncate text-left text-[13px] font-medium">
            {user?.username ?? "User"}
          </span>
          <ChevronDown
            className={cn(
              "h-3.5 w-3.5 text-sidebar-muted transition-transform",
              userMenuOpen ? "rotate-180" : ""
            )}
          />
        </button>
        {userMenuOpen && (
          <div className="mt-1 flex flex-col gap-0.5">
            <button
              type="button"
              onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
              className="flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-[13px] text-sidebar-muted transition-colors hover:bg-accent/60 hover:text-foreground"
            >
              {theme === "dark" ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
              <span>{theme === "dark" ? "Light Mode" : "Dark Mode"}</span>
            </button>
            <button
              type="button"
              onClick={handleLogout}
              className="flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-[13px] text-sidebar-muted transition-colors hover:bg-accent/60 hover:text-foreground"
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
      <div className="md:hidden">
        {mobileOpen && (
          <>
            <div
              className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm"
              onClick={onMobileClose}
              aria-hidden="true"
            />
            <div className="fixed inset-y-0 left-0 z-50 w-60 shadow-2xl">{sidebarContent}</div>
          </>
        )}
      </div>
      <div className="hidden md:block">{sidebarContent}</div>
      <OnboardNodeDialog open={onboardOpen} onOpenChange={setOnboardOpen} />
    </>
  );
}
