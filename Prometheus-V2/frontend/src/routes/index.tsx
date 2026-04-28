import { createFileRoute } from "@tanstack/react-router";
import { HealthStatus } from "@/components/health-status";

export const Route = createFileRoute("/")({
  component: HomeRoute,
});

function HomeRoute() {
  return (
    <main className="mx-auto max-w-3xl p-8">
      <h1 className="text-3xl font-semibold tracking-tight">Prometheus V2</h1>
      <p className="mt-2 text-base text-muted-foreground">Skeleton — Backend antwortet:</p>
      <div className="mt-6">
        <HealthStatus />
      </div>
    </main>
  );
}
