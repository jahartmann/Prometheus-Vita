"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Activity,
  Bell,
  Bot,
  Brain,
  ChevronDown,
  ClipboardList,
  DatabaseBackup,
  FolderLock,
  Globe,
  Key,
  Lock,
  Monitor,
  Server,
  Shield,
  ShieldCheck,
  SlidersHorizontal,
  Tag,
  Users,
  UserCog,
} from "lucide-react";
import { cn } from "@/lib/utils";

const adminGroups = [
  {
    title: "Betrieb",
    items: [
      { label: "Übersicht", href: "/settings", icon: SlidersHorizontal },
      { label: "Systemstatus", href: "/settings/system", icon: Activity },
      { label: "Nodes", href: "/settings/nodes", icon: Server },
      { label: "Backup & DR", href: "/settings/backups", icon: DatabaseBackup },
      { label: "Benachrichtigungen", href: "/settings/notifications", icon: Bell },
      { label: "Umgebungen", href: "/settings/environments", icon: Globe },
    ],
  },
  {
    title: "Zugriff",
    items: [
      { label: "Sicherheit", href: "/settings/security", icon: ShieldCheck },
      { label: "Benutzer", href: "/settings/users", icon: Users },
      { label: "Rollen & Rechte", href: "/settings/roles", icon: UserCog },
      { label: "VM-Berechtigungen", href: "/settings/vm-permissions", icon: FolderLock },
      { label: "VM-Gruppen", href: "/settings/vm-groups", icon: Monitor },
      { label: "Passwort-Richtlinie", href: "/settings/password-policy", icon: Lock },
      { label: "Audit-Log", href: "/settings/audit-log", icon: ClipboardList },
    ],
  },
  {
    title: "Integrationen",
    items: [
      { label: "API-Tokens", href: "/settings/api-tokens", icon: Shield },
      { label: "SSH-Schlüssel", href: "/settings/ssh-keys", icon: Key },
      { label: "KI-Assistent", href: "/settings/agent", icon: Bot },
      { label: "Wissensbasis", href: "/settings/brain", icon: Brain },
      { label: "Tags", href: "/settings/tags", icon: Tag },
    ],
  },
];

function isActive(pathname: string, href: string): boolean {
  if (href === "/settings") return pathname === href;
  return pathname === href || pathname.startsWith(`${href}/`);
}

export default function SettingsLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const items = adminGroups.flatMap((group) => group.items);
  const activeItem = items.find((item) => isActive(pathname, item.href));

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-3 border-b ops-divider pb-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p className="eyebrow">Admin</p>
          <h1 className="mt-1 text-2xl font-semibold tracking-tight">
            {activeItem?.label ?? "Einstellungen"}
          </h1>
          <p className="mt-1 max-w-2xl text-sm text-muted-foreground">
            System, Zugriff und Integrationen ohne zweite Navigationsspalte.
          </p>
        </div>
        <Link
          href="/settings"
          className="ops-focus-ring inline-flex h-8 items-center justify-center rounded-md border border-border/80 bg-muted/35 px-3 text-xs font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        >
          Admin-Hub
        </Link>
      </div>

      <details className="group overflow-hidden rounded-lg border bg-card text-card-foreground">
        <summary className="ops-focus-ring flex cursor-pointer list-none items-center gap-3 px-4 py-3 text-sm transition-colors hover:bg-accent/35 [&::-webkit-details-marker]:hidden">
          <span className="font-medium">Admin-Funktionen</span>
          <span className="text-xs text-muted-foreground">{items.length}</span>
          <span className="ml-auto text-xs text-muted-foreground">
            {activeItem?.label ?? "Auswählen"}
          </span>
          <ChevronDown className="h-4 w-4 text-muted-foreground transition-transform group-open:rotate-180" />
        </summary>
        <div className="grid gap-4 border-t ops-divider p-4 xl:grid-cols-3">
          {adminGroups.map((group) => (
            <div key={group.title} className="space-y-2">
              <p className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
                {group.title}
              </p>
              <div className="grid gap-1">
                {group.items.map((item) => {
                  const Icon = item.icon;
                  const active = isActive(pathname, item.href);
                  return (
                    <Link
                      key={item.href}
                      href={item.href}
                      className={cn(
                        "ops-focus-ring flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-colors",
                        active
                          ? "bg-primary/10 font-medium text-foreground"
                          : "text-muted-foreground hover:bg-accent/45 hover:text-foreground"
                      )}
                    >
                      <Icon className="h-4 w-4 shrink-0" />
                      <span className="truncate">{item.label}</span>
                    </Link>
                  );
                })}
              </div>
            </div>
          ))}
        </div>
      </details>

      {children}
    </div>
  );
}
