"use client";

import { useEffect, useState } from "react";
import { RefreshCw, ArrowDownToLine, ArrowUpFromLine, Radio } from "lucide-react";
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
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import { Skeleton } from "@/components/ui/skeleton";
import { networkApi, toArray } from "@/lib/api";
import type { NodePort, NodePortsData } from "@/types/api";

interface NodePortsProps {
  nodeId: string;
}

// Well-known port labels
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

function PortTable({ ports, showPeer }: { ports: NodePort[]; showPeer?: boolean }) {
  if (ports.length === 0) {
    return (
      <div className="py-8 text-center text-sm text-muted-foreground">
        Keine Ports gefunden.
      </div>
    );
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-[80px]">Protokoll</TableHead>
          <TableHead>Lokale Adresse</TableHead>
          <TableHead className="w-[100px]">Port</TableHead>
          {showPeer && <TableHead>Ziel-Adresse</TableHead>}
          {showPeer && <TableHead className="w-[100px]">Ziel-Port</TableHead>}
          <TableHead>Prozess</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {ports.map((port, idx) => (
          <TableRow key={`${port.protocol}-${port.local_port}-${port.peer_port}-${idx}`}>
            <TableCell>
              <Badge
                variant={port.protocol === "tcp" ? "default" : "secondary"}
                className="text-[10px]"
              >
                {port.protocol.toUpperCase()}
              </Badge>
            </TableCell>
            <TableCell className="font-mono text-sm">{port.local_address || "*"}</TableCell>
            <TableCell className="font-mono font-bold">
              {port.local_port}
              <PortLabel port={port.local_port} />
            </TableCell>
            {showPeer && (
              <TableCell className="font-mono text-sm">
                {port.peer_address || "*"}
              </TableCell>
            )}
            {showPeer && (
              <TableCell className="font-mono text-sm">
                {port.peer_port || "-"}
                {port.peer_port ? <PortLabel port={port.peer_port} /> : null}
              </TableCell>
            )}
            <TableCell className="text-sm text-muted-foreground">
              {port.process || "-"}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

export function NodePorts({ nodeId }: NodePortsProps) {
  const [portsData, setPortsData] = useState<NodePortsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

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

  const listening = portsData?.listening || [];
  const established = portsData?.established || [];
  const other = portsData?.other || [];
  const total = listening.length + established.length + other.length;

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
        <div className="grid gap-3 sm:grid-cols-3">
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
              <p className="text-lg font-bold">{total}</p>
            </div>
          </div>
        </div>

        {/* Port tabs */}
        <Tabs defaultValue="listening">
          <TabsList>
            <TabsTrigger value="listening">
              Lauschend ({listening.length})
            </TabsTrigger>
            <TabsTrigger value="established">
              Verbunden ({established.length})
            </TabsTrigger>
            {other.length > 0 && (
              <TabsTrigger value="other">
                Sonstige ({other.length})
              </TabsTrigger>
            )}
          </TabsList>

          <TabsContent value="listening" className="mt-3">
            <div className="rounded-lg border">
              <PortTable ports={listening} />
            </div>
          </TabsContent>

          <TabsContent value="established" className="mt-3">
            <div className="rounded-lg border">
              <PortTable ports={established} showPeer />
            </div>
          </TabsContent>

          {other.length > 0 && (
            <TabsContent value="other" className="mt-3">
              <div className="rounded-lg border">
                <PortTable ports={other} showPeer />
              </div>
            </TabsContent>
          )}
        </Tabs>
      </CardContent>
    </Card>
  );
}
