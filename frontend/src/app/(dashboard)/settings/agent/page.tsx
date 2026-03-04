"use client";

import { useEffect, useState, useCallback } from "react";
import { agentConfigApi, userApi } from "@/lib/api";
import { useAuthStore } from "@/stores/auth-store";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
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
  ollama: [
    { value: "llama3", label: "Llama 3" },
    { value: "llama3:70b", label: "Llama 3 70B" },
    { value: "mistral", label: "Mistral" },
    { value: "codellama", label: "Code Llama" },
    { value: "gemma2", label: "Gemma 2" },
  ],
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

interface AgentConfig {
  llm_provider: string;
  llm_model: string;
  autonomy_level: number;
}

export default function AgentSettingsPage() {
  const user = useAuthStore((s) => s.user);
  const [config, setConfig] = useState<AgentConfig>({
    llm_provider: "ollama",
    llm_model: "llama3",
    autonomy_level: 1,
  });
  const [isSaving, setIsSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  const fetchConfig = useCallback(async () => {
    setIsLoading(true);
    try {
      const resp = await agentConfigApi.get();
      const data = resp.data?.data || resp.data;
      if (data) {
        setConfig({
          llm_provider: data.llm_provider || "ollama",
          llm_model: data.llm_model || "llama3",
          autonomy_level: data.autonomy_level ?? 1,
        });
      }
    } catch {
      // Fallback: load autonomy from user profile
      if (user?.id) {
        try {
          const r = await userApi.getById(user.id);
          const userData = r.data?.data || r.data;
          setConfig((prev) => ({
            ...prev,
            autonomy_level: userData?.autonomy_level ?? 1,
          }));
        } catch {
          // Ignore
        }
      }
    } finally {
      setIsLoading(false);
    }
  }, [user?.id]);

  useEffect(() => {
    fetchConfig();
  }, [fetchConfig]);

  const handleSave = async () => {
    setIsSaving(true);
    setSaved(false);
    try {
      await agentConfigApi.update(config);
      // Also update user autonomy level
      if (user?.id) {
        await userApi.update(user.id, { autonomy_level: config.autonomy_level });
      }
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    } catch {
      // Fallback: at least save autonomy level
      if (user?.id) {
        try {
          await userApi.update(user.id, { autonomy_level: config.autonomy_level });
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

  const handleProviderChange = (provider: string) => {
    const models = MODELS_BY_PROVIDER[provider] || [];
    setConfig({
      ...config,
      llm_provider: provider,
      llm_model: models[0]?.value || "",
    });
  };

  const availableModels = MODELS_BY_PROVIDER[config.llm_provider] || [];
  const currentAutonomy = autonomyLabels[config.autonomy_level] || autonomyLabels[1];

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

      {/* LLM Provider & Model */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">LLM-Konfiguration</CardTitle>
          <CardDescription>
            Waehle den KI-Provider und das Sprachmodell fuer den Agenten.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label>Provider</Label>
              <Select value={config.llm_provider} onValueChange={handleProviderChange}>
                <SelectTrigger>
                  <SelectValue placeholder="Provider waehlen" />
                </SelectTrigger>
                <SelectContent>
                  {LLM_PROVIDERS.map((p) => (
                    <SelectItem key={p.value} value={p.value}>
                      {p.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>Modell</Label>
              <Select
                value={config.llm_model}
                onValueChange={(model) => setConfig({ ...config, llm_model: model })}
              >
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
          </div>
        </CardContent>
      </Card>

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
              <Label>Level: {config.autonomy_level}</Label>
              <Badge variant={config.autonomy_level === 2 ? "destructive" : "default"}>
                {currentAutonomy.label}
              </Badge>
            </div>
            <Slider
              min={0}
              max={2}
              step={1}
              value={config.autonomy_level}
              onValueChange={(val) => setConfig({ ...config, autonomy_level: val })}
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
