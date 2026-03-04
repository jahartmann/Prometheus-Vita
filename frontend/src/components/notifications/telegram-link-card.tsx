"use client";

import { useState, useEffect, useCallback } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { telegramApi } from "@/lib/api";
import type { TelegramStatus } from "@/types/api";

export function TelegramLinkCard() {
  const [status, setStatus] = useState<TelegramStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [linkCode, setLinkCode] = useState<string | null>(null);
  const [botUsername, setBotUsername] = useState<string | null>(null);

  const fetchStatus = useCallback(async () => {
    try {
      const resp = await telegramApi.status();
      const data = resp.data?.data || resp.data;
      setStatus(data);
    } catch {
      // Telegram might not be enabled
    }
  }, []);

  useEffect(() => {
    fetchStatus();
  }, [fetchStatus]);

  const handleLink = async () => {
    setLoading(true);
    try {
      const resp = await telegramApi.link();
      const data = resp.data?.data || resp.data;
      setLinkCode(data.verification_code);
      setBotUsername(data.bot_username);
      fetchStatus();
    } catch {
      // Error handled silently
    } finally {
      setLoading(false);
    }
  };

  const handleUnlink = async () => {
    setLoading(true);
    try {
      await telegramApi.unlink();
      setStatus(null);
      setLinkCode(null);
    } catch {
      // Error handled silently
    } finally {
      setLoading(false);
    }
  };

  if (!status?.bot_enabled && !status) return null;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Telegram-Verknuepfung</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {status?.is_verified ? (
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <Badge variant="default">Verknuepft</Badge>
              {status.telegram_username && (
                <span className="text-sm text-muted-foreground">
                  @{status.telegram_username}
                </span>
              )}
            </div>
            <p className="text-sm text-muted-foreground">
              Dein Telegram-Konto ist verknuepft. Du kannst mit dem KI-Assistenten ueber Telegram chatten.
            </p>
            <Button variant="destructive" size="sm" onClick={handleUnlink} disabled={loading}>
              Verknuepfung aufheben
            </Button>
          </div>
        ) : linkCode ? (
          <div className="space-y-2">
            <p className="text-sm">
              Oeffne den Bot{" "}
              {botUsername ? (
                <span className="font-mono font-medium">@{botUsername}</span>
              ) : (
                "in Telegram"
              )}{" "}
              und sende den folgenden Befehl:
            </p>
            <div className="rounded-md bg-muted p-3 font-mono text-sm">
              /link {linkCode}
            </div>
            <Button variant="outline" size="sm" onClick={fetchStatus}>
              Status pruefen
            </Button>
          </div>
        ) : (
          <div className="space-y-2">
            <p className="text-sm text-muted-foreground">
              Verknuepfe dein Telegram-Konto, um mit dem KI-Assistenten ueber Telegram zu chatten
              und Benachrichtigungen zu erhalten.
            </p>
            <Button onClick={handleLink} disabled={loading}>
              {loading ? "Verknuepfen..." : "Telegram verknuepfen"}
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
