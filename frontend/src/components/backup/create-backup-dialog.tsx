"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { useBackupStore } from "@/stores/backup-store";

interface CreateBackupDialogProps {
  nodeId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CreateBackupDialog({ nodeId, open, onOpenChange }: CreateBackupDialogProps) {
  const [notes, setNotes] = useState("");
  const [isCreating, setIsCreating] = useState(false);
  const { createBackup } = useBackupStore();

  if (!open) return null;

  const handleCreate = async () => {
    setIsCreating(true);
    try {
      await createBackup(nodeId, notes || undefined);
      setNotes("");
      onOpenChange(false);
    } catch {
      /* ignore */
    }
    setIsCreating(false);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>Backup erstellen</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="notes">Notizen (optional)</Label>
            <textarea
              id="notes"
              className="w-full rounded-md border bg-background px-3 py-2 text-sm"
              rows={3}
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="z.B. Vor Kernel-Update..."
            />
          </div>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => onOpenChange(false)}>
              Abbrechen
            </Button>
            <Button onClick={handleCreate} disabled={isCreating}>
              {isCreating ? "Erstelle..." : "Backup erstellen"}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
