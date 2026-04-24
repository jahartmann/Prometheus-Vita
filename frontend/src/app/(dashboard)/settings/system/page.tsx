"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { Activity, Bot, Database, RefreshCw, Server, ShieldCheck, Wifi } from "lucide-react";
import { agentConfigApi, gatewayApi, nodeApi, securityApi, systemApi, toArray } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

type CheckState = "healthy" | "degraded" | "restricted" | "unknown";

interface HealthResponse {
  status?: string;
  services?: Record<string, string>;
}

interface CheckItem {
  id: string;
  label: string;
  detail: string;
  state: CheckState;
  icon: typeof Activity;
}

export default function SystemStatusPage() {
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [checks, setChecks] = useState<CheckItem[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [lastCheckedAt, setLastCheckedAt] = useState<Date | null>(null);

  const loadStatus = useCallback(async () => {
    setIsLoading(true);
    const nextChecks: CheckItem[] = [];

    const healthResult = await settle(() => systemApi.health());
    if (healthResult.ok) {
      const healthData = healthResult.value as HealthResponse;
      setHealth(healthData);
      for (const [service, state] of Object.entries(healthData.services ?? {})) {
        nextChecks.push({
          id: `health-${service}`,
          label: service === "postgres" ? "PostgreSQL" : service === "redis" ? "Redis" : service,
          detail: state,
          state: state.startsWith("healthy") ? "healthy" : "degraded",
          icon: service === "postgres" ? Database : Wifi,
        });
      }
    } else {
      setHealth(null);
      nextChecks.push({
        id: "health",
        label: "Backend Health",
        detail: "Health-Endpunkt nicht erreichbar",
        state: "degraded",
        icon: Activity,
      });
    }

    const nodesResult = await settle(() => nodeApi.list());
    nextChecks.push({
      id: "nodes",
      label: "Node-API",
      detail: nodesResult.ok ? `${toArray<unknown>(nodesResult.value.data).length} Nodes erreichbar` : "Kein Zugriff oder API nicht erreichbar",
      state: nodesResult.ok ? "healthy" : "restricted",
      icon: Server,
    });

    const securityResult = await settle(() => securityApi.getMode());
    nextChecks.push({
      id: "security",
      label: "Security-Modus",
      detail: securityResult.ok ? `Modus: ${securityResult.value?.mode ?? "unbekannt"}` : "Kein Zugriff oder Modus nicht ladbar",
      state: securityResult.ok ? "healthy" : "restricted",
      icon: ShieldCheck,
    });

    const agentResult = await settle(() => agentConfigApi.get());
    nextChecks.push({
      id: "agent",
      label: "Agent-Konfiguration",
      detail: agentResult.ok ? `Provider: ${agentResult.value?.llm_provider ?? "ollama"}` : "Kein Zugriff oder Agent-Konfiguration nicht ladbar",
      state: agentResult.ok ? "healthy" : "restricted",
      icon: Bot,
    });

    const auditResult = await settle(() => gatewayApi.listAuditLog(1, 0));
    nextChecks.push({
      id: "audit",
      label: "Audit-Log",
      detail: auditResult.ok ? "Audit-Log erreichbar" : "Audit-Log nicht verfuegbar oder nicht berechtigt",
      state: auditResult.ok ? "healthy" : "restricted",
      icon: Activity,
    });

    setChecks(nextChecks);
    setLastCheckedAt(new Date());
    setIsLoading(false);
  }, []);

  useEffect(() => {
    loadStatus();
  }, [loadStatus]);

  const summary = useMemo(() => {
    const degraded = checks.filter((check) => check.state === "degraded").length;
    const restricted = checks.filter((check) => check.state === "restricted").length;
    if (degraded > 0) return { label: "Degraded", variant: "degraded" as const };
    if (restricted > 0) return { label: "Teilweise eingeschraenkt", variant: "warning" as const };
    return { label: "Operational", variant: "success" as const };
  }, [checks]);

  return (
    <div className="space-y-5">
      <div className="rounded-lg border bg-card p-4">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="flex items-start gap-3">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-muted">
              <Activity className="h-4 w-4" />
            </div>
            <div>
              <h2 className="text-xl font-semibold tracking-tight">Systemstatus</h2>
              <p className="mt-1 max-w-3xl text-sm text-muted-foreground">
                Backend-Dienste, zentrale APIs und Integrationen auf einen Blick.
              </p>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={summary.variant}>{summary.label}</Badge>
            <Button type="button" variant="outline" size="sm" onClick={loadStatus} disabled={isLoading}>
              <RefreshCw className={`h-4 w-4 ${isLoading ? "animate-spin" : ""}`} />
              Aktualisieren
            </Button>
          </div>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <MetricCard label="Checks" value={checks.length} />
        <MetricCard label="Healthy" value={checks.filter((check) => check.state === "healthy").length} />
        <MetricCard label="Eingeschraenkt" value={checks.filter((check) => check.state !== "healthy").length} />
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Runtime Health</CardTitle>
          <CardDescription>
            {lastCheckedAt ? `Zuletzt geprueft: ${lastCheckedAt.toLocaleString("de-DE")}` : "Noch nicht geprueft"}
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 lg:grid-cols-2">
          {checks.map((check) => {
            const Icon = check.icon;
            return (
              <div key={check.id} className="flex items-start gap-3 rounded-md border p-3">
                <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-muted">
                  <Icon className="h-4 w-4" />
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center justify-between gap-2">
                    <p className="font-medium">{check.label}</p>
                    <Badge variant={stateVariant(check.state)}>{stateLabel(check.state)}</Badge>
                  </div>
                  <p className="mt-1 text-sm text-muted-foreground">{check.detail}</p>
                </div>
              </div>
            );
          })}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Backend</CardTitle>
          <CardDescription>Direkter `/health` Status.</CardDescription>
        </CardHeader>
        <CardContent>
          <pre className="overflow-x-auto rounded-md bg-muted p-3 text-xs">
            {JSON.stringify(health ?? { status: "unreachable" }, null, 2)}
          </pre>
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

async function settle<T>(fn: () => Promise<T>): Promise<{ ok: true; value: T } | { ok: false }> {
  try {
    return { ok: true, value: await fn() };
  } catch {
    return { ok: false };
  }
}

function stateLabel(state: CheckState): string {
  switch (state) {
    case "healthy":
      return "Healthy";
    case "degraded":
      return "Degraded";
    case "restricted":
      return "Eingeschraenkt";
    default:
      return "Unbekannt";
  }
}

function stateVariant(state: CheckState): "success" | "warning" | "degraded" | "outline" {
  switch (state) {
    case "healthy":
      return "success";
    case "degraded":
      return "degraded";
    case "restricted":
      return "warning";
    default:
      return "outline";
  }
}
