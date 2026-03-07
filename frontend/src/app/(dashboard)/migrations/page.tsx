"use client";

import { useEffect, useState, useCallback, useMemo } from "react";
import { useNodeStore } from "@/stores/node-store";
import { useMigrationStore } from "@/stores/migration-store";
import { useWebSocket } from "@/hooks/use-websocket";
import { MigrationHistory } from "@/components/migration/migration-history";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  ArrowRight,
  ArrowLeft,
  Loader2,
  Play,
  Server,
  Camera,
  Pause,
  Square,
  Check,
  HardDrive,
  Cpu,
  MemoryStick,
  Search,
  AlertTriangle,
  ChevronRight,
} from "lucide-react";
import api, { toArray } from "@/lib/api";
import type { VM, VMMigration, MigrationMode } from "@/types/api";
import { formatBytes, cn } from "@/lib/utils";

// StorageInfo matches the backend proxmox.StorageInfo JSON response
interface StorageInfo {
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

const STEPS = [
  { number: 1, label: "Quell-Auswahl", description: "Node & VM waehlen" },
  { number: 2, label: "Ziel-Auswahl", description: "Ziel-Node & Storage" },
  { number: 3, label: "Optionen", description: "Migrations-Modus" },
  { number: 4, label: "Bestaetigung", description: "Zusammenfassung" },
];

const MODE_CONFIG: {
  value: MigrationMode;
  label: string;
  description: string;
  icon: React.ComponentType<{ className?: string }>;
  accent: string;
  border: string;
  bg: string;
}[] = [
  {
    value: "snapshot",
    label: "Snapshot",
    description: "Live-Backup ohne Downtime",
    icon: Camera,
    accent: "text-green-600 dark:text-green-400",
    border: "border-green-500/50",
    bg: "bg-green-500/10",
  },
  {
    value: "suspend",
    label: "Suspend",
    description: "Kurze Pause, guter Kompromiss",
    icon: Pause,
    accent: "text-amber-600 dark:text-amber-400",
    border: "border-amber-500/50",
    bg: "bg-amber-500/10",
  },
  {
    value: "stop",
    label: "Stop",
    description: "VM herunterfahren, konsistentestes Backup",
    icon: Square,
    accent: "text-red-600 dark:text-red-400",
    border: "border-red-500/50",
    bg: "bg-red-500/10",
  },
];

export default function MigrationsPage() {
  const { nodes, fetchNodes, fetchNodeVMs, nodeVMs } = useNodeStore();
  const {
    migrations,
    activeMigrations,
    fetchMigrations,
    startMigration,
    updateMigrationProgress,
  } = useMigrationStore();

  const [step, setStep] = useState(1);
  const [sourceNodeId, setSourceNodeId] = useState("");
  const [targetNodeId, setTargetNodeId] = useState("");
  const [selectedVmid, setSelectedVmid] = useState("");
  const [targetStorage, setTargetStorage] = useState("");
  const [mode, setMode] = useState<MigrationMode>("snapshot");
  const [newVmid, setNewVmid] = useState("");
  const [cleanupSource, setCleanupSource] = useState(true);
  const [cleanupTarget, setCleanupTarget] = useState(true);
  const [vmSearch, setVmSearch] = useState("");

  // Storages loaded once from the source node.
  // In a Proxmox cluster, storage config is shared across all nodes
  // via /etc/pve/storage.cfg, so source storages = target storages.
  const [storages, setStorages] = useState<StorageInfo[]>([]);
  const [loadingStorages, setLoadingStorages] = useState(false);
  const [storageError, setStorageError] = useState("");

  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState("");

  useEffect(() => {
    if (nodes.length === 0) fetchNodes();
    fetchMigrations();
  }, [nodes.length, fetchNodes, fetchMigrations]);

  // Load VMs and storages when source node changes.
  // We use the source node's API connection for storages because it's proven
  // to work (if we can list VMs, we can list storages from the same node).
  useEffect(() => {
    if (!sourceNodeId) {
      setStorages([]);
      setStorageError("");
      return;
    }
    fetchNodeVMs(sourceNodeId);
    setSelectedVmid("");
    setLoadingStorages(true);
    setStorageError("");

    api
      .get(`/nodes/${sourceNodeId}/storage`)
      .then((res) => {
        const data = toArray<StorageInfo>(res.data);
        // Filter to storages that can hold VM disk images
        const vmStorages = data.filter(
          (s) =>
            s.content.includes("images") ||
            s.content.includes("rootdir") ||
            s.content.includes("backup")
        );
        setStorages(vmStorages.length > 0 ? vmStorages : data);
        setStorageError("");
      })
      .catch((err) => {
        const msg = err?.response?.data?.error || err?.message || "Unbekannt";
        console.error("[Migration] Storage load failed:", msg);
        setStorages([]);
        setStorageError(`Storage-Abfrage fehlgeschlagen: ${msg}`);
      })
      .finally(() => setLoadingStorages(false));
  }, [sourceNodeId, fetchNodeVMs]);

  // Reset target storage when target node changes
  useEffect(() => {
    setTargetStorage("");
  }, [targetNodeId]);

  // WebSocket for live migration updates
  const handleWsMessage = useCallback(
    (data: unknown) => {
      const msg = data as { type?: string; data?: VMMigration };
      if (msg?.type === "migration_progress" && msg.data) {
        updateMigrationProgress(msg.data);
      }
    },
    [updateMigrationProgress]
  );

  useWebSocket({
    url: "/api/v1/ws",
    onMessage: handleWsMessage,
    enabled: activeMigrations.length > 0,
  });

  const onlineNodes = nodes.filter((n) => n.is_online && n.type === "pve");
  const sourceVMs = sourceNodeId ? nodeVMs[sourceNodeId] || [] : [];
  const targetNodes = onlineNodes.filter((n) => n.id !== sourceNodeId);

  const filteredVMs = useMemo(() => {
    if (!vmSearch) return sourceVMs;
    const q = vmSearch.toLowerCase();
    return sourceVMs.filter(
      (vm) =>
        vm.name?.toLowerCase().includes(q) || String(vm.vmid).includes(q)
    );
  }, [sourceVMs, vmSearch]);

  const selectedVM = sourceVMs.find((vm) => String(vm.vmid) === selectedVmid);
  const sourceNode = nodes.find((n) => n.id === sourceNodeId);
  const targetNode = nodes.find((n) => n.id === targetNodeId);
  const selectedStorageInfo = storages.find(
    (s) => s.storage === targetStorage
  );
  const selectedMode = MODE_CONFIG.find((m) => m.value === mode);

  const canProceed = (s: number) => {
    switch (s) {
      case 1:
        return !!sourceNodeId && !!selectedVmid;
      case 2:
        return !!targetNodeId && !!targetStorage;
      case 3:
        return !!mode;
      default:
        return true;
    }
  };

  const handleStart = async () => {
    setSubmitting(true);
    setSubmitError("");
    try {
      await startMigration({
        source_node_id: sourceNodeId,
        target_node_id: targetNodeId,
        vmid: parseInt(selectedVmid),
        target_storage: targetStorage,
        mode,
        new_vmid: newVmid ? parseInt(newVmid) : undefined,
        cleanup_source: cleanupSource,
        cleanup_target: cleanupTarget,
      });
      setStep(1);
      setSourceNodeId("");
      setTargetNodeId("");
      setSelectedVmid("");
      setTargetStorage("");
      setMode("snapshot");
      setNewVmid("");
    } catch (err: unknown) {
      setSubmitError(
        err instanceof Error
          ? err.message
          : "Migration konnte nicht gestartet werden"
      );
    } finally {
      setSubmitting(false);
    }
  };

  const statusColor = (status: string) => {
    switch (status) {
      case "running":
        return "bg-green-500";
      case "stopped":
        return "bg-red-500";
      case "paused":
        return "bg-amber-500";
      default:
        return "bg-gray-500";
    }
  };

  // --- Render ---

  const renderStepIndicator = () => (
    <div className="flex items-center justify-between mb-8">
      {STEPS.map((s, i) => {
        const isActive = step === s.number;
        const isCompleted = step > s.number;
        return (
          <div
            key={s.number}
            className="flex items-center flex-1 last:flex-initial"
          >
            <button
              type="button"
              onClick={() => isCompleted && setStep(s.number)}
              disabled={!isCompleted}
              className={cn(
                "flex items-center gap-3 group",
                isCompleted && "cursor-pointer"
              )}
            >
              <div
                className={cn(
                  "flex h-9 w-9 items-center justify-center rounded-full border-2 text-sm font-semibold transition-all",
                  isActive &&
                    "border-primary bg-primary text-primary-foreground",
                  isCompleted &&
                    "border-primary bg-zinc-100 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100",
                  !isActive &&
                    !isCompleted &&
                    "border-muted-foreground/30 text-muted-foreground"
                )}
              >
                {isCompleted ? <Check className="h-4 w-4" /> : s.number}
              </div>
              <div className="hidden sm:block text-left">
                <p
                  className={cn(
                    "text-sm font-medium leading-none",
                    isActive ? "text-foreground" : "text-muted-foreground"
                  )}
                >
                  {s.label}
                </p>
                <p className="text-xs text-muted-foreground mt-0.5">
                  {s.description}
                </p>
              </div>
            </button>
            {i < STEPS.length - 1 && (
              <div
                className={cn(
                  "flex-1 h-px mx-4",
                  step > s.number ? "bg-primary" : "bg-muted-foreground/20"
                )}
              />
            )}
          </div>
        );
      })}
    </div>
  );

  const renderStep1 = () => (
    <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <div className="lg:col-span-2 space-y-4">
        <div className="space-y-2">
          <Label className="text-sm font-medium">Quell-Node</Label>
          <Select value={sourceNodeId} onValueChange={setSourceNodeId}>
            <SelectTrigger>
              <SelectValue placeholder="Node auswaehlen..." />
            </SelectTrigger>
            <SelectContent>
              {onlineNodes.map((node) => (
                <SelectItem key={node.id} value={node.id}>
                  <div className="flex items-center gap-2">
                    <Server className="h-3.5 w-3.5" />
                    {node.name}
                  </div>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {sourceNodeId && (
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <Label className="text-sm font-medium">VM auswaehlen</Label>
              <div className="relative w-64">
                <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="VM suchen..."
                  value={vmSearch}
                  onChange={(e) => setVmSearch(e.target.value)}
                  className="pl-9 h-9"
                />
              </div>
            </div>

            {sourceVMs.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground text-sm">
                Keine VMs auf diesem Node gefunden.
              </div>
            ) : (
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 max-h-[400px] overflow-y-auto pr-1">
                {filteredVMs.map((vm) => {
                  const isSelected = String(vm.vmid) === selectedVmid;
                  return (
                    <button
                      key={vm.vmid}
                      type="button"
                      onClick={() => setSelectedVmid(String(vm.vmid))}
                      className={cn(
                        "flex items-start gap-3 rounded-lg border p-3 text-left transition-all hover:bg-accent/50",
                        isSelected
                          ? "border-primary ring-2 ring-primary/20 bg-primary/5"
                          : "border-border"
                      )}
                    >
                      <div
                        className={cn(
                          "mt-1 h-2.5 w-2.5 rounded-full shrink-0",
                          statusColor(vm.status)
                        )}
                      />
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="font-medium text-sm truncate">
                            {vm.name || "Unbenannt"}
                          </span>
                          <Badge
                            variant="outline"
                            className="text-[10px] shrink-0"
                          >
                            {vm.vmid}
                          </Badge>
                        </div>
                        <div className="flex items-center gap-3 mt-1 text-xs text-muted-foreground">
                          <span className="flex items-center gap-1">
                            <Cpu className="h-3 w-3" />
                            {vm.cpu_cores} vCPU
                          </span>
                          <span className="flex items-center gap-1">
                            <MemoryStick className="h-3 w-3" />
                            {formatBytes(vm.memory_total)}
                          </span>
                          <span className="flex items-center gap-1">
                            <HardDrive className="h-3 w-3" />
                            {formatBytes(vm.disk_total)}
                          </span>
                        </div>
                      </div>
                    </button>
                  );
                })}
              </div>
            )}
          </div>
        )}
      </div>

      <div>
        {selectedVM ? (
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm">VM-Details</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Name</span>
                <span className="font-medium">
                  {selectedVM.name || "Unbenannt"}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">VMID</span>
                <span className="font-medium">{selectedVM.vmid}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Typ</span>
                <Badge variant="outline">{selectedVM.type}</Badge>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Status</span>
                <Badge
                  variant={
                    selectedVM.status === "running" ? "default" : "secondary"
                  }
                >
                  {selectedVM.status}
                </Badge>
              </div>
              <div className="border-t pt-3 space-y-2">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">CPU</span>
                  <span>{selectedVM.cpu_cores} Kerne</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">RAM</span>
                  <span>{formatBytes(selectedVM.memory_total)}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Disk</span>
                  <span>{formatBytes(selectedVM.disk_total)}</span>
                </div>
              </div>
            </CardContent>
          </Card>
        ) : (
          <div className="flex items-center justify-center h-full border border-dashed rounded-lg p-8 text-center">
            <p className="text-sm text-muted-foreground">
              VM auswaehlen, um Details anzuzeigen
            </p>
          </div>
        )}
      </div>
    </div>
  );

  const renderStep2 = () => (
    <div className="space-y-6">
      <div className="space-y-2">
        <Label className="text-sm font-medium">Ziel-Node</Label>
        <Select value={targetNodeId} onValueChange={setTargetNodeId}>
          <SelectTrigger className="max-w-md">
            <SelectValue placeholder="Ziel-Node auswaehlen..." />
          </SelectTrigger>
          <SelectContent>
            {targetNodes.map((node) => (
              <SelectItem key={node.id} value={node.id}>
                <div className="flex items-center gap-2">
                  <Server className="h-3.5 w-3.5" />
                  {node.name}
                </div>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {targetNodeId && (
        <div className="space-y-3">
          <Label className="text-sm font-medium">Ziel-Storage</Label>

          {loadingStorages ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
              <span className="ml-2 text-sm text-muted-foreground">
                Storages werden geladen...
              </span>
            </div>
          ) : storageError ? (
            <div className="flex items-start gap-3 rounded-lg border border-destructive/50 bg-destructive/10 p-4">
              <AlertTriangle className="h-5 w-5 text-destructive shrink-0 mt-0.5" />
              <div>
                <p className="text-sm text-destructive">{storageError}</p>
                <p className="text-xs text-muted-foreground mt-2">
                  Tipp: Pruefe die API-Token-Berechtigungen des Quell-Nodes.
                </p>
              </div>
            </div>
          ) : storages.length === 0 ? (
            <div className="text-center py-6 text-muted-foreground text-sm">
              Keine Storages gefunden. Waehle zuerst einen Quell-Node in Schritt 1.
            </div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
              {storages.map((s) => {
                const isSelected = s.storage === targetStorage;
                const usedPercent =
                  s.total > 0 ? Math.round((s.used / s.total) * 100) : 0;
                return (
                  <button
                    key={s.storage}
                    type="button"
                    onClick={() => setTargetStorage(s.storage)}
                    className={cn(
                      "flex flex-col gap-2 rounded-lg border p-4 text-left transition-all hover:bg-accent/50",
                      isSelected
                        ? "border-primary ring-2 ring-primary/20 bg-primary/5"
                        : "border-border"
                    )}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <HardDrive className="h-4 w-4 text-muted-foreground" />
                        <span className="font-medium text-sm">{s.storage}</span>
                      </div>
                      <div className="flex items-center gap-1">
                        {s.shared && (
                          <Badge variant="secondary" className="text-[10px]">
                            shared
                          </Badge>
                        )}
                        <Badge variant="outline" className="text-[10px]">
                          {s.type}
                        </Badge>
                      </div>
                    </div>
                    <div className="space-y-1">
                      <div className="flex justify-between text-xs text-muted-foreground">
                        <span>Belegt: {usedPercent}%</span>
                        <span>Frei: {formatBytes(s.available)}</span>
                      </div>
                      <div className="h-1.5 rounded-full bg-muted overflow-hidden">
                        <div
                          className={cn(
                            "h-full rounded-full transition-all",
                            usedPercent > 90
                              ? "bg-red-500"
                              : usedPercent > 70
                              ? "bg-amber-500"
                              : "bg-green-500"
                          )}
                          style={{ width: `${usedPercent}%` }}
                        />
                      </div>
                      <p className="text-xs text-muted-foreground">
                        {formatBytes(s.used)} / {formatBytes(s.total)}
                      </p>
                    </div>
                  </button>
                );
              })}
            </div>
          )}
        </div>
      )}
    </div>
  );

  const renderStep3 = () => (
    <div className="space-y-6">
      <div className="space-y-3">
        <Label className="text-sm font-medium">Migrations-Modus</Label>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          {MODE_CONFIG.map((m) => {
            const Icon = m.icon;
            const isSelected = mode === m.value;
            return (
              <button
                key={m.value}
                type="button"
                onClick={() => setMode(m.value)}
                className={cn(
                  "flex flex-col items-center gap-3 rounded-lg border p-5 text-center transition-all hover:bg-accent/50",
                  isSelected
                    ? `${m.border} ring-2 ring-primary/20 ${m.bg}`
                    : "border-border"
                )}
              >
                <div
                  className={cn(
                    "flex h-12 w-12 items-center justify-center rounded-full",
                    m.bg
                  )}
                >
                  <Icon className={cn("h-6 w-6", m.accent)} />
                </div>
                <div>
                  <p className="font-medium text-sm">{m.label}</p>
                  <p className="text-xs text-muted-foreground mt-1">
                    {m.description}
                  </p>
                </div>
              </button>
            );
          })}
        </div>
      </div>

      {mode === "stop" && (
        <div className="flex items-start gap-3 rounded-lg border border-amber-500/50 bg-amber-500/10 p-4">
          <AlertTriangle className="h-5 w-5 text-amber-600 dark:text-amber-400 shrink-0 mt-0.5" />
          <div>
            <p className="text-sm font-medium text-amber-700 dark:text-amber-300">
              Achtung: VM wird heruntergefahren
            </p>
            <p className="text-xs text-amber-600 dark:text-amber-400 mt-1">
              Die VM ist waehrend der gesamten Migration nicht erreichbar.
            </p>
          </div>
        </div>
      )}

      <div className="space-y-2 max-w-xs">
        <Label className="text-sm font-medium">Neue VMID (optional)</Label>
        <Input
          type="number"
          placeholder="Gleiche VMID beibehalten"
          value={newVmid}
          onChange={(e) => setNewVmid(e.target.value)}
        />
      </div>

      <div className="space-y-3">
        <Label className="text-sm font-medium">Bereinigung</Label>
        <div className="space-y-2">
          <label className="flex items-center gap-3 cursor-pointer">
            <input
              type="checkbox"
              checked={cleanupSource}
              onChange={(e) => setCleanupSource(e.target.checked)}
              className="h-4 w-4 rounded border-border"
            />
            <span className="text-sm">Vzdump auf Source loeschen</span>
          </label>
          <label className="flex items-center gap-3 cursor-pointer">
            <input
              type="checkbox"
              checked={cleanupTarget}
              onChange={(e) => setCleanupTarget(e.target.checked)}
              className="h-4 w-4 rounded border-border"
            />
            <span className="text-sm">Vzdump auf Target loeschen</span>
          </label>
        </div>
      </div>
    </div>
  );

  const renderStep4 = () => (
    <div className="space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-[1fr,auto,1fr] gap-4 items-center">
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Quelle</CardDescription>
            <CardTitle className="text-base flex items-center gap-2">
              <Server className="h-4 w-4" />
              {sourceNode?.name || "-"}
            </CardTitle>
          </CardHeader>
          <CardContent className="text-sm space-y-1">
            <div className="flex justify-between">
              <span className="text-muted-foreground">VM</span>
              <span className="font-medium">
                {selectedVM?.name || "-"} ({selectedVmid})
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Typ</span>
              <Badge variant="outline">{selectedVM?.type}</Badge>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Status</span>
              <Badge
                variant={
                  selectedVM?.status === "running" ? "default" : "secondary"
                }
              >
                {selectedVM?.status}
              </Badge>
            </div>
          </CardContent>
        </Card>

        <div className="flex flex-col items-center gap-1">
          <div className="h-10 w-10 rounded-full bg-zinc-100 dark:bg-zinc-800 flex items-center justify-center">
            <ChevronRight className="h-5 w-5 text-zinc-600 dark:text-zinc-400" />
          </div>
          <span className="text-xs text-muted-foreground">Migration</span>
        </div>

        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Ziel</CardDescription>
            <CardTitle className="text-base flex items-center gap-2">
              <Server className="h-4 w-4" />
              {targetNode?.name || "-"}
            </CardTitle>
          </CardHeader>
          <CardContent className="text-sm space-y-1">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Storage</span>
              <span className="font-medium">{targetStorage || "-"}</span>
            </div>
            {selectedStorageInfo && (
              <>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Typ</span>
                  <Badge variant="outline">{selectedStorageInfo.type}</Badge>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Frei</span>
                  <span>{formatBytes(selectedStorageInfo.available)}</span>
                </div>
              </>
            )}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm">Migrations-Optionen</CardTitle>
        </CardHeader>
        <CardContent className="text-sm space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-muted-foreground">Modus</span>
            <div className="flex items-center gap-2">
              {selectedMode && (
                <selectedMode.icon
                  className={cn("h-4 w-4", selectedMode.accent)}
                />
              )}
              <span className="font-medium">{selectedMode?.label}</span>
            </div>
          </div>
          {newVmid && (
            <div className="flex justify-between">
              <span className="text-muted-foreground">Neue VMID</span>
              <span className="font-medium">{newVmid}</span>
            </div>
          )}
          <div className="flex justify-between">
            <span className="text-muted-foreground">Source bereinigen</span>
            <Badge variant={cleanupSource ? "default" : "secondary"}>
              {cleanupSource ? "Ja" : "Nein"}
            </Badge>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Target bereinigen</span>
            <Badge variant={cleanupTarget ? "default" : "secondary"}>
              {cleanupTarget ? "Ja" : "Nein"}
            </Badge>
          </div>
        </CardContent>
      </Card>

      {submitError && (
        <div className="flex items-start gap-3 rounded-lg border border-destructive/50 bg-destructive/10 p-4">
          <AlertTriangle className="h-5 w-5 text-destructive shrink-0 mt-0.5" />
          <p className="text-sm text-destructive">{submitError}</p>
        </div>
      )}
    </div>
  );

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">VM-Migrationen</h1>
        <p className="text-muted-foreground text-sm">
          VMs zwischen Nodes migrieren und Fortschritt in Echtzeit verfolgen.
        </p>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Server className="h-5 w-5" />
            <CardTitle className="text-base">
              Neue Migration starten
            </CardTitle>
          </div>
          <CardDescription>
            Schritt-fuer-Schritt Assistent zur VM-Migration zwischen Nodes.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {renderStepIndicator()}
          <div className="min-h-[300px]">
            {step === 1 && renderStep1()}
            {step === 2 && renderStep2()}
            {step === 3 && renderStep3()}
            {step === 4 && renderStep4()}
          </div>
          <div className="flex items-center justify-between mt-8 pt-4 border-t">
            <Button
              variant="outline"
              onClick={() => setStep((s) => s - 1)}
              disabled={step === 1}
            >
              <ArrowLeft className="mr-2 h-4 w-4" />
              Zurueck
            </Button>
            {step < 4 ? (
              <Button
                onClick={() => setStep((s) => s + 1)}
                disabled={!canProceed(step)}
              >
                Weiter
                <ArrowRight className="ml-2 h-4 w-4" />
              </Button>
            ) : (
              <Button
                onClick={handleStart}
                disabled={submitting}
                className="min-w-[180px]"
              >
                {submitting ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <Play className="mr-2 h-4 w-4" />
                )}
                Migration starten
              </Button>
            )}
          </div>
        </CardContent>
      </Card>

      <div>
        <h2 className="text-lg font-semibold mb-3">Migrations-Historie</h2>
        <MigrationHistory />
      </div>
    </div>
  );
}
