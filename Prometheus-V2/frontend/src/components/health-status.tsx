import { useQuery } from "@tanstack/react-query";
import { api, type SystemHealth } from "@/lib/api/client";

export function HealthStatus() {
  const { data, isLoading, error } = useQuery({
    queryKey: ["system-health"],
    queryFn: async (): Promise<SystemHealth> => {
      const { data, error } = await api.GET("/system/health");
      if (error || !data) {
        throw new Error("Backend health check failed");
      }
      return data;
    },
  });

  if (isLoading) {
    return <p data-testid="health-status">Lade...</p>;
  }
  if (error || !data) {
    return <p data-testid="health-status" className="text-red-600">Backend nicht erreichbar.</p>;
  }
  return (
    <p data-testid="health-status" className="font-mono text-sm">
      status={data.status} version={data.version}
    </p>
  );
}
