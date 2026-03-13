"use client";

import { useEffect, useState } from "react";
import { Search, RefreshCw, Skull } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { useVMCockpitStore } from "@/stores/vm-cockpit-store";
import { CockpitError } from "./cockpit-error";

export function SystemProcesses() {
  const { processes, isLoadingProcesses, fetchProcesses, killProcess, processesError } =
    useVMCockpitStore();
  const [filter, setFilter] = useState("");
  const [killPid, setKillPid] = useState<number | null>(null);

  useEffect(() => {
    fetchProcesses();
  }, [fetchProcesses]);

  if (processesError) {
    return <CockpitError {...processesError} onRetry={fetchProcesses} />;
  }

  const filtered = processes.filter((p) => {
    if (!filter) return true;
    const lower = filter.toLowerCase();
    return (
      p.command.toLowerCase().includes(lower) ||
      p.user.toLowerCase().includes(lower) ||
      p.pid.toString().includes(lower)
    );
  });

  if (isLoadingProcesses && processes.length === 0) {
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
            placeholder="Prozesse filtern..."
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="pl-9"
          />
        </div>
        <Button variant="outline" size="sm" onClick={fetchProcesses}>
          <RefreshCw className="mr-2 h-3 w-3" />
          Aktualisieren
        </Button>
      </div>

      <div className="rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[80px]">PID</TableHead>
              <TableHead className="w-[100px]">Benutzer</TableHead>
              <TableHead className="w-[80px] text-right">CPU%</TableHead>
              <TableHead className="w-[80px] text-right">MEM%</TableHead>
              <TableHead>Befehl</TableHead>
              <TableHead className="w-[60px]">Aktion</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filtered.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="text-center text-muted-foreground py-8">
                  Keine Prozesse gefunden
                </TableCell>
              </TableRow>
            ) : (
              filtered.map((proc) => (
                <TableRow key={proc.pid}>
                  <TableCell className="font-mono text-sm">{proc.pid}</TableCell>
                  <TableCell>{proc.user}</TableCell>
                  <TableCell
                    className={`text-right font-mono ${
                      proc.cpu > 50 ? "text-red-500 font-bold" : ""
                    }`}
                  >
                    {proc.cpu.toFixed(1)}
                  </TableCell>
                  <TableCell
                    className={`text-right font-mono ${
                      proc.mem > 50 ? "text-orange-500 font-bold" : ""
                    }`}
                  >
                    {proc.mem.toFixed(1)}
                  </TableCell>
                  <TableCell className="max-w-[300px] truncate font-mono text-xs">
                    {proc.command}
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-7 w-7 p-0 text-destructive hover:text-destructive"
                      onClick={() => setKillPid(proc.pid)}
                    >
                      <Skull className="h-3.5 w-3.5" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <p className="text-xs text-muted-foreground">
        {filtered.length} von {processes.length} Prozessen
      </p>

      <ConfirmDialog
        open={killPid !== null}
        onOpenChange={(open) => !open && setKillPid(null)}
        title="Prozess beenden?"
        description={`Prozess mit PID ${killPid} wirklich beenden? Dies kann zu Datenverlust fuehren.`}
        confirmLabel="Beenden"
        variant="destructive"
        onConfirm={() => {
          if (killPid !== null) {
            killProcess(killPid);
            setKillPid(null);
          }
        }}
      />
    </div>
  );
}
