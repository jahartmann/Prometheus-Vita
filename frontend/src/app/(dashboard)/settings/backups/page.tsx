"use client";

import Link from "next/link";
import { CalendarClock, DatabaseBackup, FileClock, LifeBuoy, ShieldCheck } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const settings = [
  {
    title: "Backup-Konsole",
    description: "Backups erstellen, pruefen, loeschen und Wiederherstellungen vorbereiten.",
    href: "/backups",
    icon: DatabaseBackup,
    badge: "Operativ",
  },
  {
    title: "Zeitplaene",
    description: "Node-bezogene Backup-Schedules und Retention-Werte verwalten.",
    href: "/backups?tab=schedules",
    icon: CalendarClock,
    badge: "Retention",
  },
  {
    title: "Disaster Recovery",
    description: "Profile, Readiness-Scores, Simulationen und Runbooks zentral steuern.",
    href: "/disaster-recovery",
    icon: LifeBuoy,
    badge: "DR",
  },
];

export default function BackupSettingsPage() {
  return (
    <div className="flex flex-col gap-4">
      <div className="rounded-lg border bg-card p-4">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <h2 className="text-xl font-semibold tracking-tight">Backup & Disaster Recovery</h2>
            <p className="mt-1 max-w-3xl text-sm text-muted-foreground">
              Ein zentraler Einstellungsbereich fuer Schutz, Aufbewahrung und Wiederanlauf.
            </p>
          </div>
          <Badge variant="outline">Recovery Center</Badge>
        </div>
      </div>

      <div className="grid gap-4 xl:grid-cols-3">
        {settings.map((item) => {
          const Icon = item.icon;
          return (
            <Card key={item.href}>
              <CardHeader>
                <div className="flex items-center justify-between gap-3">
                  <div className="flex items-center gap-3">
                    <div className="flex h-9 w-9 items-center justify-center rounded-md bg-muted">
                      <Icon className="h-4 w-4" />
                    </div>
                    <CardTitle className="text-base">{item.title}</CardTitle>
                  </div>
                  <Badge variant="outline">{item.badge}</Badge>
                </div>
              </CardHeader>
              <CardContent className="flex flex-col gap-4">
                <p className="text-sm text-muted-foreground">{item.description}</p>
                <Button asChild variant="outline" className="justify-start">
                  <Link href={item.href}>Oeffnen</Link>
                </Button>
              </CardContent>
            </Card>
          );
        })}
      </div>

      <Card>
        <CardContent className="flex flex-col gap-3 p-4">
          <div className="flex items-center gap-2">
            <ShieldCheck className="h-4 w-4 text-muted-foreground" />
            <h3 className="font-medium">Phase-3-Status</h3>
          </div>
          <div className="grid gap-3 md:grid-cols-3">
            <StatusItem icon={FileClock} label="Audit" text="Restore, Delete und DR-Aktionen laufen ueber Audit-Klassifizierung." />
            <StatusItem icon={CalendarClock} label="Retention" text="Schedules bleiben node-bezogen und sind ueber die Backup-Konsole erreichbar." />
            <StatusItem icon={LifeBuoy} label="Runbooks" text="DR-Runbooks und Readiness werden ueber das Recovery-Modul gepflegt." />
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function StatusItem({ icon: Icon, label, text }: { icon: typeof FileClock; label: string; text: string }) {
  return (
    <div className="rounded-md border p-3">
      <div className="flex items-center gap-2">
        <Icon className="h-4 w-4 text-muted-foreground" />
        <span className="font-medium">{label}</span>
      </div>
      <p className="mt-2 text-sm text-muted-foreground">{text}</p>
    </div>
  );
}
