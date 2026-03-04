"use client";

import { useEffect, useRef } from "react";
import { Plus, Trash2, MessageSquare } from "lucide-react";
import { useChatStore } from "@/stores/chat-store";
import { ChatMessage } from "@/components/chat/chat-message";
import { ChatInput } from "@/components/chat/chat-input";
import { ToolCallCard } from "@/components/chat/tool-call-card";
import { Button } from "@/components/ui/button";
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

  useEffect(() => {
    fetchConversations();
  }, [fetchConversations]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  return (
    <div className="flex h-[calc(100vh-3.5rem)] overflow-hidden">
      {/* Sidebar */}
      <div className="flex w-64 flex-col border-r bg-card">
        <div className="flex items-center justify-between border-b px-4 py-3">
          <h2 className="text-sm font-semibold">Konversationen</h2>
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            onClick={newConversation}
            title="Neuer Chat"
          >
            <Plus className="h-4 w-4" />
          </Button>
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
                "group flex cursor-pointer items-center gap-2 border-b px-3 py-3 hover:bg-accent",
                currentConversation?.id === conv.id && "bg-accent"
              )}
              onClick={() => selectConversation(conv.id)}
            >
              <MessageSquare className="h-4 w-4 shrink-0 text-muted-foreground" />
              <div className="min-w-0 flex-1">
                <div className="truncate text-sm">{conv.title}</div>
                <div className="text-xs text-muted-foreground">
                  {new Date(conv.updated_at).toLocaleDateString("de-DE")}
                </div>
              </div>
              <button
                className="hidden shrink-0 text-muted-foreground hover:text-destructive group-hover:block"
                onClick={(e) => {
                  e.stopPropagation();
                  deleteConversation(conv.id);
                }}
              >
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
          ))}
        </div>
      </div>

      {/* Main chat area */}
      <div className="flex flex-1 flex-col">
        {/* Header */}
        <div className="flex items-center border-b px-6 py-3">
          <h1 className="text-lg font-semibold">
            {currentConversation?.title || "AI Assistent"}
          </h1>
          {currentConversation && (
            <span className="ml-3 rounded bg-muted px-2 py-0.5 text-xs text-muted-foreground">
              {currentConversation.model}
            </span>
          )}
        </div>

        {/* Messages */}
        <div className="flex-1 overflow-y-auto">
          {messages.length === 0 && !isLoading && (
            <div className="flex h-full flex-col items-center justify-center gap-4 text-muted-foreground">
              <MessageSquare className="h-12 w-12" />
              <div className="text-center">
                <p className="text-lg font-medium">Prometheus AI Assistent</p>
                <p className="mt-1 text-sm">
                  Stelle Fragen zu deiner Proxmox-Infrastruktur
                </p>
              </div>
            </div>
          )}
          {messages.map((msg) => (
            <ChatMessage key={msg.id} message={msg} />
          ))}
          {toolCalls.length > 0 &&
            toolCalls.map((tc) => <ToolCallCard key={tc.id} toolCall={tc} />)}
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
  );
}
