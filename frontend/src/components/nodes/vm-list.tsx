"use client";

import { useState, useMemo, useCallback } from "react";
import {
  Search,
  ArrowUpDown,
  Monitor,
  Container,
  MoreHorizontal,
  Play,
  Square,
  PowerOff,
  Pause,
  Camera,
  Terminal,
  ArrowRightLeft,
  Loader2,
} from "lucide-react";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { MigrateVmDialog } from "@/components/migration/migrate-vm-dialog";
import { SnapshotDialog } from "@/components/nodes/snapshot-dialog";
import { VmConsole } from "@/components/nodes/vm-console";
import { useNodeStore } from "@/stores/node-store";
import { vmApi } from "@/lib/api";
import { toast } from "sonner";
import type { VM } from "@/types/api";
import { formatBytes, formatUptime, formatPercentage } from "@/lib/utils";

interface VmListProps {
  vms: VM[];
  nodeId: string;
  onRefresh?: () => void;
}

type SortField = "vmid" | "name" | "status" | "cpu_usage" | "memory_used";
type SortDirection = "asc" | "desc";

export function VmList({ vms, nodeId, onRefresh }: VmListProps) {
  const [search, setSearch] = useState("");
  const [sortField, setSortField] = useState<SortField>("vmid");
  const [sortDirection, setSortDirection] = useState<SortDirection>("asc");
  const [migrateVm, setMigrateVm] = useState<VM | null>(null);
  const [snapshotVm, setSnapshotVm] = useState<VM | null>(null);
  const [consoleVm, setConsoleVm] = useState<VM | null>(null);
  const [stopConfirmVm, setStopConfirmVm] = useState<VM | null>(null);
  const [actionLoading, setActionLoading] = useState<number | null>(null);
  const { nodes } = useNodeStore();
  const currentNode = nodes.find((n) => n.id === nodeId);

  const toggleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection(sortDirection === "asc" ? "desc" : "asc");
    } else {
      setSortField(field);
      setSortDirection("asc");
    }
  };

  const filteredAndSorted = useMemo(() => {
    let result = vms;

    if (search) {
      const lower = search.toLowerCase();
      result = result.filter(
        (vm) =>
          vm.name.toLowerCase().includes(lower) ||
          vm.vmid.toString().includes(lower)
      );
    }

    result = [...result].sort((a, b) => {
      let comparison = 0;
      switch (sortField) {
        case "vmid":
          comparison = a.vmid - b.vmid;
          break;
        case "name":
          comparison = a.name.localeCompare(b.name);
          break;
        case "status":
          comparison = a.status.localeCompare(b.status);
          break;
        case "cpu_usage":
          comparison = a.cpu_usage - b.cpu_usage;
          break;
        case "memory_used":
          comparison = a.memory_used - b.memory_used;
          break;
      }
      return sortDirection === "asc" ? comparison : -comparison;
    });

    return result;
  }, [vms, search, sortField, sortDirection]);

  const statusVariant = (status: string) => {
    switch (status) {
      case "running":
        return "success" as const;
      case "stopped":
        return "secondary" as const;
      case "paused":
        return "warning" as const;
      default:
        return "outline" as const;
    }
  };

  const refreshAfterAction = useCallback(() => {
    setTimeout(() => {
      onRefresh?.();
    }, 2000);
  }, [onRefresh]);

  const actionLabels: Record<string, string> = {
    start: "wird gestartet",
    shutdown: "wird heruntergefahren",
    stop: "wird gestoppt",
    suspend: "wird pausiert",
    resume: "wird fortgesetzt",
  };

  const handleAction = useCallback(
    async (vm: VM, action: "start" | "shutdown" | "stop" | "suspend" | "resume") => {
      setActionLoading(vm.vmid);
      try {
        switch (action) {
          case "start":
            await vmApi.start(nodeId, vm.vmid, vm.type);
            break;
          case "shutdown":
            await vmApi.shutdown(nodeId, vm.vmid, vm.type);
            break;
          case "stop":
            await vmApi.stop(nodeId, vm.vmid, vm.type);
            break;
          case "suspend":
            await vmApi.suspend(nodeId, vm.vmid, vm.type);
            break;
          case "resume":
            await vmApi.resume(nodeId, vm.vmid, vm.type);
            break;
        }
        toast.success(`${vm.name} ${actionLabels[action]}`);
        refreshAfterAction();
      } catch {
        toast.error(`Aktion fehlgeschlagen fuer ${vm.name}`);
      } finally {
        setActionLoading(null);
      }
    },
    [nodeId, refreshAfterAction]
  );

  const SortButton = ({
    field,
    children,
  }: {
    field: SortField;
    children: React.ReactNode;
  }) => (
    <Button
      variant="ghost"
      size="sm"
      className="-ml-3 h-8"
      onClick={() => toggleSort(field)}
    >
      {children}
      <ArrowUpDown className="ml-1 h-3 w-3" />
    </Button>
  );

  if (vms.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center rounded-xl border border-dashed py-12">
        <p className="text-muted-foreground">
          Keine VMs oder Container auf diesem Node.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <div className="relative flex-1">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="VMs suchen..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
      </div>

      <div className="rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>
                <SortButton field="vmid">ID</SortButton>
              </TableHead>
              <TableHead>Typ</TableHead>
              <TableHead>
                <SortButton field="name">Name</SortButton>
              </TableHead>
              <TableHead>
                <SortButton field="status">Status</SortButton>
              </TableHead>
              <TableHead>
                <SortButton field="cpu_usage">CPU</SortButton>
              </TableHead>
              <TableHead>
                <SortButton field="memory_used">RAM</SortButton>
              </TableHead>
              <TableHead>Disk</TableHead>
              <TableHead>Uptime</TableHead>
              <TableHead className="w-[50px]">Aktionen</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filteredAndSorted.map((vm) => (
              <TableRow key={vm.vmid}>
                <TableCell className="font-mono text-sm">{vm.vmid}</TableCell>
                <TableCell>
                  {vm.type === "qemu" ? (
                    <Monitor className="h-4 w-4 text-muted-foreground" />
                  ) : (
                    <Container className="h-4 w-4 text-muted-foreground" />
                  )}
                </TableCell>
                <TableCell className="font-medium">{vm.name}</TableCell>
                <TableCell>
                  <Badge variant={statusVariant(vm.status)}>{vm.status}</Badge>
                </TableCell>
                <TableCell>{formatPercentage(vm.cpu_usage)}</TableCell>
                <TableCell>
                  {formatBytes(vm.memory_used)} / {formatBytes(vm.memory_total)}
                </TableCell>
                <TableCell>
                  {formatBytes(vm.disk_used)} / {formatBytes(vm.disk_total)}
                </TableCell>
                <TableCell>
                  {vm.uptime > 0 ? formatUptime(vm.uptime) : "--"}
                </TableCell>
                <TableCell>
                  {actionLoading === vm.vmid ? (
                    <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                  ) : (
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                          <MoreHorizontal className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        {vm.status !== "running" && vm.status !== "paused" && (
                          <DropdownMenuItem
                            onClick={() => handleAction(vm, "start")}
                          >
                            <Play className="h-4 w-4" />
                            Starten
                          </DropdownMenuItem>
                        )}
                        {vm.status === "running" && (
                          <>
                            <DropdownMenuItem
                              onClick={() => handleAction(vm, "shutdown")}
                            >
                              <PowerOff className="h-4 w-4" />
                              Herunterfahren
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              onClick={() => setStopConfirmVm(vm)}
                              className="text-destructive focus:text-destructive"
                            >
                              <Square className="h-4 w-4" />
                              Stoppen
                            </DropdownMenuItem>
                          </>
                        )}
                        {vm.status === "running" && vm.type === "qemu" && (
                          <DropdownMenuItem
                            onClick={() => handleAction(vm, "suspend")}
                          >
                            <Pause className="h-4 w-4" />
                            Pausieren
                          </DropdownMenuItem>
                        )}
                        {(vm.status === "paused" || vm.status === "suspended") && (
                          <DropdownMenuItem
                            onClick={() => handleAction(vm, "resume")}
                          >
                            <Play className="h-4 w-4" />
                            Fortsetzen
                          </DropdownMenuItem>
                        )}

                        <DropdownMenuSeparator />

                        <DropdownMenuItem onClick={() => setSnapshotVm(vm)}>
                          <Camera className="h-4 w-4" />
                          Snapshots
                        </DropdownMenuItem>

                        {vm.status === "running" && (
                          <DropdownMenuItem onClick={() => setConsoleVm(vm)}>
                            <Terminal className="h-4 w-4" />
                            Konsole
                          </DropdownMenuItem>
                        )}

                        <DropdownMenuSeparator />

                        <DropdownMenuItem onClick={() => setMigrateVm(vm)}>
                          <ArrowRightLeft className="h-4 w-4" />
                          Migrieren
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <p className="text-xs text-muted-foreground">
        {filteredAndSorted.length} von {vms.length} VMs/Container
      </p>

      {/* Stop Confirmation */}
      <AlertDialog
        open={!!stopConfirmVm}
        onOpenChange={(open) => !open && setStopConfirmVm(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>VM stoppen?</AlertDialogTitle>
            <AlertDialogDescription>
              VM &quot;{stopConfirmVm?.name}&quot; (ID: {stopConfirmVm?.vmid}) wirklich
              stoppen? Dies entspricht einem harten Ausschalten und kann zu
              Datenverlust fuehren.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Abbrechen</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={() => {
                if (stopConfirmVm) {
                  handleAction(stopConfirmVm, "stop");
                  setStopConfirmVm(null);
                }
              }}
            >
              Stoppen
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Migration Dialog */}
      {migrateVm && (
        <MigrateVmDialog
          open={!!migrateVm}
          onOpenChange={(open) => !open && setMigrateVm(null)}
          vm={migrateVm}
          sourceNodeId={nodeId}
          sourceNodeName={currentNode?.name || nodeId}
        />
      )}

      {/* Snapshot Dialog */}
      {snapshotVm && (
        <SnapshotDialog
          open={!!snapshotVm}
          onOpenChange={(open) => !open && setSnapshotVm(null)}
          nodeId={nodeId}
          vmid={snapshotVm.vmid}
          vmType={snapshotVm.type}
          vmName={snapshotVm.name}
        />
      )}

      {/* Console Dialog */}
      {consoleVm && currentNode && (
        <VmConsole
          open={!!consoleVm}
          onOpenChange={(open) => !open && setConsoleVm(null)}
          nodeId={nodeId}
          vmid={consoleVm.vmid}
          vmType={consoleVm.type}
          vmName={consoleVm.name}
          hostname={currentNode.hostname}
          port={currentNode.port}
        />
      )}
    </div>
  );
}
