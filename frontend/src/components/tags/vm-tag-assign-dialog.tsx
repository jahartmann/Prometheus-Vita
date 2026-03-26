"use client";

import { useEffect, useState } from "react";
import { Plus, X, Tag, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { tagApi, toArray } from "@/lib/api";
import { toast } from "sonner";
import type { Tag as TagType } from "@/types/api";

const colorPresets = [
  "#ef4444",
  "#f97316",
  "#eab308",
  "#22c55e",
  "#06b6d4",
  "#3b82f6",
  "#8b5cf6",
  "#ec4899",
];

interface VMTagAssignDialogProps {
  nodeId: string;
  vmid: number;
  vmType: string;
  vmName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onTagsChanged?: () => void;
}

export function VMTagAssignDialog({
  nodeId,
  vmid,
  vmType,
  vmName,
  open,
  onOpenChange,
  onTagsChanged,
}: VMTagAssignDialogProps) {
  const [allTags, setAllTags] = useState<TagType[]>([]);
  const [vmTags, setVmTags] = useState<TagType[]>([]);
  const [loading, setLoading] = useState(false);

  // Quick-create state
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState("");
  const [newColor, setNewColor] = useState("#3b82f6");
  const [creating, setCreating] = useState(false);

  const fetchData = async () => {
    setLoading(true);
    try {
      const [allRes, vmRes] = await Promise.all([
        tagApi.list(),
        tagApi.getVMTags(nodeId, vmid),
      ]);
      setAllTags(toArray<TagType>(allRes.data));
      setVmTags(toArray<TagType>(vmRes.data));
    } catch {
      // silent
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (open) {
      fetchData();
      setShowCreate(false);
      setNewName("");
    }
  }, [open, nodeId, vmid]);

  const assignedIds = new Set(vmTags.map((t) => t.id));
  const unassigned = allTags.filter((t) => !assignedIds.has(t.id));

  const handleAdd = async (tagId: string) => {
    try {
      await tagApi.addToVM(nodeId, vmid, tagId, vmType);
      await fetchData();
      onTagsChanged?.();
    } catch {
      toast.error("Tag konnte nicht zugewiesen werden");
    }
  };

  const handleRemove = async (tagId: string) => {
    try {
      await tagApi.removeFromVM(nodeId, vmid, tagId);
      await fetchData();
      onTagsChanged?.();
    } catch {
      toast.error("Tag konnte nicht entfernt werden");
    }
  };

  const handleQuickCreate = async () => {
    if (!newName.trim()) return;
    setCreating(true);
    try {
      const res = await tagApi.create({ name: newName.trim(), color: newColor });
      const created = res.data as TagType;
      // Assign directly
      if (created?.id) {
        await tagApi.addToVM(nodeId, vmid, created.id, vmType);
      }
      setNewName("");
      setShowCreate(false);
      await fetchData();
      onTagsChanged?.();
      toast.success("Tag erstellt und zugewiesen");
    } catch {
      toast.error("Fehler beim Erstellen des Tags");
    } finally {
      setCreating(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Tag className="h-4 w-4" />
            Tags verwalten
          </DialogTitle>
          <DialogDescription>
            {vmName} (ID: {vmid})
          </DialogDescription>
        </DialogHeader>

        {loading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        ) : (
          <div className="space-y-4">
            {/* Assigned tags */}
            {vmTags.length > 0 && (
              <div>
                <p className="text-sm font-medium mb-2">Zugewiesen</p>
                <div className="flex flex-wrap gap-2">
                  {vmTags.map((tag) => (
                    <Badge
                      key={tag.id}
                      className="pr-1"
                      style={{ backgroundColor: tag.color, color: "white" }}
                    >
                      {tag.name}
                      <button
                        className="ml-1.5 rounded-full hover:bg-white/20 p-0.5"
                        onClick={() => handleRemove(tag.id)}
                      >
                        <X className="h-3 w-3" />
                      </button>
                    </Badge>
                  ))}
                </div>
              </div>
            )}

            {/* Available tags */}
            {unassigned.length > 0 && (
              <div>
                <p className="text-sm font-medium mb-2">Verfügbar</p>
                <div className="flex flex-wrap gap-2">
                  {unassigned.map((tag) => (
                    <Badge
                      key={tag.id}
                      variant="outline"
                      className="cursor-pointer hover:bg-accent"
                      onClick={() => handleAdd(tag.id)}
                    >
                      <Plus className="mr-1 h-3 w-3" />
                      {tag.name}
                    </Badge>
                  ))}
                </div>
              </div>
            )}

            {allTags.length === 0 && vmTags.length === 0 && (
              <p className="text-sm text-muted-foreground text-center py-4">
                Keine Tags vorhanden.
              </p>
            )}

            {/* Quick-create */}
            {!showCreate ? (
              <Button
                variant="outline"
                size="sm"
                className="w-full"
                onClick={() => setShowCreate(true)}
              >
                <Plus className="mr-2 h-3 w-3" />
                Neuen Tag erstellen
              </Button>
            ) : (
              <div className="space-y-2 rounded-lg border p-3">
                <div className="space-y-1">
                  <Label className="text-xs">Name</Label>
                  <input
                    className="w-full rounded-md border bg-background px-3 py-1.5 text-sm"
                    value={newName}
                    onChange={(e) => setNewName(e.target.value)}
                    placeholder="Tag-Name"
                    onKeyDown={(e) => e.key === "Enter" && handleQuickCreate()}
                    autoFocus
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs">Farbe</Label>
                  <div className="flex items-center gap-1.5">
                    {colorPresets.map((c) => (
                      <button
                        key={c}
                        className={`h-5 w-5 rounded-full border-2 ${
                          newColor === c ? "border-foreground" : "border-transparent"
                        }`}
                        style={{ backgroundColor: c }}
                        onClick={() => setNewColor(c)}
                      />
                    ))}
                  </div>
                </div>
                <div className="flex gap-2 pt-1">
                  <Button
                    size="sm"
                    onClick={handleQuickCreate}
                    disabled={!newName.trim() || creating}
                  >
                    {creating && <Loader2 className="mr-1 h-3 w-3 animate-spin" />}
                    Erstellen & Zuweisen
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => setShowCreate(false)}
                  >
                    Abbrechen
                  </Button>
                </div>
              </div>
            )}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
