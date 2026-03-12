"use client";

import { AlertTriangle, RefreshCw, WifiOff, ShieldX, Terminal, Clock, FolderX } from "lucide-react";
import { Button } from "@/components/ui/button";

interface CockpitErrorProps {
  errorCode: string;
  message: string;
  details?: string;
  hint?: string;
  onRetry?: () => void;
}

const errorIcons: Record<string, React.ElementType> = {
  VM_GUEST_AGENT_UNAVAILABLE: Terminal,
  VM_NOT_RUNNING: WifiOff,
  NODE_SSH_FAILED: WifiOff,
  NODE_UNREACHABLE: WifiOff,
  VM_PERMISSION_DENIED: ShieldX,
  VM_COMMAND_TIMEOUT: Clock,
  VM_PATH_INVALID: FolderX,
  VM_COMMAND_FAILED: AlertTriangle,
  VM_EXEC_FAILED: AlertTriangle,
};

export function CockpitError({ errorCode, message, details, hint, onRetry }: CockpitErrorProps) {
  const Icon = errorIcons[errorCode] ?? AlertTriangle;

  return (
    <div className="flex flex-col items-center justify-center gap-4 py-12 px-6 text-center">
      <div className="rounded-full bg-destructive/10 p-4">
        <Icon className="h-8 w-8 text-destructive" />
      </div>
      <div className="space-y-2 max-w-md">
        <h3 className="font-semibold text-lg">{message}</h3>
        {details && (
          <p className="text-sm text-muted-foreground">{details}</p>
        )}
        {hint && (
          <div className="mt-3 rounded-md bg-muted p-3 text-left">
            <p className="text-xs font-mono text-muted-foreground">{hint}</p>
          </div>
        )}
      </div>
      {onRetry && (
        <Button variant="outline" size="sm" onClick={onRetry} className="mt-2">
          <RefreshCw className="mr-2 h-4 w-4" />
          Erneut versuchen
        </Button>
      )}
    </div>
  );
}
