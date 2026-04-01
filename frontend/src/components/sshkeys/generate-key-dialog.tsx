"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { sshKeyApi } from "@/lib/api";
import { toast } from "sonner";

interface GenerateKeyDialogProps {
  nodeId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function GenerateKeyDialog({ nodeId, open, onOpenChange, onSuccess }: GenerateKeyDialogProps) {
  const [name, setName] = useState("");
  const [keyType, setKeyType] = useState("ed25519");
  const [deploy, setDeploy] = useState(true);
  const [saving, setSaving] = useState(false);

  const handleSubmit = async () => {
    if (!name.trim()) return;
    setSaving(true);
    try {
      await sshKeyApi.generate(nodeId, { name, key_type: keyType, deploy });
      onSuccess();
      onOpenChange(false);
      setName("");
      toast.success("SSH-Schlüssel generiert");
    } catch {
      toast.error("Fehler beim Generieren des Schlüssels");
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>SSH-Schlüssel generieren</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label>Name</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="z.B. deploy-key-prod" />
          </div>
          <div className="space-y-2">
            <Label>Schlüsseltyp</Label>
            <Select value={keyType} onValueChange={setKeyType}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="ed25519">Ed25519 (empfohlen)</SelectItem>
                <SelectItem value="rsa">RSA (4096 bit)</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="flex items-center gap-2">
            <Checkbox
              id="deploy"
              checked={deploy}
              onCheckedChange={(v) => setDeploy(v === true)}
            />
            <Label htmlFor="deploy" className="cursor-pointer">Direkt deployen</Label>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Abbrechen</Button>
          <Button onClick={handleSubmit} disabled={saving || !name.trim()}>
            {saving ? "Generiere..." : "Generieren"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
