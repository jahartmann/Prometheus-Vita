"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { useDRStore } from "@/stores/dr-store";
import { PlayCircle, CheckCircle, XCircle, Loader2 } from "lucide-react";

interface SimulationDialogProps {
  nodeId: string;
}

const scenarios = [
  { value: "node_replacement", label: "Node-Austausch" },
  { value: "disk_failure", label: "Festplattenausfall" },
  { value: "network_failure", label: "Netzwerkausfall" },
  { value: "cluster_recovery", label: "Cluster-Wiederherstellung" },
  { value: "full_restore", label: "Vollständige Wiederherstellung" },
];

export function SimulationDialog({ nodeId }: SimulationDialogProps) {
  const [open, setOpen] = useState(false);
  const [scenario, setScenario] = useState("");
  const [loading, setLoading] = useState(false);
  const { simulationResult, simulate } = useDRStore();

  const handleSimulate = async () => {
    if (!scenario) return;
    setLoading(true);
    try {
      await simulate(nodeId, scenario);
    } catch {
      // Error handled in store
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          <PlayCircle className="mr-2 h-4 w-4" />
          DR simulieren
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>DR-Simulation</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="flex gap-2">
            <Select value={scenario} onValueChange={setScenario}>
              <SelectTrigger className="flex-1">
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
            <Button onClick={handleSimulate} disabled={!scenario || loading}>
              {loading ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                "Starten"
              )}
            </Button>
          </div>

          {simulationResult && (
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                {simulationResult.ready ? (
                  <Badge className="bg-green-500">Bereit</Badge>
                ) : (
                  <Badge variant="destructive">Nicht bereit</Badge>
                )}
                <span className="text-sm text-muted-foreground">
                  {simulationResult.summary}
                </span>
              </div>

              <div className="space-y-2">
                {simulationResult.checks.map((check, i) => (
                  <div key={i} className="flex items-center gap-2 text-sm">
                    {check.passed ? (
                      <CheckCircle className="h-4 w-4 text-green-500" />
                    ) : (
                      <XCircle className="h-4 w-4 text-red-500" />
                    )}
                    <span className="font-medium">{check.name}</span>
                    <span className="text-muted-foreground">
                      {check.message}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
