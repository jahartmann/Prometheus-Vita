"use client";

import { useEffect } from "react";
import { RefreshCw, Package, ShieldAlert } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useUpdateStore } from "@/stores/update-store";
import { useNodeStore } from "@/stores/node-store";
import { PackageList } from "@/components/updates/package-list";
import type { PackageUpdate } from "@/types/api";

export default function UpdatesPage() {
  const { checks, isLoading, fetchAll } = useUpdateStore();
  const { nodes, fetchNodes } = useNodeStore();

  useEffect(() => {
    fetchAll();
    fetchNodes();
  }, [fetchAll, fetchNodes]);

  const getNodeName = (nodeId: string) => {
    const node = nodes.find((n) => n.id === nodeId);
    return node?.name || nodeId.slice(0, 8);
  };

  const totalUpdates = checks.reduce((sum, c) => sum + (c.total_updates || 0), 0);
  const securityUpdates = checks.reduce((sum, c) => sum + (c.security_updates || 0), 0);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">Update-Uebersicht</h2>
          <p className="text-sm text-muted-foreground">
            Verfuegbare Paket-Updates fuer alle Nodes.
          </p>
        </div>
        <Button variant="outline" onClick={fetchAll} disabled={isLoading}>
          <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? "animate-spin" : ""}`} />
          Aktualisieren
        </Button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <CardContent className="p-4 flex items-center gap-3">
            <Package className="h-8 w-8 text-blue-500" />
            <div>
              <p className="text-2xl font-bold">{totalUpdates}</p>
              <p className="text-sm text-muted-foreground">Updates gesamt</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4 flex items-center gap-3">
            <ShieldAlert className="h-8 w-8 text-red-500" />
            <div>
              <p className="text-2xl font-bold">{securityUpdates}</p>
              <p className="text-sm text-muted-foreground">Sicherheitsupdates</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4 flex items-center gap-3">
            <Package className="h-8 w-8 text-green-500" />
            <div>
              <p className="text-2xl font-bold">{checks.length}</p>
              <p className="text-sm text-muted-foreground">Nodes geprueft</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {checks.length === 0 && !isLoading ? (
        <Card>
          <CardContent className="p-8 text-center text-muted-foreground">
            Keine Update-Checks vorhanden.
          </CardContent>
        </Card>
      ) : (
        checks.map((check) => {
          const packages: PackageUpdate[] = Array.isArray(check.packages) ? check.packages : [];
          return (
            <Card key={check.id}>
              <CardContent className="p-4">
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{getNodeName(check.node_id)}</span>
                    <Badge variant={check.status === "completed" ? "default" : "secondary"}>
                      {check.status}
                    </Badge>
                  </div>
                  <div className="flex gap-2 text-sm text-muted-foreground">
                    <span>{check.total_updates} Updates</span>
                    {check.security_updates > 0 && (
                      <Badge variant="default" className="bg-red-500">
                        {check.security_updates} Sicherheit
                      </Badge>
                    )}
                  </div>
                </div>
                {packages.length > 0 && <PackageList packages={packages} />}
                {check.error_message && (
                  <p className="text-sm text-destructive mt-2">{check.error_message}</p>
                )}
                <p className="text-xs text-muted-foreground mt-2">
                  Geprueft: {new Date(check.checked_at).toLocaleString("de-DE")}
                </p>
              </CardContent>
            </Card>
          );
        })
      )}
    </div>
  );
}
