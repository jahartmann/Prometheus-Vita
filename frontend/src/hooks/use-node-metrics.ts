"use client";

import { useState, useCallback } from "react";
import { useWebSocket } from "./use-websocket";

interface RawMetric {
  node_id?: string;
  recorded_at?: string;
  timestamp?: string;
  cpu_usage?: number;
  memory_used?: number;
  memory_total?: number;
  disk_used?: number;
  disk_total?: number;
  net_in?: number;
  net_out?: number;
}

export interface ProcessedMetric {
  timestamp: string;
  cpu_usage: number;
  memory_usage: number;
  disk_usage: number;
  network_in: number;
  network_out: number;
}

const MAX_DATA_POINTS = 60;

export function useNodeMetrics(nodeId: string, enabled = true) {
  const [metrics, setMetrics] = useState<ProcessedMetric[]>([]);
  const [latestMetrics, setLatestMetrics] = useState<ProcessedMetric | null>(
    null
  );

  const handleMessage = useCallback(
    (data: unknown) => {
      if (!data || typeof data !== "object") return;
      const msg = data as Record<string, unknown>;

      // Only process node_metrics messages
      if (msg.type && msg.type !== "node_metrics" && msg.type !== "metrics") {
        return;
      }

      const raw = (msg.data ?? msg) as RawMetric;

      // Filter by node_id - only process metrics for OUR node
      if (raw.node_id && raw.node_id !== nodeId) return;

      // Accept both recorded_at and timestamp
      const ts = raw.recorded_at || raw.timestamp;
      if (!ts || raw.cpu_usage === undefined) return;

      const metric: ProcessedMetric = {
        timestamp: ts,
        cpu_usage: raw.cpu_usage ?? 0,
        memory_usage:
          raw.memory_total && raw.memory_total > 0
            ? ((raw.memory_used ?? 0) / raw.memory_total) * 100
            : 0,
        disk_usage:
          raw.disk_total && raw.disk_total > 0
            ? ((raw.disk_used ?? 0) / raw.disk_total) * 100
            : 0,
        network_in: raw.net_in ?? 0,
        network_out: raw.net_out ?? 0,
      };

      setLatestMetrics(metric);
      setMetrics((prev) => {
        const next = [...prev, metric];
        return next.length > MAX_DATA_POINTS
          ? next.slice(next.length - MAX_DATA_POINTS)
          : next;
      });
    },
    [nodeId]
  );

  const { isConnected } = useWebSocket({
    url: `/api/v1/ws`,
    onMessage: handleMessage,
    enabled: enabled && !!nodeId,
  });

  return { metrics, latestMetrics, isConnected };
}
