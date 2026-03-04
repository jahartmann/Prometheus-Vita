"use client";

import { useEffect, useState } from "react";
import { Plus, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { tagApi } from "@/lib/api";
import type { Tag } from "@/types/api";

interface TagAssignDialogProps {
  nodeId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function TagAssignDialog({ nodeId, open, onOpenChange }: TagAssignDialogProps) {
  const [allTags, setAllTags] = useState<Tag[]>([]);
  const [nodeTags, setNodeTags] = useState<Tag[]>([]);

  const fetchData = () => {
    tagApi
      .list()
      .then((res) => setAllTags(res.data?.data || res.data || []))
      .catch(() => {});
    tagApi
      .getNodeTags(nodeId)
      .then((res) => setNodeTags(res.data?.data || res.data || []))
      .catch(() => {});
  };

  useEffect(() => {
    if (open) fetchData();
  }, [open, nodeId]);

  if (!open) return null;

  const assignedIds = new Set(nodeTags.map((t) => t.id));
  const unassigned = allTags.filter((t) => !assignedIds.has(t.id));

  const handleAdd = async (tagId: string) => {
    await tagApi.addToNode(nodeId, tagId);
    fetchData();
  };

  const handleRemove = async (tagId: string) => {
    await tagApi.removeFromNode(nodeId, tagId);
    fetchData();
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <Card className="w-full max-w-md">
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Tags zuweisen</CardTitle>
            <Button variant="ghost" onClick={() => onOpenChange(false)}>
              Schliessen
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {nodeTags.length > 0 && (
            <div>
              <p className="text-sm font-medium mb-2">Zugewiesen</p>
              <div className="flex flex-wrap gap-2">
                {nodeTags.map((tag) => (
                  <Badge key={tag.id} style={{ backgroundColor: tag.color, color: "white" }}>
                    {tag.name}
                    <button className="ml-1" onClick={() => handleRemove(tag.id)}>
                      <X className="h-3 w-3" />
                    </button>
                  </Badge>
                ))}
              </div>
            </div>
          )}

          {unassigned.length > 0 && (
            <div>
              <p className="text-sm font-medium mb-2">Verfuegbar</p>
              <div className="flex flex-wrap gap-2">
                {unassigned.map((tag) => (
                  <Badge
                    key={tag.id}
                    variant="outline"
                    className="cursor-pointer"
                    onClick={() => handleAdd(tag.id)}
                  >
                    <Plus className="mr-1 h-3 w-3" />
                    {tag.name}
                  </Badge>
                ))}
              </div>
            </div>
          )}

          {allTags.length === 0 && (
            <p className="text-sm text-muted-foreground text-center py-4">
              Keine Tags vorhanden. Erstelle Tags unter Einstellungen.
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
