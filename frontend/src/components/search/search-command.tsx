"use client";

import { useEffect, useState, useMemo, useCallback, useRef } from "react";
import { useRouter } from "next/navigation";
import {
  Search,
  Server,
  Monitor,
  HardDrive,
  Network,
  Activity,
  Shield,
  Archive,
  Package,
  Settings,
  MessageSquare,
  TrendingDown,
  ArrowLeftRight,
  Map,
  Newspaper,
  LayoutDashboard,
  Cpu,
  MemoryStick,
  Clock,
  Zap,
  AlertTriangle,
  ArrowRight,
  Command,
  CornerDownLeft,
  Loader2,
  Sparkles,
  Hash,
  FolderArchive,
  BarChart3,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useNodeStore } from "@/stores/node-store";

// ------- Types -------
interface SearchResult {
  id: string;
  category: SearchCategory;
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  description?: string;
  href?: string;
  action?: () => void;
  tags?: string[];
  priority?: number;
}

type SearchCategory =
  | "navigation"
  | "nodes"
  | "vms"
  | "actions"
  | "settings"
  | "recent";

const categoryLabels: Record<SearchCategory, string> = {
  recent: "Zuletzt besucht",
  nodes: "Nodes",
  vms: "VMs & Container",
  navigation: "Navigation",
  actions: "Schnellaktionen",
  settings: "Einstellungen",
};

const categoryOrder: SearchCategory[] = [
  "recent",
  "nodes",
  "vms",
  "navigation",
  "actions",
  "settings",
];

// ------- Component -------
export function SearchCommand() {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [activeIndex, setActiveIndex] = useState(0);
  const [recentPaths, setRecentPaths] = useState<string[]>([]);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const router = useRouter();
  const { nodes } = useNodeStore();

  // Load recent paths from localStorage
  useEffect(() => {
    try {
      const stored = localStorage.getItem("prometheus-recent-paths");
      if (stored) setRecentPaths(JSON.parse(stored));
    } catch {}
  }, [open]);

  // Keyboard shortcut: Ctrl+K / Cmd+K
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "k") {
        e.preventDefault();
        setOpen((v) => !v);
      }
      if (e.key === "Escape") {
        setOpen(false);
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, []);

  // Focus input when opening
  useEffect(() => {
    if (open) {
      setQuery("");
      setActiveIndex(0);
      setTimeout(() => inputRef.current?.focus(), 50);
    }
  }, [open]);

  // Save recent path
  const saveRecent = useCallback((href: string) => {
    setRecentPaths((prev) => {
      const next = [href, ...prev.filter((p) => p !== href)].slice(0, 8);
      localStorage.setItem("prometheus-recent-paths", JSON.stringify(next));
      return next;
    });
  }, []);

  // Navigation items
  const navigationItems: SearchResult[] = useMemo(
    () => [
      {
        id: "nav-dashboard",
        category: "navigation" as SearchCategory,
        icon: LayoutDashboard,
        label: "Dashboard",
        description: "Übersicht aller Nodes und KPIs",
        href: "/",
        tags: ["home", "start", "übersicht", "kpi"],
      },
      {
        id: "nav-nodes",
        category: "navigation" as SearchCategory,
        icon: Server,
        label: "Nodes",
        description: "Alle Proxmox-Nodes verwalten",
        href: "/nodes",
        tags: ["server", "proxmox", "pve", "pbs"],
      },
      {
        id: "nav-backups",
        category: "navigation" as SearchCategory,
        icon: Archive,
        label: "Backups",
        description: "Konfigurationsbackups verwalten",
        href: "/backups",
        tags: ["sicherung", "backup", "restore"],
      },
      {
        id: "nav-monitoring",
        category: "navigation" as SearchCategory,
        icon: Activity,
        label: "Monitoring",
        description: "Metriken und Auslastung",
        href: "/monitoring",
        tags: ["cpu", "ram", "metriken", "auslastung", "grafik"],
      },
      {
        id: "nav-briefing",
        category: "navigation" as SearchCategory,
        icon: Newspaper,
        label: "Morning Briefing",
        description: "Tägliche Zusammenfassung",
        href: "/briefing",
        tags: ["morgen", "zusammenfassung", "bericht", "daily"],
      },
      {
        id: "nav-dr",
        category: "navigation" as SearchCategory,
        icon: Shield,
        label: "Disaster Recovery",
        description: "Notfallpläne und Readiness",
        href: "/disaster-recovery",
        tags: ["notfall", "wiederherstellung", "recovery", "runbook"],
      },
      {
        id: "nav-migrations",
        category: "navigation" as SearchCategory,
        icon: ArrowLeftRight,
        label: "Migrationen",
        description: "VM-Migrationen zwischen Nodes",
        href: "/migrations",
        tags: ["migration", "verschieben", "umzug", "vm"],
      },
      {
        id: "nav-updates",
        category: "navigation" as SearchCategory,
        icon: Package,
        label: "Updates",
        description: "Paket-Updates und Sicherheitsupdates",
        href: "/updates",
        tags: ["pakete", "update", "sicherheit", "apt"],
      },
      {
        id: "nav-recommendations",
        category: "navigation" as SearchCategory,
        icon: TrendingDown,
        label: "Empfehlungen",
        description: "Ressourcen-Optimierungen",
        href: "/recommendations",
        tags: ["optimierung", "ressourcen", "right-sizing"],
      },
      {
        id: "nav-topology",
        category: "navigation" as SearchCategory,
        icon: Map,
        label: "Topologie",
        description: "Netzwerk-Topologie anzeigen",
        href: "/topology",
        tags: ["netzwerk", "karte", "graph", "verbindungen"],
      },
      {
        id: "nav-chat",
        category: "navigation" as SearchCategory,
        icon: MessageSquare,
        label: "AI Chat",
        description: "KI-Assistent für Infrastrukturfragen",
        href: "/chat",
        tags: ["ki", "ai", "assistent", "frage"],
      },
    ],
    []
  );

  // Node items
  const nodeItems: SearchResult[] = useMemo(
    () =>
      nodes.map((node) => ({
        id: `node-${node.id}`,
        category: "nodes" as SearchCategory,
        icon: Server,
        label: node.name,
        description: `${node.type.toUpperCase()} - ${node.hostname}:${node.port} ${node.is_online ? "Online" : "Offline"}`,
        href: `/nodes/${node.id}/vms`,
        tags: [
          node.hostname,
          node.type,
          node.is_online ? "online" : "offline",
        ],
        priority: node.is_online ? 1 : 0,
      })),
    [nodes]
  );

  // Quick-action items
  const actionItems: SearchResult[] = useMemo(
    () => [
      {
        id: "action-add-node",
        category: "actions" as SearchCategory,
        icon: Zap,
        label: "Neuen Node hinzufügen",
        description: "Proxmox-Node onboarden",
        href: "/settings/nodes",
        tags: ["node", "hinzufügen", "neu", "onboard"],
      },
      {
        id: "action-backup-now",
        category: "actions" as SearchCategory,
        icon: FolderArchive,
        label: "Backup erstellen",
        description: "Manuelles Konfigurationsbackup starten",
        href: "/backups",
        tags: ["backup", "erstellen", "sicherung"],
      },
    ],
    []
  );

  // Settings items
  const settingsItems: SearchResult[] = useMemo(
    () => [
      {
        id: "settings-nodes",
        category: "settings" as SearchCategory,
        icon: Server,
        label: "Node-Verwaltung",
        href: "/settings/nodes",
        tags: ["node", "verwaltung"],
      },
      {
        id: "settings-users",
        category: "settings" as SearchCategory,
        icon: Settings,
        label: "Benutzerverwaltung",
        href: "/settings/users",
        tags: ["benutzer", "user", "rechte"],
      },
      {
        id: "settings-tags",
        category: "settings" as SearchCategory,
        icon: Hash,
        label: "Tags verwalten",
        href: "/settings/tags",
        tags: ["tag", "label", "kategorie"],
      },
      {
        id: "settings-api",
        category: "settings" as SearchCategory,
        icon: Zap,
        label: "API-Tokens",
        href: "/settings/api-tokens",
        tags: ["api", "token", "key", "schlüssel"],
      },
      {
        id: "settings-ssh",
        category: "settings" as SearchCategory,
        icon: Shield,
        label: "SSH-Keys",
        href: "/settings/ssh-keys",
        tags: ["ssh", "key", "schlüssel"],
      },
      {
        id: "settings-env",
        category: "settings" as SearchCategory,
        icon: Settings,
        label: "Environments",
        href: "/settings/environments",
        tags: ["environment", "umgebung", "env"],
      },
    ],
    []
  );

  // Recent items
  const recentItems: SearchResult[] = useMemo(() => {
    const allItems = [
      ...navigationItems,
      ...nodeItems,
      ...settingsItems,
    ];
    return recentPaths
      .map((path) => {
        const found = allItems.find((item) => item.href === path);
        if (!found) return null;
        return { ...found, id: `recent-${found.id}`, category: "recent" as SearchCategory };
      })
      .filter(Boolean) as SearchResult[];
  }, [recentPaths, navigationItems, nodeItems, settingsItems]);

  // All items
  const allItems = useMemo(
    () => [
      ...recentItems,
      ...nodeItems,
      ...navigationItems,
      ...actionItems,
      ...settingsItems,
    ],
    [recentItems, nodeItems, navigationItems, actionItems, settingsItems]
  );

  // Fuzzy match scoring
  const scoreMatch = useCallback((item: SearchResult, q: string): number => {
    const lower = q.toLowerCase();
    const label = item.label.toLowerCase();
    const desc = (item.description || "").toLowerCase();
    const tags = (item.tags || []).join(" ").toLowerCase();

    // Exact label match
    if (label === lower) return 100;
    // Label starts with query
    if (label.startsWith(lower)) return 90;
    // Label contains query
    if (label.includes(lower)) return 70;
    // Description contains query
    if (desc.includes(lower)) return 50;
    // Tags match
    if (tags.includes(lower)) return 40;

    // Fuzzy: every character of query appears in order in label
    let qi = 0;
    for (let i = 0; i < label.length && qi < lower.length; i++) {
      if (label[i] === lower[qi]) qi++;
    }
    if (qi === lower.length) return 20;

    // Check description fuzzy
    qi = 0;
    for (let i = 0; i < desc.length && qi < lower.length; i++) {
      if (desc[i] === lower[qi]) qi++;
    }
    if (qi === lower.length) return 10;

    return 0;
  }, []);

  // Filtered + scored results
  const results = useMemo(() => {
    if (!query.trim()) {
      // Show recents + top nav when no query
      return allItems.filter(
        (item) =>
          item.category === "recent" ||
          item.category === "navigation" ||
          item.category === "actions"
      );
    }

    return allItems
      .map((item) => ({ item, score: scoreMatch(item, query) }))
      .filter(({ score }) => score > 0)
      .sort((a, b) => b.score - a.score)
      .map(({ item }) => item);
  }, [query, allItems, scoreMatch]);

  // Group results by category
  const grouped = useMemo(() => {
    const groups: Record<string, SearchResult[]> = {};
    for (const item of results) {
      if (!groups[item.category]) groups[item.category] = [];
      groups[item.category].push(item);
    }
    return categoryOrder
      .filter((cat) => groups[cat]?.length)
      .map((cat) => ({ category: cat, items: groups[cat] }));
  }, [results]);

  // Flat list for keyboard navigation
  const flatResults = useMemo(
    () => grouped.flatMap((g) => g.items),
    [grouped]
  );

  // Reset active index when results change
  useEffect(() => {
    setActiveIndex(0);
  }, [query]);

  // Scroll active item into view
  useEffect(() => {
    const el = listRef.current?.querySelector(`[data-index="${activeIndex}"]`);
    el?.scrollIntoView({ block: "nearest" });
  }, [activeIndex]);

  // Execute a result
  const execute = useCallback(
    (item: SearchResult) => {
      if (item.action) {
        item.action();
      } else if (item.href) {
        saveRecent(item.href);
        router.push(item.href);
      }
      setOpen(false);
    },
    [router, saveRecent]
  );

  // Keyboard navigation
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      switch (e.key) {
        case "ArrowDown":
          e.preventDefault();
          setActiveIndex((i) => Math.min(i + 1, flatResults.length - 1));
          break;
        case "ArrowUp":
          e.preventDefault();
          setActiveIndex((i) => Math.max(i - 1, 0));
          break;
        case "Enter":
          e.preventDefault();
          if (flatResults[activeIndex]) {
            execute(flatResults[activeIndex]);
          }
          break;
        case "Escape":
          e.preventDefault();
          setOpen(false);
          break;
      }
    },
    [flatResults, activeIndex, execute]
  );

  if (!open) return null;

  let flatIndex = 0;

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 z-[100] bg-black/50 backdrop-blur-sm"
        onClick={() => setOpen(false)}
      />

      {/* Dialog */}
      <div className="fixed inset-x-0 top-[12%] z-[101] mx-auto w-full max-w-[640px] px-4">
        <div className="overflow-hidden rounded-xl border bg-popover shadow-2xl">
          {/* Search Input */}
          <div className="flex items-center gap-3 border-b px-4 py-3">
            <Search className="h-5 w-5 shrink-0 text-muted-foreground" />
            <input
              ref={inputRef}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Suche nach Nodes, VMs, Seiten, Aktionen..."
              className="flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
              autoComplete="off"
              spellCheck={false}
            />
            {query && (
              <button
                onClick={() => setQuery("")}
                className="rounded px-1.5 py-0.5 text-xs text-muted-foreground hover:bg-accent"
              >
                Löschen
              </button>
            )}
            <kbd className="hidden rounded border bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground sm:inline-block">
              ESC
            </kbd>
          </div>

          {/* Results */}
          <div
            ref={listRef}
            className="max-h-[400px] overflow-y-auto overscroll-contain p-2"
          >
            {flatResults.length === 0 && query.trim() && (
              <div className="flex flex-col items-center gap-2 py-8 text-center text-sm text-muted-foreground">
                <Search className="h-8 w-8 opacity-30" />
                <p>Keine Ergebnisse für &ldquo;{query}&rdquo;</p>
                <p className="text-xs">
                  Versuche andere Begriffe oder weniger spezifische Suche
                </p>
              </div>
            )}

            {grouped.map((group) => (
              <div key={group.category} className="mb-2">
                <div className="px-2 py-1.5 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                  {categoryLabels[group.category]}
                </div>
                {group.items.map((item) => {
                  const currentFlatIndex = flatIndex++;
                  const isActive = currentFlatIndex === activeIndex;
                  const Icon = item.icon;

                  return (
                    <button
                      key={item.id}
                      data-index={currentFlatIndex}
                      onClick={() => execute(item)}
                      onMouseEnter={() => setActiveIndex(currentFlatIndex)}
                      className={cn(
                        "flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left text-sm transition-colors",
                        isActive
                          ? "bg-accent text-accent-foreground"
                          : "text-foreground hover:bg-accent/50"
                      )}
                    >
                      <div
                        className={cn(
                          "flex h-8 w-8 shrink-0 items-center justify-center rounded-md",
                          isActive
                            ? "bg-primary/15 text-primary"
                            : "bg-muted text-muted-foreground"
                        )}
                      >
                        <Icon className="h-4 w-4" />
                      </div>
                      <div className="min-w-0 flex-1">
                        <div className="truncate font-medium">{item.label}</div>
                        {item.description && (
                          <div className="truncate text-xs text-muted-foreground">
                            {item.description}
                          </div>
                        )}
                      </div>
                      {isActive && (
                        <div className="flex shrink-0 items-center gap-1 text-xs text-muted-foreground">
                          <CornerDownLeft className="h-3 w-3" />
                          <span>Öffnen</span>
                        </div>
                      )}
                    </button>
                  );
                })}
              </div>
            ))}
          </div>

          {/* Footer */}
          <div className="flex items-center justify-between border-t px-4 py-2 text-[11px] text-muted-foreground">
            <div className="flex items-center gap-3">
              <span className="flex items-center gap-1">
                <kbd className="rounded border bg-muted px-1 py-0.5 font-mono text-[10px]">
                  ↑↓
                </kbd>
                Navigieren
              </span>
              <span className="flex items-center gap-1">
                <kbd className="rounded border bg-muted px-1 py-0.5 font-mono text-[10px]">
                  ↵
                </kbd>
                Öffnen
              </span>
              <span className="flex items-center gap-1">
                <kbd className="rounded border bg-muted px-1 py-0.5 font-mono text-[10px]">
                  esc
                </kbd>
                Schliessen
              </span>
            </div>
            <div className="flex items-center gap-1">
              <Sparkles className="h-3 w-3 text-primary" />
              <span>Intelligente Suche</span>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}

// Trigger button for the header
export function SearchTrigger() {
  const [open, setOpen] = useState(false);

  const handleClick = () => {
    // Dispatch Ctrl+K event to open search
    window.dispatchEvent(
      new KeyboardEvent("keydown", {
        key: "k",
        ctrlKey: true,
        bubbles: true,
      })
    );
  };

  return (
    <button
      onClick={handleClick}
      className="flex h-9 w-64 items-center gap-2 rounded-lg border bg-muted/50 px-3 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
    >
      <Search className="h-4 w-4 shrink-0" />
      <span className="flex-1 text-left">Suchen...</span>
      <kbd className="hidden rounded border bg-background px-1.5 py-0.5 text-[10px] font-medium sm:inline-block">
        Ctrl K
      </kbd>
    </button>
  );
}
