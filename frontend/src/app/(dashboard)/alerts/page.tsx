"use client";

import { useEffect, useState, useCallback } from "react";
import {
  AlertTriangle,
  Bell,
  CheckCircle2,
  Info,
  RefreshCw,
  Shield,
  XCircle,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { KpiCard } from "@/components/ui/kpi-card";
import { escalationApi } from "@/lib/api";
import { useNodeStore } from "@/stores/node-store";
import type { AlertIncident, IncidentStatus } from "@/types/api";

const severityConfig: Record<
  IncidentStatus,
  { label: string; icon: typeof AlertTriangle; color: string; badgeVariant: "destructive" | "default" | "secondary" | "outline" }
> = {
  triggered: {
    label: "Ausgeloest",
    icon: XCircle,
    color: "text-red-500",
    badgeVariant: "destructive",
  },
  acknowledged: {
    label: "Bestaetigt",
    icon: AlertTriangle,
    color: "text-yellow-500",
    badgeVariant: "default",
  },
  resolved: {
    label: "Geloest",
    icon: CheckCircle2,
    color: "text-green-500",
    badgeVariant: "secondary",
  },
};

export default function AlertsPage() {
  const [incidents, setIncidents] = useState<AlertIncident[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const { nodes, fetchNodes } = useNodeStore();

  const fetchIncidents = useCallback(async () => {
    setIsLoading(true);
    try {
      const resp = await escalationApi.listIncidents(100);
      const data = resp.data;
      setIncidents(Array.isArray(data) ? data : []);
    } catch {
      // Fehler
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchIncidents();
    fetchNodes();
  }, [fetchIncidents, fetchNodes]);

  const handleAcknowledge = async (id: string) => {
    try {
      await escalationApi.acknowledgeIncident(id);
      fetchIncidents();
    } catch {
      // Fehler
    }
  };

  const handleResolve = async (id: string) => {
    try {
      await escalationApi.resolveIncident(id);
      fetchIncidents();
    } catch {
      // Fehler
    }
  };

  const handleAcknowledgeAll = async () => {
    const triggered = incidents.filter((i) => i.status === "triggered");
    for (const inc of triggered) {
      try {
        await escalationApi.acknowledgeIncident(inc.id);
      } catch {
        // continue
      }
    }
    fetchIncidents();
  };

  const filtered =
    statusFilter === "all"
      ? incidents
      : incidents.filter((i) => i.status === statusFilter);

  const triggeredCount = incidents.filter((i) => i.status === "triggered").length;
  const acknowledgedCount = incidents.filter((i) => i.status === "acknowledged").length;
  const resolvedCount = incidents.filter((i) => i.status === "resolved").length;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-bold">Alerts</h2>
          <p className="text-sm text-muted-foreground">
            Uebersicht aller ausgeloesten Alarme und Vorfaelle.
          </p>
        </div>
        <div className="flex gap-2">
          {triggeredCount > 0 && (
            <Button variant="outline" size="sm" onClick={handleAcknowledgeAll}>
              <CheckCircle2 className="mr-2 h-4 w-4" />
              Alle bestaetigen ({triggeredCount})
            </Button>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={fetchIncidents}
            disabled={isLoading}
          >
            <RefreshCw
              className={`h-4 w-4 mr-2 ${isLoading ? "animate-spin" : ""}`}
            />
            Aktualisieren
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <KpiCard
          title="Gesamt"
          value={incidents.length}
          subtitle="Vorfaelle"
          icon={Bell}
          color="blue"
        />
        <KpiCard
          title="Ausgeloest"
          value={triggeredCount}
          subtitle="Offene Alarme"
          icon={XCircle}
          color="red"
        />
        <KpiCard
          title="Bestaetigt"
          value={acknowledgedCount}
          subtitle="In Bearbeitung"
          icon={AlertTriangle}
          color="orange"
        />
        <KpiCard
          title="Geloest"
          value={resolvedCount}
          subtitle="Abgeschlossen"
          icon={Shield}
          color="green"
        />
      </div>

      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-base">Vorfaelle</CardTitle>
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="w-[180px]">
                <SelectValue placeholder="Status filtern" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Alle Status</SelectItem>
                <SelectItem value="triggered">Ausgeloest</SelectItem>
                <SelectItem value="acknowledged">Bestaetigt</SelectItem>
                <SelectItem value="resolved">Geloest</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardHeader>
        <CardContent className="p-0">
          {isLoading ? (
            <div className="space-y-3 p-4">
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : filtered.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16">
              <Info className="mb-4 h-12 w-12 text-muted-foreground" />
              <p className="text-lg font-medium">Keine Vorfaelle</p>
              <p className="mt-1 text-sm text-muted-foreground">
                {statusFilter === "all"
                  ? "Es sind derzeit keine Alarme vorhanden."
                  : "Keine Vorfaelle mit diesem Status."}
              </p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Status</TableHead>
                  <TableHead>Zeitpunkt</TableHead>
                  <TableHead>Stufe</TableHead>
                  <TableHead>Bestaetigt</TableHead>
                  <TableHead>Geloest</TableHead>
                  <TableHead className="text-right">Aktionen</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filtered.map((incident) => {
                  const cfg = severityConfig[incident.status];
                  const Icon = cfg.icon;
                  return (
                    <TableRow key={incident.id}>
                      <TableCell>
                        <Badge variant={cfg.badgeVariant} className="gap-1">
                          <Icon className="h-3 w-3" />
                          {cfg.label}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-sm">
                        {new Date(incident.triggered_at).toLocaleString("de-DE")}
                      </TableCell>
                      <TableCell className="text-sm font-mono">
                        Stufe {incident.current_step}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {incident.acknowledged_at
                          ? new Date(incident.acknowledged_at).toLocaleString("de-DE")
                          : "-"}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {incident.resolved_at
                          ? new Date(incident.resolved_at).toLocaleString("de-DE")
                          : "-"}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex justify-end gap-1">
                          {incident.status === "triggered" && (
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handleAcknowledge(incident.id)}
                            >
                              Bestaetigen
                            </Button>
                          )}
                          {(incident.status === "triggered" ||
                            incident.status === "acknowledged") && (
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handleResolve(incident.id)}
                            >
                              Loesen
                            </Button>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
