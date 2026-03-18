"use client";

import { useState } from "react";
import { useNetworkStore } from "@/stores/network-store";
import { networkApi } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";
import { BookMarked, Plus, Check, Trash2, Pencil } from "lucide-react";
import { toast } from "sonner";

interface BaselineManagerProps {
  nodeId: string;
}

export function BaselineManager({ nodeId }: BaselineManagerProps) {
  const rawBaselines = useNetworkStore((s) => s.baselines);
  const baselines = Array.isArray(rawBaselines) ? rawBaselines : [];
  const fetchBaselines = useNetworkStore((s) => s.fetchBaselines);
  const activateBaseline = useNetworkStore((s) => s.activateBaseline);

  const [creating, setCreating] = useState(false);
  const [newLabel, setNewLabel] = useState("");
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editLabel, setEditLabel] = useState("");
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const handleCreate = async () => {
    setCreating(true);
    try {
      await networkApi.createBaseline(nodeId, { label: newLabel || undefined });
      toast.success("Baseline erstellt");
      setNewLabel("");
      fetchBaselines(nodeId);
    } catch {
      toast.error("Baseline konnte nicht erstellt werden");
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (id: string) => {
    setDeletingId(id);
    try {
      await networkApi.deleteBaseline(id);
      toast.success("Baseline gelöscht");
      fetchBaselines(nodeId);
    } catch {
      toast.error("Löschen fehlgeschlagen");
    } finally {
      setDeletingId(null);
    }
  };

  const handleSaveLabel = async (id: string) => {
    try {
      await networkApi.updateBaseline(id, { label: editLabel });
      toast.success("Label gespeichert");
      fetchBaselines(nodeId);
    } catch {
      toast.error("Speichern fehlgeschlagen");
    } finally {
      setEditingId(null);
    }
  };

  return (
    <div className="space-y-4">
      {/* Create new baseline */}
      <div className="flex items-center gap-2">
        <Input
          placeholder="Label (optional)..."
          value={newLabel}
          onChange={(e) => setNewLabel(e.target.value)}
          className="max-w-xs bg-zinc-900 border-zinc-700 text-sm h-8"
          onKeyDown={(e) => e.key === "Enter" && handleCreate()}
        />
        <Button
          size="sm"
          variant="outline"
          className="h-8 gap-1.5 text-xs"
          disabled={creating}
          onClick={handleCreate}
        >
          <Plus className="h-3.5 w-3.5" />
          {creating ? "Erstelle..." : "Aktuellen Zustand als Baseline setzen"}
        </Button>
      </div>

      {/* Baseline list */}
      {baselines.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-10 text-zinc-600">
          <BookMarked className="h-7 w-7 mb-2 opacity-40" />
          <p className="text-sm">Keine Baselines vorhanden.</p>
        </div>
      ) : (
        <div className="space-y-2">
          {baselines.map((b) => (
            <Card
              key={b.id}
              className={`border-zinc-800 ${b.is_active ? "bg-blue-950/20 border-blue-800/40" : "bg-zinc-900/40"}`}
            >
              <CardContent className="py-2.5 px-4">
                <div className="flex items-center gap-3">
                  <BookMarked
                    className={`h-4 w-4 shrink-0 ${b.is_active ? "text-blue-400" : "text-zinc-600"}`}
                  />

                  {/* Label */}
                  <div className="flex-1 min-w-0">
                    {editingId === b.id ? (
                      <div className="flex items-center gap-2">
                        <Input
                          value={editLabel}
                          onChange={(e) => setEditLabel(e.target.value)}
                          className="h-6 text-xs bg-zinc-800 border-zinc-700 max-w-[200px]"
                          onKeyDown={(e) => {
                            if (e.key === "Enter") handleSaveLabel(b.id);
                            if (e.key === "Escape") setEditingId(null);
                          }}
                          autoFocus
                        />
                        <Button
                          size="sm"
                          variant="ghost"
                          className="h-6 w-6 p-0"
                          onClick={() => handleSaveLabel(b.id)}
                        >
                          <Check className="h-3 w-3 text-green-400" />
                        </Button>
                      </div>
                    ) : (
                      <div className="flex items-center gap-2">
                        <span className="text-sm text-zinc-300 truncate">
                          {b.label || <span className="text-zinc-600 italic">Kein Label</span>}
                        </span>
                        {b.is_active && (
                          <Badge className="bg-blue-500/20 text-blue-400 border-blue-500/30 text-[10px]">
                            Aktiv
                          </Badge>
                        )}
                      </div>
                    )}
                    <p className="text-[10px] text-zinc-600 mt-0.5">
                      {new Date(b.created_at).toLocaleString("de-DE")}
                    </p>
                  </div>

                  {/* Actions */}
                  <div className="flex items-center gap-1 shrink-0">
                    {!b.is_active && (
                      <Button
                        size="sm"
                        variant="outline"
                        className="h-7 text-xs gap-1"
                        onClick={() => activateBaseline(b.id)}
                      >
                        <Check className="h-3 w-3" />
                        Aktivieren
                      </Button>
                    )}
                    <Button
                      size="sm"
                      variant="ghost"
                      className="h-7 w-7 p-0 text-zinc-500 hover:text-zinc-200"
                      onClick={() => {
                        setEditingId(b.id);
                        setEditLabel(b.label ?? "");
                      }}
                    >
                      <Pencil className="h-3 w-3" />
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      className="h-7 w-7 p-0 text-zinc-500 hover:text-red-400"
                      disabled={deletingId === b.id}
                      onClick={() => handleDelete(b.id)}
                    >
                      <Trash2 className="h-3 w-3" />
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
