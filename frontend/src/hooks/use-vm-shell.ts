"use client";

import { useRef, useCallback, useState } from "react";
import { useAuthStore } from "@/stores/auth-store";

interface UseVMShellOptions {
  nodeId: string;
  vmid: number;
  vmType: string;
}

export function useVMShell({ nodeId, vmid, vmType }: UseVMShellOptions) {
  const wsRef = useRef<WebSocket | null>(null);
  const onDataRef = useRef<((data: string) => void) | null>(null);
  const [isConnected, setIsConnected] = useState(false);

  const connect = useCallback(() => {
    const accessToken = useAuthStore.getState().accessToken;
    if (!accessToken) return;

    // Close existing connection
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    const wsOverride = process.env.NEXT_PUBLIC_WS_URL;
    const url = `/api/v1/nodes/${nodeId}/vms/${vmid}/cockpit/shell?type=${vmType}&token=${accessToken}`;
    let wsUrl: string;
    if (wsOverride) {
      wsUrl = `${wsOverride}${url}`;
    } else {
      const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      wsUrl = `${protocol}//${window.location.host}${url}`;
    }

    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      setIsConnected(true);
    };

    ws.onmessage = (event) => {
      if (onDataRef.current) {
        onDataRef.current(event.data);
      }
    };

    ws.onclose = () => {
      setIsConnected(false);
    };

    ws.onerror = () => {
      setIsConnected(false);
    };
  }, [nodeId, vmid, vmType]);

  const disconnect = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    setIsConnected(false);
  }, []);

  const send = useCallback((data: string) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(data);
    }
  }, []);

  const setOnData = useCallback((handler: (data: string) => void) => {
    onDataRef.current = handler;
  }, []);

  return { connect, disconnect, send, setOnData, isConnected };
}
