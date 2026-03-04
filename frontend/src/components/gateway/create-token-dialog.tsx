"use client";

import { useState } from "react";
import { Copy, Check } from "lucide-react";
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
import { gatewayApi } from "@/lib/api";

interface CreateTokenDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function CreateTokenDialog({ open, onOpenChange, onSuccess }: CreateTokenDialogProps) {
  const [name, setName] = useState("");
  const [saving, setSaving] = useState(false);
  const [createdToken, setCreatedToken] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const handleSubmit = async () => {
    if (!name.trim()) return;
    setSaving(true);
    try {
      const resp = await gatewayApi.createToken({ name });
      const data = resp.data?.data || resp.data;
      setCreatedToken(data.token);
      onSuccess();
    } finally {
      setSaving(false);
    }
  };

  const handleCopy = () => {
    if (createdToken) {
      navigator.clipboard.writeText(createdToken);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const handleClose = () => {
    setName("");
    setCreatedToken(null);
    setCopied(false);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {createdToken ? "Token erstellt" : "Neues API-Token erstellen"}
          </DialogTitle>
        </DialogHeader>

        {createdToken ? (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Kopieren Sie den Token jetzt. Er wird nicht erneut angezeigt.
            </p>
            <div className="flex items-center gap-2">
              <code className="flex-1 bg-muted p-3 rounded text-sm font-mono break-all">
                {createdToken}
              </code>
              <Button variant="outline" size="icon" onClick={handleCopy}>
                {copied ? <Check className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4" />}
              </Button>
            </div>
            <DialogFooter>
              <Button onClick={handleClose}>Schliessen</Button>
            </DialogFooter>
          </div>
        ) : (
          <>
            <div className="space-y-4">
              <div className="space-y-2">
                <Label>Token-Name</Label>
                <Input
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="z.B. CI/CD Pipeline"
                />
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={handleClose}>Abbrechen</Button>
              <Button onClick={handleSubmit} disabled={saving || !name.trim()}>
                {saving ? "Erstelle..." : "Erstellen"}
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
