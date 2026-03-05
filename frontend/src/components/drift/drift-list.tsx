"use client";

import { useState } from "react";
import { GitCompare, AlertTriangle, Clock, CheckCircle, XCircle, Loader2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { DriftCheck } from "@/types/api";
import { DriftDetailDialog } from "./drift-detail-dialog";

interface DriftListProps {
  checks: DriftCheck[];
  nodeNames: Record<string, string>;
  isLoading: boolean;
}

function severityFromCheck(check: DriftCheck): "critical" | "warning" | "info" {
  if (check.status === "failed") return "critical";
  const total = check.changed_files + check.added_files + check.removed_files;
  if (total >= 5) return "critical";
  if (total >= 1) return "warning";
  return "info";
}

const severityBadge: Record<string, { label: string; className: string }> = {
  critical: { label: "Kritisch", className: "bg-red-500/10 text-red-500 border-red-500/20" },
  warning: { label: "Warnung", className: "bg-yellow-500/10 text-yellow-500 border-yellow-500/20" },
  info: { label: "OK", className: "bg-blue-500/10 text-blue-500 border-blue-500/20" },
};

const statusIcon: Record<string, React.ReactNode> = {
  completed: <CheckCircle className="h-4 w-4 text-green-500" />,
  failed: <XCircle className="h-4 w-4 text-red-500" />,
  running: <Loader2 className="h-4 w-4 text-blue-500 animate-spin" />,
  pending: <Clock className="h-4 w-4 text-muted-foreground" />,
};

export function DriftList({ checks, nodeNames, isLoading }: DriftListProps) {
  const [selectedCheck, setSelectedCheck] = useState<DriftCheck | null>(null);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (checks.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12">
        <GitCompare className="mb-3 h-10 w-10 text-muted-foreground" />
        <p className="text-muted-foreground">Noch keine Drift-Checks vorhanden.</p>
      </div>
    );
  }

  return (
    <>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Status</TableHead>
            <TableHead>Node</TableHead>
            <TableHead>Aenderungen</TableHead>
            <TableHead>Severity</TableHead>
            <TableHead>Zeitpunkt</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {checks.map((check) => {
            const severity = severityFromCheck(check);
            const badge = severityBadge[severity];
            const totalChanges = check.changed_files + check.added_files + check.removed_files;

            return (
              <TableRow
                key={check.id}
                className="cursor-pointer hover:bg-muted/50"
                onClick={() => setSelectedCheck(check)}
              >
                <TableCell>
                  <div className="flex items-center gap-2">
                    {statusIcon[check.status] || statusIcon.pending}
                    <span className="text-sm capitalize">{check.status}</span>
                  </div>
                </TableCell>
                <TableCell className="font-medium">
                  {nodeNames[check.node_id] || check.node_id.slice(0, 8)}
                </TableCell>
                <TableCell>
                  {check.status === "completed" ? (
                    <div className="flex items-center gap-2">
                      {totalChanges > 0 ? (
                        <>
                          <AlertTriangle className="h-4 w-4 text-yellow-500" />
                          <span className="text-sm">
                            {check.changed_files > 0 && `${check.changed_files} geaendert`}
                            {check.added_files > 0 && `${check.changed_files > 0 ? ", " : ""}${check.added_files} neu`}
                            {check.removed_files > 0 && `${(check.changed_files + check.added_files) > 0 ? ", " : ""}${check.removed_files} entfernt`}
                          </span>
                        </>
                      ) : (
                        <span className="text-sm text-muted-foreground">Keine Aenderungen</span>
                      )}
                    </div>
                  ) : check.status === "failed" ? (
                    <span className="text-sm text-red-500">{check.error_message || "Fehler"}</span>
                  ) : (
                    <span className="text-sm text-muted-foreground">-</span>
                  )}
                </TableCell>
                <TableCell>
                  <Badge variant="outline" className={badge.className}>
                    {badge.label}
                  </Badge>
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-1 text-sm text-muted-foreground">
                    <Clock className="h-3.5 w-3.5" />
                    {new Date(check.checked_at).toLocaleString("de-DE")}
                  </div>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>

      <DriftDetailDialog
        check={selectedCheck}
        open={!!selectedCheck}
        onOpenChange={(open) => !open && setSelectedCheck(null)}
      />
    </>
  );
}
