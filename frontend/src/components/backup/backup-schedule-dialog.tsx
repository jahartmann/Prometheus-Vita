"use client";

import { useEffect, useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import { useBackupStore } from "@/stores/backup-store";
import { scheduleApi } from "@/lib/api";
import type { BackupSchedule } from "@/types/api";

interface BackupScheduleDialogProps {
  nodeId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const cronPresets = [
  { label: "Taeglich 2:00", value: "0 2 * * *" },
  { label: "Woechentlich Sonntag 3:00", value: "0 3 * * 0" },
  { label: "Alle 6 Stunden", value: "0 */6 * * *" },
  { label: "Alle 12 Stunden", value: "0 */12 * * *" },
];

export function BackupScheduleDialog({ nodeId, open, onOpenChange }: BackupScheduleDialogProps) {
  const { schedules, fetchSchedules } = useBackupStore();
  const [cron, setCron] = useState("0 2 * * *");
  const [retention, setRetention] = useState(10);
  const [isCreating, setIsCreating] = useState(false);

  useEffect(() => {
    if (open) fetchSchedules(nodeId);
  }, [open, nodeId, fetchSchedules]);

  if (!open) return null;

  const handleCreate = async () => {
    setIsCreating(true);
    try {
      await scheduleApi.createSchedule(nodeId, {
        cron_expression: cron,
        is_active: true,
        retention_count: retention,
      });
      fetchSchedules(nodeId);
      setCron("0 2 * * *");
    } catch {
      /* ignore */
    }
    setIsCreating(false);
  };

  const handleDelete = async (id: string) => {
    await scheduleApi.deleteSchedule(id);
    fetchSchedules(nodeId);
  };

  const handleToggle = async (schedule: BackupSchedule) => {
    await scheduleApi.updateSchedule(schedule.id, { is_active: !schedule.is_active });
    fetchSchedules(nodeId);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <Card className="w-full max-w-lg">
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Backup-Zeitplaene</CardTitle>
            <Button variant="ghost" onClick={() => onOpenChange(false)}>
              Schliessen
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {schedules.length > 0 && (
            <div className="space-y-2">
              {schedules.map((s) => (
                <div key={s.id} className="flex items-center justify-between rounded border p-3">
                  <div>
                    <div className="flex items-center gap-2">
                      <code className="text-sm">{s.cron_expression}</code>
                      <Badge variant={s.is_active ? "success" : "outline"}>
                        {s.is_active ? "Aktiv" : "Inaktiv"}
                      </Badge>
                    </div>
                    <p className="text-xs text-muted-foreground mt-1">
                      Aufbewahrung: {s.retention_count} |{" "}
                      {s.next_run_at
                        ? `Naechster Lauf: ${new Date(s.next_run_at).toLocaleString("de-DE")}`
                        : ""}
                    </p>
                  </div>
                  <div className="flex gap-1">
                    <Button variant="ghost" size="sm" onClick={() => handleToggle(s)}>
                      {s.is_active ? "Pause" : "Start"}
                    </Button>
                    <Button variant="ghost" size="icon" onClick={() => handleDelete(s.id)}>
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}

          <div className="space-y-3 border-t pt-4">
            <h4 className="text-sm font-medium">Neuen Zeitplan erstellen</h4>
            <div className="space-y-2">
              <Label>Cron-Ausdruck</Label>
              <input
                className="w-full rounded-md border bg-background px-3 py-2 text-sm font-mono"
                value={cron}
                onChange={(e) => setCron(e.target.value)}
              />
              <div className="flex flex-wrap gap-1">
                {cronPresets.map((p) => (
                  <Button key={p.value} variant="outline" size="sm" onClick={() => setCron(p.value)}>
                    {p.label}
                  </Button>
                ))}
              </div>
            </div>
            <div className="space-y-2">
              <Label>Aufbewahrung (Anzahl Backups)</Label>
              <input
                type="number"
                min={1}
                max={100}
                className="w-full rounded-md border bg-background px-3 py-2 text-sm"
                value={retention}
                onChange={(e) => setRetention(parseInt(e.target.value) || 10)}
              />
            </div>
            <Button onClick={handleCreate} disabled={isCreating} className="w-full">
              <Plus className="mr-2 h-4 w-4" />
              {isCreating ? "Erstelle..." : "Zeitplan erstellen"}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
