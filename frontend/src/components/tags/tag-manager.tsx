"use client";

import { useEffect, useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import { tagApi } from "@/lib/api";
import type { Tag } from "@/types/api";

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

  const fetchTags = () => {
    tagApi
      .list()
      .then((res) => setTags(res.data?.data || res.data || []))
      .catch(() => {});
  };

  useEffect(() => {
    fetchTags();
  }, []);

  const handleCreate = async () => {
    if (!name.trim()) return;
    await tagApi.create({ name: name.trim(), color, category: category || undefined });
    setName("");
    setCategory("");
    fetchTags();
  };

  const handleDelete = async (id: string) => {
    await tagApi.delete(id);
    fetchTags();
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
