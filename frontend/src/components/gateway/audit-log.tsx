"use client";

import { useEffect, useState } from "react";
import { RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { gatewayApi, toArray } from "@/lib/api";
import type { AuditLogEntry } from "@/types/api";

export function AuditLog() {
  const [entries, setEntries] = useState<AuditLogEntry[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  const fetchAudit = async () => {
    setIsLoading(true);
    try {
      const resp = await gatewayApi.listAuditLog(50);
      setEntries(toArray<AuditLogEntry>(resp.data));
    } catch {
      // Fehler
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchAudit();
  }, []);

  const statusColor = (code: number) => {
    if (code < 300) return "default";
    if (code < 400) return "secondary";
    return "outline";
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <span className="font-medium">Audit-Log</span>
        <Button variant="outline" size="sm" onClick={fetchAudit}>
          <RefreshCw className="h-3 w-3 mr-1" />
          Aktualisieren
        </Button>
      </div>

      <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Zeitpunkt</TableHead>
            <TableHead>Methode</TableHead>
            <TableHead>Pfad</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>IP</TableHead>
            <TableHead>Dauer</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading && entries.length === 0 ? (
            <TableRow>
              <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                Laden...
              </TableCell>
            </TableRow>
          ) : entries.length === 0 ? (
            <TableRow>
              <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                Keine Audit-Einträge vorhanden.
              </TableCell>
            </TableRow>
          ) : (
            entries.map((entry) => (
              <TableRow key={entry.id}>
                <TableCell className="text-muted-foreground text-sm">
                  {new Date(entry.created_at).toLocaleString("de-DE")}
                </TableCell>
                <TableCell>
                  <Badge variant="outline">{entry.method}</Badge>
                </TableCell>
                <TableCell className="font-mono text-xs max-w-[300px] truncate">
                  {entry.path}
                </TableCell>
                <TableCell>
                  <Badge variant={statusColor(entry.status_code) as "default" | "secondary" | "outline"}>
                    {entry.status_code}
                  </Badge>
                </TableCell>
                <TableCell className="text-muted-foreground text-sm">{entry.ip_address || "-"}</TableCell>
                <TableCell className="text-muted-foreground text-sm">{entry.duration_ms}ms</TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
      </div>
    </div>
  );
}
