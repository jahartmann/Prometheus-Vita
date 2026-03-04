"use client";

import { ArrowDown, ArrowUp, Minus } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { ResourceRecommendation } from "@/types/api";

interface RecommendationListProps {
  recommendations: ResourceRecommendation[];
  getNodeName?: (nodeId: string) => string;
}

const typeIcons: Record<string, typeof ArrowDown> = {
  downsize: ArrowDown,
  upsize: ArrowUp,
  optimal: Minus,
};

const typeLabels: Record<string, string> = {
  downsize: "Verkleinern",
  upsize: "Vergroessern",
  optimal: "Optimal",
};

const typeBadge: Record<string, "default" | "secondary" | "outline"> = {
  downsize: "secondary",
  upsize: "default",
  optimal: "outline",
};

function formatValue(value: number, type: string): string {
  if (type === "memory" || type === "disk") {
    if (value >= 1073741824) return `${(value / 1073741824).toFixed(1)} GB`;
    if (value >= 1048576) return `${(value / 1048576).toFixed(0)} MB`;
    return `${value} B`;
  }
  return String(value);
}

export function RecommendationList({ recommendations, getNodeName }: RecommendationListProps) {
  if (recommendations.length === 0) {
    return (
      <p className="text-sm text-muted-foreground py-8 text-center">
        Keine Empfehlungen vorhanden.
      </p>
    );
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          {getNodeName && <TableHead>Node</TableHead>}
          <TableHead>VM</TableHead>
          <TableHead>Ressource</TableHead>
          <TableHead>Aktuell</TableHead>
          <TableHead>Empfohlen</TableHead>
          <TableHead>Auslastung</TableHead>
          <TableHead>Empfehlung</TableHead>
          <TableHead>Grund</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {recommendations.map((rec) => {
          const Icon = typeIcons[rec.recommendation_type] || Minus;
          return (
            <TableRow key={rec.id}>
              {getNodeName && (
                <TableCell className="text-muted-foreground">
                  {getNodeName(rec.node_id)}
                </TableCell>
              )}
              <TableCell className="font-medium">
                {rec.vm_name || `VM ${rec.vmid}`}
                <span className="text-muted-foreground text-xs ml-1">({rec.vm_type})</span>
              </TableCell>
              <TableCell>
                <Badge variant="outline">{rec.resource_type}</Badge>
              </TableCell>
              <TableCell className="font-mono text-sm">
                {formatValue(rec.current_value, rec.resource_type)}
              </TableCell>
              <TableCell className="font-mono text-sm">
                {formatValue(rec.recommended_value, rec.resource_type)}
              </TableCell>
              <TableCell className="text-sm">
                <span className="text-muted-foreground">
                  avg {(rec.avg_usage ?? 0).toFixed(1)}% / max {(rec.max_usage ?? 0).toFixed(1)}%
                </span>
              </TableCell>
              <TableCell>
                <Badge variant={typeBadge[rec.recommendation_type] || "outline"}>
                  <Icon className="h-3 w-3 mr-1" />
                  {typeLabels[rec.recommendation_type] || rec.recommendation_type}
                </Badge>
              </TableCell>
              <TableCell className="text-muted-foreground text-sm max-w-[200px] truncate">
                {rec.reason || "-"}
              </TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}
