"use client";

import { useEffect } from "react";
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
import { Skeleton } from "@/components/ui/skeleton";
import { useVMCockpitStore } from "@/stores/vm-cockpit-store";
import { CockpitError } from "./cockpit-error";

export function SystemPorts() {
  const { ports, isLoadingPorts, fetchPorts, portsError } = useVMCockpitStore();

  useEffect(() => {
    fetchPorts();
  }, [fetchPorts]);

  if (portsError) {
    return <CockpitError {...portsError} onRetry={fetchPorts} />;
  }

  if (isLoadingPorts && ports.length === 0) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-end">
        <Button variant="outline" size="sm" onClick={fetchPorts}>
          <RefreshCw className="mr-2 h-3 w-3" />
          Aktualisieren
        </Button>
      </div>

      <div className="rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[100px]">Port</TableHead>
              <TableHead className="w-[100px]">Protokoll</TableHead>
              <TableHead>Adresse</TableHead>
              <TableHead>Prozess</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {ports.length === 0 ? (
              <TableRow>
                <TableCell colSpan={4} className="text-center text-muted-foreground py-8">
                  Keine offenen Ports gefunden
                </TableCell>
              </TableRow>
            ) : (
              ports.map((port, idx) => (
                <TableRow key={`${port.protocol}-${port.port}-${idx}`}>
                  <TableCell className="font-mono font-bold">{port.port}</TableCell>
                  <TableCell>
                    <Badge variant={port.protocol === "tcp" ? "default" : "secondary"}>
                      {port.protocol.toUpperCase()}
                    </Badge>
                  </TableCell>
                  <TableCell className="font-mono text-sm">{port.address}</TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {port.process || "-"}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <p className="text-xs text-muted-foreground">
        {ports.length} offene Ports
      </p>
    </div>
  );
}
