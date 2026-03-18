"use client";

import { useEffect } from "react";
import { toast } from "sonner";

/**
 * Listens for anomaly events dispatched by the existing WebSocket connections
 * via custom DOM events. Does NOT create its own WebSocket connection.
 *
 * To dispatch anomaly toasts, any WS message handler can call:
 *   window.dispatchEvent(new CustomEvent("ws-anomaly", { detail: data }))
 */
export function AnomalyToastListener() {
  useEffect(() => {
    const handler = (e: Event) => {
      const msg = (e as CustomEvent).detail as {
        type?: string;
        data?: Record<string, unknown>;
      };
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
    };

    window.addEventListener("ws-anomaly", handler);
    return () => window.removeEventListener("ws-anomaly", handler);
  }, []);

  return null;
}
