"use client";

import { useEffect, useState } from "react";
import { briefingApi } from "@/lib/api";
import type { MorningBriefing } from "@/types/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

export default function BriefingPage() {
  const [briefing, setBriefing] = useState<MorningBriefing | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    briefingApi
      .getLatest()
      .then((data) => setBriefing(data))
      .catch(() => setError("Noch kein Briefing verfuegbar"))
      .finally(() => setIsLoading(false));
  }, []);

  if (isLoading) {
    return (
      <div className="p-6">
        <h1 className="text-2xl font-bold mb-6">Morning Briefing</h1>
        <div className="text-muted-foreground">Lade Briefing...</div>
      </div>
    );
  }

  if (error || !briefing) {
    return (
      <div className="p-6">
        <h1 className="text-2xl font-bold mb-6">Morning Briefing</h1>
        <Card>
          <CardContent className="py-6 text-center text-muted-foreground">
            {error || "Noch kein Briefing verfuegbar"}
          </CardContent>
        </Card>
      </div>
    );
  }

  const data = briefing.data;

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Morning Briefing</h1>
        <span className="text-sm text-muted-foreground">
          {new Date(briefing.generated_at).toLocaleString("de-DE")}
        </span>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Zusammenfassung</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="whitespace-pre-wrap">{briefing.summary}</p>
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <CardContent className="pt-6">
            <div className="text-2xl font-bold">{data.online_nodes}/{data.total_nodes}</div>
            <div className="text-sm text-muted-foreground">Nodes Online</div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="text-2xl font-bold">{data.unresolved_anomalies}</div>
            <div className="text-sm text-muted-foreground">Ungeloeste Anomalien</div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="text-2xl font-bold">{data.critical_predictions}</div>
            <div className="text-sm text-muted-foreground">Kritische Vorhersagen</div>
          </CardContent>
        </Card>
      </div>

      {data.node_summaries && data.node_summaries.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Node-Uebersicht</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {data.node_summaries.map((ns) => (
                <div
                  key={ns.node_id}
                  className="flex items-center justify-between p-2 rounded border"
                >
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{ns.node_name}</span>
                    <Badge variant={ns.is_online ? "success" : "destructive"}>
                      {ns.is_online ? "Online" : "Offline"}
                    </Badge>
                  </div>
                  <div className="flex gap-4 text-sm text-muted-foreground">
                    <span>CPU: {(ns.cpu_avg ?? 0).toFixed(1)}%</span>
                    <span>RAM: {(ns.mem_pct ?? 0).toFixed(1)}%</span>
                    <span>Disk: {(ns.disk_pct ?? 0).toFixed(1)}%</span>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
