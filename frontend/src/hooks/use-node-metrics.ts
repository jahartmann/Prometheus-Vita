"use client";

import { useState, useCallback } from "react";
import { useWebSocket } from "./use-websocket";
import type { NodeMetrics } from "@/types/api";

const MAX_DATA_POINTS = 60;

export function useNodeMetrics(nodeId: string, enabled = true) {
  const [metrics, setMetrics] = useState<NodeMetrics[]>([]);
  const [latestMetrics, setLatestMetrics] = useState<NodeMetrics | null>(null);

  const handleMessage = useCallback((data: unknown) => {
    const metric = data as NodeMetrics;
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
