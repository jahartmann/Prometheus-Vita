import type { Node, NodeStatus } from "@/types/api";

export type AttentionSeverity = "critical" | "warning" | "info";

export interface AttentionItem {
  id: string;
  severity: AttentionSeverity;
  title: string;
  description: string;
  href: string;
}

export interface DashboardSummary {
  onlineNodes: number;
  offlineNodes: number;
  totalNodes: number;
  totalWorkloads: number;
  runningWorkloads: number;
  avgCpu: number;
  avgMemory: number;
  pendingStatusNodes: number;
  healthLabel: string;
  healthTone: "ok" | "warning" | "critical" | "info" | "muted";
  attentionItems: AttentionItem[];
}

export function buildDashboardSummary(
  nodes: Node[],
  nodeStatus: Record<string, NodeStatus | undefined>
): DashboardSummary {
  const onlineNodeList = nodes.filter((node) => node.is_online);
  const onlineNodes = onlineNodeList.length;
  const offlineNodes = nodes.length - onlineNodes;
  const statuses = onlineNodeList.map((node) => nodeStatus[node.id]).filter(Boolean) as NodeStatus[];
  const pendingStatusNodes = onlineNodes - statuses.length;

  const totalWorkloads = statuses.reduce(
    (sum, status) => sum + status.vm_count + status.ct_count,
    0
  );
  const runningWorkloads = statuses.reduce(
    (sum, status) => sum + status.vm_running + status.ct_running,
    0
  );
  const avgCpu = average(statuses.map((status) => status.cpu_usage));
  const memoryUsageValues = statuses
    .filter((status) => status.memory_total > 0)
    .map((status) => (status.memory_used / status.memory_total) * 100);
  const avgMemory = average(memoryUsageValues);

  const attentionItems = buildAttentionItems(
    nodes,
    statuses,
    offlineNodes,
    pendingStatusNodes,
    avgCpu,
    avgMemory
  );
  const criticalCount = attentionItems.filter((item) => item.severity === "critical").length;
  const warningCount = attentionItems.filter((item) => item.severity === "warning").length;
  const isEmpty = nodes.length === 0;

  return {
    onlineNodes,
    offlineNodes,
    totalNodes: nodes.length,
    totalWorkloads,
    runningWorkloads,
    avgCpu,
    avgMemory,
    pendingStatusNodes,
    healthLabel:
      isEmpty
        ? "Keine Nodes konfiguriert"
        : criticalCount > 0
        ? `${criticalCount} kritisch`
        : warningCount > 0
        ? `${warningCount} Hinweise`
        : pendingStatusNodes > 0
        ? "Status wird geladen"
        : "Cluster operativ",
    healthTone: isEmpty
      ? "muted"
      : criticalCount > 0
      ? "critical"
      : warningCount > 0
      ? "warning"
      : pendingStatusNodes > 0
      ? "info"
      : "ok",
    attentionItems,
  };
}

function buildAttentionItems(
  nodes: Node[],
  statuses: NodeStatus[],
  offlineNodes: number,
  pendingStatusNodes: number,
  avgCpu: number,
  avgMemory: number
): AttentionItem[] {
  const items: AttentionItem[] = [];

  if (offlineNodes > 0) {
    items.push({
      id: "offline-nodes",
      severity: "critical",
      title: `${offlineNodes} Node${offlineNodes === 1 ? "" : "s"} offline`,
      description: "Prüfen Sie Erreichbarkeit, Token und Netzwerkpfad.",
      href: "/nodes",
    });
  }

  if (pendingStatusNodes > 0) {
    items.push({
      id: "status-loading",
      severity: "info",
      title: `${pendingStatusNodes} Statusabfrage${pendingStatusNodes === 1 ? "" : "n"} ausstehend`,
      description: "Metriken werden geladen, bevor die Lage als ruhig bewertet wird.",
      href: "/monitoring",
    });
  }

  const hotNodes = statuses.filter((status) => status.cpu_usage >= 80);
  if (hotNodes.length > 0) {
    items.push({
      id: "cpu-pressure",
      severity: "warning",
      title: `${hotNodes.length} Node${hotNodes.length === 1 ? "" : "s"} mit hoher CPU`,
      description: `Cluster-Durchschnitt ${avgCpu.toFixed(1)} Prozent.`,
      href: "/monitoring",
    });
  }

  if (avgMemory >= 80) {
    items.push({
      id: "memory-pressure",
      severity: "warning",
      title: "RAM-Auslastung erhöht",
      description: `Durchschnittlich ${avgMemory.toFixed(1)} Prozent belegt.`,
      href: "/monitoring",
    });
  }

  if (items.length === 0 && nodes.length > 0) {
    items.push({
      id: "all-clear",
      severity: "info",
      title: "Keine akute Aufmerksamkeit",
      description: "Alle bekannten Nodes melden einen stabilen Grundzustand.",
      href: "/monitoring",
    });
  }

  if (nodes.length === 0) {
    items.push({
      id: "no-nodes",
      severity: "info",
      title: "Noch keine Nodes konfiguriert",
      description: "Fügen Sie den ersten Proxmox Node in den Einstellungen hinzu.",
      href: "/settings/nodes",
    });
  }

  return items;
}

function average(values: number[]): number {
  if (values.length === 0) return 0;
  return values.reduce((sum, value) => sum + value, 0) / values.length;
}
