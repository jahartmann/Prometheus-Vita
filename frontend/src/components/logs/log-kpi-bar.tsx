"use client";

import { Card, CardContent } from "@/components/ui/card";
import { useLogStore } from "@/stores/log-store";
import { AlertCircle, AlertTriangle, Activity, Zap } from "lucide-react";

export function LogKpiBar() {
  const kpis = useLogStore((s) => s.kpis);

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
      <Card>
        <CardContent className="flex items-center gap-3 p-4">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-red-500/10">
            <AlertCircle className="h-4 w-4 text-red-500" />
          </div>
          <div className="min-w-0">
            <p className="text-2xl font-bold text-red-500">
              {kpis.errorsPerMin.toFixed(1)}
            </p>
            <p className="text-xs text-muted-foreground">Errors/min</p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="flex items-center gap-3 p-4">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-amber-500/10">
            <AlertTriangle className="h-4 w-4 text-amber-500" />
          </div>
          <div className="min-w-0">
            <p className="text-2xl font-bold text-amber-500">
              {kpis.warningsPerMin.toFixed(1)}
            </p>
            <p className="text-xs text-muted-foreground">Warnings/min</p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="flex items-center gap-3 p-4">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-amber-500/10">
            <Activity className="h-4 w-4 text-amber-500" />
          </div>
          <div className="min-w-0">
            <p className="text-2xl font-bold text-amber-500">
              {kpis.activeAnomalies}
            </p>
            <p className="text-xs text-muted-foreground">Active Anomalies</p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="flex items-center gap-3 p-4">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-sky-500/10">
            <Zap className="h-4 w-4 text-sky-500" />
          </div>
          <div className="min-w-0">
            <p className="text-2xl font-bold text-sky-500">
              {kpis.throughput.toFixed(1)}
            </p>
            <p className="text-xs text-muted-foreground">Throughput (lines/s)</p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
