"use client";

import { useState, useCallback, useEffect, useRef } from "react";
import { useWebSocket } from "./use-websocket";
import { metricsApi, toArray } from "@/lib/api";
import type { MetricsRecord } from "@/types/api";

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
const POLL_INTERVAL = 15_000; // 15s fallback polling when WS is disconnected

function recordToProcessed(m: MetricsRecord): ProcessedMetric {
  return {
    timestamp: m.recorded_at,
    cpu_usage: m.cpu_usage ?? 0,
    memory_usage:
      m.memory_total && m.memory_total > 0
        ? ((m.memory_used ?? 0) / m.memory_total) * 100
        : 0,
    disk_usage:
      m.disk_total && m.disk_total > 0
        ? ((m.disk_used ?? 0) / m.disk_total) * 100
        : 0,
    network_in: m.net_in ?? 0,
    network_out: m.net_out ?? 0,
  };
}

export function useNodeMetrics(nodeId: string, enabled = true) {
  const [metrics, setMetrics] = useState<ProcessedMetric[]>([]);
  const [latestMetrics, setLatestMetrics] = useState<ProcessedMetric | null>(
    null
  );
  const pollTimerRef = useRef<ReturnType<typeof setInterval>>(undefined);
  const lastPollRef = useRef<string>("");

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

  // Fallback REST polling when WebSocket is not connected
  const pollMetrics = useCallback(async () => {
    if (!nodeId || !enabled) return;
    try {
      const since = new Date();
      since.setMinutes(since.getMinutes() - 2); // Last 2 minutes
      const res = await metricsApi.getHistory(
        nodeId,
        since.toISOString(),
        new Date().toISOString()
      );
      const records = toArray<MetricsRecord>(res.data);
      if (records.length === 0) return;

      // Only process new records (after last poll timestamp)
      const newRecords = lastPollRef.current
        ? records.filter((r) => r.recorded_at > lastPollRef.current)
        : records.slice(-1); // First poll: only latest

      if (newRecords.length === 0) return;
      lastPollRef.current = newRecords[newRecords.length - 1].recorded_at;

      const processed = newRecords.map(recordToProcessed);
      setLatestMetrics(processed[processed.length - 1]);
      setMetrics((prev) => {
        const next = [...prev, ...processed];
        return next.length > MAX_DATA_POINTS
          ? next.slice(next.length - MAX_DATA_POINTS)
          : next;
      });
    } catch {
      // Silent - will retry on next poll
    }
  }, [nodeId, enabled]);

  // Start/stop fallback polling based on WS connection status
  useEffect(() => {
    if (isConnected) {
      // WS is connected, stop polling
      if (pollTimerRef.current) {
        clearInterval(pollTimerRef.current);
        pollTimerRef.current = undefined;
      }
      return;
    }

    if (!enabled || !nodeId) return;

    // WS is not connected, start polling as fallback
    pollMetrics(); // immediate first poll
    pollTimerRef.current = setInterval(pollMetrics, POLL_INTERVAL);

    return () => {
      if (pollTimerRef.current) {
        clearInterval(pollTimerRef.current);
        pollTimerRef.current = undefined;
      }
    };
  }, [isConnected, enabled, nodeId, pollMetrics]);

  return { metrics, latestMetrics, isConnected };
}
