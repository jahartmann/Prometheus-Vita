"use client";

import { useEffect, useRef, useCallback, useState } from "react";
import { useAuthStore } from "@/stores/auth-store";

interface UseWebSocketOptions {
  url: string;
  onMessage?: (data: unknown) => void;
  onOpen?: () => void;
  onClose?: () => void;
  onError?: (error: Event) => void;
  enabled?: boolean;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
}

const BASE_RECONNECT_INTERVAL = 1000;
const MAX_RECONNECT_INTERVAL = 30000;

export function useWebSocket({
  url,
  onMessage,
  onOpen,
  onClose,
  onError,
  enabled = true,
  maxReconnectAttempts = Infinity,
}: UseWebSocketOptions) {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  const isConnectingRef = useRef(false);
  const enabledRef = useRef(enabled);
  const [isConnected, setIsConnected] = useState(false);
  const accessToken = useAuthStore((s) => s.accessToken);

  // Keep enabledRef in sync so closures see the latest value
  useEffect(() => {
    enabledRef.current = enabled;
  }, [enabled]);

  const getBackoffDelay = useCallback((attempt: number) => {
    const delay = BASE_RECONNECT_INTERVAL * Math.pow(2, attempt);
    return Math.min(delay, MAX_RECONNECT_INTERVAL);
  }, []);

  const connect = useCallback(() => {
    if (!enabledRef.current || !accessToken) return;
    if (document.visibilityState === "hidden") return;
    if (isConnectingRef.current) return;
    if (wsRef.current?.readyState === WebSocket.OPEN) return;
    if (wsRef.current?.readyState === WebSocket.CONNECTING) return;

    isConnectingRef.current = true;

    // Build WS URL: use NEXT_PUBLIC_API_URL for direct backend connection
    // (Next.js rewrites only handle HTTP, not WebSocket upgrades)
    const apiUrl = process.env.NEXT_PUBLIC_API_URL || "";
    let wsUrl: string;
    if (apiUrl) {
      // Direct backend connection: http://host:port → ws://host:port
      const base = apiUrl.replace(/^http/, "ws");
      wsUrl = `${base}${url}?token=${accessToken}`;
    } else {
      // Same-origin fallback (works behind a reverse proxy that handles WS)
      const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      wsUrl = `${protocol}//${window.location.host}${url}?token=${accessToken}`;
    }

    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      isConnectingRef.current = false;
      setIsConnected(true);
      reconnectAttemptsRef.current = 0;
      onOpen?.();
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        onMessage?.(data);
      } catch {
        onMessage?.(event.data);
      }
    };

    ws.onclose = () => {
      isConnectingRef.current = false;
      setIsConnected(false);
      onClose?.();

      if (
        enabledRef.current &&
        reconnectAttemptsRef.current < maxReconnectAttempts
      ) {
        const delay = getBackoffDelay(reconnectAttemptsRef.current);
        reconnectAttemptsRef.current++;
        reconnectTimerRef.current = setTimeout(connect, delay);
      }
    };

    ws.onerror = (error) => {
      isConnectingRef.current = false;
      onError?.(error);
    };
  }, [
    url,
    accessToken,
    maxReconnectAttempts,
    getBackoffDelay,
    onMessage,
    onOpen,
    onClose,
    onError,
  ]);

  useEffect(() => {
    if (!enabled) return;

    connect();

    const handleVisibilityChange = () => {
      if (document.visibilityState === "visible") {
        // Clear any pending reconnect timer and try immediately
        if (reconnectTimerRef.current) {
          clearTimeout(reconnectTimerRef.current);
          reconnectTimerRef.current = undefined;
        }
        if (
          !wsRef.current ||
          wsRef.current.readyState === WebSocket.CLOSED ||
          wsRef.current.readyState === WebSocket.CLOSING
        ) {
          reconnectAttemptsRef.current = 0;
          connect();
        }
      }
    };

    document.addEventListener("visibilitychange", handleVisibilityChange);

    return () => {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
      isConnectingRef.current = false;
    };
  }, [connect, enabled]);

  const send = useCallback((data: unknown) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data));
    }
  }, []);

  return { isConnected, send };
}
