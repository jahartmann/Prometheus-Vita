"use client";

import { useState, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Slider } from "@/components/ui/slider";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";

interface ScheduleFormProps {
  onSubmit: (cronExpression: string, retentionDays: number) => void;
  isSubmitting?: boolean;
}

const WEEKDAYS = [
  { value: "0", label: "Sonntag" },
  { value: "1", label: "Montag" },
  { value: "2", label: "Dienstag" },
  { value: "3", label: "Mittwoch" },
  { value: "4", label: "Donnerstag" },
  { value: "5", label: "Freitag" },
  { value: "6", label: "Samstag" },
];

function getNextExecution(cron: string): string {
  const parts = cron.split(" ");
  if (parts.length !== 5) return "Ungueltiger Cron-Ausdruck";

  const [minute, hour, dayOfMonth, , dayOfWeek] = parts;
  const now = new Date();
  const next = new Date(now);

  next.setSeconds(0);
  next.setMilliseconds(0);
  next.setMinutes(parseInt(minute) || 0);
  next.setHours(parseInt(hour) || 0);

  if (dayOfWeek !== "*") {
    const targetDay = parseInt(dayOfWeek);
    const currentDay = next.getDay();
    let daysUntil = targetDay - currentDay;
    if (daysUntil < 0 || (daysUntil === 0 && next <= now)) {
      daysUntil += 7;
    }
    next.setDate(next.getDate() + daysUntil);
  } else if (dayOfMonth !== "*") {
    const targetDate = parseInt(dayOfMonth);
    next.setDate(targetDate);
    if (next <= now) {
      next.setMonth(next.getMonth() + 1);
    }
  } else {
    if (next <= now) {
      next.setDate(next.getDate() + 1);
    }
  }

  return next.toLocaleString("de-DE", {
    weekday: "long",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function ScheduleForm({ onSubmit, isSubmitting }: ScheduleFormProps) {
  const [frequency, setFrequency] = useState<"daily" | "weekly" | "monthly">("daily");
  const [hour, setHour] = useState("2");
  const [minute, setMinute] = useState("0");
  const [weekday, setWeekday] = useState("0");
  const [monthDay, setMonthDay] = useState("1");
  const [retentionDays, setRetentionDays] = useState(30);

  const cronExpression = useMemo(() => {
    const m = minute;
    const h = hour;

    switch (frequency) {
      case "daily":
        return `${m} ${h} * * *`;
      case "weekly":
        return `${m} ${h} * * ${weekday}`;
      case "monthly":
        return `${m} ${h} ${monthDay} * *`;
    }
  }, [frequency, hour, minute, weekday, monthDay]);

  const nextExecution = useMemo(() => getNextExecution(cronExpression), [cronExpression]);

  return (
    <div className="space-y-5">
      <Tabs value={frequency} onValueChange={(v) => setFrequency(v as typeof frequency)}>
        <div className="space-y-2">
          <Label>Frequenz</Label>
          <TabsList className="w-full">
            <TabsTrigger value="daily" className="flex-1">Taeglich</TabsTrigger>
            <TabsTrigger value="weekly" className="flex-1">Woechentlich</TabsTrigger>
            <TabsTrigger value="monthly" className="flex-1">Monatlich</TabsTrigger>
          </TabsList>
        </div>

        <TabsContent value="weekly">
          <div className="space-y-2">
            <Label>Wochentag</Label>
            <Select value={weekday} onValueChange={setWeekday}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {WEEKDAYS.map((d) => (
                  <SelectItem key={d.value} value={d.value}>
                    {d.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </TabsContent>

        <TabsContent value="monthly">
          <div className="space-y-2">
            <Label>Tag des Monats</Label>
            <Select value={monthDay} onValueChange={setMonthDay}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {Array.from({ length: 28 }, (_, i) => i + 1).map((d) => (
                  <SelectItem key={d} value={String(d)}>
                    {d}.
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </TabsContent>
      </Tabs>

      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-2">
          <Label>Stunde</Label>
          <Select value={hour} onValueChange={setHour}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {Array.from({ length: 24 }, (_, i) => i).map((h) => (
                <SelectItem key={h} value={String(h)}>
                  {String(h).padStart(2, "0")}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-2">
          <Label>Minute</Label>
          <Select value={minute} onValueChange={setMinute}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {[0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55].map((m) => (
                <SelectItem key={m} value={String(m)}>
                  {String(m).padStart(2, "0")}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="space-y-2">
        <Label>Aufbewahrung: {retentionDays} Backups</Label>
        <Slider
          min={1}
          max={90}
          step={1}
          value={[retentionDays]}
          onValueChange={(v) => setRetentionDays(v[0])}
        />
        <div className="flex justify-between text-xs text-muted-foreground">
          <span>1 Backup</span>
          <span>90 Backups</span>
        </div>
      </div>

      <div className="rounded border p-3 space-y-1">
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">Cron-Ausdruck:</span>
          <code className="font-mono text-xs bg-muted px-2 py-1 rounded">{cronExpression}</code>
        </div>
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">Naechste Ausfuehrung:</span>
          <span className="text-xs">{nextExecution}</span>
        </div>
      </div>

      <Button
        className="w-full"
        onClick={() => onSubmit(cronExpression, retentionDays)}
        disabled={isSubmitting}
      >
        {isSubmitting ? "Erstelle..." : "Zeitplan erstellen"}
      </Button>
    </div>
  );
}
