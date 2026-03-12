"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { SearchAddon } from "@xterm/addon-search";
import "@xterm/xterm/css/xterm.css";
import { Plus, X, Wifi, WifiOff } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { useVMShell } from "@/hooks/use-vm-shell";
import { CockpitError } from "./cockpit-error";

interface ShellSession {
  id: number;
  label: string;
  terminal: Terminal;
  fitAddon: FitAddon;
  searchAddon: SearchAddon;
}

interface ShellTabProps {
  nodeId: string;
  vmid: number;
  vmType: string;
}

const MAX_SESSIONS = 4;

export function ShellTab({ nodeId, vmid, vmType }: ShellTabProps) {
  const [sessions, setSessions] = useState<ShellSession[]>([]);
  const [activeSessionId, setActiveSessionId] = useState<number | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const nextIdRef = useRef(1);
  const { connect, disconnect, send, setOnData, isConnected, shellError, clearShellError } = useVMShell({
    nodeId,
    vmid,
    vmType,
  });

  const createSession = useCallback(() => {
    if (sessions.length >= MAX_SESSIONS) return;

    const id = nextIdRef.current++;
    const terminal = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', Menlo, monospace",
      theme: {
        background: "#09090b",
        foreground: "#fafafa",
        cursor: "#fafafa",
        selectionBackground: "#27272a",
        black: "#09090b",
        red: "#ef4444",
        green: "#22c55e",
        yellow: "#eab308",
        blue: "#3b82f6",
        magenta: "#a855f7",
        cyan: "#06b6d4",
        white: "#fafafa",
      },
      allowProposedApi: true,
    });

    const fitAddon = new FitAddon();
    const searchAddon = new SearchAddon();
    terminal.loadAddon(fitAddon);
    terminal.loadAddon(searchAddon);

    terminal.onData((data) => {
      send(data);
    });

    const session: ShellSession = {
      id,
      label: `Shell ${id}`,
      terminal,
      fitAddon,
      searchAddon,
    };

    setSessions((prev) => [...prev, session]);
    setActiveSessionId(id);

    return session;
  }, [sessions.length, send]);

  const closeSession = useCallback(
    (sessionId: number) => {
      setSessions((prev) => {
        const updated = prev.filter((s) => s.id !== sessionId);
        const closed = prev.find((s) => s.id === sessionId);
        if (closed) {
          closed.terminal.dispose();
        }
        if (activeSessionId === sessionId && updated.length > 0) {
          setActiveSessionId(updated[updated.length - 1].id);
        } else if (updated.length === 0) {
          setActiveSessionId(null);
        }
        return updated;
      });
    },
    [activeSessionId]
  );

  // Mount active session terminal
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const activeSession = sessions.find((s) => s.id === activeSessionId);
    if (!activeSession) return;

    // Clear container using DOM API
    while (container.firstChild) {
      container.removeChild(container.firstChild);
    }

    activeSession.terminal.open(container);

    // Small delay to let the DOM settle before fitting
    requestAnimationFrame(() => {
      try {
        activeSession.fitAddon.fit();
      } catch {
        // ignore fit errors during initialization
      }
    });

    const resizeObserver = new ResizeObserver(() => {
      try {
        activeSession.fitAddon.fit();
      } catch {
        // ignore fit errors
      }
    });

    resizeObserver.observe(container);

    return () => {
      resizeObserver.disconnect();
    };
  }, [activeSessionId, sessions]);

  // Handle incoming data
  useEffect(() => {
    setOnData((data: string) => {
      const activeSession = sessions.find((s) => s.id === activeSessionId);
      if (activeSession) {
        activeSession.terminal.write(data);
      }
    });
  }, [activeSessionId, sessions, setOnData]);

  // Connect on mount, create initial session
  useEffect(() => {
    connect();
    return () => {
      disconnect();
    };
  }, [connect, disconnect]);

  // Create initial session once connected
  useEffect(() => {
    if (isConnected && sessions.length === 0) {
      createSession();
    }
  }, [isConnected, sessions.length, createSession]);

  return (
    <div className="flex h-[600px] flex-col rounded-lg border bg-[#09090b]">
      {/* Session tab bar */}
      <div className="flex items-center gap-1 border-b border-zinc-800 bg-zinc-950 px-2 py-1">
        {sessions.map((session) => (
          <div
            key={session.id}
            className={`flex items-center gap-1 rounded-t px-3 py-1.5 text-xs cursor-pointer transition-colors ${
              session.id === activeSessionId
                ? "bg-[#09090b] text-white"
                : "text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900"
            }`}
            onClick={() => setActiveSessionId(session.id)}
          >
            <span>{session.label}</span>
            {sessions.length > 1 && (
              <button
                className="ml-1 rounded p-0.5 hover:bg-zinc-700"
                onClick={(e) => {
                  e.stopPropagation();
                  closeSession(session.id);
                }}
              >
                <X className="h-3 w-3" />
              </button>
            )}
          </div>
        ))}

        {sessions.length < MAX_SESSIONS && (
          <Button
            variant="ghost"
            size="sm"
            className="h-7 w-7 p-0 text-zinc-400 hover:text-white hover:bg-zinc-800"
            onClick={createSession}
          >
            <Plus className="h-3.5 w-3.5" />
          </Button>
        )}

        <div className="ml-auto">
          <Badge
            variant={isConnected ? "success" : "destructive"}
            className="text-[10px] gap-1"
          >
            {isConnected ? (
              <>
                <Wifi className="h-3 w-3" />
                Verbunden
              </>
            ) : (
              <>
                <WifiOff className="h-3 w-3" />
                Getrennt
              </>
            )}
          </Badge>
        </div>
      </div>

      {/* Error display */}
      {shellError && !isConnected && (
        <CockpitError
          {...shellError}
          onRetry={() => {
            clearShellError();
            connect();
          }}
        />
      )}

      {/* Terminal container */}
      <div ref={containerRef} className="flex-1 p-1" />
    </div>
  );
}
