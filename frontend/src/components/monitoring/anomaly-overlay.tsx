"use client";

import { useMemo } from "react";
import { ReferenceDot } from "recharts";

interface AnomalyPoint {
  id: string;
  metric: string;
  value: number;
  z_score: number;
  mean: number;
  stddev: number;
  severity: string;
  detected_at: string;
}

interface AnomalyOverlayProps {
  anomalies: AnomalyPoint[];
  metric: "cpu" | "memory" | "disk";
  chartDataTimes: string[];
}

const severityColors: Record<string, string> = {
  critical: "hsl(0, 72%, 51%)",
  warning: "hsl(38, 92%, 50%)",
  info: "hsl(210, 80%, 55%)",
};

export function AnomalyOverlay({
  anomalies,
  metric,
  chartDataTimes,
}: AnomalyOverlayProps) {
  const metricMap: Record<string, string> = {
    cpu: "cpu",
    memory: "memory",
    disk: "disk",
  };

  const filtered = useMemo(
    () =>
      anomalies.filter((a) => a.metric.toLowerCase().includes(metricMap[metric])),
    [anomalies, metric]
  );

  if (filtered.length === 0) return null;

  return (
    <>
      {filtered.map((a) => {
        const time = new Date(a.detected_at).toLocaleTimeString("de-DE", {
          hour: "2-digit",
          minute: "2-digit",
        });
        const idx = chartDataTimes.indexOf(time);
        if (idx === -1) return null;

        const color = severityColors[a.severity] || severityColors.info;
        return (
          <ReferenceDot
            key={a.id}
            x={time}
            y={a.value}
            r={6}
            fill={color}
            stroke={color}
            strokeWidth={2}
            fillOpacity={0.7}
          />
        );
      })}
    </>
  );
}

export function AnomalyTooltipContent({
  anomalies,
  time,
}: {
  anomalies: AnomalyPoint[];
  time: string;
}) {
  const matching = anomalies.filter((a) => {
    const t = new Date(a.detected_at).toLocaleTimeString("de-DE", {
      hour: "2-digit",
      minute: "2-digit",
    });
    return t === time;
  });

  if (matching.length === 0) return null;

  return (
    <div className="mt-2 border-t border-red-500/30 pt-2">
      {matching.map((a) => (
        <div key={a.id} className="text-xs">
          <span
            className="mr-1 inline-block h-2 w-2 rounded-full"
            style={{
              backgroundColor:
                severityColors[a.severity] || severityColors.info,
            }}
          />
          Anomalie: {a.metric} = {a.value.toFixed(1)}% (Z-Score:{" "}
          {a.z_score.toFixed(1)})
        </div>
      ))}
    </div>
  );
}
