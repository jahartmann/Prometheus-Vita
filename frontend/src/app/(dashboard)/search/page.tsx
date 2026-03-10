"use client";

import { useState, useEffect, useMemo, useCallback, useRef } from "react";
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
  Settings,
  Zap,
  ArrowRight,
  Clock,
  X,
  Sparkles,
  AlertTriangle,
  Filter,
  Tag,
  LayoutGrid,
  List,
  TrendingUp,
  Cpu,
  MemoryStick,
  Globe,
  Box,
  BarChart3,
  FileText,
  ShieldCheck,
  GitCompare,
  GitBranch,
  Disc,
  Lightbulb,
} from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useNodeStore } from "@/stores/node-store";

type SearchCategory =
  | "all"
  | "nodes"
  | "vms"
  | "navigation"
  | "settings"
  | "alerts";

interface SearchResult {
  id: string;
  category: Exclude<SearchCategory, "all">;
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  subtitle?: string;
  description?: string;
  href: string;
  tags?: string[];
  status?: "online" | "offline" | "running" | "stopped" | "warning";
  metrics?: { cpu?: number; ram?: number };
}

const categoryConfig: Record<
  Exclude<SearchCategory, "all">,
  {
    label: string;
    icon: React.ComponentType<{ className?: string }>;
    color: string;
  }
> = {
  nodes: {
    label: "Nodes",
    icon: Server,
    color: "bg-blue-500/10 text-blue-500 border-blue-500/20",
  },
  vms: {
    label: "VMs & Container",
    icon: Monitor,
    color: "bg-purple-500/10 text-purple-500 border-purple-500/20",
  },
  navigation: {
    label: "Seiten",
    icon: LayoutGrid,
    color: "bg-green-500/10 text-green-500 border-green-500/20",
  },
  settings: {
    label: "Einstellungen",
    icon: Settings,
    color: "bg-amber-500/10 text-amber-500 border-amber-500/20",
  },
  alerts: {
    label: "Alerts",
    icon: AlertTriangle,
    color: "bg-red-500/10 text-red-500 border-red-500/20",
  },
};

const RECENT_SEARCHES_KEY = "prometheus-recent-searches";

export default function SearchPage() {
  const [query, setQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  const [activeCategory, setActiveCategory] = useState<SearchCategory>("all");
  const [recentSearches, setRecentSearches] = useState<string[]>([]);
  const [viewMode, setViewMode] = useState<"grid" | "list">("list");
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const inputRef = useRef<HTMLInputElement>(null);
  const resultsRef = useRef<HTMLDivElement>(null);
  const router = useRouter();
  const { nodes, nodeVMs } = useNodeStore();

  // Debounce search query
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedQuery(query), 200);
    return () => clearTimeout(timer);
  }, [query]);

  // Load recent searches from localStorage
  useEffect(() => {
    try {
      const stored = localStorage.getItem(RECENT_SEARCHES_KEY);
      if (stored) setRecentSearches(JSON.parse(stored));
    } catch {
      // ignore
    }
  }, []);

  // Focus input on mount
  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  // Reset selected index when query or category changes
  useEffect(() => {
    setSelectedIndex(-1);
  }, [debouncedQuery, activeCategory]);

  const saveRecentSearch = useCallback((term: string) => {
    if (!term.trim()) return;
    setRecentSearches((prev) => {
      const next = [term, ...prev.filter((s) => s !== term)].slice(0, 10);
      try {
        localStorage.setItem(RECENT_SEARCHES_KEY, JSON.stringify(next));
      } catch {
        // ignore
      }
      return next;
    });
  }, []);

  const clearRecentSearches = useCallback(() => {
    setRecentSearches([]);
    try {
      localStorage.removeItem(RECENT_SEARCHES_KEY);
    } catch {
      // ignore
    }
  }, []);

  const removeRecentSearch = useCallback((term: string) => {
    setRecentSearches((prev) => {
      const next = prev.filter((s) => s !== term);
      try {
        localStorage.setItem(RECENT_SEARCHES_KEY, JSON.stringify(next));
      } catch {
        // ignore
      }
      return next;
    });
  }, []);

  // Build all searchable items
  const allItems = useMemo(() => {
    const items: SearchResult[] = [];

    // Nodes
    nodes.forEach((node) => {
      items.push({
        id: `node-${node.id}`,
        category: "nodes",
        icon: Server,
        title: node.name,
        subtitle: `${node.type.toUpperCase()} - ${node.hostname}:${node.port}`,
        href: `/nodes/${node.id}/vms`,
        tags: [node.hostname, node.type, node.is_online ? "online" : "offline"],
        status: node.is_online ? "online" : "offline",
      });

      // VMs for this node
      const vms = nodeVMs[node.id];
      if (vms) {
        vms.forEach((vm) => {
          items.push({
            id: `vm-${node.id}-${vm.vmid}`,
            category: "vms",
            icon: vm.type === "lxc" ? Box : Monitor,
            title: vm.name || `VM ${vm.vmid}`,
            subtitle: `${node.name} - VMID ${vm.vmid} - ${vm.type.toUpperCase()}`,
            href: `/nodes/${node.id}/vms`,
            tags: [
              vm.type,
              vm.status,
              String(vm.vmid),
              node.name,
            ],
            status: vm.status === "running" ? "running" : "stopped",
            metrics: {
              cpu: typeof vm.cpu_usage === "number" ? Math.round(vm.cpu_usage * 100) : undefined,
              ram:
                typeof vm.memory_used === "number" && typeof vm.memory_total === "number" && vm.memory_total > 0
                  ? Math.round((vm.memory_used / vm.memory_total) * 100)
                  : undefined,
            },
          });
        });
      }
    });

    // Navigation pages
    const pages: Omit<SearchResult, "id" | "category">[] = [
      {
        title: "Dashboard",
        subtitle: "Uebersicht aller Nodes und KPIs",
        href: "/",
        icon: LayoutGrid,
        tags: ["home", "start", "uebersicht", "dashboard"],
      },
      {
        title: "Monitoring",
        subtitle: "Metriken, CPU, RAM, Netzwerk",
        href: "/monitoring",
        icon: BarChart3,
        tags: ["metriken", "cpu", "ram", "auslastung", "monitoring"],
      },
      {
        title: "Backups",
        subtitle: "Konfigurationsbackups verwalten",
        href: "/backups",
        icon: Archive,
        tags: ["sicherung", "backup", "restore"],
      },
      {
        title: "Morning Briefing",
        subtitle: "KI-generierte Zusammenfassung",
        href: "/briefing",
        icon: TrendingUp,
        tags: ["morgen", "zusammenfassung", "ki", "briefing"],
      },
      {
        title: "Disaster Recovery",
        subtitle: "Notfallplaene und Readiness",
        href: "/disaster-recovery",
        icon: Shield,
        tags: ["notfall", "recovery", "disaster"],
      },
      {
        title: "Topologie",
        subtitle: "Netzwerk-Topologie visualisieren",
        href: "/topology",
        icon: Globe,
        tags: ["netzwerk", "karte", "verbindungen", "topologie"],
      },
      {
        title: "Migrationen",
        subtitle: "VM-Migrationen verwalten",
        href: "/migrations",
        icon: Box,
        tags: ["migration", "verschieben"],
      },
      {
        title: "Empfehlungen",
        subtitle: "Ressourcen-Optimierungen",
        href: "/recommendations",
        icon: Lightbulb,
        tags: ["optimierung", "rightsizing", "empfehlungen"],
      },
      {
        title: "Reflex-Regeln",
        subtitle: "Automatische Reaktionen",
        href: "/reflex",
        icon: Zap,
        tags: ["regeln", "automatisch", "trigger", "reflex"],
      },
      {
        title: "AI Chat",
        subtitle: "KI-Assistent fuer Infrastruktur",
        href: "/chat",
        icon: Sparkles,
        tags: ["ki", "ai", "assistent", "frage", "chat"],
      },
      {
        title: "Speicher",
        subtitle: "Storage-Verwaltung",
        href: "/storage",
        icon: HardDrive,
        tags: ["speicher", "storage", "festplatte"],
      },
      {
        title: "Sicherheit",
        subtitle: "Security-Uebersicht und Events",
        href: "/security",
        icon: ShieldCheck,
        tags: ["sicherheit", "security", "firewall"],
      },
      {
        title: "Alerts",
        subtitle: "Warnungen und Benachrichtigungen",
        href: "/alerts",
        icon: AlertTriangle,
        tags: ["alerts", "warnung", "alarm"],
      },
      {
        title: "Drift-Erkennung",
        subtitle: "Konfigurationsabweichungen erkennen",
        href: "/drift",
        icon: GitCompare,
        tags: ["drift", "konfiguration", "abweichung"],
      },
      {
        title: "Tags",
        subtitle: "Ressourcen taggen und organisieren",
        href: "/tags",
        icon: Tag,
        tags: ["tag", "label", "organisation"],
      },
      {
        title: "ISOs & Vorlagen",
        subtitle: "ISO-Images und Templates verwalten",
        href: "/isos",
        icon: Disc,
        tags: ["iso", "template", "vorlage", "image"],
      },
    ];
    pages.forEach((p) => {
      items.push({
        ...p,
        id: `nav-${p.href}`,
        category: "navigation",
      });
    });

    // Settings
    const settings: Omit<SearchResult, "id" | "category" | "icon">[] = [
      {
        title: "Benutzerverwaltung",
        href: "/settings/users",
        tags: ["benutzer", "user", "rechte", "verwaltung"],
      },
      {
        title: "Node-Verwaltung",
        href: "/settings/nodes",
        tags: ["node", "hinzufuegen", "server"],
      },
      {
        title: "API-Tokens",
        href: "/settings/api-tokens",
        tags: ["api", "token", "key", "schluessel"],
      },
      {
        title: "SSH-Keys",
        href: "/settings/ssh-keys",
        tags: ["ssh", "schluessel", "key"],
      },
      {
        title: "Benachrichtigungen",
        href: "/settings/notifications",
        tags: ["notification", "telegram", "email", "benachrichtigung"],
      },
      {
        title: "KI-Einstellungen",
        href: "/settings/agent",
        tags: ["ki", "ai", "modell", "llm", "agent"],
      },
      {
        title: "Audit-Log",
        href: "/settings/audit-log",
        tags: ["audit", "log", "protokoll"],
      },
      {
        title: "Sicherheit",
        subtitle: "Login-Einstellungen",
        href: "/settings/security",
        tags: ["sicherheit", "security", "login"],
      },
      {
        title: "Passwort-Richtlinie",
        href: "/settings/password-policy",
        tags: ["passwort", "policy", "richtlinie"],
      },
      {
        title: "Environments",
        href: "/settings/environments",
        tags: ["environment", "umgebung"],
      },
      {
        title: "Wissensbasis",
        href: "/settings/brain",
        tags: ["wissen", "brain", "knowledge", "wissensbasis"],
      },
      {
        title: "Tags-Einstellungen",
        href: "/settings/tags",
        tags: ["tag", "label", "einstellungen"],
      },
    ];
    settings.forEach((s) => {
      items.push({
        ...s,
        id: `settings-${s.href}`,
        category: "settings",
        icon: Settings,
      });
    });

    return items;
  }, [nodes, nodeVMs]);

  // Scoring function for search relevance
  const scoreMatch = useCallback(
    (item: SearchResult, q: string): number => {
      const lower = q.toLowerCase();
      const title = item.title.toLowerCase();
      const subtitle = (item.subtitle || "").toLowerCase();
      const tags = (item.tags || []).join(" ").toLowerCase();

      // Exact match
      if (title === lower) return 100;
      // Starts with
      if (title.startsWith(lower)) return 90;
      // Word boundary match in title
      if (
        title.split(/[\s\-_]+/).some((word) => word.startsWith(lower))
      )
        return 80;
      // Contains in title
      if (title.includes(lower)) return 70;
      // Starts with in subtitle
      if (subtitle.startsWith(lower)) return 60;
      // Contains in subtitle
      if (subtitle.includes(lower)) return 50;
      // Tag match
      if (
        (item.tags || []).some((t) => t.toLowerCase().startsWith(lower))
      )
        return 45;
      // Contains in tags
      if (tags.includes(lower)) return 40;

      // Multi-word search: all words must match somewhere
      const words = lower.split(/\s+/).filter(Boolean);
      if (words.length > 1) {
        const combined = `${title} ${subtitle} ${tags}`;
        if (words.every((w) => combined.includes(w))) return 35;
      }

      // Fuzzy matching on title
      let qi = 0;
      for (let i = 0; i < title.length && qi < lower.length; i++) {
        if (title[i] === lower[qi]) qi++;
      }
      if (qi === lower.length) return 20;

      return 0;
    },
    []
  );

  // Filtered and scored results
  const results = useMemo(() => {
    let items = allItems;
    if (activeCategory !== "all") {
      items = items.filter((i) => i.category === activeCategory);
    }
    if (!debouncedQuery.trim()) return items;

    return items
      .map((item) => ({ item, score: scoreMatch(item, debouncedQuery) }))
      .filter(({ score }) => score > 0)
      .sort((a, b) => b.score - a.score)
      .map(({ item }) => item);
  }, [debouncedQuery, activeCategory, allItems, scoreMatch]);

  // Group results by category
  const grouped = useMemo(() => {
    const groups: Record<string, SearchResult[]> = {};
    for (const item of results) {
      const cat = item.category;
      if (!groups[cat]) groups[cat] = [];
      groups[cat].push(item);
    }
    return Object.entries(groups).map(([category, items]) => ({
      category: category as Exclude<SearchCategory, "all">,
      items,
    }));
  }, [results]);

  // Flat list of all visible results for keyboard navigation
  const flatResults = useMemo(() => {
    return grouped.flatMap((g) => g.items);
  }, [grouped]);

  const handleSelect = useCallback(
    (item: SearchResult) => {
      if (query.trim()) {
        saveRecentSearch(query.trim());
      }
      router.push(item.href);
    },
    [query, saveRecentSearch, router]
  );

  const handleSearch = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      const target = selectedIndex >= 0 ? flatResults[selectedIndex] : flatResults[0];
      if (target) {
        handleSelect(target);
      }
    },
    [selectedIndex, flatResults, handleSelect]
  );

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setSelectedIndex((prev) =>
          prev < flatResults.length - 1 ? prev + 1 : 0
        );
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setSelectedIndex((prev) =>
          prev > 0 ? prev - 1 : flatResults.length - 1
        );
      } else if (e.key === "Escape") {
        if (query) {
          setQuery("");
        } else {
          inputRef.current?.blur();
        }
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [flatResults.length, query]);

  const statusColors: Record<string, string> = {
    online: "bg-green-500",
    running: "bg-green-500",
    offline: "bg-red-500",
    stopped: "bg-gray-400",
    warning: "bg-amber-500",
  };

  const statusLabels: Record<string, string> = {
    online: "Online",
    running: "Laeuft",
    offline: "Offline",
    stopped: "Gestoppt",
    warning: "Warnung",
  };

  const suggestions = [
    "Dashboard",
    "Monitoring",
    "Backups",
    "Netzwerk-Topologie",
    "Sicherheit",
    "Benachrichtigungen",
  ];

  // Track the flat index for highlighting
  let flatIndex = -1;

  return (
    <div className="mx-auto max-w-4xl space-y-6 pb-12">
      {/* Header */}
      <div className="space-y-2">
        <h1 className="text-3xl font-bold tracking-tight">Suche</h1>
        <p className="text-muted-foreground">
          Durchsuche Nodes, VMs, Einstellungen und mehr.
        </p>
      </div>

      {/* Search Input */}
      <form onSubmit={handleSearch}>
        <div className="relative">
          <Search className="absolute left-4 top-1/2 h-5 w-5 -translate-y-1/2 text-muted-foreground" />
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Suche nach Nodes, VMs, Seiten, Einstellungen..."
            className="h-14 w-full rounded-xl border bg-card pl-12 pr-12 text-lg outline-none ring-ring placeholder:text-muted-foreground focus:ring-2"
            autoComplete="off"
            spellCheck={false}
          />
          {query && (
            <button
              type="button"
              onClick={() => {
                setQuery("");
                inputRef.current?.focus();
              }}
              className="absolute right-4 top-1/2 -translate-y-1/2 rounded-md p-1 text-muted-foreground hover:bg-accent"
            >
              <X className="h-4 w-4" />
            </button>
          )}
        </div>
      </form>

      {/* Category Filter Chips */}
      <div className="flex flex-wrap items-center gap-2">
        <Button
          variant={activeCategory === "all" ? "default" : "outline"}
          size="sm"
          onClick={() => setActiveCategory("all")}
          className="rounded-full"
        >
          <Filter className="mr-1.5 h-3.5 w-3.5" />
          Alle
        </Button>
        {(
          Object.entries(categoryConfig) as [
            Exclude<SearchCategory, "all">,
            (typeof categoryConfig)[keyof typeof categoryConfig],
          ][]
        ).map(([key, config]) => {
          const Icon = config.icon;
          const count = results.filter((r) => r.category === key).length;
          return (
            <Button
              key={key}
              variant={activeCategory === key ? "default" : "outline"}
              size="sm"
              onClick={() => setActiveCategory(key)}
              className="rounded-full"
            >
              <Icon className="mr-1.5 h-3.5 w-3.5" />
              {config.label}
              {query.trim() && count > 0 && (
                <Badge
                  variant="secondary"
                  className="ml-1.5 h-5 px-1.5 text-[10px]"
                >
                  {count}
                </Badge>
              )}
            </Button>
          );
        })}

        <div className="ml-auto flex gap-1">
          <Button
            variant={viewMode === "list" ? "secondary" : "ghost"}
            size="icon"
            className="h-8 w-8"
            onClick={() => setViewMode("list")}
            title="Listenansicht"
          >
            <List className="h-4 w-4" />
          </Button>
          <Button
            variant={viewMode === "grid" ? "secondary" : "ghost"}
            size="icon"
            className="h-8 w-8"
            onClick={() => setViewMode("grid")}
            title="Rasteransicht"
          >
            <LayoutGrid className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Empty query state: show recent searches + suggestions */}
      {!query.trim() && (
        <div className="space-y-6">
          {/* Recent Searches */}
          {recentSearches.length > 0 && (
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="flex items-center gap-2 text-sm font-semibold text-muted-foreground">
                  <Clock className="h-4 w-4" />
                  Letzte Suchen
                </h3>
                <button
                  onClick={clearRecentSearches}
                  className="text-xs text-muted-foreground hover:text-foreground transition-colors"
                >
                  Alle loeschen
                </button>
              </div>
              <div className="flex flex-wrap gap-2">
                {recentSearches.map((term, i) => (
                  <span
                    key={i}
                    className="group inline-flex items-center gap-1 rounded-full border px-3 py-1 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                  >
                    <button
                      onClick={() => setQuery(term)}
                      className="hover:text-foreground"
                    >
                      {term}
                    </button>
                    <button
                      onClick={() => removeRecentSearch(term)}
                      className="ml-0.5 rounded-full p-0.5 opacity-0 transition-opacity group-hover:opacity-100 hover:bg-muted"
                    >
                      <X className="h-3 w-3" />
                    </button>
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* Suggestions */}
          <div className="space-y-3">
            <h3 className="flex items-center gap-2 text-sm font-semibold text-muted-foreground">
              <Sparkles className="h-4 w-4" />
              Vorschlaege
            </h3>
            <div className="flex flex-wrap gap-2">
              {suggestions.map((s, i) => (
                <button
                  key={i}
                  onClick={() => setQuery(s)}
                  className="rounded-full border border-dashed px-3 py-1 text-sm text-muted-foreground transition-colors hover:border-primary hover:bg-primary/5 hover:text-foreground"
                >
                  {s}
                </button>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Search Results */}
      {query.trim() && results.length === 0 ? (
        <div className="flex flex-col items-center gap-3 py-16 text-center">
          <Search className="h-12 w-12 text-muted-foreground/30" />
          <p className="text-lg font-medium">Keine Ergebnisse</p>
          <p className="text-sm text-muted-foreground">
            Keine Treffer fuer &ldquo;{query}&rdquo;. Versuche andere
            Suchbegriffe.
          </p>
        </div>
      ) : query.trim() ? (
        <div ref={resultsRef} className="space-y-6">
          {/* Result count */}
          <p className="text-sm text-muted-foreground">
            {results.length} {results.length === 1 ? "Ergebnis" : "Ergebnisse"}{" "}
            fuer &ldquo;{debouncedQuery}&rdquo;
          </p>

          {grouped.map((group) => {
            const config = categoryConfig[group.category];
            if (!config) return null;
            const GroupIcon = config.icon;
            return (
              <div key={group.category} className="space-y-3">
                <h3 className="flex items-center gap-2 text-sm font-semibold text-muted-foreground">
                  <GroupIcon className="h-4 w-4" />
                  {config.label}
                  <Badge variant="secondary" className="ml-1 text-[10px]">
                    {group.items.length}
                  </Badge>
                </h3>
                <div
                  className={cn(
                    viewMode === "grid"
                      ? "grid gap-3 sm:grid-cols-2 lg:grid-cols-3"
                      : "space-y-2"
                  )}
                >
                  {group.items.map((item) => {
                    flatIndex++;
                    const isSelected = flatIndex === selectedIndex;
                    const ItemIcon = item.icon;
                    return (
                      <button
                        key={item.id}
                        onClick={() => handleSelect(item)}
                        data-selected={isSelected || undefined}
                        className={cn(
                          "group w-full text-left transition-all",
                          viewMode === "grid"
                            ? "rounded-xl border bg-card p-4 hover:shadow-md hover:border-primary/30"
                            : "flex items-center gap-3 rounded-lg border bg-card px-4 py-3 hover:shadow-sm hover:border-primary/30",
                          isSelected &&
                            "ring-2 ring-primary border-primary/30 shadow-sm"
                        )}
                      >
                        <div
                          className={cn(
                            "flex items-center justify-center rounded-lg shrink-0",
                            viewMode === "grid"
                              ? "mb-3 h-10 w-10"
                              : "h-9 w-9",
                            config.color
                          )}
                        >
                          <ItemIcon
                            className={
                              viewMode === "grid" ? "h-5 w-5" : "h-4 w-4"
                            }
                          />
                        </div>
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center gap-2">
                            <span
                              className={cn(
                                "font-medium truncate",
                                viewMode === "grid" ? "text-base" : "text-sm"
                              )}
                            >
                              {item.title}
                            </span>
                            {item.status && (
                              <Badge
                                variant="outline"
                                className={cn(
                                  "shrink-0 gap-1.5 text-[10px] px-1.5 py-0",
                                  item.status === "online" || item.status === "running"
                                    ? "border-green-500/30 text-green-600"
                                    : item.status === "warning"
                                      ? "border-amber-500/30 text-amber-600"
                                      : "border-red-500/30 text-red-500"
                                )}
                              >
                                <span
                                  className={cn(
                                    "h-1.5 w-1.5 rounded-full",
                                    statusColors[item.status]
                                  )}
                                />
                                {statusLabels[item.status]}
                              </Badge>
                            )}
                          </div>
                          {item.subtitle && (
                            <p className="mt-0.5 truncate text-xs text-muted-foreground">
                              {item.subtitle}
                            </p>
                          )}
                          {/* Metrics preview for VMs */}
                          {item.metrics &&
                            (item.metrics.cpu !== undefined ||
                              item.metrics.ram !== undefined) && (
                              <div className="mt-1.5 flex items-center gap-3 text-[10px] text-muted-foreground">
                                {item.metrics.cpu !== undefined && (
                                  <span className="flex items-center gap-1">
                                    <Cpu className="h-3 w-3" />
                                    CPU {item.metrics.cpu}%
                                  </span>
                                )}
                                {item.metrics.ram !== undefined && (
                                  <span className="flex items-center gap-1">
                                    <MemoryStick className="h-3 w-3" />
                                    RAM {item.metrics.ram}%
                                  </span>
                                )}
                              </div>
                            )}
                        </div>
                        <ArrowRight className="h-4 w-4 shrink-0 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100" />
                      </button>
                    );
                  })}
                </div>
              </div>
            );
          })}
        </div>
      ) : null}

      {/* Keyboard hints */}
      <div className="flex items-center justify-center gap-4 py-4 text-[11px] text-muted-foreground">
        <span className="flex items-center gap-1">
          <kbd className="rounded border bg-muted px-1.5 py-0.5 font-mono text-[10px]">
            Ctrl K
          </kbd>
          Schnellsuche
        </span>
        <span className="flex items-center gap-1">
          <kbd className="rounded border bg-muted px-1.5 py-0.5 font-mono text-[10px]">
            &uarr;&darr;
          </kbd>
          Navigieren
        </span>
        <span className="flex items-center gap-1">
          <kbd className="rounded border bg-muted px-1.5 py-0.5 font-mono text-[10px]">
            Enter
          </kbd>
          Oeffnen
        </span>
        <span className="flex items-center gap-1">
          <kbd className="rounded border bg-muted px-1.5 py-0.5 font-mono text-[10px]">
            Esc
          </kbd>
          Zuruecksetzen
        </span>
      </div>
    </div>
  );
}
