"use client";

import { useEffect, useState } from "react";
import { Key, Trash2, Upload, RefreshCw } from "lucide-react";
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
import { sshKeyApi } from "@/lib/api";
import type { SSHKey } from "@/types/api";

interface KeyListProps {
  nodeId: string;
  onGenerate: () => void;
}

export function KeyList({ nodeId, onGenerate }: KeyListProps) {
  const [keys, setKeys] = useState<SSHKey[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  const fetchKeys = async () => {
    setIsLoading(true);
    try {
      const resp = await sshKeyApi.listByNode(nodeId);
      setKeys(resp.data?.data || resp.data || []);
    } catch {
      // Fehler
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchKeys();
  }, [nodeId]);

  const handleDeploy = async (keyId: string) => {
    try {
      await sshKeyApi.deploy(nodeId, keyId);
      fetchKeys();
    } catch {
      // Fehler
    }
  };

  const handleDelete = async (keyId: string) => {
    try {
      await sshKeyApi.delete(nodeId, keyId);
      fetchKeys();
    } catch {
      // Fehler
    }
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Key className="h-4 w-4" />
          <span className="font-medium">SSH-Schluessel</span>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={fetchKeys}>
            <RefreshCw className="h-3 w-3 mr-1" />
            Aktualisieren
          </Button>
          <Button size="sm" onClick={onGenerate}>
            Schluessel generieren
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
            <TableHead className="w-24"></TableHead>
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
                Keine SSH-Schluessel vorhanden.
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
                    <Button variant="ghost" size="icon" onClick={() => handleDelete(key.id)} title="Loeschen">
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
