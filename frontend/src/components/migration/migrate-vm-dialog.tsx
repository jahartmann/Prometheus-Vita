"use client";

import { useState, useEffect, useMemo } from "react";
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
import {
  ArrowRight,
  Loader2,
  HardDrive,
  Cpu,
  MemoryStick,
  Tag,
  CheckCircle2,
  AlertTriangle,
} from "lucide-react";

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
  content: string;
  total: number;
  used: number;
  available: number;
  usage_percent: number;
  active: boolean;
  shared: boolean;
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

function parseTags(tags: string | string[] | undefined): string[] {
  if (!tags) return [];
  if (Array.isArray(tags)) return tags;
  return tags
    .split(/[;,]/)
    .map((t) => t.trim())
    .filter(Boolean);
}

export function MigrateVmDialog({
  open,
  onOpenChange,
  vm,
  sourceNodeId,
  sourceNodeName,
}: MigrateVmDialogProps) {
  const { nodes, nodeStatus } = useNodeStore();
  const { startMigration } = useMigrationStore();

  const [step, setStep] = useState<Step>("target");
  const [targetNodeId, setTargetNodeId] = useState("");
  const [targetStorage, setTargetStorage] = useState("");
  const [storages, setStorages] = useState<StorageOption[]>([]);
  const [loadingStorages, setLoadingStorages] = useState(false);
  const [storageError, setStorageError] = useState("");
  const [targetResources, setTargetResources] = useState<{
    cpuFree: number;
    memFree: number;
    memTotal: number;
  } | null>(null);
  const [mode, setMode] = useState<MigrationMode>("snapshot");
  const [newVmid, setNewVmid] = useState<string>("");
  const [cleanupSource, setCleanupSource] = useState(true);
  const [cleanupTarget, setCleanupTarget] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const vmTags = parseTags(vm.tags);

  const targetNodes = nodes.filter(
    (n) => n.id !== sourceNodeId && n.is_online && n.type === "pve"
  );

  useEffect(() => {
    if (!open) {
      setStep("target");
      setTargetNodeId("");
      setTargetStorage("");
      setStorages([]);
      setMode("snapshot");
      setNewVmid("");
      setError("");
      setStorageError("");
    }
  }, [open]);

  // Load target node storages when target node is selected
  useEffect(() => {
    if (!targetNodeId) {
      setStorages([]);
      setTargetStorage("");
      setStorageError("");
      return;
    }

    let cancelled = false;
    const loadStorages = async (attempt = 1) => {
      setLoadingStorages(true);
      setStorageError("");
      try {
        const res = await api.get(`/nodes/${targetNodeId}/storage`);
        if (!cancelled) {
          setStorages(toArray<StorageOption>(res.data));
          setTargetStorage(""); // reset selection when node changes
        }
      } catch (err: unknown) {
        if (!cancelled) {
          if (attempt < 3) {
            // Retry after 2 seconds
            setTimeout(() => loadStorages(attempt + 1), 2000);
            return;
          }
          const errObj = err as { response?: { data?: { error?: string } }; message?: string };
          const msg = errObj?.response?.data?.error || errObj?.message || "Unbekannt";
          setStorageError(`Storage-Abfrage fehlgeschlagen (${attempt} Versuche): ${msg}`);
          setStorages([]);
        }
      } finally {
        if (!cancelled) setLoadingStorages(false);
      }
    };

    loadStorages();
    return () => { cancelled = true; };
  }, [targetNodeId]);

  // Load target node resources when target is selected
  useEffect(() => {
    if (!targetNodeId) {
      setTargetResources(null);
      return;
    }
    const st = nodeStatus[targetNodeId];
    if (st) {
      setTargetResources({
        cpuFree: 100 - st.cpu_usage,
        memFree: st.memory_total - st.memory_used,
        memTotal: st.memory_total,
      });
    }
  }, [targetNodeId, nodeStatus]);

  // Filter storages: only those that can hold VM images/rootdir
  const vmStorages = useMemo(() => {
    const contentNeeded = vm.type === "lxc" ? "rootdir" : "images";
    const filtered = storages.filter(
      (s) => s.active && s.content.includes(contentNeeded)
    );
    return filtered.length > 0 ? filtered : storages.filter((s) => s.active);
  }, [storages, vm.type]);

  // Suggest best matching storage
  const suggestedStorage = useMemo(() => {
    if (vmStorages.length === 0) return null;
    const shared = vmStorages.find((s) => s.shared && s.available > 0);
    if (shared) return shared.storage;
    const sorted = [...vmStorages].sort((a, b) => b.available - a.available);
    return sorted[0]?.storage || null;
  }, [vmStorages]);

  // Auto-select suggested storage
  useEffect(() => {
    if (suggestedStorage && !targetStorage) {
      setTargetStorage(suggestedStorage);
    }
  }, [suggestedStorage, targetStorage]);

  const targetNode = nodes.find((n) => n.id === targetNodeId);
  const selectedStorage = vmStorages.find((s) => s.storage === targetStorage);
  const vmDiskSize = vm.disk_total || 0;
  const hasEnoughSpace =
    !selectedStorage || selectedStorage.available >= vmDiskSize;

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
        err instanceof Error
          ? err.message
          : "Migration konnte nicht gestartet werden";
      setError(msg);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {vm.type === "lxc" ? "Container" : "VM"} migrieren
          </DialogTitle>
          <DialogDescription>
            {vm.type === "lxc" ? "CT" : "VM"} {vm.vmid} ({vm.name}) von{" "}
            {sourceNodeName} migrieren
          </DialogDescription>
        </DialogHeader>

        {/* VM Info Summary */}
        <div className="flex flex-wrap items-center gap-2 rounded-lg border bg-muted/30 p-3 text-sm">
          <div className="flex items-center gap-1">
            <Cpu className="h-3.5 w-3.5 text-blue-500" />
            <span>{vm.cpu_cores} Cores</span>
          </div>
          <span className="text-muted-foreground">·</span>
          <div className="flex items-center gap-1">
            <MemoryStick className="h-3.5 w-3.5 text-purple-500" />
            <span>{formatBytes(vm.memory_total)}</span>
          </div>
          <span className="text-muted-foreground">·</span>
          <div className="flex items-center gap-1">
            <HardDrive className="h-3.5 w-3.5 text-orange-500" />
            <span>{formatBytes(vm.disk_total)}</span>
          </div>
          {vmTags.length > 0 && (
            <>
              <span className="text-muted-foreground">·</span>
              <div className="flex items-center gap-1">
                <Tag className="h-3.5 w-3.5 text-muted-foreground" />
                {vmTags.map((tag) => (
                  <Badge key={tag} variant="secondary" className="text-xs">
                    {tag}
                  </Badge>
                ))}
              </div>
            </>
          )}
          <Badge
            variant={vm.status === "running" ? "success" : "secondary"}
            className="ml-auto"
          >
            {vm.status}
          </Badge>
        </div>

        {/* Step indicator */}
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          {["Ziel-Node", "Storage", "Optionen", "Bestätigung"].map(
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
            <Label>Ziel-Node wählen</Label>
            {targetNodes.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                Keine weiteren Online-Nodes verfügbar.
              </p>
            ) : (
              <div className="grid gap-2">
                {targetNodes.map((node) => {
                  const st = nodeStatus[node.id];
                  return (
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
                        {st && (
                          <p className="text-xs text-muted-foreground mt-0.5">
                            CPU: {st.cpu_usage.toFixed(1)}% · RAM:{" "}
                            {st.memory_total > 0
                              ? (
                                  (st.memory_used / st.memory_total) *
                                  100
                                ).toFixed(1)
                              : 0}
                            % · VMs: {st.vm_running}/{st.vm_count}
                          </p>
                        )}
                      </div>
                      <Badge variant="success">online</Badge>
                    </button>
                  );
                })}
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
            <Label>
              Ziel-Storage
              <span className="ml-2 text-xs font-normal text-muted-foreground">
                ({vm.type === "lxc" ? "rootdir" : "images"}-fähig)
              </span>
            </Label>

            <div className="text-xs text-muted-foreground bg-muted/50 rounded-lg p-2 mb-2">
              <span className="font-medium">Aktuelle VM-Disk:</span>{" "}
              {formatBytes(vm.disk_total)} auf {sourceNodeName}
            </div>

            {loadingStorages ? (
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin" />
                Storages werden geladen...
              </div>
            ) : storageError ? (
              <div className="flex items-start gap-2 rounded-lg border border-destructive/50 bg-destructive/5 p-3">
                <AlertTriangle className="h-4 w-4 text-destructive mt-0.5 flex-shrink-0" />
                <p className="text-sm text-destructive">{storageError}</p>
              </div>
            ) : vmStorages.length === 0 ? (
              <div className="text-sm text-muted-foreground space-y-1">
                <p>Keine kompatiblen Storages auf dem Ziel-Node gefunden.</p>
                <p className="text-xs">
                  Benötigter Content-Typ: <code className="bg-muted px-1 rounded">{vm.type === "lxc" ? "rootdir" : "images"}</code>
                </p>
                {storages.length > 0 && (
                  <p className="text-xs">
                    Verfügbare Storages: {storages.map(s => s.storage).join(", ")}
                  </p>
                )}
              </div>
            ) : (
              <div className="grid gap-2">
                {vmStorages.map((s) => {
                  const isSuggested = s.storage === suggestedStorage;
                  const tooSmall = vmDiskSize > 0 && s.available < vmDiskSize;
                  return (
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
                        <div className="flex items-center gap-2">
                          <p className="font-medium">{s.storage}</p>
                          <Badge variant="outline" className="text-xs">
                            {s.type}
                          </Badge>
                          {s.shared && (
                            <Badge
                              variant="secondary"
                              className="text-xs"
                            >
                              shared
                            </Badge>
                          )}
                          {isSuggested && (
                            <Badge className="text-xs bg-green-100 text-green-700 border-green-300">
                              empfohlen
                            </Badge>
                          )}
                        </div>
                        <p className="text-xs text-muted-foreground mt-0.5">
                          {s.content}
                        </p>
                      </div>
                      <div className="text-right text-xs">
                        <p
                          className={
                            tooSmall
                              ? "text-destructive font-medium"
                              : "text-muted-foreground"
                          }
                        >
                          {formatBytes(s.available)} frei
                        </p>
                        <p className="text-muted-foreground">
                          {formatBytes(s.used)} / {formatBytes(s.total)}
                        </p>
                        {tooSmall && (
                          <p className="text-destructive text-xs mt-0.5 flex items-center gap-1">
                            <AlertTriangle className="h-3 w-3" />
                            Zu wenig Platz
                          </p>
                        )}
                      </div>
                    </button>
                  );
                })}
              </div>
            )}
            <DialogFooter>
              <Button variant="outline" onClick={() => setStep("target")}>
                Zurück
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
                Vzdump auf Source löschen
              </label>
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={cleanupTarget}
                  onChange={(e) => setCleanupTarget(e.target.checked)}
                  className="rounded"
                />
                Vzdump auf Target löschen
              </label>
            </div>

            <DialogFooter>
              <Button variant="outline" onClick={() => setStep("storage")}>
                Zurück
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
                <span className="text-muted-foreground">
                  {vm.type === "lxc" ? "Container" : "VM"}
                </span>
                <span className="font-medium">
                  {vm.vmid} ({vm.name})
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Ressourcen</span>
                <span>
                  {vm.cpu_cores} CPU · {formatBytes(vm.memory_total)} RAM ·{" "}
                  {formatBytes(vm.disk_total)} Disk
                </span>
              </div>
              {vmTags.length > 0 && (
                <div className="flex justify-between items-center">
                  <span className="text-muted-foreground">Tags</span>
                  <div className="flex gap-1">
                    {vmTags.map((tag) => (
                      <Badge key={tag} variant="secondary" className="text-xs">
                        {tag}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}
              <hr className="my-1" />
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
                <span>
                  {targetStorage}
                  {selectedStorage && (
                    <span className="text-muted-foreground ml-1">
                      ({selectedStorage.type}, {formatBytes(selectedStorage.available)} frei)
                    </span>
                  )}
                </span>
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

            {targetResources && vm.memory_total > targetResources.memFree && (
              <div className="flex items-center gap-2 text-sm text-amber-600 bg-amber-50 dark:bg-amber-950/30 rounded-lg p-3">
                <AlertTriangle className="h-4 w-4 flex-shrink-0" />
                <span>
                  Ziel-Node hat möglicherweise nicht genug RAM (
                  {formatBytes(targetResources.memFree)} frei, VM benötigt{" "}
                  {formatBytes(vm.memory_total)}).
                </span>
              </div>
            )}

            {!hasEnoughSpace && (
              <div className="flex items-center gap-2 text-sm text-amber-600 bg-amber-50 dark:bg-amber-950/30 rounded-lg p-3">
                <AlertTriangle className="h-4 w-4 flex-shrink-0" />
                <span>
                  Der Ziel-Storage hat möglicherweise nicht genug Platz (
                  {formatBytes(selectedStorage?.available ?? 0)} frei,{" "}
                  {formatBytes(vmDiskSize)} benötigt).
                </span>
              </div>
            )}

            {mode === "stop" && (
              <div className="flex items-center gap-2 text-xs text-amber-600 bg-amber-50 dark:bg-amber-950/30 rounded-lg p-2">
                <AlertTriangle className="h-3.5 w-3.5 flex-shrink-0" />
                Die VM wird während der Migration heruntergefahren und ist
                nicht erreichbar.
              </div>
            )}

            {hasEnoughSpace && (
              <div className="flex items-center gap-2 text-xs text-green-600">
                <CheckCircle2 className="h-3.5 w-3.5" />
                Alle Voraussetzungen erfuellt. Tags und Konfiguration werden
                übernommen.
              </div>
            )}

            {error && (
              <p className="text-sm text-destructive">{error}</p>
            )}

            <DialogFooter>
              <Button variant="outline" onClick={() => setStep("options")}>
                Zurück
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
