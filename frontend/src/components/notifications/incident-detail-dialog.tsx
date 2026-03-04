"use client";

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import type { AlertIncident, IncidentStatus } from "@/types/api";

interface IncidentDetailDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  incident: AlertIncident | null;
  onAcknowledge: (id: string) => void;
  onResolve: (id: string) => void;
}

const statusVariant: Record<IncidentStatus, "default" | "secondary" | "destructive" | "outline"> = {
  triggered: "destructive",
  acknowledged: "secondary",
  resolved: "outline",
};

const statusLabel: Record<IncidentStatus, string> = {
  triggered: "Ausgeloest",
  acknowledged: "Bestaetigt",
  resolved: "Geloest",
};

const formatDate = (dateStr?: string | null) => {
  if (!dateStr) return "-";
  return new Date(dateStr).toLocaleString("de-DE", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
};

export function IncidentDetailDialog({
  open,
  onOpenChange,
  incident,
  onAcknowledge,
  onResolve,
}: IncidentDetailDialogProps) {
  if (!incident) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Vorfall-Details</DialogTitle>
          <DialogDescription>
            ID: {incident.id}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <span className="text-sm text-muted-foreground">Status:</span>
            <Badge variant={statusVariant[incident.status]}>
              {statusLabel[incident.status]}
            </Badge>
          </div>

          <div className="grid grid-cols-2 gap-3 text-sm">
            <div>
              <span className="text-muted-foreground">Alert-Regel-ID:</span>
              <p className="font-mono text-xs">{incident.alert_rule_id}</p>
            </div>
            <div>
              <span className="text-muted-foreground">Aktuelle Stufe:</span>
              <p>{incident.current_step}</p>
            </div>
            <div>
              <span className="text-muted-foreground">Ausgeloest:</span>
              <p>{formatDate(incident.triggered_at)}</p>
            </div>
            <div>
              <span className="text-muted-foreground">Letzte Eskalation:</span>
              <p>{formatDate(incident.last_escalated_at)}</p>
            </div>
            <div>
              <span className="text-muted-foreground">Bestaetigt:</span>
              <p>{formatDate(incident.acknowledged_at)}</p>
            </div>
            <div>
              <span className="text-muted-foreground">Geloest:</span>
              <p>{formatDate(incident.resolved_at)}</p>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Schliessen
          </Button>
          {incident.status === "triggered" && (
            <Button
              variant="secondary"
              onClick={() => {
                onAcknowledge(incident.id);
                onOpenChange(false);
              }}
            >
              Bestaetigen
            </Button>
          )}
          {incident.status !== "resolved" && (
            <Button
              onClick={() => {
                onResolve(incident.id);
                onOpenChange(false);
              }}
            >
              Loesen
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
