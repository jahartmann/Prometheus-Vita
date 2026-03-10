"use client";

import { useMemo } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatBandwidth } from "@/lib/utils";

interface LiveBandwidthGaugeProps {
  netIn: number;
  netOut: number;
  maxRate?: number;
}

function GaugeArc({
  value,
  maxValue,
  color,
  label,
  size = 140,
}: {
  value: number;
  maxValue: number;
  color: string;
  label: string;
  size?: number;
}) {
  const radius = (size - 20) / 2;
  const cx = size / 2;
  const cy = size / 2 + 10;
  const startAngle = -210;
  const endAngle = 30;
  const totalAngle = endAngle - startAngle;
  const clampedRatio = Math.min(value / Math.max(maxValue, 1), 1);
  const sweepAngle = totalAngle * clampedRatio;

  const toRad = (deg: number) => (deg * Math.PI) / 180;
  const polarToCart = (angle: number, r: number) => ({
    x: cx + r * Math.cos(toRad(angle)),
    y: cy + r * Math.sin(toRad(angle)),
  });

  const bgStart = polarToCart(startAngle, radius);
  const bgEnd = polarToCart(endAngle, radius);
  const bgPath = `M ${bgStart.x} ${bgStart.y} A ${radius} ${radius} 0 1 1 ${bgEnd.x} ${bgEnd.y}`;

  const valEnd = polarToCart(startAngle + sweepAngle, radius);
  const largeArc = sweepAngle > 180 ? 1 : 0;
  const valPath = `M ${bgStart.x} ${bgStart.y} A ${radius} ${radius} 0 ${largeArc} 1 ${valEnd.x} ${valEnd.y}`;

  return (
    <div className="flex flex-col items-center">
      <svg width={size} height={size * 0.75} viewBox={`0 0 ${size} ${size * 0.75}`}>
        <path
          d={bgPath}
          fill="none"
          stroke="hsl(var(--muted))"
          strokeWidth={8}
          strokeLinecap="round"
        />
        {clampedRatio > 0 && (
          <path
            d={valPath}
            fill="none"
            stroke={color}
            strokeWidth={8}
            strokeLinecap="round"
            style={{
              transition: "all 0.5s ease-out",
            }}
          />
        )}
        <text
          x={cx}
          y={cy - 8}
          textAnchor="middle"
          className="fill-foreground text-sm font-bold"
          style={{ fontSize: 14 }}
        >
          {formatBandwidth(value)}
        </text>
      </svg>
      <p className="mt-1 text-xs font-medium text-muted-foreground">{label}</p>
    </div>
  );
}

export function LiveBandwidthGauge({ netIn, netOut, maxRate }: LiveBandwidthGaugeProps) {
  const effectiveMax = useMemo(() => {
    if (maxRate && maxRate > 0) return maxRate;
    const currentMax = Math.max(netIn, netOut);
    if (currentMax === 0) return 1024 * 1024; // 1 MB/s default
    return currentMax * 2;
  }, [netIn, netOut, maxRate]);

  const noData = netIn === 0 && netOut === 0;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Live-Bandbreite</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex items-center justify-around">
          <GaugeArc
            value={netIn}
            maxValue={effectiveMax}
            color="hsl(210, 80%, 55%)"
            label="Eingehend"
          />
          <GaugeArc
            value={netOut}
            maxValue={effectiveMax}
            color="hsl(142, 71%, 45%)"
            label="Ausgehend"
          />
        </div>
        <div className="mt-3 flex justify-center gap-6 text-xs text-muted-foreground">
          <span>
            Gesamt: <strong className="text-foreground">{formatBandwidth(netIn + netOut)}</strong>
          </span>
        </div>
        {noData && (
          <p className="mt-2 text-center text-xs text-muted-foreground">
            Warte auf Netzwerk-Daten...
          </p>
        )}
      </CardContent>
    </Card>
  );
}
