"use client";

import { useMemo, useState } from "react";
import { useNetworkStore } from "@/stores/network-store";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { ChevronDown, ChevronRight, ArrowUpDown } from "lucide-react";
import { Button } from "@/components/ui/button";

interface PortEntry {
  port: number;
  protocol: string;
  state: string;
  service?: string;
  version?: string;
  process?: string;
  source?: string;
}

type SortKey = "port" | "protocol" | "state" | "service";
type SortDir = "asc" | "desc";

const WELL_KNOWN: Record<number, string> = {
  22: "ssh", 80: "http", 443: "https", 3306: "mysql",
  5432: "postgres", 6379: "redis", 8080: "http-alt",
  8443: "https-alt", 27017: "mongodb",
};

function getPortSeverity(entry: PortEntry): "green" | "yellow" | "red" {
  if (entry.state !== "open") return "green";
  const known = WELL_KNOWN[entry.port] || entry.service;
  if (!known) return "yellow";
  return "green";
}

function parseResultsJson(results: unknown): PortEntry[] {
  if (!results || typeof results !== "object") return [];
  const obj = results as Record<string, unknown>;

  const ports: PortEntry[] = [];

  // Node-level ports (from /proc/net or ss)
  if (Array.isArray(obj.ports)) {
    for (const p of obj.ports) {
      ports.push({
        port: Number(p.port ?? p.local_port ?? 0),
        protocol: String(p.protocol ?? p.proto ?? "tcp"),
        state: String(p.state ?? "open"),
        service: p.service as string | undefined,
        version: p.version as string | undefined,
        process: p.process as string | undefined,
        source: "Node",
      });
    }
  }

  // Nmap scan results
  if (Array.isArray(obj.nmap_results)) {
    for (const host of obj.nmap_results) {
      const h = host as Record<string, unknown>;
      const hostPorts = Array.isArray(h.ports) ? h.ports : [];
      for (const p of hostPorts) {
        const pe = p as Record<string, unknown>;
        ports.push({
          port: Number(pe.port ?? pe.portid ?? 0),
          protocol: String(pe.protocol ?? "tcp"),
          state: String(pe.state ?? "open"),
          service: pe.service as string | undefined,
          version: pe.version as string | undefined,
          process: undefined,
          source: String(h.ip ?? h.address ?? "External"),
        });
      }
    }
  }

  // VM ports
  if (Array.isArray(obj.vm_ports)) {
    for (const p of obj.vm_ports) {
      ports.push({
        port: Number(p.port ?? 0),
        protocol: String(p.protocol ?? "tcp"),
        state: String(p.state ?? "open"),
        service: p.service as string | undefined,
        version: p.version as string | undefined,
        process: p.process as string | undefined,
        source: `VM ${p.vmid ?? ""}`,
      });
    }
  }

  return ports;
}

interface PortGroupProps {
  label: string;
  ports: PortEntry[];
  filter: string;
  sortKey: SortKey;
  sortDir: SortDir;
}

function PortGroup({ label, ports, filter, sortKey, sortDir }: PortGroupProps) {
  const [open, setOpen] = useState(true);

  const filtered = useMemo(() => {
    const q = filter.toLowerCase();
    return ports
      .filter((p) =>
        !q ||
        String(p.port).includes(q) ||
        (p.service ?? "").toLowerCase().includes(q) ||
        (p.protocol ?? "").toLowerCase().includes(q) ||
        (p.state ?? "").toLowerCase().includes(q) ||
        (p.process ?? "").toLowerCase().includes(q)
      )
      .sort((a, b) => {
        let cmp = 0;
        if (sortKey === "port") cmp = a.port - b.port;
        else if (sortKey === "protocol") cmp = a.protocol.localeCompare(b.protocol);
        else if (sortKey === "state") cmp = a.state.localeCompare(b.state);
        else if (sortKey === "service") cmp = (a.service ?? "").localeCompare(b.service ?? "");
        return sortDir === "asc" ? cmp : -cmp;
      });
  }, [ports, filter, sortKey, sortDir]);

  if (filtered.length === 0) return null;

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger className="flex items-center gap-2 w-full px-3 py-2 rounded-md bg-zinc-800/50 hover:bg-zinc-800 text-sm font-medium text-zinc-300 transition-colors">
        {open ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
        {label}
        <Badge variant="secondary" className="ml-auto text-xs">{filtered.length}</Badge>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <div className="rounded-lg border border-zinc-800 mt-1 overflow-hidden">
          <Table>
            <TableBody>
              {filtered.map((p, i) => {
                const sev = getPortSeverity(p);
                return (
                  <TableRow key={`${p.port}-${p.protocol}-${i}`} className="border-zinc-800/50">
                    <TableCell className="font-mono font-bold text-sm w-24">
                      <span className={
                        sev === "red" ? "text-red-400" :
                        sev === "yellow" ? "text-yellow-400" :
                        "text-green-400"
                      }>
                        {p.port}
                      </span>
                    </TableCell>
                    <TableCell className="w-20">
                      <Badge variant={p.protocol === "tcp" ? "default" : "secondary"} className="text-xs">
                        {p.protocol.toUpperCase()}
                      </Badge>
                    </TableCell>
                    <TableCell className="w-24">
                      <Badge
                        variant="outline"
                        className={`text-xs ${
                          p.state === "open" ? "text-green-400 border-green-400/30" :
                          p.state === "closed" ? "text-zinc-500 border-zinc-700" :
                          "text-yellow-400 border-yellow-400/30"
                        }`}
                      >
                        {p.state}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm text-zinc-300">{p.service ?? "-"}</TableCell>
                    <TableCell className="text-xs text-zinc-500">{p.version ?? "-"}</TableCell>
                    <TableCell className="text-xs text-zinc-500">{p.process ?? "-"}</TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}

interface PortTableProps {
  nodeId: string;
}

export function PortTable({ nodeId: _nodeId }: PortTableProps) {
  const rawScans = useNetworkStore((s) => s.scans);
  const scans = Array.isArray(rawScans) ? rawScans : [];
  const [filter, setFilter] = useState("");
  const [sortKey, setSortKey] = useState<SortKey>("port");
  const [sortDir, setSortDir] = useState<SortDir>("asc");

  const latestScan = scans[0];
  const allPorts = useMemo(
    () => parseResultsJson(latestScan?.results_json),
    [latestScan]
  );

  const groups = useMemo(() => {
    const map = new Map<string, PortEntry[]>();
    for (const p of allPorts) {
      const key = p.source ?? "Node";
      if (!map.has(key)) map.set(key, []);
      map.get(key)!.push(p);
    }
    return map;
  }, [allPorts]);

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    else { setSortKey(key); setSortDir("asc"); }
  };

  if (!latestScan) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-zinc-500">
        <p className="text-sm">Noch kein Scan durchgeführt.</p>
        <p className="text-xs mt-1">Starte einen Quick Scan oder Full Scan.</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {/* Filter + Sort controls */}
      <div className="flex items-center gap-3">
        <Input
          placeholder="Port, Service, Prozess filtern..."
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          className="max-w-xs bg-zinc-900 border-zinc-700 text-sm h-8"
        />
        <div className="flex items-center gap-1 ml-auto text-xs text-zinc-500">
          <span>Sortierung:</span>
          {(["port", "protocol", "state", "service"] as SortKey[]).map((k) => (
            <Button
              key={k}
              variant="ghost"
              size="sm"
              className={`h-6 px-2 text-xs gap-1 ${sortKey === k ? "text-zinc-200" : "text-zinc-500"}`}
              onClick={() => toggleSort(k)}
            >
              {k.charAt(0).toUpperCase() + k.slice(1)}
              {sortKey === k && <ArrowUpDown className="h-3 w-3" />}
            </Button>
          ))}
        </div>
      </div>

      {/* Table header labels (visual only) */}
      <div className="hidden md:grid grid-cols-[96px_80px_96px_1fr_1fr_1fr] gap-0 px-4 text-[10px] uppercase tracking-wide text-zinc-600">
        <span>Port</span>
        <span>Proto</span>
        <span>State</span>
        <span>Service</span>
        <span>Version</span>
        <span>Prozess</span>
      </div>

      {/* Grouped sections */}
      {groups.size === 0 ? (
        <div className="text-center py-8 text-zinc-600 text-sm">Keine Ports gefunden</div>
      ) : (
        Array.from(groups.entries()).map(([label, ports]) => (
          <PortGroup
            key={label}
            label={label}
            ports={ports}
            filter={filter}
            sortKey={sortKey}
            sortDir={sortDir}
          />
        ))
      )}

      <p className="text-xs text-zinc-600">
        {allPorts.length} Ports total · Letzter Scan:{" "}
        {latestScan.started_at
          ? new Date(latestScan.started_at).toLocaleString("de-DE")
          : "—"}
      </p>
    </div>
  );
}
