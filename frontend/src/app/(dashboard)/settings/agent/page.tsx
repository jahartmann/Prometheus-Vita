"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { Bot, Cpu, CheckCircle2, Download, HardDrive, KeyRound, MemoryStick, RotateCcw, Save, Sparkles, Server, ShieldAlert, ShieldCheck, Trash2, Wifi, Wrench, Zap } from "lucide-react";
import { toast } from "sonner";
import { agentConfigApi, userApi } from "@/lib/api";
import { useAuthStore } from "@/stores/auth-store";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Slider } from "@/components/ui/slider";

type Provider = "ollama" | "openai" | "anthropic";

interface AgentConfig {
  llm_provider?: Provider | string;
  llm_model?: string;
  ollama_url?: string;
  openai_key?: string;
  openai_key_configured?: string;
  anthropic_key?: string;
  anthropic_key_configured?: string;
  agent_approval_low_risk?: string;
  agent_approval_medium_risk?: string;
  agent_approval_high_risk?: string;
  agent_approval_critical_risk?: string;
  agent_full_auto_allow_low_risk?: string;
}

interface ToolCatalogEntry {
  name: string;
  description: string;
  read_only: boolean;
  supports_dry_run: boolean;
  security: {
    risk: "low" | "medium" | "high" | "critical" | string;
    permission: string;
    action: string;
    requires_dry_run: boolean;
  };
}

interface OllamaModel {
  name: string;
  size: number;
  modified_at: string;
}

interface ModelRecommendation {
  name: string;
  tier: "minimal" | "light" | "standard" | "pro" | "enterprise" | string;
  size_gb: number;
  tool_calling: boolean;
  reasoning?: boolean;
  description: string;
  best_for?: string[];
  pulled: boolean;
  recommended: boolean;
  default: boolean;
}

interface SystemRecommendation {
  system: {
    os: string;
    arch: string;
    cpu_cores: number;
    total_ram_gb: number;
    gpus: Array<{ name: string; vram_gb: number; driver?: string; vendor?: string }>;
    tier: string;
    tier_label: string;
    notes?: string[];
  };
  models: ModelRecommendation[];
  default_model: string;
  ollama_ready: boolean;
  ollama_url: string;
}

const tierTone: Record<string, "outline" | "warning" | "degraded" | "success" | "secondary"> = {
  minimal: "outline",
  light: "secondary",
  standard: "warning",
  pro: "success",
  enterprise: "success",
};

const providers: Array<{ value: Provider; label: string; detail: string }> = [
  { value: "ollama", label: "Ollama", detail: "Lokal oder privat" },
  { value: "openai", label: "OpenAI", detail: "GPT-Modelle" },
  { value: "anthropic", label: "Anthropic", detail: "Claude-Modelle" },
];

const providerModels: Record<Provider, Array<{ value: string; label: string }>> = {
  ollama: [
    { value: "llama3.1:8b", label: "llama3.1:8b — empfohlener Mindeststandard" },
    { value: "llama3.3:70b", label: "llama3.3:70b — Top-Tier" },
    { value: "qwen2.5:14b", label: "qwen2.5:14b — Standard, sehr gut" },
    { value: "qwen2.5:32b", label: "qwen2.5:32b — Pro" },
    { value: "qwen2.5:72b", label: "qwen2.5:72b — Enterprise" },
    { value: "qwen2.5-coder:32b", label: "qwen2.5-coder:32b — Coding-Fokus" },
    { value: "mistral-small:24b", label: "mistral-small:24b" },
    { value: "qwq:32b", label: "qwq:32b — Reasoning" },
    { value: "llama3.2:3b", label: "llama3.2:3b — Minimal" },
  ],
  openai: [
    { value: "gpt-4o", label: "gpt-4o" },
    { value: "gpt-4o-mini", label: "gpt-4o-mini" },
    { value: "gpt-4-turbo", label: "gpt-4-turbo" },
  ],
  anthropic: [
    { value: "claude-sonnet-4-20250514", label: "claude-sonnet-4" },
    { value: "claude-haiku-4-5-20251001", label: "claude-haiku-4.5" },
  ],
};

const autonomyLabels: Record<number, { label: string; description: string; tone: "outline" | "warning" | "degraded" }> = {
  0: {
    label: "Nur Lesen",
    description: "Der Agent darf nur lesende Tools verwenden.",
    tone: "outline",
  },
  1: {
    label: "Mit Freigabe",
    description: "Schreibende Aktionen laufen ueber manuelle Freigaben.",
    tone: "warning",
  },
  2: {
    label: "Vollautomatik",
    description: "Der Agent darf erlaubte Aktionen ohne Rueckfrage ausfuehren.",
    tone: "degraded",
  },
};

export default function AgentSettingsPage() {
  const user = useAuthStore((s) => s.user);
  const [provider, setProvider] = useState<Provider>("ollama");
  const [model, setModel] = useState("llama3.1:8b");
  const [ollamaUrl, setOllamaUrl] = useState("http://localhost:11434");
  const [openaiKey, setOpenaiKey] = useState("");
  const [anthropicKey, setAnthropicKey] = useState("");
  const [openaiConfigured, setOpenaiConfigured] = useState(false);
  const [anthropicConfigured, setAnthropicConfigured] = useState(false);
  const [autonomyLevel, setAutonomyLevel] = useState(1);
  const [approvalLow, setApprovalLow] = useState(false);
  const [approvalMedium, setApprovalMedium] = useState(true);
  const [approvalHigh, setApprovalHigh] = useState(true);
  const [approvalCritical, setApprovalCritical] = useState(true);
  const [fullAutoLowRisk, setFullAutoLowRisk] = useState(false);
  const [tools, setTools] = useState<ToolCatalogEntry[]>([]);
  const [savedSnapshot, setSavedSnapshot] = useState("");
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [secretAction, setSecretAction] = useState<Provider | "">("");
  const [isDiscovering, setIsDiscovering] = useState(false);
  const [ollamaModels, setOllamaModels] = useState<OllamaModel[]>([]);
  const [recommendation, setRecommendation] = useState<SystemRecommendation | null>(null);
  const [pullingModel, setPullingModel] = useState<string>("");

  const snapshot = useMemo(
    () => JSON.stringify({
      provider,
      model,
      ollamaUrl,
      openaiKey,
      anthropicKey,
      autonomyLevel,
      approvalLow,
      approvalMedium,
      approvalHigh,
      approvalCritical,
      fullAutoLowRisk,
    }),
    [provider, model, ollamaUrl, openaiKey, anthropicKey, autonomyLevel, approvalLow, approvalMedium, approvalHigh, approvalCritical, fullAutoLowRisk]
  );
  const isDirty = savedSnapshot !== "" && snapshot !== savedSnapshot;
  const autonomy = autonomyLabels[autonomyLevel] ?? autonomyLabels[1];

  const modelOptions = useMemo(() => {
    if (provider !== "ollama" || ollamaModels.length === 0) {
      return providerModels[provider];
    }
    return ollamaModels.map((entry) => ({
      value: entry.name,
      label: `${entry.name} (${formatSize(entry.size)})`,
    }));
  }, [ollamaModels, provider]);

  const applyConfig = useCallback((config: AgentConfig) => {
    const nextProvider = toProvider(config.llm_provider);
    const nextModel = config.llm_model || "llama3.1:8b";
    const nextOllamaUrl = config.ollama_url || "http://localhost:11434";
    setProvider(nextProvider);
    setModel(nextModel);
    setOllamaUrl(nextOllamaUrl);
    setOpenaiKey("");
    setAnthropicKey("");
    setOpenaiConfigured(config.openai_key_configured === "true");
    setAnthropicConfigured(config.anthropic_key_configured === "true");
    setApprovalLow(config.agent_approval_low_risk === "true");
    setApprovalMedium(config.agent_approval_medium_risk !== "false");
    setApprovalHigh(config.agent_approval_high_risk !== "false");
    setApprovalCritical(config.agent_approval_critical_risk !== "false");
    setFullAutoLowRisk(config.agent_full_auto_allow_low_risk === "true");
    return { provider: nextProvider, model: nextModel, ollamaUrl: nextOllamaUrl };
  }, []);

  const loadConfig = useCallback(async () => {
    setIsLoading(true);
    try {
      const config = (await agentConfigApi.get()) as AgentConfig;
      const applied = applyConfig(config);
      try {
        const toolCatalog = (await agentConfigApi.getTools()) as ToolCatalogEntry[];
        setTools(Array.isArray(toolCatalog) ? toolCatalog : []);
      } catch {
        setTools([]);
      }
      let nextAutonomy = 1;
      if (user?.id) {
        try {
          const response = await userApi.getById(user.id);
          const userData = response.data?.data ?? response.data;
          if (typeof userData?.autonomy_level === "number") {
            nextAutonomy = userData.autonomy_level;
            setAutonomyLevel(nextAutonomy);
          }
        } catch {
          // Some roles can manage agent config without reading user administration.
        }
      }
      setSavedSnapshot(JSON.stringify({
        ...applied,
        openaiKey: "",
        anthropicKey: "",
        autonomyLevel: nextAutonomy,
        approvalLow: config.agent_approval_low_risk === "true",
        approvalMedium: config.agent_approval_medium_risk !== "false",
        approvalHigh: config.agent_approval_high_risk !== "false",
        approvalCritical: config.agent_approval_critical_risk !== "false",
        fullAutoLowRisk: config.agent_full_auto_allow_low_risk === "true",
      }));
    } catch {
      toast.error("Agent-Einstellungen konnten nicht geladen werden");
    } finally {
      setIsLoading(false);
    }
  }, [applyConfig, user?.id]);

  useEffect(() => {
    loadConfig();
  }, [loadConfig]);

  const loadRecommendation = useCallback(async () => {
    try {
      const data = (await agentConfigApi.getRecommendations()) as SystemRecommendation;
      setRecommendation(data);
    } catch {
      // Recommendations are optional — if the endpoint fails (e.g. on a non-Linux
      // host), the rest of the page should still work.
    }
  }, []);

  useEffect(() => {
    loadRecommendation();
  }, [loadRecommendation]);

  const pullModelHandler = useCallback(async (modelName: string) => {
    setPullingModel(modelName);
    toast.message(`Lade ${modelName} — kann einige Minuten dauern…`);
    try {
      await agentConfigApi.pullModel(modelName);
      toast.success(`${modelName} ist geladen`);
      await loadRecommendation();
      // Refresh the Ollama model list so the dropdown picks it up.
      try {
        const models = (await agentConfigApi.testOllamaConnection(ollamaUrl)) as OllamaModel[];
        setOllamaModels(Array.isArray(models) ? models : []);
      } catch {
        // ignore — the user can hit "Test" manually.
      }
    } catch (error: unknown) {
      toast.error(getErrorMessage(error, `${modelName} konnte nicht geladen werden`));
    } finally {
      setPullingModel("");
    }
  }, [loadRecommendation, ollamaUrl]);

  const handleProviderChange = (value: Provider) => {
    setProvider(value);
    setModel(providerModels[value][0]?.value ?? "");
  };

  const discoverOllamaModels = async () => {
    setIsDiscovering(true);
    try {
      const models = (await agentConfigApi.testOllamaConnection(ollamaUrl)) as OllamaModel[];
      setOllamaModels(Array.isArray(models) ? models : []);
      toast.success(`${Array.isArray(models) ? models.length : 0} Ollama-Modelle gefunden`);
    } catch (error: unknown) {
      toast.error(getErrorMessage(error, "Ollama ist nicht erreichbar"));
      setOllamaModels([]);
    } finally {
      setIsDiscovering(false);
    }
  };

  const saveSettings = async () => {
    setIsSaving(true);
    try {
      const payload: Record<string, string> = {
        llm_provider: provider,
        llm_model: model,
        ollama_url: ollamaUrl,
        agent_approval_low_risk: String(approvalLow),
        agent_approval_medium_risk: String(approvalMedium),
        agent_approval_high_risk: String(approvalHigh),
        agent_approval_critical_risk: String(approvalCritical),
        agent_full_auto_allow_low_risk: String(fullAutoLowRisk),
      };
      if (openaiKey.trim() !== "") payload.openai_key = openaiKey.trim();
      if (anthropicKey.trim() !== "") payload.anthropic_key = anthropicKey.trim();

      const updated = (await agentConfigApi.update(payload)) as AgentConfig;
      applyConfig(updated);

      let nextAutonomy = autonomyLevel;
      if (user?.id) {
        await userApi.update(user.id, { autonomy_level: autonomyLevel });
        nextAutonomy = autonomyLevel;
      }
      setSavedSnapshot(JSON.stringify({
        provider,
        model,
        ollamaUrl,
        openaiKey: "",
        anthropicKey: "",
        autonomyLevel: nextAutonomy,
        approvalLow,
        approvalMedium,
        approvalHigh,
        approvalCritical,
        fullAutoLowRisk,
      }));
      toast.success("Agent-Einstellungen gespeichert");
    } catch (error: unknown) {
      toast.error(getErrorMessage(error, "Agent-Einstellungen konnten nicht gespeichert werden"));
    } finally {
      setIsSaving(false);
    }
  };

  const rotateProviderSecret = async (secretProvider: "openai" | "anthropic") => {
    const key = secretProvider === "openai" ? openaiKey.trim() : anthropicKey.trim();
    if (!key) {
      toast.error("Bitte zuerst einen neuen API-Key eintragen");
      return;
    }
    setSecretAction(secretProvider);
    try {
      const updated = (await agentConfigApi.rotateSecret(secretProvider, key)) as AgentConfig;
      applyConfig(updated);
      setOpenaiKey("");
      setAnthropicKey("");
      setSavedSnapshot(snapshotWithoutSecrets(provider, model, ollamaUrl, autonomyLevel, approvalLow, approvalMedium, approvalHigh, approvalCritical, fullAutoLowRisk));
      toast.success("API-Key wurde rotiert");
    } catch (error: unknown) {
      toast.error(getErrorMessage(error, "API-Key konnte nicht rotiert werden"));
    } finally {
      setSecretAction("");
    }
  };

  const deleteProviderSecret = async (secretProvider: "openai" | "anthropic") => {
    if (!window.confirm("API-Key wirklich loeschen? Aktive Cloud-Provider fallen danach auf Ollama zurueck.")) {
      return;
    }
    setSecretAction(secretProvider);
    try {
      const updated = (await agentConfigApi.deleteSecret(secretProvider)) as AgentConfig;
      const applied = applyConfig(updated);
      setOpenaiKey("");
      setAnthropicKey("");
      setSavedSnapshot(snapshotWithoutSecrets(applied.provider, applied.model, applied.ollamaUrl, autonomyLevel, approvalLow, approvalMedium, approvalHigh, approvalCritical, fullAutoLowRisk));
      toast.success("API-Key wurde geloescht");
    } catch (error: unknown) {
      toast.error(getErrorMessage(error, "API-Key konnte nicht geloescht werden"));
    } finally {
      setSecretAction("");
    }
  };

  if (isLoading) {
    return (
      <Card>
        <CardContent className="p-6 text-sm text-muted-foreground">Lade Agent-Einstellungen...</CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-5">
      <div className="rounded-lg border bg-card p-4">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="flex items-start gap-3">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-muted">
              <Bot className="h-4 w-4" />
            </div>
            <div>
              <h2 className="text-xl font-semibold tracking-tight">Agent & LLM</h2>
              <p className="mt-1 max-w-3xl text-sm text-muted-foreground">
                Provider, Modell, lokale Ollama-Anbindung und Autonomie des Prometheus-Agenten.
              </p>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={isDirty ? "warning" : "outline"}>{isDirty ? "Geaendert" : "Aktuell"}</Badge>
            <Button type="button" variant="outline" size="sm" onClick={loadConfig} disabled={isSaving}>
              <RotateCcw className="h-4 w-4" />
              Verwerfen
            </Button>
            <Button type="button" size="sm" onClick={saveSettings} disabled={!isDirty || isSaving}>
              <Save className="h-4 w-4" />
              Speichern
            </Button>
          </div>
        </div>
      </div>

      {recommendation && provider === "ollama" && (
        <RecommendationCard
          data={recommendation}
          activeModel={model}
          onPick={(name) => setModel(name)}
          onPull={pullModelHandler}
          pullingModel={pullingModel}
        />
      )}

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Server className="h-4 w-4" />
            Provider & Modell
          </CardTitle>
          <CardDescription>Der Provider bestimmt, welches Backend fuer Agent-Antworten verwendet wird.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          <div className="grid gap-3 md:grid-cols-3">
            {providers.map((entry) => (
              <button
                key={entry.value}
                type="button"
                onClick={() => handleProviderChange(entry.value)}
                className={`rounded-md border p-3 text-left transition-colors ${
                  provider === entry.value ? "border-primary bg-primary/5" : "hover:bg-muted/50"
                }`}
              >
                <div className="font-medium">{entry.label}</div>
                <div className="text-xs text-muted-foreground">{entry.detail}</div>
              </button>
            ))}
          </div>

          <div className="grid gap-4 lg:grid-cols-2">
            <div className="space-y-2">
              <Label>Modell</Label>
              <Select value={model} onValueChange={setModel}>
                <SelectTrigger>
                  <SelectValue placeholder="Modell waehlen" />
                </SelectTrigger>
                <SelectContent>
                  {modelOptions.map((entry) => (
                    <SelectItem key={entry.value} value={entry.value}>
                      {entry.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {provider === "ollama" && (
              <div className="space-y-2">
                <Label>Ollama URL</Label>
                <div className="flex gap-2">
                  <Input value={ollamaUrl} onChange={(event) => setOllamaUrl(event.target.value)} />
                  <Button type="button" variant="outline" onClick={discoverOllamaModels} disabled={isDiscovering}>
                    <Wifi className="h-4 w-4" />
                    {isDiscovering ? "Teste..." : "Test"}
                  </Button>
                </div>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {provider !== "ollama" && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <KeyRound className="h-4 w-4" />
              API-Schluessel
            </CardTitle>
            <CardDescription>
              Gespeicherte Keys werden aus Sicherheitsgruenden nicht angezeigt. Ein neuer Wert ersetzt den vorhandenen Key.
            </CardDescription>
          </CardHeader>
          <CardContent className="grid gap-4 lg:grid-cols-2">
            {provider === "openai" && (
              <div className="space-y-3">
                <SecretInput
                  label="OpenAI API-Key"
                  configured={openaiConfigured}
                  value={openaiKey}
                  onChange={setOpenaiKey}
                  placeholder="sk-..."
                />
                <SecretActions
                  provider="openai"
                  configured={openaiConfigured}
                  busy={secretAction === "openai"}
                  onRotate={rotateProviderSecret}
                  onDelete={deleteProviderSecret}
                />
              </div>
            )}
            {provider === "anthropic" && (
              <div className="space-y-3">
                <SecretInput
                  label="Anthropic API-Key"
                  configured={anthropicConfigured}
                  value={anthropicKey}
                  onChange={setAnthropicKey}
                  placeholder="sk-ant-..."
                />
                <SecretActions
                  provider="anthropic"
                  configured={anthropicConfigured}
                  busy={secretAction === "anthropic"}
                  onRotate={rotateProviderSecret}
                  onDelete={deleteProviderSecret}
                />
              </div>
            )}
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <ShieldCheck className="h-4 w-4" />
            Autonomie
          </CardTitle>
          <CardDescription>Dieses Level steuert, wie weit der Agent ohne weitere Rueckfrage gehen darf.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between gap-3">
            <Label>Level {autonomyLevel}</Label>
            <Badge variant={autonomy.tone}>{autonomy.label}</Badge>
          </div>
          <Slider min={0} max={2} step={1} value={autonomyLevel} onValueChange={setAutonomyLevel} />
          <div className="grid grid-cols-3 text-xs text-muted-foreground">
            <span>Nur Lesen</span>
            <span className="text-center">Mit Freigabe</span>
            <span className="text-right">Vollautomatik</span>
          </div>
          <p className="text-sm text-muted-foreground">{autonomy.description}</p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <ShieldAlert className="h-4 w-4" />
            Approval-Regeln
          </CardTitle>
          <CardDescription>
            Risikostufen steuern, ob Agent-Aktionen vor Ausfuehrung manuell freigegeben werden muessen.
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
          <ApprovalToggle label="Low" checked={approvalLow} onChange={setApprovalLow} />
          <ApprovalToggle label="Medium" checked={approvalMedium} onChange={setApprovalMedium} />
          <ApprovalToggle label="High" checked={approvalHigh} onChange={setApprovalHigh} />
          <ApprovalToggle label="Critical" checked={approvalCritical} onChange={setApprovalCritical} />
          <ApprovalToggle label="Full-Auto Low Risk" checked={fullAutoLowRisk} onChange={setFullAutoLowRisk} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Wrench className="h-4 w-4" />
            Tool-Risiken
          </CardTitle>
          <CardDescription>
            Katalog aller Agent-Tools mit Risiko, benoetigter Permission und Dry-run-Faehigkeit.
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          {tools.length === 0 ? (
            <p className="text-sm text-muted-foreground">Tool-Katalog konnte nicht geladen werden.</p>
          ) : (
            tools.map((tool) => (
              <div key={tool.name} className="rounded-md border p-3">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="font-medium">{tool.name}</div>
                    <div className="mt-1 line-clamp-2 text-xs text-muted-foreground">{tool.description}</div>
                  </div>
                  <Badge variant={riskVariant(tool.security.risk)}>{tool.security.risk}</Badge>
                </div>
                <div className="mt-3 flex flex-wrap gap-2">
                  <Badge variant="outline">{tool.security.permission}</Badge>
                  {tool.read_only && <Badge variant="outline">read-only</Badge>}
                  {tool.supports_dry_run && <Badge variant="secondary">dry-run</Badge>}
                  {tool.security.requires_dry_run && <Badge variant="warning">preview required</Badge>}
                </div>
              </div>
            ))
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function ApprovalToggle({ label, checked, onChange }: { label: string; checked: boolean; onChange: (value: boolean) => void }) {
  return (
    <button
      type="button"
      onClick={() => onChange(!checked)}
      className={`rounded-md border p-3 text-left transition-colors ${checked ? "border-primary bg-primary/5" : "hover:bg-muted/50"}`}
    >
      <div className="font-medium">{label}</div>
      <div className="mt-1 text-xs text-muted-foreground">{checked ? "Approval erforderlich" : "Kein Approval"}</div>
    </button>
  );
}

function riskVariant(risk: string): "outline" | "warning" | "degraded" | "destructive" {
  if (risk === "critical") return "destructive";
  if (risk === "high") return "degraded";
  if (risk === "medium") return "warning";
  return "outline";
}

function SecretInput({
  label,
  configured,
  value,
  onChange,
  placeholder,
}: {
  label: string;
  configured: boolean;
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
}) {
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between gap-2">
        <Label>{label}</Label>
        <Badge variant={configured ? "success" : "outline"}>
          {configured ? (
            <span className="inline-flex items-center gap-1">
              <CheckCircle2 className="h-3.5 w-3.5" />
              Gesetzt
            </span>
          ) : (
            "Nicht gesetzt"
          )}
        </Badge>
      </div>
      <Input
        type="password"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={configured ? "Neuen Key eingeben" : placeholder}
        autoComplete="off"
      />
    </div>
  );
}

function SecretActions({
  provider,
  configured,
  busy,
  onRotate,
  onDelete,
}: {
  provider: "openai" | "anthropic";
  configured: boolean;
  busy: boolean;
  onRotate: (provider: "openai" | "anthropic") => void;
  onDelete: (provider: "openai" | "anthropic") => void;
}) {
  return (
    <div className="flex flex-wrap gap-2">
      <Button type="button" variant="outline" size="sm" onClick={() => onRotate(provider)} disabled={busy}>
        <RotateCcw className="h-4 w-4" />
        Rotieren
      </Button>
      <Button type="button" variant="destructive" size="sm" onClick={() => onDelete(provider)} disabled={busy || !configured}>
        <Trash2 className="h-4 w-4" />
        Loeschen
      </Button>
    </div>
  );
}

function toProvider(value: AgentConfig["llm_provider"]): Provider {
  if (value === "openai" || value === "anthropic" || value === "ollama") {
    return value;
  }
  return "ollama";
}

function formatSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const index = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  return `${(bytes / Math.pow(1024, index)).toFixed(1)} ${units[index]}`;
}

function snapshotWithoutSecrets(
  provider: Provider,
  model: string,
  ollamaUrl: string,
  autonomyLevel: number,
  approvalLow: boolean,
  approvalMedium: boolean,
  approvalHigh: boolean,
  approvalCritical: boolean,
  fullAutoLowRisk: boolean
) {
  return JSON.stringify({
    provider,
    model,
    ollamaUrl,
    openaiKey: "",
    anthropicKey: "",
    autonomyLevel,
    approvalLow,
    approvalMedium,
    approvalHigh,
    approvalCritical,
    fullAutoLowRisk,
  });
}

function RecommendationCard({
  data,
  activeModel,
  onPick,
  onPull,
  pullingModel,
}: {
  data: SystemRecommendation;
  activeModel: string;
  onPick: (name: string) => void;
  onPull: (name: string) => void;
  pullingModel: string;
}) {
  const tier = data.system.tier;
  const tone = tierTone[tier] ?? "outline";
  const ramFmt = data.system.total_ram_gb > 0 ? `${data.system.total_ram_gb.toFixed(1)} GB` : "n/v";

  return (
    <Card className="border-primary/30 bg-primary/5">
      <CardHeader>
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="flex items-start gap-3">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-primary/15">
              <Sparkles className="h-4 w-4 text-primary" />
            </div>
            <div>
              <CardTitle className="text-base">Empfehlungen für deine Hardware</CardTitle>
              <CardDescription>
                Erkannte Maschine, kuratierte Modell-Liste und Auto-Pull über Ollama.
              </CardDescription>
            </div>
          </div>
          <Badge variant={tone}>{data.system.tier_label}</Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-5">
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <SpecChip icon={MemoryStick} label="RAM" value={ramFmt} />
          <SpecChip icon={Cpu} label="CPU-Kerne" value={String(data.system.cpu_cores)} />
          <SpecChip
            icon={Zap}
            label="GPUs"
            value={
              data.system.gpus.length > 0
                ? data.system.gpus
                    .map((g) => `${g.name.replace(/^NVIDIA /, "")} (${g.vram_gb.toFixed(0)} GB)`)
                    .join(", ")
                : "keine"
            }
          />
          <SpecChip
            icon={HardDrive}
            label="Ollama"
            value={data.ollama_ready ? "verbunden" : "nicht erreichbar"}
            tone={data.ollama_ready ? "success" : "warning"}
          />
        </div>

        {data.system.notes && data.system.notes.length > 0 && (
          <div className="rounded-md border border-amber-300/40 bg-amber-300/10 px-3 py-2 text-xs text-amber-200">
            {data.system.notes.map((n, i) => (
              <div key={i}>• {n}</div>
            ))}
          </div>
        )}

        <div className="space-y-2">
          {data.models.map((m) => {
            const isActive = m.name === activeModel;
            const isPulling = pullingModel === m.name;
            return (
              <div
                key={m.name}
                className={`flex flex-wrap items-center gap-3 rounded-md border bg-background/60 p-3 transition-colors ${
                  isActive ? "border-primary" : "border-border"
                }`}
              >
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-mono text-sm font-medium">{m.name}</span>
                    {m.recommended && <Badge variant="success">Empfohlen</Badge>}
                    {m.pulled && <Badge variant="secondary">Installiert</Badge>}
                    {!m.tool_calling && <Badge variant="warning">kein Tool-Calling</Badge>}
                    {m.reasoning && <Badge variant="outline">Reasoning</Badge>}
                    <Badge variant="outline" className="text-xs">
                      {m.tier}
                    </Badge>
                    <span className="text-xs text-muted-foreground">
                      ~{m.size_gb < 1 ? `${(m.size_gb * 1024).toFixed(0)} MB` : `${m.size_gb.toFixed(1)} GB`}
                    </span>
                  </div>
                  <p className="mt-1 text-xs text-muted-foreground">{m.description}</p>
                </div>
                <div className="flex shrink-0 gap-2">
                  {!m.pulled && (
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => onPull(m.name)}
                      disabled={isPulling || !data.ollama_ready}
                    >
                      <Download className="h-4 w-4" />
                      {isPulling ? "Lädt…" : "Pull"}
                    </Button>
                  )}
                  <Button
                    type="button"
                    variant={isActive ? "default" : "outline"}
                    size="sm"
                    onClick={() => onPick(m.name)}
                    disabled={isActive}
                  >
                    {isActive ? "Aktiv" : "Auswählen"}
                  </Button>
                </div>
              </div>
            );
          })}
        </div>

        <p className="text-xs text-muted-foreground">
          Hinweis: Modelle ohne Tool-Calling können der Agent zwar antworten, aber keine Aktionen
          ausführen. Wenn die KI „dumm" wirkt, liegt es meistens an einem zu kleinen Modell oder
          einem ohne Tool-Calling.
        </p>
      </CardContent>
    </Card>
  );
}

function SpecChip({
  icon: Icon,
  label,
  value,
  tone = "default",
}: {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  value: string;
  tone?: "default" | "success" | "warning";
}) {
  const toneClass =
    tone === "success"
      ? "text-emerald-400"
      : tone === "warning"
      ? "text-amber-400"
      : "text-foreground";
  return (
    <div className="flex items-start gap-2 rounded-md border bg-background/60 p-2.5">
      <Icon className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
      <div className="min-w-0">
        <div className="text-[10px] font-medium uppercase tracking-wide text-muted-foreground">{label}</div>
        <div className={`truncate text-sm font-medium ${toneClass}`}>{value}</div>
      </div>
    </div>
  );
}

function getErrorMessage(error: unknown, fallback: string): string {
  if (error && typeof error === "object") {
    const candidate = error as { response?: { data?: { error?: string; message?: string } }; message?: string };
    return candidate.response?.data?.error || candidate.response?.data?.message || candidate.message || fallback;
  }
  return fallback;
}
