import { createFileRoute } from "@tanstack/react-router";
import { Activity, Bell, Boxes, Server } from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import { KpiCard } from "@/components/ui/kpi-card";
import { FeatureStatusCard } from "@/components/ui/feature-status-card";
import { HealthStatus } from "@/components/health-status";

export const Route = createFileRoute("/")({
  component: HomeRoute,
});

function HomeRoute() {
  return (
    <PageShell
      title="Lagezentrum"
      description="Skeleton-Build von Prometheus V2. Domains werden in Folge-Plaenen angelegt."
      eyebrow="Operations"
    >
      <section className="surface-panel-strong p-5">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              Backend-Status
            </p>
            <h2 className="mt-1 text-xl font-semibold">Live-Verbindung</h2>
            <p className="mt-1 max-w-xl text-sm text-muted-foreground">
              Sobald Auth, Hosts und Realtime-Bus implementiert sind, fliessen hier echte Daten ein.
            </p>
          </div>
          <HealthStatus />
        </div>
      </section>

      <section className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <KpiCard title="Hosts" value="0" delta="kommt mit Plan 6" icon={Server} tone="neutral" />
        <KpiCard title="VMs" value="0" delta="kommt mit Plan 7" icon={Boxes} tone="neutral" />
        <KpiCard title="Aktive Tasks" value="0" delta="kommt mit Plan 14" icon={Activity} tone="neutral" />
        <KpiCard title="Notifications" value="0" delta="kommt mit Plan 13" icon={Bell} tone="neutral" />
      </section>

      <section className="grid gap-3 md:grid-cols-2">
        <FeatureStatusCard
          title="Skeleton bereit"
          description="Backend, Frontend und Build-Pipeline laufen."
          icon={Activity}
          tone="ok"
          status="Stabil"
          details={<p className="text-sm text-muted-foreground">Folge-Plaene fuellen die Domains.</p>}
        />
        <FeatureStatusCard
          title="Domains kommen"
          description="Auth, Audit, Realtime, Approval, Host, VM, ..."
          icon={Bell}
          tone="info"
          status="Geplant"
          details={<p className="text-sm text-muted-foreground">Siehe Plan-Reihenfolge im Skeleton-Plan.</p>}
        />
      </section>
    </PageShell>
  );
}
