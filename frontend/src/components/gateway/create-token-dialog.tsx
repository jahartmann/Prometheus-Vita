"use client";

import { useState } from "react";
import { Check, Copy } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { gatewayApi } from "@/lib/api";
import { toast } from "sonner";

const permissionOptions = [
  { value: "nodes.read", label: "Nodes lesen", description: "Inventar, Status und Metriken abrufen" },
  { value: "nodes.write", label: "Nodes verwalten", description: "Nodes aendern, Tags synchronisieren und Alias setzen" },
  { value: "vms.read", label: "VMs lesen", description: "VMs, Container und VM-Metriken abrufen" },
  { value: "vms.power", label: "VM-Power", description: "VMs starten, stoppen, pausieren und fortsetzen" },
  { value: "vms.write", label: "VMs verwalten", description: "Snapshots, Console, Migration und VM-Cockpit Aktionen" },
  { value: "backups.read", label: "Backups lesen", description: "Backup-Listen, Dateien und Recovery-Guides abrufen" },
  { value: "backups.create", label: "Backups erstellen", description: "Konfigurations- und VM-Backups anstossen" },
  { value: "backups.restore", label: "Backups wiederherstellen", description: "Restore- und DR-Aktionen ausfuehren" },
  { value: "backups.delete", label: "Backups loeschen", description: "Backups und Backup-Zeitplaene entfernen" },
  { value: "logs.read", label: "Logs lesen", description: "Logs, Analysen und Anomalien abrufen" },
  { value: "logs.manage", label: "Logs verwalten", description: "Log-Analysen, Quellen und Reports steuern" },
  { value: "security.manage", label: "Security verwalten", description: "Security-Modus, Incidents und Baselines bearbeiten" },
  { value: "agent.use", label: "KI nutzen", description: "Chat und Wissensabfragen verwenden" },
  { value: "agent.execute", label: "KI-Aktionen", description: "Agent-Aktionen und Approvals ausfuehren" },
  { value: "agent.manage", label: "KI verwalten", description: "Agent-, LLM- und Wissensbasis-Einstellungen aendern" },
  { value: "api_tokens.manage", label: "API-Tokens verwalten", description: "Tokens erstellen, widerrufen und loeschen" },
  { value: "users.manage", label: "Benutzer verwalten", description: "Benutzerkonten und Rollen pflegen" },
  { value: "settings.manage", label: "Einstellungen verwalten", description: "System-, Sicherheits- und Integrationssettings aendern" },
  { value: "audit.read", label: "Audit lesen", description: "Gateway- und Audit-Logs einsehen" },
];

interface CreateTokenDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function CreateTokenDialog({ open, onOpenChange, onSuccess }: CreateTokenDialogProps) {
  const [name, setName] = useState("");
  const [permissions, setPermissions] = useState<string[]>([]);
  const [saving, setSaving] = useState(false);
  const [createdToken, setCreatedToken] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const handleSubmit = async () => {
    if (!name.trim()) return;
    if (permissions.length === 0) {
      toast.error("Waehlen Sie mindestens eine Berechtigung aus");
      return;
    }
    setSaving(true);
    try {
      const resp = await gatewayApi.createToken({ name, permissions });
      const data = resp.data?.data || resp.data;
      setCreatedToken(data.token);
      onSuccess();
      toast.success("Token erstellt");
    } catch {
      toast.error("Fehler beim Erstellen des Tokens");
    } finally {
      setSaving(false);
    }
  };

  const handleCopy = () => {
    if (createdToken) {
      navigator.clipboard.writeText(createdToken);
      setCopied(true);
      toast.success("Token in Zwischenablage kopiert");
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const handleClose = () => {
    setName("");
    setPermissions([]);
    setCreatedToken(null);
    setCopied(false);
    onOpenChange(false);
  };

  const togglePermission = (permission: string, checked: boolean) => {
    setPermissions((current) =>
      checked ? [...current, permission] : current.filter((item) => item !== permission)
    );
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
              <code className="flex-1 break-all rounded bg-muted p-3 font-mono text-sm">
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
              <div className="space-y-2">
                <Label>Berechtigungen</Label>
                <div className="grid max-h-72 gap-2 overflow-y-auto rounded-md border p-2 sm:grid-cols-2">
                  {permissionOptions.map((permission) => (
                    <label
                      key={permission.value}
                      className="flex cursor-pointer items-start gap-2 rounded-md p-2 hover:bg-accent"
                    >
                      <Checkbox
                        checked={permissions.includes(permission.value)}
                        onCheckedChange={(checked) => togglePermission(permission.value, checked === true)}
                        className="mt-0.5"
                      />
                      <span className="space-y-0.5">
                        <span className="block text-sm font-medium">{permission.label}</span>
                        <span className="block text-xs text-muted-foreground">{permission.description}</span>
                      </span>
                    </label>
                  ))}
                </div>
                <p className="text-xs text-muted-foreground">
                  API-Tokens werden zusaetzlich durch die Rolle des Besitzers begrenzt.
                </p>
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={handleClose}>Abbrechen</Button>
              <Button onClick={handleSubmit} disabled={saving || !name.trim() || permissions.length === 0}>
                {saving ? "Erstelle..." : "Erstellen"}
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
