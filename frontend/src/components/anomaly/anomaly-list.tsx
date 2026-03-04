"use client";

import { useEffect } from "react";
import { useAnomalyStore } from "@/stores/anomaly-store";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export function AnomalyList() {
  const { anomalies, isLoading, fetchUnresolved, resolve } = useAnomalyStore();

  useEffect(() => {
    fetchUnresolved();
  }, [fetchUnresolved]);

  if (isLoading) {
    return <div className="text-muted-foreground text-sm">Lade Anomalien...</div>;
  }

  if (anomalies.length === 0) {
    return (
      <Card>
        <CardContent className="py-6 text-center text-muted-foreground">
          Keine ungeloesten Anomalien erkannt.
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-3">
      {anomalies.map((a) => (
        <Card key={a.id}>
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium">
                {a.metric.replace("_", " ").toUpperCase()}
              </CardTitle>
              <Badge
                variant={
                  a.severity === "critical" ? "destructive" : "warning"
                }
              >
                {a.severity}
              </Badge>
            </div>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-2 text-sm text-muted-foreground mb-3">
              <div>Wert: {a.value.toFixed(2)}%</div>
              <div>Z-Score: {a.z_score.toFixed(2)}</div>
              <div>Mittelwert: {a.mean.toFixed(2)}%</div>
              <div>Std.Abw.: {a.stddev.toFixed(2)}</div>
              <div className="col-span-2">
                Erkannt: {new Date(a.detected_at).toLocaleString("de-DE")}
              </div>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => resolve(a.id)}
            >
              Als geloest markieren
            </Button>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
