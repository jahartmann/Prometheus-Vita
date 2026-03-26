"use client";

import { cn } from "@/lib/utils";

interface ReadinessGaugeProps {
  score: number;
  label?: string;
  size?: "sm" | "md" | "lg";
}

function getScoreColor(score: number): string {
  if (score >= 80) return "text-green-500";
  if (score >= 60) return "text-yellow-500";
  if (score >= 40) return "text-orange-500";
  return "text-red-500";
}

function getScoreLabel(score: number): string {
  if (score >= 80) return "Bereit";
  if (score >= 60) return "Teilweise bereit";
  if (score >= 40) return "Eingeschränkt";
  return "Nicht bereit";
}

function getTrackColor(score: number): string {
  if (score >= 80) return "stroke-green-500";
  if (score >= 60) return "stroke-yellow-500";
  if (score >= 40) return "stroke-orange-500";
  return "stroke-red-500";
}

export function ReadinessGauge({ score, label, size = "md" }: ReadinessGaugeProps) {
  const sizes = {
    sm: { width: 80, textSize: "text-lg", labelSize: "text-xs" },
    md: { width: 120, textSize: "text-2xl", labelSize: "text-sm" },
    lg: { width: 160, textSize: "text-4xl", labelSize: "text-base" },
  };

  const { width, textSize, labelSize } = sizes[size];
  const radius = (width - 16) / 2;
  const circumference = 2 * Math.PI * radius;
  const progress = (score / 100) * circumference;

  return (
    <div className="flex flex-col items-center gap-1">
      <div className="relative" style={{ width, height: width }}>
        <svg width={width} height={width} className="-rotate-90">
          <circle
            cx={width / 2}
            cy={width / 2}
            r={radius}
            fill="none"
            strokeWidth={8}
            className="stroke-muted"
          />
          <circle
            cx={width / 2}
            cy={width / 2}
            r={radius}
            fill="none"
            strokeWidth={8}
            strokeDasharray={circumference}
            strokeDashoffset={circumference - progress}
            strokeLinecap="round"
            className={cn("transition-all duration-500", getTrackColor(score))}
          />
        </svg>
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span className={cn("font-bold", textSize, getScoreColor(score))}>
            {score}
          </span>
        </div>
      </div>
      <span className={cn("font-medium", labelSize, getScoreColor(score))}>
        {label || getScoreLabel(score)}
      </span>
    </div>
  );
}
