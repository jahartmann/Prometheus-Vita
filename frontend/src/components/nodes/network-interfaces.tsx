"use client";

import { useState } from "react";
import { Edit2, Check, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { networkApi } from "@/lib/api";
import type { NetworkInterface } from "@/types/api";

interface NetworkInterfacesProps {
  nodeId: string;
  interfaces: NetworkInterface[];
  onRefresh: () => void;
}

export function NetworkInterfaces({ nodeId, interfaces, onRefresh }: NetworkInterfacesProps) {
  const [editingIface, setEditingIface] = useState<string | null>(null);
  const [aliasName, setAliasName] = useState("");

  const handleSaveAlias = async (iface: string) => {
    await networkApi.setAlias(nodeId, iface, { display_name: aliasName });
    setEditingIface(null);
    onRefresh();
  };

  return (
    <div className="space-y-3">
      {interfaces.map((iface) => (
        <Card key={iface.iface}>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div
                  className="h-3 w-3 rounded-full"
                  style={{
                    backgroundColor: iface.color || (iface.active ? "#22c55e" : "#6b7280"),
                  }}
                />
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-mono font-medium">{iface.iface}</span>
                    {iface.display_name && (
                      <span className="text-sm text-muted-foreground">
                        ({iface.display_name})
                      </span>
                    )}
                    <Badge variant={iface.active ? "success" : "outline"}>
                      {iface.active ? "Aktiv" : "Inaktiv"}
                    </Badge>
                    <Badge variant="outline">{iface.type}</Badge>
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {iface.cidr || iface.address || "Keine IP"}
                    {iface.gateway && ` | GW: ${iface.gateway}`}
                    {iface.method && ` | ${iface.method}`}
                  </div>
                  {iface.description && (
                    <p className="text-xs text-muted-foreground mt-1">{iface.description}</p>
                  )}
                </div>
              </div>
              <div>
                {editingIface === iface.iface ? (
                  <div className="flex items-center gap-2">
                    <input
                      className="rounded border bg-background px-2 py-1 text-sm"
                      value={aliasName}
                      onChange={(e) => setAliasName(e.target.value)}
                      placeholder="Alias..."
                    />
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleSaveAlias(iface.iface)}
                    >
                      <Check className="h-4 w-4" />
                    </Button>
                    <Button variant="ghost" size="icon" onClick={() => setEditingIface(null)}>
                      <X className="h-4 w-4" />
                    </Button>
                  </div>
                ) : (
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => {
                      setEditingIface(iface.iface);
                      setAliasName(iface.display_name || "");
                    }}
                  >
                    <Edit2 className="h-4 w-4" />
                  </Button>
                )}
              </div>
            </div>
          </CardContent>
        </Card>
      ))}
      {interfaces.length === 0 && (
        <Card>
          <CardContent className="py-8 text-center text-sm text-muted-foreground">
            Keine Netzwerk-Interfaces gefunden.
          </CardContent>
        </Card>
      )}
    </div>
  );
}
