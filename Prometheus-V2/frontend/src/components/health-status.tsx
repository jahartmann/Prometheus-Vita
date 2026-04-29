import { useQuery } from "@tanstack/react-query";
import { api, ApiError, type SystemHealth } from "@/lib/api/client";

export function HealthStatus() {
  const { data, isLoading, error } = useQuery({
    queryKey: ["system-health"],
    queryFn: async (): Promise<SystemHealth> => {
      const { data, error, response } = await api.GET("/system/health");
      if (error || !data) {
        throw new ApiError(response?.status ?? 0, error, "Backend health check failed");
      }
      return data;
    },
  });

  if (isLoading) {
    return <p data-testid="health-status">Lade...</p>;
  }
  if (error || !data) {
    const apiErr = error instanceof ApiError ? error : null;
    const detail = apiErr?.requestId ? ` (request_id=${apiErr.requestId})` : "";
    return (
      <p data-testid="health-status" className="text-red-600">
        Backend nicht erreichbar.{detail}
      </p>
    );
  }
  return (
    <p data-testid="health-status" className="font-mono text-sm">
      status={data.status} version={data.version}
    </p>
  );
}
