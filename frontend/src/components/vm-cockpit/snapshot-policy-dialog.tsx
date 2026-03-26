"use client";

import { useEffect, useState, useCallback } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Plus, Trash2, Clock, Camera, RefreshCw } from "lucide-react";
import { snapshotPolicyApi, toArray } from "@/lib/api";
import type { SnapshotPolicy } from "@/types/api";

interface SnapshotPolicyDialogProps {
  nodeId: string;
  vmid: number;
  vmType: string;
}

export function SnapshotPolicySection({ nodeId, vmid, vmType }: SnapshotPolicyDialogProps) {
  const [policies, setPolicies] = useState<SnapshotPolicy[]>([]);
  const [loading, setLoading] = useState(false);
  const [dialogOpen, setDialogOpen] = useState(false);

  const fetchPolicies = useCallback(async () => {
    if (!nodeId || !vmid) return;
    setLoading(true);
    try {
      const res = await snapshotPolicyApi.list(nodeId, vmid);
      setPolicies(toArray<SnapshotPolicy>(res.data));
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [nodeId, vmid]);

  useEffect(() => {
    fetchPolicies();
  }, [fetchPolicies]);

  const handleDelete = async (policyId: string) => {
    try {
      await snapshotPolicyApi.delete(nodeId, vmid, policyId);
      fetchPolicies();
    } catch {
      // ignore
    }
  };

  const handleToggle = async (policy: SnapshotPolicy) => {
    try {
      await snapshotPolicyApi.update(nodeId, vmid, policy.id, {
        is_active: !policy.is_active,
      });
      fetchPolicies();
    } catch {
      // ignore
    }
  };

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-3">
        <CardTitle className="text-base flex items-center gap-2">
          <Camera className="h-4 w-4" />
          Snapshot-Richtlinien
        </CardTitle>
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="icon" onClick={fetchPolicies} disabled={loading}>
            <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          </Button>
          <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
            <DialogTrigger asChild>
              <Button size="sm">
                <Plus className="h-4 w-4 mr-1" />
                Neue Richtlinie
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Snapshot-Richtlinie erstellen</DialogTitle>
              </DialogHeader>
              <CreatePolicyForm
                nodeId={nodeId}
                vmid={vmid}
                vmType={vmType}
                onCreated={() => {
                  setDialogOpen(false);
                  fetchPolicies();
                }}
              />
            </DialogContent>
          </Dialog>
        </div>
      </CardHeader>
      <CardContent>
        {policies.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-4">
            Keine Snapshot-Richtlinien konfiguriert.
          </p>
        ) : (
          <div className="space-y-3">
            {policies.map((policy) => (
              <div
                key={policy.id}
                className="flex items-center justify-between rounded-lg border p-3"
              >
                <div className="space-y-1">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-sm">{policy.name}</span>
                    <Badge variant={policy.is_active ? "success" : "secondary"}>
                      {policy.is_active ? "Aktiv" : "Inaktiv"}
                    </Badge>
                  </div>
                  <div className="flex items-center gap-3 text-xs text-muted-foreground">
                    <span className="flex items-center gap-1">
                      <Clock className="h-3 w-3" />
                      {policy.schedule_cron}
                    </span>
                    <span>
                      Behalten: {policy.keep_daily} täglich
                      {policy.keep_weekly > 0 && `, ${policy.keep_weekly} wöchentlich`}
                      {policy.keep_monthly > 0 && `, ${policy.keep_monthly} monatlich`}
                    </span>
                  </div>
                  {policy.last_run && (
                    <p className="text-xs text-muted-foreground">
                      Letzter Lauf: {new Date(policy.last_run).toLocaleString("de-DE")}
                    </p>
                  )}
                </div>
                <div className="flex items-center gap-2">
                  <Switch
                    checked={policy.is_active}
                    onCheckedChange={() => handleToggle(policy)}
                  />
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => handleDelete(policy.id)}
                  >
                    <Trash2 className="h-4 w-4 text-destructive" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function CreatePolicyForm({
  nodeId,
  vmid,
  vmType,
  onCreated,
}: {
  nodeId: string;
  vmid: number;
  vmType: string;
  onCreated: () => void;
}) {
  const [name, setName] = useState("");
  const [keepDaily, setKeepDaily] = useState(5);
  const [keepWeekly, setKeepWeekly] = useState(4);
  const [keepMonthly, setKeepMonthly] = useState(0);
  const [cron, setCron] = useState("0 2 * * *");
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      await snapshotPolicyApi.create(nodeId, vmid, {
        node_id: nodeId,
        vmid,
        vm_type: vmType,
        name,
        keep_daily: keepDaily,
        keep_weekly: keepWeekly,
        keep_monthly: keepMonthly,
        schedule_cron: cron,
        is_active: true,
      });
      onCreated();
    } catch {
      // ignore
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="space-y-2">
        <Label>Name</Label>
        <Input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="z.B. Täglich-Backup"
          required
        />
      </div>

      <div className="grid grid-cols-3 gap-3">
        <div className="space-y-2">
          <Label>Täglich behalten</Label>
          <Input
            type="number"
            min={0}
            value={keepDaily}
            onChange={(e) => setKeepDaily(Number(e.target.value))}
          />
        </div>
        <div className="space-y-2">
          <Label>Wöchentlich</Label>
          <Input
            type="number"
            min={0}
            value={keepWeekly}
            onChange={(e) => setKeepWeekly(Number(e.target.value))}
          />
        </div>
        <div className="space-y-2">
          <Label>Monatlich</Label>
          <Input
            type="number"
            min={0}
            value={keepMonthly}
            onChange={(e) => setKeepMonthly(Number(e.target.value))}
          />
        </div>
      </div>

      <div className="space-y-2">
        <Label>Zeitplan (Cron)</Label>
        <Input
          value={cron}
          onChange={(e) => setCron(e.target.value)}
          placeholder="0 2 * * *"
        />
        <p className="text-xs text-muted-foreground">
          Standard: Jeden Tag um 02:00 Uhr
        </p>
      </div>

      <Button type="submit" className="w-full" disabled={submitting || !name}>
        {submitting ? "Erstelle..." : "Richtlinie erstellen"}
      </Button>
    </form>
  );
}
