"use client";

import { useEffect, useState } from "react";
import { Plus, Trash2, RefreshCw, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import { tagApi, nodeApi, toArray } from "@/lib/api";
import { toast } from "sonner";
import type { Tag, Node } from "@/types/api";

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

export function TagManager() {
  const [tags, setTags] = useState<Tag[]>([]);
  const [name, setName] = useState("");
  const [color, setColor] = useState("#3b82f6");
  const [category, setCategory] = useState("");
  const [nodes, setNodes] = useState<Node[]>([]);
  const [syncNodeId, setSyncNodeId] = useState("");
  const [syncLoading, setSyncLoading] = useState(false);

  const fetchTags = () => {
    tagApi
      .list()
      .then((res) => setTags(toArray<Tag>(res.data)))
      .catch(() => {});
  };

  useEffect(() => {
    fetchTags();
    nodeApi
      .list()
      .then((res) => {
        const nodeList = toArray<Node>(res.data);
        setNodes(nodeList);
        if (nodeList.length > 0 && !syncNodeId) {
          setSyncNodeId(nodeList[0].id);
        }
      })
      .catch(() => {});
  }, []);

  const handleCreate = async () => {
    if (!name.trim()) return;
    try {
      await tagApi.create({ name: name.trim(), color, category: category || undefined });
      setName("");
      setCategory("");
      fetchTags();
      toast.success("Tag erstellt");
    } catch {
      toast.error("Fehler beim Erstellen des Tags");
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await tagApi.delete(id);
      fetchTags();
      toast.success("Tag geloescht");
    } catch {
      toast.error("Fehler beim Loeschen des Tags");
    }
  };

  const handleSync = async () => {
    if (!syncNodeId) return;
    setSyncLoading(true);
    try {
      const res = await tagApi.syncFromProxmox(syncNodeId);
      const data = res.data as { imported?: number };
      const count = data?.imported ?? 0;
      toast.success(`${count} Tags von Proxmox importiert`);
      fetchTags();
    } catch {
      toast.error("Tag-Sync fehlgeschlagen");
    } finally {
      setSyncLoading(false);
    }
  };

  return (
    <div className="space-y-4">
      <Card>
        <CardContent className="p-4 space-y-3">
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

      <Card>
        <CardContent className="p-4 space-y-3">
          <h3 className="text-sm font-medium">Tags von Proxmox importieren</h3>
          <div className="flex items-center gap-3">
            <select
              className="rounded-md border bg-background px-3 py-2 text-sm"
              value={syncNodeId}
              onChange={(e) => setSyncNodeId(e.target.value)}
            >
              {nodes.filter((n) => n.type === "pve").map((n) => (
                <option key={n.id} value={n.id}>
                  {n.name}
                </option>
              ))}
            </select>
            <Button onClick={handleSync} disabled={syncLoading || !syncNodeId} variant="outline">
              {syncLoading ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <RefreshCw className="mr-2 h-4 w-4" />
              )}
              Tags importieren
            </Button>
          </div>
        </CardContent>
      </Card>

      <div className="space-y-2">
        {tags.map((tag) => (
          <Card key={tag.id}>
            <CardContent className="flex items-center justify-between p-3">
              <div className="flex items-center gap-3">
                <div className="h-4 w-4 rounded-full" style={{ backgroundColor: tag.color }} />
                <span className="font-medium">{tag.name}</span>
                {tag.category && <Badge variant="outline">{tag.category}</Badge>}
              </div>
              <Button variant="ghost" size="icon" onClick={() => handleDelete(tag.id)}>
                <Trash2 className="h-4 w-4 text-destructive" />
              </Button>
            </CardContent>
          </Card>
        ))}
        {tags.length === 0 && (
          <p className="text-sm text-muted-foreground text-center py-8">
            Noch keine Tags erstellt.
          </p>
        )}
      </div>
    </div>
  );
}
