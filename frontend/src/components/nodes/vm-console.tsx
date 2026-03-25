"use client";

import { useState } from "react";
import { ExternalLink, Loader2, Terminal } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { vmApi } from "@/lib/api";

interface VmConsoleProps {
  nodeId: string;
  vmid: number;
  vmType: string;
  vmName: string;
  hostname: string;
  port: number;
  pveNode?: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function VmConsole({
  nodeId,
  vmid,
  vmType,
  vmName,
  hostname,
  port,
  pveNode,
  open,
  onOpenChange,
}: VmConsoleProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const openConsole = async () => {
    setLoading(true);
    setError(null);
    try {
      await vmApi.getVNCProxy(nodeId, vmid, vmType);
      const consoleType = vmType === "qemu" ? "kvm" : "lxc";
      const nodeName = pveNode || hostname;
      const url = `https://${hostname}:${port}/?console=${consoleType}&novnc=1&vmid=${vmid}&vmname=${encodeURIComponent(vmName)}&node=${encodeURIComponent(nodeName)}&resize=off`;
      window.open(url, "_blank");
    } catch {
      setError("VNC-Proxy konnte nicht erstellt werden.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Terminal className="h-5 w-5" />
            Konsole: {vmName} (ID: {vmid})
          </DialogTitle>
          <DialogDescription>
            Öffnet die Proxmox-Konsole in einem neuen Tab.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="rounded-lg border p-4 text-sm text-muted-foreground space-y-1">
            <p>Host: {hostname}:{port}</p>
            <p>Typ: {vmType === "qemu" ? "KVM Virtual Machine" : "LXC Container"}</p>
          </div>

          {error && (
            <p className="text-sm text-destructive">{error}</p>
          )}

          <Button
            onClick={openConsole}
            disabled={loading}
            className="w-full"
          >
            {loading ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <ExternalLink className="mr-2 h-4 w-4" />
            )}
            Konsole öffnen
          </Button>

          <p className="text-xs text-muted-foreground">
            Die Konsole wird über die Proxmox-noVNC-Oberfläche im Browser geöffnet.
          </p>
        </div>
      </DialogContent>
    </Dialog>
  );
}
