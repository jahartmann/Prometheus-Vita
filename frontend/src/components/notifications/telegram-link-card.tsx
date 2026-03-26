"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { telegramApi } from "@/lib/api";
import type { TelegramStatus } from "@/types/api";
import {
  MessageCircle,
  Copy,
  Check,
  ExternalLink,
  Loader2,
  LinkIcon,
  Unlink,
  RefreshCw,
} from "lucide-react";

export function TelegramLinkCard() {
  const [status, setStatus] = useState<TelegramStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [linkCode, setLinkCode] = useState<string | null>(null);
  const [botUsername, setBotUsername] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const fetchStatus = useCallback(async () => {
    try {
      const resp = await telegramApi.status();
      const data = resp.data?.data || resp.data;
      setStatus(data);
      if (data.bot_username) setBotUsername(data.bot_username);
      setError(null);
      return data;
    } catch {
      setError("Telegram-Service ist nicht erreichbar.");
      return null;
    }
  }, []);

  // Initial fetch
  useEffect(() => {
    fetchStatus();
  }, [fetchStatus]);

  // Auto-poll while waiting for verification
  useEffect(() => {
    if (!linkCode) return;

    pollRef.current = setInterval(async () => {
      const data = await fetchStatus();
      if (data?.is_verified) {
        setLinkCode(null);
        if (pollRef.current) clearInterval(pollRef.current);
      }
    }, 3000);

    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, [linkCode, fetchStatus]);

  const handleLink = async () => {
    setLoading(true);
    setError(null);
    try {
      const resp = await telegramApi.link();
      const data = resp.data?.data || resp.data;
      setLinkCode(data.verification_code);
      if (data.bot_username) setBotUsername(data.bot_username);
    } catch {
      setError("Telegram-Verknüpfung fehlgeschlagen. Bitte versuchen Sie es später erneut.");
    } finally {
      setLoading(false);
    }
  };

  const handleUnlink = async () => {
    setLoading(true);
    setError(null);
    try {
      await telegramApi.unlink();
      setStatus(null);
      setLinkCode(null);
    } catch {
      setError("Verknüpfung konnte nicht aufgehoben werden.");
    } finally {
      setLoading(false);
    }
  };

  const copyCode = async () => {
    if (!linkCode) return;
    await navigator.clipboard.writeText(linkCode);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  // Build the t.me deep link that auto-sends /start CODE
  const deepLink = botUsername && linkCode
    ? `https://t.me/${botUsername}?start=${linkCode}`
    : null;

  if (!status?.bot_enabled && !status && !error) return null;

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-base flex items-center gap-2">
          <MessageCircle className="h-4 w-4" />
          Telegram-Verknüpfung
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {error && (
          <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
            {error}
          </div>
        )}

        {!status?.bot_enabled && !linkCode && (
          <div className="rounded-md bg-muted p-3 text-sm text-muted-foreground">
            Der Telegram-Bot ist nicht konfiguriert. Setzen Sie die Umgebungsvariable{" "}
            <code className="rounded bg-muted-foreground/10 px-1.5 py-0.5">TELEGRAM_BOT_TOKEN</code>{" "}
            und starten Sie den Server neu.
          </div>
        )}

        {/* Verified / Linked State */}
        {status?.is_verified ? (
          <div className="space-y-3">
            <div className="flex items-center gap-2">
              <Badge variant="default" className="gap-1">
                <Check className="h-3 w-3" />
                Verknüpft
              </Badge>
              {status.telegram_username && (
                <span className="text-sm text-muted-foreground">
                  @{status.telegram_username}
                </span>
              )}
            </div>
            <p className="text-sm text-muted-foreground">
              Dein Telegram-Konto ist verknüpft. Du kannst mit dem KI-Assistenten über Telegram chatten.
            </p>
            <Button variant="outline" size="sm" onClick={handleUnlink} disabled={loading}>
              <Unlink className="h-3 w-3 mr-1.5" />
              Verknüpfung aufheben
            </Button>
          </div>
        ) : linkCode ? (
          /* Linking in progress - waiting for verification */
          <div className="space-y-4">
            {/* Step 1: Click the link */}
            <div className="space-y-2">
              <div className="flex items-center gap-2 text-sm font-medium">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-primary text-primary-foreground text-xs">
                  1
                </span>
                Bot in Telegram öffnen
              </div>

              {deepLink ? (
                <a
                  href={deepLink}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-2 rounded-md bg-[#2AABEE] px-4 py-2.5 text-sm font-medium text-white hover:bg-[#229ED9] transition-colors"
                >
                  <MessageCircle className="h-4 w-4" />
                  In Telegram öffnen
                  <ExternalLink className="h-3 w-3" />
                </a>
              ) : (
                <p className="text-sm text-muted-foreground">
                  Öffne <span className="font-mono font-medium">@{botUsername || "den Bot"}</span>{" "}
                  in Telegram.
                </p>
              )}
            </div>

            {/* Step 2: Auto or manual */}
            <div className="space-y-2">
              <div className="flex items-center gap-2 text-sm font-medium">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-primary text-primary-foreground text-xs">
                  2
                </span>
                {deepLink
                  ? "Klicke \"Starten\" im Bot"
                  : "Sende den Verifikationscode"}
              </div>

              {!deepLink && (
                <div className="flex items-center gap-2">
                  <div className="rounded-md bg-muted px-3 py-2 font-mono text-sm flex-1">
                    /link {linkCode}
                  </div>
                  <Button variant="outline" size="icon" onClick={copyCode} className="shrink-0">
                    {copied ? (
                      <Check className="h-4 w-4 text-green-500" />
                    ) : (
                      <Copy className="h-4 w-4" />
                    )}
                  </Button>
                </div>
              )}

              {deepLink && (
                <p className="text-sm text-muted-foreground">
                  Der Code wird automatisch übermittelt. Falls es nicht klappt, sende{" "}
                  <button
                    onClick={copyCode}
                    className="inline-flex items-center gap-1 font-mono text-foreground hover:underline cursor-pointer"
                  >
                    /link {linkCode}
                    {copied ? (
                      <Check className="h-3 w-3 text-green-500 inline" />
                    ) : (
                      <Copy className="h-3 w-3 inline" />
                    )}
                  </button>{" "}
                  manuell an den Bot.
                </p>
              )}
            </div>

            {/* Step 3: Waiting */}
            <div className="space-y-2">
              <div className="flex items-center gap-2 text-sm font-medium">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-muted text-muted-foreground text-xs">
                  3
                </span>
                Warte auf Bestätigung
              </div>
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
                Warte auf Verifikation...
              </div>
            </div>
          </div>
        ) : status?.bot_enabled ? (
          /* Initial state - not linked */
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">
              Verknüpfe dein Telegram-Konto, um mit dem KI-Assistenten über Telegram zu chatten
              und Benachrichtigungen zu erhalten.
            </p>
            <Button onClick={handleLink} disabled={loading}>
              {loading ? (
                <Loader2 className="h-4 w-4 mr-1.5 animate-spin" />
              ) : (
                <LinkIcon className="h-4 w-4 mr-1.5" />
              )}
              {loading ? "Verknüpfen..." : "Telegram verknüpfen"}
            </Button>
          </div>
        ) : null}

        {/* Unverified pending code from a previous session */}
        {status && !status.is_verified && status.verification_code && !linkCode && status.bot_enabled && (
          <div className="rounded-md border border-dashed p-3 space-y-2">
            <p className="text-sm text-muted-foreground">
              Es gibt einen offenen Verifikationscode. Moechtest du fortfahren?
            </p>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  setLinkCode(status.verification_code!);
                  if (status.bot_username) setBotUsername(status.bot_username);
                }}
              >
                <RefreshCw className="h-3 w-3 mr-1.5" />
                Fortfahren
              </Button>
              <Button variant="outline" size="sm" onClick={handleLink} disabled={loading}>
                Neuen Code generieren
              </Button>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
