"use client";

import { useState, useEffect } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useVMGroupStore } from "@/stores/vm-group-store";
import type { VMGroup } from "@/types/api";

interface VMGroupDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  group?: VMGroup | null;
}

export function VMGroupDialog({ open, onOpenChange, group }: VMGroupDialogProps) {
  const { createGroup, updateGroup } = useVMGroupStore();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [tagFilter, setTagFilter] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const isEdit = !!group;

  useEffect(() => {
    if (group) {
      setName(group.name);
      setDescription(group.description || "");
      setTagFilter(group.tag_filter || "");
    } else {
      setName("");
      setDescription("");
      setTagFilter("");
    }
  }, [group, open]);

  const handleSubmit = async () => {
    if (!name.trim()) return;
    setIsSubmitting(true);
    try {
      if (isEdit && group) {
        await updateGroup(group.id, {
          name: name.trim(),
          description: description.trim(),
          tag_filter: tagFilter.trim(),
        });
      } else {
        await createGroup({
          name: name.trim(),
          description: description.trim(),
          tag_filter: tagFilter.trim(),
        });
      }
      onOpenChange(false);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>
            {isEdit ? "VM-Gruppe bearbeiten" : "VM-Gruppe erstellen"}
          </DialogTitle>
          <DialogDescription>
            {isEdit
              ? "Ändern Sie die Eigenschaften der VM-Gruppe."
              : "Erstellen Sie eine neue Gruppe zur Organisation Ihrer VMs."}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="name">Name</Label>
            <Input
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="z.B. Produktion, Entwicklung"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="description">Beschreibung</Label>
            <Input
              id="description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Optionale Beschreibung der Gruppe"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="tagFilter">Tag-Filter (optional)</Label>
            <Input
              id="tagFilter"
              value={tagFilter}
              onChange={(e) => setTagFilter(e.target.value)}
              placeholder="z.B. production"
            />
            <p className="text-xs text-muted-foreground">
              VMs mit diesem Tag werden automatisch der Gruppe zugeordnet.
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button onClick={handleSubmit} disabled={!name.trim() || isSubmitting}>
            {isSubmitting
              ? "Speichern..."
              : isEdit
                ? "Speichern"
                : "Erstellen"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
