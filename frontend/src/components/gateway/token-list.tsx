"use client";

import { useEffect, useState } from "react";
import { Trash2, Ban, RefreshCw } from "lucide-react";
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
import { gatewayApi, toArray } from "@/lib/api";
import { toast } from "sonner";
import type { APIToken } from "@/types/api";

interface TokenListProps {
  onRefresh?: () => void;
  refreshKey?: number;
}

export function TokenList({ refreshKey }: TokenListProps) {
  const [tokens, setTokens] = useState<APIToken[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  const fetchTokens = async () => {
    setIsLoading(true);
    try {
      const resp = await gatewayApi.listTokens();
      setTokens(toArray<APIToken>(resp.data));
    } catch {
      // Fehler
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchTokens();
  }, [refreshKey]);

  const handleRevoke = async (id: string) => {
    try {
      await gatewayApi.revokeToken(id);
      fetchTokens();
      toast.success("Token widerrufen");
    } catch {
      toast.error("Fehler beim Widerrufen des Tokens");
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await gatewayApi.deleteToken(id);
      fetchTokens();
      toast.success("Token geloescht");
    } catch {
      toast.error("Fehler beim Loeschen des Tokens");
    }
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <span className="font-medium">API-Tokens</span>
        <Button variant="outline" size="sm" onClick={fetchTokens}>
          <RefreshCw className="h-3 w-3 mr-1" />
          Aktualisieren
        </Button>
      </div>

      <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Praefix</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Berechtigungen</TableHead>
            <TableHead>Zuletzt benutzt</TableHead>
            <TableHead>Ablauf</TableHead>
            <TableHead className="w-24"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading && tokens.length === 0 ? (
            <TableRow>
              <TableCell colSpan={7} className="text-center py-8 text-muted-foreground">
                Laden...
              </TableCell>
            </TableRow>
          ) : tokens.length === 0 ? (
            <TableRow>
              <TableCell colSpan={7} className="text-center py-8 text-muted-foreground">
                Keine API-Tokens vorhanden.
              </TableCell>
            </TableRow>
          ) : (
            tokens.map((token) => (
              <TableRow key={token.id}>
                <TableCell className="font-medium">{token.name}</TableCell>
                <TableCell className="font-mono text-sm text-muted-foreground">
                  {token.token_prefix}...
                </TableCell>
                <TableCell>
                  <Badge variant={token.is_active ? "default" : "secondary"}>
                    {token.is_active ? "Aktiv" : "Widerrufen"}
                  </Badge>
                </TableCell>
                <TableCell>
                  <div className="flex gap-1 flex-wrap">
                    {Array.isArray(token.permissions) && token.permissions.length > 0
                      ? token.permissions.map((p) => (
                          <Badge key={p} variant="outline" className="text-xs">
                            {p}
                          </Badge>
                        ))
                      : <span className="text-muted-foreground text-sm">Alle</span>}
                  </div>
                </TableCell>
                <TableCell className="text-muted-foreground text-sm">
                  {token.last_used_at
                    ? new Date(token.last_used_at).toLocaleString("de-DE")
                    : "Nie"}
                </TableCell>
                <TableCell className="text-muted-foreground text-sm">
                  {token.expires_at
                    ? new Date(token.expires_at).toLocaleDateString("de-DE")
                    : "Nie"}
                </TableCell>
                <TableCell>
                  <div className="flex gap-1">
                    {token.is_active && (
                      <Button variant="ghost" size="icon" onClick={() => handleRevoke(token.id)} title="Widerrufen">
                        <Ban className="h-4 w-4 text-yellow-500" />
                      </Button>
                    )}
                    <Button variant="ghost" size="icon" onClick={() => handleDelete(token.id)} title="Loeschen">
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
    </div>
  );
}
