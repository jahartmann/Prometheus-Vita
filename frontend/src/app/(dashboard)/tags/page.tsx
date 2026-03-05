"use client";

import { useEffect, useState, useCallback } from "react";
import {
  Plus,
  Trash2,
  RefreshCw,
  Loader2,
  Tag,
  Server,
  Monitor,
  ShieldCheck,
  ShieldAlert,
  ChevronDown,
  ChevronRight,
  Tags,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { BulkTagDialog } from "@/components/tags/bulk-tag-dialog";
import { tagApi, nodeApi, toArray } from "@/lib/api";
import { toast } from "sonner";
import type { Tag as TagType, Node, VMTag, TagSyncAllResult } from "@/types/api";

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

interface TagWithCounts extends TagType {
  nodeCount: number;
  nodeIds: string[];
  vmCount: number;
  vmTags: VMTag[];
}

// Tag policy definitions (UI-only, future-ready)
interface TagPolicy {
  id: string;
  name: string;
  description: string;
  requiredTag: string;
  scope: "all" | "production";
  compliance: number; // 0-100
}

const defaultPolicies: TagPolicy[] = [
  {
    id: "1",
    name: "Backup-Tag Pflicht",
    description: "Alle Produktions-VMs sollten den Tag 'backup' besitzen",
    requiredTag: "backup",
    scope: "production",
    compliance: 0,
  },
  {
    id: "2",
    name: "Umgebungs-Tag Pflicht",
    description: "Alle VMs sollten einen Umgebungs-Tag (production/staging/dev) besitzen",
    requiredTag: "production",
    scope: "all",
    compliance: 0,
  },
  {
    id: "3",
    name: "Owner-Tag Pflicht",
    description: "Jede VM sollte einen Owner-Tag fuer die Verantwortlichkeit besitzen",
    requiredTag: "owner",
    scope: "all",
    compliance: 0,
  },
];

export default function TagsPage() {
  const [tags, setTags] = useState<TagWithCounts[]>([]);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [loading, setLoading] = useState(true);
  const [syncAllLoading, setSyncAllLoading] = useState(false);

  // Create form
  const [name, setName] = useState("");
  const [color, setColor] = useState("#3b82f6");
  const [category, setCategory] = useState("");

  // Bulk assign
  const [bulkTagId, setBulkTagId] = useState("");
  const [selectedNodeIds, setSelectedNodeIds] = useState<string[]>([]);
  const [bulkAssigning, setBulkAssigning] = useState(false);

  // Policies section
  const [policiesOpen, setPoliciesOpen] = useState(false);

  // Bulk tag dialog
  const [bulkTagOpen, setBulkTagOpen] = useState(false);
  const [bulkPreselectedTag, setBulkPreselectedTag] = useState<string | undefined>();

  // Expanded VM lists
  const [expandedTags, setExpandedTags] = useState<Set<string>>(new Set());

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const [tagsRes, nodesRes] = await Promise.all([
        tagApi.list(),
        nodeApi.list(),
      ]);

      const tagList = toArray<TagType>(tagsRes.data);
      const nodeList = toArray<Node>(nodesRes.data);
      setNodes(nodeList);

      // Enrich tags with node counts and VM counts
      const enriched: TagWithCounts[] = await Promise.all(
        tagList.map(async (tag) => {
          try {
            const [nodeResults, vmTagsRes] = await Promise.all([
              Promise.all(
                nodeList.map(async (node) => {
                  const res = await tagApi.getNodeTags(node.id);
                  const nodeTags = toArray<TagType>(res.data);
                  return nodeTags.some((t) => t.id === tag.id) ? node.id : null;
                })
              ),
              tagApi.getVMsByTag(tag.id).catch(() => ({ data: [] })),
            ]);
            const nodeIds = nodeResults.filter((id): id is string => id !== null);
            const vmTagList = toArray<VMTag>(vmTagsRes.data);
            return {
              ...tag,
              nodeCount: nodeIds.length,
              nodeIds,
              vmCount: vmTagList.length,
              vmTags: vmTagList,
            };
          } catch {
            return { ...tag, nodeCount: 0, nodeIds: [], vmCount: 0, vmTags: [] };
          }
        })
      );

      setTags(enriched);
    } catch {
      toast.error("Fehler beim Laden der Tags");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleCreate = async () => {
    if (!name.trim()) return;
    try {
      await tagApi.create({
        name: name.trim(),
        color,
        category: category || undefined,
      });
      setName("");
      setCategory("");
      toast.success("Tag erstellt");
      fetchData();
    } catch {
      toast.error("Fehler beim Erstellen des Tags");
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await tagApi.delete(id);
      toast.success("Tag geloescht");
      fetchData();
    } catch {
      toast.error("Fehler beim Loeschen des Tags");
    }
  };

  const handleSyncAll = async () => {
    setSyncAllLoading(true);
    try {
      const res = await tagApi.syncAll();
      const data = res.data as TagSyncAllResult;
      const total = data?.total_imported ?? 0;
      toast.success(`${total} Tags von allen Nodes importiert`);
      fetchData();
    } catch {
      toast.error("Tag-Sync fehlgeschlagen");
    } finally {
      setSyncAllLoading(false);
    }
  };

  const handleBulkAssign = async () => {
    if (!bulkTagId || selectedNodeIds.length === 0) return;
    setBulkAssigning(true);
    try {
      let successCount = 0;
      for (const nodeId of selectedNodeIds) {
        try {
          await tagApi.addToNode(nodeId, bulkTagId);
          successCount++;
        } catch {
          // skip individual errors
        }
      }
      toast.success(
        `Tag an ${successCount}/${selectedNodeIds.length} Nodes zugewiesen`
      );
      setSelectedNodeIds([]);
      setBulkTagId("");
      fetchData();
    } catch {
      toast.error("Fehler bei der Bulk-Zuweisung");
    } finally {
      setBulkAssigning(false);
    }
  };

  const toggleNodeSelection = (nodeId: string) => {
    setSelectedNodeIds((prev) =>
      prev.includes(nodeId)
        ? prev.filter((id) => id !== nodeId)
        : [...prev, nodeId]
    );
  };

  const pveNodes = nodes.filter((n) => n.type === "pve");

  // Group tags by category
  const categories = Array.from(
    new Set(tags.map((t) => t.category || "Ohne Kategorie"))
  );

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Tag className="h-6 w-6" />
            Tag-Verwaltung
          </h1>
          <p className="text-sm text-muted-foreground mt-1">
            Cluster-weite Tag-Verwaltung und Synchronisation
          </p>
        </div>
        <Button
          onClick={handleSyncAll}
          disabled={syncAllLoading}
          variant="outline"
        >
          {syncAllLoading ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <RefreshCw className="mr-2 h-4 w-4" />
          )}
          Alle Nodes synchronisieren
        </Button>
      </div>

      {/* Create Tag */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Neuen Tag erstellen</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="grid gap-3 sm:grid-cols-3">
            <div className="space-y-1">
              <Label>Name</Label>
              <input
                className="w-full rounded-md border bg-background px-3 py-2 text-sm"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="z.B. Production"
              />
            </div>
            <div className="space-y-1">
              <Label>Kategorie (optional)</Label>
              <input
                className="w-full rounded-md border bg-background px-3 py-2 text-sm"
                value={category}
                onChange={(e) => setCategory(e.target.value)}
                placeholder="z.B. Environment"
              />
            </div>
            <div className="space-y-1">
              <Label>Farbe</Label>
              <div className="flex items-center gap-2">
                {colorPresets.map((c) => (
                  <button
                    key={c}
                    className={`h-6 w-6 rounded-full border-2 ${
                      color === c ? "border-foreground" : "border-transparent"
                    }`}
                    style={{ backgroundColor: c }}
                    onClick={() => setColor(c)}
                  />
                ))}
              </div>
            </div>
          </div>
          <Button onClick={handleCreate} disabled={!name.trim()}>
            <Plus className="mr-2 h-4 w-4" />
            Tag erstellen
          </Button>
        </CardContent>
      </Card>

      {/* Tag Overview */}
      {loading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : (
        <>
          {categories.map((cat) => {
            const catTags = tags.filter(
              (t) => (t.category || "Ohne Kategorie") === cat
            );
            return (
              <div key={cat} className="space-y-2">
                <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
                  {cat}
                </h3>
                <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
                  {catTags.map((tag) => (
                    <Card key={tag.id}>
                      <CardContent className="p-4">
                        <div className="flex items-center justify-between mb-3">
                          <div className="flex items-center gap-2">
                            <div
                              className="h-4 w-4 rounded-full"
                              style={{ backgroundColor: tag.color }}
                            />
                            <span className="font-medium">{tag.name}</span>
                          </div>
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => handleDelete(tag.id)}
                            className="h-8 w-8"
                          >
                            <Trash2 className="h-3.5 w-3.5 text-destructive" />
                          </Button>
                        </div>
                        <div className="flex items-center gap-4 text-xs text-muted-foreground">
                          <span className="flex items-center gap-1">
                            <Server className="h-3 w-3" />
                            {tag.nodeCount} Nodes
                          </span>
                          <span className="flex items-center gap-1">
                            <Monitor className="h-3 w-3" />
                            {tag.vmCount} VMs
                          </span>
                        </div>
                        <div className="mt-2 flex items-center gap-2 flex-wrap">
                          {tag.nodeIds.map((nid) => {
                            const node = nodes.find((n) => n.id === nid);
                            return (
                              <Badge
                                key={nid}
                                variant="outline"
                                className="text-xs"
                              >
                                {node?.name || nid.slice(0, 8)}
                              </Badge>
                            );
                          })}
                          <Button
                            variant="outline"
                            size="sm"
                            className="h-6 text-xs px-2"
                            onClick={() => {
                              setBulkPreselectedTag(tag.id);
                              setBulkTagOpen(true);
                            }}
                          >
                            <Tags className="mr-1 h-3 w-3" />
                            VMs zuweisen
                          </Button>
                        </div>
                        {tag.vmCount > 0 && (
                          <div className="mt-2">
                            <button
                              className="text-xs text-muted-foreground hover:text-foreground flex items-center gap-1"
                              onClick={() =>
                                setExpandedTags((prev) => {
                                  const next = new Set(prev);
                                  if (next.has(tag.id)) {
                                    next.delete(tag.id);
                                  } else {
                                    next.add(tag.id);
                                  }
                                  return next;
                                })
                              }
                            >
                              {expandedTags.has(tag.id) ? (
                                <ChevronDown className="h-3 w-3" />
                              ) : (
                                <ChevronRight className="h-3 w-3" />
                              )}
                              {tag.vmCount} VMs anzeigen
                            </button>
                            {expandedTags.has(tag.id) && (
                              <div className="mt-1 space-y-0.5 pl-4">
                                {tag.vmTags.map((vt) => {
                                  const node = nodes.find(
                                    (n) => n.id === vt.node_id
                                  );
                                  return (
                                    <div
                                      key={`${vt.node_id}-${vt.vmid}`}
                                      className="flex items-center gap-2 text-xs text-muted-foreground"
                                    >
                                      <Monitor className="h-3 w-3" />
                                      <span className="font-mono">
                                        {vt.vmid}
                                      </span>
                                      <span>
                                        ({node?.name || vt.node_id.slice(0, 8)})
                                      </span>
                                      <Badge
                                        variant="outline"
                                        className="text-[10px] h-4 px-1"
                                      >
                                        {vt.vm_type}
                                      </Badge>
                                    </div>
                                  );
                                })}
                              </div>
                            )}
                          </div>
                        )}
                      </CardContent>
                    </Card>
                  ))}
                </div>
              </div>
            );
          })}

          {tags.length === 0 && (
            <Card>
              <CardContent className="py-12 text-center text-muted-foreground">
                Noch keine Tags erstellt. Erstelle einen neuen Tag oder
                synchronisiere von Proxmox.
              </CardContent>
            </Card>
          )}
        </>
      )}

      {/* Bulk Assignment */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Bulk-Zuweisung</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            Waehle einen Tag und mehrere Nodes aus, um den Tag gleichzeitig
            zuzuweisen.
          </p>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label>Tag auswaehlen</Label>
              <select
                className="w-full rounded-md border bg-background px-3 py-2 text-sm"
                value={bulkTagId}
                onChange={(e) => setBulkTagId(e.target.value)}
              >
                <option value="">Tag auswaehlen...</option>
                {tags.map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.name}
                    {t.category ? ` (${t.category})` : ""}
                  </option>
                ))}
              </select>
            </div>
            <div className="space-y-2">
              <Label>Nodes auswaehlen</Label>
              <div className="max-h-40 overflow-y-auto space-y-1 rounded-md border p-2">
                {pveNodes.map((node) => (
                  <label
                    key={node.id}
                    className="flex items-center gap-2 px-1 py-0.5 text-sm cursor-pointer hover:bg-accent rounded"
                  >
                    <Checkbox
                      checked={selectedNodeIds.includes(node.id)}
                      onCheckedChange={() => toggleNodeSelection(node.id)}
                    />
                    <span
                      className={`h-2 w-2 rounded-full ${
                        node.is_online ? "bg-green-500" : "bg-red-500"
                      }`}
                    />
                    {node.name}
                  </label>
                ))}
                {pveNodes.length === 0 && (
                  <p className="text-xs text-muted-foreground py-2 text-center">
                    Keine PVE Nodes verfuegbar
                  </p>
                )}
              </div>
            </div>
          </div>
          <Button
            onClick={handleBulkAssign}
            disabled={!bulkTagId || selectedNodeIds.length === 0 || bulkAssigning}
          >
            {bulkAssigning ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Plus className="mr-2 h-4 w-4" />
            )}
            Tag an {selectedNodeIds.length} Nodes zuweisen
          </Button>
        </CardContent>
      </Card>

      {/* Bulk VM Tag Dialog */}
      <BulkTagDialog
        open={bulkTagOpen}
        onOpenChange={(open) => {
          setBulkTagOpen(open);
          if (!open) setBulkPreselectedTag(undefined);
        }}
        preselectedTagId={bulkPreselectedTag}
        onComplete={fetchData}
      />

      {/* Tag Policies (future-ready) */}
      <Card>
        <CardHeader
          className="cursor-pointer"
          onClick={() => setPoliciesOpen(!policiesOpen)}
        >
          <div className="flex items-center justify-between">
            <CardTitle className="text-base flex items-center gap-2">
              <ShieldCheck className="h-4 w-4" />
              Tag-Richtlinien
            </CardTitle>
            {policiesOpen ? (
              <ChevronDown className="h-4 w-4 text-muted-foreground" />
            ) : (
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
            )}
          </div>
        </CardHeader>
        {policiesOpen && (
          <CardContent className="space-y-3">
            <p className="text-sm text-muted-foreground">
              Empfohlene Richtlinien fuer die konsistente Tag-Verwendung im
              Cluster.
            </p>
            {defaultPolicies.map((policy) => {
              const matchingTag = tags.find(
                (t) =>
                  t.name.toLowerCase() === policy.requiredTag.toLowerCase()
              );
              const compliance = matchingTag
                ? Math.round(
                    (matchingTag.nodeCount / Math.max(pveNodes.length, 1)) * 100
                  )
                : 0;

              return (
                <div
                  key={policy.id}
                  className="flex items-start gap-3 p-3 rounded-lg border"
                >
                  <div className="mt-0.5">
                    {compliance >= 80 ? (
                      <ShieldCheck className="h-5 w-5 text-green-500" />
                    ) : compliance >= 40 ? (
                      <ShieldAlert className="h-5 w-5 text-amber-500" />
                    ) : (
                      <ShieldAlert className="h-5 w-5 text-red-500" />
                    )}
                  </div>
                  <div className="flex-1">
                    <p className="text-sm font-medium">{policy.name}</p>
                    <p className="text-xs text-muted-foreground">
                      {policy.description}
                    </p>
                  </div>
                  <div className="text-right">
                    <span
                      className={`text-sm font-bold ${
                        compliance >= 80
                          ? "text-green-500"
                          : compliance >= 40
                          ? "text-amber-500"
                          : "text-red-500"
                      }`}
                    >
                      {compliance}%
                    </span>
                    <p className="text-xs text-muted-foreground">Compliance</p>
                  </div>
                </div>
              );
            })}
          </CardContent>
        )}
      </Card>
    </div>
  );
}
