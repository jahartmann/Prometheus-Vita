"use client";

import { useEffect, useState } from "react";
import { predictionApi } from "@/lib/api";
import type { MaintenancePrediction } from "@/types/api";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export function PredictionCard() {
  const [predictions, setPredictions] = useState<MaintenancePrediction[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    predictionApi
      .listCritical()
      .then((data) => setPredictions(data || []))
      .finally(() => setIsLoading(false));
  }, []);

  if (isLoading) {
    return <div className="text-muted-foreground text-sm">Lade Vorhersagen...</div>;
  }

  if (predictions.length === 0) {
    return (
      <Card>
        <CardContent className="py-6 text-center text-muted-foreground">
          Keine kritischen Vorhersagen.
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-3">
      {predictions.map((p) => (
        <Card key={p.id}>
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium">
                {p.metric.replace("_", " ").toUpperCase()}
              </CardTitle>
              <Badge
                variant={
                  p.severity === "critical"
                    ? "destructive"
                    : p.severity === "warning"
                    ? "warning"
                    : "secondary"
                }
              >
                {p.severity}
              </Badge>
            </div>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-2 text-sm text-muted-foreground">
              <div>Aktuell: {p.current_value.toFixed(1)}%</div>
              <div>Schwellenwert: {p.threshold}%</div>
              {p.days_until_threshold != null && (
                <div className="col-span-2 font-medium text-foreground">
                  Geschaetzte Tage bis Schwellenwert:{" "}
                  {p.days_until_threshold.toFixed(1)}
                </div>
              )}
              <div>R²: {p.r_squared.toFixed(3)}</div>
              <div>
                Berechnet:{" "}
                {new Date(p.predicted_at).toLocaleString("de-DE")}
              </div>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
