"use client";

import { useState, useEffect } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Plus, Trash2 } from "lucide-react";
import { escalationApi } from "@/lib/api";
import { toast } from "sonner";
import type { EscalationPolicy, NotificationChannel, CreateEscalationStepInput } from "@/types/api";

interface EscalationPolicyDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
  policy?: EscalationPolicy | null;
  channels: NotificationChannel[];
}

export function EscalationPolicyDialog({
  open,
  onOpenChange,
  onSuccess,
  policy,
  channels,
}: EscalationPolicyDialogProps) {
  const isEdit = !!policy;
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [steps, setSteps] = useState<CreateEscalationStepInput[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (policy) {
      setName(policy.name);
      setDescription(policy.description || "");
      setSteps(
        (policy.steps || []).map((s) => ({
          step_order: s.step_order,
          delay_seconds: s.delay_seconds,
          channel_ids: s.channel_ids || [],
        }))
      );
    } else {
      setName("");
      setDescription("");
      setSteps([{ step_order: 1, delay_seconds: 300, channel_ids: [] }]);
    }
    setError(null);
  }, [policy, open]);

  const addStep = () => {
    setSteps((prev) => [
      ...prev,
      { step_order: prev.length + 1, delay_seconds: 600, channel_ids: [] },
    ]);
  };

  const removeStep = (index: number) => {
    setSteps((prev) => {
      const updated = prev.filter((_, i) => i !== index);
      return updated.map((s, i) => ({ ...s, step_order: i + 1 }));
    });
  };

  const updateStep = (index: number, field: string, value: unknown) => {
    setSteps((prev) =>
      prev.map((s, i) => (i === index ? { ...s, [field]: value } : s))
    );
  };

  const toggleStepChannel = (index: number, channelId: string) => {
    setSteps((prev) =>
      prev.map((s, i) => {
        if (i !== index) return s;
        const ids = s.channel_ids.includes(channelId)
          ? s.channel_ids.filter((id) => id !== channelId)
          : [...s.channel_ids, channelId];
        return { ...s, channel_ids: ids };
      })
    );
  };

  const handleSubmit = async () => {
    setLoading(true);
    setError(null);
    try {
      if (isEdit && policy) {
        await escalationApi.updatePolicy(policy.id, {
          name,
          description,
          steps,
        });
      } else {
        await escalationApi.createPolicy({ name, description, steps });
      }
      onSuccess();
      onOpenChange(false);
      toast.success(isEdit ? "Richtlinie aktualisiert" : "Richtlinie erstellt");
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Fehler beim Speichern");
      toast.error("Fehler beim Speichern der Richtlinie");
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>
            {isEdit ? "Eskalationsrichtlinie bearbeiten" : "Neue Eskalationsrichtlinie"}
          </DialogTitle>
          <DialogDescription>
            Definieren Sie Stufen mit Verzögerungen und Benachrichtigungskanälen.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 max-h-[60vh] overflow-y-auto pr-2">
          <div className="space-y-2">
            <Label>Name</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Kritische Eskalation" />
          </div>

          <div className="space-y-2">
            <Label>Beschreibung</Label>
            <Input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Optionale Beschreibung"
            />
          </div>

          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <Label>Eskalationsstufen</Label>
              <Button variant="outline" size="sm" onClick={addStep}>
                <Plus className="mr-1 h-3 w-3" />
                Stufe
              </Button>
            </div>

            {steps.map((step, index) => (
              <div key={index} className="rounded-md border p-3 space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">Stufe {step.step_order}</span>
                  {steps.length > 1 && (
                    <Button variant="ghost" size="icon" onClick={() => removeStep(index)}>
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  )}
                </div>

                <div className="space-y-1">
                  <Label className="text-xs">Verzögerung (Sekunden)</Label>
                  <Input
                    type="number"
                    value={step.delay_seconds}
                    onChange={(e) => updateStep(index, "delay_seconds", parseInt(e.target.value) || 0)}
                  />
                </div>

                {channels.length > 0 && (
                  <div className="space-y-1">
                    <Label className="text-xs">Kanäle</Label>
                    <div className="space-y-1 rounded border p-2">
                      {channels.map((ch) => (
                        <label key={ch.id} className="flex items-center gap-2 text-xs cursor-pointer">
                          <input
                            type="checkbox"
                            checked={step.channel_ids.includes(ch.id)}
                            onChange={() => toggleStepChannel(index, ch.id)}
                            className="rounded"
                          />
                          {ch.name}
                          <span className="text-muted-foreground">({ch.type})</span>
                        </label>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            ))}
          </div>

          {error && <p className="text-sm text-destructive">{error}</p>}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button onClick={handleSubmit} disabled={loading || !name}>
            {loading ? "Speichern..." : isEdit ? "Aktualisieren" : "Erstellen"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
