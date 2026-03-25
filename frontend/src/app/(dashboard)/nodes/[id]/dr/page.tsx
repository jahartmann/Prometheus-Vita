"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { ArrowLeft, RefreshCw, Plus, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useNodeStore } from "@/stores/node-store";
import { useDRStore } from "@/stores/dr-store";
import { ReadinessGauge } from "@/components/dr/readiness-gauge";
import { ProfileViewer } from "@/components/dr/profile-viewer";
import { RunbookViewer } from "@/components/dr/runbook-viewer";
import { SimulationDialog } from "@/components/dr/simulation-dialog";
import { Skeleton } from "@/components/ui/skeleton";

const scenarios = [
  { value: "node_replacement", label: "Node-Austausch" },
  { value: "disk_failure", label: "Festplattenausfall" },
  { value: "network_failure", label: "Netzwerkausfall" },
  { value: "cluster_recovery", label: "Cluster-Wiederherstellung" },
  { value: "full_restore", label: "Vollständige Wiederherstellung" },
];

export default function NodeDRPage() {
  const params = useParams<{ id: string }>();
  const nodeId = params.id;
  const { nodes, fetchNodes } = useNodeStore();
  const {
    profile,
    currentScore,
    runbooks,
    fetchProfile,
    collectProfile,
    fetchReadiness,
    calculateReadiness,
    fetchRunbooks,
    generateRunbook,
  } = useDRStore();

  const [collectingProfile, setCollectingProfile] = useState(false);
  const [calculatingScore, setCalculatingScore] = useState(false);
  const [generatingRunbook, setGeneratingRunbook] = useState(false);
  const [selectedScenario, setSelectedScenario] = useState("");

  useEffect(() => {
    if (nodes.length === 0) fetchNodes();
  }, [nodes.length, fetchNodes]);

  useEffect(() => {
    if (nodeId) {
      fetchProfile(nodeId);
      fetchReadiness(nodeId);
      fetchRunbooks(nodeId);
    }
  }, [nodeId, fetchProfile, fetchReadiness, fetchRunbooks]);

  const node = nodes.find((n) => n.id === nodeId);
  if (!node) return <Skeleton className="h-64 w-full" />;

  const handleCollectProfile = async () => {
    setCollectingProfile(true);
    try {
      await collectProfile(nodeId);
    } catch {
      // Error handled in store
    } finally {
      setCollectingProfile(false);
    }
  };

  const handleCalculateScore = async () => {
    setCalculatingScore(true);
    try {
      await calculateReadiness(nodeId);
    } catch {
      // Error handled in store
    } finally {
      setCalculatingScore(false);
    }
  };

  const handleGenerateRunbook = async () => {
    if (!selectedScenario) return;
    setGeneratingRunbook(true);
    try {
      await generateRunbook(nodeId, selectedScenario);
      setSelectedScenario("");
    } catch {
      // Error handled in store
    } finally {
      setGeneratingRunbook(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href={`/nodes/${nodeId}`}>
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <h1 className="text-2xl font-bold">Disaster Recovery - {node.name}</h1>
        </div>
        <SimulationDialog nodeId={nodeId} />
      </div>

      {/* Readiness Score */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card className="md:col-span-1">
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium">DR-Bereitschaft</CardTitle>
              <Button
                variant="ghost"
                size="icon"
                onClick={handleCalculateScore}
                disabled={calculatingScore}
              >
                {calculatingScore ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <RefreshCw className="h-4 w-4" />
                )}
              </Button>
            </div>
          </CardHeader>
          <CardContent className="flex justify-center">
            <ReadinessGauge
              score={currentScore?.overall_score || 0}
              size="lg"
            />
          </CardContent>
        </Card>

        <Card className="md:col-span-3">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Score-Details</CardTitle>
          </CardHeader>
          <CardContent>
            {currentScore ? (
              <div className="grid gap-4 md:grid-cols-3">
                <div className="flex flex-col items-center gap-1">
                  <ReadinessGauge score={currentScore.backup_score} size="sm" label="Backup" />
                  <span className="text-xs text-muted-foreground">Gewichtung: 40%</span>
                </div>
                <div className="flex flex-col items-center gap-1">
                  <ReadinessGauge score={currentScore.profile_score} size="sm" label="Profil" />
                  <span className="text-xs text-muted-foreground">Gewichtung: 30%</span>
                </div>
                <div className="flex flex-col items-center gap-1">
                  <ReadinessGauge score={currentScore.config_score} size="sm" label="Config" />
                  <span className="text-xs text-muted-foreground">Gewichtung: 30%</span>
                </div>
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">
                Noch kein Score berechnet.
              </p>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Hardware Profile */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Hardware-/Software-Profil</CardTitle>
            <Button
              variant="outline"
              size="sm"
              onClick={handleCollectProfile}
              disabled={collectingProfile}
            >
              {collectingProfile ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <RefreshCw className="mr-2 h-4 w-4" />
              )}
              Profil erfassen
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {profile ? (
            <ProfileViewer profile={profile} />
          ) : (
            <p className="text-sm text-muted-foreground">
              Noch kein Profil erfasst. Klicken Sie auf &quot;Profil erfassen&quot;, um die Hardware- und Software-Informationen zu sammeln.
            </p>
          )}
        </CardContent>
      </Card>

      {/* Recovery Runbooks */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Recovery-Runbooks</CardTitle>
            <div className="flex gap-2">
              <Select value={selectedScenario} onValueChange={setSelectedScenario}>
                <SelectTrigger className="w-[220px]">
                  <SelectValue placeholder="Szenario wählen..." />
                </SelectTrigger>
                <SelectContent>
                  {scenarios.map((s) => (
                    <SelectItem key={s.value} value={s.value}>
                      {s.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Button
                variant="outline"
                size="sm"
                onClick={handleGenerateRunbook}
                disabled={!selectedScenario || generatingRunbook}
              >
                {generatingRunbook ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <Plus className="mr-2 h-4 w-4" />
                )}
                Generieren
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {runbooks.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              Noch keine Runbooks erstellt. Wählen Sie ein Szenario und generieren Sie ein Runbook.
            </p>
          ) : (
            <div className="space-y-4">
              {runbooks.map((runbook) => (
                <RunbookViewer key={runbook.id} runbook={runbook} />
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
