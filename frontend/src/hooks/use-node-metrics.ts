"use client";

import { useState, useCallback } from "react";
import { useWebSocket } from "./use-websocket";
import type { NodeMetrics } from "@/types/api";

const MAX_DATA_POINTS = 60;

export function useNodeMetrics(nodeId: string, enabled = true) {
  const [metrics, setMetrics] = useState<NodeMetrics[]>([]);
  const [latestMetrics, setLatestMetrics] = useState<NodeMetrics | null>(null);

  const handleMessage = useCallback((data: unknown) => {
    if (!data || typeof data !== "object") return;
    const msg = data as Record<string, unknown>;
    // Only process metrics messages (skip heartbeats, errors, etc.)
    if (msg.type && msg.type !== "node_metrics" && msg.type !== "metrics") {
      return;
    }
    const metric = (msg.data ?? msg) as NodeMetrics;
    if (!metric.timestamp || metric.cpu_usage === undefined) return;

    setLatestMetrics(metric);
    setMetrics((prev) => {
      const next = [...prev, metric];
      if (next.length > MAX_DATA_POINTS) {
        return next.slice(next.length - MAX_DATA_POINTS);
      }
      return next;
    });
  }, []);

  const { isConnected } = useWebSocket({
    url: `/api/v1/ws`,
    onMessage: handleMessage,
    enabled: enabled && !!nodeId,
  });

  return { metrics, latestMetrics, isConnected };
}
