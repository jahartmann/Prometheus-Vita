"use client";

import { useEffect, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { nodeApi, vzdumpApi, toArray } from "@/lib/api";
import type { Node, VM } from "@/types/api";

interface StorageInfo {
  storage: string;
  type: string;
  content: string;
  total: number;
  used: number;
  available: number;
}

interface VzdumpDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function VzdumpDialog({ open, onOpenChange }: VzdumpDialogProps) {
  const [nodes, setNodes] = useState<Node[]>([]);
  const [selectedNodeId, setSelectedNodeId] = useState("");
  const [vms, setVMs] = useState<VM[]>([]);
  const [storages, setStorages] = useState<StorageInfo[]>([]);
  const [vmid, setVmid] = useState("");
  const [storage, setStorage] = useState("");
  const [mode, setMode] = useState("snapshot");
  const [compress, setCompress] = useState("zstd");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [result, setResult] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      nodeApi.list().then((res) => {
        setNodes(toArray<Node>(res.data));
      });
    }
  }, [open]);

  useEffect(() => {
    if (!selectedNodeId) {
      setVMs([]);
      setStorages([]);
      return;
    }
    nodeApi.getVMs(selectedNodeId).then((res) => {
      setVMs(toArray<VM>(res.data));
    });
    nodeApi.getStorage(selectedNodeId).then((res) => {
      setStorages(toArray<StorageInfo>(res.data));
    });
  }, [selectedNodeId]);

  const handleSubmit = async () => {
    if (!selectedNodeId || !vmid) return;
    setIsSubmitting(true);
    setError(null);
    setResult(null);
    try {
      const res = await vzdumpApi.create(selectedNodeId, {
        vmid: parseInt(vmid),
        storage: storage || undefined,
        mode,
        compress,
      });
      const data = res.data?.data || res.data;
      setResult(data?.upid || JSON.stringify(data));
    } catch {
      setError("Vzdump-Backup konnte nicht gestartet werden.");
    }
    setIsSubmitting(false);
  };

  const handleClose = () => {
    setResult(null);
    setError(null);
    setSelectedNodeId("");
    setVmid("");
    setStorage("");
    setMode("snapshot");
    setCompress("zstd");
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Vzdump-Backup erstellen</DialogTitle>
          <DialogDescription>
            VM/CT-Backup ueber die Proxmox vzdump-Schnittstelle starten.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label>Node</Label>
            <Select value={selectedNodeId} onValueChange={setSelectedNodeId}>
              <SelectTrigger>
                <SelectValue placeholder="Node waehlen..." />
              </SelectTrigger>
              <SelectContent>
                {nodes.map((n) => (
                  <SelectItem key={n.id} value={n.id}>
                    {n.name} ({n.hostname})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label>VM / Container</Label>
            <Select value={vmid} onValueChange={setVmid} disabled={!selectedNodeId}>
              <SelectTrigger>
                <SelectValue placeholder="VM waehlen..." />
              </SelectTrigger>
              <SelectContent>
                {vms.map((vm) => (
                  <SelectItem key={vm.vmid} value={String(vm.vmid)}>
                    {vm.vmid} - {vm.name} ({vm.type}, {vm.status})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label>Storage (optional)</Label>
            <Select value={storage} onValueChange={setStorage} disabled={!selectedNodeId}>
              <SelectTrigger>
                <SelectValue placeholder="Standard-Storage" />
              </SelectTrigger>
              <SelectContent>
                {storages.map((s) => (
                  <SelectItem key={s.storage} value={s.storage}>
                    {s.storage} ({s.type})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label>Modus</Label>
            <div className="flex gap-2">
              {(["snapshot", "suspend", "stop"] as const).map((m) => (
                <Button
                  key={m}
                  variant={mode === m ? "default" : "outline"}
                  size="sm"
                  onClick={() => setMode(m)}
                >
                  {m === "snapshot" ? "Snapshot" : m === "suspend" ? "Suspend" : "Stop"}
                </Button>
              ))}
            </div>
          </div>

          <div className="space-y-2">
            <Label>Kompression</Label>
            <Select value={compress} onValueChange={setCompress}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="zstd">zstd</SelectItem>
                <SelectItem value="lzo">lzo</SelectItem>
                <SelectItem value="gzip">gzip</SelectItem>
                <SelectItem value="none">Keine</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {result && (
            <div className="rounded border border-green-500/30 bg-green-500/10 p-3">
              <p className="text-sm font-medium text-green-600">Backup gestartet</p>
              <p className="mt-1 text-xs font-mono break-all">{result}</p>
            </div>
          )}

          {error && (
            <div className="rounded border border-destructive/30 bg-destructive/10 p-3">
              <p className="text-sm text-destructive">{error}</p>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose}>
            Schliessen
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={isSubmitting || !selectedNodeId || !vmid}
          >
            {isSubmitting ? "Wird gestartet..." : "Backup starten"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
