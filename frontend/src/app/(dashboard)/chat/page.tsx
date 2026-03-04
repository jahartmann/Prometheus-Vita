"use client";

import { useEffect, useRef, useState } from "react";
import { Plus, Trash2, MessageSquare, Bot } from "lucide-react";
import { useChatStore } from "@/stores/chat-store";
import { ChatMessage } from "@/components/chat/chat-message";
import { ChatInput } from "@/components/chat/chat-input";
import { ToolCallCard } from "@/components/chat/tool-call-card";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";

const AVAILABLE_MODELS = [
  { value: "llama3", label: "Ollama - Llama 3", group: "Ollama" },
  { value: "mistral", label: "Ollama - Mistral", group: "Ollama" },
  { value: "codellama", label: "Ollama - Code Llama", group: "Ollama" },
  { value: "gpt-4o", label: "GPT-4o", group: "OpenAI" },
  { value: "gpt-4o-mini", label: "GPT-4o Mini", group: "OpenAI" },
  { value: "claude-sonnet-4-20250514", label: "Claude Sonnet 4", group: "Anthropic" },
];

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
  const [selectedModel, setSelectedModel] = useState("llama3");

  useEffect(() => {
    fetchConversations();
  }, [fetchConversations]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, toolCalls]);

  useEffect(() => {
    if (currentConversation?.model) {
      setSelectedModel(currentConversation.model);
    }
  }, [currentConversation?.model]);

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

  return (
    <div className="flex h-[calc(100vh-3.5rem)] overflow-hidden">
      {/* Sidebar */}
      <div className="flex w-64 flex-col border-r bg-card/50">
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
                "group flex cursor-pointer items-center gap-2 px-3 py-3 transition-colors hover:bg-accent/50",
                currentConversation?.id === conv.id &&
                  "bg-accent border-l-2 border-l-primary"
              )}
              onClick={() => selectConversation(conv.id)}
            >
              <MessageSquare className="h-4 w-4 shrink-0 text-muted-foreground" />
              <div className="min-w-0 flex-1">
                <div className="truncate text-sm font-medium">{conv.title}</div>
                <div className="flex items-center gap-1.5 mt-0.5">
                  <span className="text-[10px] text-muted-foreground font-mono">
                    {conv.model}
                  </span>
                  <span className="text-[10px] text-muted-foreground">
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

      {/* Main chat area */}
      <div className="flex flex-1 flex-col">
        {/* Header with model selector */}
        <div className="flex items-center justify-between border-b px-6 py-3">
          <div className="flex items-center gap-3">
            <h1 className="text-lg font-semibold">
              {currentConversation?.title || "AI Assistent"}
            </h1>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs text-muted-foreground">Modell:</span>
            <Select
              value={selectedModel}
              onValueChange={setSelectedModel}
            >
              <SelectTrigger className="h-8 w-48 text-xs">
                <SelectValue placeholder="Modell waehlen" />
              </SelectTrigger>
              <SelectContent>
                {AVAILABLE_MODELS.map((m) => (
                  <SelectItem key={m.value} value={m.value} className="text-xs">
                    {m.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>

        {/* Messages */}
        <div className="flex-1 overflow-y-auto">
          {messages.length === 0 && !isLoading && (
            <div className="flex h-full flex-col items-center justify-center gap-4 text-muted-foreground">
              <div className="flex h-16 w-16 items-center justify-center rounded-full bg-muted">
                <Bot className="h-8 w-8" />
              </div>
              <div className="text-center">
                <p className="text-lg font-medium">Prometheus AI Assistent</p>
                <p className="mt-1 text-sm">
                  Stelle Fragen zu deiner Proxmox-Infrastruktur
                </p>
              </div>
            </div>
          )}
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
          onSend={(msg) => sendMessage(msg, selectedModel)}
          isSending={isSending}
          disabled={isLoading}
        />
      </div>
    </div>
  );
}
