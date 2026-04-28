import { Activity, Boxes, Bell, Server, ShieldCheck, Wrench, ListChecks, Bot } from "lucide-react";
import { Link } from "@tanstack/react-router";
import { cn } from "@/lib/utils";

const navItems = [
  { to: "/", label: "Lagezentrum", icon: Activity },
  { to: "/hosts", label: "Hosts", icon: Server },
  { to: "/vms", label: "VMs", icon: Boxes },
  { to: "/migrations", label: "Migrationen", icon: Wrench },
  { to: "/backups", label: "Backups", icon: ShieldCheck },
  { to: "/notifications", label: "Notifications", icon: Bell },
  { to: "/agent", label: "Agent", icon: Bot },
  { to: "/tasks", label: "Task-Center", icon: ListChecks },
];

export function Sidebar({ className }: { className?: string }) {
  return (
    <aside className={cn("flex h-full w-60 shrink-0 flex-col gap-2 border-r border-border bg-card p-3", className)}>
      <div className="px-2 py-3">
        <p className="text-sm font-semibold tracking-tight">Prometheus V2</p>
        <p className="text-[10px] uppercase tracking-wide text-muted-foreground">Operations Cockpit</p>
      </div>
      <nav className="flex flex-col gap-0.5">
        {navItems.map((item) => {
          const Icon = item.icon;
          return (
            <Link
              key={item.to}
              to={item.to}
              activeProps={{ className: "bg-accent text-accent-foreground" }}
              inactiveProps={{ className: "text-muted-foreground hover:bg-muted hover:text-foreground" }}
              className="flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition"
            >
              <Icon className="h-4 w-4" />
              <span>{item.label}</span>
            </Link>
          );
        })}
      </nav>
    </aside>
  );
}
