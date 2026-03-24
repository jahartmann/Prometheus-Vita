"use client";

import { useState, useMemo } from "react";
import { useRouter } from "next/navigation";
import { ArrowUpDown, ArrowUp, ArrowDown } from "lucide-react";
import {
  AreaChart,
  Area,
  ResponsiveContainer,
} from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { formatPercentage, formatBandwidth } from "@/lib/utils";
import type { VM } from "@/types/api";

interface VMMetricsTableProps {
  vms: VM[];
  nodeId: string;
  vmHistory?: Record<number, Array<{ cpu: number; mem: number }>>;
}

type SortKey = "name" | "vmid" | "status" | "cpu" | "ram" | "netIn" | "netOut";
type SortDir = "asc" | "desc";

function MiniSparkline({ data, color }: { data: Array<{ v: number }>; color: string }) {
  if (data.length < 2) return null;
  return (
    <div className="h-8 w-20">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={data}>
          <Area
            type="monotone"
            dataKey="v"
            stroke={color}
            fill={color}
            fillOpacity={0.15}
            strokeWidth={1}
            dot={false}
            isAnimationActive={false}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}

export function VMMetricsTable({ vms, nodeId, vmHistory }: VMMetricsTableProps) {
  const router = useRouter();
  const [sortKey, setSortKey] = useState<SortKey>("cpu");
  const [sortDir, setSortDir] = useState<SortDir>("desc");

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortKey(key);
      setSortDir("desc");
    }
  };

  const sortedVMs = useMemo(() => {
    const copy = [...vms];
    copy.sort((a, b) => {
      let valA: number | string = 0;
      let valB: number | string = 0;
      switch (sortKey) {
        case "name":
          valA = a.name || "";
          valB = b.name || "";
          break;
        case "vmid":
          valA = a.vmid;
          valB = b.vmid;
          break;
        case "status":
          valA = a.status;
          valB = b.status;
          break;
        case "cpu":
          valA = a.cpu_usage;
          valB = b.cpu_usage;
          break;
        case "ram":
          valA = a.memory_total > 0 ? a.memory_used / a.memory_total : 0;
          valB = b.memory_total > 0 ? b.memory_used / b.memory_total : 0;
          break;
        case "netIn":
          valA = a.net_in;
          valB = b.net_in;
          break;
        case "netOut":
          valA = a.net_out;
          valB = b.net_out;
          break;
      }
      if (typeof valA === "string" && typeof valB === "string") {
        return sortDir === "asc" ? valA.localeCompare(valB) : valB.localeCompare(valA);
      }
      return sortDir === "asc"
        ? (valA as number) - (valB as number)
        : (valB as number) - (valA as number);
    });
    return copy;
  }, [vms, sortKey, sortDir]);

  const SortHeader = ({ label, field }: { label: string; field: SortKey }) => {
    const isActive = sortKey === field;
    return (
      <th
        className="cursor-pointer select-none p-3 text-left font-medium hover:text-foreground"
        onClick={() => toggleSort(field)}
      >
        <div className="flex items-center gap-1">
          {label}
          {isActive ? (
            sortDir === "asc" ? (
              <ArrowUp className="h-3 w-3" />
            ) : (
              <ArrowDown className="h-3 w-3" />
            )
          ) : (
            <ArrowUpDown className="h-3 w-3 opacity-30" />
          )}
        </div>
      </th>
    );
  };

  const statusColor: Record<string, string> = {
    running: "bg-green-500/15 text-green-500 border-green-500/30",
    stopped: "bg-red-500/15 text-red-500 border-red-500/30",
    paused: "bg-amber-500/15 text-amber-500 border-amber-500/30",
    suspended: "bg-blue-500/15 text-blue-500 border-blue-500/30",
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">VMs auf diesem Node</CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        {sortedVMs.length === 0 ? (
          <div className="p-6 text-center text-sm text-muted-foreground">
            Keine VMs gefunden.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-muted-foreground">
                  <SortHeader label="VM Name" field="name" />
                  <SortHeader label="VMID" field="vmid" />
                  <SortHeader label="Status" field="status" />
                  <SortHeader label="CPU %" field="cpu" />
                  <th className="p-3 text-left font-medium">Trend</th>
                  <SortHeader label="RAM %" field="ram" />
                  <SortHeader label="Net In" field="netIn" />
                  <SortHeader label="Net Out" field="netOut" />
                </tr>
              </thead>
              <tbody>
                {sortedVMs.map((vm) => {
                  const memPct =
                    vm.memory_total > 0
                      ? (vm.memory_used / vm.memory_total) * 100
                      : 0;
                  const history = vmHistory?.[vm.vmid];
                  const cpuSparkData = history?.map((h) => ({ v: h.cpu })) ?? [];

                  return (
                    <tr
                      key={vm.vmid}
                      className="cursor-pointer border-b last:border-0 transition-colors hover:bg-muted/50"
                      onClick={() =>
                        router.push(`/nodes/${nodeId}/vms/${vm.vmid}`)
                      }
                    >
                      <td className="p-3">
                        <div className="flex items-center gap-2">
                          <span className="font-medium">{vm.name || `VM ${vm.vmid}`}</span>
                          <Badge variant="outline" className="text-xs">
                            {vm.type}
                          </Badge>
                        </div>
                      </td>
                      <td className="p-3 font-mono text-xs">{vm.vmid}</td>
                      <td className="p-3">
                        <Badge
                          variant="outline"
                          className={statusColor[vm.status] || ""}
                        >
                          {vm.status}
                        </Badge>
                      </td>
                      <td className="p-3 font-mono">
                        {formatPercentage(vm.cpu_usage)}
                      </td>
                      <td className="p-3">
                        <MiniSparkline
                          data={cpuSparkData}
                          color="hsl(210, 80%, 55%)"
                        />
                      </td>
                      <td className="p-3 font-mono">
                        {formatPercentage(memPct)}
                      </td>
                      <td className="p-3 text-blue-500">
                        {formatBandwidth(vm.net_in)}
                      </td>
                      <td className="p-3 text-green-500">
                        {formatBandwidth(vm.net_out)}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
