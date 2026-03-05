"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { X, Plus, Trash2, MessageSquare, Menu, Settings } from "lucide-react";
import Link from "next/link";
import { useChatStore } from "@/stores/chat-store";
import { ChatMessage } from "./chat-message";
import { ChatInput } from "./chat-input";
import { ToolCallCard } from "./tool-call-card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { agentConfigApi } from "@/lib/api";
import { cn } from "@/lib/utils";

export function ChatPanel() {
  const {
    isOpen,
    setOpen,
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
  const [showSidebar, setShowSidebar] = useState(false);
  const [configuredModel, setConfiguredModel] = useState("");

  const fetchAgentConfig = useCallback(async () => {
    try {
      const data = await agentConfigApi.get();
      const cfg = (data?.data || data || {}) as Record<string, string>;
      setConfiguredModel(cfg.llm_model || "");
    } catch {
      // ignore
    }
  }, []);

  useEffect(() => {
    if (isOpen) {
      fetchConversations();
      fetchAgentConfig();
    }
  }, [isOpen, fetchConversations, fetchAgentConfig]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  if (!isOpen) return null;

  const displayModel = currentConversation?.model || configuredModel || "Standard";

  // Build tool call map for inline display
  const toolCallsByMessage = new Map<string, typeof toolCalls>();
  for (const tc of toolCalls) {
    const existing = toolCallsByMessage.get(tc.message_id) || [];
    existing.push(tc);
    toolCallsByMessage.set(tc.message_id, existing);
  }

  const orphanToolCalls = toolCalls.filter(
    (tc) => !messages.some((m) => m.id === tc.message_id)
  );

  return (
    <>
      {/* Backdrop for mobile */}
      <div
        className="fixed inset-0 z-40 bg-black/30 sm:hidden"
        onClick={() => setOpen(false)}
      />

      <div className="fixed inset-y-0 right-0 z-50 flex w-full sm:w-[420px] md:w-[480px] max-w-full shadow-xl">
        {/* Conversation sidebar */}
        <div
          className={cn(
            "flex-col border-l border-r bg-card w-[160px] shrink-0",
            showSidebar ? "flex" : "hidden sm:flex"
          )}
        >
          <div className="flex items-center justify-between border-b px-2 py-2">
            <span className="text-xs font-semibold">Chats</span>
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6"
              onClick={newConversation}
              title="Neue Unterhaltung"
            >
              <Plus className="h-3.5 w-3.5" />
            </Button>
          </div>
          <div className="flex-1 overflow-y-auto">
            {conversations.map((conv) => (
              <div
                key={conv.id}
                className={cn(
                  "group flex cursor-pointer items-center gap-1 border-b px-2 py-2 text-xs hover:bg-accent",
                  currentConversation?.id === conv.id && "bg-accent"
                )}
                onClick={() => {
                  selectConversation(conv.id);
                  setShowSidebar(false);
                }}
              >
                <MessageSquare className="h-3 w-3 shrink-0 text-muted-foreground" />
                <span className="flex-1 truncate">{conv.title}</span>
                <button
                  className="hidden shrink-0 text-muted-foreground hover:text-destructive group-hover:block"
                  onClick={(e) => {
                    e.stopPropagation();
                    deleteConversation(conv.id);
                  }}
                >
                  <Trash2 className="h-3 w-3" />
                </button>
              </div>
            ))}
          </div>
        </div>

        {/* Main chat area */}
        <div className="flex flex-1 min-w-0 flex-col border-l bg-background">
          {/* Header */}
          <div className="flex items-center justify-between border-b px-3 py-2">
            <div className="flex items-center gap-2 min-w-0">
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7 sm:hidden shrink-0"
                onClick={() => setShowSidebar((v) => !v)}
              >
                <Menu className="h-4 w-4" />
              </Button>
              <h3 className="text-sm font-semibold truncate">
                {currentConversation?.title || "AI Assistent"}
              </h3>
            </div>
            <div className="flex items-center gap-1.5 shrink-0">
              <Badge variant="secondary" className="text-[10px] px-1.5 py-0 h-5 font-mono">
                {displayModel}
              </Badge>
              <Link href="/settings/agent">
                <Button variant="ghost" size="icon" className="h-6 w-6" title="Modell aendern">
                  <Settings className="h-3.5 w-3.5" />
                </Button>
              </Link>
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7"
                onClick={() => setOpen(false)}
              >
                <X className="h-4 w-4" />
              </Button>
            </div>
          </div>

          {/* Messages */}
          <div className="flex-1 overflow-y-auto">
            {messages.length === 0 && !isLoading && (
              <div className="flex h-full items-center justify-center p-4 text-center text-sm text-muted-foreground">
                Stelle eine Frage zu deiner Proxmox-Infrastruktur
              </div>
            )}
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
              <div className="flex gap-3 px-4 py-3">
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

          {/* Input */}
          <ChatInput
            onSend={(msg) => sendMessage(msg)}
            isSending={isSending}
            disabled={isLoading}
          />
        </div>
      </div>
    </>
  );
}
