"use client";

import { useEffect, useState } from "react";
import { useNodeStore } from "@/stores/node-store";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { CheckCircle2, Network, Server, XCircle } from "lucide-react";
import { sshKeyApi } from "@/lib/api";
import { toast } from "sonner";
import type { NodeTrustResult } from "@/types/api";

export default function SSHKeysPage() {
  const { nodes, fetchNodes } = useNodeStore();
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<NodeTrustResult | null>(null);

  useEffect(() => {
    fetchNodes();
  }, [fetchNodes]);

  const handleTrustAll = async () => {
    setLoading(true);
    setResult(null);
    try {
      const resp = await sshKeyApi.trustNodes();
      const data: NodeTrustResult = resp.data;
      setResult(data);
      const ok = data.distributed?.length ?? 0;
      const failed = data.failed?.length ?? 0;
      if (ok === 0 && failed === 0) {
        toast.info("Keine Node-Paare gefunden");
      } else if (failed === 0) {
        toast.success(`Vertrauen eingerichtet (${ok} Verbindungen)`);
      } else {
        toast.warning(`${ok} erfolgreich, ${failed} fehlgeschlagen`);
      }
    } catch {
      toast.error("Fehler beim Einrichten des Vertrauens");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-lg font-semibold">SSH-Vertrauen</h2>
        <p className="text-sm text-muted-foreground">
          Verteilt den SSH-Public-Key jeder Node auf alle anderen — entspricht{" "}
          <code className="text-xs bg-muted px-1 py-0.5 rounded">ssh-copy-id root@IP</code>{" "}
          mit den gespeicherten Zugangsdaten.
        </p>
      </div>

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {nodes.map((node) => (
          <Card key={node.id}>
            <CardContent className="flex items-center gap-3 p-4">
              <Server className="h-5 w-5 text-muted-foreground shrink-0" />
              <div className="min-w-0 flex-1">
                <p className="font-medium truncate">{node.name}</p>
                <p className="text-xs text-muted-foreground truncate">{node.hostname}</p>
              </div>
              <Badge variant={node.is_online ? "default" : "secondary"} className="shrink-0">
                {node.is_online ? "Online" : "Offline"}
              </Badge>
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="flex items-center gap-3">
        <Button
          onClick={handleTrustAll}
          disabled={loading || nodes.length < 2}
          className="gap-2"
        >
          <Network className="h-4 w-4" />
          {loading ? "Wird eingerichtet..." : "Gegenseitiges Vertrauen einrichten"}
        </Button>
        {nodes.length < 2 && (
          <p className="text-sm text-muted-foreground">Mindestens 2 Nodes erforderlich.</p>
        )}
      </div>

      {result && (
        <Card>
          <CardHeader className="pb-2 pt-4 px-4">
            <CardTitle className="text-sm font-medium">Ergebnis</CardTitle>
          </CardHeader>
          <CardContent className="px-4 pb-4 space-y-1">
            {result.distributed?.map((pair) => (
              <div key={pair} className="flex items-center gap-2 text-sm">
                <CheckCircle2 className="h-4 w-4 text-green-500 shrink-0" />
                <span>{pair}</span>
              </div>
            ))}
            {result.failed?.map((f) => (
              <div key={f.node} className="flex items-start gap-2 text-sm">
                <XCircle className="h-4 w-4 text-destructive shrink-0 mt-0.5" />
                <span className="text-destructive">{f.node}: {f.error}</span>
              </div>
            ))}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
