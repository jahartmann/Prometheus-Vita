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
import { escalationApi, getApiErrorMessage } from "@/lib/api";
import { useNodeStore } from "@/stores/node-store";
import type { AlertIncident, IncidentStatus } from "@/types/api";
import { toast } from "sonner";

const severityConfig: Record<
  IncidentStatus,
  { label: string; icon: typeof AlertTriangle; color: string; badgeVariant: "destructive" | "default" | "secondary" | "outline" }
> = {
  triggered: {
    label: "Ausgelöst",
    icon: XCircle,
    color: "text-red-500",
    badgeVariant: "destructive",
  },
  acknowledged: {
    label: "Bestätigt",
    icon: AlertTriangle,
    color: "text-yellow-500",
    badgeVariant: "default",
  },
  resolved: {
    label: "Gelöst",
    icon: CheckCircle2,
    color: "text-green-500",
    badgeVariant: "secondary",
  },
};

export default function AlertsPage() {
  const [incidents, setIncidents] = useState<AlertIncident[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [actionId, setActionId] = useState<string | null>(null);
  const [bulkActionRunning, setBulkActionRunning] = useState(false);
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const { nodes, fetchNodes } = useNodeStore();

  const fetchIncidents = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const resp = await escalationApi.listIncidents(100);
      const data = resp.data;
      setIncidents(Array.isArray(data) ? data : []);
    } catch (err: unknown) {
      const message = getApiErrorMessage(err, "Alarme konnten nicht geladen werden");
      setError(message);
      setIncidents([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchIncidents();
    fetchNodes();
  }, [fetchIncidents, fetchNodes]);

  const handleAcknowledge = async (id: string) => {
    setActionId(id);
    setError(null);
    try {
      await escalationApi.acknowledgeIncident(id);
      toast.success("Alarm bestätigt");
      await fetchIncidents();
    } catch (err: unknown) {
      const message = getApiErrorMessage(err, "Alarm konnte nicht bestätigt werden");
      setError(message);
      toast.error(message);
    } finally {
      setActionId(null);
    }
  };

  const handleResolve = async (id: string) => {
    setActionId(id);
    setError(null);
    try {
      await escalationApi.resolveIncident(id);
      toast.success("Alarm gelöst");
      await fetchIncidents();
    } catch (err: unknown) {
      const message = getApiErrorMessage(err, "Alarm konnte nicht gelöst werden");
      setError(message);
      toast.error(message);
    } finally {
      setActionId(null);
    }
  };

  const handleAcknowledgeAll = async () => {
    const triggered = incidents.filter((i) => i.status === "triggered");
    if (triggered.length === 0) return;
    setBulkActionRunning(true);
    setError(null);
    const results = await Promise.allSettled(
      triggered.map((inc) => escalationApi.acknowledgeIncident(inc.id))
    );
    const failed = results.filter((result) => result.status === "rejected").length;
    const partialFailureMessage =
      failed > 0 ? `${failed} von ${triggered.length} Alarmen konnten nicht bestätigt werden` : null;
    if (partialFailureMessage) {
      toast.error(partialFailureMessage);
    } else {
      toast.success(`${triggered.length} Alarme bestätigt`);
    }
    await fetchIncidents();
    if (partialFailureMessage) setError(partialFailureMessage);
    setBulkActionRunning(false);
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
          <h2 className="text-xl font-bold">Alarme</h2>
          <p className="text-sm text-muted-foreground">
            Übersicht aller ausgelösten Alarme und Vorfälle.
          </p>
        </div>
        <div className="flex gap-2">
          {triggeredCount > 0 && (
            <Button
              variant="outline"
              size="sm"
              onClick={handleAcknowledgeAll}
              disabled={bulkActionRunning || isLoading}
            >
              <CheckCircle2 className="mr-2 h-4 w-4" />
              Alle bestätigen ({triggeredCount})
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

      {error && (
        <div className="flex items-start gap-3 rounded-lg border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm">
          <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-destructive" />
          <div>
            <p className="font-medium text-destructive">Alert-Daten nicht aktuell</p>
            <p className="text-muted-foreground">{error}</p>
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <KpiCard
          title="Gesamt"
          value={incidents.length}
          subtitle="Vorfälle"
          icon={Bell}
          color="blue"
        />
        <KpiCard
          title="Ausgelöst"
          value={triggeredCount}
          subtitle="Offene Alarme"
          icon={XCircle}
          color="red"
        />
        <KpiCard
          title="Bestätigt"
          value={acknowledgedCount}
          subtitle="In Bearbeitung"
          icon={AlertTriangle}
          color="orange"
        />
        <KpiCard
          title="Gelöst"
          value={resolvedCount}
          subtitle="Abgeschlossen"
          icon={Shield}
          color="green"
        />
      </div>

      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-base">Vorfälle</CardTitle>
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="w-[180px]" aria-label="Status filtern">
                <SelectValue placeholder="Status filtern" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Alle Status</SelectItem>
                <SelectItem value="triggered">Ausgelöst</SelectItem>
                <SelectItem value="acknowledged">Bestätigt</SelectItem>
                <SelectItem value="resolved">Gelöst</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardHeader>
        <CardContent className="p-0">
          {isLoading ? (
            <div className="space-y-3 p-4" aria-busy="true" aria-label="Vorfälle werden geladen">
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : filtered.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16">
              <Info className="mb-4 h-12 w-12 text-muted-foreground" />
              <p className="text-lg font-medium">Keine Vorfälle</p>
              <p className="mt-1 text-sm text-muted-foreground">
                {statusFilter === "all"
                  ? "Es sind derzeit keine Alarme vorhanden."
                  : "Keine Vorfälle mit diesem Status."}
              </p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Status</TableHead>
                  <TableHead>Zeitpunkt</TableHead>
                  <TableHead>Stufe</TableHead>
                  <TableHead>Bestätigt</TableHead>
                  <TableHead>Gelöst</TableHead>
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
                              disabled={actionId === incident.id || bulkActionRunning}
                            >
                              Bestätigen
                            </Button>
                          )}
                          {(incident.status === "triggered" ||
                            incident.status === "acknowledged") && (
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handleResolve(incident.id)}
                              disabled={actionId === incident.id || bulkActionRunning}
                            >
                              Lösen
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
