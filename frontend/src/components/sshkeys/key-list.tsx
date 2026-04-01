"use client";

import { useCallback, useEffect, useState } from "react";
import { Key, Network, Trash2, Upload, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { sshKeyApi, toArray } from "@/lib/api";
import { toast } from "sonner";
import type { SSHKey, TrustResult } from "@/types/api";

interface KeyListProps {
  nodeId: string;
  onGenerate: () => void;
}

export function KeyList({ nodeId, onGenerate }: KeyListProps) {
  const [keys, setKeys] = useState<SSHKey[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  const fetchKeys = useCallback(async () => {
    setIsLoading(true);
    try {
      const resp = await sshKeyApi.listByNode(nodeId);
      setKeys(toArray<SSHKey>(resp.data));
    } catch {
      toast.error("Fehler beim Laden der SSH-Schlüssel");
    } finally {
      setIsLoading(false);
    }
  }, [nodeId]);

  useEffect(() => {
    fetchKeys();
  }, [fetchKeys]);

  const handleDeploy = async (keyId: string) => {
    try {
      await sshKeyApi.deploy(nodeId, keyId);
      fetchKeys();
      toast.success("SSH-Schlüssel deployed");
    } catch {
      toast.error("Fehler beim Deployen des Schlüssels");
    }
  };

  const handleTrust = async (keyId: string) => {
    try {
      const resp = await sshKeyApi.trustAll(nodeId, keyId);
      const result: TrustResult = resp.data;
      const ok = result.distributed_to?.length ?? 0;
      const failed = result.failed?.length ?? 0;
      if (ok === 0 && failed === 0) {
        toast.info("Keine anderen Nodes vorhanden");
      } else if (failed === 0) {
        toast.success(`Schlüssel auf ${ok} Node(s) verteilt`);
      } else if (ok > 0) {
        toast.warning(`${ok} erfolgreich, ${failed} fehlgeschlagen`);
      } else {
        toast.error("Verteilung auf allen Nodes fehlgeschlagen");
      }
    } catch {
      toast.error("Fehler beim Verteilen des Schlüssels");
    }
  };

  const handleDelete = async (keyId: string) => {
    try {
      await sshKeyApi.delete(nodeId, keyId);
      fetchKeys();
      toast.success("SSH-Schlüssel gelöscht");
    } catch {
      toast.error("Fehler beim Löschen des Schlüssels");
    }
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Key className="h-4 w-4" />
          <span className="font-medium">SSH-Schlüssel</span>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={fetchKeys}>
            <RefreshCw className="h-3 w-3 mr-1" />
            Aktualisieren
          </Button>
          <Button size="sm" onClick={onGenerate}>
            Schlüssel generieren
          </Button>
        </div>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Typ</TableHead>
            <TableHead>Fingerprint</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Erstellt</TableHead>
            <TableHead>Ablauf</TableHead>
            <TableHead className="w-28"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading && keys.length === 0 ? (
            <TableRow>
              <TableCell colSpan={7} className="text-center py-8 text-muted-foreground">
                Laden...
              </TableCell>
            </TableRow>
          ) : keys.length === 0 ? (
            <TableRow>
              <TableCell colSpan={7} className="text-center py-8 text-muted-foreground">
                Keine SSH-Schlüssel vorhanden.
              </TableCell>
            </TableRow>
          ) : (
            keys.map((key) => (
              <TableRow key={key.id}>
                <TableCell className="font-medium">{key.name}</TableCell>
                <TableCell>
                  <Badge variant="outline">{key.key_type}</Badge>
                </TableCell>
                <TableCell className="font-mono text-xs text-muted-foreground">
                  {key.fingerprint?.slice(0, 24)}...
                </TableCell>
                <TableCell>
                  <Badge variant={key.is_deployed ? "default" : "secondary"}>
                    {key.is_deployed ? "Deployed" : "Nicht deployed"}
                  </Badge>
                </TableCell>
                <TableCell className="text-muted-foreground text-sm">
                  {new Date(key.created_at).toLocaleDateString("de-DE")}
                </TableCell>
                <TableCell className="text-muted-foreground text-sm">
                  {key.expires_at ? new Date(key.expires_at).toLocaleDateString("de-DE") : "-"}
                </TableCell>
                <TableCell>
                  <div className="flex gap-1">
                    {!key.is_deployed && (
                      <Button variant="ghost" size="icon" onClick={() => handleDeploy(key.id)} title="Deployen">
                        <Upload className="h-4 w-4" />
                      </Button>
                    )}
                    <Button variant="ghost" size="icon" onClick={() => handleTrust(key.id)} title="Auf alle Nodes verteilen (Trust)">
                      <Network className="h-4 w-4" />
                    </Button>
                    <Button variant="ghost" size="icon" onClick={() => handleDelete(key.id)} title="Löschen">
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </div>
  );
}
