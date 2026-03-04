"use client";

import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { DriftCheck, DriftFileDetail } from "@/types/api";

interface DriftDetailDialogProps {
  check: DriftCheck | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const statusLabel: Record<string, string> = {
  added: "Hinzugefuegt",
  removed: "Entfernt",
  modified: "Geaendert",
  unchanged: "Unveraendert",
};

const statusColor: Record<string, "default" | "secondary" | "outline"> = {
  added: "default",
  removed: "secondary",
  modified: "outline",
};

export function DriftDetailDialog({ check, open, onOpenChange }: DriftDetailDialogProps) {
  if (!check) return null;

  const details: DriftFileDetail[] = Array.isArray(check.details)
    ? check.details
    : [];

  const filtered = details.filter((d) => d.status !== "unchanged");

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Drift-Details</DialogTitle>
        </DialogHeader>

        <div className="space-y-2 text-sm mb-4">
          <p className="text-muted-foreground">
            Geprueft: {new Date(check.checked_at).toLocaleString("de-DE")} |{" "}
            {check.total_files} Dateien total
          </p>
          <div className="flex gap-2">
            {check.changed_files > 0 && <Badge variant="outline">{check.changed_files} geaendert</Badge>}
            {check.added_files > 0 && <Badge variant="default">{check.added_files} hinzugefuegt</Badge>}
            {check.removed_files > 0 && <Badge variant="secondary">{check.removed_files} entfernt</Badge>}
          </div>
        </div>

        {filtered.length === 0 ? (
          <p className="text-muted-foreground text-sm">Keine geaenderten Dateien.</p>
        ) : (
          <div className="space-y-3">
            {filtered.map((file) => (
              <div key={file.file_path} className="border rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <code className="text-xs font-mono">{file.file_path}</code>
                  <Badge variant={statusColor[file.status] || "outline"}>
                    {statusLabel[file.status] || file.status}
                  </Badge>
                </div>
                {file.diff && (
                  <pre className="mt-2 text-xs bg-muted p-2 rounded overflow-x-auto whitespace-pre-wrap">
                    {file.diff}
                  </pre>
                )}
              </div>
            ))}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
