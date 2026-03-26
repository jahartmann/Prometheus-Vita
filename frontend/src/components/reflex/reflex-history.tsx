"use client";

import { ArrowLeft, Zap, Clock } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import type { ReflexRule } from "@/types/api";

const actionLabels: Record<string, string> = {
  restart_service: "Service neustarten",
  clear_cache: "Cache leeren",
  notify: "Benachrichtigung",
  run_command: "Befehl ausführen",
  start_vm: "VM starten",
  stop_vm: "VM stoppen",
};

const metricLabels: Record<string, string> = {
  cpu_usage: "CPU-Auslastung",
  memory_usage: "RAM-Auslastung",
  disk_usage: "Festplatten-Auslastung",
  load_avg: "Load Average",
};

interface ReflexHistoryProps {
  rule: ReflexRule;
  onBack: () => void;
}

export function ReflexHistory({ rule, onBack }: ReflexHistoryProps) {
  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" onClick={onBack}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{rule.name}</h1>
          <p className="text-muted-foreground">Ausführungsverlauf und Details</p>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Regeldetails</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span className="text-muted-foreground">Metrik</span>
              <p className="font-medium">{metricLabels[rule.trigger_metric] || rule.trigger_metric}</p>
            </div>
            <div>
              <span className="text-muted-foreground">Bedingung</span>
              <p className="font-medium">{rule.operator} {rule.threshold}</p>
            </div>
            <div>
              <span className="text-muted-foreground">Aktion</span>
              <p className="font-medium">{actionLabels[rule.action_type] || rule.action_type}</p>
            </div>
            <div>
              <span className="text-muted-foreground">Cooldown</span>
              <p className="font-medium">{rule.cooldown_seconds} Sekunden</p>
            </div>
            <div>
              <span className="text-muted-foreground">Status</span>
              <p>
                <Badge variant={rule.is_active ? "success" : "outline"}>
                  {rule.is_active ? "Aktiv" : "Inaktiv"}
                </Badge>
              </p>
            </div>
            <div>
              <span className="text-muted-foreground">Erstellt am</span>
              <p className="font-medium">{new Date(rule.created_at).toLocaleString("de-DE")}</p>
            </div>
            {rule.description && (
              <div className="col-span-2">
                <span className="text-muted-foreground">Beschreibung</span>
                <p className="font-medium">{rule.description}</p>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <Clock className="h-4 w-4" />
            Ausführungsverlauf
          </CardTitle>
        </CardHeader>
        <CardContent>
          {rule.trigger_count === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <Zap className="mb-3 h-10 w-10 text-muted-foreground" />
              <p className="font-medium">Noch keine Ausführungen</p>
              <p className="text-sm text-muted-foreground mt-1">
                Diese Regel wurde noch nicht ausgelöst.
              </p>
            </div>
          ) : (
            <div className="space-y-3">
              <div className="flex items-center justify-between text-sm text-muted-foreground border-b pb-2">
                <span>Insgesamt {rule.trigger_count}x ausgelöst</span>
                {rule.last_triggered_at && (
                  <span>
                    Zuletzt: {new Date(rule.last_triggered_at).toLocaleString("de-DE")}
                  </span>
                )}
              </div>
              <p className="text-sm text-muted-foreground">
                Detaillierte Ausführungslogs werden in einer zukünftigen Version verfügbar sein.
              </p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
