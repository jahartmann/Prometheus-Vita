"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import {
  Plus,
  Trash2,
  MessageSquare,
  Bot,
  Settings,
  PanelLeftClose,
  PanelLeftOpen,
} from "lucide-react";
import Link from "next/link";
import { useChatStore } from "@/stores/chat-store";
import { ChatMessage } from "@/components/chat/chat-message";
import { ChatInput } from "@/components/chat/chat-input";
import { ToolCallCard } from "@/components/chat/tool-call-card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { agentConfigApi } from "@/lib/api";
import { cn } from "@/lib/utils";

export default function ChatPage() {
  const {
    conversations,
    currentConversation,
    messages,
    toolCalls,
    isLoading,
    isSending,
    fetchConversations,
    selectConversation,
    sendMessage,
    newConversation,
    deleteConversation,
  } = useChatStore();

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [configuredModel, setConfiguredModel] = useState("");
  const [configuredProvider, setConfiguredProvider] = useState("");

  // Fetch configured model from agent settings
  const fetchAgentConfig = useCallback(async () => {
    try {
      const data = await agentConfigApi.get();
      const cfg = (data?.data || data || {}) as Record<string, string>;
      setConfiguredModel(cfg.llm_model || "");
      setConfiguredProvider(cfg.llm_provider || "");
    } catch {
      // Settings not configured yet, that is fine
    }
  }, []);

  useEffect(() => {
    fetchConversations();
    fetchAgentConfig();
  }, [fetchConversations, fetchAgentConfig]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, toolCalls]);

  // Build a map of tool calls by message_id for inline display
  const toolCallsByMessage = new Map<string, typeof toolCalls>();
  for (const tc of toolCalls) {
    const existing = toolCallsByMessage.get(tc.message_id) || [];
    existing.push(tc);
    toolCallsByMessage.set(tc.message_id, existing);
  }

  // Tool calls not associated with any message
  const orphanToolCalls = toolCalls.filter(
    (tc) => !messages.some((m) => m.id === tc.message_id)
  );

  const displayModel = currentConversation?.model || configuredModel || "Standard";
  const displayProvider = configuredProvider
    ? configuredProvider.charAt(0).toUpperCase() + configuredProvider.slice(1)
    : "";

  return (
    <div className="flex h-[calc(100vh-3.5rem)] overflow-hidden">
      {/* Sidebar */}
      <div
        className={cn(
          "flex flex-col border-r bg-card transition-all duration-200",
          sidebarOpen
            ? "w-64 md:w-72 lg:w-80"
            : "w-0 overflow-hidden"
        )}
      >
        <div className="flex items-center justify-between border-b px-4 py-3">
          <h2 className="text-sm font-semibold whitespace-nowrap">Konversationen</h2>
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
              className="h-7 w-7 md:hidden"
              onClick={() => setSidebarOpen(false)}
              title="Sidebar schliessen"
            >
              <PanelLeftClose className="h-4 w-4" />
            </Button>
          </div>
        </div>
        <div className="flex-1 overflow-y-auto">
          {conversations.length === 0 && (
            <div className="p-4 text-center text-xs text-muted-foreground">
              Noch keine Konversationen
            </div>
          )}
          {conversations.map((conv) => (
            <div
              key={conv.id}
              className={cn(
                "group flex cursor-pointer items-center gap-2 px-3 py-3 transition-colors hover:bg-accent/50",
                currentConversation?.id === conv.id &&
                  "bg-accent border-l-2 border-l-primary"
              )}
              onClick={() => {
                selectConversation(conv.id);
                // Auto-close sidebar on small screens
                if (window.innerWidth < 768) {
                  setSidebarOpen(false);
                }
              }}
            >
              <MessageSquare className="h-4 w-4 shrink-0 text-muted-foreground" />
              <div className="min-w-0 flex-1">
                <div className="truncate text-sm font-medium">{conv.title}</div>
                <div className="flex items-center gap-1.5 mt-0.5">
                  <span className="text-[10px] text-muted-foreground font-mono truncate">
                    {conv.model}
                  </span>
                  <span className="text-[10px] text-muted-foreground whitespace-nowrap">
                    {new Date(conv.updated_at).toLocaleDateString("de-DE")}
                  </span>
                </div>
              </div>
              <button
                className="hidden shrink-0 rounded p-1 text-muted-foreground hover:bg-destructive/10 hover:text-destructive group-hover:block"
                onClick={(e) => {
                  e.stopPropagation();
                  deleteConversation(conv.id);
                }}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            </div>
          ))}
        </div>
      </div>

      {/* Mobile sidebar overlay */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-20 bg-black/30 md:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Sidebar (mobile: absolute overlay) */}
      {sidebarOpen && (
        <div className="fixed inset-y-14 left-0 z-30 flex w-72 flex-col border-r bg-card md:hidden">
          <div className="flex items-center justify-between border-b px-4 py-3">
            <h2 className="text-sm font-semibold">Konversationen</h2>
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
              >
                <PanelLeftClose className="h-4 w-4" />
              </Button>
            </div>
          </div>
          <div className="flex-1 overflow-y-auto">
            {conversations.length === 0 && (
              <div className="p-4 text-center text-xs text-muted-foreground">
                Noch keine Konversationen
              </div>
            )}
            {conversations.map((conv) => (
              <div
                key={conv.id}
                className={cn(
                  "group flex cursor-pointer items-center gap-2 px-3 py-3 transition-colors hover:bg-accent/50",
                  currentConversation?.id === conv.id &&
                    "bg-accent border-l-2 border-l-primary"
                )}
                onClick={() => {
                  selectConversation(conv.id);
                  setSidebarOpen(false);
                }}
              >
                <MessageSquare className="h-4 w-4 shrink-0 text-muted-foreground" />
                <div className="min-w-0 flex-1">
                  <div className="truncate text-sm font-medium">{conv.title}</div>
                  <div className="flex items-center gap-1.5 mt-0.5">
                    <span className="text-[10px] text-muted-foreground font-mono truncate">
                      {conv.model}
                    </span>
                  </div>
                </div>
                <button
                  className="hidden shrink-0 rounded p-1 text-muted-foreground hover:bg-destructive/10 hover:text-destructive group-hover:block"
                  onClick={(e) => {
                    e.stopPropagation();
                    deleteConversation(conv.id);
                  }}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Main chat area */}
      <div className="flex min-w-0 flex-1 flex-col">
        {/* Header with model badge */}
        <div className="flex items-center justify-between border-b px-3 py-2.5 md:px-6 md:py-3">
          <div className="flex items-center gap-2 min-w-0">
            {!sidebarOpen && (
              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8 shrink-0"
                onClick={() => setSidebarOpen(true)}
                title="Sidebar öffnen"
              >
                <PanelLeftOpen className="h-4 w-4" />
              </Button>
            )}
            <h1 className="text-base font-semibold truncate md:text-lg">
              {currentConversation?.title || "AI Assistent"}
            </h1>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            <div className="flex items-center gap-1.5">
              {displayProvider && (
                <Badge variant="outline" className="text-[10px] px-1.5 py-0 h-5 hidden sm:inline-flex">
                  {displayProvider}
                </Badge>
              )}
              <Badge variant="secondary" className="text-[10px] px-1.5 py-0 h-5 font-mono">
                {displayModel}
              </Badge>
            </div>
            <Link href="/settings/agent">
              <Button variant="ghost" size="sm" className="h-7 gap-1 text-xs text-muted-foreground hover:text-foreground">
                <Settings className="h-3.5 w-3.5" />
                <span className="hidden sm:inline">Modell ändern</span>
              </Button>
            </Link>
          </div>
        </div>

        {/* Messages */}
        <div className="flex-1 overflow-y-auto">
          {messages.length === 0 && !isLoading && (
            <div className="flex h-full flex-col items-center justify-center gap-4 text-muted-foreground px-4">
              <div className="flex h-16 w-16 items-center justify-center rounded-full bg-muted">
                <Bot className="h-8 w-8" />
              </div>
              <div className="text-center max-w-md">
                <p className="text-lg font-medium">Prometheus AI Assistent</p>
                <p className="mt-1 text-sm">
                  Stelle Fragen zu deiner Proxmox-Infrastruktur oder gib Anweisungen.
                </p>
                {configuredModel && (
                  <p className="mt-2 text-xs">
                    Aktives Modell: <span className="font-mono">{configuredModel}</span>
                  </p>
                )}
              </div>
            </div>
          )}
          <div className="mx-auto w-full max-w-4xl">
            {messages.map((msg) => (
              <div key={msg.id}>
                <ChatMessage message={msg} />
                {/* Inline tool calls for this message */}
                {toolCallsByMessage.get(msg.id)?.map((tc) => (
                  <ToolCallCard key={tc.id} toolCall={tc} />
                ))}
              </div>
            ))}
            {/* Orphan tool calls (not associated with a specific message) */}
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

        {/* Input */}
        <div className="mx-auto w-full max-w-4xl">
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
