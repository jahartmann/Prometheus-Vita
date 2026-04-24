"use client";

import { useMemo, useState } from "react";
import { useNetworkStore } from "@/stores/network-store";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { ServiceRiskBadge } from "@/components/network/service-risk-badge";
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
import {
  normalizeNetworkScanResults,
  type NormalizedPortEntry,
  type PortRisk,
} from "@/lib/network-scan-normalizer";

type SortKey = "port" | "protocol" | "state" | "service" | "risk";
type SortDir = "asc" | "desc";

const PORT_RISK_CLASSES: Record<PortRisk, string> = {
  high: "text-red-400",
  medium: "text-yellow-400",
  low: "text-green-400",
  info: "text-zinc-500",
};

const RISK_SORT_ORDER: Record<PortRisk, number> = {
  high: 0,
  medium: 1,
  low: 2,
  info: 3,
};

function endpointText(port: NormalizedPortEntry): string {
  const local = withPort(port.localAddr, port.port);
  const peer = port.peerAddr
    ? `${port.peerAddr}${port.peerPort ? `:${port.peerPort}` : ""}`
    : "";

  if (local && peer) return `${local} -> ${peer}`;
  if (local) return local;
  if (peer) return peer;
  return "-";
}

function withPort(address: string | undefined, port: number | undefined): string {
  if (!address) return "";
  if (!port) return address;
  return address.endsWith(`:${port}`) ? address : `${address}:${port}`;
}

interface PortGroupProps {
  label: string;
  ports: NormalizedPortEntry[];
  filter: string;
  sortKey: SortKey;
  sortDir: SortDir;
}

function PortGroup({ label, ports, filter, sortKey, sortDir }: PortGroupProps) {
  const [open, setOpen] = useState(true);

  const filtered = useMemo(() => {
    const q = filter.toLowerCase();
    return ports
      .filter((p) => {
        const peerPort = p.peerPort ? String(p.peerPort) : "";
        return (
          !q ||
          String(p.port).includes(q) ||
          (p.service ?? "").toLowerCase().includes(q) ||
          (p.protocol ?? "").toLowerCase().includes(q) ||
          (p.state ?? "").toLowerCase().includes(q) ||
          (p.process ?? "").toLowerCase().includes(q) ||
          (p.localAddr ?? "").toLowerCase().includes(q) ||
          (p.peerAddr ?? "").toLowerCase().includes(q) ||
          peerPort.includes(q) ||
          p.risk.toLowerCase().includes(q) ||
          p.riskReason.toLowerCase().includes(q)
        );
      })
      .sort((a, b) => {
        let cmp = 0;
        if (sortKey === "port") cmp = a.port - b.port;
        else if (sortKey === "protocol") cmp = a.protocol.localeCompare(b.protocol);
        else if (sortKey === "state") cmp = a.state.localeCompare(b.state);
        else if (sortKey === "service") cmp = (a.service ?? "").localeCompare(b.service ?? "");
        else if (sortKey === "risk") cmp = RISK_SORT_ORDER[a.risk] - RISK_SORT_ORDER[b.risk];
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
                return (
                  <TableRow key={p.id || `${p.port}-${p.protocol}-${i}`} className="border-zinc-800/50">
                    <TableCell className="font-mono font-bold text-sm w-24">
                      <span className={PORT_RISK_CLASSES[p.risk]}>
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
                    <TableCell className="text-xs text-zinc-500 font-mono">{endpointText(p)}</TableCell>
                    <TableCell className="w-28">
                      <ServiceRiskBadge risk={p.risk} reason={p.riskReason} />
                    </TableCell>
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

export function PortTable({ nodeId }: PortTableProps) {
  const rawScans = useNetworkStore((s) => s.scans);
  const scans = Array.isArray(rawScans)
    ? rawScans.filter((scan) => scan.node_id === nodeId)
    : [];
  const [filter, setFilter] = useState("");
  const [sortKey, setSortKey] = useState<SortKey>("port");
  const [sortDir, setSortDir] = useState<SortDir>("asc");

  const latestScan = scans[0];
  const normalized = useMemo(
    () => normalizeNetworkScanResults(latestScan?.results_json),
    [latestScan]
  );
  const allPorts = normalized.ports;

  const groups = useMemo(() => {
    const map = new Map<string, NormalizedPortEntry[]>();
    for (const p of allPorts) {
      const key = p.source;
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
          placeholder="Port, Service, Prozess, Endpunkt filtern..."
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          className="max-w-xs bg-zinc-900 border-zinc-700 text-sm h-8"
        />
        <div className="flex items-center gap-1 ml-auto text-xs text-zinc-500">
          <span>Sortierung:</span>
          {(["port", "protocol", "state", "service", "risk"] as SortKey[]).map((k) => (
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
      <div className="hidden md:grid grid-cols-[96px_80px_96px_1fr_1fr_1fr_1.25fr_112px] gap-0 px-4 text-[10px] uppercase tracking-wide text-zinc-600">
        <span>Port</span>
        <span>Proto</span>
        <span>State</span>
        <span>Service</span>
        <span>Version</span>
        <span>Prozess</span>
        <span>Endpunkt</span>
        <span>Risiko</span>
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
