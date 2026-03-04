"use client";

import { useEffect, useState, useCallback } from "react";
import { Plus, Trash2, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { brainApi, toArray } from "@/lib/api";
import { toast } from "sonner";

interface BrainEntry {
  id: string;
  category: string;
  subject: string;
  content: string;
  relevance_score: number;
  access_count: number;
  created_at: string;
}

export default function BrainSettingsPage() {
  const [entries, setEntries] = useState<BrainEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [newEntry, setNewEntry] = useState({ category: "", subject: "", content: "" });
  const [creating, setCreating] = useState(false);
  const fetchEntries = useCallback(async () => {
    try {
      setLoading(true);
      const res = await brainApi.list();
      setEntries(toArray<BrainEntry>(res.data));
    } catch {
      toast.error("Eintraege konnten nicht geladen werden.");
    } finally {
      setLoading(false);
    }
  }, []);

  const handleSearch = useCallback(async () => {
    if (!searchQuery.trim()) {
      fetchEntries();
      return;
    }
    try {
      setLoading(true);
      const res = await brainApi.search(searchQuery);
      setEntries(toArray<BrainEntry>(res.data));
    } catch {
      toast.error("Suche fehlgeschlagen.");
    } finally {
      setLoading(false);
    }
  }, [searchQuery, fetchEntries]);

  useEffect(() => {
    fetchEntries();
  }, [fetchEntries]);

  const handleCreate = async () => {
    if (!newEntry.category || !newEntry.subject || !newEntry.content) return;
    try {
      setCreating(true);
      await brainApi.create(newEntry);
      toast.success("Wissenseintrag wurde gespeichert.");
      setCreateOpen(false);
      setNewEntry({ category: "", subject: "", content: "" });
      fetchEntries();
    } catch {
      toast.error("Eintrag konnte nicht erstellt werden.");
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await brainApi.delete(id);
      toast.success("Eintrag wurde entfernt.");
      setEntries((prev) => prev.filter((e) => e.id !== id));
    } catch {
      toast.error("Eintrag konnte nicht geloescht werden.");
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">Wissensbasis</h2>
          <p className="text-sm text-muted-foreground">
            Wissen des KI-Assistenten verwalten.
          </p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Eintrag erstellen
        </Button>
      </div>

      <div className="flex gap-2">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Wissensbasis durchsuchen..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleSearch()}
            className="pl-9"
          />
        </div>
        <Button variant="outline" onClick={handleSearch}>
          Suchen
        </Button>
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Kategorie</TableHead>
                <TableHead>Thema</TableHead>
                <TableHead>Zugriffe</TableHead>
                <TableHead>Erstellt</TableHead>
                <TableHead className="w-[60px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-center text-muted-foreground py-8">
                    Laden...
                  </TableCell>
                </TableRow>
              ) : entries.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-center text-muted-foreground py-8">
                    Keine Eintraege vorhanden.
                  </TableCell>
                </TableRow>
              ) : (
                entries.map((entry) => (
                  <TableRow key={entry.id}>
                    <TableCell>
                      <Badge variant="secondary">{entry.category}</Badge>
                    </TableCell>
                    <TableCell className="font-medium">{entry.subject}</TableCell>
                    <TableCell>{entry.access_count}</TableCell>
                    <TableCell>
                      {new Date(entry.created_at).toLocaleDateString("de-DE")}
                    </TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleDelete(entry.id)}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Wissenseintrag erstellen</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Kategorie</Label>
              <Input
                placeholder="z.B. node_config, troubleshooting, best_practice"
                value={newEntry.category}
                onChange={(e) => setNewEntry((p) => ({ ...p, category: e.target.value }))}
              />
            </div>
            <div className="space-y-2">
              <Label>Thema</Label>
              <Input
                placeholder="Kurze Beschreibung"
                value={newEntry.subject}
                onChange={(e) => setNewEntry((p) => ({ ...p, subject: e.target.value }))}
              />
            </div>
            <div className="space-y-2">
              <Label>Inhalt</Label>
              <textarea
                className="flex min-h-[120px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                placeholder="Wissensinhalt..."
                value={newEntry.content}
                onChange={(e) => setNewEntry((p) => ({ ...p, content: e.target.value }))}
                rows={5}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)}>
              Abbrechen
            </Button>
            <Button onClick={handleCreate} disabled={creating || !newEntry.category || !newEntry.subject || !newEntry.content}>
              {creating ? "Speichern..." : "Speichern"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
