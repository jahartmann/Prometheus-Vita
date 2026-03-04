"use client";

import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { useEnvironmentStore } from "@/stores/environment-store";
import type { Environment } from "@/types/api";

interface EnvironmentFormProps {
  environment?: Environment | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function EnvironmentForm({ environment, open, onOpenChange, onSuccess }: EnvironmentFormProps) {
  const { createEnvironment, updateEnvironment } = useEnvironmentStore();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [color, setColor] = useState("#3b82f6");
  const [saving, setSaving] = useState(false);

  const isEdit = !!environment;

  useEffect(() => {
    if (environment) {
      setName(environment.name);
      setDescription(environment.description || "");
      setColor(environment.color || "#3b82f6");
    } else {
      setName("");
      setDescription("");
      setColor("#3b82f6");
    }
  }, [environment, open]);

  const handleSubmit = async () => {
    if (!name.trim()) return;
    setSaving(true);
    try {
      if (isEdit && environment) {
        await updateEnvironment(environment.id, { name, description, color });
      } else {
        await createEnvironment({ name, description, color });
      }
      onSuccess();
      onOpenChange(false);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEdit ? "Umgebung bearbeiten" : "Neue Umgebung"}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label>Name</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="z.B. Produktion" />
          </div>
          <div className="space-y-2">
            <Label>Beschreibung</Label>
            <Input value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Optionale Beschreibung" />
          </div>
          <div className="space-y-2">
            <Label>Farbe</Label>
            <div className="flex items-center gap-2">
              <input
                type="color"
                value={color}
                onChange={(e) => setColor(e.target.value)}
                className="h-8 w-8 rounded cursor-pointer"
              />
              <Input value={color} onChange={(e) => setColor(e.target.value)} className="flex-1" />
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Abbrechen</Button>
          <Button onClick={handleSubmit} disabled={saving || !name.trim()}>
            {saving ? "Speichern..." : isEdit ? "Aktualisieren" : "Erstellen"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
