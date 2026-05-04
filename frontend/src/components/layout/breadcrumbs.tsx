"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { ChevronRight } from "lucide-react";

const segmentLabels: Record<string, string> = {
  nodes: "Nodes",
  settings: "Einstellungen",
  users: "Benutzer",
  backups: "Backups",
  monitoring: "Monitoring",
  migrations: "Migrationen",
  topology: "Topologie",
  updates: "Updates",
  recommendations: "Empfehlungen",
  briefing: "Lagebrief",
  chat: "KI-Chat",
  "disaster-recovery": "Notfallplanung",
  notifications: "Benachrichtigungen",
  environments: "Umgebungen",
  "ssh-keys": "SSH-Schlüssel",
  "api-tokens": "API-Tokens",
  tags: "Tags",
  agent: "KI-Assistent",
  "password-policy": "Passwort-Richtlinie",
  vms: "VMs & Container",
  network: "Netzwerk",
  storage: "Speicher",
  dr: "Notfallplanung",
  "change-password": "Passwort ändern",
  alerts: "Alarme",
  "task-center": "Aufgaben",
  security: "Sicherheit",
  drift: "Drift-Erkennung",
  health: "VM-Gesundheit",
  isos: "ISOs & Vorlagen",
  reflex: "Reflex-Regeln",
  dependencies: "Abhängigkeiten",
  "knowledge-graph": "Wissensgraph",
  "root-cause": "Ursachenanalyse",
};

export function Breadcrumbs() {
  const pathname = usePathname();
  const segments = pathname.split("/").filter(Boolean);

  if (segments.length === 0) return null;

  const crumbs = segments.map((segment, index) => {
    const href = "/" + segments.slice(0, index + 1).join("/");
    const label = segmentLabels[segment] || segment;
    const isLast = index === segments.length - 1;

    return { href, label, isLast, segment };
  });

  return (
    <nav className="flex items-center gap-1 text-sm">
      {crumbs.map((crumb, index) => (
        <div key={crumb.href} className="flex items-center gap-1">
          {index > 0 && (
            <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
          )}
          {crumb.isLast ? (
            <span className="font-medium text-foreground">{crumb.label}</span>
          ) : (
            <Link
              href={crumb.href}
              className="text-muted-foreground hover:text-foreground transition-colors"
            >
              {crumb.label}
            </Link>
          )}
        </div>
      ))}
    </nav>
  );
}
