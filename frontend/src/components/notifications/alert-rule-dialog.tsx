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
import { alertApi } from "@/lib/api";
import { toast } from "sonner";
import type { AlertRule, AlertSeverity, NotificationChannel, Node, EscalationPolicy } from "@/types/api";

interface AlertRuleDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
  rule?: AlertRule | null;
  nodes: Node[];
  channels: NotificationChannel[];
  escalationPolicies?: EscalationPolicy[];
}

const metrics = [
  { value: "cpu_usage", label: "CPU-Auslastung (%)" },
  { value: "memory_usage", label: "RAM-Auslastung (%)" },
  { value: "disk_usage", label: "Festplatten-Auslastung (%)" },
  { value: "load_avg", label: "Load Average" },
  { value: "net_in", label: "Netzwerk Eingang (Bytes)" },
  { value: "net_out", label: "Netzwerk Ausgang (Bytes)" },
];

const operators = [">", ">=", "<", "<=", "==", "!="];
const severities: AlertSeverity[] = ["info", "warning", "critical"];

export function AlertRuleDialog({
  open,
  onOpenChange,
  onSuccess,
  rule,
  nodes,
  channels,
  escalationPolicies = [],
}: AlertRuleDialogProps) {
  const isEdit = !!rule;
  const [name, setName] = useState("");
  const [nodeId, setNodeId] = useState("");
  const [metric, setMetric] = useState("cpu_usage");
  const [operator, setOperator] = useState(">");
  const [threshold, setThreshold] = useState("80");
  const [durationSeconds, setDurationSeconds] = useState("60");
  const [severity, setSeverity] = useState<AlertSeverity>("warning");
  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);
  const [escalationPolicyId, setEscalationPolicyId] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (rule) {
      setName(rule.name);
      setNodeId(rule.node_id);
      setMetric(rule.metric);
      setOperator(rule.operator);
      setThreshold(String(rule.threshold));
      setDurationSeconds(String(rule.duration_seconds));
      setSeverity(rule.severity);
      setSelectedChannels(rule.channel_ids || []);
      setEscalationPolicyId(rule.escalation_policy_id || "");
    } else {
      setName("");
      setNodeId(nodes.length > 0 ? nodes[0].id : "");
      setMetric("cpu_usage");
      setOperator(">");
      setThreshold("80");
      setDurationSeconds("60");
      setSeverity("warning");
      setSelectedChannels([]);
      setEscalationPolicyId("");
    }
    setError(null);
  }, [rule, open, nodes]);

  const toggleChannel = (id: string) => {
    setSelectedChannels((prev) =>
      prev.includes(id) ? prev.filter((c) => c !== id) : [...prev, id]
    );
  };

  const handleSubmit = async () => {
    setLoading(true);
    setError(null);
    try {
      if (isEdit && rule) {
        await alertApi.updateRule(rule.id, {
          name,
          metric,
          operator,
          threshold: parseFloat(threshold),
          duration_seconds: parseInt(durationSeconds) || 0,
          severity,
          channel_ids: selectedChannels,
          escalation_policy_id: escalationPolicyId || undefined,
        });
      } else {
        await alertApi.createRule({
          name,
          node_id: nodeId,
          metric,
          operator,
          threshold: parseFloat(threshold),
          duration_seconds: parseInt(durationSeconds) || 0,
          severity,
          channel_ids: selectedChannels,
          escalation_policy_id: escalationPolicyId || undefined,
        });
      }
      onSuccess();
      onOpenChange(false);
      toast.success(isEdit ? "Regel aktualisiert" : "Regel erstellt");
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Fehler beim Speichern");
      toast.error("Fehler beim Speichern der Regel");
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {isEdit ? "Alert-Regel bearbeiten" : "Neue Alert-Regel"}
          </DialogTitle>
          <DialogDescription>
            {isEdit
              ? "Alert-Regel Konfiguration aktualisieren."
              : "Definieren Sie, wann eine Benachrichtigung ausgelöst werden soll."}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 max-h-[60vh] overflow-y-auto pr-2">
          <div className="space-y-2">
            <Label>Name</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="CPU-Alert Production" />
          </div>

          {!isEdit && (
            <div className="space-y-2">
              <Label>Node</Label>
              <select
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                value={nodeId}
                onChange={(e) => setNodeId(e.target.value)}
              >
                {nodes.map((node) => (
                  <option key={node.id} value={node.id}>
                    {node.name}
                  </option>
                ))}
              </select>
            </div>
          )}

          <div className="space-y-2">
            <Label>Metrik</Label>
            <select
              className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              value={metric}
              onChange={(e) => setMetric(e.target.value)}
            >
              {metrics.map((m) => (
                <option key={m.value} value={m.value}>
                  {m.label}
                </option>
              ))}
            </select>
          </div>

          <div className="grid grid-cols-2 gap-2">
            <div className="space-y-2">
              <Label>Operator</Label>
              <select
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                value={operator}
                onChange={(e) => setOperator(e.target.value)}
              >
                {operators.map((op) => (
                  <option key={op} value={op}>
                    {op}
                  </option>
                ))}
              </select>
            </div>
            <div className="space-y-2">
              <Label>Schwellenwert</Label>
              <Input value={threshold} onChange={(e) => setThreshold(e.target.value)} type="number" />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-2">
            <div className="space-y-2">
              <Label>Cooldown (Sekunden)</Label>
              <Input value={durationSeconds} onChange={(e) => setDurationSeconds(e.target.value)} type="number" />
            </div>
            <div className="space-y-2">
              <Label>Schweregrad</Label>
              <select
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                value={severity}
                onChange={(e) => setSeverity(e.target.value as AlertSeverity)}
              >
                {severities.map((s) => (
                  <option key={s} value={s}>
                    {s === "info" ? "Info" : s === "warning" ? "Warnung" : "Kritisch"}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {channels.length > 0 && (
            <div className="space-y-2">
              <Label>Benachrichtigungskanäle</Label>
              <div className="space-y-1 rounded-md border p-2">
                {channels.map((ch) => (
                  <label key={ch.id} className="flex items-center gap-2 text-sm cursor-pointer">
                    <input
                      type="checkbox"
                      checked={selectedChannels.includes(ch.id)}
                      onChange={() => toggleChannel(ch.id)}
                      className="rounded"
                    />
                    {ch.name}
                    <span className="text-muted-foreground">({ch.type})</span>
                  </label>
                ))}
              </div>
            </div>
          )}

          {escalationPolicies.length > 0 && (
            <div className="space-y-2">
              <Label>Eskalationsrichtlinie</Label>
              <select
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                value={escalationPolicyId}
                onChange={(e) => setEscalationPolicyId(e.target.value)}
              >
                <option value="">Keine Eskalation</option>
                {escalationPolicies.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name}
                  </option>
                ))}
              </select>
            </div>
          )}

          {error && <p className="text-sm text-destructive">{error}</p>}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button onClick={handleSubmit} disabled={loading || !name || !nodeId}>
            {loading ? "Speichern..." : isEdit ? "Aktualisieren" : "Erstellen"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
