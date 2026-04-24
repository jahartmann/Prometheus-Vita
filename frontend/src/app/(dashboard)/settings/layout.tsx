"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Activity,
  Bell,
  Bot,
  Brain,
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

const settingsNav = [
  { label: "Uebersicht", href: "/settings", icon: SlidersHorizontal },
  { label: "Systemstatus", href: "/settings/system", icon: Activity },
  { label: "Sicherheit", href: "/settings/security", icon: ShieldCheck },
  { label: "Nodes", href: "/settings/nodes", icon: Server },
  { label: "Backup & DR", href: "/settings/backups", icon: DatabaseBackup },
  { label: "Benutzer", href: "/settings/users", icon: Users },
  { label: "Rollen & Rechte", href: "/settings/roles", icon: UserCog },
  { label: "VM-Berechtigungen", href: "/settings/vm-permissions", icon: FolderLock },
  { label: "VM-Gruppen", href: "/settings/vm-groups", icon: Monitor },
  { label: "Benachrichtigungen", href: "/settings/notifications", icon: Bell },
  { label: "Tags", href: "/settings/tags", icon: Tag },
  { label: "Umgebungen", href: "/settings/environments", icon: Globe },
  { label: "SSH-Schluessel", href: "/settings/ssh-keys", icon: Key },
  { label: "API-Tokens", href: "/settings/api-tokens", icon: Shield },
  { label: "KI-Assistent", href: "/settings/agent", icon: Bot },
  { label: "Wissensbasis", href: "/settings/brain", icon: Brain },
  { label: "Passwort-Richtlinie", href: "/settings/password-policy", icon: Lock },
  { label: "Audit-Log", href: "/settings/audit-log", icon: ClipboardList },
];

export default function SettingsLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Einstellungen</h1>
        <p className="text-muted-foreground">
          Verwalten Sie Sicherheit, Rechte, Integrationen und Betriebseinstellungen.
        </p>
      </div>

      <div className="flex flex-col gap-6 lg:flex-row">
        <nav className="flex gap-1 overflow-x-auto pb-1 lg:w-56 lg:flex-col lg:overflow-visible lg:pb-0">
          {settingsNav.map((item) => {
            const Icon = item.icon;
            const active = pathname === item.href || pathname.startsWith(item.href + "/");
            return (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  "flex shrink-0 items-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                  active
                    ? "bg-zinc-100 text-zinc-900 dark:bg-zinc-800 dark:text-zinc-100"
                    : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
                )}
              >
                <Icon className="h-4 w-4" />
                {item.label}
              </Link>
            );
          })}
        </nav>

        <div className="min-w-0 flex-1">{children}</div>
      </div>
    </div>
  );
}
