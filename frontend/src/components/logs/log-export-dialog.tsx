"use client";

import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { logAnalysisApi } from "@/lib/api";
import { Download } from "lucide-react";

type ExportFormat = "text" | "csv" | "json";

interface LogExportDialogProps {
  nodeId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function LogExportDialog({
  nodeId,
  open,
  onOpenChange,
}: LogExportDialogProps) {
  const [format, setFormat] = useState<ExportFormat>("text");
  const [includeAnnotations, setIncludeAnnotations] = useState(true);
  const [isLoading, setIsLoading] = useState(false);

  const handleDownload = async () => {
    setIsLoading(true);
    try {
      const response = await logAnalysisApi.exportLogs({
        node_id: nodeId,
        format,
        include_annotations: includeAnnotations ? "true" : "false",
      });

      const mimeTypes: Record<ExportFormat, string> = {
        text: "text/plain",
        csv: "text/csv",
        json: "application/json",
      };

      const blob = new Blob([response.data as BlobPart], {
        type: mimeTypes[format],
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `logs-${nodeId}-${new Date().toISOString().slice(0, 10)}.${format}`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);

      onOpenChange(false);
    } catch {
      // ignore
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-sm border-zinc-800 bg-zinc-900">
        <DialogHeader>
          <DialogTitle>Logs exportieren</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {/* Format selection */}
          <div className="space-y-1.5">
            <label className="text-sm text-zinc-400">Format</label>
            <Select
              value={format}
              onValueChange={(v) => setFormat(v as ExportFormat)}
            >
              <SelectTrigger className="border-zinc-700 bg-zinc-800">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="text">Plaintext (.txt)</SelectItem>
                <SelectItem value="csv">CSV (.csv)</SelectItem>
                <SelectItem value="json">JSON (.json)</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Include AI annotations */}
          <div className="flex items-center justify-between">
            <label
              className="text-sm text-zinc-400 cursor-pointer select-none"
              onClick={() => setIncludeAnnotations(!includeAnnotations)}
            >
              KI-Annotationen einbeziehen
            </label>
            <Switch
              checked={includeAnnotations}
              onCheckedChange={setIncludeAnnotations}
            />
          </div>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            className="border-zinc-700"
          >
            Abbrechen
          </Button>
          <Button
            onClick={handleDownload}
            disabled={isLoading}
            className="gap-1.5"
          >
            <Download className="h-4 w-4" />
            {isLoading ? "Exportiere..." : "Herunterladen"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
