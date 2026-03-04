"use client";

import { useState, useEffect } from "react";
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
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { useNodeStore } from "@/stores/node-store";
import { useMigrationStore } from "@/stores/migration-store";
import { formatBytes } from "@/lib/utils";
import api, { toArray } from "@/lib/api";
import type { VM, MigrationMode } from "@/types/api";
import { ArrowRight, Loader2 } from "lucide-react";

interface MigrateVmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  vm: VM;
  sourceNodeId: string;
  sourceNodeName: string;
}

interface StorageOption {
  storage: string;
  type: string;
  total: number;
  used: number;
  available: number;
}

type Step = "target" | "storage" | "options" | "confirm";

const MODES: { value: MigrationMode; label: string; description: string }[] = [
  {
    value: "snapshot",
    label: "Snapshot",
    description: "Live-Backup ohne Downtime, potenziell inkonsistent",
  },
  {
    value: "suspend",
    label: "Suspend",
    description: "Kurze Pause, guter Kompromiss",
  },
  {
    value: "stop",
    label: "Stop",
    description: "VM herunterfahren, konsistentestes Backup",
  },
];

export function MigrateVmDialog({
  open,
  onOpenChange,
  vm,
  sourceNodeId,
  sourceNodeName,
}: MigrateVmDialogProps) {
  const { nodes } = useNodeStore();
  const { startMigration } = useMigrationStore();

  const [step, setStep] = useState<Step>("target");
  const [targetNodeId, setTargetNodeId] = useState("");
  const [targetStorage, setTargetStorage] = useState("");
  const [storages, setStorages] = useState<StorageOption[]>([]);
  const [loadingStorages, setLoadingStorages] = useState(false);
  const [mode, setMode] = useState<MigrationMode>("snapshot");
  const [newVmid, setNewVmid] = useState<string>("");
  const [cleanupSource, setCleanupSource] = useState(true);
  const [cleanupTarget, setCleanupTarget] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  // Filter: only online PVE nodes, not the source
  const targetNodes = nodes.filter(
    (n) => n.id !== sourceNodeId && n.is_online && n.type === "pve"
  );

  useEffect(() => {
    if (!open) {
      setStep("target");
      setTargetNodeId("");
      setTargetStorage("");
      setMode("snapshot");
      setNewVmid("");
      setError("");
    }
  }, [open]);

  // Load storages when target node changes
  useEffect(() => {
    if (!targetNodeId) return;
    setLoadingStorages(true);
    api
      .get(`/nodes/${targetNodeId}/storage`)
      .then((res) => {
        const data = toArray<StorageOption>(res.data);
        setStorages(
          data.filter(
            (s: StorageOption) =>
              s.type !== "dir" || s.available > 0
          )
        );
      })
      .catch(() => setStorages([]))
      .finally(() => setLoadingStorages(false));
  }, [targetNodeId]);

  const targetNode = nodes.find((n) => n.id === targetNodeId);

  const handleSubmit = async () => {
    setSubmitting(true);
    setError("");
    try {
      await startMigration({
        source_node_id: sourceNodeId,
        target_node_id: targetNodeId,
        vmid: vm.vmid,
        target_storage: targetStorage,
        mode,
        new_vmid: newVmid ? parseInt(newVmid) : undefined,
        cleanup_source: cleanupSource,
        cleanup_target: cleanupTarget,
      });
      onOpenChange(false);
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : "Migration konnte nicht gestartet werden";
      setError(msg);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>VM migrieren</DialogTitle>
          <DialogDescription>
            VM {vm.vmid} ({vm.name}) von {sourceNodeName} migrieren
          </DialogDescription>
        </DialogHeader>

        {/* Step indicator */}
        <div className="flex items-center gap-2 text-xs text-muted-foreground mb-2">
          {["Ziel-Node", "Storage", "Optionen", "Bestaetigung"].map(
            (label, i) => (
              <div key={label} className="flex items-center gap-1">
                <span
                  className={
                    i <=
                    ["target", "storage", "options", "confirm"].indexOf(step)
                      ? "font-bold text-foreground"
                      : ""
                  }
                >
                  {label}
                </span>
                {i < 3 && <ArrowRight className="h-3 w-3" />}
              </div>
            )
          )}
        </div>

        {/* STEP 1: Target Node */}
        {step === "target" && (
          <div className="space-y-3">
            <Label>Ziel-Node waehlen</Label>
            {targetNodes.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                Keine weiteren Online-Nodes verfuegbar.
              </p>
            ) : (
              <div className="grid gap-2">
                {targetNodes.map((node) => (
                  <button
                    key={node.id}
                    type="button"
                    className={`flex items-center justify-between p-3 rounded-lg border text-left transition-colors ${
                      targetNodeId === node.id
                        ? "border-primary bg-primary/5"
                        : "hover:bg-muted/50"
                    }`}
                    onClick={() => setTargetNodeId(node.id)}
                  >
                    <div>
                      <p className="font-medium">{node.name}</p>
                      <p className="text-xs text-muted-foreground">
                        {node.hostname}
                      </p>
                    </div>
                    <Badge variant="success">online</Badge>
                  </button>
                ))}
              </div>
            )}
            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => onOpenChange(false)}
              >
                Abbrechen
              </Button>
              <Button
                disabled={!targetNodeId}
                onClick={() => setStep("storage")}
              >
                Weiter
              </Button>
            </DialogFooter>
          </div>
        )}

        {/* STEP 2: Storage */}
        {step === "storage" && (
          <div className="space-y-3">
            <Label>Ziel-Storage auf {targetNode?.name}</Label>
            {loadingStorages ? (
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin" />
                Storages werden geladen...
              </div>
            ) : storages.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                Keine Storages verfuegbar.
              </p>
            ) : (
              <div className="grid gap-2">
                {storages.map((s) => (
                  <button
                    key={s.storage}
                    type="button"
                    className={`flex items-center justify-between p-3 rounded-lg border text-left transition-colors ${
                      targetStorage === s.storage
                        ? "border-primary bg-primary/5"
                        : "hover:bg-muted/50"
                    }`}
                    onClick={() => setTargetStorage(s.storage)}
                  >
                    <div>
                      <p className="font-medium">{s.storage}</p>
                      <p className="text-xs text-muted-foreground">{s.type}</p>
                    </div>
                    <div className="text-right text-xs text-muted-foreground">
                      <p>{formatBytes(s.available)} frei</p>
                      <p>
                        {formatBytes(s.used)} / {formatBytes(s.total)}
                      </p>
                    </div>
                  </button>
                ))}
              </div>
            )}
            <DialogFooter>
              <Button variant="outline" onClick={() => setStep("target")}>
                Zurueck
              </Button>
              <Button
                disabled={!targetStorage}
                onClick={() => setStep("options")}
              >
                Weiter
              </Button>
            </DialogFooter>
          </div>
        )}

        {/* STEP 3: Options */}
        {step === "options" && (
          <div className="space-y-4">
            <div>
              <Label>Migrations-Modus</Label>
              <div className="grid gap-2 mt-2">
                {MODES.map((m) => (
                  <button
                    key={m.value}
                    type="button"
                    className={`flex flex-col p-3 rounded-lg border text-left transition-colors ${
                      mode === m.value
                        ? "border-primary bg-primary/5"
                        : "hover:bg-muted/50"
                    }`}
                    onClick={() => setMode(m.value)}
                  >
                    <p className="font-medium">{m.label}</p>
                    <p className="text-xs text-muted-foreground">
                      {m.description}
                    </p>
                  </button>
                ))}
              </div>
            </div>

            <div>
              <Label htmlFor="new-vmid">
                Neue VMID (optional, Standard: {vm.vmid})
              </Label>
              <Input
                id="new-vmid"
                type="number"
                placeholder={String(vm.vmid)}
                value={newVmid}
                onChange={(e) => setNewVmid(e.target.value)}
                className="mt-1"
              />
            </div>

            <div className="flex items-center gap-4">
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={cleanupSource}
                  onChange={(e) => setCleanupSource(e.target.checked)}
                  className="rounded"
                />
                Vzdump auf Source loeschen
              </label>
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={cleanupTarget}
                  onChange={(e) => setCleanupTarget(e.target.checked)}
                  className="rounded"
                />
                Vzdump auf Target loeschen
              </label>
            </div>

            <DialogFooter>
              <Button variant="outline" onClick={() => setStep("storage")}>
                Zurueck
              </Button>
              <Button onClick={() => setStep("confirm")}>Weiter</Button>
            </DialogFooter>
          </div>
        )}

        {/* STEP 4: Confirm */}
        {step === "confirm" && (
          <div className="space-y-4">
            <div className="rounded-lg border p-4 space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-muted-foreground">VM</span>
                <span className="font-medium">
                  {vm.vmid} ({vm.name})
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Von</span>
                <span>{sourceNodeName}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Nach</span>
                <span>{targetNode?.name}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Storage</span>
                <span>{targetStorage}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Modus</span>
                <Badge variant="outline">{mode}</Badge>
              </div>
              {newVmid && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Neue VMID</span>
                  <span>{newVmid}</span>
                </div>
              )}
            </div>

            {mode === "stop" && (
              <p className="text-xs text-amber-600">
                Die VM wird waehrend der Migration heruntergefahren und ist
                nicht erreichbar.
              </p>
            )}

            {error && (
              <p className="text-sm text-destructive">{error}</p>
            )}

            <DialogFooter>
              <Button variant="outline" onClick={() => setStep("options")}>
                Zurueck
              </Button>
              <Button onClick={handleSubmit} disabled={submitting}>
                {submitting && (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                )}
                Migration starten
              </Button>
            </DialogFooter>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
