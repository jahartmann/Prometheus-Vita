"use client";

import { useEffect, useState, useCallback } from "react";
import { agentConfigApi, userApi } from "@/lib/api";
import { useAuthStore } from "@/stores/auth-store";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Slider } from "@/components/ui/slider";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const LLM_PROVIDERS = [
  { value: "ollama", label: "Ollama (lokal)" },
  { value: "openai", label: "OpenAI" },
  { value: "anthropic", label: "Anthropic" },
];

const MODELS_BY_PROVIDER: Record<string, { value: string; label: string }[]> = {
  openai: [
    { value: "gpt-4o", label: "GPT-4o" },
    { value: "gpt-4o-mini", label: "GPT-4o Mini" },
    { value: "gpt-4-turbo", label: "GPT-4 Turbo" },
  ],
  anthropic: [
    { value: "claude-sonnet-4-20250514", label: "Claude Sonnet 4" },
    { value: "claude-haiku-4-5-20251001", label: "Claude Haiku 4.5" },
  ],
};

const autonomyLabels: Record<number, { label: string; description: string }> = {
  0: {
    label: "Nur Lesen",
    description: "Der Agent darf nur lesende Tools verwenden. Schreibende Aktionen werden blockiert.",
  },
  1: {
    label: "Mit Bestaetigung",
    description: "Schreibende Aktionen erfordern eine manuelle Genehmigung vor der Ausfuehrung.",
  },
  2: {
    label: "Voll-Automatisch",
    description: "Der Agent fuehrt alle Aktionen sofort aus, ohne Bestaetigung.",
  },
};

interface OllamaModel {
  name: string;
  size: number;
  modified_at: string;
}

export default function AgentSettingsPage() {
  const user = useAuthStore((s) => s.user);
  const [provider, setProvider] = useState("ollama");
  const [model, setModel] = useState("llama3");
  const [ollamaUrl, setOllamaUrl] = useState("http://localhost:11434");
  const [openaiKey, setOpenaiKey] = useState("");
  const [anthropicKey, setAnthropicKey] = useState("");
  const [autonomyLevel, setAutonomyLevel] = useState(1);
  const [isSaving, setIsSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  // Ollama model discovery
  const [ollamaModels, setOllamaModels] = useState<OllamaModel[]>([]);
  const [isDiscovering, setIsDiscovering] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState<"idle" | "success" | "error">("idle");
  const [connectionError, setConnectionError] = useState("");

  const fetchConfig = useCallback(async () => {
    setIsLoading(true);
    try {
      const resp = await agentConfigApi.get();
      const data = resp.data?.data || resp.data || resp;
      if (data && typeof data === "object") {
        const cfg = data as Record<string, string>;
        setProvider(cfg.llm_provider || "ollama");
        setModel(cfg.llm_model || "llama3");
        setOllamaUrl(cfg.ollama_url || "http://localhost:11434");
        setOpenaiKey(cfg.openai_key || "");
        setAnthropicKey(cfg.anthropic_key || "");
      }
    } catch {
      // Fallback: load autonomy from user profile
    }
    // Load autonomy from user profile
    if (user?.id) {
      try {
        const r = await userApi.getById(user.id);
        const userData = r.data?.data || r.data;
        if (userData?.autonomy_level !== undefined) {
          setAutonomyLevel(userData.autonomy_level);
        }
      } catch {
        // Ignore
      }
    }
    setIsLoading(false);
  }, [user?.id]);

  useEffect(() => {
    fetchConfig();
  }, [fetchConfig]);

  const handleTestConnection = async () => {
    setIsDiscovering(true);
    setConnectionStatus("idle");
    setConnectionError("");
    try {
      const resp = await agentConfigApi.getModels();
      const models = resp.data?.data || resp.data || resp;
      if (Array.isArray(models)) {
        setOllamaModels(models);
        setConnectionStatus("success");
      } else {
        setOllamaModels([]);
        setConnectionStatus("success");
      }
    } catch (err: unknown) {
      setConnectionStatus("error");
      const msg = err instanceof Error ? err.message : "Verbindung fehlgeschlagen";
      setConnectionError(msg);
      setOllamaModels([]);
    } finally {
      setIsDiscovering(false);
    }
  };

  const handleSave = async () => {
    setIsSaving(true);
    setSaved(false);
    try {
      await agentConfigApi.update({
        llm_provider: provider,
        llm_model: model,
        ollama_url: ollamaUrl,
        openai_key: openaiKey,
        anthropic_key: anthropicKey,
      });
      // Also update user autonomy level
      if (user?.id) {
        await userApi.update(user.id, { autonomy_level: autonomyLevel });
      }
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    } catch {
      // Fallback: at least save autonomy level
      if (user?.id) {
        try {
          await userApi.update(user.id, { autonomy_level: autonomyLevel });
          setSaved(true);
          setTimeout(() => setSaved(false), 2000);
        } catch {
          // Ignore
        }
      }
    } finally {
      setIsSaving(false);
    }
  };

  const handleProviderChange = (newProvider: string) => {
    setProvider(newProvider);
    if (newProvider === "ollama") {
      const firstOllama = ollamaModels[0]?.name || "llama3";
      setModel(firstOllama);
    } else {
      const models = MODELS_BY_PROVIDER[newProvider] || [];
      setModel(models[0]?.value || "");
    }
  };

  const currentAutonomy = autonomyLabels[autonomyLevel] || autonomyLabels[1];

  // Build model list depending on provider
  const getModelOptions = () => {
    if (provider === "ollama") {
      if (ollamaModels.length > 0) {
        return ollamaModels.map((m) => ({
          value: m.name,
          label: `${m.name} (${formatSize(m.size)})`,
        }));
      }
      // Fallback static list
      return [
        { value: "llama3", label: "Llama 3" },
        { value: "llama3:70b", label: "Llama 3 70B" },
        { value: "mistral", label: "Mistral" },
        { value: "codellama", label: "Code Llama" },
        { value: "gemma2", label: "Gemma 2" },
      ];
    }
    return MODELS_BY_PROVIDER[provider] || [];
  };

  const availableModels = getModelOptions();

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div>
          <h2 className="text-xl font-bold">Agent-Einstellungen</h2>
          <p className="text-sm text-muted-foreground mt-1">Laden...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-bold">Agent-Einstellungen</h2>
        <p className="text-sm text-muted-foreground mt-1">
          Konfiguriere den KI-Agenten: LLM-Provider, Modell und Autonomie-Level.
        </p>
      </div>

      {/* Active Provider */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Aktiver Provider</CardTitle>
          <CardDescription>
            Waehle den KI-Provider fuer den Agenten.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex gap-3">
            {LLM_PROVIDERS.map((p) => (
              <Button
                key={p.value}
                variant={provider === p.value ? "default" : "outline"}
                onClick={() => handleProviderChange(p.value)}
                className="flex-1"
              >
                {p.label}
              </Button>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Ollama Section */}
      {provider === "ollama" && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Ollama-Konfiguration</CardTitle>
            <CardDescription>
              Verbinde dich mit deiner lokalen oder entfernten Ollama-Instanz.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label>Ollama URL</Label>
              <div className="flex gap-2">
                <Input
                  value={ollamaUrl}
                  onChange={(e) => setOllamaUrl(e.target.value)}
                  placeholder="http://localhost:11434"
                  className="flex-1"
                />
                <Button
                  variant="outline"
                  onClick={handleTestConnection}
                  disabled={isDiscovering}
                >
                  {isDiscovering ? "Teste..." : "Verbindung testen"}
                </Button>
              </div>
              {connectionStatus === "success" && (
                <p className="text-sm text-green-600">
                  Verbindung erfolgreich. {ollamaModels.length} Modell(e) gefunden.
                </p>
              )}
              {connectionStatus === "error" && (
                <p className="text-sm text-red-600">
                  Verbindung fehlgeschlagen: {connectionError}
                </p>
              )}
            </div>
            <div className="space-y-2">
              <Label>Modell</Label>
              <Select value={model} onValueChange={setModel}>
                <SelectTrigger>
                  <SelectValue placeholder="Modell waehlen" />
                </SelectTrigger>
                <SelectContent>
                  {availableModels.map((m) => (
                    <SelectItem key={m.value} value={m.value}>
                      {m.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>
      )}

      {/* OpenAI Section */}
      {provider === "openai" && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">OpenAI-Konfiguration</CardTitle>
            <CardDescription>
              Gib deinen OpenAI API-Key ein, um GPT-Modelle zu verwenden.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label>API Key</Label>
              <Input
                type="password"
                value={openaiKey}
                onChange={(e) => setOpenaiKey(e.target.value)}
                placeholder="sk-..."
              />
            </div>
            <div className="space-y-2">
              <Label>Modell</Label>
              <Select value={model} onValueChange={setModel}>
                <SelectTrigger>
                  <SelectValue placeholder="Modell waehlen" />
                </SelectTrigger>
                <SelectContent>
                  {availableModels.map((m) => (
                    <SelectItem key={m.value} value={m.value}>
                      {m.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Anthropic Section */}
      {provider === "anthropic" && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Anthropic-Konfiguration</CardTitle>
            <CardDescription>
              Gib deinen Anthropic API-Key ein, um Claude-Modelle zu verwenden.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label>API Key</Label>
              <Input
                type="password"
                value={anthropicKey}
                onChange={(e) => setAnthropicKey(e.target.value)}
                placeholder="sk-ant-..."
              />
            </div>
            <div className="space-y-2">
              <Label>Modell</Label>
              <Select value={model} onValueChange={setModel}>
                <SelectTrigger>
                  <SelectValue placeholder="Modell waehlen" />
                </SelectTrigger>
                <SelectContent>
                  {availableModels.map((m) => (
                    <SelectItem key={m.value} value={m.value}>
                      {m.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Autonomy Level */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Autonomie-Level</CardTitle>
          <CardDescription>
            Bestimme, wie selbststaendig der Agent handeln darf.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <Label>Level: {autonomyLevel}</Label>
              <Badge variant={autonomyLevel === 2 ? "destructive" : "default"}>
                {currentAutonomy.label}
              </Badge>
            </div>
            <Slider
              min={0}
              max={2}
              step={1}
              value={autonomyLevel}
              onValueChange={(val) => setAutonomyLevel(val)}
            />
            <div className="flex justify-between text-xs text-muted-foreground">
              <span>Nur Lesen</span>
              <span>Mit Bestaetigung</span>
              <span>Voll-Automatisch</span>
            </div>
          </div>
          <p className="text-sm text-muted-foreground">
            {currentAutonomy.description}
          </p>
        </CardContent>
      </Card>

      {/* Save */}
      <div className="flex items-center gap-3">
        <Button onClick={handleSave} disabled={isSaving}>
          {isSaving ? "Speichere..." : "Einstellungen speichern"}
        </Button>
        {saved && (
          <span className="text-sm text-green-600">Erfolgreich gespeichert.</span>
        )}
      </div>
    </div>
  );
}

function formatSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  const size = (bytes / Math.pow(1024, i)).toFixed(1);
  return `${size} ${units[i]}`;
}
