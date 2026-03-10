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
import { notificationApi } from "@/lib/api";
import { toast } from "sonner";
import type { NotificationChannel, NotificationChannelType } from "@/types/api";

interface ChannelFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
  channel?: NotificationChannel | null;
}

const channelTypes: { value: NotificationChannelType; label: string }[] = [
  { value: "email", label: "E-Mail (SMTP)" },
  { value: "telegram", label: "Telegram" },
  { value: "webhook", label: "Webhook" },
];

export function ChannelFormDialog({
  open,
  onOpenChange,
  onSuccess,
  channel,
}: ChannelFormDialogProps) {
  const isEdit = !!channel;
  const [name, setName] = useState("");
  const [type, setType] = useState<NotificationChannelType>("email");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Email fields
  const [smtpHost, setSmtpHost] = useState("");
  const [smtpPort, setSmtpPort] = useState("587");
  const [smtpUser, setSmtpUser] = useState("");
  const [smtpPassword, setSmtpPassword] = useState("");
  const [fromAddress, setFromAddress] = useState("");
  const [toAddresses, setToAddresses] = useState("");

  // Telegram fields
  const [botToken, setBotToken] = useState("");
  const [chatId, setChatId] = useState("");

  // Webhook fields
  const [webhookUrl, setWebhookUrl] = useState("");
  const [webhookSecret, setWebhookSecret] = useState("");

  useEffect(() => {
    if (channel) {
      setName(channel.name);
      setType(channel.type);
      const cfg = channel.config || {};
      if (channel.type === "email") {
        setSmtpHost((cfg.smtp_host as string) || "");
        setSmtpPort(String((cfg.smtp_port as number) || 587));
        setSmtpUser((cfg.smtp_user as string) || "");
        setSmtpPassword("");
        setFromAddress((cfg.from_address as string) || "");
        setToAddresses(
          Array.isArray(cfg.to_addresses) ? (cfg.to_addresses as string[]).join(", ") : ""
        );
      } else if (channel.type === "telegram") {
        setBotToken("");
        setChatId((cfg.chat_id as string) || "");
      } else if (channel.type === "webhook") {
        setWebhookUrl((cfg.url as string) || "");
        setWebhookSecret("");
      }
    } else {
      setName("");
      setType("email");
      setSmtpHost("");
      setSmtpPort("587");
      setSmtpUser("");
      setSmtpPassword("");
      setFromAddress("");
      setToAddresses("");
      setBotToken("");
      setChatId("");
      setWebhookUrl("");
      setWebhookSecret("");
    }
    setError(null);
  }, [channel, open]);

  const buildConfig = (): Record<string, unknown> => {
    switch (type) {
      case "email":
        return {
          smtp_host: smtpHost,
          smtp_port: parseInt(smtpPort) || 587,
          smtp_user: smtpUser,
          smtp_password: smtpPassword,
          from_address: fromAddress,
          to_addresses: toAddresses.split(",").map((s) => s.trim()).filter(Boolean),
        };
      case "telegram":
        return {
          bot_token: botToken,
          chat_id: chatId,
        };
      case "webhook":
        return {
          url: webhookUrl,
          method: "POST",
          secret: webhookSecret,
        };
      default:
        return {};
    }
  };

  const handleSubmit = async () => {
    setLoading(true);
    setError(null);
    try {
      const config = buildConfig();
      if (isEdit && channel) {
        await notificationApi.updateChannel(channel.id, { name, config });
        toast.success("Kanal aktualisiert");
      } else {
        await notificationApi.createChannel({ name, type, config });
        toast.success("Kanal erstellt");
      }
      onSuccess();
      onOpenChange(false);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Fehler beim Speichern");
      toast.error("Fehler beim Speichern des Kanals");
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {isEdit ? "Kanal bearbeiten" : "Neuen Kanal erstellen"}
          </DialogTitle>
          <DialogDescription>
            {isEdit
              ? "Kanal-Konfiguration aktualisieren."
              : "Einen neuen Benachrichtigungskanal konfigurieren."}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label>Name</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Mein Kanal" />
          </div>

          {!isEdit && (
            <div className="space-y-2">
              <Label>Typ</Label>
              <select
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                value={type}
                onChange={(e) => setType(e.target.value as NotificationChannelType)}
              >
                {channelTypes.map((ct) => (
                  <option key={ct.value} value={ct.value}>
                    {ct.label}
                  </option>
                ))}
              </select>
            </div>
          )}

          {type === "email" && (
            <>
              <div className="grid grid-cols-2 gap-2">
                <div className="space-y-2">
                  <Label>SMTP Host</Label>
                  <Input value={smtpHost} onChange={(e) => setSmtpHost(e.target.value)} placeholder="smtp.example.com" />
                </div>
                <div className="space-y-2">
                  <Label>SMTP Port</Label>
                  <Input value={smtpPort} onChange={(e) => setSmtpPort(e.target.value)} placeholder="587" />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-2">
                <div className="space-y-2">
                  <Label>SMTP Benutzer</Label>
                  <Input value={smtpUser} onChange={(e) => setSmtpUser(e.target.value)} />
                </div>
                <div className="space-y-2">
                  <Label>SMTP Passwort</Label>
                  <Input type="password" value={smtpPassword} onChange={(e) => setSmtpPassword(e.target.value)} />
                </div>
              </div>
              <div className="space-y-2">
                <Label>Absender</Label>
                <Input value={fromAddress} onChange={(e) => setFromAddress(e.target.value)} placeholder="alerts@example.com" />
              </div>
              <div className="space-y-2">
                <Label>Empfaenger (kommagetrennt)</Label>
                <Input value={toAddresses} onChange={(e) => setToAddresses(e.target.value)} placeholder="admin@example.com, ops@example.com" />
              </div>
            </>
          )}

          {type === "telegram" && (
            <>
              <div className="space-y-2">
                <Label>Bot Token</Label>
                <Input type="password" value={botToken} onChange={(e) => setBotToken(e.target.value)} placeholder="123456:ABC-..." />
              </div>
              <div className="space-y-2">
                <Label>Chat ID</Label>
                <Input value={chatId} onChange={(e) => setChatId(e.target.value)} placeholder="-1001234567890" />
              </div>
            </>
          )}

          {type === "webhook" && (
            <>
              <div className="space-y-2">
                <Label>URL</Label>
                <Input value={webhookUrl} onChange={(e) => setWebhookUrl(e.target.value)} placeholder="https://hooks.example.com/webhook" />
              </div>
              <div className="space-y-2">
                <Label>Secret (optional, fuer HMAC-Signatur)</Label>
                <Input type="password" value={webhookSecret} onChange={(e) => setWebhookSecret(e.target.value)} />
              </div>
            </>
          )}

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
