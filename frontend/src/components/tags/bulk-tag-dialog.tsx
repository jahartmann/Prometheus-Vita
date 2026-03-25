"use client";

import { useEffect, useState, useMemo } from "react";
import {
  Tags,
  Plus,
  Search,
  Loader2,
  Monitor,
  Container,
  Server,
  Check,
  X,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Progress } from "@/components/ui/progress";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { tagApi, nodeApi, toArray } from "@/lib/api";
import { toast } from "sonner";
import type { Tag, Node, VM } from "@/types/api";

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

interface VMWithNode {
  nodeId: string;
  nodeName: string;
  vmid: number;
  name: string;
  type: string;
  status: string;
}

interface BulkTagDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  preselectedTagId?: string;
  preselectedVMs?: Array<{ nodeId: string; vmid: number; vmType: string }>;
  onComplete?: () => void;
}

export function BulkTagDialog({
  open,
  onOpenChange,
  preselectedTagId,
  preselectedVMs,
  onComplete,
}: BulkTagDialogProps) {
  const [tags, setTags] = useState<Tag[]>([]);
  const [selectedTagId, setSelectedTagId] = useState<string>("");
  const [nodes, setNodes] = useState<Node[]>([]);
  const [allVMs, setAllVMs] = useState<VMWithNode[]>([]);
  const [selectedVMs, setSelectedVMs] = useState<Set<string>>(new Set());
  const [vmSearch, setVmSearch] = useState("");
  const [loading, setLoading] = useState(false);
  const [operating, setOperating] = useState(false);
  const [progress, setProgress] = useState(0);
  const [mode, setMode] = useState<"assign" | "remove">("assign");

  // Quick-create
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState("");
  const [newColor, setNewColor] = useState("#3b82f6");
  const [creating, setCreating] = useState(false);

  const vmKey = (nodeId: string, vmid: number) => `${nodeId}:${vmid}`;

  const fetchData = async () => {
    setLoading(true);
    try {
      const [tagsRes, nodesRes] = await Promise.all([
        tagApi.list(),
        nodeApi.list(),
      ]);
      const tagList = toArray<Tag>(tagsRes.data);
      const nodeList = toArray<Node>(nodesRes.data).filter(
        (n) => n.type === "pve"
      );
      setTags(tagList);
      setNodes(nodeList);

      // Fetch VMs from all nodes in parallel
      const vmPromises = nodeList.map(async (node) => {
        try {
          const res = await nodeApi.getVMs(node.id);
          const vms = toArray<VM>(res.data);
          return vms.map((vm) => ({
            nodeId: node.id,
            nodeName: node.name,
            vmid: vm.vmid,
            name: vm.name,
            type: vm.type,
            status: vm.status,
          }));
        } catch {
          return [];
        }
      });
      const results = await Promise.all(vmPromises);
      setAllVMs(results.flat());

      // Preselect
      if (preselectedTagId) {
        setSelectedTagId(preselectedTagId);
      }
      if (preselectedVMs && preselectedVMs.length > 0) {
        setSelectedVMs(
          new Set(preselectedVMs.map((v) => vmKey(v.nodeId, v.vmid)))
        );
      }
    } catch {
      toast.error("Fehler beim Laden der Daten");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (open) {
      fetchData();
      setProgress(0);
      setOperating(false);
      setMode("assign");
      setVmSearch("");
      if (!preselectedTagId) setSelectedTagId("");
      if (!preselectedVMs) setSelectedVMs(new Set());
    }
  }, [open]);

  // Group VMs by node
  const vmsByNode = useMemo(() => {
    const map = new Map<string, VMWithNode[]>();
    const lower = vmSearch.toLowerCase();
    for (const vm of allVMs) {
      if (
        lower &&
        !vm.name.toLowerCase().includes(lower) &&
        !vm.vmid.toString().includes(lower)
      ) {
        continue;
      }
      const list = map.get(vm.nodeId) || [];
      list.push(vm);
      map.set(vm.nodeId, list);
    }
    return map;
  }, [allVMs, vmSearch]);

  const toggleVM = (nodeId: string, vmid: number) => {
    const key = vmKey(nodeId, vmid);
    setSelectedVMs((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  };

  const toggleNodeAll = (nodeId: string) => {
    const nodeVMs = vmsByNode.get(nodeId) || [];
    const allSelected = nodeVMs.every((vm) =>
      selectedVMs.has(vmKey(vm.nodeId, vm.vmid))
    );
    setSelectedVMs((prev) => {
      const next = new Set(prev);
      for (const vm of nodeVMs) {
        const key = vmKey(vm.nodeId, vm.vmid);
        if (allSelected) {
          next.delete(key);
        } else {
          next.add(key);
        }
      }
      return next;
    });
  };

  const handleQuickCreate = async () => {
    if (!newName.trim()) return;
    setCreating(true);
    try {
      const res = await tagApi.create({ name: newName.trim(), color: newColor });
      const created = res.data as Tag;
      if (created?.id) {
        setTags((prev) => [...prev, created]);
        setSelectedTagId(created.id);
      }
      setNewName("");
      setShowCreate(false);
      toast.success("Tag erstellt");
    } catch {
      toast.error("Fehler beim Erstellen des Tags");
    } finally {
      setCreating(false);
    }
  };

  const handleExecute = async () => {
    if (!selectedTagId || selectedVMs.size === 0) return;
    setOperating(true);
    setProgress(0);

    const targets = Array.from(selectedVMs).map((key) => {
      const [nodeId, vmidStr] = key.split(":");
      const vmid = parseInt(vmidStr, 10);
      const vm = allVMs.find(
        (v) => v.nodeId === nodeId && v.vmid === vmid
      );
      return {
        node_id: nodeId,
        vmid,
        vm_type: vm?.type || "qemu",
      };
    });

    try {
      if (mode === "assign") {
        await tagApi.bulkAssign(selectedTagId, targets);
      } else {
        await tagApi.bulkRemove(
          selectedTagId,
          targets.map((t) => ({ node_id: t.node_id, vmid: t.vmid }))
        );
      }
      setProgress(100);
      toast.success(
        mode === "assign"
          ? `Tag an ${targets.length} VMs zugewiesen`
          : `Tag von ${targets.length} VMs entfernt`
      );
      onComplete?.();
      setTimeout(() => onOpenChange(false), 500);
    } catch {
      toast.error("Bulk-Operation fehlgeschlagen");
    } finally {
      setOperating(false);
    }
  };

  const selectedTag = tags.find((t) => t.id === selectedTagId);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl max-h-[85vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Tags className="h-5 w-5" />
            Bulk-Zuweisung
          </DialogTitle>
          <DialogDescription>
            Tag an mehrere VMs/Container gleichzeitig zuweisen oder entfernen
          </DialogDescription>
        </DialogHeader>

        {loading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        ) : (
          <div className="flex gap-4 flex-1 min-h-0 overflow-hidden">
            {/* Left panel: Tag selection */}
            <div className="w-56 shrink-0 space-y-3 overflow-y-auto">
              <Label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                Tag auswählen
              </Label>
              <div className="space-y-1">
                {tags.map((tag) => (
                  <button
                    key={tag.id}
                    onClick={() => setSelectedTagId(tag.id)}
                    className={`w-full flex items-center gap-2 rounded-md px-2 py-1.5 text-sm text-left transition-colors ${
                      selectedTagId === tag.id
                        ? "bg-accent font-medium"
                        : "hover:bg-accent/50"
                    }`}
                  >
                    <div
                      className="h-3 w-3 rounded-full shrink-0"
                      style={{ backgroundColor: tag.color }}
                    />
                    <span className="truncate">{tag.name}</span>
                    {selectedTagId === tag.id && (
                      <Check className="h-3 w-3 ml-auto shrink-0 text-primary" />
                    )}
                  </button>
                ))}
              </div>

              {tags.length === 0 && (
                <p className="text-xs text-muted-foreground text-center py-2">
                  Keine Tags vorhanden
                </p>
              )}

              {!showCreate ? (
                <Button
                  variant="outline"
                  size="sm"
                  className="w-full"
                  onClick={() => setShowCreate(true)}
                >
                  <Plus className="mr-1 h-3 w-3" />
                  Neuer Tag
                </Button>
              ) : (
                <div className="space-y-2 rounded-lg border p-2">
                  <input
                    className="w-full rounded-md border bg-background px-2 py-1 text-xs"
                    value={newName}
                    onChange={(e) => setNewName(e.target.value)}
                    placeholder="Tag-Name"
                    onKeyDown={(e) =>
                      e.key === "Enter" && handleQuickCreate()
                    }
                    autoFocus
                  />
                  <div className="flex flex-wrap gap-1">
                    {colorPresets.map((c) => (
                      <button
                        key={c}
                        className={`h-4 w-4 rounded-full border-2 ${
                          newColor === c
                            ? "border-foreground"
                            : "border-transparent"
                        }`}
                        style={{ backgroundColor: c }}
                        onClick={() => setNewColor(c)}
                      />
                    ))}
                  </div>
                  <div className="flex gap-1">
                    <Button
                      size="sm"
                      className="h-7 text-xs"
                      onClick={handleQuickCreate}
                      disabled={!newName.trim() || creating}
                    >
                      {creating && (
                        <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                      )}
                      Erstellen
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      className="h-7 text-xs"
                      onClick={() => setShowCreate(false)}
                    >
                      <X className="h-3 w-3" />
                    </Button>
                  </div>
                </div>
              )}
            </div>

            {/* Right panel: VM selection */}
            <div className="flex-1 min-h-0 flex flex-col space-y-2">
              <Label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                VMs auswählen
              </Label>
              <div className="relative">
                <Search className="absolute left-2.5 top-2.5 h-3.5 w-3.5 text-muted-foreground" />
                <Input
                  placeholder="VMs suchen..."
                  value={vmSearch}
                  onChange={(e) => setVmSearch(e.target.value)}
                  className="pl-8 h-8 text-sm"
                />
              </div>

              <div className="flex-1 overflow-y-auto space-y-3 rounded-lg border p-2 min-h-[200px] max-h-[400px]">
                {Array.from(vmsByNode.entries()).map(([nodeId, nodeVMs]) => {
                  const node = nodes.find((n) => n.id === nodeId);
                  const allNodeSelected = nodeVMs.every((vm) =>
                    selectedVMs.has(vmKey(vm.nodeId, vm.vmid))
                  );
                  const someNodeSelected =
                    !allNodeSelected &&
                    nodeVMs.some((vm) =>
                      selectedVMs.has(vmKey(vm.nodeId, vm.vmid))
                    );

                  return (
                    <div key={nodeId}>
                      <label className="flex items-center gap-2 px-1 py-1 text-sm font-medium cursor-pointer hover:bg-accent/50 rounded">
                        <Checkbox
                          checked={allNodeSelected || (someNodeSelected ? "indeterminate" : false)}
                          onCheckedChange={() => toggleNodeAll(nodeId)}
                        />
                        <Server className="h-3.5 w-3.5 text-muted-foreground" />
                        <span>{node?.name || nodeId.slice(0, 8)}</span>
                        <Badge variant="outline" className="ml-auto text-xs">
                          {nodeVMs.length} VMs
                        </Badge>
                      </label>
                      <div className="ml-6 space-y-0.5 mt-0.5">
                        {nodeVMs.map((vm) => {
                          const key = vmKey(vm.nodeId, vm.vmid);
                          return (
                            <label
                              key={key}
                              className="flex items-center gap-2 px-1 py-0.5 text-sm cursor-pointer hover:bg-accent/50 rounded"
                            >
                              <Checkbox
                                checked={selectedVMs.has(key)}
                                onCheckedChange={() =>
                                  toggleVM(vm.nodeId, vm.vmid)
                                }
                              />
                              {vm.type === "qemu" ? (
                                <Monitor className="h-3 w-3 text-muted-foreground" />
                              ) : (
                                <Container className="h-3 w-3 text-muted-foreground" />
                              )}
                              <span className="font-mono text-xs text-muted-foreground w-10">
                                {vm.vmid}
                              </span>
                              <span className="truncate">{vm.name}</span>
                              <span
                                className={`ml-auto h-2 w-2 rounded-full shrink-0 ${
                                  vm.status === "running"
                                    ? "bg-green-500"
                                    : "bg-gray-400"
                                }`}
                              />
                            </label>
                          );
                        })}
                      </div>
                    </div>
                  );
                })}
                {vmsByNode.size === 0 && (
                  <p className="text-sm text-muted-foreground text-center py-8">
                    {vmSearch
                      ? "Keine VMs gefunden"
                      : "Keine VMs auf den Nodes verfügbar"}
                  </p>
                )}
              </div>
            </div>
          </div>
        )}

        {/* Footer */}
        {operating && (
          <Progress value={progress} className="h-1.5" />
        )}

        <DialogFooter className="flex items-center justify-between sm:justify-between">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            {selectedTag && (
              <Badge
                style={{ backgroundColor: selectedTag.color, color: "white" }}
              >
                {selectedTag.name}
              </Badge>
            )}
            <span>{selectedVMs.size} VMs ausgewählt</span>
          </div>
          <div className="flex gap-2">
            <Button
              onClick={() => {
                setMode("remove");
                handleExecute();
              }}
              variant="outline"
              disabled={
                !selectedTagId || selectedVMs.size === 0 || operating
              }
            >
              {operating && mode === "remove" && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Tag entfernen
            </Button>
            <Button
              onClick={() => {
                setMode("assign");
                handleExecute();
              }}
              disabled={
                !selectedTagId || selectedVMs.size === 0 || operating
              }
            >
              {operating && mode === "assign" && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Tag zuweisen
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
