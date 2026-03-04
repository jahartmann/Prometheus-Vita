"use client";

import type { RecoveryRunbook, RunbookStep } from "@/types/api";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { CheckCircle, Terminal, Hand } from "lucide-react";

interface RunbookViewerProps {
  runbook: RecoveryRunbook;
}

function parseSteps(steps: unknown): RunbookStep[] {
  if (Array.isArray(steps)) return steps;
  if (typeof steps === "string") {
    try {
      return JSON.parse(steps);
    } catch {
      return [];
    }
  }
  return [];
}

export function RunbookViewer({ runbook }: RunbookViewerProps) {
  const steps = parseSteps(runbook.steps);

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg">{runbook.title}</CardTitle>
          <Badge variant="outline">{runbook.scenario}</Badge>
        </div>
        <p className="text-xs text-muted-foreground">
          Erstellt: {new Date(runbook.generated_at).toLocaleString("de-DE")}
        </p>
      </CardHeader>
      <CardContent>
        <ol className="space-y-4">
          {steps.map((step, index) => (
            <li key={index} className="flex gap-3">
              <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-medium">
                {index + 1}
              </div>
              <div className="space-y-1 flex-1">
                <div className="flex items-center gap-2">
                  <span className="font-medium text-sm">{step.title}</span>
                  {step.is_manual ? (
                    <Badge variant="secondary" className="text-xs">
                      <Hand className="mr-1 h-3 w-3" />
                      Manuell
                    </Badge>
                  ) : (
                    <Badge variant="outline" className="text-xs">
                      <Terminal className="mr-1 h-3 w-3" />
                      Automatisch
                    </Badge>
                  )}
                </div>
                <p className="text-sm text-muted-foreground">{step.description}</p>
                {step.command && (
                  <pre className="mt-1 rounded bg-muted p-2 text-xs font-mono overflow-x-auto">
                    {step.command}
                  </pre>
                )}
                {step.expected_output && (
                  <div className="mt-1 flex items-center gap-1 text-xs text-muted-foreground">
                    <CheckCircle className="h-3 w-3" />
                    Erwartet: {step.expected_output}
                  </div>
                )}
              </div>
            </li>
          ))}
        </ol>
      </CardContent>
    </Card>
  );
}
