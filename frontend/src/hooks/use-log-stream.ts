"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useAuthStore } from "@/stores/auth-store";
import { useLogStore } from "@/stores/log-store";

interface UseLogStreamOptions {
  nodeIds: string[];
  sources?: string[];
  severityFilter?: string[];
  enabled?: boolean;
}

export function useLogStream({
  nodeIds,
  sources,
  severityFilter,
  enabled = true,
}: UseLogStreamOptions) {
  const [isConnected, setIsConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isMountedRef = useRef(true);
  const shouldReconnectRef = useRef(false);

  useEffect(() => {
    return () => { isMountedRef.current = false; };
  }, []);
  const addEntry = useLogStore((s) => s.addEntry);
  const updateKpis = useLogStore((s) => s.updateKpis);
  const accessToken = useAuthStore((s) => s.accessToken);

  const connect = useCallback(() => {
    if (!enabled || !accessToken || nodeIds.length === 0) return;
    if (wsRef.current?.readyState === WebSocket.OPEN || wsRef.current?.readyState === WebSocket.CONNECTING) return;
    shouldReconnectRef.current = true;

    // Next.js cannot proxy WebSocket upgrades; on the direct frontend port (3000)
    // target the backend's published port (8080). Same-origin behind a proxy.
    const wsProto = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsHost =
      window.location.port === "3000"
        ? `${window.location.hostname}:8080`
        : window.location.host;
    const wsUrl = process.env.NEXT_PUBLIC_WS_URL
      ? `${process.env.NEXT_PUBLIC_WS_URL}/api/v1/ws/logs?token=${accessToken}`
      : `${wsProto}//${wsHost}/api/v1/ws/logs?token=${accessToken}`;

    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      if (isMountedRef.current) setIsConnected(true);
      // Send subscription
      ws.send(JSON.stringify({
        type: "subscribe",
        node_ids: nodeIds,
        sources: sources || [],
        severity_filter: severityFilter || [],
      }));
    };

    ws.onmessage = (event) => {
      if (!isMountedRef.current) return;
      try {
        const data = JSON.parse(event.data);
        if (data.type === "log") {
          addEntry(data.data);
        } else if (data.type === "kpi_update") {
          updateKpis(data.data);
        }
      } catch { /* ignore parse errors */ }
    };

    ws.onclose = () => {
      if (isMountedRef.current) setIsConnected(false);
      wsRef.current = null;
      // Auto-reconnect after 3 seconds
      if (shouldReconnectRef.current && enabled) {
        reconnectTimeoutRef.current = setTimeout(connect, 3000);
      }
    };

    ws.onerror = () => {
      ws.close();
    };
  }, [enabled, accessToken, nodeIds, sources, severityFilter, addEntry, updateKpis]);

  useEffect(() => {
    connect();
    return () => {
      shouldReconnectRef.current = false;
      if (reconnectTimeoutRef.current) clearTimeout(reconnectTimeoutRef.current);
      if (wsRef.current) {
        wsRef.current.onclose = null;
        wsRef.current.close();
        wsRef.current = null;
      }
      setIsConnected(false);
    };
  }, [connect]);

  return { isConnected };
}
