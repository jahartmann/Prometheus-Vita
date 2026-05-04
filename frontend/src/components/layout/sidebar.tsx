"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useTheme } from "next-themes";
import {
  Activity,
  AlertTriangle,
  Archive,
  Bell,
  Bot,
  Brain,
  ChevronDown,
  Flame,
  GitBranch,
  GitCompare,
  HardDrive,
  HeartPulse,
  KeyRound,
  LayoutDashboard,
  Link2,
  ListChecks,
  LogOut,
  Moon,
  Network,
  Plus,
  RadioTower,
  Search,
  SearchCheck,
  Server,
  Settings,
  Shield,
  ShieldCheck,
  Sun,
  Tag,
  UserCog,
  Users,
  Workflow,
  Zap,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth-store";
import { useNodeStore } from "@/stores/node-store";
import { OnboardNodeDialog } from "@/components/nodes/onboard-node-dialog";

interface NavItem {
  label: string;
  href: string;
  icon: React.ComponentType<{ className?: string }>;
  matchPrefix?: string;
  keywords?: string;
}

interface WorkspaceNav {
  id: string;
  label: string;
  description: string;
  icon: React.ComponentType<{ className?: string }>;
  items: NavItem[];
}

const workspaces: WorkspaceNav[] = [
  {
    id: "lage",
    label: "Lage",
    description: "Status, Aufgaben, Alarme",
    icon: LayoutDashboard,
    items: [
      { label: "Lagezentrum", href: "/", icon: LayoutDashboard, keywords: "dashboard start cockpit" },
      { label: "Monitoring", href: "/monitoring", icon: Activity, matchPrefix: "/monitoring" },
      { label: "Aufgaben", href: "/task-center", icon: ListChecks, matchPrefix: "/task-center", keywords: "tasks todo timeline" },
      { label: "Alarme", href: "/alerts", icon: AlertTriangle, matchPrefix: "/alerts", keywords: "alerts warnungen" },
    ],
  },
  {
    id: "infra",
    label: "Infrastruktur",
    description: "Nodes, VMs, Netz, Speicher",
    icon: Server,
    items: [
      { label: "Nodes", href: "/nodes", icon: Server, matchPrefix: "/nodes" },
      { label: "Cluster", href: "/cluster", icon: RadioTower, matchPrefix: "/cluster" },
      { label: "Speicher", href: "/storage", icon: HardDrive, matchPrefix: "/storage" },
      { label: "Netzwerk", href: "/network", icon: Network, matchPrefix: "/network" },
      { label: "Topologie", href: "/topology", icon: Workflow, matchPrefix: "/topology" },
      { label: "ISOs & Vorlagen", href: "/isos", icon: Archive, matchPrefix: "/isos" },
    ],
  },
  {
    id: "schutz",
    label: "Schutz",
    description: "Sicherheit, Backup, Recovery",
    icon: ShieldCheck,
    items: [
      { label: "Sicherheit", href: "/security", icon: ShieldCheck, matchPrefix: "/security" },
      { label: "Backups", href: "/backups", icon: Archive, matchPrefix: "/backups" },
      { label: "Notfallplanung", href: "/disaster-recovery", icon: Shield, matchPrefix: "/disaster-recovery", keywords: "dr disaster recovery runbook" },
      { label: "Migrationen", href: "/migrations", icon: GitBranch, matchPrefix: "/migrations" },
      { label: "Berichte", href: "/reports", icon: Activity, matchPrefix: "/reports", keywords: "reports" },
    ],
  },
  {
    id: "automation",
    label: "KI & Regeln",
    description: "Assistent, Wissen, Reflex",
    icon: Bot,
    items: [
      { label: "KI-Chat", href: "/chat", icon: Bot, matchPrefix: "/chat" },
      { label: "Lagebrief", href: "/briefing", icon: Bell, matchPrefix: "/briefing", keywords: "briefing daily morning" },
      { label: "Empfehlungen", href: "/recommendations", icon: HeartPulse, matchPrefix: "/recommendations" },
      { label: "Drift", href: "/drift", icon: GitCompare, matchPrefix: "/drift" },
      { label: "Ursachenanalyse", href: "/root-cause", icon: SearchCheck, matchPrefix: "/root-cause", keywords: "root cause rca" },
      { label: "Reflex-Regeln", href: "/reflex", icon: Zap, matchPrefix: "/reflex" },
      { label: "Wissensgraph", href: "/knowledge-graph", icon: Brain, matchPrefix: "/knowledge-graph", keywords: "knowledge graph wissensbasis brain" },
      { label: "Abhängigkeiten", href: "/dependencies", icon: Link2, matchPrefix: "/dependencies" },
      { label: "VM-Gesundheit", href: "/health", icon: HeartPulse, matchPrefix: "/health" },
    ],
  },
  {
    id: "admin",
    label: "Admin",
    description: "Rechte, Tokens, System",
    icon: Settings,
    items: [
      { label: "Admin-Hub", href: "/settings", icon: Settings },
      { label: "Systemstatus", href: "/settings/system", icon: Activity, matchPrefix: "/settings/system" },
      { label: "Benutzer", href: "/settings/users", icon: Users, matchPrefix: "/settings/users" },
      { label: "Rollen & Rechte", href: "/settings/roles", icon: UserCog, matchPrefix: "/settings/roles" },
      { label: "API-Tokens", href: "/settings/api-tokens", icon: KeyRound, matchPrefix: "/settings/api-tokens" },
      { label: "Tags", href: "/settings/tags", icon: Tag, matchPrefix: "/settings/tags" },
    ],
  },
];

function matchesNavItem(pathname: string, item: NavItem): boolean {
  if (item.href === "/") return pathname === "/";
  if (item.matchPrefix) return pathname.startsWith(item.matchPrefix);
  return pathname === item.href;
}

function fuzzyMatch(query: string, item: NavItem): boolean {
  const q = query.trim().toLowerCase();
  if (!q) return true;
  return `${item.label} ${item.keywords ?? ""} ${item.href}`.toLowerCase().includes(q);
}

function workspaceForPath(pathname: string): string {
  const match = workspaces.find((workspace) =>
    workspace.items.some((item) => matchesNavItem(pathname, item))
  );
  return match?.id ?? "lage";
}

interface SidebarProps {
  mobileOpen?: boolean;
  onMobileClose?: () => void;
}

export function Sidebar({ mobileOpen = false, onMobileClose }: SidebarProps) {
  const pathname = usePathname();
  const router = useRouter();
  const { theme, setTheme } = useTheme();
  const { fetchNodes } = useNodeStore();
  const { user, logout } = useAuthStore();
  const [onboardOpen, setOnboardOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [openWorkspace, setOpenWorkspace] = useState(() => workspaceForPath(pathname));

  useEffect(() => {
    const token = useAuthStore.getState().accessToken;
    if (token) fetchNodes();
    const unsub = useAuthStore.subscribe((state) => {
      if (state.accessToken) fetchNodes();
    });
    return () => unsub();
  }, [fetchNodes]);

  useEffect(() => {
    setOpenWorkspace(workspaceForPath(pathname));
  }, [pathname]);

  const filteredWorkspaces = useMemo(() => {
    if (!search.trim()) return workspaces;
    return workspaces
      .map((workspace) => ({
        ...workspace,
        items: workspace.items.filter((item) => fuzzyMatch(search, item)),
      }))
      .filter((workspace) => workspace.items.length > 0);
  }, [search]);

  const handleLogout = () => {
    logout();
    router.push("/login");
  };

  const initials = user?.username ? user.username.slice(0, 2).toUpperCase() : "??";

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
            className="absolute bottom-1.5 left-0 top-1.5 w-0.5 rounded-r-full bg-primary"
          />
        )}
        <Icon
          className={cn(
            "h-4 w-4 shrink-0",
            active ? "text-foreground" : "text-sidebar-muted group-hover:text-foreground"
          )}
        />
        <span className="truncate">{item.label}</span>
      </Link>
    );
  };

  const renderWorkspace = (workspace: WorkspaceNav) => {
    const open = search.trim() ? true : openWorkspace === workspace.id;
    const active = workspaceForPath(pathname) === workspace.id;
    const Icon = workspace.icon;

    return (
      <div key={workspace.id} className="rounded-lg">
        <button
          type="button"
          onClick={() => setOpenWorkspace(workspace.id)}
          className={cn(
            "flex w-full items-center gap-2.5 rounded-md px-2.5 py-2 text-left transition-colors",
            active
              ? "bg-accent/55 text-foreground"
              : "text-sidebar-muted hover:bg-accent/35 hover:text-foreground"
          )}
        >
          <Icon className="h-4 w-4 shrink-0" />
          <span className="min-w-0 flex-1">
            <span className="block truncate text-[13px] font-medium">{workspace.label}</span>
            {!open && (
              <span className="block truncate text-[10.5px] text-sidebar-muted/75">
                {workspace.description}
              </span>
            )}
          </span>
          <ChevronDown
            className={cn("h-3.5 w-3.5 shrink-0 transition-transform", open ? "rotate-0" : "-rotate-90")}
          />
        </button>
        {open && (
          <div className="mt-1 space-y-0.5 border-l border-border/70 pl-2">
            {workspace.items.map(renderNavLink)}
            {workspace.id === "infra" && (
              <button
                type="button"
                onClick={() => setOnboardOpen(true)}
                className="flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-[13px] text-sidebar-muted transition-colors hover:bg-accent/45 hover:text-foreground"
              >
                <Plus className="h-4 w-4 shrink-0" />
                <span>Node hinzufügen</span>
              </button>
            )}
          </div>
        )}
      </div>
    );
  };

  const sidebarContent = (
    <aside
      className="flex h-screen w-60 flex-col border-r ops-divider bg-sidebar"
      role="navigation"
      aria-label="Hauptnavigation"
    >
      <div className="flex h-12 items-center gap-2.5 border-b border-border px-3.5">
        <div className="flex h-7 w-7 items-center justify-center rounded-md bg-primary text-primary-foreground shadow-sm">
          <Flame className="h-4 w-4 text-white" />
        </div>
        <div className="leading-tight">
          <div className="text-sm font-semibold tracking-tight">Prometheus</div>
          <div className="text-[10px] text-sidebar-muted">Operations</div>
        </div>
      </div>

      <div className="border-b border-border/70 px-2.5 py-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-sidebar-muted" />
          <input
            type="text"
            placeholder="Funktion suchen..."
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            className="h-7 w-full rounded-md border border-border/80 bg-background/65 pl-8 pr-2.5 text-[12px] outline-none transition-colors placeholder:text-sidebar-muted/70 focus:border-ring"
          />
        </div>
      </div>

      <nav className="flex-1 overflow-y-auto px-2 py-2.5">
        <div className="space-y-1">{filteredWorkspaces.map(renderWorkspace)}</div>
        {filteredWorkspaces.length === 0 && (
          <p className="px-3 py-6 text-center text-[12px] text-sidebar-muted">
            Keine Treffer für "{search}".
          </p>
        )}
      </nav>

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
              <span>{theme === "dark" ? "Heller Modus" : "Dunkler Modus"}</span>
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
