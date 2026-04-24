"use client";

import Link from "next/link";
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
  Tag,
  Users,
  UserCog,
} from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

const settingsGroups = [
  {
    title: "Sicherheit & Rechte",
    description: "Benutzer, Rollen, API-Zugriffe und Audit-Spuren verwalten.",
    items: [
      { label: "Sicherheit", href: "/settings/security", icon: ShieldCheck, badge: "Basis" },
      { label: "Benutzer", href: "/settings/users", icon: Users, badge: "Admin" },
      { label: "Rollen & Rechte", href: "/settings/roles", icon: UserCog, badge: "RBAC" },
      { label: "VM-Berechtigungen", href: "/settings/vm-permissions", icon: FolderLock, badge: "Scopes" },
      { label: "VM-Gruppen", href: "/settings/vm-groups", icon: Monitor, badge: "Scopes" },
      { label: "API-Tokens", href: "/settings/api-tokens", icon: Shield, badge: "Scopes" },
      { label: "Audit-Log", href: "/settings/audit-log", icon: ClipboardList, badge: "Nachweis" },
      { label: "Passwort-Richtlinie", href: "/settings/password-policy", icon: Lock, badge: "Policy" },
    ],
  },
  {
    title: "System",
    description: "Runtime-Status, Dienste und Integrationen pruefen.",
    items: [
      { label: "Systemstatus", href: "/settings/system", icon: Activity, badge: "Health" },
    ],
  },
  {
    title: "Infrastruktur",
    description: "Proxmox-Zugänge, Umgebungen und operative Metadaten pflegen.",
    items: [
      { label: "Nodes", href: "/settings/nodes", icon: Server, badge: "Proxmox" },
      { label: "Backup & DR", href: "/settings/backups", icon: DatabaseBackup, badge: "Recovery" },
      { label: "SSH-Schlüssel", href: "/settings/ssh-keys", icon: Key, badge: "Zugriff" },
      { label: "Umgebungen", href: "/settings/environments", icon: Globe, badge: "Scope" },
      { label: "Tags", href: "/settings/tags", icon: Tag, badge: "Inventar" },
    ],
  },
  {
    title: "Automatisierung & KI",
    description: "Agent, Local AI, Wissensbasis und Benachrichtigungen steuern.",
    items: [
      { label: "KI-Assistent", href: "/settings/agent", icon: Bot, badge: "LLM" },
      { label: "Wissensbasis", href: "/settings/brain", icon: Brain, badge: "Memory" },
      { label: "Benachrichtigungen", href: "/settings/notifications", icon: Bell, badge: "Alerts" },
    ],
  },
];

export default function SettingsPage() {
  return (
    <div className="space-y-5">
      <div className="rounded-lg border bg-card p-4">
        <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Admin-Konsole</p>
        <h2 className="text-xl font-semibold tracking-tight">Einstellungszentrale</h2>
        <p className="mt-1 text-sm text-muted-foreground">
          Alle sicherheitsrelevanten und operativen Einstellungen an einem Ort, gruppiert nach Verantwortungsbereich.
        </p>
      </div>

      <div className="grid gap-4 xl:grid-cols-3">
        {settingsGroups.map((group) => (
          <Card key={group.title}>
            <CardContent className="p-4">
              <div className="mb-4">
                <h3 className="text-base font-semibold">{group.title}</h3>
                <p className="mt-1 text-sm text-muted-foreground">{group.description}</p>
              </div>
              <div className="flex flex-col gap-2">
                {group.items.map((item) => {
                  const Icon = item.icon;
                  return (
                    <Link
                      key={item.href}
                      href={item.href}
                      className="flex items-center gap-3 rounded-md border p-3 transition-colors hover:bg-accent hover:text-accent-foreground"
                    >
                      <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-muted">
                        <Icon className="h-4 w-4" />
                      </div>
                      <span className="flex-1 text-sm font-medium">{item.label}</span>
                      <Badge variant="outline">{item.badge}</Badge>
                    </Link>
                  );
                })}
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
