"use client";

import { useEffect, useState } from "react";
import { Search, RefreshCw, Play, Square, RotateCw } from "lucide-react";
import { Input } from "@/components/ui/input";
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

export function SystemServices() {
  const { services, isLoadingServices, fetchServices, serviceAction, servicesError } =
    useVMCockpitStore();
  const [filter, setFilter] = useState("");

  useEffect(() => {
    fetchServices();
  }, [fetchServices]);

  if (servicesError) {
    return <CockpitError {...servicesError} onRetry={fetchServices} />;
  }

  const filtered = services.filter((s) => {
    if (!filter) return true;
    const lower = filter.toLowerCase();
    return (
      s.unit.toLowerCase().includes(lower) ||
      s.description.toLowerCase().includes(lower)
    );
  });

  const statusVariant = (state: string) => {
    switch (state) {
      case "active":
        return "success" as const;
      case "failed":
        return "destructive" as const;
      default:
        return "secondary" as const;
    }
  };

  if (isLoadingServices && services.length === 0) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <div className="relative flex-1">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Services filtern..."
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="pl-9"
          />
        </div>
        <Button variant="outline" size="sm" onClick={fetchServices}>
          <RefreshCw className="mr-2 h-3 w-3" />
          Aktualisieren
        </Button>
      </div>

      <div className="rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Service</TableHead>
              <TableHead className="w-[100px]">Status</TableHead>
              <TableHead>Beschreibung</TableHead>
              <TableHead className="w-[140px]">Aktionen</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filtered.length === 0 ? (
              <TableRow>
                <TableCell colSpan={4} className="text-center text-muted-foreground py-8">
                  Keine Services gefunden
                </TableCell>
              </TableRow>
            ) : (
              filtered.map((svc) => (
                <TableRow key={svc.unit}>
                  <TableCell className="font-mono text-sm">{svc.unit}</TableCell>
                  <TableCell>
                    <Badge variant={statusVariant(svc.active_state)}>
                      {svc.active_state}
                    </Badge>
                  </TableCell>
                  <TableCell className="max-w-[300px] truncate text-sm text-muted-foreground">
                    {svc.description}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      {svc.active_state !== "active" && (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-7 w-7 p-0"
                          title="Starten"
                          onClick={() => serviceAction(svc.unit, "start")}
                        >
                          <Play className="h-3.5 w-3.5" />
                        </Button>
                      )}
                      {svc.active_state === "active" && (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-7 w-7 p-0"
                          title="Stoppen"
                          onClick={() => serviceAction(svc.unit, "stop")}
                        >
                          <Square className="h-3.5 w-3.5" />
                        </Button>
                      )}
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-7 w-7 p-0"
                        title="Neustarten"
                        onClick={() => serviceAction(svc.unit, "restart")}
                      >
                        <RotateCw className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <p className="text-xs text-muted-foreground">
        {filtered.length} von {services.length} Services
      </p>
    </div>
  );
}
