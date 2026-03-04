"use client";

import { useEffect } from "react";
import { Shield } from "lucide-react";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useNodeStore } from "@/stores/node-store";
import { useDRStore } from "@/stores/dr-store";
import { ReadinessGauge } from "@/components/dr/readiness-gauge";
import { DRWizard } from "@/components/dr/dr-wizard";
import { Skeleton } from "@/components/ui/skeleton";
import Link from "next/link";

export default function DisasterRecoveryPage() {
  const { nodes, fetchNodes } = useNodeStore();
  const { scores, fetchAllScores, isLoading } = useDRStore();

  useEffect(() => {
    if (nodes.length === 0) fetchNodes();
    fetchAllScores();
  }, [nodes.length, fetchNodes, fetchAllScores]);

  const getNodeName = (nodeId: string) => {
    const node = nodes.find((n) => n.id === nodeId);
    return node?.name || nodeId.slice(0, 8);
  };

  const avgScore =
    scores.length > 0
      ? Math.round(scores.reduce((sum, s) => sum + s.overall_score, 0) / scores.length)
      : 0;

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Shield className="h-6 w-6 text-primary" />
        <div>
          <h1 className="text-2xl font-bold">Disaster Recovery</h1>
          <p className="text-sm text-muted-foreground">
            Wiederherstellungs-Wizard und DR-Bereitschaft Ihrer Nodes.
          </p>
        </div>
      </div>

      <Tabs defaultValue="wizard">
        <TabsList>
          <TabsTrigger value="wizard">Wiederherstellungs-Wizard</TabsTrigger>
          <TabsTrigger value="scores">DR-Scores</TabsTrigger>
        </TabsList>

        <TabsContent value="wizard" className="mt-4">
          <DRWizard />
        </TabsContent>

        <TabsContent value="scores" className="mt-4 space-y-4">
          {isLoading && scores.length === 0 ? (
            <Skeleton className="h-64 w-full" />
          ) : (
            <>
              <div className="grid gap-4 md:grid-cols-3">
                <Card>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm font-medium text-muted-foreground">
                      Durchschnittlicher DR-Score
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="flex justify-center">
                    <ReadinessGauge score={avgScore} size="lg" />
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm font-medium text-muted-foreground">
                      Erfasste Nodes
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="text-3xl font-bold">{scores.length}</div>
                    <p className="text-sm text-muted-foreground">
                      von {nodes.length} Nodes bewertet
                    </p>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm font-medium text-muted-foreground">
                      Kritische Nodes
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="text-3xl font-bold text-red-500">
                      {scores.filter((s) => s.overall_score < 40).length}
                    </div>
                    <p className="text-sm text-muted-foreground">
                      Score unter 40
                    </p>
                  </CardContent>
                </Card>
              </div>

              <Card>
                <CardHeader>
                  <CardTitle>Node DR-Bereitschaft</CardTitle>
                </CardHeader>
                <CardContent>
                  {scores.length === 0 ? (
                    <p className="text-sm text-muted-foreground">
                      Noch keine DR-Scores berechnet. Navigieren Sie zu einem Node, um die DR-Analyse zu starten.
                    </p>
                  ) : (
                    <div className="space-y-3">
                      {scores.map((score) => (
                        <Link
                          key={score.id}
                          href={`/nodes/${score.node_id}/dr`}
                          className="flex items-center justify-between rounded-lg border p-3 transition-colors hover:bg-accent"
                        >
                          <div className="flex items-center gap-4">
                            <ReadinessGauge score={score.overall_score} size="sm" />
                            <div>
                              <div className="font-medium">{getNodeName(score.node_id)}</div>
                              <div className="text-xs text-muted-foreground">
                                Berechnet: {new Date(score.calculated_at).toLocaleString("de-DE")}
                              </div>
                            </div>
                          </div>
                          <div className="flex gap-4 text-xs text-muted-foreground">
                            <div>Backup: {score.backup_score}%</div>
                            <div>Profil: {score.profile_score}%</div>
                            <div>Config: {score.config_score}%</div>
                          </div>
                        </Link>
                      ))}
                    </div>
                  )}
                </CardContent>
              </Card>
            </>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}
