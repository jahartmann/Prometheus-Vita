"use client";

import { useEffect, useState } from "react";
import { Zap, Plus, Pencil, Trash2, History, RefreshCw } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";
import { useReflexStore } from "@/stores/reflex-store";
import { useNodeStore } from "@/stores/node-store";
import type { ReflexRule } from "@/types/api";
import { ReflexRuleDialog } from "@/components/notifications/reflex-rule-dialog";
import { ReflexHistory } from "./reflex-history";

const actionLabels: Record<string, string> = {
  restart_service: "Service neustarten",
  clear_cache: "Cache leeren",
  notify: "Benachrichtigung",
  run_command: "Befehl ausfuehren",
  start_vm: "VM starten",
  stop_vm: "VM stoppen",
};

const metricLabels: Record<string, string> = {
  cpu_usage: "CPU",
  memory_usage: "RAM",
  disk_usage: "Festplatte",
  load_avg: "Load Average",
};

export function ReflexList() {
  const { rules, isLoading, fetchRules, toggleRule, deleteRule } = useReflexStore();
  const { nodes, fetchNodes } = useNodeStore();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editRule, setEditRule] = useState<ReflexRule | null>(null);
  const [historyRule, setHistoryRule] = useState<ReflexRule | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);

  useEffect(() => {
    fetchRules();
    fetchNodes();
  }, [fetchRules, fetchNodes]);

  const getNodeName = (nodeId?: string) => {
    if (!nodeId) return "Alle Nodes";
    const node = nodes.find((n) => n.id === nodeId);
    return node?.name ?? nodeId.slice(0, 8);
  };

  const handleEdit = (rule: ReflexRule) => {
    setEditRule(rule);
    setDialogOpen(true);
  };

  const handleCreate = () => {
    setEditRule(null);
    setDialogOpen(true);
  };

  const handleDelete = async (id: string) => {
    await deleteRule(id);
    setDeleteConfirm(null);
  };

  if (historyRule) {
    return (
      <ReflexHistory
        rule={historyRule}
        onBack={() => setHistoryRule(null)}
      />
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Reflex-Regeln</h1>
          <p className="text-muted-foreground">
            Automatische Reaktionen auf Metriken und Events.
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={() => fetchRules()}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Aktualisieren
          </Button>
          <Button size="sm" onClick={handleCreate}>
            <Plus className="mr-2 h-4 w-4" />
            Neue Regel
          </Button>
        </div>
      </div>

      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-24 w-full" />
          ))}
        </div>
      ) : rules.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-16">
            <Zap className="mb-4 h-12 w-12 text-muted-foreground" />
            <p className="text-lg font-medium">Keine Reflex-Regeln vorhanden</p>
            <p className="mt-1 text-sm text-muted-foreground">
              Erstellen Sie Ihre erste Regel, um automatisch auf Metriken zu reagieren.
            </p>
            <Button className="mt-4" onClick={handleCreate}>
              <Plus className="mr-2 h-4 w-4" />
              Erste Regel erstellen
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {rules.map((rule) => (
            <Card key={rule.id} className="transition-all hover:shadow-md">
              <CardContent className="p-0">
                <div className="flex items-stretch">
                  <div
                    className={`w-1 rounded-l-lg shrink-0 ${
                      rule.is_active ? "bg-green-500" : "bg-gray-300"
                    }`}
                  />
                  <div className="flex-1 p-4">
                    <div className="flex items-start justify-between">
                      <div className="space-y-1.5">
                        <div className="flex items-center gap-2 flex-wrap">
                          <Zap className={`h-4 w-4 shrink-0 ${rule.is_active ? "text-yellow-500" : "text-muted-foreground"}`} />
                          <span className="font-semibold">{rule.name}</span>
                          <Badge variant={rule.is_active ? "success" : "outline"}>
                            {rule.is_active ? "Aktiv" : "Inaktiv"}
                          </Badge>
                          <Badge variant="outline">
                            {actionLabels[rule.action_type] || rule.action_type}
                          </Badge>
                          {rule.node_id && (
                            <Badge variant="outline" className="text-xs">
                              {getNodeName(rule.node_id)}
                            </Badge>
                          )}
                        </div>
                        {rule.description && (
                          <p className="text-sm text-muted-foreground">{rule.description}</p>
                        )}
                        <div className="flex items-center gap-3 text-sm text-muted-foreground">
                          <span>
                            {metricLabels[rule.trigger_metric] || rule.trigger_metric}{" "}
                            {rule.operator} {rule.threshold}
                          </span>
                          <span>Cooldown: {rule.cooldown_seconds}s</span>
                          {rule.trigger_count > 0 && (
                            <span>{rule.trigger_count}x ausgeloest</span>
                          )}
                          {rule.last_triggered_at && (
                            <span>
                              Zuletzt: {new Date(rule.last_triggered_at).toLocaleString("de-DE")}
                            </span>
                          )}
                        </div>
                      </div>
                      <div className="flex items-center gap-2 shrink-0 ml-4">
                        <Switch
                          checked={rule.is_active}
                          onCheckedChange={(checked) => toggleRule(rule.id, checked)}
                        />
                        <Button
                          variant="ghost"
                          size="icon"
                          title="Verlauf"
                          onClick={() => setHistoryRule(rule)}
                        >
                          <History className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          title="Bearbeiten"
                          onClick={() => handleEdit(rule)}
                        >
                          <Pencil className="h-4 w-4" />
                        </Button>
                        {deleteConfirm === rule.id ? (
                          <div className="flex items-center gap-1">
                            <Button
                              variant="destructive"
                              size="sm"
                              className="text-xs h-8"
                              onClick={() => handleDelete(rule.id)}
                            >
                              Ja
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="text-xs h-8"
                              onClick={() => setDeleteConfirm(null)}
                            >
                              Nein
                            </Button>
                          </div>
                        ) : (
                          <Button
                            variant="ghost"
                            size="icon"
                            title="Loeschen"
                            onClick={() => setDeleteConfirm(rule.id)}
                          >
                            <Trash2 className="h-4 w-4 text-destructive" />
                          </Button>
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <ReflexRuleDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        onSuccess={() => fetchRules()}
        rule={editRule}
        nodes={nodes}
      />
    </div>
  );
}
