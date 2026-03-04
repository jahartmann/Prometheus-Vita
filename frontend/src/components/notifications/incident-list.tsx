"use client";

import { useState } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { escalationApi } from "@/lib/api";
import { IncidentDetailDialog } from "./incident-detail-dialog";
import type { AlertIncident, IncidentStatus } from "@/types/api";

interface IncidentListProps {
  incidents: AlertIncident[];
  isLoading: boolean;
  onRefresh: () => void;
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
  });
};

export function IncidentList({ incidents, isLoading, onRefresh }: IncidentListProps) {
  const [selectedIncident, setSelectedIncident] = useState<AlertIncident | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  const handleAcknowledge = async (id: string) => {
    await escalationApi.acknowledgeIncident(id);
    onRefresh();
  };

  const handleResolve = async (id: string) => {
    await escalationApi.resolveIncident(id);
    onRefresh();
  };

  return (
    <>
      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Vorfall-ID</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Stufe</TableHead>
                <TableHead>Ausgeloest</TableHead>
                <TableHead>Letzte Eskalation</TableHead>
                <TableHead className="w-48">Aktionen</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading && incidents.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                    Laden...
                  </TableCell>
                </TableRow>
              ) : incidents.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                    Keine Vorfaelle vorhanden.
                  </TableCell>
                </TableRow>
              ) : (
                incidents.map((incident) => (
                  <TableRow
                    key={incident.id}
                    className="cursor-pointer"
                    onClick={() => {
                      setSelectedIncident(incident);
                      setDetailOpen(true);
                    }}
                  >
                    <TableCell className="font-mono text-xs">
                      {incident.id.slice(0, 8)}...
                    </TableCell>
                    <TableCell>
                      <Badge variant={statusVariant[incident.status]}>
                        {statusLabel[incident.status]}
                      </Badge>
                    </TableCell>
                    <TableCell>{incident.current_step}</TableCell>
                    <TableCell className="text-muted-foreground">
                      {formatDate(incident.triggered_at)}
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {formatDate(incident.last_escalated_at)}
                    </TableCell>
                    <TableCell>
                      <div className="flex gap-1" onClick={(e) => e.stopPropagation()}>
                        {incident.status === "triggered" && (
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleAcknowledge(incident.id)}
                          >
                            Bestaetigen
                          </Button>
                        )}
                        {incident.status !== "resolved" && (
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
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <IncidentDetailDialog
        open={detailOpen}
        onOpenChange={setDetailOpen}
        incident={selectedIncident}
        onAcknowledge={handleAcknowledge}
        onResolve={handleResolve}
      />
    </>
  );
}
