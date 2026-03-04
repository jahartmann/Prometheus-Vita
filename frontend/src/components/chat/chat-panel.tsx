"use client";

import { useEffect, useRef } from "react";
import { X, Plus, Trash2, MessageSquare } from "lucide-react";
import { useChatStore } from "@/stores/chat-store";
import { ChatMessage } from "./chat-message";
import { ChatInput } from "./chat-input";
import { ToolCallCard } from "./tool-call-card";
import { Button } from "@/components/ui/button";
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

  useEffect(() => {
    if (isOpen) {
      fetchConversations();
    }
  }, [isOpen, fetchConversations]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-y-0 right-0 z-50 flex w-[420px] max-w-full shadow-xl">
      {/* Conversation sidebar */}
      <div className="flex w-[140px] flex-col border-l border-r bg-card">
        <div className="flex items-center justify-between border-b px-2 py-2">
          <span className="text-xs font-semibold">Chats</span>
          <Button
            variant="ghost"
            size="icon"
            className="h-6 w-6"
            onClick={newConversation}
            title="Neuer Chat"
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
              onClick={() => selectConversation(conv.id)}
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
      <div className="flex flex-1 flex-col border-l bg-background">
        {/* Header */}
        <div className="flex items-center justify-between border-b px-4 py-2">
          <h3 className="text-sm font-semibold">
            {currentConversation?.title || "AI Assistent"}
          </h3>
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            onClick={() => setOpen(false)}
          >
            <X className="h-4 w-4" />
          </Button>
        </div>

        {/* Messages */}
        <div className="flex-1 overflow-y-auto">
          {messages.length === 0 && !isLoading && (
            <div className="flex h-full items-center justify-center p-4 text-center text-sm text-muted-foreground">
              Stelle eine Frage zu deiner Proxmox-Infrastruktur
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
