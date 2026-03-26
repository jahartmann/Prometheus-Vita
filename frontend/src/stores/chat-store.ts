import { create } from "zustand";
import { toast } from "sonner";
import type {
  ChatConversation,
  ChatMessage,
  ChatResponse,
  AgentToolCall,
} from "@/types/api";
import { chatApi, toArray } from "@/lib/api";

// generateId() is unavailable in non-secure contexts (HTTP without TLS).
function generateId(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  // Fallback: random hex string
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}

interface ChatState {
  conversations: ChatConversation[];
  currentConversation: ChatConversation | null;
  messages: ChatMessage[];
  toolCalls: AgentToolCall[];
  isLoading: boolean;
  isSending: boolean;
  isOpen: boolean;
  error: string | null;

  toggleOpen: () => void;
  setOpen: (open: boolean) => void;
  fetchConversations: () => Promise<void>;
  selectConversation: (id: string) => Promise<void>;
  sendMessage: (message: string, model?: string) => Promise<void>;
  newConversation: () => void;
  deleteConversation: (id: string) => Promise<void>;
}

export const useChatStore = create<ChatState>()((set, get) => ({
  conversations: [],
  currentConversation: null,
  messages: [],
  toolCalls: [],
  isLoading: false,
  isSending: false,
  isOpen: false,
  error: null,

  toggleOpen: () => set((s) => ({ isOpen: !s.isOpen })),
  setOpen: (open: boolean) => set({ isOpen: open }),

  fetchConversations: async () => {
    set({ isLoading: true, error: null });
    try {
      const convs = await chatApi.listConversations();
      set({ conversations: toArray<ChatConversation>(convs), isLoading: false });
    } catch {
      toast.error("Konversationen konnten nicht geladen werden");
      set({ error: "Konversationen konnten nicht geladen werden", isLoading: false });
    }
  },

  selectConversation: async (id: string) => {
    set({ isLoading: true, error: null });
    try {
      const [conv, msgs] = await Promise.all([
        chatApi.getConversation(id),
        chatApi.getMessages(id),
      ]);
      set({
        currentConversation: conv,
        messages: toArray<ChatMessage>(msgs),
        isLoading: false,
      });
    } catch {
      toast.error("Konversation konnte nicht geladen werden");
      set({ error: "Konversation konnte nicht geladen werden", isLoading: false });
    }
  },

  sendMessage: async (message: string, model?: string) => {
    const { currentConversation } = get();

    // Optimistically add user message
    const tempId = "temp-" + generateId();
    const tempUserMsg: ChatMessage = {
      id: tempId,
      conversation_id: currentConversation?.id || "",
      role: "user",
      content: message,
      created_at: new Date().toISOString(),
    };
    set((s) => ({
      messages: [...s.messages, tempUserMsg],
      isSending: true,
      error: null,
    }));

    try {
      // Send without model - backend picks it from agent_config
      const resp: ChatResponse = await chatApi.chat({
        conversation_id: currentConversation?.id,
        message,
        model: model || undefined,
      });

      // If new conversation, update state
      if (!currentConversation) {
        const convs = toArray<ChatConversation>(await chatApi.listConversations());
        const newConv = convs.find(
          (c: ChatConversation) => c.id === resp.conversation_id
        );
        set({
          currentConversation: newConv || null,
          conversations: convs,
        });
      }

      // Replace temp message and add assistant response
      set((s) => ({
        messages: [
          ...s.messages.filter((m) => m.id !== tempUserMsg.id),
          {
            ...tempUserMsg,
            id: "user-" + generateId(),
            conversation_id: resp.conversation_id,
          },
          resp.message,
        ],
        toolCalls: resp.tool_calls || [],
        isSending: false,
      }));

      // Refresh conversation list
      get().fetchConversations();
    } catch (err: unknown) {
      const apiError =
        err && typeof err === "object" && "response" in err
          ? (err as { response?: { data?: { error?: string } } }).response?.data?.error
          : null;
      const errorMsg = apiError || "Nachricht konnte nicht gesendet werden";
      toast.error(errorMsg);
      set((s) => ({
        messages: s.messages.filter((m) => m.id !== tempUserMsg.id),
        error: errorMsg,
        isSending: false,
      }));
    }
  },

  newConversation: () => {
    set({
      currentConversation: null,
      messages: [],
      toolCalls: [],
      error: null,
    });
  },

  deleteConversation: async (id: string) => {
    try {
      await chatApi.deleteConversation(id);
      const { currentConversation } = get();
      if (currentConversation?.id === id) {
        set({ currentConversation: null, messages: [], toolCalls: [] });
      }
      set((s) => ({
        conversations: s.conversations.filter((c) => c.id !== id),
      }));
    } catch {
      toast.error("Konversation konnte nicht gelöscht werden");
      set({ error: "Konversation konnte nicht gelöscht werden" });
    }
  },
}));
