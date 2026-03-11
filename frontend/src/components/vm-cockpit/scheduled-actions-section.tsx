"use client";

import { useEffect, useState, useCallback } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Plus, Trash2, Clock, CalendarClock, RefreshCw } from "lucide-react";
import { scheduledActionApi, toArray } from "@/lib/api";
import type { ScheduledAction } from "@/types/api";

interface ScheduledActionsSectionProps {
  nodeId: string;
  vmid: number;
  vmType: string;
}

export function ScheduledActionsSection({ nodeId, vmid, vmType }: ScheduledActionsSectionProps) {
  const [actions, setActions] = useState<ScheduledAction[]>([]);
  const [loading, setLoading] = useState(false);
  const [dialogOpen, setDialogOpen] = useState(false);

  const fetchActions = useCallback(async () => {
    if (!nodeId || !vmid) return;
    setLoading(true);
    try {
      const res = await scheduledActionApi.list(nodeId, vmid);
      setActions(toArray<ScheduledAction>(res.data));
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [nodeId, vmid]);

  useEffect(() => {
    fetchActions();
  }, [fetchActions]);

  const handleDelete = async (actionId: string) => {
    try {
      await scheduledActionApi.delete(nodeId, vmid, actionId);
      fetchActions();
    } catch {
      // ignore
    }
  };

  const actionLabel = (action: string): string => {
    switch (action) {
      case "start":
        return "Starten";
      case "stop":
        return "Stoppen";
      case "restart":
        return "Neustarten";
      case "shutdown":
        return "Herunterfahren";
      default:
        return action;
    }
  };

  const actionVariant = (action: string) => {
    switch (action) {
      case "start":
        return "success" as const;
      case "stop":
        return "destructive" as const;
      case "restart":
        return "warning" as const;
      case "shutdown":
        return "secondary" as const;
      default:
        return "outline" as const;
    }
  };

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-3">
        <CardTitle className="text-base flex items-center gap-2">
          <CalendarClock className="h-4 w-4" />
          Geplante Aktionen
        </CardTitle>
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="icon" onClick={fetchActions} disabled={loading}>
            <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          </Button>
          <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
            <DialogTrigger asChild>
              <Button size="sm">
                <Plus className="h-4 w-4 mr-1" />
                Neue Aktion
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Geplante Aktion erstellen</DialogTitle>
              </DialogHeader>
              <CreateActionForm
                nodeId={nodeId}
                vmid={vmid}
                vmType={vmType}
                onCreated={() => {
                  setDialogOpen(false);
                  fetchActions();
                }}
              />
            </DialogContent>
          </Dialog>
        </div>
      </CardHeader>
      <CardContent>
        {actions.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-4">
            Keine geplanten Aktionen konfiguriert.
          </p>
        ) : (
          <div className="space-y-3">
            {actions.map((action) => (
              <div
                key={action.id}
                className="flex items-center justify-between rounded-lg border p-3"
              >
                <div className="space-y-1">
                  <div className="flex items-center gap-2">
                    <Badge variant={actionVariant(action.action)}>
                      {actionLabel(action.action)}
                    </Badge>
                    <Badge variant={action.is_active ? "success" : "secondary"}>
                      {action.is_active ? "Aktiv" : "Inaktiv"}
                    </Badge>
                  </div>
                  <div className="flex items-center gap-3 text-xs text-muted-foreground">
                    <span className="flex items-center gap-1">
                      <Clock className="h-3 w-3" />
                      {action.schedule_cron}
                    </span>
                    {action.description && <span>{action.description}</span>}
                  </div>
                </div>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => handleDelete(action.id)}
                >
                  <Trash2 className="h-4 w-4 text-destructive" />
                </Button>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function CreateActionForm({
  nodeId,
  vmid,
  vmType,
  onCreated,
}: {
  nodeId: string;
  vmid: number;
  vmType: string;
  onCreated: () => void;
}) {
  const [action, setAction] = useState("start");
  const [cron, setCron] = useState("0 6 * * 1-5");
  const [description, setDescription] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      await scheduledActionApi.create(nodeId, vmid, {
        node_id: nodeId,
        vmid,
        vm_type: vmType,
        action,
        schedule_cron: cron,
        is_active: true,
        description: description || undefined,
      });
      onCreated();
    } catch {
      // ignore
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="space-y-2">
        <Label>Aktion</Label>
        <select
          value={action}
          onChange={(e) => setAction(e.target.value)}
          className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
        >
          <option value="start">Starten</option>
          <option value="shutdown">Herunterfahren</option>
          <option value="stop">Stoppen</option>
          <option value="restart">Neustarten</option>
        </select>
      </div>

      <div className="space-y-2">
        <Label>Zeitplan (Cron)</Label>
        <Input
          value={cron}
          onChange={(e) => setCron(e.target.value)}
          placeholder="0 6 * * 1-5"
        />
        <p className="text-xs text-muted-foreground">
          Beispiel: &quot;0 6 * * 1-5&quot; = Montag-Freitag um 06:00 Uhr
        </p>
      </div>

      <div className="space-y-2">
        <Label>Beschreibung (optional)</Label>
        <Input
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="z.B. Morgens starten"
        />
      </div>

      <Button type="submit" className="w-full" disabled={submitting || !cron}>
        {submitting ? "Erstelle..." : "Aktion erstellen"}
      </Button>
    </form>
  );
}
