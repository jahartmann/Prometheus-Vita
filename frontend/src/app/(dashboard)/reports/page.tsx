"use client";

import { useCallback, useEffect, useState } from "react";
import { CalendarClock, FileBarChart, Filter, RefreshCw, Wand2 } from "lucide-react";
import { agentConfigApi, logAnalysisApi, operationsApi } from "@/lib/api";
import type { LogAnalysis, OperationsReportResponse } from "@/types/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";

type DomainFilter = "all" | "security" | "capacity" | "operations";

export default function ReportsPage() {
  const [prompt, setPrompt] = useState("Erstelle einen kompakten Betriebsbericht fuer die letzten kritischen Ereignisse.");
  const [domain, setDomain] = useState<DomainFilter>("all");
  const [severity, setSeverity] = useState("all");
  const [query, setQuery] = useState("");
  const [useLLM, setUseLLM] = useState(false);
  const [model, setModel] = useState("llama3");
  const [modelOptions, setModelOptions] = useState<string[]>(["llama3", "mistral", "codellama"]);
  const [report, setReport] = useState<OperationsReportResponse | null>(null);
  const [history, setHistory] = useState<LogAnalysis[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  const load = useCallback(async () => {
    setIsLoading(true);
    try {
      const result = await operationsApi.generateReport({
        prompt,
        domain,
        severity,
        query,
        use_llm: useLLM,
        model,
      }) as OperationsReportResponse;
      setReport(result);
    } catch {
      setReport(null);
    }
    setIsLoading(false);
  }, [domain, model, prompt, query, severity, useLLM]);

  const loadHistory = useCallback(async () => {
    try {
      const response = await logAnalysisApi.getAnalyses({ limit: 8 });
      const items = Array.isArray(response.data) ? response.data : [];
      setHistory(items as LogAnalysis[]);
    } catch {
      setHistory([]);
    }
  }, []);

  useEffect(() => {
    load();
    loadHistory();
  }, [load, loadHistory]);

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

  const counts = report?.counts ?? {};

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Reports</h1>
          <p className="text-sm text-muted-foreground">Promptbare Betriebsberichte mit Dashboard-Filtern aus Security, Kapazitaet und Operations.</p>
        </div>
        <Button variant="outline" size="sm" onClick={load} disabled={isLoading}>
          <RefreshCw className={cn("mr-2 h-4 w-4", isLoading && "animate-spin")} />
          Aktualisieren
        </Button>
      </div>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="flex items-center gap-2 text-sm">
            <CalendarClock className="h-4 w-4" />
            Report-Historie
          </CardTitle>
        </CardHeader>
        <CardContent>
          {history.length === 0 ? (
            <p className="text-sm text-muted-foreground">Noch keine geplanten oder manuellen Log-Reports vorhanden.</p>
          ) : (
            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
              {history.map((entry) => {
                const summary = readReportSummary(entry.report_json);
                return (
                  <div key={entry.id} className="rounded-md border p-3">
                    <div className="flex items-center justify-between gap-2">
                      <Badge variant={entry.schedule_id ? "secondary" : "outline"}>{entry.schedule_id ? "Geplant" : "Manuell"}</Badge>
                      <span className="text-xs text-muted-foreground">{formatDate(entry.created_at)}</span>
                    </div>
                    <p className="mt-2 line-clamp-3 text-sm">{summary}</p>
                    <div className="mt-3 flex flex-wrap gap-2">
                      <Badge variant="outline">{entry.node_ids.length} Nodes</Badge>
                      {entry.model_used && <Badge variant="outline">{entry.model_used}</Badge>}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>

      <div className="grid gap-4 xl:grid-cols-[380px,1fr]">
        <div className="space-y-4">
          <Card>
            <CardHeader className="pb-2"><CardTitle className="flex items-center gap-2 text-sm"><Wand2 className="h-4 w-4" /> Report-Prompt</CardTitle></CardHeader>
            <CardContent className="space-y-3">
              <textarea
                value={prompt}
                onChange={(event) => setPrompt(event.target.value)}
                className="min-h-28 w-full rounded-md border bg-background px-3 py-2 text-sm outline-none ring-offset-background focus-visible:ring-2 focus-visible:ring-ring"
                aria-label="Report-Prompt"
              />
              <div className="space-y-2">
                <Label htmlFor="report-query">Suche</Label>
                <Input id="report-query" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Node, VM, Metric, Pfad..." />
              </div>
              <div className="flex items-center justify-between gap-3 rounded-md border px-3 py-2">
                <Label htmlFor="report-llm">LLM-Zusammenfassung</Label>
                <Switch id="report-llm" checked={useLLM} onCheckedChange={setUseLLM} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="report-model">Modell</Label>
                <Input id="report-model" value={model} onChange={(event) => setModel(event.target.value)} list="report-models" disabled={!useLLM} />
                <datalist id="report-models">
                  {modelOptions.map((entry) => <option key={entry} value={entry} />)}
                </datalist>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2"><CardTitle className="flex items-center gap-2 text-sm"><Filter className="h-4 w-4" /> Dashboard-Filter</CardTitle></CardHeader>
            <CardContent className="space-y-3">
              <div className="space-y-2">
                <Label>Domaene</Label>
                <Select value={domain} onValueChange={(value) => setDomain(value as DomainFilter)}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">Alle</SelectItem>
                    <SelectItem value="security">Security</SelectItem>
                    <SelectItem value="capacity">Kapazitaet</SelectItem>
                    <SelectItem value="operations">Operations</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>Severity</Label>
                <Select value={severity} onValueChange={setSeverity}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">Alle</SelectItem>
                    <SelectItem value="info">Info</SelectItem>
                    <SelectItem value="warning">Warning</SelectItem>
                    <SelectItem value="critical">Critical</SelectItem>
                    <SelectItem value="emergency">Emergency</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="flex flex-wrap gap-2 pt-1">
                <Badge variant="outline">{counts.security ?? 0} Security</Badge>
                <Badge variant="outline">{counts.capacity ?? 0} Kapazitaet</Badge>
                <Badge variant="outline">{counts.operations ?? 0} Operations</Badge>
                <Badge variant="outline">{counts.timeline ?? 0} Timeline</Badge>
              </div>
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader className="pb-2"><CardTitle className="flex items-center gap-2 text-sm"><FileBarChart className="h-4 w-4" /> Generierter Bericht</CardTitle></CardHeader>
          <CardContent>
            <pre className="max-h-[680px] overflow-auto whitespace-pre-wrap rounded-md border bg-muted/30 p-4 text-sm leading-6">
              {isLoading ? "Daten werden geladen..." : report?.text ?? "Kein Bericht verfuegbar."}
            </pre>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function formatDate(value?: string) {
  if (!value) return "-";
  return new Date(value).toLocaleString("de-DE", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" });
}

function readReportSummary(report: unknown): string {
  if (!report) return "Kein Summary verfuegbar.";
  if (typeof report === "string") {
    try {
      const parsed = JSON.parse(report) as { summary?: string };
      return parsed.summary || report;
    } catch {
      return report;
    }
  }
  if (typeof report === "object" && "summary" in report) {
    return String((report as { summary?: unknown }).summary || "Kein Summary verfuegbar.");
  }
  return "Kein Summary verfuegbar.";
}
