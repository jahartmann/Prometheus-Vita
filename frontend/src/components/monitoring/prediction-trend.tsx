"use client";

import { useMemo } from "react";
import { Line } from "recharts";

interface Prediction {
  id: string;
  metric: string;
  current_value: number;
  predicted_value: number;
  threshold: number;
  slope: number;
  intercept: number;
  r_squared: number;
  severity: string;
}

interface PredictionTrendProps {
  predictions: Prediction[];
  metric: "cpu" | "memory" | "disk";
  chartDataLength: number;
}

export function usePredictionData(
  predictions: Prediction[],
  metric: "cpu" | "memory" | "disk",
  chartDataLength: number,
  existingData: Array<Record<string, unknown>>
) {
  return useMemo(() => {
    const metricKey = metric === "cpu" ? "cpu" : metric === "memory" ? "memory" : "disk";
    const pred = predictions.find((p) =>
      p.metric.toLowerCase().includes(metricKey)
    );
    if (!pred || existingData.length === 0) return existingData;

    const extendedPoints = 12; // 6 extra points into the future
    const lastTime = existingData[existingData.length - 1];
    const extended = [...existingData];

    for (let i = 1; i <= extendedPoints; i++) {
      const progress = i / extendedPoints;
      const predictedVal = Math.min(
        100,
        Math.max(
          0,
          pred.current_value +
            (pred.predicted_value - pred.current_value) * progress
        )
      );
      extended.push({
        ...lastTime,
        time: `+${i * 30}m`,
        [`${metric}Prediction`]: predictedVal,
        // Keep the original metric key undefined for prediction area
        cpu: undefined,
        ram: undefined,
        disk: undefined,
        memPercent: undefined,
        diskPercent: undefined,
      });
    }

    // Add prediction values to existing data points (last 3 connect smoothly)
    const startIdx = Math.max(0, existingData.length - 3);
    for (let i = startIdx; i < existingData.length; i++) {
      const progress =
        (i - startIdx) / (existingData.length - startIdx + extendedPoints);
      const predictedVal = Math.min(
        100,
        Math.max(
          0,
          pred.current_value +
            (pred.predicted_value - pred.current_value) * progress
        )
      );
      (extended[i] as Record<string, unknown>)[`${metric}Prediction`] =
        predictedVal;
    }

    return extended;
  }, [predictions, metric, existingData, chartDataLength]);
}

export function PredictionTrendLine({
  metric,
  predictions,
}: {
  metric: "cpu" | "memory" | "disk";
  predictions: Prediction[];
}) {
  const metricKey = metric === "cpu" ? "cpu" : metric === "memory" ? "memory" : "disk";
  const pred = predictions.find((p) =>
    p.metric.toLowerCase().includes(metricKey)
  );
  if (!pred) return null;

  return (
    <Line
      type="monotone"
      dataKey={`${metric}Prediction`}
      stroke="hsl(0, 72%, 51%)"
      strokeWidth={2}
      strokeDasharray="8 4"
      dot={false}
      name={`${metric === "cpu" ? "CPU" : metric === "memory" ? "RAM" : "Disk"} Prognose`}
      connectNulls
    />
  );
}

export function PredictionLegendItem({
  predictions,
  metric,
}: {
  predictions: Prediction[];
  metric: "cpu" | "memory" | "disk";
}) {
  const metricKey = metric === "cpu" ? "cpu" : metric === "memory" ? "memory" : "disk";
  const pred = predictions.find((p) =>
    p.metric.toLowerCase().includes(metricKey)
  );
  if (!pred) return null;

  const label =
    metric === "cpu" ? "CPU" : metric === "memory" ? "RAM" : "Disk";

  return (
    <div className="flex items-center gap-2 text-xs text-muted-foreground">
      <span
        className="inline-block h-0.5 w-6"
        style={{
          borderTop: "2px dashed hsl(0, 72%, 51%)",
        }}
      />
      {label} Prognose: {pred.predicted_value.toFixed(1)}%
      {pred.r_squared < 0.5 && (
        <span className="text-amber-500">(unsicher)</span>
      )}
    </div>
  );
}
