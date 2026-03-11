"use client";

import { cn } from "@/lib/utils";
import type { ChatMessage as ChatMessageType } from "@/types/api";
import { Bot, User } from "lucide-react";

interface ChatMessageProps {
  message: ChatMessageType;
}

export function ChatMessage({ message }: ChatMessageProps) {
  const isUser = message.role === "user";
  const isAssistant = message.role === "assistant";
  const isTool = message.role === "tool";

  if (isTool) {
    return null;
  }

  return (
    <div
      className={cn(
        "flex gap-3 px-4 py-3 md:px-6",
        isUser ? "flex-row-reverse" : "flex-row"
      )}
    >
      <div
        className={cn(
          "flex h-8 w-8 shrink-0 items-center justify-center rounded-full",
          isUser
            ? "bg-primary text-primary-foreground"
            : "bg-muted text-muted-foreground"
        )}
      >
        {isUser ? <User className="h-4 w-4" /> : <Bot className="h-4 w-4" />}
      </div>
      <div
        className={cn(
          "max-w-[85%] sm:max-w-[75%] rounded-2xl px-4 py-2.5 text-sm leading-relaxed",
          isUser
            ? "bg-primary text-primary-foreground rounded-br-md"
            : isAssistant
            ? "bg-muted text-foreground rounded-bl-md"
            : "bg-muted/50 text-foreground/70 text-xs"
        )}
      >
        <div className="whitespace-pre-wrap break-words">
          {message.content}
        </div>
        <div
          className={cn(
            "mt-1 text-[10px]",
            isUser ? "text-primary-foreground/60" : "text-muted-foreground/60"
          )}
        >
          {new Date(message.created_at).toLocaleTimeString("de-DE", {
            hour: "2-digit",
            minute: "2-digit",
          })}
        </div>
      </div>
    </div>
  );
}
