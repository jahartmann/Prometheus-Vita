"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  Server,
  Settings,
  Flame,
  Archive,
  Activity,
  Shield,
  MessageSquare,
  Package,
  TrendingDown,
  GitCompare,
} from "lucide-react";
import { cn } from "@/lib/utils";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

const navItems = [
  {
    label: "Dashboard",
    href: "/",
    icon: LayoutDashboard,
  },
  {
    label: "Nodes",
    href: "/nodes",
    icon: Server,
    matchPrefix: "/nodes",
  },
  {
    label: "Backups",
    href: "/backups",
    icon: Archive,
    matchPrefix: "/backups",
  },
  {
    label: "Monitoring",
    href: "/monitoring",
    icon: Activity,
    matchPrefix: "/monitoring",
  },
  {
    label: "Disaster Recovery",
    href: "/disaster-recovery",
    icon: Shield,
    matchPrefix: "/disaster-recovery",
  },
  {
    label: "Updates",
    href: "/updates",
    icon: Package,
    matchPrefix: "/updates",
  },
  {
    label: "Empfehlungen",
    href: "/recommendations",
    icon: TrendingDown,
    matchPrefix: "/recommendations",
  },
  {
    label: "Topologie",
    href: "/topology",
    icon: GitCompare,
    matchPrefix: "/topology",
  },
  {
    label: "AI Chat",
    href: "/chat",
    icon: MessageSquare,
    matchPrefix: "/chat",
  },
  {
    label: "Einstellungen",
    href: "/settings/nodes",
    icon: Settings,
    matchPrefix: "/settings",
  },
];

interface SidebarProps {
  collapsed?: boolean;
}

export function Sidebar({ collapsed = false }: SidebarProps) {
  const pathname = usePathname();

  const isActive = (item: (typeof navItems)[0]) => {
    if (item.matchPrefix) {
      return pathname.startsWith(item.matchPrefix);
    }
    return pathname === item.href;
  };

  return (
    <aside
      className={cn(
        "flex h-screen flex-col border-r bg-card transition-all duration-300",
        collapsed ? "w-16" : "w-60"
      )}
    >
      <div className="flex h-14 items-center gap-2 border-b px-4">
        <Flame className="h-6 w-6 shrink-0 text-primary" />
        {!collapsed && (
          <span className="text-lg font-bold tracking-tight">
            Prometheus
          </span>
        )}
      </div>

      <nav className="flex-1 space-y-1 p-2">
        {navItems.map((item) => {
          const active = isActive(item);
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
        })}
      </nav>
    </aside>
  );
}
