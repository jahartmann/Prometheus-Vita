"use client";

import { useState } from "react";
import { Edit2, Check, X, Network, Wifi, ChevronDown, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
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
  const [showDetails, setShowDetails] = useState(false);

  const handleSaveAlias = async (iface: string) => {
    await networkApi.setAlias(nodeId, iface, { display_name: aliasName });
    setEditingIface(null);
    onRefresh();
  };

  const activeCount = interfaces.filter((i) => i.active).length;
  const inactiveCount = interfaces.length - activeCount;
  const withIpCount = interfaces.filter((i) => i.cidr || i.address).length;

  return (
    <div className="space-y-4">
      {/* Summary Cards */}
      <div className="grid gap-4 sm:grid-cols-3">
        <Card hover className="gradient-blue">
          <CardContent className="p-5">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-kpi-blue/15">
                <Network className="h-5 w-5 text-kpi-blue" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Interfaces</p>
                <p className="text-xl font-bold">{interfaces.length}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card hover className="gradient-green">
          <CardContent className="p-5">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-kpi-green/15">
                <Wifi className="h-5 w-5 text-kpi-green" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Aktiv</p>
                <p className="text-xl font-bold">{activeCount}</p>
                {inactiveCount > 0 && (
                  <p className="text-xs text-muted-foreground">{inactiveCount} inaktiv</p>
                )}
              </div>
            </div>
          </CardContent>
        </Card>

        <Card hover className="gradient-orange">
          <CardContent className="p-5">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-kpi-orange/15">
                <Network className="h-5 w-5 text-kpi-orange" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Mit IP</p>
                <p className="text-xl font-bold">{withIpCount}</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Expandable Detail Section */}
      <Card>
        <CardHeader className="py-3 px-4">
          <Button
            variant="ghost"
            className="flex w-full items-center justify-between p-0 h-auto hover:bg-transparent"
            onClick={() => setShowDetails(!showDetails)}
          >
            <CardTitle className="text-base">
              Interface-Details ({interfaces.length})
            </CardTitle>
            {showDetails ? (
              <ChevronDown className="h-4 w-4 text-muted-foreground" />
            ) : (
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
            )}
          </Button>
        </CardHeader>
        {showDetails && (
          <CardContent className="p-0">
            <div className="divide-y">
              {interfaces.map((iface) => (
                <div key={iface.iface} className="flex items-center justify-between p-4">
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
                        <Badge variant={iface.active ? "success" : "outline"} className="text-xs">
                          {iface.active ? "Aktiv" : "Inaktiv"}
                        </Badge>
                        <Badge variant="outline" className="text-xs">{iface.type}</Badge>
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
              ))}
              {interfaces.length === 0 && (
                <div className="py-8 text-center text-sm text-muted-foreground">
                  Keine Netzwerk-Interfaces gefunden.
                </div>
              )}
            </div>
          </CardContent>
        )}
      </Card>
    </div>
  );
}
