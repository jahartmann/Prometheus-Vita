"use client";

import { useState, useMemo } from "react";
import { ArrowLeft, Save, Undo2, Eye, Loader2, FileCode } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

interface FileEditorProps {
  path: string;
  content: string;
  original: string;
  isLoading: boolean;
  isSaving: boolean;
  onChange: (content: string) => void;
  onSave: () => void;
  onDiscard: () => void;
  onClose: () => void;
}

function computeDiff(original: string, current: string): { added: number; removed: number; lines: DiffLine[] } {
  const origLines = original.split("\n");
  const currLines = current.split("\n");
  const lines: DiffLine[] = [];
  let added = 0;
  let removed = 0;

  const maxLen = Math.max(origLines.length, currLines.length);
  for (let i = 0; i < maxLen; i++) {
    const origLine = i < origLines.length ? origLines[i] : undefined;
    const currLine = i < currLines.length ? currLines[i] : undefined;

    if (origLine === currLine) {
      lines.push({ type: "unchanged", content: currLine ?? "", lineNumber: i + 1 });
    } else {
      if (origLine !== undefined && currLine !== undefined) {
        lines.push({ type: "removed", content: origLine, lineNumber: i + 1 });
        lines.push({ type: "added", content: currLine, lineNumber: i + 1 });
        added++;
        removed++;
      } else if (origLine === undefined) {
        lines.push({ type: "added", content: currLine ?? "", lineNumber: i + 1 });
        added++;
      } else {
        lines.push({ type: "removed", content: origLine, lineNumber: i + 1 });
        removed++;
      }
    }
  }
  return { added, removed, lines };
}

interface DiffLine {
  type: "added" | "removed" | "unchanged";
  content: string;
  lineNumber: number;
}

export function FileEditor({
  path,
  content,
  original,
  isLoading,
  isSaving,
  onChange,
  onSave,
  onDiscard,
  onClose,
}: FileEditorProps) {
  const [showDiff, setShowDiff] = useState(false);
  const hasChanges = content !== original;
  const fileName = path.split("/").pop() || path;

  const diff = useMemo(() => {
    if (!showDiff || !hasChanges) return null;
    return computeDiff(original, content);
  }, [showDiff, hasChanges, original, content]);

  const lineNumbers = useMemo(() => {
    const lines = content.split("\n");
    return lines.map((_, i) => i + 1);
  }, [content]);

  if (isLoading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          <span className="ml-2 text-muted-foreground">Datei wird geladen...</span>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="sm" onClick={onClose}>
          <ArrowLeft className="mr-1.5 h-3.5 w-3.5" />
          Zurueck
        </Button>
        <div className="flex items-center gap-2 min-w-0 flex-1">
          <FileCode className="h-4 w-4 text-muted-foreground shrink-0" />
          <span className="font-mono text-sm truncate">{path}</span>
          {hasChanges && (
            <Badge variant="warning" className="shrink-0">
              Geaendert
            </Badge>
          )}
        </div>
      </div>

      {/* Actions */}
      <div className="flex items-center gap-2">
        <Button size="sm" onClick={onSave} disabled={!hasChanges || isSaving}>
          {isSaving ? (
            <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
          ) : (
            <Save className="mr-1.5 h-3.5 w-3.5" />
          )}
          Speichern
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={onDiscard}
          disabled={!hasChanges}
        >
          <Undo2 className="mr-1.5 h-3.5 w-3.5" />
          Verwerfen
        </Button>
        <Button
          variant={showDiff ? "secondary" : "outline"}
          size="sm"
          onClick={() => setShowDiff(!showDiff)}
          disabled={!hasChanges}
        >
          <Eye className="mr-1.5 h-3.5 w-3.5" />
          Diff anzeigen
        </Button>
      </div>

      {/* Diff view */}
      {showDiff && diff && (
        <Card>
          <CardHeader className="py-3 px-4">
            <CardTitle className="text-sm flex items-center gap-2">
              Aenderungen
              <Badge variant="success" className="text-xs">
                +{diff.added}
              </Badge>
              <Badge variant="destructive" className="text-xs">
                -{diff.removed}
              </Badge>
            </CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <div className="max-h-[300px] overflow-auto">
              <pre className="text-xs font-mono p-4">
                {diff.lines.map((line, i) => (
                  <div
                    key={i}
                    className={
                      line.type === "added"
                        ? "bg-green-500/10 text-green-700 dark:text-green-400"
                        : line.type === "removed"
                        ? "bg-red-500/10 text-red-700 dark:text-red-400"
                        : "text-muted-foreground"
                    }
                  >
                    <span className="inline-block w-5 text-right mr-3 select-none opacity-50">
                      {line.type === "removed" ? "-" : line.type === "added" ? "+" : " "}
                    </span>
                    {line.content}
                  </div>
                ))}
              </pre>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Editor */}
      <Card>
        <CardContent className="p-0">
          <div className="relative flex max-h-[600px] overflow-auto">
            {/* Line numbers */}
            <div className="sticky left-0 select-none border-r bg-muted/30 px-3 py-4 text-right font-mono text-xs text-muted-foreground">
              {lineNumbers.map((n) => (
                <div key={n} className="leading-5">
                  {n}
                </div>
              ))}
            </div>
            {/* Text area */}
            <textarea
              value={content}
              onChange={(e) => onChange(e.target.value)}
              className="flex-1 resize-none bg-transparent p-4 font-mono text-xs leading-5 outline-none min-h-[400px] w-full"
              spellCheck={false}
              wrap="off"
            />
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
