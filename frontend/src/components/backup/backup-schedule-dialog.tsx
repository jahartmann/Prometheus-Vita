"use client";

import { useEffect, useState } from "react";
import { Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useBackupStore } from "@/stores/backup-store";
import { scheduleApi } from "@/lib/api";
import { ScheduleForm } from "./schedule-form";
import type { BackupSchedule } from "@/types/api";

interface BackupScheduleDialogProps {
  nodeId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function BackupScheduleDialog({ nodeId, open, onOpenChange }: BackupScheduleDialogProps) {
  const { schedules, fetchSchedules } = useBackupStore();
  const [isCreating, setIsCreating] = useState(false);

  useEffect(() => {
    if (open) fetchSchedules(nodeId);
  }, [open, nodeId, fetchSchedules]);

  if (!open) return null;

  const handleCreate = async (cronExpression: string, retentionDays: number) => {
    setIsCreating(true);
    try {
      await scheduleApi.createSchedule(nodeId, {
        cron_expression: cronExpression,
        is_active: true,
        retention_count: retentionDays,
      });
      fetchSchedules(nodeId);
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

          <div className="border-t pt-4">
            <h4 className="text-sm font-medium mb-3">Neuen Zeitplan erstellen</h4>
            <ScheduleForm onSubmit={handleCreate} isSubmitting={isCreating} />
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
