"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Check, Copy, ExternalLink, LinkIcon, MessageCircle, RefreshCw, Unlink } from "lucide-react";
import { Button } from "@/components/ui/button";
import { FeatureStatusCard } from "@/components/ui/feature-status-card";
import { getApiErrorMessage, telegramApi } from "@/lib/api";
import type { TelegramStatus } from "@/types/api";

interface TelegramLinkPayload {
  verification_code?: string;
  bot_username?: string;
}

export function TelegramLinkCard() {
  const [status, setStatus] = useState<TelegramStatus | null>(null);
  const [statusLoaded, setStatusLoaded] = useState(false);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [isLinking, setIsLinking] = useState(false);
  const [isUnlinking, setIsUnlinking] = useState(false);
  const [linkCode, setLinkCode] = useState<string | null>(null);
  const [botUsername, setBotUsername] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const fetchStatus = useCallback(async (showPending = false) => {
    if (showPending) setIsRefreshing(true);
    try {
      const response = await telegramApi.status();
      const data = response.data as TelegramStatus;
      setStatus(data);
      if (data.bot_username) setBotUsername(data.bot_username);
      if (data.verification_code) setLinkCode((current) => current ?? data.verification_code ?? null);
      if (data.is_verified) setLinkCode(null);
      setError(null);
      return data;
    } catch (err: unknown) {
      setError(getApiErrorMessage(err, "Telegram-Status konnte nicht geladen werden"));
      return null;
    } finally {
      setStatusLoaded(true);
      if (showPending) setIsRefreshing(false);
    }
  }, []);

  useEffect(() => {
    fetchStatus(true);
  }, [fetchStatus]);

  useEffect(() => {
    if (!linkCode) return;

    pollRef.current = setInterval(async () => {
      const data = await fetchStatus();
      if (data?.is_verified && pollRef.current) {
        clearInterval(pollRef.current);
      }
    }, 3000);

    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, [fetchStatus, linkCode]);

  const handleLink = async () => {
    setIsLinking(true);
    setError(null);
    try {
      const response = await telegramApi.link();
      const data = response.data as TelegramLinkPayload;
      setLinkCode(data.verification_code ?? null);
      if (data.bot_username) setBotUsername(data.bot_username);
      await fetchStatus();
    } catch (err: unknown) {
      setError(getApiErrorMessage(err, "Telegram-Verknüpfung konnte nicht gestartet werden"));
    } finally {
      setIsLinking(false);
    }
  };

  const handleUnlink = async () => {
    setIsUnlinking(true);
    setError(null);
    try {
      await telegramApi.unlink();
      setLinkCode(null);
      await fetchStatus();
    } catch (err: unknown) {
      setError(getApiErrorMessage(err, "Telegram-Verknüpfung konnte nicht aufgehoben werden"));
    } finally {
      setIsUnlinking(false);
    }
  };

  const copyStartCommand = async () => {
    if (!linkCode) return;
    await navigator.clipboard.writeText(`/start ${linkCode}`);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const effectiveBotUsername = botUsername ?? status?.bot_username ?? null;
  const deepLink = effectiveBotUsername && linkCode
    ? `https://t.me/${effectiveBotUsername}?start=${linkCode}`
    : null;
  const isPending = isRefreshing || isLinking || isUnlinking;

  const { tone, label } = useMemo(() => {
    if (!statusLoaded) return { tone: "muted" as const, label: "Lade Status" };
    if (!status?.bot_enabled) return { tone: "warning" as const, label: "Bot inaktiv" };
    if (status.linked && status.is_verified) return { tone: "ok" as const, label: "Verbunden" };
    return { tone: "info" as const, label: "Nicht verbunden" };
  }, [status, statusLoaded]);

  const details = (
    <div className="space-y-4">
      <dl className="grid gap-2 text-sm sm:grid-cols-2">
        <div>
          <dt className="text-muted-foreground">Link</dt>
          <dd className="font-medium">{statusLoaded ? (status?.linked ? "vorhanden" : "nicht vorhanden") : "wird geladen"}</dd>
        </div>
        <div>
          <dt className="text-muted-foreground">Verifiziert</dt>
          <dd className="font-medium">{statusLoaded ? (status?.is_verified ? "ja" : "nein") : "wird geladen"}</dd>
        </div>
        <div>
          <dt className="text-muted-foreground">Bot</dt>
          <dd className="font-medium">{statusLoaded ? (status?.bot_enabled ? "aktiv" : "inaktiv") : "wird geladen"}</dd>
        </div>
        <div>
          <dt className="text-muted-foreground">Bot-Username</dt>
          <dd className="font-medium">{effectiveBotUsername ? `@${effectiveBotUsername}` : "-"}</dd>
        </div>
        {status?.telegram_username && (
          <div>
            <dt className="text-muted-foreground">Telegram-User</dt>
            <dd className="font-medium">@{status.telegram_username}</dd>
          </div>
        )}
      </dl>

      {!statusLoaded && (
        <p className="text-sm text-muted-foreground">Telegram-Status wird geladen.</p>
      )}

      {statusLoaded && !status?.bot_enabled && (
        <p className="rounded-md bg-muted p-3 text-sm text-muted-foreground">
          Der Telegram-Bot ist nicht konfiguriert. Setze TELEGRAM_BOT_TOKEN und starte den Server neu.
        </p>
      )}

      {linkCode && !status?.is_verified && (
        <div className="space-y-3 rounded-md border border-dashed p-3">
          <p className="text-sm text-muted-foreground">
            Sende den Startbefehl an den Telegram-Bot, um die Verknüpfung zu verifizieren.
          </p>
          <div className="flex flex-wrap items-center gap-2">
            <code className="rounded-full bg-muted px-3 py-1.5 font-mono text-sm">
              /start {linkCode}
            </code>
            <Button variant="outline" size="sm" onClick={copyStartCommand}>
              {copied ? <Check className="mr-2 h-4 w-4" /> : <Copy className="mr-2 h-4 w-4" />}
              {copied ? "Kopiert" : "Kopieren"}
            </Button>
            {deepLink && (
              <Button variant="outline" size="sm" asChild>
                <a href={deepLink} target="_blank" rel="noopener noreferrer">
                  <ExternalLink className="mr-2 h-4 w-4" />
                  Telegram öffnen
                </a>
              </Button>
            )}
          </div>
          <p className="text-xs text-muted-foreground">
            Nach dem Senden wird der Status automatisch aktualisiert.
          </p>
        </div>
      )}

      {status?.bot_enabled && status.linked && status.is_verified && (
        <p className="text-sm text-muted-foreground">
          Telegram ist für Benachrichtigungen und den KI-Assistenten verknüpft.
        </p>
      )}

      <div className="flex flex-wrap gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={() => fetchStatus(true)}
          disabled={isPending}
        >
          <RefreshCw className={`mr-2 h-4 w-4 ${isRefreshing ? "animate-spin" : ""}`} />
          Aktualisieren
        </Button>
        {status?.bot_enabled && !(status.linked && status.is_verified) && (
          <Button size="sm" onClick={handleLink} disabled={isPending}>
            {isLinking ? <RefreshCw className="mr-2 h-4 w-4 animate-spin" /> : <LinkIcon className="mr-2 h-4 w-4" />}
            {linkCode ? "Neuen Code erzeugen" : "Telegram verknüpfen"}
          </Button>
        )}
        {status?.linked && (
          <Button variant="outline" size="sm" onClick={handleUnlink} disabled={isPending}>
            {isUnlinking ? <RefreshCw className="mr-2 h-4 w-4 animate-spin" /> : <Unlink className="mr-2 h-4 w-4" />}
            Verknüpfung aufheben
          </Button>
        )}
      </div>
    </div>
  );

  return (
    <FeatureStatusCard
      title="Telegram-Verknüpfung"
      description="Bot-Status, Konto-Link und Verifikation für Telegram-Benachrichtigungen."
      icon={MessageCircle}
      tone={tone}
      status={label}
      details={details}
      error={error}
    />
  );
}
