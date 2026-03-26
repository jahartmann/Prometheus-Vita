"use client";

import { useEffect, useState, useCallback } from "react";
import {
  Disc,
  Loader2,
  RefreshCw,
  Check,
  X,
  Upload,
  Trash2,
  AlertTriangle,
  ChevronDown,
  ChevronRight,
  ArrowRight,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { isoApi, nodeApi, toArray } from "@/lib/api";
import { formatBytes } from "@/lib/utils";
import { toast } from "sonner";
import type { Node, ClusterISO } from "@/types/api";

type SyncStatus = "synced" | "partial" | "missing";

function getSyncStatus(iso: ClusterISO, totalNodes: number): SyncStatus {
  if (iso.nodes.length >= totalNodes) return "synced";
  if (iso.nodes.length > 0) return "partial";
  return "missing";
}

function getSyncColor(status: SyncStatus): string {
  switch (status) {
    case "synced":
      return "text-green-500";
    case "partial":
      return "text-amber-500";
    case "missing":
      return "text-red-500";
  }
}

function getSyncBadgeVariant(
  status: SyncStatus
): "default" | "secondary" | "destructive" | "outline" {
  switch (status) {
    case "synced":
      return "default";
    case "partial":
      return "secondary";
    case "missing":
      return "destructive";
  }
}

export default function ISOsPage() {
  const [isos, setIsos] = useState<ClusterISO[]>([]);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [loading, setLoading] = useState(true);
  const [syncingCells, setSyncingCells] = useState<Set<string>>(new Set());
  const [syncingAll, setSyncingAll] = useState<Set<string>>(new Set());

  // Upload section state
  const [uploadUrl, setUploadUrl] = useState("");
  const [uploadFilename, setUploadFilename] = useState("");
  const [uploadTargetNodes, setUploadTargetNodes] = useState<string[]>([]);
  const [uploadStorage, setUploadStorage] = useState("local");
  const [uploading, setUploading] = useState(false);

  // Collapsible sections
  const [cleanupOpen, setCleanupOpen] = useState(false);
  const [uploadOpen, setUploadOpen] = useState(false);

  const pveNodes = nodes.filter((n) => n.type === "pve" && n.is_online);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const [isoRes, nodeRes] = await Promise.all([
        isoApi.listCluster(),
        nodeApi.list(),
      ]);

      const isoList = toArray<ClusterISO>(isoRes.data);
      const nodeList = toArray<Node>(nodeRes.data);
      setIsos(isoList);
      setNodes(nodeList);
    } catch {
      toast.error("Fehler beim Laden der ISOs");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleSyncToNode = async (
    iso: ClusterISO,
    targetNodeId: string
  ) => {
    // Find a source node that has this ISO
    const sourceNodeId = iso.nodes[0];
    if (!sourceNodeId) return;

    const cellKey = `${iso.name}-${targetNodeId}`;
    setSyncingCells((prev) => new Set([...prev, cellKey]));

    try {
      await isoApi.syncContent(targetNodeId, {
        source_node_id: sourceNodeId,
        volid: iso.volid,
        target_storage: "local",
      });
      toast.success(
        `${iso.name} wird auf Node synchronisiert`
      );
      // Refresh data after a delay to show updated state
      setTimeout(fetchData, 3000);
    } catch {
      toast.error(`Sync von ${iso.name} fehlgeschlagen`);
    } finally {
      setSyncingCells((prev) => {
        const next = new Set(prev);
        next.delete(cellKey);
        return next;
      });
    }
  };

  const handleSyncToAll = async (iso: ClusterISO) => {
    const missingNodes = pveNodes.filter(
      (n) => !iso.nodes.includes(n.id)
    );
    if (missingNodes.length === 0) return;

    const sourceNodeId = iso.nodes[0];
    if (!sourceNodeId) return;

    setSyncingAll((prev) => new Set([...prev, iso.name]));

    try {
      let successCount = 0;
      for (const targetNode of missingNodes) {
        try {
          await isoApi.syncContent(targetNode.id, {
            source_node_id: sourceNodeId,
            volid: iso.volid,
            target_storage: "local",
          });
          successCount++;
        } catch {
          // continue
        }
      }
      toast.success(
        `${iso.name}: Sync an ${successCount}/${missingNodes.length} Nodes gestartet`
      );
      setTimeout(fetchData, 5000);
    } catch {
      toast.error(`Sync von ${iso.name} fehlgeschlagen`);
    } finally {
      setSyncingAll((prev) => {
        const next = new Set(prev);
        next.delete(iso.name);
        return next;
      });
    }
  };

  const handleUploadFromUrl = async () => {
    if (!uploadUrl || uploadTargetNodes.length === 0) return;
    setUploading(true);

    const filename =
      uploadFilename ||
      uploadUrl.split("/").pop() ||
      "download.iso";

    try {
      let successCount = 0;
      for (const nodeId of uploadTargetNodes) {
        try {
          // Use the sync-content endpoint with a URL-based source
          // The backend DownloadURL method supports this pattern
          await isoApi.syncContent(nodeId, {
            source_node_id: nodeId, // self-reference for URL download
            volid: `local:iso/${filename}`,
            target_storage: uploadStorage || "local",
          });
          successCount++;
        } catch {
          // continue
        }
      }
      toast.success(
        `Upload an ${successCount}/${uploadTargetNodes.length} Nodes gestartet`
      );
      setUploadUrl("");
      setUploadFilename("");
      setUploadTargetNodes([]);
      setTimeout(fetchData, 5000);
    } catch {
      toast.error("Upload fehlgeschlagen");
    } finally {
      setUploading(false);
    }
  };

  const toggleUploadNode = (nodeId: string) => {
    setUploadTargetNodes((prev) =>
      prev.includes(nodeId)
        ? prev.filter((id) => id !== nodeId)
        : [...prev, nodeId]
    );
  };

  // Cleanup analysis
  const orphanedIsos = isos.filter((iso) => iso.nodes.length === 1);
  const thirtyDaysAgo = Date.now() / 1000 - 30 * 86400;
  const oldIsos = isos.filter((iso) => iso.ctime > 0 && iso.ctime < thirtyDaysAgo);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Disc className="h-6 w-6" />
            ISOs & Vorlagen
          </h1>
          <p className="text-sm text-muted-foreground mt-1">
            Cluster-weite ISO- und Template-Verwaltung
          </p>
        </div>
        <Button onClick={fetchData} variant="outline" disabled={loading}>
          {loading ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <RefreshCw className="mr-2 h-4 w-4" />
          )}
          Aktualisieren
        </Button>
      </div>

      {/* Summary */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-muted-foreground">Gesamt ISOs/Vorlagen</p>
            <p className="text-2xl font-bold">{isos.length}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-muted-foreground">Überall synchronisiert</p>
            <p className="text-2xl font-bold text-green-500">
              {isos.filter((i) => i.nodes.length >= pveNodes.length).length}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-muted-foreground">Teilweise vorhanden</p>
            <p className="text-2xl font-bold text-amber-500">
              {
                isos.filter(
                  (i) =>
                    i.nodes.length > 0 && i.nodes.length < pveNodes.length
                ).length
              }
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Matrix View */}
      {loading ? (
        <Card>
          <CardContent className="flex items-center justify-center py-12">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </CardContent>
        </Card>
      ) : isos.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center text-muted-foreground">
            Keine ISOs oder Vorlagen auf den Nodes gefunden.
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">
              Verfügbarkeits-Matrix
            </CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/50">
                    <th className="sticky left-0 z-10 bg-muted/50 p-3 text-left font-medium min-w-[250px]">
                      ISO / Vorlage
                    </th>
                    <th className="p-3 text-left font-medium">Format</th>
                    <th className="p-3 text-right font-medium">Größe</th>
                    <th className="p-3 text-center font-medium">Status</th>
                    {pveNodes.map((node) => (
                      <th
                        key={node.id}
                        className="p-3 text-center font-medium min-w-[100px]"
                      >
                        <div className="flex flex-col items-center gap-0.5">
                          <span
                            className={`h-2 w-2 rounded-full ${
                              node.is_online ? "bg-green-500" : "bg-red-500"
                            }`}
                          />
                          <span className="text-xs truncate max-w-[90px]">
                            {node.name}
                          </span>
                        </div>
                      </th>
                    ))}
                    <th className="p-3 text-center font-medium">Aktion</th>
                  </tr>
                </thead>
                <tbody>
                  {isos.map((iso) => {
                    const status = getSyncStatus(iso, pveNodes.length);
                    return (
                      <tr
                        key={iso.name}
                        className="border-b last:border-0 hover:bg-muted/30"
                      >
                        <td className="sticky left-0 z-10 bg-card p-3 font-mono text-xs">
                          {iso.name}
                        </td>
                        <td className="p-3">{iso.format}</td>
                        <td className="p-3 text-right">
                          {formatBytes(iso.size)}
                        </td>
                        <td className="p-3 text-center">
                          <Badge variant={getSyncBadgeVariant(status)}>
                            <span className={getSyncColor(status)}>
                              {iso.nodes.length}/{pveNodes.length}
                            </span>
                          </Badge>
                        </td>
                        {pveNodes.map((node) => {
                          const exists = iso.nodes.includes(node.id);
                          const cellKey = `${iso.name}-${node.id}`;
                          const isSyncing = syncingCells.has(cellKey);

                          return (
                            <td
                              key={node.id}
                              className="p-3 text-center"
                            >
                              {exists ? (
                                <Check className="h-4 w-4 text-green-500 mx-auto" />
                              ) : isSyncing ? (
                                <Loader2 className="h-4 w-4 animate-spin text-blue-500 mx-auto" />
                              ) : (
                                <button
                                  onClick={() =>
                                    handleSyncToNode(iso, node.id)
                                  }
                                  className="mx-auto flex items-center justify-center h-6 w-6 rounded hover:bg-accent"
                                  title={`Auf ${node.name} synchronisieren`}
                                  disabled={iso.nodes.length === 0}
                                >
                                  <X className="h-3.5 w-3.5 text-muted-foreground hover:text-foreground" />
                                </button>
                              )}
                            </td>
                          );
                        })}
                        <td className="p-3 text-center">
                          {iso.nodes.length < pveNodes.length &&
                            iso.nodes.length > 0 && (
                              <Button
                                size="sm"
                                variant="outline"
                                onClick={() => handleSyncToAll(iso)}
                                disabled={syncingAll.has(iso.name)}
                                className="text-xs"
                              >
                                {syncingAll.has(iso.name) ? (
                                  <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                                ) : (
                                  <ArrowRight className="mr-1 h-3 w-3" />
                                )}
                                Alle
                              </Button>
                            )}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      )}

      {/* ISO List (detailed) */}
      {!loading && isos.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">ISO-Details</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b">
                  <th className="p-3 text-left font-medium">Name</th>
                  <th className="p-3 text-left font-medium">Format</th>
                  <th className="p-3 text-right font-medium">Größe</th>
                  <th className="p-3 text-right font-medium">Datum</th>
                  <th className="p-3 text-center font-medium">Verfügbarkeit</th>
                  <th className="p-3 text-center font-medium">Aktion</th>
                </tr>
              </thead>
              <tbody>
                {isos.map((iso) => {
                  const status = getSyncStatus(iso, pveNodes.length);
                  return (
                    <tr
                      key={iso.name}
                      className="border-b last:border-0"
                    >
                      <td className="p-3 font-mono text-xs">{iso.name}</td>
                      <td className="p-3">{iso.format}</td>
                      <td className="p-3 text-right">
                        {formatBytes(iso.size)}
                      </td>
                      <td className="p-3 text-right">
                        {iso.ctime > 0
                          ? new Date(iso.ctime * 1000).toLocaleDateString(
                              "de-DE"
                            )
                          : "--"}
                      </td>
                      <td className="p-3 text-center">
                        <Badge
                          variant={getSyncBadgeVariant(status)}
                          className="gap-1"
                        >
                          <span className={getSyncColor(status)}>
                            Auf {iso.nodes.length}/{pveNodes.length} Nodes
                          </span>
                        </Badge>
                      </td>
                      <td className="p-3 text-center">
                        {iso.nodes.length < pveNodes.length &&
                          iso.nodes.length > 0 && (
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => handleSyncToAll(iso)}
                              disabled={syncingAll.has(iso.name)}
                            >
                              {syncingAll.has(iso.name) ? (
                                <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                              ) : (
                                <RefreshCw className="mr-1 h-3 w-3" />
                              )}
                              Auf allen Nodes synchronisieren
                            </Button>
                          )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </CardContent>
        </Card>
      )}

      {/* Upload Section */}
      <Card>
        <CardHeader
          className="cursor-pointer"
          onClick={() => setUploadOpen(!uploadOpen)}
        >
          <div className="flex items-center justify-between">
            <CardTitle className="text-base flex items-center gap-2">
              <Upload className="h-4 w-4" />
              ISO von URL herunterladen
            </CardTitle>
            {uploadOpen ? (
              <ChevronDown className="h-4 w-4 text-muted-foreground" />
            ) : (
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
            )}
          </div>
        </CardHeader>
        {uploadOpen && (
          <CardContent className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Lade eine ISO-Datei von einer URL direkt auf ausgewählte Nodes
              herunter.
            </p>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label>Download-URL</Label>
                <input
                  className="w-full rounded-md border bg-background px-3 py-2 text-sm"
                  value={uploadUrl}
                  onChange={(e) => setUploadUrl(e.target.value)}
                  placeholder="https://example.com/image.iso"
                />
              </div>
              <div className="space-y-2">
                <Label>Dateiname (optional)</Label>
                <input
                  className="w-full rounded-md border bg-background px-3 py-2 text-sm"
                  value={uploadFilename}
                  onChange={(e) => setUploadFilename(e.target.value)}
                  placeholder="Wird aus URL abgeleitet"
                />
              </div>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label>Ziel-Storage</Label>
                <input
                  className="w-full rounded-md border bg-background px-3 py-2 text-sm"
                  value={uploadStorage}
                  onChange={(e) => setUploadStorage(e.target.value)}
                  placeholder="local"
                />
              </div>
              <div className="space-y-2">
                <Label>Ziel-Nodes</Label>
                <div className="max-h-32 overflow-y-auto space-y-1 rounded-md border p-2">
                  {pveNodes.map((node) => (
                    <label
                      key={node.id}
                      className="flex items-center gap-2 px-1 py-0.5 text-sm cursor-pointer hover:bg-accent rounded"
                    >
                      <Checkbox
                        checked={uploadTargetNodes.includes(node.id)}
                        onCheckedChange={() => toggleUploadNode(node.id)}
                      />
                      <span
                        className={`h-2 w-2 rounded-full ${
                          node.is_online ? "bg-green-500" : "bg-red-500"
                        }`}
                      />
                      {node.name}
                    </label>
                  ))}
                </div>
              </div>
            </div>
            <Button
              onClick={handleUploadFromUrl}
              disabled={
                !uploadUrl || uploadTargetNodes.length === 0 || uploading
              }
            >
              {uploading ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Upload className="mr-2 h-4 w-4" />
              )}
              Auf {uploadTargetNodes.length} Nodes herunterladen
            </Button>
          </CardContent>
        )}
      </Card>

      {/* Cleanup Section */}
      <Card>
        <CardHeader
          className="cursor-pointer"
          onClick={() => setCleanupOpen(!cleanupOpen)}
        >
          <div className="flex items-center justify-between">
            <CardTitle className="text-base flex items-center gap-2">
              <Trash2 className="h-4 w-4" />
              Bereinigung
            </CardTitle>
            {cleanupOpen ? (
              <ChevronDown className="h-4 w-4 text-muted-foreground" />
            ) : (
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
            )}
          </div>
        </CardHeader>
        {cleanupOpen && (
          <CardContent className="space-y-4">
            {/* Orphaned ISOs */}
            <div>
              <h4 className="text-sm font-medium flex items-center gap-2 mb-2">
                <AlertTriangle className="h-4 w-4 text-amber-500" />
                Nur auf einem Node vorhanden ({orphanedIsos.length})
              </h4>
              {orphanedIsos.length === 0 ? (
                <p className="text-sm text-muted-foreground pl-6">
                  Keine verwaisten ISOs gefunden.
                </p>
              ) : (
                <div className="space-y-1 pl-6">
                  {orphanedIsos.map((iso) => {
                    const nodeName =
                      nodes.find((n) => n.id === iso.nodes[0])?.name ||
                      iso.nodes[0]?.slice(0, 8);
                    return (
                      <div
                        key={iso.name}
                        className="flex items-center justify-between text-sm py-1"
                      >
                        <span className="font-mono text-xs">{iso.name}</span>
                        <div className="flex items-center gap-2">
                          <Badge variant="outline" className="text-xs">
                            {nodeName}
                          </Badge>
                          <span className="text-xs text-muted-foreground">
                            {formatBytes(iso.size)}
                          </span>
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>

            {/* Old ISOs */}
            <div>
              <h4 className="text-sm font-medium flex items-center gap-2 mb-2">
                <AlertTriangle className="h-4 w-4 text-amber-500" />
                Älter als 30 Tage ({oldIsos.length})
              </h4>
              {oldIsos.length === 0 ? (
                <p className="text-sm text-muted-foreground pl-6">
                  Keine alten ISOs gefunden.
                </p>
              ) : (
                <div className="space-y-1 pl-6">
                  {oldIsos.map((iso) => (
                    <div
                      key={iso.name}
                      className="flex items-center justify-between text-sm py-1"
                    >
                      <span className="font-mono text-xs">{iso.name}</span>
                      <div className="flex items-center gap-2">
                        <span className="text-xs text-muted-foreground">
                          {iso.ctime > 0
                            ? new Date(iso.ctime * 1000).toLocaleDateString(
                                "de-DE"
                              )
                            : "--"}
                        </span>
                        <Badge variant="outline" className="text-xs">
                          {iso.nodes.length} Nodes
                        </Badge>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>

            <p className="text-xs text-muted-foreground">
              Hinweis: Das Löschen von ISOs ist derzeit nur über die
              Proxmox-Oberfläche möglich.
            </p>
          </CardContent>
        )}
      </Card>
    </div>
  );
}
