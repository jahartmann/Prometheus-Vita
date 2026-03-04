"use client";

import { useState, useEffect } from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { notificationApi } from "@/lib/api";
import type { NotificationChannel } from "@/types/api";
import { Mail, Send, Loader2, CheckCircle2, XCircle, Save } from "lucide-react";

interface SmtpConfigCardProps {
  channels: NotificationChannel[];
  onSaved: () => void;
}

export function SmtpConfigCard({ channels, onSaved }: SmtpConfigCardProps) {
  const smtpChannel = channels.find((c) => c.type === "email");

  const [smtpHost, setSmtpHost] = useState("");
  const [smtpPort, setSmtpPort] = useState("587");
  const [smtpUser, setSmtpUser] = useState("");
  const [smtpPassword, setSmtpPassword] = useState("");
  const [fromAddress, setFromAddress] = useState("");
  const [toAddresses, setToAddresses] = useState("");
  const [useTls, setUseTls] = useState(true);

  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<"success" | "error" | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (smtpChannel) {
      const cfg = smtpChannel.config || {};
      setSmtpHost((cfg.smtp_host as string) || "");
      setSmtpPort(String((cfg.smtp_port as number) || 587));
      setSmtpUser((cfg.smtp_user as string) || "");
      setSmtpPassword("");
      setFromAddress((cfg.from_address as string) || "");
      setToAddresses(
        Array.isArray(cfg.to_addresses) ? (cfg.to_addresses as string[]).join(", ") : ""
      );
      setUseTls((cfg.use_tls as boolean) ?? true);
    }
  }, [smtpChannel]);

  const buildConfig = (): Record<string, unknown> => ({
    smtp_host: smtpHost,
    smtp_port: parseInt(smtpPort) || 587,
    smtp_user: smtpUser,
    smtp_password: smtpPassword,
    from_address: fromAddress,
    to_addresses: toAddresses.split(",").map((s) => s.trim()).filter(Boolean),
    use_tls: useTls,
  });

  const handleSave = async () => {
    setSaving(true);
    setError(null);
    try {
      if (smtpChannel) {
        await notificationApi.updateChannel(smtpChannel.id, {
          name: smtpChannel.name,
          config: buildConfig(),
        });
      } else {
        await notificationApi.createChannel({
          name: "SMTP E-Mail",
          type: "email",
          config: buildConfig(),
        });
      }
      onSaved();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Fehler beim Speichern");
    } finally {
      setSaving(false);
    }
  };

  const handleTest = async () => {
    if (!smtpChannel) return;
    setTesting(true);
    setTestResult(null);
    try {
      await notificationApi.testChannel(smtpChannel.id);
      setTestResult("success");
    } catch {
      setTestResult("error");
    } finally {
      setTesting(false);
    }
  };

  const isValid = smtpHost && smtpPort && fromAddress && toAddresses;

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Mail className="h-5 w-5" />
            <CardTitle className="text-base">SMTP-Konfiguration</CardTitle>
          </div>
          {smtpChannel && (
            <Badge variant={smtpChannel.is_active ? "default" : "secondary"}>
              {smtpChannel.is_active ? "Aktiv" : "Inaktiv"}
            </Badge>
          )}
        </div>
        <CardDescription>
          E-Mail-Benachrichtigungen ueber SMTP versenden.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label>SMTP-Server</Label>
            <Input
              value={smtpHost}
              onChange={(e) => setSmtpHost(e.target.value)}
              placeholder="smtp.example.com"
            />
          </div>
          <div className="space-y-2">
            <Label>Port</Label>
            <Input
              value={smtpPort}
              onChange={(e) => setSmtpPort(e.target.value)}
              placeholder="587"
              type="number"
            />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label>Benutzername</Label>
            <Input
              value={smtpUser}
              onChange={(e) => setSmtpUser(e.target.value)}
              placeholder="user@example.com"
            />
          </div>
          <div className="space-y-2">
            <Label>Passwort</Label>
            <Input
              type="password"
              value={smtpPassword}
              onChange={(e) => setSmtpPassword(e.target.value)}
              placeholder={smtpChannel ? "Unveraendert lassen" : "Passwort"}
            />
          </div>
        </div>

        <div className="space-y-2">
          <Label>Von-Adresse</Label>
          <Input
            type="email"
            value={fromAddress}
            onChange={(e) => setFromAddress(e.target.value)}
            placeholder="alerts@example.com"
          />
        </div>

        <div className="space-y-2">
          <Label>Empfaenger (kommagetrennt)</Label>
          <Input
            value={toAddresses}
            onChange={(e) => setToAddresses(e.target.value)}
            placeholder="admin@example.com, ops@example.com"
          />
        </div>

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={useTls}
            onChange={(e) => setUseTls(e.target.checked)}
            className="rounded"
          />
          TLS aktivieren
        </label>

        {error && <p className="text-sm text-destructive">{error}</p>}

        {testResult === "success" && (
          <div className="flex items-center gap-2 text-sm text-green-600">
            <CheckCircle2 className="h-4 w-4" />
            Test-E-Mail erfolgreich versendet
          </div>
        )}
        {testResult === "error" && (
          <div className="flex items-center gap-2 text-sm text-destructive">
            <XCircle className="h-4 w-4" />
            Test fehlgeschlagen. Konfiguration pruefen.
          </div>
        )}

        <div className="flex gap-2 justify-end">
          {smtpChannel && (
            <Button
              variant="outline"
              onClick={handleTest}
              disabled={testing}
            >
              {testing ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Send className="mr-2 h-4 w-4" />
              )}
              Verbindung testen
            </Button>
          )}
          <Button onClick={handleSave} disabled={saving || !isValid}>
            {saving ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Save className="mr-2 h-4 w-4" />
            )}
            Speichern
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
