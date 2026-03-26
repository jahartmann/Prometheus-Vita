"use client";

import { useRef, useCallback, useState } from "react";
import { useAuthStore } from "@/stores/auth-store";

interface UseVMShellOptions {
  nodeId: string;
  vmid: number;
  vmType: string;
}

export interface ShellError {
  errorCode: string;
  message: string;
  details?: string;
  hint?: string;
}

export function useVMShell({ nodeId, vmid, vmType }: UseVMShellOptions) {
  const wsRef = useRef<WebSocket | null>(null);
  const onDataRef = useRef<((data: string) => void) | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [shellError, setShellError] = useState<ShellError | null>(null);

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

    ws.onclose = (event) => {
      setIsConnected(false);
      if (event.code === 1008) {
        setShellError({
          errorCode: "VM_PERMISSION_DENIED",
          message: "Keine Berechtigung für Shell-Zugriff",
          details: "Sie haben keine Shell-Berechtigung für diese VM.",
        });
      } else if (event.code === 1011 || event.code === 1006) {
        setShellError({
          errorCode: "VM_EXEC_FAILED",
          message: "Terminal-Verbindung fehlgeschlagen",
          details: "Die Verbindung zum VM-Terminal konnte nicht hergestellt werden.",
          hint: "Stellen Sie sicher, dass die VM läuft und der VNC-Proxy erreichbar ist.",
        });
      }
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

  const clearShellError = useCallback(() => {
    setShellError(null);
  }, []);

  return { connect, disconnect, send, setOnData, isConnected, shellError, clearShellError };
}
