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
import { toast } from "sonner";

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
  const [sourceStorages, setSourceStorages] = useState<StorageOption[]>([]);
  const [loadingStorages, setLoadingStorages] = useState(false);
  const [storageError, setStorageError] = useState("");
  const [useSourceAsFallback, setUseSourceAsFallback] = useState(false);
  const [manualStorage, setManualStorage] = useState("");
  const [mode, setMode] = useState<MigrationMode>("snapshot");
  const [newVmid, setNewVmid] = useState<string>("");
  const [cleanupSource, setCleanupSource] = useState(true);
  const [cleanupTarget, setCleanupTarget] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const vmTags = parseTags(vm.tags);

  // Filter: only online PVE nodes, not the source
  const targetNodes = nodes.filter(
    (n) => n.id !== sourceNodeId && n.is_online && n.type === "pve"
  );

  useEffect(() => {
    if (!open) {
      setStep("target");
      setTargetNodeId("");
      setTargetStorage("");
      setStorages([]);
      setSourceStorages([]);
      setMode("snapshot");
      setNewVmid("");
      setError("");
      setStorageError("");
      setUseSourceAsFallback(false);
      setManualStorage("");
    }
  }, [open]);

  // Load source storages to detect VM's current storage type
  useEffect(() => {
    if (!open || !sourceNodeId) return;
    api
      .get(`/nodes/${sourceNodeId}/storage`)
      .then((res) => {
        setSourceStorages(toArray<StorageOption>(res.data));
      })
      .catch(() => setSourceStorages([]));
  }, [open, sourceNodeId]);

  // Load storages when target node changes
  useEffect(() => {
    if (!targetNodeId) return;
    setLoadingStorages(true);
    setTargetStorage("");
    setStorageError("");
    setUseSourceAsFallback(false);
    setManualStorage("");
    api
      .get(`/nodes/${targetNodeId}/storage`)
      .then((res) => {
        const all = toArray<StorageOption>(res.data);
        setStorages(all);
        setStorageError("");
      })
      .catch((err) => {
        const status = err?.response?.status;
        const serverMsg =
          err?.response?.data?.error || err?.response?.data?.message || "";
        console.error(`[Migration] Storage fetch failed for node ${targetNodeId}:`, status, serverMsg, err?.response?.data);

        // Fallback: use source node's storages (same names usually exist on target)
        if (sourceStorages.length > 0) {
          setStorages(sourceStorages);
          setUseSourceAsFallback(true);
          setStorageError("");
          toast.info("Ziel-Storages nicht ladbar - verwende Source-Storages als Referenz");
        } else {
          setStorages([]);
          setStorageError(
            `Storage-Abfrage fehlgeschlagen (${status || "Netzwerk"}): ${serverMsg || "Unbekannter Fehler"}. Du kannst den Storage-Namen manuell eingeben.`
          );
          toast.error("Storages konnten nicht geladen werden");
        }
      })
      .finally(() => setLoadingStorages(false));
  }, [targetNodeId, sourceStorages]);

  // Filter storages: only those that can hold VM images/rootdir
  const vmStorages = useMemo(() => {
    const contentNeeded = vm.type === "lxc" ? "rootdir" : "images";
    return storages.filter(
      (s) => s.active && s.content.includes(contentNeeded)
    );
  }, [storages, vm.type]);

  // Suggest best matching storage (same type as source, or shared)
  const suggestedStorage = useMemo(() => {
    if (vmStorages.length === 0) return null;
    // Prefer shared storage
    const shared = vmStorages.find((s) => s.shared && s.available > 0);
    if (shared) return shared.storage;
    // Prefer same storage type as source storages
    const sourceTypes = sourceStorages.map((s) => s.type);
    const sameType = vmStorages.find(
      (s) => sourceTypes.includes(s.type) && s.available > 0
    );
    if (sameType) return sameType.storage;
    // Fallback: most available space
    const sorted = [...vmStorages].sort((a, b) => b.available - a.available);
    return sorted[0]?.storage || null;
  }, [vmStorages, sourceStorages]);

  // Auto-select suggested storage
  useEffect(() => {
    if (suggestedStorage && !targetStorage) {
      setTargetStorage(suggestedStorage);
    }
  }, [suggestedStorage, targetStorage]);

  const targetNode = nodes.find((n) => n.id === targetNodeId);
  const targetStatus = targetNodeId ? nodeStatus[targetNodeId] : null;

  // Check if target has enough space for VM
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
              Ziel-Storage auf {targetNode?.name}
              <span className="ml-2 text-xs font-normal text-muted-foreground">
                ({vm.type === "lxc" ? "rootdir" : "images"}-faehig)
              </span>
            </Label>

            {useSourceAsFallback && (
              <div className="flex items-start gap-2 rounded-lg border border-amber-300 bg-amber-50 dark:bg-amber-950/30 p-2.5">
                <AlertTriangle className="h-4 w-4 text-amber-600 mt-0.5 flex-shrink-0" />
                <p className="text-xs text-amber-700 dark:text-amber-400">
                  Ziel-Node Storages nicht ladbar. Zeige Source-Node Storages als Referenz
                  (Storage-Namen sind in Proxmox-Clustern meist identisch).
                </p>
              </div>
            )}

            {loadingStorages ? (
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin" />
                Storages werden geladen...
              </div>
            ) : storageError ? (
              <div className="space-y-3">
                <div className="flex items-start gap-2 rounded-lg border border-destructive/50 bg-destructive/5 p-3">
                  <AlertTriangle className="h-4 w-4 text-destructive mt-0.5 flex-shrink-0" />
                  <p className="text-sm text-destructive">{storageError}</p>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="manual-storage" className="text-sm">Storage-Name manuell eingeben</Label>
                  <Input
                    id="manual-storage"
                    placeholder="z.B. local-lvm, ceph-pool, nfs-share"
                    value={manualStorage}
                    onChange={(e) => {
                      setManualStorage(e.target.value);
                      setTargetStorage(e.target.value);
                    }}
                  />
                  <p className="text-xs text-muted-foreground">
                    Gib den exakten Storage-Namen des Ziel-Nodes ein (z.B. &quot;local-lvm&quot;).
                  </p>
                </div>
              </div>
            ) : vmStorages.length === 0 ? (
              <div className="space-y-2">
                <p className="text-sm text-muted-foreground">
                  Keine kompatiblen Storages auf diesem Node.
                </p>
                {storages.length > 0 && (
                  <p className="text-xs text-amber-600">
                    {storages.length} Storage(s) vorhanden, aber keiner
                    unterstuetzt{" "}
                    {vm.type === "lxc" ? "rootdir" : "images"}-Content.
                    Verfuegbare Typen:{" "}
                    {[...new Set(storages.map((s) => `${s.storage} (${s.content})`))].join(", ")}
                  </p>
                )}
                <div className="space-y-2 pt-2">
                  <Label htmlFor="manual-storage-2" className="text-sm">Storage-Name manuell eingeben</Label>
                  <Input
                    id="manual-storage-2"
                    placeholder="z.B. local-lvm"
                    value={manualStorage}
                    onChange={(e) => {
                      setManualStorage(e.target.value);
                      setTargetStorage(e.target.value);
                    }}
                  />
                </div>
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

            {!hasEnoughSpace && (
              <div className="flex items-center gap-2 text-sm text-amber-600 bg-amber-50 dark:bg-amber-950/30 rounded-lg p-3">
                <AlertTriangle className="h-4 w-4 flex-shrink-0" />
                <span>
                  Der Ziel-Storage hat moeglicherweise nicht genug Platz (
                  {formatBytes(selectedStorage?.available ?? 0)} frei,{" "}
                  {formatBytes(vmDiskSize)} benoetigt).
                </span>
              </div>
            )}

            {mode === "stop" && (
              <div className="flex items-center gap-2 text-xs text-amber-600 bg-amber-50 dark:bg-amber-950/30 rounded-lg p-2">
                <AlertTriangle className="h-3.5 w-3.5 flex-shrink-0" />
                Die VM wird waehrend der Migration heruntergefahren und ist
                nicht erreichbar.
              </div>
            )}

            {hasEnoughSpace && (
              <div className="flex items-center gap-2 text-xs text-green-600">
                <CheckCircle2 className="h-3.5 w-3.5" />
                Alle Voraussetzungen erfuellt. Tags und Konfiguration werden
                uebernommen.
              </div>
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
