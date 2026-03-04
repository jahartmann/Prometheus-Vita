"use client";

import { useState, useMemo } from "react";
import { Search, ArrowUpDown, Monitor, Container, ArrowRightLeft } from "lucide-react";
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
import { MigrateVmDialog } from "@/components/migration/migrate-vm-dialog";
import { useNodeStore } from "@/stores/node-store";
import type { VM } from "@/types/api";
import { formatBytes, formatUptime, formatPercentage } from "@/lib/utils";

interface VmListProps {
  vms: VM[];
  nodeId: string;
}

type SortField = "vmid" | "name" | "status" | "cpu_usage" | "memory_used";
type SortDirection = "asc" | "desc";

export function VmList({ vms, nodeId }: VmListProps) {
  const [search, setSearch] = useState("");
  const [sortField, setSortField] = useState<SortField>("vmid");
  const [sortDirection, setSortDirection] = useState<SortDirection>("asc");
  const [migrateVm, setMigrateVm] = useState<VM | null>(null);
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
              <TableHead></TableHead>
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
                <TableCell>{vm.uptime > 0 ? formatUptime(vm.uptime) : "--"}</TableCell>
                <TableCell>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setMigrateVm(vm)}
                    title="VM migrieren"
                  >
                    <ArrowRightLeft className="h-3.5 w-3.5" />
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <p className="text-xs text-muted-foreground">
        {filteredAndSorted.length} von {vms.length} VMs/Container
      </p>

      {migrateVm && (
        <MigrateVmDialog
          open={!!migrateVm}
          onOpenChange={(open) => !open && setMigrateVm(null)}
          vm={migrateVm}
          sourceNodeId={nodeId}
          sourceNodeName={currentNode?.name || nodeId}
        />
      )}
    </div>
  );
}
