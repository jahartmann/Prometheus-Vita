"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import {
  Bot,
  User,
  Send,
  Loader2,
  Monitor,
  Container,
  Terminal,
  CheckCircle2,
  XCircle,
  Wrench,
  ChevronDown,
  ChevronRight,
  Zap,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";
import { chatApi } from "@/lib/api";
import { vmCockpitApi } from "@/lib/vm-api";
import { cn } from "@/lib/utils";
import type { VM, ChatMessage, ChatResponse, AgentToolCall } from "@/types/api";

interface AITabProps {
  vm: VM;
  nodeId: string;
}

interface CommandSuggestion {
  command: string;
}

function extractCommands(text: string): CommandSuggestion[] {
  const commands: CommandSuggestion[] = [];
  const codeBlockRegex = /```(?:bash|sh|shell)?\s*\n([\s\S]*?)```/g;
  let match;
  while ((match = codeBlockRegex.exec(text)) !== null) {
    const lines = match[1].trim().split("\n");
    for (const line of lines) {
      const trimmed = line.trim();
      if (trimmed && !trimmed.startsWith("#")) {
        commands.push({ command: trimmed });
      }
    }
  }
  return commands;
}

export function AITab({ vm, nodeId }: AITabProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [toolCalls, setToolCalls] = useState<AgentToolCall[]>([]);
  const [conversationId, setConversationId] = useState<string | null>(null);
  const [input, setInput] = useState("");
  const [isSending, setIsSending] = useState(false);
  const [proactiveMode, setProactiveMode] = useState(false);
  const [expandedToolCalls, setExpandedToolCalls] = useState<Set<string>>(new Set());
  const [executingCommands, setExecutingCommands] = useState<Set<string>>(new Set());
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, toolCalls]);

  const vmContext = `VM: ${vm.name} (VMID: ${vm.vmid}, Typ: ${vm.type === "qemu" ? "QEMU" : "LXC"}, Status: ${vm.status}, Node: ${nodeId})`;

  const sendMessage = useCallback(
    async (text: string) => {
      if (!text.trim() || isSending) return;

      const contextPrefix = conversationId
        ? ""
        : `[VM-Kontext] ${vmContext}\n\n`;

      const tempUserMsg: ChatMessage = {
        id: "temp-" + Date.now(),
        conversation_id: conversationId || "",
        role: "user",
        content: text,
        created_at: new Date().toISOString(),
      };

      setMessages((prev) => [...prev, tempUserMsg]);
      setIsSending(true);
      setInput("");

      try {
        const resp: ChatResponse = await chatApi.chat({
          conversation_id: conversationId || undefined,
          message: contextPrefix + text,
        });

        if (!conversationId) {
          setConversationId(resp.conversation_id);
        }

        setMessages((prev) => [
          ...prev.filter((m) => m.id !== tempUserMsg.id),
          { ...tempUserMsg, id: "user-" + Date.now(), conversation_id: resp.conversation_id },
          resp.message,
        ]);
        setToolCalls((prev) => [...prev, ...(resp.tool_calls || [])]);
      } catch {
        toast.error("Nachricht konnte nicht gesendet werden");
        setMessages((prev) => prev.filter((m) => m.id !== tempUserMsg.id));
      } finally {
        setIsSending(false);
      }
    },
    [conversationId, isSending, vmContext]
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        sendMessage(input);
      }
    },
    [input, sendMessage]
  );

  const handleInput = useCallback((e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setInput(e.target.value);
    const el = e.target;
    el.style.height = "auto";
    el.style.height = Math.min(el.scrollHeight, 150) + "px";
  }, []);

  const toggleToolCall = useCallback((id: string) => {
    setExpandedToolCalls((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const runCommand = useCallback(
    async (command: string) => {
      if (executingCommands.has(command)) return;
      setExecutingCommands((prev) => new Set(prev).add(command));
      try {
        const result = await vmCockpitApi.exec(nodeId, vm.vmid, command.split(/\s+/), vm.type);
        const data = result.data as unknown as { exitcode: number; "out-data": string; "err-data": string };
        const output = data["out-data"] || data["err-data"] || "";
        toast.success(`Befehl ausgefuehrt (Exit: ${data.exitcode})`);

        const resultMsg: ChatMessage = {
          id: "run-" + Date.now(),
          conversation_id: conversationId || "",
          role: "assistant",
          content: `**Befehl ausgefuehrt:** \`${command}\`\n\nExit-Code: ${data.exitcode}\n\`\`\`\n${output}\n\`\`\``,
          created_at: new Date().toISOString(),
        };
        setMessages((prev) => [...prev, resultMsg]);
      } catch {
        toast.error("Befehl konnte nicht ausgefuehrt werden");
      } finally {
        setExecutingCommands((prev) => {
          const next = new Set(prev);
          next.delete(command);
          return next;
        });
      }
    },
    [nodeId, vm.vmid, vm.type, conversationId, executingCommands]
  );

  // Build tool call map
  const toolCallsByMessage = new Map<string, AgentToolCall[]>();
  for (const tc of toolCalls) {
    const existing = toolCallsByMessage.get(tc.message_id) || [];
    existing.push(tc);
    toolCallsByMessage.set(tc.message_id, existing);
  }

  return (
    <Card className="flex flex-col h-[600px]">
      {/* Header with VM Badge */}
      <div className="flex items-center justify-between border-b px-4 py-3">
        <div className="flex items-center gap-2">
          <Bot className="h-5 w-5 text-primary" />
          <span className="font-semibold text-sm">KI-Assistent</span>
        </div>
        <div className="flex items-center gap-3">
          <Badge variant="outline" className="gap-1.5 font-mono text-xs">
            {vm.type === "qemu" ? (
              <Monitor className="h-3 w-3" />
            ) : (
              <Container className="h-3 w-3" />
            )}
            {vm.name} ({vm.vmid})
          </Badge>
          <div className="flex items-center gap-1.5">
            <Switch
              id="proactive-mode"
              checked={proactiveMode}
              onCheckedChange={setProactiveMode}
              className="scale-75"
            />
            <Label htmlFor="proactive-mode" className="text-xs text-muted-foreground cursor-pointer">
              <Zap className="h-3 w-3 inline mr-0.5" />
              Proaktiv
            </Label>
          </div>
        </div>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto">
        {messages.length === 0 && (
          <div className="flex flex-col items-center justify-center h-full px-6 text-center">
            <Bot className="h-10 w-10 text-muted-foreground mb-3" />
            <p className="text-sm text-muted-foreground">
              Stelle eine Frage zu dieser VM. Der Assistent hat Zugriff auf Systeminformationen,
              Dateien und kann Befehle ausfuehren.
            </p>
            <div className="flex flex-wrap gap-2 mt-4 justify-center">
              {[
                "Zeige die laufenden Services",
                "Pruefe den Speicherplatz",
                "Welche Ports sind offen?",
                "Analysiere die Systemlast",
              ].map((suggestion) => (
                <Button
                  key={suggestion}
                  variant="outline"
                  size="sm"
                  className="text-xs"
                  onClick={() => sendMessage(suggestion)}
                >
                  {suggestion}
                </Button>
              ))}
            </div>
          </div>
        )}
        {messages.map((msg) => {
          const isUser = msg.role === "user";
          const msgToolCalls = toolCallsByMessage.get(msg.id) || [];
          const commands = !isUser ? extractCommands(msg.content) : [];

          return (
            <div key={msg.id}>
              <div
                className={cn(
                  "flex gap-3 px-4 py-3",
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
                    "max-w-[85%] rounded-2xl px-4 py-2.5 text-sm leading-relaxed",
                    isUser
                      ? "bg-primary text-primary-foreground rounded-br-md"
                      : "bg-muted text-foreground rounded-bl-md"
                  )}
                >
                  <div className="whitespace-pre-wrap break-words">{msg.content}</div>
                  <div
                    className={cn(
                      "mt-1 text-[10px]",
                      isUser ? "text-primary-foreground/60" : "text-muted-foreground/60"
                    )}
                  >
                    {new Date(msg.created_at).toLocaleTimeString("de-DE", {
                      hour: "2-digit",
                      minute: "2-digit",
                    })}
                  </div>
                </div>
              </div>

              {/* Inline tool call results */}
              {msgToolCalls.map((tc) => {
                const isExpanded = expandedToolCalls.has(tc.id);
                const statusIcon =
                  tc.status === "success" ? (
                    <CheckCircle2 className="h-3.5 w-3.5 text-green-500" />
                  ) : tc.status === "error" ? (
                    <XCircle className="h-3.5 w-3.5 text-destructive" />
                  ) : (
                    <Loader2 className="h-3.5 w-3.5 animate-spin text-blue-500" />
                  );

                return (
                  <div
                    key={tc.id}
                    className={cn(
                      "mx-4 my-1.5 rounded-lg border text-xs",
                      tc.status === "success"
                        ? "border-green-500/20 bg-green-500/5"
                        : tc.status === "error"
                        ? "border-destructive/20 bg-destructive/5"
                        : "border-blue-500/20 bg-blue-500/5"
                    )}
                  >
                    <button
                      onClick={() => toggleToolCall(tc.id)}
                      className="flex w-full items-center gap-2 px-3 py-2 text-left hover:bg-muted/30 rounded-lg"
                    >
                      {isExpanded ? (
                        <ChevronDown className="h-3 w-3 text-muted-foreground" />
                      ) : (
                        <ChevronRight className="h-3 w-3 text-muted-foreground" />
                      )}
                      <Wrench className="h-3 w-3 text-muted-foreground" />
                      <span className="font-medium font-mono">{tc.tool_name}</span>
                      {statusIcon}
                      {tc.duration_ms > 0 && (
                        <span className="ml-auto text-muted-foreground tabular-nums">
                          {tc.duration_ms}ms
                        </span>
                      )}
                    </button>
                    {isExpanded && (
                      <div className="space-y-2 border-t px-3 py-2">
                        <div>
                          <div className="font-medium text-muted-foreground mb-1">Argumente:</div>
                          <pre className="overflow-auto rounded-md bg-muted/50 p-2 text-[11px] max-h-32">
                            {typeof tc.arguments === "string"
                              ? tc.arguments
                              : JSON.stringify(tc.arguments, null, 2)}
                          </pre>
                        </div>
                        {tc.result != null && (
                          <div>
                            <div className="font-medium text-muted-foreground mb-1">Ergebnis:</div>
                            <pre className="max-h-48 overflow-auto rounded-md bg-muted/50 p-2 text-[11px]">
                              {typeof tc.result === "string"
                                ? tc.result
                                : JSON.stringify(tc.result, null, 2)}
                            </pre>
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                );
              })}

              {/* Command suggestion blocks */}
              {commands.length > 0 && (
                <div className="mx-4 my-2 space-y-2">
                  {commands.map((cmd, i) => (
                    <div
                      key={i}
                      className="flex items-center gap-2 rounded-lg border bg-muted/30 px-3 py-2"
                    >
                      <Terminal className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                      <code className="flex-1 font-mono text-xs truncate">{cmd.command}</code>
                      <div className="flex items-center gap-1 shrink-0">
                        <Button
                          size="sm"
                          variant="default"
                          className="h-7 text-xs px-2"
                          disabled={executingCommands.has(cmd.command)}
                          onClick={() => runCommand(cmd.command)}
                        >
                          {executingCommands.has(cmd.command) ? (
                            <Loader2 className="h-3 w-3 animate-spin" />
                          ) : (
                            "Ausfuehren"
                          )}
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          className="h-7 text-xs px-2 text-muted-foreground"
                          onClick={() => toast.info("Befehl abgelehnt")}
                        >
                          Ablehnen
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          );
        })}
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
      <div className="border-t bg-background p-3">
        <div className="flex items-end gap-2">
          <textarea
            ref={textareaRef}
            value={input}
            onChange={handleInput}
            onKeyDown={handleKeyDown}
            placeholder="Frage zur VM stellen... (Shift+Enter fuer Zeilenumbruch)"
            disabled={isSending}
            rows={1}
            className="flex-1 resize-none rounded-xl border bg-muted/50 px-4 py-2.5 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:bg-background disabled:opacity-50 transition-colors"
          />
          <Button
            size="icon"
            className="h-10 w-10 rounded-xl shrink-0"
            onClick={() => sendMessage(input)}
            disabled={!input.trim() || isSending}
          >
            {isSending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Send className="h-4 w-4" />
            )}
          </Button>
        </div>
      </div>
    </Card>
  );
}
