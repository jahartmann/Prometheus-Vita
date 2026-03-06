"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { briefingApi } from "@/lib/api";
import type { LiveBriefingSummary } from "@/types/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Sun,
  Server,
  ServerCog,
  Cpu,
  MemoryStick,
  AlertTriangle,
  ArrowRight,
} from "lucide-react";
import { Button } from "@/components/ui/button";

function getGreeting(): string {
  const hour = new Date().getHours();
  if (hour < 12) return "Guten Morgen";
  if (hour < 18) return "Guten Tag";
  return "Guten Abend";
}

export function BriefingWidget() {
  const [data, setData] = useState<LiveBriefingSummary | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    briefingApi
      .getLive()
      .then((d) => setData(d))
      .catch(() => {})
      .finally(() => setIsLoading(false));
  }, []);

  if (isLoading) {
    return (
      <Card>
        <CardContent className="p-5">
          <div className="h-20 animate-pulse rounded bg-muted" />
        </CardContent>
      </Card>
    );
  }

  if (!data) return null;

  const hasIssues = data.nodes_offline > 0 || data.unresolved_anomalies > 0 || data.critical_predictions > 0;

  return (
    <Card hover>
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2 text-base">
            <Sun className="h-4 w-4 text-orange-400" />
            {getGreeting()}
          </CardTitle>
          <Button variant="ghost" size="sm" asChild>
            <Link href="/briefing" className="flex items-center gap-1 text-xs">
              Details
              <ArrowRight className="h-3 w-3" />
            </Link>
          </Button>
        </div>
      </CardHeader>
      <CardContent className="pt-0">
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <div className="flex items-center gap-2">
            <Server className="h-4 w-4 text-muted-foreground" />
            <div>
              <p className="text-lg font-bold leading-none">{data.nodes_online}/{data.nodes_total}</p>
              <p className="text-xs text-muted-foreground">Nodes</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <ServerCog className="h-4 w-4 text-muted-foreground" />
            <div>
              <p className="text-lg font-bold leading-none">{data.vms_running}/{data.vms_total}</p>
              <p className="text-xs text-muted-foreground">VMs aktiv</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Cpu className="h-4 w-4 text-muted-foreground" />
            <div>
              <p className="text-lg font-bold leading-none">{data.avg_cpu.toFixed(1)}%</p>
              <p className="text-xs text-muted-foreground">CPU</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <MemoryStick className="h-4 w-4 text-muted-foreground" />
            <div>
              <p className="text-lg font-bold leading-none">{data.avg_ram.toFixed(1)}%</p>
              <p className="text-xs text-muted-foreground">RAM</p>
            </div>
          </div>
        </div>

        {hasIssues && (
          <div className="mt-3 flex flex-wrap gap-2">
            {data.nodes_offline > 0 && (
              <Badge variant="destructive" className="text-xs">
                <AlertTriangle className="mr-1 h-3 w-3" />
                {data.nodes_offline} Node{data.nodes_offline > 1 ? "s" : ""} offline
              </Badge>
            )}
            {data.unresolved_anomalies > 0 && (
              <Badge variant="warning" className="text-xs">
                {data.unresolved_anomalies} Anomalie{data.unresolved_anomalies > 1 ? "n" : ""}
              </Badge>
            )}
            {data.critical_predictions > 0 && (
              <Badge variant="outline" className="text-xs border-orange-500/50 text-orange-500">
                {data.critical_predictions} Vorhersage{data.critical_predictions > 1 ? "n" : ""}
              </Badge>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
