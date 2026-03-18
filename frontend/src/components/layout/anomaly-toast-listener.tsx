"use client";

import { useWebSocket } from "@/hooks/use-websocket";
import { toast } from "sonner";

export function AnomalyToastListener() {
  useWebSocket({
    url: "/api/v1/ws",
    onMessage: (data: unknown) => {
      const msg = data as { type?: string; data?: Record<string, unknown> };
      if (!msg?.type || !msg?.data) return;

      if (msg.type === "log_anomaly") {
        const d = msg.data;
        const score = Number(d.anomaly_score ?? 0);
        if (score > 0.5) {
          toast.error(`Log Anomalie: ${d.summary ?? "Unbekannt"}`, {
            description: `Node: ${d.node_id ?? "?"} | Score: ${score.toFixed(2)} | ${d.category ?? ""}`,
          });
        }
      }

      if (msg.type === "network_anomaly") {
        const d = msg.data;
        toast.warning(`Netzwerk Anomalie: ${d.anomaly_type ?? "Unbekannt"}`, {
          description: `Risk: ${Number(d.risk_score ?? 0).toFixed(2)} | Node: ${d.node_id ?? "?"}`,
        });
      }
    },
  });

  return null;
}
