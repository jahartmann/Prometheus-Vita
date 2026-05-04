"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import {
  Activity,
  AlertCircle,
  Bot,
  Brain,
  CheckCircle2,
  MessageSquare,
  PanelLeftClose,
  PanelLeftOpen,
  Plus,
  Server,
  Settings,
  ShieldCheck,
  Sparkles,
  Trash2,
} from "lucide-react";
import Link from "next/link";
import { ToolApprovalCard } from "@/components/approval/tool-approval-card";
import { ChatInput } from "@/components/chat/chat-input";
import { ChatMessage } from "@/components/chat/chat-message";
import { ToolCallCard } from "@/components/chat/tool-call-card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { agentConfigApi } from "@/lib/api";
import { cn } from "@/lib/utils";
import { useChatStore } from "@/stores/chat-store";

const promptSuggestions = [
  {
    icon: Activity,
    label: "Lage bewerten",
    prompt:
      "Fasse die aktuelle globale Infrastruktur-Lage zusammen und nenne die wichtigsten Risiken.",
  },
  {
    icon: Server,
    label: "VMs pruefen",
    prompt:
      "Welche VMs oder Container brauchen gerade Aufmerksamkeit? Pruefe Metriken, Anomalien und Prognosen.",
  },
  {
    icon: ShieldCheck,
    label: "Backups checken",
    prompt:
      "Bewerte Backup- und Migrationsbereitschaft global. Wo fehlen Schutz oder robuste Recovery-Pfade?",
  },
  {
    icon: Brain,
    label: "Wissen nutzen",
    prompt:
      "Nutze gespeichertes Wissen und aktuelle Tools: Welche wiederkehrenden Muster oder Empfehlungen gibt es?",
  },
];

const contextSignals = [
  { icon: Brain, label: "Globaler Kontext", detail: "Wissen + Verlauf" },
  { icon: Activity, label: "Live-Signale", detail: "Tools statt Raten" },
  { icon: ShieldCheck, label: "Freigaben", detail: "Aktionen kontrolliert" },
];

export default function ChatPage() {
  const {
    conversations,
    currentConversation,
    messages,
    toolCalls,
    isLoading,
    isSending,
    error,
    fetchConversations,
    selectConversation,
    sendMessage,
    newConversation,
    deleteConversation,
  } = useChatStore();

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [configuredModel, setConfiguredModel] = useState("");
  const [configuredProvider, setConfiguredProvider] = useState("");

  const fetchAgentConfig = useCallback(async () => {
    try {
      const data = await agentConfigApi.get();
      const cfg = (data?.data || data || {}) as Record<string, string>;
      setConfiguredModel(cfg.llm_model || "");
      setConfiguredProvider(cfg.llm_provider || "");
    } catch {
      // Agent settings are optional during first setup.
    }
  }, []);

  useEffect(() => {
    fetchConversations();
    fetchAgentConfig();
  }, [fetchConversations, fetchAgentConfig]);

  useEffect(() => {
    if (messages.length > 0 || toolCalls.length > 0 || isSending) {
      messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [messages.length, toolCalls.length, isSending]);

  const toolCallsByMessage = new Map<string, typeof toolCalls>();
  for (const tc of toolCalls) {
    const existing = toolCallsByMessage.get(tc.message_id) || [];
    existing.push(tc);
    toolCallsByMessage.set(tc.message_id, existing);
  }

  const orphanToolCalls = toolCalls.filter(
    (tc) => !messages.some((m) => m.id === tc.message_id)
  );

  const displayModel = currentConversation?.model || configuredModel || "Standard";
  const displayProvider = configuredProvider
    ? configuredProvider.charAt(0).toUpperCase() + configuredProvider.slice(1)
    : "";

  const renderConversationList = (closeOnSelect: boolean) => (
    <div className="flex-1 overflow-y-auto p-2">
      {conversations.length === 0 && (
        <div className="rounded-lg border border-dashed px-3 py-6 text-center text-xs text-muted-foreground">
          Noch kein Verlauf
        </div>
      )}
      {conversations.map((conv) => (
        <div
          key={conv.id}
          className={cn(
            "group mb-1 flex cursor-pointer items-center gap-2 rounded-lg px-2.5 py-2.5 transition-colors hover:bg-accent/70",
            currentConversation?.id === conv.id &&
              "bg-accent text-accent-foreground shadow-sm"
          )}
          onClick={() => {
            selectConversation(conv.id);
            if (closeOnSelect) setSidebarOpen(false);
          }}
        >
          <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
            <MessageSquare className="h-3.5 w-3.5" />
          </div>
          <div className="min-w-0 flex-1">
            <div className="truncate text-sm font-medium">{conv.title}</div>
            <div className="mt-0.5 flex items-center gap-1.5">
              <span className="truncate font-mono text-[10px] text-muted-foreground">
                {conv.model}
              </span>
              <span className="whitespace-nowrap text-[10px] text-muted-foreground">
                {new Date(conv.updated_at).toLocaleDateString("de-DE")}
              </span>
            </div>
          </div>
          <button
            className="hidden shrink-0 rounded-md p-1.5 text-muted-foreground hover:bg-destructive/10 hover:text-destructive group-hover:block"
            onClick={(e) => {
              e.stopPropagation();
              deleteConversation(conv.id);
            }}
            aria-label="Konversation loeschen"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </button>
        </div>
      ))}
    </div>
  );

  return (
    <div className="relative flex h-[calc(100vh-3.5rem)] overflow-hidden bg-background">
      {sidebarOpen && (
        <div className="absolute inset-y-0 left-0 z-20 hidden w-72 flex-col border-r bg-background/95 shadow-xl backdrop-blur md:flex">
          <div className="flex items-center justify-between border-b px-3 py-3">
            <div className="min-w-0">
              <h2 className="truncate text-sm font-semibold">Verlauf</h2>
              <p className="text-[11px] text-muted-foreground">
                Kontext bleibt je Unterhaltung erhalten
              </p>
            </div>
            <div className="flex items-center gap-1">
              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8"
                onClick={newConversation}
                title="Neue Unterhaltung"
              >
                <Plus className="h-4 w-4" />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8"
                onClick={() => setSidebarOpen(false)}
                title="Sidebar schliessen"
              >
                <PanelLeftClose className="h-4 w-4" />
              </Button>
            </div>
          </div>
          {renderConversationList(false)}
        </div>
      )}

      {sidebarOpen && (
        <div
          className="fixed inset-0 z-20 bg-black/30 md:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {sidebarOpen && (
        <div className="fixed inset-y-14 left-0 z-30 flex w-72 flex-col border-r bg-background md:hidden">
          <div className="flex items-center justify-between border-b px-4 py-3">
            <div>
              <h2 className="text-sm font-semibold">Verlauf</h2>
              <p className="text-[11px] text-muted-foreground">Konversationen</p>
            </div>
            <div className="flex items-center gap-1">
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7"
                onClick={newConversation}
                title="Neue Unterhaltung"
              >
                <Plus className="h-4 w-4" />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7"
                onClick={() => setSidebarOpen(false)}
                title="Sidebar schliessen"
              >
                <PanelLeftClose className="h-4 w-4" />
              </Button>
            </div>
          </div>
          {renderConversationList(true)}
        </div>
      )}

      <div className="flex min-w-0 flex-1 flex-col">
        <div className="flex items-center justify-between border-b bg-background/95 px-3 py-2.5 md:px-6 md:py-3">
          <div className="flex min-w-0 items-center gap-3">
            {!sidebarOpen && (
              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8 shrink-0"
                onClick={() => setSidebarOpen(true)}
                title="Sidebar oeffnen"
              >
                <PanelLeftOpen className="h-4 w-4" />
              </Button>
            )}
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <Sparkles className="h-4 w-4 shrink-0 text-primary" />
                <h1 className="truncate text-base font-semibold md:text-lg">
                  {currentConversation?.title || "AI Assistant"}
                </h1>
              </div>
              <p className="hidden text-xs text-muted-foreground xl:block">
                Infrastruktur fragen, Kontext pruefen, Aktionen kontrolliert ausfuehren
              </p>
            </div>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            <div className="flex items-center gap-1.5">
              <Badge
                variant={error ? "destructive" : "success"}
                className="hidden h-5 px-1.5 py-0 text-[10px] sm:inline-flex"
              >
                {error ? "Offline" : "Bereit"}
              </Badge>
              {displayProvider && (
                <Badge
                  variant="outline"
                  className="hidden h-5 px-1.5 py-0 text-[10px] sm:inline-flex"
                >
                  {displayProvider}
                </Badge>
              )}
              <Badge variant="secondary" className="h-5 px-1.5 py-0 font-mono text-[10px]">
                {displayModel}
              </Badge>
            </div>
            <Link href="/settings/agent">
              <Button
                variant="ghost"
                size="sm"
                className="h-7 gap-1 text-xs text-muted-foreground hover:text-foreground"
              >
                <Settings className="h-3.5 w-3.5" />
                <span className="hidden sm:inline">Modell aendern</span>
              </Button>
            </Link>
          </div>
        </div>

        <div className="flex-1 overflow-y-auto">
          <div className="mx-auto w-full max-w-5xl px-4 pt-4 md:px-6">
            <div className="mb-4 grid gap-2 min-[900px]:grid-cols-3">
              {contextSignals.map((item) => {
                const Icon = item.icon;
                return (
                  <div
                    key={item.label}
                    className="flex items-center gap-3 rounded-lg border bg-card/70 px-3 py-2.5"
                  >
                    <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
                      <Icon className="h-4 w-4" />
                    </div>
                    <div className="min-w-0">
                      <p className="truncate text-xs font-medium">{item.label}</p>
                      <p className="truncate text-[11px] text-muted-foreground">{item.detail}</p>
                    </div>
                  </div>
                );
              })}
            </div>
            <ToolApprovalCard />
            {error && (
              <div className="mt-3 flex items-start gap-3 rounded-lg border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm">
                <AlertCircle className="mt-0.5 h-4 w-4 shrink-0 text-destructive" />
                <div>
                  <p className="font-medium text-destructive">
                    KI-Assistent momentan nicht bereit
                  </p>
                  <p className="text-muted-foreground">{error}</p>
                </div>
              </div>
            )}
          </div>

          {messages.length === 0 && !isLoading && (
            <div className="mx-auto flex w-full max-w-5xl flex-col gap-5 px-4 py-8 md:px-6 md:py-10">
              <div className="rounded-xl border bg-card p-5 md:p-6">
                <div className="flex flex-col gap-4 min-[900px]:flex-row min-[900px]:items-start min-[900px]:justify-between">
                  <div className="max-w-2xl">
                    <div className="mb-3 flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
                      <Bot className="h-5 w-5" />
                    </div>
                    <p className="text-xl font-semibold tracking-tight">
                      Prometheus AI Assistant
                    </p>
                    <p className="mt-2 text-sm leading-6 text-muted-foreground">
                      Frage global oder node-spezifisch. Der Assistent soll Live-Daten
                      ueber Tools holen, gespeichertes Wissen gezielt abrufen und
                      riskante Aktionen erst nach Freigabe ausfuehren.
                    </p>
                  </div>
                  <div className="min-w-44 rounded-lg border bg-muted/35 p-3 text-xs">
                    <div className="mb-2 flex items-center gap-2 font-medium">
                      <CheckCircle2 className="h-3.5 w-3.5 text-green-500" />
                      Kontextstatus
                    </div>
                    <div className="space-y-1 text-muted-foreground">
                      <div className="flex justify-between gap-4">
                        <span>Scope</span>
                        <span className="font-medium text-foreground">Global</span>
                      </div>
                      <div className="flex justify-between gap-4">
                        <span>Modell</span>
                        <span className="truncate font-mono text-foreground">
                          {displayModel}
                        </span>
                      </div>
                      <div className="flex justify-between gap-4">
                        <span>Tools</span>
                        <span className="font-medium text-foreground">aktiv</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <div className="grid gap-2 min-[900px]:grid-cols-2">
                {promptSuggestions.map((suggestion) => {
                  const Icon = suggestion.icon;
                  return (
                    <button
                      key={suggestion.label}
                      type="button"
                      disabled={isSending || isLoading}
                      onClick={() => sendMessage(suggestion.prompt)}
                      className="group flex min-h-16 items-center gap-3 rounded-lg border bg-card/70 px-4 py-3 text-left transition-colors hover:bg-accent disabled:pointer-events-none disabled:opacity-50"
                    >
                      <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground transition-colors group-hover:text-foreground">
                        <Icon className="h-4 w-4" />
                      </div>
                      <div>
                        <p className="text-sm font-medium">{suggestion.label}</p>
                        <p className="mt-0.5 text-xs leading-5 text-muted-foreground">
                          {suggestion.prompt}
                        </p>
                      </div>
                    </button>
                  );
                })}
              </div>
            </div>
          )}

          <div className="mx-auto w-full max-w-5xl pb-4">
            {messages.map((msg) => (
              <div key={msg.id}>
                <ChatMessage message={msg} />
                {toolCallsByMessage.get(msg.id)?.map((tc) => (
                  <ToolCallCard key={tc.id} toolCall={tc} />
                ))}
              </div>
            ))}
            {orphanToolCalls.map((tc) => (
              <ToolCallCard key={tc.id} toolCall={tc} />
            ))}
            {isSending && (
              <div className="flex gap-3 px-4 py-3 md:px-6">
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-muted">
                  <div className="flex gap-1">
                    <div className="h-1.5 w-1.5 animate-bounce rounded-full bg-muted-foreground [animation-delay:-0.3s]" />
                    <div className="h-1.5 w-1.5 animate-bounce rounded-full bg-muted-foreground [animation-delay:-0.15s]" />
                    <div className="h-1.5 w-1.5 animate-bounce rounded-full bg-muted-foreground" />
                  </div>
                </div>
              </div>
            )}
            <div ref={messagesEndRef} />
          </div>
        </div>

        <div className="mx-auto w-full max-w-5xl">
          <ChatInput
            onSend={(msg) => sendMessage(msg)}
            isSending={isSending}
            disabled={isLoading}
          />
        </div>
      </div>
    </div>
  );
}
