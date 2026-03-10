"use client";

import { useState, useEffect } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import { reflexApi } from "@/lib/api";
import type { ReflexRule, ReflexActionType, Node } from "@/types/api";

interface ReflexRuleDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
  rule?: ReflexRule | null;
  nodes: Node[];
}

const metrics = [
  { value: "cpu_usage", label: "CPU-Auslastung (%)" },
  { value: "memory_usage", label: "RAM-Auslastung (%)" },
  { value: "disk_usage", label: "Festplatten-Auslastung (%)" },
  { value: "load_avg", label: "Load Average" },
];

const operators = [">", ">=", "<", "<=", "==", "!="];

const actionTypes: { value: ReflexActionType; label: string }[] = [
  { value: "restart_service", label: "Service neustarten" },
  { value: "clear_cache", label: "Cache leeren" },
  { value: "notify", label: "Benachrichtigung senden" },
  { value: "run_command", label: "Befehl ausfuehren" },
  { value: "start_vm", label: "VM starten" },
  { value: "stop_vm", label: "VM stoppen" },
  { value: "scale_up", label: "Ressourcen hochskalieren" },
  { value: "scale_down", label: "Ressourcen herunterskalieren" },
  { value: "snapshot", label: "Snapshot erstellen" },
  { value: "ai_analyze", label: "KI-Analyse starten" },
];

const selectClass =
  "flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring";

export function ReflexRuleDialog({
  open,
  onOpenChange,
  onSuccess,
  rule,
  nodes,
}: ReflexRuleDialogProps) {
  const isEdit = !!rule;
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [triggerMetric, setTriggerMetric] = useState("cpu_usage");
  const [operator, setOperator] = useState(">");
  const [threshold, setThreshold] = useState("80");
  const [actionType, setActionType] = useState<ReflexActionType>("notify");
  const [cooldownSeconds, setCooldownSeconds] = useState("300");
  const [nodeId, setNodeId] = useState<string>("");
  // Action config fields
  const [serviceName, setServiceName] = useState("");
  const [command, setCommand] = useState("");
  const [vmid, setVmid] = useState("");
  const [vmType, setVmType] = useState("qemu");
  // Time scheduling fields
  const [scheduleType, setScheduleType] = useState("always");
  const [timeStart, setTimeStart] = useState("08:00");
  const [timeEnd, setTimeEnd] = useState("18:00");
  const [timeDays, setTimeDays] = useState<number[]>([]);
  const [cronExpr, setCronExpr] = useState("");
  // AI and priority fields
  const [aiEnabled, setAiEnabled] = useState(false);
  const [priority, setPriority] = useState("0");
  const [tags, setTags] = useState("");

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (rule) {
      setName(rule.name);
      setDescription(rule.description || "");
      setTriggerMetric(rule.trigger_metric);
      setOperator(rule.operator);
      setThreshold(String(rule.threshold));
      setActionType(rule.action_type);
      setCooldownSeconds(String(rule.cooldown_seconds));
      setNodeId(rule.node_id || "");
      const config = rule.action_config || {};
      setServiceName((config.service_name as string) || "");
      setCommand((config.command as string) || "");
      setVmid(config.vmid ? String(config.vmid) : "");
      setVmType((config.vm_type as string) || "qemu");
      // Time scheduling
      setScheduleType(rule.schedule_type || "always");
      setTimeStart(rule.time_window_start || "08:00");
      setTimeEnd(rule.time_window_end || "18:00");
      setTimeDays(rule.time_window_days || []);
      setCronExpr(rule.schedule_cron || "");
      // AI and priority
      setAiEnabled(rule.ai_enabled || false);
      setPriority(String(rule.priority ?? 0));
      setTags((rule.tags || []).join(", "));
    } else {
      setName("");
      setDescription("");
      setTriggerMetric("cpu_usage");
      setOperator(">");
      setThreshold("80");
      setActionType("notify");
      setCooldownSeconds("300");
      setNodeId("");
      setServiceName("");
      setCommand("");
      setVmid("");
      setVmType("qemu");
      setScheduleType("always");
      setTimeStart("08:00");
      setTimeEnd("18:00");
      setTimeDays([]);
      setCronExpr("");
      setAiEnabled(false);
      setPriority("0");
      setTags("");
    }
    setError(null);
  }, [rule, open]);

  const buildActionConfig = (): Record<string, unknown> => {
    switch (actionType) {
      case "restart_service":
        return { service_name: serviceName };
      case "run_command":
        return { command };
      case "start_vm":
      case "stop_vm":
      case "snapshot":
        return { vmid: parseInt(vmid) || 0, vm_type: vmType };
      default:
        return {};
    }
  };

  const parseTags = (): string[] => {
    if (!tags.trim()) return [];
    return tags.split(",").map((t) => t.trim()).filter(Boolean);
  };

  const handleSubmit = async () => {
    setLoading(true);
    setError(null);
    try {
      const actionConfig = buildActionConfig();
      const commonData = {
        name,
        description: description || undefined,
        trigger_metric: triggerMetric,
        operator,
        threshold: parseFloat(threshold),
        action_type: actionType,
        action_config: actionConfig,
        cooldown_seconds: parseInt(cooldownSeconds) || 300,
        node_id: nodeId || undefined,
        schedule_type: scheduleType,
        schedule_cron: scheduleType === "cron" ? cronExpr : undefined,
        time_window_start: scheduleType === "time_window" ? timeStart : undefined,
        time_window_end: scheduleType === "time_window" ? timeEnd : undefined,
        time_window_days: scheduleType === "time_window" ? timeDays : undefined,
        ai_enabled: aiEnabled,
        priority: parseInt(priority) || 0,
        tags: parseTags(),
      };
      if (isEdit && rule) {
        await reflexApi.update(rule.id, commonData);
      } else {
        await reflexApi.create(commonData);
      }
      onSuccess();
      onOpenChange(false);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Fehler beim Speichern");
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {isEdit ? "Reflex-Regel bearbeiten" : "Neue Reflex-Regel"}
          </DialogTitle>
          <DialogDescription>
            {isEdit
              ? "Reflex-Regel Konfiguration aktualisieren."
              : "Definieren Sie eine automatische Reaktion auf Metriken."}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 max-h-[60vh] overflow-y-auto pr-2">
          <div className="space-y-2">
            <Label>Name</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="CPU-Reflex Production" />
          </div>

          <div className="space-y-2">
            <Label>Beschreibung</Label>
            <textarea
              className="flex min-h-[60px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Optionale Beschreibung..."
              rows={2}
            />
          </div>

          <div className="space-y-2">
            <Label>Node (optional)</Label>
            <select className={selectClass} value={nodeId} onChange={(e) => setNodeId(e.target.value)}>
              <option value="">Alle Nodes</option>
              {nodes.map((node) => (
                <option key={node.id} value={node.id}>
                  {node.name}
                </option>
              ))}
            </select>
          </div>

          <div className="space-y-2">
            <Label>Metrik</Label>
            <select className={selectClass} value={triggerMetric} onChange={(e) => setTriggerMetric(e.target.value)}>
              {metrics.map((m) => (
                <option key={m.value} value={m.value}>{m.label}</option>
              ))}
            </select>
          </div>

          <div className="grid grid-cols-2 gap-2">
            <div className="space-y-2">
              <Label>Operator</Label>
              <select className={selectClass} value={operator} onChange={(e) => setOperator(e.target.value)}>
                {operators.map((op) => (
                  <option key={op} value={op}>{op}</option>
                ))}
              </select>
            </div>
            <div className="space-y-2">
              <Label>Schwellenwert</Label>
              <Input value={threshold} onChange={(e) => setThreshold(e.target.value)} type="number" />
            </div>
          </div>

          <div className="space-y-2">
            <Label>Aktion</Label>
            <select
              className={selectClass}
              value={actionType}
              onChange={(e) => setActionType(e.target.value as ReflexActionType)}
            >
              {actionTypes.map((a) => (
                <option key={a.value} value={a.value}>{a.label}</option>
              ))}
            </select>
          </div>

          {actionType === "restart_service" && (
            <div className="space-y-2">
              <Label>Service-Name</Label>
              <Input value={serviceName} onChange={(e) => setServiceName(e.target.value)} placeholder="nginx" />
            </div>
          )}

          {actionType === "run_command" && (
            <div className="space-y-2">
              <Label>Befehl</Label>
              <Input value={command} onChange={(e) => setCommand(e.target.value)} placeholder="systemctl reload nginx" />
            </div>
          )}

          {(actionType === "start_vm" || actionType === "stop_vm" || actionType === "snapshot") && (
            <div className="grid grid-cols-2 gap-2">
              <div className="space-y-2">
                <Label>VMID</Label>
                <Input value={vmid} onChange={(e) => setVmid(e.target.value)} type="number" placeholder="100" />
              </div>
              <div className="space-y-2">
                <Label>VM-Typ</Label>
                <select className={selectClass} value={vmType} onChange={(e) => setVmType(e.target.value)}>
                  <option value="qemu">QEMU</option>
                  <option value="lxc">LXC</option>
                </select>
              </div>
            </div>
          )}

          <div className="space-y-2">
            <Label>Cooldown (Sekunden)</Label>
            <Input value={cooldownSeconds} onChange={(e) => setCooldownSeconds(e.target.value)} type="number" />
          </div>

          {/* Zeitplanung */}
          <div className="space-y-2">
            <Label>Zeitplanung</Label>
            <select className={selectClass} value={scheduleType} onChange={(e) => setScheduleType(e.target.value)}>
              <option value="always">Immer aktiv</option>
              <option value="time_window">Zeitfenster</option>
              <option value="cron">Cron-Ausdruck</option>
            </select>
          </div>

          {scheduleType === "time_window" && (
            <>
              <div className="grid grid-cols-2 gap-2">
                <div className="space-y-2">
                  <Label>Von</Label>
                  <Input type="time" value={timeStart} onChange={(e) => setTimeStart(e.target.value)} />
                </div>
                <div className="space-y-2">
                  <Label>Bis</Label>
                  <Input type="time" value={timeEnd} onChange={(e) => setTimeEnd(e.target.value)} />
                </div>
              </div>
              <div className="space-y-2">
                <Label>Wochentage</Label>
                <div className="flex flex-wrap gap-2">
                  {["So", "Mo", "Di", "Mi", "Do", "Fr", "Sa"].map((day, i) => (
                    <button
                      key={i}
                      type="button"
                      onClick={() => {
                        setTimeDays(prev => prev.includes(i) ? prev.filter(d => d !== i) : [...prev, i]);
                      }}
                      className={cn(
                        "rounded-md border px-3 py-1 text-sm transition-colors",
                        timeDays.includes(i) ? "bg-primary text-primary-foreground" : "hover:bg-accent"
                      )}
                    >
                      {day}
                    </button>
                  ))}
                </div>
              </div>
            </>
          )}

          {scheduleType === "cron" && (
            <div className="space-y-2">
              <Label>Cron-Ausdruck</Label>
              <Input value={cronExpr} onChange={(e) => setCronExpr(e.target.value)} placeholder="*/5 * * * *" />
              <p className="text-xs text-muted-foreground">z.B. &ldquo;*/5 * * * *&rdquo; = alle 5 Minuten</p>
            </div>
          )}

          {/* KI-Integration */}
          <div className="flex items-center justify-between">
            <div>
              <Label>KI-Analyse</Label>
              <p className="text-xs text-muted-foreground">KI bewertet Regel-Trigger intelligent</p>
            </div>
            <Switch checked={aiEnabled} onCheckedChange={setAiEnabled} />
          </div>

          <div className="space-y-2">
            <Label>Prioritaet</Label>
            <Input value={priority} onChange={(e) => setPriority(e.target.value)} type="number" min="0" max="100" placeholder="0 = hoechste" />
          </div>

          <div className="space-y-2">
            <Label>Tags</Label>
            <Input value={tags} onChange={(e) => setTags(e.target.value)} placeholder="production, critical, netzwerk" />
            <p className="text-xs text-muted-foreground">Kommagetrennte Tags zur Kategorisierung</p>
          </div>

          {error && <p className="text-sm text-destructive">{error}</p>}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button onClick={handleSubmit} disabled={loading || !name}>
            {loading ? "Speichern..." : isEdit ? "Aktualisieren" : "Erstellen"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
