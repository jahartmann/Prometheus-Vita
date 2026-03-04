"use client";

import { MessageSquare } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useChatStore } from "@/stores/chat-store";

export function ChatToggle() {
  const { toggleOpen, isOpen } = useChatStore();

  if (isOpen) return null;

  return (
    <Button
      onClick={toggleOpen}
      size="icon"
      className="fixed bottom-6 right-6 z-50 h-12 w-12 rounded-full shadow-lg"
    >
      <MessageSquare className="h-5 w-5" />
    </Button>
  );
}
