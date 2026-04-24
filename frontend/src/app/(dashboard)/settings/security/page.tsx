"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { toast } from "sonner";
import { AlertTriangle, CheckCircle2, Clock, Key, Monitor, RotateCcw, Save, Shield, ShieldCheck } from "lucide-react";
import { gatewayApi, securityApi, toArray } from "@/lib/api";
import { useAuthStore } from "@/stores/auth-store";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import type { AuditLogEntry } from "@/types/api";

type SecurityMode = "rule_only" | "hybrid" | "full_llm";

interface SecurityStats {
  total?: number;
  unacknowledged?: number;
  by_severity?: Record<string, number>;
  by_category?: Record<string, number>;
}

const securityModes: Array<{
  value: SecurityMode;
  label: string;
  description: string;
  risk: "success" | "warning" | "degraded";
}> = [
  {
    value: "hybrid",
    label: "Hybrid",
    description: "Regeln liefern stabile Signale, KI ergaenzt Kontext und Priorisierung.",
    risk: "success",
  },
  {
    value: "rule_only",
    label: "Nur Regeln",
    description: "Deterministische Analyse ohne LLM. Gut fuer isolierte oder sensible Umgebungen.",
    risk: "warning",
  },
  {
    value: "full_llm",
    label: "Volle KI",
    description: "LLM priorisiert und bewertet umfassend. Nur mit bewusst konfiguriertem Provider nutzen.",
    risk: "degraded",
  },
];

export default function SecurityPage() {
  const { user } = useAuthStore();
  const [mode, setMode] = useState<SecurityMode>("hybrid");
  const [savedMode, setSavedMode] = useState<SecurityMode>("hybrid");
  const [stats, setStats] = useState<SecurityStats | null>(null);
  const [loginEntries, setLoginEntries] = useState<AuditLogEntry[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);

  const isDirty = mode !== savedMode;
  const activeMode = securityModes.find((entry) => entry.value === mode) ?? securityModes[0];
  const warnings = useMemo(() => buildWarnings(user), [user]);

  const loadSecuritySettings = useCallback(async () => {
    setIsLoading(true);
    try {
      const [modeResponse, statsResponse] = await Promise.all([
        securityApi.getMode(),
        securityApi.getStats().catch(() => null),
      ]);
      const nextMode = toSecurityMode(modeResponse?.mode);
      setMode(nextMode);
      setSavedMode(nextMode);
      setStats((statsResponse ?? null) as SecurityStats | null);
    } catch {
      toast.error("Security-Einstellungen konnten nicht geladen werden");
    }

    try {
      const res = await gatewayApi.listAuditLog(200, 0);
      const all = toArray<AuditLogEntry>(res.data);
      const logins = all.filter((entry) => entry.path === "/api/v1/auth/login" && entry.method === "POST");
      setLoginEntries(logins.slice(0, 20));
    } catch {
      setLoginEntries([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    loadSecuritySettings();
  }, [loadSecuritySettings]);

  const saveMode = async () => {
    setIsSaving(true);
    try {
      const response = await securityApi.setMode(mode);
      const nextMode = toSecurityMode(response?.mode ?? mode);
      setMode(nextMode);
      setSavedMode(nextMode);
      toast.success("Security-Modus gespeichert");
    } catch (error: unknown) {
      toast.error(getErrorMessage(error, "Security-Modus konnte nicht gespeichert werden"));
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="space-y-5">
      <div className="rounded-lg border bg-card p-4">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="flex items-start gap-3">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-muted">
              <Shield className="h-4 w-4" />
            </div>
            <div>
              <h2 className="text-xl font-semibold tracking-tight">Sicherheit</h2>
              <p className="mt-1 max-w-3xl text-sm text-muted-foreground">
                Analysemodus, Kontostatus, Security-Signale und Login-Aktivitaet zentral verwalten.
              </p>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={isDirty ? "warning" : "outline"}>{isDirty ? "Geaendert" : "Aktuell"}</Badge>
            <Button type="button" variant="outline" size="sm" onClick={loadSecuritySettings} disabled={isSaving}>
              <RotateCcw className="h-4 w-4" />
              Aktualisieren
            </Button>
            <Button type="button" size="sm" onClick={saveMode} disabled={!isDirty || isSaving}>
              <Save className="h-4 w-4" />
              Speichern
            </Button>
          </div>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <ShieldCheck className="h-4 w-4" />
            Analysemodus
          </CardTitle>
          <CardDescription>Der Modus steuert, wie Security-Events bewertet und priorisiert werden.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-3 lg:grid-cols-3">
            {securityModes.map((entry) => (
              <button
                key={entry.value}
                type="button"
                onClick={() => setMode(entry.value)}
                className={`rounded-md border p-3 text-left transition-colors ${
                  mode === entry.value ? "border-primary bg-primary/5" : "hover:bg-muted/50"
                }`}
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="font-medium">{entry.label}</span>
                  <Badge variant={entry.risk}>{mode === entry.value ? "Aktiv" : "Option"}</Badge>
                </div>
                <p className="mt-2 text-xs text-muted-foreground">{entry.description}</p>
              </button>
            ))}
          </div>
          <div className="rounded-md border p-3 text-sm text-muted-foreground">
            Aktueller Zielmodus: <span className="font-medium text-foreground">{activeMode.label}</span>
          </div>
        </CardContent>
      </Card>

      <div className="grid gap-4 md:grid-cols-3">
        <MetricCard label="Events gesamt" value={stats?.total ?? 0} />
        <MetricCard label="Offen" value={stats?.unacknowledged ?? 0} />
        <MetricCard label="Kritisch" value={stats?.by_severity?.critical ?? stats?.by_severity?.high ?? 0} />
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Shield className="h-4 w-4" />
            Konto-Uebersicht
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <InfoItem label="Benutzername" value={user?.username ?? "-"} />
            <div className="space-y-1">
              <p className="text-sm text-muted-foreground">Rolle</p>
              <Badge variant="secondary">{user?.role ?? "-"}</Badge>
            </div>
            <div className="space-y-1">
              <p className="text-sm text-muted-foreground">Status</p>
              <Badge variant={user?.is_active ? "success" : "outline"}>{user?.is_active ? "Aktiv" : "Inaktiv"}</Badge>
            </div>
            <InfoItem label="Letzter Login" value={user?.last_login ? formatTimestamp(user.last_login) : "-"} />
            <InfoItem label="E-Mail" value={user?.email || "-"} />
            <div className="flex items-end">
              <Button variant="outline" size="sm" asChild>
                <Link href="/settings/users">
                  <Key className="h-4 w-4" />
                  Passwort aendern
                </Link>
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            {warnings.length > 0 ? <AlertTriangle className="h-4 w-4 text-orange-500" /> : <CheckCircle2 className="h-4 w-4 text-green-600" />}
            Sicherheitsempfehlungen
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {warnings.length === 0 ? (
            <div className="flex items-start gap-3 rounded-md border border-green-200 bg-green-50 p-3 text-sm text-green-800 dark:border-green-800 dark:bg-green-500/10 dark:text-green-300">
              <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0" />
              Keine akuten Sicherheitswarnungen vorhanden.
            </div>
          ) : (
            warnings.map((warning) => (
              <div key={warning} className="flex items-start gap-3 rounded-md border border-orange-200 bg-orange-50 p-3 text-sm text-orange-800 dark:border-orange-800 dark:bg-orange-500/10 dark:text-orange-300">
                <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
                {warning}
              </div>
            ))
          )}
          <div className="rounded-md border p-3 text-sm">
            Passwort-Richtlinie konfigurieren:{" "}
            <Link href="/settings/password-policy" className="text-primary underline-offset-4 hover:underline">
              Richtlinien-Einstellungen
            </Link>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Clock className="h-4 w-4" />
            Login-Aktivitaet
          </CardTitle>
          <CardDescription>Die letzten Login-Versuche aus dem Audit-Log.</CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <p className="py-4 text-center text-sm text-muted-foreground">Laden...</p>
          ) : loginEntries.length === 0 ? (
            <p className="py-4 text-center text-sm text-muted-foreground">Keine Login-Eintraege gefunden.</p>
          ) : (
            <div className="space-y-3">
              {loginEntries.map((entry) => {
                const success = entry.status_code >= 200 && entry.status_code < 300;
                return <LoginActivityItem key={entry.id} entry={entry} success={success} />;
              })}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function MetricCard({ label, value }: { label: string; value: number }) {
  return (
    <Card>
      <CardContent className="p-4">
        <p className="text-sm text-muted-foreground">{label}</p>
        <p className="mt-2 text-2xl font-semibold tabular-nums">{value}</p>
      </CardContent>
    </Card>
  );
}

function InfoItem({ label, value }: { label: string; value: string }) {
  return (
    <div className="space-y-1">
      <p className="text-sm text-muted-foreground">{label}</p>
      <p className="font-medium">{value}</p>
    </div>
  );
}

function LoginActivityItem({ entry, success }: { entry: AuditLogEntry; success: boolean }) {
  return (
    <div className="flex items-start gap-3 rounded-md border p-3">
      <div className={`mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-full ${success ? "bg-green-100 dark:bg-green-900" : "bg-red-100 dark:bg-red-900"}`}>
        {success ? <CheckCircle2 className="h-4 w-4 text-green-600" /> : <AlertTriangle className="h-4 w-4 text-red-600" />}
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant={success ? "success" : "degraded"}>
            {success ? "Erfolgreich" : `Fehlgeschlagen (${entry.status_code})`}
          </Badge>
          <span className="text-xs text-muted-foreground">{formatTimestamp(entry.created_at)}</span>
        </div>
        <div className="mt-1 flex flex-wrap items-center gap-4 text-xs text-muted-foreground">
          {entry.ip_address && <span>IP: {entry.ip_address}</span>}
          {entry.user_agent && (
            <span className="inline-flex min-w-0 items-center gap-1 truncate">
              <Monitor className="h-3 w-3 shrink-0" />
              {entry.user_agent}
            </span>
          )}
        </div>
      </div>
    </div>
  );
}

function buildWarnings(user: ReturnType<typeof useAuthStore.getState>["user"]): string[] {
  const warnings: string[] = [];
  if (user?.username === "admin") {
    warnings.push('Der Standard-Benutzername "admin" wird verwendet. Ein individueller Admin-Name ist sicherer.');
  }
  if (user?.must_change_password) {
    warnings.push("Das Passwort muss geaendert werden. Bitte zeitnah aktualisieren.");
  }
  return warnings;
}

function toSecurityMode(value: unknown): SecurityMode {
  if (value === "rule_only" || value === "hybrid" || value === "full_llm") {
    return value;
  }
  return "hybrid";
}

function formatTimestamp(dateStr: string): string {
  return new Date(dateStr).toLocaleString("de-DE", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

function getErrorMessage(error: unknown, fallback: string): string {
  if (error && typeof error === "object") {
    const candidate = error as { response?: { data?: { error?: string; message?: string } }; message?: string };
    return candidate.response?.data?.error || candidate.response?.data?.message || candidate.message || fallback;
  }
  return fallback;
}
