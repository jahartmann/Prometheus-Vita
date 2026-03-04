"use client";

import { Cpu, MemoryStick, HardDrive } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { formatPercentage } from "@/lib/utils";
import type { MetricsSummary } from "@/types/api";

interface MetricsSummaryCardsProps {
  summary: MetricsSummary;
}

export function MetricsSummaryCards({ summary }: MetricsSummaryCardsProps) {
  const cards = [
    {
      label: "CPU",
      icon: Cpu,
      current: summary.cpu_current * 100,
      avg: summary.cpu_avg * 100,
      max: summary.cpu_max * 100,
    },
    {
      label: "RAM",
      icon: MemoryStick,
      current: summary.memory_avg_percent,
      avg: summary.memory_avg_percent,
      max: summary.memory_max_percent,
    },
    {
      label: "Disk",
      icon: HardDrive,
      current: summary.disk_avg_percent,
      avg: summary.disk_avg_percent,
      max: summary.disk_max_percent,
    },
  ];

  return (
    <div className="grid gap-4 sm:grid-cols-3">
      {cards.map((c) => {
        const Icon = c.icon;
        return (
          <Card key={c.label}>
            <CardContent className="flex items-center gap-4 p-4">
              <Icon className="h-8 w-8 text-primary" />
              <div>
                <p className="text-xs text-muted-foreground">{c.label}</p>
                <p className="text-xl font-bold">{formatPercentage(c.current)}</p>
                <p className="text-xs text-muted-foreground">
                  Avg: {formatPercentage(c.avg)} | Max: {formatPercentage(c.max)}
                </p>
              </div>
            </CardContent>
          </Card>
        );
      })}
    </div>
  );
}
