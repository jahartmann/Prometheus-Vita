"use client";

import { useState, useEffect, useCallback } from "react";
import { Camera, Loader2, RotateCcw, Trash2 } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Badge } from "@/components/ui/badge";
import { vmApi, toArray } from "@/lib/api";
import { toast } from "sonner";
import type { VMSnapshot } from "@/types/api";

interface SnapshotDialogProps {
  nodeId: string;
  vmid: number;
  vmType: string;
  vmName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function SnapshotDialog({
  nodeId,
  vmid,
  vmType,
  vmName,
  open,
  onOpenChange,
}: SnapshotDialogProps) {
  const [snapshots, setSnapshots] = useState<VMSnapshot[]>([]);
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  const [newName, setNewName] = useState("");
  const [newDescription, setNewDescription] = useState("");
  const [includeRam, setIncludeRam] = useState(false);

  const [confirmAction, setConfirmAction] = useState<{
    type: "rollback" | "delete";
    snapname: string;
  } | null>(null);

  const loadSnapshots = useCallback(async () => {
    setLoading(true);
    try {
      const res = await vmApi.listSnapshots(nodeId, vmid, vmType);
      const all = toArray<VMSnapshot>(res.data);
      setSnapshots(all.filter((s) => s.name !== "current"));
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [nodeId, vmid, vmType]);

  useEffect(() => {
    if (open) {
      loadSnapshots();
    }
  }, [open, loadSnapshots]);

  const handleCreate = async () => {
    if (!newName.trim()) return;
    setCreating(true);
    try {
      await vmApi.createSnapshot(nodeId, vmid, vmType, {
        name: newName.trim(),
        description: newDescription.trim() || undefined,
        vmstate: includeRam,
      });
      toast.success(`Snapshot "${newName.trim()}" wird erstellt`);
      setNewName("");
      setNewDescription("");
      setIncludeRam(false);
      await loadSnapshots();
    } catch {
      toast.error("Snapshot konnte nicht erstellt werden");
    } finally {
      setCreating(false);
    }
  };

  const handleRollback = async (snapname: string) => {
    setActionLoading(snapname);
    try {
      await vmApi.rollbackSnapshot(nodeId, vmid, vmType, snapname);
      toast.success(`Snapshot "${snapname}" wird wiederhergestellt`);
      await loadSnapshots();
    } catch {
      toast.error("Rollback fehlgeschlagen");
    } finally {
      setActionLoading(null);
      setConfirmAction(null);
    }
  };

  const handleDelete = async (snapname: string) => {
    setActionLoading(snapname);
    try {
      await vmApi.deleteSnapshot(nodeId, vmid, vmType, snapname);
      toast.success(`Snapshot "${snapname}" geloescht`);
      await loadSnapshots();
    } catch {
      toast.error("Snapshot konnte nicht geloescht werden");
    } finally {
      setActionLoading(null);
      setConfirmAction(null);
    }
  };

  const formatDate = (timestamp: number) => {
    if (!timestamp) return "--";
    return new Date(timestamp * 1000).toLocaleString("de-DE", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Camera className="h-5 w-5" />
              Snapshots: {vmName} (ID: {vmid})
            </DialogTitle>
            <DialogDescription>
              Snapshots verwalten und erstellen
            </DialogDescription>
          </DialogHeader>

          {loading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : snapshots.length === 0 ? (
            <p className="text-sm text-muted-foreground py-4">
              Keine Snapshots vorhanden.
            </p>
          ) : (
            <div className="space-y-3">
              {snapshots.map((snap) => (
                <div
                  key={snap.name}
                  className="rounded-lg border p-4 space-y-2"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <Camera className="h-4 w-4 text-muted-foreground" />
                      <span className="font-medium">{snap.name}</span>
                      {snap.vmstate === 1 && (
                        <Badge variant="outline" className="text-xs">
                          RAM
                        </Badge>
                      )}
                    </div>
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={actionLoading === snap.name}
                        onClick={() =>
                          setConfirmAction({ type: "rollback", snapname: snap.name })
                        }
                        title="Wiederherstellen"
                      >
                        {actionLoading === snap.name ? (
                          <Loader2 className="h-3.5 w-3.5 animate-spin" />
                        ) : (
                          <RotateCcw className="h-3.5 w-3.5" />
                        )}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={actionLoading === snap.name}
                        onClick={() =>
                          setConfirmAction({ type: "delete", snapname: snap.name })
                        }
                        title="Loeschen"
                        className="text-destructive hover:text-destructive"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  </div>
                  <div className="text-xs text-muted-foreground space-y-0.5">
                    <p>Erstellt: {formatDate(snap.snaptime)}</p>
                    {snap.description && <p>Beschreibung: {snap.description}</p>}
                    {snap.parent && <p>Parent: {snap.parent}</p>}
                  </div>
                </div>
              ))}
            </div>
          )}

          <div className="border-t pt-4 space-y-3">
            <h4 className="text-sm font-medium">Neuer Snapshot erstellen</h4>
            <div className="grid gap-3">
              <div className="grid gap-1.5">
                <Label htmlFor="snap-name">Name</Label>
                <Input
                  id="snap-name"
                  placeholder="z.B. before-update"
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="snap-desc">Beschreibung (optional)</Label>
                <Input
                  id="snap-desc"
                  placeholder="Kurze Beschreibung..."
                  value={newDescription}
                  onChange={(e) => setNewDescription(e.target.value)}
                />
              </div>
              {vmType === "qemu" && (
                <div className="flex items-center gap-2">
                  <Checkbox
                    id="snap-ram"
                    checked={includeRam}
                    onCheckedChange={(checked) => setIncludeRam(checked === true)}
                  />
                  <Label htmlFor="snap-ram" className="text-sm">
                    RAM-Zustand einschliessen
                  </Label>
                </div>
              )}
              <Button
                onClick={handleCreate}
                disabled={!newName.trim() || creating}
                className="w-full"
              >
                {creating && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Snapshot erstellen
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={!!confirmAction}
        onOpenChange={(open) => !open && setConfirmAction(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {confirmAction?.type === "rollback"
                ? "Snapshot wiederherstellen?"
                : "Snapshot loeschen?"}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {confirmAction?.type === "rollback"
                ? `Snapshot "${confirmAction.snapname}" wiederherstellen? Die VM wird auf diesen Zustand zurueckgesetzt.`
                : `Snapshot "${confirmAction?.snapname}" wirklich loeschen? Diese Aktion kann nicht rueckgaengig gemacht werden.`}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Abbrechen</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (!confirmAction) return;
                if (confirmAction.type === "rollback") {
                  handleRollback(confirmAction.snapname);
                } else {
                  handleDelete(confirmAction.snapname);
                }
              }}
              className={
                confirmAction?.type === "delete"
                  ? "bg-destructive text-destructive-foreground hover:bg-destructive/90"
                  : undefined
              }
            >
              {confirmAction?.type === "rollback" ? "Wiederherstellen" : "Loeschen"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
