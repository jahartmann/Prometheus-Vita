"use client";

import { useEffect, useState } from "react";
import {
  RefreshCw,
  ArrowDownToLine,
  ArrowUpFromLine,
  Radio,
  Server,
  Monitor,
  ChevronDown,
  ChevronRight,
  Box,
  AlertTriangle,
  ShieldOff,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { networkApi } from "@/lib/api";
import type { NodePort, NodePortsData, VMPortGroup } from "@/types/api";

interface NodePortsProps {
  nodeId: string;
}

const knownPorts: Record<number, string> = {
  22: "SSH",
  25: "SMTP",
  53: "DNS",
  80: "HTTP",
  443: "HTTPS",
  993: "IMAPS",
  995: "POP3S",
  3128: "Proxy",
  3306: "MySQL",
  3389: "RDP",
  5432: "PostgreSQL",
  5900: "VNC",
  6379: "Redis",
  8006: "Proxmox",
  8080: "HTTP Alt",
  8443: "HTTPS Alt",
  9090: "Prometheus",
  27017: "MongoDB",
};

function PortLabel({ port }: { port: number }) {
  const label = knownPorts[port];
  if (!label) return null;
  return (
    <Badge variant="outline" className="ml-2 text-[10px] font-normal">
      {label}
    </Badge>
  );
}

function GroupIcon({ type }: { type: string }) {
  switch (type) {
    case "node":
      return <Server className="h-4 w-4 text-blue-500" />;
    case "qemu":
      return <Monitor className="h-4 w-4 text-green-500" />;
    case "lxc":
      return <Box className="h-4 w-4 text-orange-500" />;
    default:
      return <Server className="h-4 w-4" />;
  }
}

function ScanStatusBadge({ group }: { group: VMPortGroup }) {
  if (group.scan_status === "ok") return null;
  if (group.scan_status === "no_agent") {
    return (
      <Badge variant="outline" className="text-[10px] text-amber-600 border-amber-300 shrink-0">
        <ShieldOff className="mr-1 h-3 w-3" />
        Kein Agent
      </Badge>
    );
  }
  return (
    <Badge variant="outline" className="text-[10px] text-red-600 border-red-300 shrink-0">
      <AlertTriangle className="mr-1 h-3 w-3" />
      Fehler
    </Badge>
  );
}

function PortGroup({
  group,
  filter,
}: {
  group: VMPortGroup;
  filter: string;
}) {
  const [expanded, setExpanded] = useState(group.type === "node");
  const ports = group.ports || [];

  const filteredPorts = ports.filter((p) => {
    if (!filter) return true;
    const lower = filter.toLowerCase();
    return (
      String(p.local_port).includes(lower) ||
      p.protocol.toLowerCase().includes(lower) ||
      (p.process || "").toLowerCase().includes(lower) ||
      p.local_address.includes(lower) ||
      (knownPorts[p.local_port] || "").toLowerCase().includes(lower)
    );
  });

  const listeningCount = filteredPorts.filter(
    (p) => p.state === "LISTEN" || p.state === "LISTENING"
  ).length;
  const estabCount = filteredPorts.filter(
    (p) => p.state === "ESTAB" || p.state === "ESTABLISHED"
  ).length;

  const hasError = group.scan_status !== "ok";

  if (filter && filteredPorts.length === 0 && !hasError) return null;

  return (
    <div className="rounded-lg border">
      <button
        className="flex w-full items-center gap-3 p-3 hover:bg-muted/50 transition-colors"
        onClick={() => setExpanded(!expanded)}
      >
        {expanded ? (
          <ChevronDown className="h-4 w-4 text-muted-foreground shrink-0" />
        ) : (
          <ChevronRight className="h-4 w-4 text-muted-foreground shrink-0" />
        )}
        <GroupIcon type={group.type} />
        <div className="flex items-center gap-2 min-w-0">
          <span className="font-medium text-sm truncate">{group.name}</span>
          {group.type !== "node" && (
            <Badge variant="outline" className="text-[10px] shrink-0">
              {group.type === "qemu" ? "QEMU" : "LXC"} {group.vmid}
            </Badge>
          )}
          {group.type === "node" && (
            <Badge variant="secondary" className="text-[10px] shrink-0">
              Node
            </Badge>
          )}
          <ScanStatusBadge group={group} />
        </div>
        <div className="ml-auto flex items-center gap-2 shrink-0 text-xs text-muted-foreground">
          {!hasError && (
            <>
              <span className="text-green-600">{listeningCount} lauschend</span>
              <span className="text-blue-600">{estabCount} verbunden</span>
              <span>{filteredPorts.length} gesamt</span>
            </>
          )}
        </div>
      </button>

      {expanded && hasError && (
        <div className="border-t px-4 py-3">
          <div className="rounded-md bg-amber-500/10 p-3 text-sm text-amber-700 dark:text-amber-400">
            {group.scan_status === "no_agent" ? (
              <>
                <p className="font-medium">QEMU Guest Agent nicht verfuegbar</p>
                <p className="mt-1 text-xs text-muted-foreground">
                  Der Guest Agent muss in der VM installiert sein, um Ports auszulesen.
                </p>
                <p className="mt-1 text-xs font-mono text-muted-foreground">
                  Linux: apt install qemu-guest-agent &amp;&amp; systemctl enable --now qemu-guest-agent
                </p>
              </>
            ) : (
              <p>{group.scan_error || "Scan fehlgeschlagen"}</p>
            )}
          </div>
        </div>
      )}

      {expanded && !hasError && filteredPorts.length > 0 && (
        <div className="border-t">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[80px]">Protokoll</TableHead>
                <TableHead className="w-[90px]">Status</TableHead>
                <TableHead>Lokale Adresse</TableHead>
                <TableHead className="w-[100px]">Port</TableHead>
                <TableHead>Ziel</TableHead>
                <TableHead>Prozess</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredPorts.map((port, idx) => (
                <TableRow key={`${port.protocol}-${port.local_port}-${port.peer_port}-${idx}`}>
                  <TableCell>
                    <Badge
                      variant={port.protocol === "tcp" ? "default" : "secondary"}
                      className="text-[10px]"
                    >
                      {port.protocol.toUpperCase()}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={
                        port.state === "LISTEN" || port.state === "LISTENING"
                          ? "success"
                          : port.state === "ESTAB" || port.state === "ESTABLISHED"
                            ? "default"
                            : "outline"
                      }
                      className="text-[10px]"
                    >
                      {port.state}
                    </Badge>
                  </TableCell>
                  <TableCell className="font-mono text-sm">
                    {port.local_address || "*"}
                  </TableCell>
                  <TableCell className="font-mono font-bold">
                    {port.local_port}
                    <PortLabel port={port.local_port} />
                  </TableCell>
                  <TableCell className="font-mono text-sm text-muted-foreground">
                    {port.peer_address && port.peer_port
                      ? `${port.peer_address}:${port.peer_port}`
                      : "-"}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {port.process || "-"}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {expanded && !hasError && filteredPorts.length === 0 && (
        <div className="border-t py-4 text-center text-sm text-muted-foreground">
          Keine Ports gefunden.
        </div>
      )}
    </div>
  );
}

export function NodePorts({ nodeId }: NodePortsProps) {
  const [portsData, setPortsData] = useState<NodePortsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState("");

  const fetchPorts = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await networkApi.getPorts(nodeId);
      const data = res.data?.data || res.data;
      setPortsData(data);
    } catch {
      setError("Port-Informationen konnten nicht abgerufen werden.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchPorts();
  }, [nodeId]);

  if (loading && !portsData) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Port-Uebersicht</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-48 w-full" />
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Port-Uebersicht</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
            {error}
          </div>
          <Button variant="outline" size="sm" className="mt-3" onClick={fetchPorts}>
            <RefreshCw className="mr-2 h-3 w-3" />
            Erneut versuchen
          </Button>
        </CardContent>
      </Card>
    );
  }

  const groups = portsData?.groups || [];
  const listening = portsData?.listening || [];
  const established = portsData?.established || [];
  const totalPorts = groups.reduce((sum, g) => sum + (g.ports || []).length, 0);

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-base">Port-Uebersicht</CardTitle>
          <Button variant="outline" size="sm" onClick={fetchPorts} disabled={loading}>
            <RefreshCw className={`mr-2 h-3 w-3 ${loading ? "animate-spin" : ""}`} />
            Aktualisieren
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Summary */}
        <div className="grid gap-3 sm:grid-cols-4">
          <div className="flex items-center gap-3 rounded-lg border p-3">
            <ArrowDownToLine className="h-5 w-5 text-green-500" />
            <div>
              <p className="text-sm text-muted-foreground">Lauschend</p>
              <p className="text-lg font-bold">{listening.length}</p>
            </div>
          </div>
          <div className="flex items-center gap-3 rounded-lg border p-3">
            <ArrowUpFromLine className="h-5 w-5 text-blue-500" />
            <div>
              <p className="text-sm text-muted-foreground">Verbunden</p>
              <p className="text-lg font-bold">{established.length}</p>
            </div>
          </div>
          <div className="flex items-center gap-3 rounded-lg border p-3">
            <Radio className="h-5 w-5 text-zinc-500" />
            <div>
              <p className="text-sm text-muted-foreground">Gesamt</p>
              <p className="text-lg font-bold">{totalPorts}</p>
            </div>
          </div>
          <div className="flex items-center gap-3 rounded-lg border p-3">
            <Server className="h-5 w-5 text-purple-500" />
            <div>
              <p className="text-sm text-muted-foreground">Quellen</p>
              <p className="text-lg font-bold">{groups.length}</p>
            </div>
          </div>
        </div>

        {/* Filter */}
        <Input
          placeholder="Port, Protokoll, Prozess oder Service filtern..."
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          className="max-w-sm"
        />

        {/* Groups */}
        <div className="space-y-3">
          {groups.map((group) => (
            <PortGroup
              key={`${group.type}-${group.vmid}`}
              group={group}
              filter={filter}
            />
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
