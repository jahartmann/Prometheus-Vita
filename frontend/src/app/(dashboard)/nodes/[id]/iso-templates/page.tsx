"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { ArrowLeft, Disc } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useNodeStore } from "@/stores/node-store";
import { Skeleton } from "@/components/ui/skeleton";
import { isoApi, nodeApi, toArray } from "@/lib/api";
import type { Node, StorageContent } from "@/types/api";
import { formatBytes } from "@/lib/utils";

export default function NodeISOTemplatesPage() {
  const params = useParams<{ id: string }>();
  const nodeId = params.id;
  const { nodes, fetchNodes } = useNodeStore();
  const [isos, setIsos] = useState<StorageContent[]>([]);
  const [templates, setTemplates] = useState<StorageContent[]>([]);
  const [allNodes, setAllNodes] = useState<Node[]>([]);
  const [syncDialogOpen, setSyncDialogOpen] = useState(false);
  const [syncType, setSyncType] = useState<"iso" | "template">("iso");
  const [syncSourceNode, setSyncSourceNode] = useState("");
  const [syncSourceContent, setSyncSourceContent] = useState<StorageContent[]>([]);
  const [syncSelectedVolid, setSyncSelectedVolid] = useState("");
  const [syncTargetStorage, setSyncTargetStorage] = useState("local");
  const [syncLoading, setSyncLoading] = useState(false);

  useEffect(() => {
    if (nodes.length === 0) fetchNodes();
  }, [nodes.length, fetchNodes]);

  useEffect(() => {
    if (nodeId) {
      isoApi.listISOs(nodeId).then((res) => setIsos(toArray(res.data))).catch(() => {});
      isoApi.listTemplates(nodeId).then((res) => setTemplates(toArray(res.data))).catch(() => {});
      nodeApi.list().then((res) => setAllNodes(toArray(res.data))).catch(() => {});
    }
  }, [nodeId]);

  const node = nodes.find((n) => n.id === nodeId);
  if (!node) return <Skeleton className="h-64 w-full" />;

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link href={`/nodes/${nodeId}`}>
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <h1 className="text-2xl font-bold">ISOs & Templates - {node.name}</h1>
      </div>

      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">ISO Images</h2>
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            setSyncType("iso");
            setSyncDialogOpen(true);
            setSyncSourceNode("");
            setSyncSourceContent([]);
            setSyncSelectedVolid("");
          }}
        >
          <Disc className="mr-2 h-4 w-4" />
          Von Node synchronisieren
        </Button>
      </div>
      <Card>
        <CardContent className="p-0">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b">
                <th className="p-3 text-left font-medium">Name</th>
                <th className="p-3 text-left font-medium">Format</th>
                <th className="p-3 text-right font-medium">Größe</th>
                <th className="p-3 text-right font-medium">Datum</th>
              </tr>
            </thead>
            <tbody>
              {isos.length === 0 ? (
                <tr>
                  <td colSpan={4} className="p-6 text-center text-muted-foreground">
                    Keine ISO Images gefunden
                  </td>
                </tr>
              ) : (
                isos.map((iso) => (
                  <tr key={iso.volid} className="border-b last:border-0">
                    <td className="p-3 font-mono text-xs">{iso.volid}</td>
                    <td className="p-3">{iso.format}</td>
                    <td className="p-3 text-right">{formatBytes(iso.size)}</td>
                    <td className="p-3 text-right">
                      {new Date(iso.ctime * 1000).toLocaleDateString("de-DE")}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </CardContent>
      </Card>

      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Container Templates</h2>
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            setSyncType("template");
            setSyncDialogOpen(true);
            setSyncSourceNode("");
            setSyncSourceContent([]);
            setSyncSelectedVolid("");
          }}
        >
          <Disc className="mr-2 h-4 w-4" />
          Von Node synchronisieren
        </Button>
      </div>
      <Card>
        <CardContent className="p-0">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b">
                <th className="p-3 text-left font-medium">Name</th>
                <th className="p-3 text-left font-medium">Format</th>
                <th className="p-3 text-right font-medium">Größe</th>
                <th className="p-3 text-right font-medium">Datum</th>
              </tr>
            </thead>
            <tbody>
              {templates.length === 0 ? (
                <tr>
                  <td colSpan={4} className="p-6 text-center text-muted-foreground">
                    Keine Container Templates gefunden
                  </td>
                </tr>
              ) : (
                templates.map((tpl) => (
                  <tr key={tpl.volid} className="border-b last:border-0">
                    <td className="p-3 font-mono text-xs">{tpl.volid}</td>
                    <td className="p-3">{tpl.format}</td>
                    <td className="p-3 text-right">{formatBytes(tpl.size)}</td>
                    <td className="p-3 text-right">
                      {new Date(tpl.ctime * 1000).toLocaleDateString("de-DE")}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </CardContent>
      </Card>

      {syncDialogOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <Card className="w-full max-w-lg">
            <CardHeader>
              <CardTitle>
                {syncType === "iso" ? "ISO" : "Template"} von anderem Node synchronisieren
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <label className="text-sm font-medium">Quell-Node</label>
                <select
                  className="mt-1 w-full rounded-md border bg-background px-3 py-2 text-sm"
                  value={syncSourceNode}
                  onChange={(e) => {
                    const sourceId = e.target.value;
                    setSyncSourceNode(sourceId);
                    setSyncSelectedVolid("");
                    if (sourceId) {
                      const fetchFn = syncType === "iso" ? isoApi.listISOs : isoApi.listTemplates;
                      fetchFn(sourceId)
                        .then((res) => setSyncSourceContent(toArray(res.data)))
                        .catch(() => setSyncSourceContent([]));
                    } else {
                      setSyncSourceContent([]);
                    }
                  }}
                >
                  <option value="">Node auswählen...</option>
                  {allNodes
                    .filter((n) => n.id !== nodeId)
                    .map((n) => (
                      <option key={n.id} value={n.id}>
                        {n.name} ({n.hostname})
                      </option>
                    ))}
                </select>
              </div>

              {syncSourceContent.length > 0 && (
                <div>
                  <label className="text-sm font-medium">Inhalt</label>
                  <select
                    className="mt-1 w-full rounded-md border bg-background px-3 py-2 text-sm"
                    value={syncSelectedVolid}
                    onChange={(e) => setSyncSelectedVolid(e.target.value)}
                  >
                    <option value="">Datei auswählen...</option>
                    {syncSourceContent.map((c) => (
                      <option key={c.volid} value={c.volid}>
                        {c.volid} ({formatBytes(c.size)})
                      </option>
                    ))}
                  </select>
                </div>
              )}

              <div>
                <label className="text-sm font-medium">Ziel-Storage</label>
                <input
                  className="mt-1 w-full rounded-md border bg-background px-3 py-2 text-sm"
                  value={syncTargetStorage}
                  onChange={(e) => setSyncTargetStorage(e.target.value)}
                  placeholder="local"
                />
              </div>

              <div className="flex justify-end gap-2">
                <Button variant="outline" onClick={() => setSyncDialogOpen(false)}>
                  Abbrechen
                </Button>
                <Button
                  disabled={!syncSourceNode || !syncSelectedVolid || syncLoading}
                  onClick={async () => {
                    setSyncLoading(true);
                    try {
                      await isoApi.syncContent(nodeId, {
                        source_node_id: syncSourceNode,
                        volid: syncSelectedVolid,
                        target_storage: syncTargetStorage || "local",
                      });
                      setSyncDialogOpen(false);
                      isoApi.listISOs(nodeId).then((res) => setIsos(toArray(res.data))).catch(() => {});
                      isoApi.listTemplates(nodeId).then((res) => setTemplates(toArray(res.data))).catch(() => {});
                    } catch {
                      // Error handling via interceptor
                    } finally {
                      setSyncLoading(false);
                    }
                  }}
                >
                  {syncLoading ? "Synchronisiere..." : "Synchronisieren"}
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
