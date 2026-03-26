"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { useBackupStore } from "@/stores/backup-store";
import { toast } from "sonner";

interface CreateBackupDialogProps {
  nodeId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CreateBackupDialog({ nodeId, open, onOpenChange }: CreateBackupDialogProps) {
  const [notes, setNotes] = useState("");
  const [isCreating, setIsCreating] = useState(false);
  const { createBackup } = useBackupStore();

  const handleCreate = async () => {
    setIsCreating(true);
    try {
      await createBackup(nodeId, notes || undefined);
      toast.success("Backup erfolgreich erstellt");
      setNotes("");
      onOpenChange(false);
    } catch (e) {
      console.error('Failed to create backup:', e);
    }
    setIsCreating(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Backup erstellen</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
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
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button onClick={handleCreate} disabled={isCreating}>
            {isCreating ? "Erstelle..." : "Backup erstellen"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
