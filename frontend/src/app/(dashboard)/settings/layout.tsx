"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Bell, Server, Users, Key, Globe, Tag, Shield, Bot, Brain } from "lucide-react";
import { cn } from "@/lib/utils";

const settingsNav = [
  {
    label: "Nodes",
    href: "/settings/nodes",
    icon: Server,
  },
  {
    label: "Benutzer",
    href: "/settings/users",
    icon: Users,
  },
  {
    label: "Benachrichtigungen",
    href: "/settings/notifications",
    icon: Bell,
  },
  {
    label: "Tags",
    href: "/settings/tags",
    icon: Tag,
  },
  {
    label: "Umgebungen",
    href: "/settings/environments",
    icon: Globe,
  },
  {
    label: "SSH-Schluessel",
    href: "/settings/ssh-keys",
    icon: Key,
  },
  {
    label: "API-Tokens",
    href: "/settings/api-tokens",
    icon: Shield,
  },
  {
    label: "KI-Assistent",
    href: "/settings/agent",
    icon: Bot,
  },
  {
    label: "Wissensbasis",
    href: "/settings/brain",
    icon: Brain,
  },
];

export default function SettingsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Einstellungen</h1>
        <p className="text-muted-foreground">
          Konfigurieren Sie Ihre Prometheus-Instanz.
        </p>
      </div>

      <div className="flex flex-col gap-6 lg:flex-row">
        <nav className="flex gap-1 lg:w-48 lg:flex-col">
          {settingsNav.map((item) => {
            const Icon = item.icon;
            const active = pathname === item.href || pathname.startsWith(item.href + "/");
            return (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  "flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                  active
                    ? "bg-primary/10 text-primary"
                    : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
                )}
              >
                <Icon className="h-4 w-4" />
                {item.label}
              </Link>
            );
          })}
        </nav>

        <div className="flex-1">{children}</div>
      </div>
    </div>
  );
}
