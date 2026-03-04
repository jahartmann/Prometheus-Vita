"use client";

import { useState } from "react";
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
import { sshKeyApi } from "@/lib/api";

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
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>SSH-Schluessel generieren</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label>Name</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="z.B. deploy-key-prod" />
          </div>
          <div className="space-y-2">
            <Label>Schluesseltyp</Label>
            <select
              value={keyType}
              onChange={(e) => setKeyType(e.target.value)}
              className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm"
            >
              <option value="ed25519">Ed25519 (empfohlen)</option>
              <option value="rsa">RSA (4096 bit)</option>
            </select>
          </div>
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="deploy"
              checked={deploy}
              onChange={(e) => setDeploy(e.target.checked)}
              className="rounded"
            />
            <Label htmlFor="deploy">Direkt deployen</Label>
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
