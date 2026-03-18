"use client";

import { useMemo, useState } from "react";
import { useNetworkStore } from "@/stores/network-store";
import { Badge } from "@/components/ui/badge";
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
import { ChevronDown, ChevronRight, Wifi, WifiOff } from "lucide-react";

interface DeviceTableProps {
  nodeId: string;
}

function isNew(firstSeen: string): boolean {
  return Date.now() - new Date(firstSeen).getTime() < 24 * 60 * 60 * 1000;
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString("de-DE", {
    day: "2-digit",
    month: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function DeviceTable({ nodeId: _nodeId }: DeviceTableProps) {
  const rawDevices = useNetworkStore((s) => s.devices);
  const rawScans = useNetworkStore((s) => s.scans);
  const devices = Array.isArray(rawDevices) ? rawDevices : [];
  const scans = Array.isArray(rawScans) ? rawScans : [];

  const [filter, setFilter] = useState("");
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const latestScanTime = scans[0]?.completed_at ?? scans[0]?.started_at;

  // Build set of IPs seen in latest scan
  const latestScanIPs = useMemo<Set<string>>(() => {
    if (!scans[0]?.results_json) return new Set();
    const obj = scans[0].results_json as Record<string, unknown>;
    const ips = new Set<string>();
    if (Array.isArray(obj.nmap_results)) {
      for (const host of obj.nmap_results) {
        const h = host as Record<string, unknown>;
        if (h.ip) ips.add(String(h.ip));
        if (h.address) ips.add(String(h.address));
      }
    }
    return ips;
  }, [scans]);

  // Extract port counts per device from latest scan
  const portCounts = useMemo<Record<string, number>>(() => {
    const obj = scans[0]?.results_json as Record<string, unknown> | undefined;
    if (!obj) return {};
    const counts: Record<string, number> = {};
    if (Array.isArray(obj.nmap_results)) {
      for (const host of obj.nmap_results) {
        const h = host as Record<string, unknown>;
        const ip = String(h.ip ?? h.address ?? "");
        if (ip && Array.isArray(h.ports)) counts[ip] = h.ports.length;
      }
    }
    return counts;
  }, [scans]);

  const filtered = useMemo(() => {
    const q = filter.toLowerCase();
    return devices.filter((d) =>
      !q ||
      d.ip.includes(q) ||
      (d.mac ?? "").toLowerCase().includes(q) ||
      (d.hostname ?? "").toLowerCase().includes(q)
    );
  }, [devices, filter]);

  if (devices.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-zinc-500">
        <Wifi className="h-8 w-8 mb-3 opacity-30" />
        <p className="text-sm">Keine Geräte erkannt.</p>
        <p className="text-xs mt-1">Führe einen Full Scan durch, um Netzwerkgeräte zu entdecken.</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-3">
        <Input
          placeholder="IP, MAC, Hostname filtern..."
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          className="max-w-xs bg-zinc-900 border-zinc-700 text-sm h-8"
        />
        <span className="ml-auto text-xs text-zinc-500">{filtered.length} Geräte</span>
      </div>

      <div className="rounded-lg border border-zinc-800 overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow className="border-zinc-800">
              <TableHead className="text-zinc-400 w-36">IP</TableHead>
              <TableHead className="text-zinc-400 w-40">MAC</TableHead>
              <TableHead className="text-zinc-400">Hostname</TableHead>
              <TableHead className="text-zinc-400 w-20 text-center">Ports</TableHead>
              <TableHead className="text-zinc-400 w-36">Zuerst gesehen</TableHead>
              <TableHead className="text-zinc-400 w-36">Zuletzt gesehen</TableHead>
              <TableHead className="text-zinc-400 w-24">Status</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filtered.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} className="text-center text-zinc-600 py-8">
                  Keine Geräte gefunden
                </TableCell>
              </TableRow>
            ) : (
              filtered.map((device) => {
                const disappeared = latestScanTime
                  ? new Date(device.last_seen) < new Date(latestScanTime) && !latestScanIPs.has(device.ip)
                  : false;
                const newDevice = isNew(device.first_seen);
                const expanded = expandedId === device.id;
                const portCount = portCounts[device.ip] ?? 0;

                return (
                  <>
                    <TableRow
                      key={device.id}
                      className={`border-zinc-800/50 cursor-pointer hover:bg-zinc-800/30 transition-colors ${
                        disappeared ? "opacity-50" : ""
                      }`}
                      onClick={() => setExpandedId(expanded ? null : device.id)}
                    >
                      <TableCell className="font-mono text-sm text-zinc-300">
                        <div className="flex items-center gap-1.5">
                          {expanded ? (
                            <ChevronDown className="h-3 w-3 text-zinc-500" />
                          ) : (
                            <ChevronRight className="h-3 w-3 text-zinc-500" />
                          )}
                          {device.ip}
                        </div>
                      </TableCell>
                      <TableCell className="font-mono text-xs text-zinc-500">
                        {device.mac ?? "—"}
                      </TableCell>
                      <TableCell className="text-sm text-zinc-300">
                        {device.hostname ?? "—"}
                      </TableCell>
                      <TableCell className="text-center">
                        {portCount > 0 ? (
                          <Badge variant="secondary" className="text-xs">{portCount}</Badge>
                        ) : (
                          <span className="text-zinc-600 text-xs">—</span>
                        )}
                      </TableCell>
                      <TableCell className="text-xs text-zinc-500">
                        {formatDate(device.first_seen)}
                      </TableCell>
                      <TableCell className="text-xs text-zinc-500">
                        {formatDate(device.last_seen)}
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-wrap gap-1">
                          {newDevice && (
                            <Badge className="bg-blue-500/20 text-blue-400 border-blue-500/30 text-[10px] px-1.5">
                              Neu
                            </Badge>
                          )}
                          {disappeared ? (
                            <Badge variant="outline" className="text-red-400 border-red-400/30 text-[10px] px-1.5 gap-1">
                              <WifiOff className="h-2.5 w-2.5" />
                              Weg
                            </Badge>
                          ) : (
                            <Badge variant="outline" className="text-green-400 border-green-400/30 text-[10px] px-1.5 gap-1">
                              <Wifi className="h-2.5 w-2.5" />
                              Online
                            </Badge>
                          )}
                          {device.is_known && (
                            <Badge variant="outline" className="text-zinc-400 border-zinc-600 text-[10px] px-1.5">
                              Bekannt
                            </Badge>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>

                    {/* Inline expanded port list */}
                    {expanded && (
                      <TableRow key={`${device.id}-expanded`} className="border-zinc-800/30 bg-zinc-900/40">
                        <TableCell colSpan={7} className="py-3 px-6">
                          <DevicePortExpand ip={device.ip} scans={scans} />
                        </TableCell>
                      </TableRow>
                    )}
                  </>
                );
              })
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}

function DevicePortExpand({
  ip,
  scans,
}: {
  ip: string;
  scans: ReturnType<typeof useNetworkStore.getState>["scans"];
}) {
  const obj = scans[0]?.results_json as Record<string, unknown> | undefined;
  if (!obj) return <p className="text-xs text-zinc-600">Keine Scandaten vorhanden.</p>;

  const hosts = Array.isArray(obj.nmap_results) ? obj.nmap_results : [];
  const host = hosts.find((h) => {
    const hh = h as Record<string, unknown>;
    return hh.ip === ip || hh.address === ip;
  }) as Record<string, unknown> | undefined;

  if (!host || !Array.isArray(host.ports) || host.ports.length === 0) {
    return <p className="text-xs text-zinc-600">Keine offenen Ports gefunden.</p>;
  }

  return (
    <div className="space-y-1">
      <p className="text-xs font-medium text-zinc-400 mb-2">Offene Ports von {ip}</p>
      <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
        {(host.ports as Array<Record<string, unknown>>).map((p, i) => (
          <div key={i} className="flex items-center gap-2 text-xs bg-zinc-800/60 rounded px-2 py-1.5">
            <span className="font-mono font-bold text-green-400">{String(p.port ?? p.portid ?? "?")}</span>
            <span className="text-zinc-500">{String(p.protocol ?? "tcp").toUpperCase()}</span>
            <span className="text-zinc-400 truncate">{String(p.service ?? "—")}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
