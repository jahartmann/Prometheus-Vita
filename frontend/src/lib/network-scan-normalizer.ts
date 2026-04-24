export type PortRisk = "high" | "medium" | "low" | "info";

export interface NormalizedPortEntry {
  id: string;
  port: number;
  protocol: string;
  state: string;
  service?: string;
  version?: string;
  process?: string;
  source: string;
  sourceType: "node" | "device" | "vm" | "connection";
  localAddr?: string;
  peerAddr?: string;
  peerPort?: number;
  risk: PortRisk;
  riskReason: string;
}

export interface NormalizedScanSummary {
  ports: NormalizedPortEntry[];
  listeningCount: number;
  connectionCount: number;
  highRiskCount: number;
  mediumRiskCount: number;
  unknownServiceCount: number;
}

const HIGH_RISK_PORTS = new Set([
  21, 22, 23, 25, 53, 111, 135, 139, 389, 445, 3389, 5432, 5900, 6379,
  8006, 8086, 9200, 9300, 11211, 27017,
]);

const WELL_KNOWN_SERVICES: Record<number, string> = {
  22: "ssh",
  25: "smtp",
  53: "dns",
  80: "http",
  443: "https",
  445: "smb",
  3389: "rdp",
  5432: "postgres",
  6379: "redis",
  8006: "proxmox",
  8080: "http-alt",
  8443: "https-alt",
  27017: "mongodb",
};

function asRecord(value: unknown): Record<string, unknown> {
  return value && typeof value === "object" ? (value as Record<string, unknown>) : {};
}

function asArray(value: unknown): Record<string, unknown>[] {
  return Array.isArray(value) ? value.map(asRecord) : [];
}

function text(value: unknown): string | undefined {
  return typeof value === "string" && value.trim() ? value : undefined;
}

function numberValue(value: unknown): number {
  const n = Number(value);
  return Number.isFinite(n) ? n : 0;
}

export function getPortRisk(entry: {
  port: number;
  state?: string;
  service?: string;
  version?: string;
  sourceType?: string;
}): { risk: PortRisk; reason: string } {
  const state = (entry.state ?? "").toLowerCase();
  if (entry.sourceType === "connection") {
    return { risk: "info", reason: "Bestehende Verbindung, kein Listening-Port" };
  }
  if (state && state !== "open" && state !== "listen" && state !== "listening" && state !== "unconn") {
    return { risk: "info", reason: "Port ist nicht offen" };
  }
  if (HIGH_RISK_PORTS.has(entry.port)) {
    return { risk: "high", reason: "Administrations-, Datenbank- oder Infrastruktur-Port" };
  }
  if (!entry.service && !WELL_KNOWN_SERVICES[entry.port]) {
    return { risk: "medium", reason: "Offener Port ohne bekannte Dienstzuordnung" };
  }
  if (entry.service && !entry.version && entry.port !== 80 && entry.port !== 443) {
    return { risk: "medium", reason: "Dienst erkannt, aber ohne Versionsinformation" };
  }
  return { risk: "low", reason: "Bekannter Dienst mit niedriger MVP-Risikoeinstufung" };
}

function entryFromSocket(
  raw: Record<string, unknown>,
  source: string,
  sourceType: NormalizedPortEntry["sourceType"],
  index: number
): NormalizedPortEntry {
  const port = numberValue(raw.port ?? raw.local_port);
  const protocol = (text(raw.protocol ?? raw.proto) ?? "tcp").toLowerCase();
  const state = text(raw.state) ?? (sourceType === "connection" ? "established" : "open");
  const service = text(raw.service) ?? WELL_KNOWN_SERVICES[port];
  const version = text(raw.version);
  const risk = getPortRisk({ port, state, service, version, sourceType });
  return {
    id: `${sourceType}-${source}-${protocol}-${port}-${index}`,
    port,
    protocol,
    state,
    service,
    version,
    process: text(raw.process),
    source,
    sourceType,
    localAddr: text(raw.local_addr),
    peerAddr: text(raw.peer_addr),
    peerPort: numberValue(raw.peer_port),
    risk: risk.risk,
    riskReason: risk.reason,
  };
}

function entryFromFullScan(raw: Record<string, unknown>, source: string, index: number): NormalizedPortEntry {
  const port = numberValue(raw.port ?? raw.portid);
  const protocol = (text(raw.protocol) ?? "tcp").toLowerCase();
  const state = text(raw.state) ?? "open";
  const service = text(raw.service ?? raw.service_name) ?? WELL_KNOWN_SERVICES[port];
  const version = text(raw.version ?? raw.service_version);
  const risk = getPortRisk({ port, state, service, version, sourceType: "node" });
  return {
    id: `full-${source}-${protocol}-${port}-${index}`,
    port,
    protocol,
    state,
    service,
    version,
    source,
    sourceType: source.startsWith("VM ") ? "vm" : "node",
    risk: risk.risk,
    riskReason: risk.reason,
  };
}

export function normalizeNetworkScanResults(results: unknown): NormalizedScanSummary {
  const obj = asRecord(results);
  const ports: NormalizedPortEntry[] = [];

  asArray(obj.listening_tcp).forEach((raw, index) => {
    ports.push(entryFromSocket(raw, "Node TCP", "node", index));
  });
  asArray(obj.listening_udp).forEach((raw, index) => {
    ports.push(entryFromSocket(raw, "Node UDP", "node", index));
  });
  asArray(obj.established).forEach((raw, index) => {
    ports.push(entryFromSocket(raw, "Established", "connection", index));
  });
  asArray(obj.ports).forEach((raw, index) => {
    ports.push(entryFromFullScan(raw, "Full Scan", index));
  });
  asArray(obj.vm_ports).forEach((raw, index) => {
    const vmid = text(raw.vmid) ?? String(raw.vmid ?? "");
    ports.push(entryFromFullScan(raw, `VM ${vmid}`.trim(), index));
  });
  asArray(obj.nmap_results).forEach((host, hostIndex) => {
    const source = text(host.ip ?? host.address) ?? `Host ${hostIndex + 1}`;
    asArray(host.ports).forEach((raw, index) => {
      ports.push(entryFromFullScan(raw, source, index));
    });
  });

  return {
    ports,
    listeningCount: ports.filter((p) => p.sourceType !== "connection").length,
    connectionCount: ports.filter((p) => p.sourceType === "connection").length,
    highRiskCount: ports.filter((p) => p.risk === "high").length,
    mediumRiskCount: ports.filter((p) => p.risk === "medium").length,
    unknownServiceCount: ports.filter((p) => !p.service && p.sourceType !== "connection").length,
  };
}
