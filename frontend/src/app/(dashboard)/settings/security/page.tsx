"use client";

import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { useAuthStore } from "@/stores/auth-store";
import { gatewayApi, toArray } from "@/lib/api";
import type { AuditLogEntry } from "@/types/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  AlertTriangle,
  CheckCircle,
  Key,
  Shield,
  Clock,
  Monitor,
} from "lucide-react";

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

export default function SecurityPage() {
  const { user } = useAuthStore();
  const [loginEntries, setLoginEntries] = useState<AuditLogEntry[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  const fetchLoginActivity = useCallback(async () => {
    setIsLoading(true);
    try {
      const res = await gatewayApi.listAuditLog(200, 0);
      const all = toArray<AuditLogEntry>(res.data);
      const logins = all.filter(
        (e) => e.path === "/api/v1/auth/login" && e.method === "POST"
      );
      setLoginEntries(logins.slice(0, 20));
    } catch {
      setLoginEntries([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchLoginActivity();
  }, [fetchLoginActivity]);

  const warnings: { message: string; severity: "warning" | "info" }[] = [];

  if (user?.username === "admin") {
    warnings.push({
      message:
        'Der Standard-Benutzername "admin" wird verwendet. Es wird empfohlen, einen individuellen Benutzernamen zu wählen.',
      severity: "warning",
    });
  }

  if (user?.must_change_password) {
    warnings.push({
      message:
        "Sie müssen Ihr Passwort ändern. Bitte aktualisieren Sie Ihr Passwort umgehend.",
      severity: "warning",
    });
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold">Sicherheit</h2>
        <p className="text-sm text-muted-foreground">
          Kontostatus, Login-Aktivität und Sicherheitsempfehlungen.
        </p>
      </div>

      {/* Account Overview */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Shield className="h-4 w-4" />
            Konto-Übersicht
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <div className="space-y-1">
              <p className="text-sm text-muted-foreground">Benutzername</p>
              <p className="font-medium">{user?.username ?? "-"}</p>
            </div>
            <div className="space-y-1">
              <p className="text-sm text-muted-foreground">Rolle</p>
              <Badge variant="secondary">{user?.role ?? "-"}</Badge>
            </div>
            <div className="space-y-1">
              <p className="text-sm text-muted-foreground">Status</p>
              <div className="flex items-center gap-2">
                <Badge variant={user?.is_active ? "default" : "secondary"}>
                  {user?.is_active ? "Aktiv" : "Inaktiv"}
                </Badge>
                {user?.must_change_password && (
                  <Badge variant="outline" className="text-amber-600 border-amber-400">
                    Passwort-Änderung erforderlich
                  </Badge>
                )}
              </div>
            </div>
            <div className="space-y-1">
              <p className="text-sm text-muted-foreground">Letzter Login</p>
              <p className="font-medium">
                {user?.last_login ? formatTimestamp(user.last_login) : "-"}
              </p>
            </div>
            <div className="space-y-1">
              <p className="text-sm text-muted-foreground">E-Mail</p>
              <p className="font-medium">{user?.email || "-"}</p>
            </div>
            <div className="flex items-end">
              <Button variant="outline" size="sm" asChild>
                <Link href="/settings/users">
                  <Key className="mr-2 h-4 w-4" />
                  Passwort ändern
                </Link>
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Security Recommendations */}
      {warnings.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <AlertTriangle className="h-4 w-4 text-amber-500" />
              Sicherheitsempfehlungen
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {warnings.map((w, i) => (
                <div
                  key={i}
                  className="flex items-start gap-3 rounded-lg border border-amber-200 bg-amber-50 p-3 dark:border-amber-800 dark:bg-amber-950"
                >
                  <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-amber-500" />
                  <p className="text-sm">{w.message}</p>
                </div>
              ))}
              <div className="flex items-start gap-3 rounded-lg border p-3">
                <Shield className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
                <div className="text-sm">
                  <span>Passwort-Richtlinie konfigurieren: </span>
                  <Link
                    href="/settings/password-policy"
                    className="text-primary underline-offset-4 hover:underline"
                  >
                    Richtlinien-Einstellungen
                  </Link>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {warnings.length === 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <CheckCircle className="h-4 w-4 text-green-500" />
              Sicherheitsempfehlungen
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              <div className="flex items-start gap-3 rounded-lg border border-green-200 bg-green-50 p-3 text-green-800 dark:border-green-800 dark:bg-green-950 dark:text-green-300">
                <CheckCircle className="mt-0.5 h-4 w-4 shrink-0 text-green-500" />
                <p className="text-sm text-green-700 dark:text-green-300">Keine Sicherheitswarnungen vorhanden.</p>
              </div>
              <div className="flex items-start gap-3 rounded-lg border p-3">
                <Shield className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
                <div className="text-sm">
                  <span>Passwort-Richtlinie konfigurieren: </span>
                  <Link
                    href="/settings/password-policy"
                    className="text-primary underline-offset-4 hover:underline"
                  >
                    Richtlinien-Einstellungen
                  </Link>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Login Activity */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Clock className="h-4 w-4" />
            Login-Aktivität
          </CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <p className="text-sm text-muted-foreground py-4 text-center">
              Laden...
            </p>
          ) : loginEntries.length === 0 ? (
            <p className="text-sm text-muted-foreground py-4 text-center">
              Keine Login-Einträge gefunden.
            </p>
          ) : (
            <div className="space-y-3">
              {loginEntries.map((entry) => {
                const success = entry.status_code >= 200 && entry.status_code < 300;
                return (
                  <div
                    key={entry.id}
                    className="flex items-start gap-3 rounded-lg border p-3"
                  >
                    <div
                      className={`mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-full ${
                        success
                          ? "bg-green-100 dark:bg-green-900"
                          : "bg-red-100 dark:bg-red-900"
                      }`}
                    >
                      {success ? (
                        <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400" />
                      ) : (
                        <AlertTriangle className="h-4 w-4 text-red-600 dark:text-red-400" />
                      )}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <Badge
                          variant="secondary"
                          className={
                            success
                              ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300"
                              : "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300"
                          }
                        >
                          {success ? "Erfolgreich" : `Fehlgeschlagen (${entry.status_code})`}
                        </Badge>
                        <span className="text-xs text-muted-foreground">
                          {formatTimestamp(entry.created_at)}
                        </span>
                      </div>
                      <div className="mt-1 flex items-center gap-4 text-xs text-muted-foreground">
                        {entry.ip_address && (
                          <span className="flex items-center gap-1">
                            IP: {entry.ip_address}
                          </span>
                        )}
                        {entry.user_agent && (
                          <span className="flex items-center gap-1 truncate">
                            <Monitor className="h-3 w-3 shrink-0" />
                            {entry.user_agent}
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
