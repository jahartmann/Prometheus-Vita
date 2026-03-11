"use client";

import { useState, useMemo, useCallback, useEffect } from "react";
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
  CheckSquare,
  Tag,
  Tags,
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
import { Checkbox } from "@/components/ui/checkbox";
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
import { VMTagAssignDialog } from "@/components/tags/vm-tag-assign-dialog";
import { BulkTagDialog } from "@/components/tags/bulk-tag-dialog";
import { ErrorBoundary } from "@/components/error-boundary";
import { useNodeStore } from "@/stores/node-store";
import { vmApi, bulkVmApi, tagApi, toArray } from "@/lib/api";
import { toast } from "sonner";
import type { VM, BulkVMResult, Tag as TagType } from "@/types/api";
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
  const [selectedVmIds, setSelectedVmIds] = useState<Set<number>>(new Set());
  const [bulkAction, setBulkAction] = useState<string | null>(null);
  const [bulkLoading, setBulkLoading] = useState(false);
  const [tagDialogVm, setTagDialogVm] = useState<VM | null>(null);
  const [bulkTagOpen, setBulkTagOpen] = useState(false);
  const [vmTagsMap, setVmTagsMap] = useState<Record<number, TagType[]>>({});
  const { nodes, nodeStatus } = useNodeStore();
  const currentNode = nodes.find((n) => n.id === nodeId);

  // Fetch tags for all VMs
  const fetchAllVmTags = useCallback(async () => {
    const tagPromises = vms.map(async (vm) => {
      try {
        const res = await tagApi.getVMTags(nodeId, vm.vmid);
        return { vmid: vm.vmid, tags: toArray<TagType>(res.data) };
      } catch {
        return { vmid: vm.vmid, tags: [] };
      }
    });
    const results = await Promise.all(tagPromises);
    const map: Record<number, TagType[]> = {};
    for (const r of results) {
      map[r.vmid] = r.tags;
    }
    setVmTagsMap(map);
  }, [nodeId, vms]);

  useEffect(() => {
    if (vms.length > 0) {
      fetchAllVmTags();
    }
  }, [vms, fetchAllVmTags]);

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

  const allSelected =
    filteredAndSorted.length > 0 &&
    filteredAndSorted.every((vm) => selectedVmIds.has(vm.vmid));

  const toggleSelectAll = () => {
    if (allSelected) {
      setSelectedVmIds(new Set());
    } else {
      setSelectedVmIds(new Set(filteredAndSorted.map((vm) => vm.vmid)));
    }
  };

  const toggleSelect = (vmid: number) => {
    setSelectedVmIds((prev) => {
      const next = new Set(prev);
      if (next.has(vmid)) {
        next.delete(vmid);
      } else {
        next.add(vmid);
      }
      return next;
    });
  };

  const handleBulkAction = useCallback(
    async (action: string) => {
      const vmids = Array.from(selectedVmIds);
      if (vmids.length === 0) return;

      setBulkLoading(true);
      try {
        const res = await bulkVmApi.execute(nodeId, { vmids, action });
        const results = toArray<BulkVMResult>(res.data);

        const succeeded = results.filter((r) => r.success).length;
        const failed = results.filter((r) => !r.success).length;

        if (failed === 0) {
          toast.success(`${succeeded} VMs: Aktion "${action}" erfolgreich`);
        } else {
          toast.warning(
            `${succeeded} erfolgreich, ${failed} fehlgeschlagen`
          );
        }

        setSelectedVmIds(new Set());
        setTimeout(() => onRefresh?.(), 2000);
      } catch {
        toast.error("Bulk-Aktion fehlgeschlagen");
      } finally {
        setBulkLoading(false);
        setBulkAction(null);
      }
    },
    [nodeId, selectedVmIds, onRefresh]
  );

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

      {selectedVmIds.size > 0 && (
        <div className="flex items-center gap-2 rounded-lg border bg-muted/50 p-3">
          <CheckSquare className="h-4 w-4 text-primary" />
          <span className="text-sm font-medium">
            {selectedVmIds.size} VMs ausgewaehlt
          </span>
          <div className="ml-auto flex gap-2">
            <Button
              size="sm"
              variant="outline"
              onClick={() => setBulkAction("start")}
              disabled={bulkLoading}
            >
              <Play className="mr-1 h-3 w-3" />
              Starten
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={() => setBulkAction("shutdown")}
              disabled={bulkLoading}
            >
              <PowerOff className="mr-1 h-3 w-3" />
              Herunterfahren
            </Button>
            <Button
              size="sm"
              variant="destructive"
              onClick={() => setBulkAction("stop")}
              disabled={bulkLoading}
            >
              <Square className="mr-1 h-3 w-3" />
              Stoppen
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={() => setBulkTagOpen(true)}
              disabled={bulkLoading}
            >
              <Tags className="mr-1 h-3 w-3" />
              Tags zuweisen
            </Button>
            <Button
              size="sm"
              variant="ghost"
              onClick={() => setSelectedVmIds(new Set())}
            >
              Auswahl aufheben
            </Button>
          </div>
        </div>
      )}

      <div className="rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[40px]">
                <Checkbox
                  checked={allSelected}
                  onCheckedChange={toggleSelectAll}
                />
              </TableHead>
              <TableHead>
                <SortButton field="vmid">ID</SortButton>
              </TableHead>
              <TableHead>Typ</TableHead>
              <TableHead>
                <SortButton field="name">Name</SortButton>
              </TableHead>
              <TableHead>Tags</TableHead>
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
              <TableHead>Net I/O</TableHead>
              <TableHead>Uptime</TableHead>
              <TableHead className="w-[50px]">Aktionen</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filteredAndSorted.map((vm) => (
              <TableRow key={vm.vmid} data-state={selectedVmIds.has(vm.vmid) ? "selected" : undefined}>
                <TableCell>
                  <Checkbox
                    checked={selectedVmIds.has(vm.vmid)}
                    onCheckedChange={() => toggleSelect(vm.vmid)}
                  />
                </TableCell>
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
                  <div className="flex flex-wrap gap-1 items-center">
                    {/* Proxmox native tags */}
                    {(Array.isArray(vm.tags) ? vm.tags : typeof vm.tags === 'string' && vm.tags ? vm.tags.split(/[,;]/).map(t => t.trim()).filter(Boolean) : []).map((tag) => (
                      <Badge
                        key={`pve-${tag}`}
                        variant="outline"
                        className="text-[10px] px-1.5 py-0 h-5"
                      >
                        {tag}
                      </Badge>
                    ))}
                    {/* System tags */}
                    {(vmTagsMap[vm.vmid] || []).map((tag) => (
                      <Badge
                        key={tag.id}
                        className="text-[10px] px-1.5 py-0 h-5 cursor-pointer"
                        style={{ backgroundColor: tag.color, color: "white" }}
                        onClick={() => setTagDialogVm(vm)}
                      >
                        {tag.name}
                      </Badge>
                    ))}
                    <button
                      className="h-5 w-5 rounded-full border border-dashed border-muted-foreground/40 flex items-center justify-center hover:bg-accent transition-colors"
                      onClick={() => setTagDialogVm(vm)}
                      title="Tags verwalten"
                    >
                      <Tag className="h-2.5 w-2.5 text-muted-foreground" />
                    </button>
                  </div>
                </TableCell>
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
                <TableCell className="text-xs">
                  <span className="text-green-600">{formatBytes(vm.net_in ?? 0)}</span>
                  {" / "}
                  <span className="text-red-500">{formatBytes(vm.net_out ?? 0)}</span>
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

                        <DropdownMenuItem onClick={() => setTagDialogVm(vm)}>
                          <Tag className="h-4 w-4" />
                          Tags
                        </DropdownMenuItem>

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

      {/* Bulk Action Confirmation */}
      <AlertDialog
        open={!!bulkAction}
        onOpenChange={(open) => !open && setBulkAction(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Bulk-Aktion ausfuehren?</AlertDialogTitle>
            <AlertDialogDescription>
              {selectedVmIds.size} VMs werden mit der Aktion &quot;{bulkAction}&quot;
              ausgefuehrt. {bulkAction === "stop" &&
                "Dies entspricht einem harten Ausschalten und kann zu Datenverlust fuehren."}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Abbrechen</AlertDialogCancel>
            <AlertDialogAction
              className={
                bulkAction === "stop"
                  ? "bg-destructive text-destructive-foreground hover:bg-destructive/90"
                  : ""
              }
              onClick={() => {
                if (bulkAction) {
                  handleBulkAction(bulkAction);
                }
              }}
            >
              {bulkLoading ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : null}
              Ausfuehren
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
        <ErrorBoundary>
          <VmConsole
            open={!!consoleVm}
            onOpenChange={(open) => !open && setConsoleVm(null)}
            nodeId={nodeId}
            vmid={consoleVm.vmid}
            vmType={consoleVm.type}
            vmName={consoleVm.name}
            hostname={currentNode.hostname}
            port={currentNode.port}
            pveNode={nodeStatus[nodeId]?.node}
          />
        </ErrorBoundary>
      )}

      {/* VM Tag Assign Dialog */}
      {tagDialogVm && (
        <VMTagAssignDialog
          open={!!tagDialogVm}
          onOpenChange={(open) => !open && setTagDialogVm(null)}
          nodeId={nodeId}
          vmid={tagDialogVm.vmid}
          vmType={tagDialogVm.type}
          vmName={tagDialogVm.name}
          onTagsChanged={fetchAllVmTags}
        />
      )}

      {/* Bulk Tag Dialog */}
      <BulkTagDialog
        open={bulkTagOpen}
        onOpenChange={setBulkTagOpen}
        preselectedVMs={Array.from(selectedVmIds).map((vmid) => {
          const vm = vms.find((v) => v.vmid === vmid);
          return { nodeId, vmid, vmType: vm?.type || "qemu" };
        })}
        onComplete={() => {
          fetchAllVmTags();
          setSelectedVmIds(new Set());
        }}
      />
    </div>
  );
}
