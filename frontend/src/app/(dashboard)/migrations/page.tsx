"use client";

import { useEffect, useState, useCallback } from "react";
import { useNodeStore } from "@/stores/node-store";
import { useMigrationStore } from "@/stores/migration-store";
import { useWebSocket } from "@/hooks/use-websocket";
import { MigrationHistory } from "@/components/migration/migration-history";
import { MigrationProgress } from "@/components/migration/migration-progress";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
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
import { ArrowRight, Loader2, Play, Server } from "lucide-react";
import api from "@/lib/api";
import type { VM, VMMigration, MigrationMode } from "@/types/api";
import { toArray } from "@/lib/api";
import { formatBytes } from "@/lib/utils";

const MODES: { value: MigrationMode; label: string; description: string }[] = [
  { value: "snapshot", label: "Snapshot", description: "Live-Backup ohne Downtime" },
  { value: "suspend", label: "Suspend", description: "Kurze Pause, guter Kompromiss" },
  { value: "stop", label: "Stop", description: "VM herunterfahren, konsistentestes Backup" },
];

interface StorageOption {
  storage: string;
  type: string;
  total: number;
  used: number;
  available: number;
}

export default function MigrationsPage() {
  const { nodes, fetchNodes, fetchNodeVMs, nodeVMs } = useNodeStore();
  const {
    migrations,
    activeMigrations,
    fetchMigrations,
    startMigration,
    updateMigrationProgress,
  } = useMigrationStore();

  // Form state
  const [sourceNodeId, setSourceNodeId] = useState("");
  const [targetNodeId, setTargetNodeId] = useState("");
  const [selectedVmid, setSelectedVmid] = useState("");
  const [targetStorage, setTargetStorage] = useState("");
  const [mode, setMode] = useState<MigrationMode>("snapshot");
  const [newVmid, setNewVmid] = useState("");
  const [cleanupSource, setCleanupSource] = useState(true);
  const [cleanupTarget, setCleanupTarget] = useState(true);

  // UI state
  const [storages, setStorages] = useState<StorageOption[]>([]);
  const [loadingStorages, setLoadingStorages] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState("");

  useEffect(() => {
    if (nodes.length === 0) fetchNodes();
    fetchMigrations();
  }, [nodes.length, fetchNodes, fetchMigrations]);

  // Load VMs when source node changes
  useEffect(() => {
    if (sourceNodeId) {
      fetchNodeVMs(sourceNodeId);
      // Reset dependent fields
      setSelectedVmid("");
    }
  }, [sourceNodeId, fetchNodeVMs]);

  // Load storages when target node changes
  useEffect(() => {
    if (!targetNodeId) {
      setStorages([]);
      return;
    }
    setTargetStorage("");
    setLoadingStorages(true);
    api
      .get(`/nodes/${targetNodeId}/storage`)
      .then((res) => {
        const data = Array.isArray(res.data) ? res.data : (res.data?.data || []);
        setStorages(data.filter((s: StorageOption) => s.available > 0));
      })
      .catch(() => setStorages([]))
      .finally(() => setLoadingStorages(false));
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
  const sourceVMs = sourceNodeId ? (nodeVMs[sourceNodeId] || []) : [];
  const targetNodes = onlineNodes.filter((n) => n.id !== sourceNodeId);

  const canSubmit =
    sourceNodeId && targetNodeId && selectedVmid && targetStorage && !submitting;

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
      // Reset form
      setSourceNodeId("");
      setTargetNodeId("");
      setSelectedVmid("");
      setTargetStorage("");
      setMode("snapshot");
      setNewVmid("");
    } catch (err: unknown) {
      setSubmitError(
        err instanceof Error ? err.message : "Migration konnte nicht gestartet werden"
      );
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">VM-Migrationen</h1>
        <p className="text-muted-foreground text-sm">
          VMs zwischen Nodes migrieren und Fortschritt in Echtzeit verfolgen.
        </p>
      </div>

      {/* Active migrations with live progress */}
      {activeMigrations.length > 0 && (
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm flex items-center gap-2">
              <Loader2 className="h-4 w-4 animate-spin" />
              Aktive Migrationen ({activeMigrations.length})
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {activeMigrations.map((m) => (
              <MigrationProgress key={m.id} migration={m} />
            ))}
          </CardContent>
        </Card>
      )}

      {/* New migration form */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Server className="h-5 w-5" />
            <CardTitle className="text-base">Neue Migration starten</CardTitle>
          </div>
          <CardDescription>
            VM von einem Quell-Node auf einen Ziel-Node migrieren.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Node selection row */}
          <div className="grid grid-cols-[1fr,auto,1fr] gap-4 items-end">
            <div className="space-y-2">
              <Label>Quell-Node</Label>
              <Select value={sourceNodeId} onValueChange={setSourceNodeId}>
                <SelectTrigger>
                  <SelectValue placeholder="Node waehlen..." />
                </SelectTrigger>
                <SelectContent>
                  {onlineNodes.map((node) => (
                    <SelectItem key={node.id} value={node.id}>
                      {node.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <ArrowRight className="h-5 w-5 text-muted-foreground mb-2" />

            <div className="space-y-2">
              <Label>Ziel-Node</Label>
              <Select
                value={targetNodeId}
                onValueChange={setTargetNodeId}
                disabled={!sourceNodeId}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Node waehlen..." />
                </SelectTrigger>
                <SelectContent>
                  {targetNodes.map((node) => (
                    <SelectItem key={node.id} value={node.id}>
                      {node.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* VM selection */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>VM</Label>
              <Select
                value={selectedVmid}
                onValueChange={setSelectedVmid}
                disabled={!sourceNodeId || sourceVMs.length === 0}
              >
                <SelectTrigger>
                  <SelectValue
                    placeholder={
                      !sourceNodeId
                        ? "Zuerst Quell-Node waehlen"
                        : sourceVMs.length === 0
                        ? "Keine VMs verfuegbar"
                        : "VM waehlen..."
                    }
                  />
                </SelectTrigger>
                <SelectContent>
                  {sourceVMs.map((vm) => (
                    <SelectItem key={vm.vmid} value={String(vm.vmid)}>
                      {vm.vmid} - {vm.name || "Unbenannt"}
                      {vm.status && ` (${vm.status})`}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>Ziel-Storage</Label>
              <Select
                value={targetStorage}
                onValueChange={setTargetStorage}
                disabled={!targetNodeId || loadingStorages}
              >
                <SelectTrigger>
                  <SelectValue
                    placeholder={
                      loadingStorages
                        ? "Laden..."
                        : !targetNodeId
                        ? "Zuerst Ziel-Node waehlen"
                        : "Storage waehlen..."
                    }
                  />
                </SelectTrigger>
                <SelectContent>
                  {storages.map((s) => (
                    <SelectItem key={s.storage} value={s.storage}>
                      {s.storage} ({s.type}) - {formatBytes(s.available)} frei
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Mode & options */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Migrations-Modus</Label>
              <Select value={mode} onValueChange={(v) => setMode(v as MigrationMode)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {MODES.map((m) => (
                    <SelectItem key={m.value} value={m.value}>
                      {m.label} - {m.description}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>Neue VMID (optional)</Label>
              <Input
                type="number"
                placeholder="Gleiche VMID beibehalten"
                value={newVmid}
                onChange={(e) => setNewVmid(e.target.value)}
              />
            </div>
          </div>

          <div className="flex items-center gap-6">
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

          {submitError && (
            <p className="text-sm text-destructive">{submitError}</p>
          )}

          {mode === "stop" && (
            <p className="text-xs text-amber-600">
              Die VM wird waehrend der Migration heruntergefahren und ist nicht erreichbar.
            </p>
          )}

          <div className="flex justify-end">
            <Button onClick={handleStart} disabled={!canSubmit}>
              {submitting ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Play className="mr-2 h-4 w-4" />
              )}
              Migration starten
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Migration history */}
      <div>
        <h2 className="text-lg font-semibold mb-3">Migrations-Historie</h2>
        <MigrationHistory />
      </div>
    </div>
  );
}
