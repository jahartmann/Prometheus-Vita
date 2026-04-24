"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { AlertTriangle, BrainCircuit, CheckCircle2, GitCompare, RefreshCw, SearchCheck, ShieldAlert, TrendingUp } from "lucide-react";
import { agentConfigApi, operationsApi } from "@/lib/api";
import type { RCACandidate, RCAAnalyzeResponse } from "@/types/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";

function severityClass(severity: RCACandidate["severity"]) {
  if (severity === "critical") return "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300";
  if (severity === "warning") return "bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-300";
  return "bg-slate-100 text-slate-800 dark:bg-slate-900/40 dark:text-slate-300";
}

export default function RootCausePage() {
  const [analysis, setAnalysis] = useState<RCAAnalyzeResponse | null>(null);
  const [useLLM, setUseLLM] = useState(false);
  const [model, setModel] = useState("llama3");
  const [modelOptions, setModelOptions] = useState<string[]>(["llama3", "mistral", "codellama"]);
  const [isLoading, setIsLoading] = useState(true);

  const load = useCallback(async () => {
    setIsLoading(true);
    try {
      const result = await operationsApi.analyzeRCA({ prompt: "Root-Cause-Analyse", limit: 20, use_llm: useLLM, model }) as RCAAnalyzeResponse;
      setAnalysis(result);
    } catch {
      setAnalysis(null);
    }
    setIsLoading(false);
  }, [model, useLLM]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    agentConfigApi.getModels()
      .then((models) => {
        if (Array.isArray(models) && models.length > 0) {
          const names = models.map((entry) => typeof entry === "string" ? entry : entry?.name).filter(Boolean);
          if (names.length > 0) setModelOptions(names);
        }
      })
      .catch(() => undefined);
  }, []);

  const candidates = analysis?.candidates ?? [];
  const healthLabel = candidates.some((candidate) => candidate.severity === "critical") ? "Kritisch" : candidates.some((candidate) => candidate.severity === "warning") ? "Auffaellig" : "Ruhig";

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Root-Cause-Analyse</h1>
          <p className="text-sm text-muted-foreground">Korreliert Metriken, Security, Predictions, Backups, Migrationen und Changes zu konkreten Verdachtsmomenten.</p>
        </div>
        <Button variant="outline" size="sm" onClick={load} disabled={isLoading}>
          <RefreshCw className={cn("mr-2 h-4 w-4", isLoading && "animate-spin")} />
          Aktualisieren
        </Button>
      </div>

      <div className="grid gap-3 md:grid-cols-4">
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">Lage</CardTitle></CardHeader><CardContent className="text-2xl font-semibold">{healthLabel}</CardContent></Card>
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">Timeline</CardTitle></CardHeader><CardContent className="text-2xl font-semibold">{analysis?.timeline.length ?? 0}</CardContent></Card>
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">Modell</CardTitle></CardHeader><CardContent className="text-2xl font-semibold">{analysis?.model_used || "Regel"}</CardContent></Card>
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">Signals</CardTitle></CardHeader><CardContent className="text-2xl font-semibold">{candidates.length}</CardContent></Card>
      </div>

      <div className="grid gap-4 xl:grid-cols-[1fr,340px]">
        <div className="rounded-md border">
          <div className="flex items-center gap-2 border-b px-4 py-3 text-sm font-medium">
            <SearchCheck className="h-4 w-4" />
            Verdachtsliste
          </div>
          <div className="divide-y">
            {isLoading ? (
              <div className="px-4 py-10 text-center text-sm text-muted-foreground">Analyse laeuft...</div>
            ) : candidates.length === 0 ? (
              <div className="flex items-center gap-2 px-4 py-10 text-sm text-muted-foreground">
                <CheckCircle2 className="h-4 w-4 text-green-500" />
                Keine offenen Ursachen-Kandidaten gefunden.
              </div>
            ) : (
              candidates.map((candidate) => (
                <Link key={candidate.id} href={candidate.href} className="block px-4 py-4 transition-colors hover:bg-muted/50">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-medium">{candidate.title}</span>
                    <Badge variant="secondary" className={severityClass(candidate.severity)}>{candidate.severity}</Badge>
                    {candidate.node_id && <Badge variant="outline">{candidate.node_id}</Badge>}
                  </div>
                  <div className="mt-2 flex flex-wrap gap-2">
                    {candidate.evidence.slice(0, 3).map((entry, index) => (
                      <span key={`${candidate.id}-${index}`} className="rounded-md bg-muted px-2 py-1 text-xs text-muted-foreground">{entry}</span>
                    ))}
                  </div>
                  <p className="mt-2 text-sm text-muted-foreground">{candidate.recommendation}</p>
                </Link>
              ))
            )}
          </div>
        </div>

        <div className="space-y-3">
          <Card>
            <CardHeader className="pb-2"><CardTitle className="flex items-center gap-2 text-sm"><BrainCircuit className="h-4 w-4" /> Lokale Analyse</CardTitle></CardHeader>
            <CardContent className="space-y-3">
              <div className="flex items-center justify-between gap-3">
                <Label htmlFor="rca-llm">LLM-Zusammenfassung</Label>
                <Switch id="rca-llm" checked={useLLM} onCheckedChange={setUseLLM} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="rca-model">Modell</Label>
                <Input id="rca-model" value={model} onChange={(event) => setModel(event.target.value)} list="rca-models" disabled={!useLLM} />
                <datalist id="rca-models">
                  {modelOptions.map((entry) => <option key={entry} value={entry} />)}
                </datalist>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2"><CardTitle className="flex items-center gap-2 text-sm"><BrainCircuit className="h-4 w-4" /> Analyse-Logik</CardTitle></CardHeader>
            <CardContent className="space-y-2 text-sm text-muted-foreground">
              <p>1. Node-Erreichbarkeit und Kapazitaet zuerst.</p>
              <p>2. Danach Security-Events, Anomalien und Predictions.</p>
              <p>3. Fehlgeschlagene Jobs werden mit letzten Schreibaktionen korreliert.</p>
              {analysis?.summary && <p>{analysis.summary}</p>}
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2"><CardTitle className="flex items-center gap-2 text-sm"><GitCompare className="h-4 w-4" /> Naechste Schritte</CardTitle></CardHeader>
            <CardContent className="space-y-2 text-sm">
              <Link href="/flight-recorder" className="flex items-center gap-2 text-muted-foreground hover:text-foreground"><ShieldAlert className="h-4 w-4" /> Timeline gegenpruefen</Link>
              <Link href="/recommendations" className="flex items-center gap-2 text-muted-foreground hover:text-foreground"><TrendingUp className="h-4 w-4" /> Kapazitaetsprognosen pruefen</Link>
              <Link href="/logs" className="flex items-center gap-2 text-muted-foreground hover:text-foreground"><AlertTriangle className="h-4 w-4" /> Log-Anomalien untersuchen</Link>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
