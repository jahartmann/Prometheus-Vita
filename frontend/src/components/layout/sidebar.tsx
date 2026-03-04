"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
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
  Plus,
  Brain,
  Bell,
  Archive,
  Shield,
  Disc,
} from "lucide-react";
import { cn } from "@/lib/utils";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
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

const mainNavItems: NavItem[] = [
  { label: "Dashboard", href: "/", icon: LayoutDashboard },
];

const featureNavItems: NavItem[] = [
  { label: "Backups", href: "/backups", icon: Archive, matchPrefix: "/backups" },
  { label: "Disaster Recovery", href: "/disaster-recovery", icon: Shield, matchPrefix: "/disaster-recovery" },
];

const bottomNavItems: NavItem[] = [
  { label: "Benachrichtigungen", href: "/settings/notifications", icon: Bell, matchPrefix: "/settings/notifications" },
  { label: "KI-Assistent", href: "/chat", icon: Brain, matchPrefix: "/chat" },
  {
    label: "Einstellungen",
    href: "/settings/nodes",
    icon: Settings,
    matchPrefix: "/settings",
    excludePrefix: ["/settings/notifications"],
  },
];

interface NodeSubItem {
  label: string;
  path: string;
  icon: React.ComponentType<{ className?: string }>;
}

const nodeSubItems: NodeSubItem[] = [
  { label: "VMs & Container", path: "vms", icon: Monitor },
  { label: "Storage", path: "storage", icon: HardDrive },
  { label: "Netzwerk", path: "network", icon: Network },
  { label: "Backups", path: "backups", icon: FolderArchive },
  { label: "ISOs & Templates", path: "iso-templates", icon: Disc },
  { label: "Monitoring", path: "monitoring", icon: BarChart3 },
];

interface SidebarProps {
  collapsed?: boolean;
}

export function Sidebar({ collapsed = false }: SidebarProps) {
  const pathname = usePathname();
  const { nodes, fetchNodes } = useNodeStore();
  const [nodesOpen, setNodesOpen] = useState(pathname.startsWith("/nodes"));
  const [openNodes, setOpenNodes] = useState<Record<string, boolean>>({});
  const [onboardOpen, setOnboardOpen] = useState(false);

  useEffect(() => {
    const token = useAuthStore.getState().accessToken;
    if (token) {
      fetchNodes();
    }
    const unsub = useAuthStore.subscribe((state) => {
      if (state.accessToken) {
        fetchNodes();
      }
    });
    return () => unsub();
  }, [fetchNodes]);

  useEffect(() => {
    if (pathname.startsWith("/nodes/")) {
      const segments = pathname.split("/");
      const activeNodeId = segments[2];
      if (activeNodeId) {
        setNodesOpen(true);
        setOpenNodes((prev) => ({ ...prev, [activeNodeId]: true }));
      }
    }
  }, [pathname]);

  const isActive = (href: string, matchPrefix?: string, excludePrefix?: string[]) => {
    if (matchPrefix) {
      if (excludePrefix?.some((ex) => pathname.startsWith(ex))) {
        return false;
      }
      return pathname.startsWith(matchPrefix);
    }
    return pathname === href;
  };

  const isNodeActive = (nodeId: string) => {
    return pathname.startsWith(`/nodes/${nodeId}`);
  };

  const isNodeSubActive = (nodeId: string, subPath: string) => {
    return pathname === `/nodes/${nodeId}/${subPath}`;
  };

  const toggleNode = (nodeId: string) => {
    setOpenNodes((prev) => ({ ...prev, [nodeId]: !prev[nodeId] }));
  };

  const renderNavLink = (item: NavItem) => {
    const active = isActive(item.href, item.matchPrefix, item.excludePrefix);
    const Icon = item.icon;

    const link = (
      <Link
        key={item.href}
        href={item.href}
        className={cn(
          "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
          active
            ? "bg-primary/10 text-primary"
            : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
        )}
      >
        <Icon className="h-4 w-4 shrink-0" />
        {!collapsed && <span>{item.label}</span>}
      </Link>
    );

    if (collapsed) {
      return (
        <Tooltip key={item.href} delayDuration={0}>
          <TooltipTrigger asChild>{link}</TooltipTrigger>
          <TooltipContent side="right">{item.label}</TooltipContent>
        </Tooltip>
      );
    }

    return link;
  };

  if (collapsed) {
    const allCollapsedItems: NavItem[] = [
      ...mainNavItems,
      { label: "Server", href: "/nodes", icon: Server, matchPrefix: "/nodes" },
      ...featureNavItems,
      ...bottomNavItems,
    ];

    return (
      <>
        <aside className="flex h-screen w-16 flex-col border-r bg-card/80 backdrop-blur-sm transition-all duration-300">
          <div className="flex h-14 items-center justify-center border-b">
            <Flame className="h-6 w-6 shrink-0 text-primary" />
          </div>
          <nav className="flex-1 space-y-1 p-2">
            {allCollapsedItems.map((item) => renderNavLink(item))}
          </nav>
        </aside>
        <OnboardNodeDialog open={onboardOpen} onOpenChange={setOnboardOpen} />
      </>
    );
  }

  return (
    <>
      <aside className="flex h-screen w-60 flex-col border-r bg-card/80 backdrop-blur-sm transition-all duration-300">
        <div className="flex h-14 items-center gap-2 border-b px-4">
          <Flame className="h-6 w-6 shrink-0 text-primary" />
          <span className="text-lg font-bold tracking-tight">Prometheus</span>
        </div>

        <nav className="flex-1 space-y-1 overflow-y-auto p-2">
          {mainNavItems.map((item) => renderNavLink(item))}

          {/* Server (Nodes) Tree */}
          <Collapsible open={nodesOpen} onOpenChange={setNodesOpen}>
            <CollapsibleTrigger
              className={cn(
                "flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                pathname.startsWith("/nodes")
                  ? "bg-primary/10 text-primary"
                  : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
              )}
            >
              <Server className="h-4 w-4 shrink-0" />
              <span className="flex-1 text-left">Server</span>
              {nodesOpen ? (
                <ChevronDown className="h-3.5 w-3.5 shrink-0" />
              ) : (
                <ChevronRight className="h-3.5 w-3.5 shrink-0" />
              )}
            </CollapsibleTrigger>

            <CollapsibleContent className="ml-2 space-y-0.5 border-l border-muted pl-2">
              {nodes.map((node) => (
                <Collapsible
                  key={node.id}
                  open={openNodes[node.id] ?? false}
                  onOpenChange={() => toggleNode(node.id)}
                >
                  <CollapsibleTrigger
                    className={cn(
                      "flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-sm transition-colors",
                      isNodeActive(node.id)
                        ? "bg-primary/10 text-primary font-medium"
                        : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
                    )}
                  >
                    <span
                      className={cn(
                        "h-2 w-2 shrink-0 rounded-full",
                        node.is_online ? "bg-green-500" : "bg-red-500"
                      )}
                    />
                    <span className="flex-1 truncate text-left">{node.name}</span>
                    {openNodes[node.id] ? (
                      <ChevronDown className="h-3 w-3 shrink-0" />
                    ) : (
                      <ChevronRight className="h-3 w-3 shrink-0" />
                    )}
                  </CollapsibleTrigger>

                  <CollapsibleContent className="ml-3 space-y-0.5 border-l border-muted pl-2">
                    {nodeSubItems.map((sub) => {
                      const SubIcon = sub.icon;
                      const subActive = isNodeSubActive(node.id, sub.path);
                      return (
                        <Link
                          key={sub.path}
                          href={`/nodes/${node.id}/${sub.path}`}
                          className={cn(
                            "flex items-center gap-2 rounded-lg px-2 py-1 text-xs transition-colors",
                            subActive
                              ? "bg-primary/10 text-primary font-medium"
                              : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
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
                className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
              >
                <Plus className="h-3.5 w-3.5 shrink-0" />
                <span>Server hinzufuegen</span>
              </button>
            </CollapsibleContent>
          </Collapsible>

          {/* Feature Nav Items */}
          {featureNavItems.map((item) => renderNavLink(item))}

          <div className="my-2 border-t border-border" />

          {/* Bottom Nav Items */}
          {bottomNavItems.map((item) => renderNavLink(item))}
        </nav>
      </aside>

      <OnboardNodeDialog open={onboardOpen} onOpenChange={setOnboardOpen} />
    </>
  );
}
