"use client";

import { useEffect, useState, useMemo } from "react";
import { ChevronRight, ChevronDown, File, Folder } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { backupApi, toArray } from "@/lib/api";
import type { BackupFile } from "@/types/api";

interface RestoreDialogProps {
  backupId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

interface TreeNode {
  name: string;
  path: string;
  isDir: boolean;
  children: TreeNode[];
  file?: BackupFile;
}

function buildTree(files: BackupFile[]): TreeNode[] {
  const root: TreeNode = { name: "", path: "", isDir: true, children: [] };

  for (const file of files) {
    const parts = file.file_path.split("/").filter(Boolean);
    let current = root;

    for (let i = 0; i < parts.length; i++) {
      const part = parts[i];
      const isLast = i === parts.length - 1;
      const fullPath = "/" + parts.slice(0, i + 1).join("/");

      let child = current.children.find((c) => c.name === part);
      if (!child) {
        child = {
          name: part,
          path: fullPath,
          isDir: !isLast,
          children: [],
          file: isLast ? file : undefined,
        };
        current.children.push(child);
      }
      current = child;
    }
  }

  return root.children;
}

function TreeItem({
  node,
  selectedPaths,
  onToggle,
  expandedDirs,
  onToggleExpand,
}: {
  node: TreeNode;
  selectedPaths: Set<string>;
  onToggle: (path: string) => void;
  expandedDirs: Set<string>;
  onToggleExpand: (path: string) => void;
}) {
  const isExpanded = expandedDirs.has(node.path);
  const isSelected = selectedPaths.has(node.path);

  const allChildPaths = useMemo(() => {
    const paths: string[] = [];
    function collect(n: TreeNode) {
      if (!n.isDir) paths.push(n.path);
      n.children.forEach(collect);
    }
    collect(node);
    return paths;
  }, [node]);

  const allChildrenSelected =
    node.isDir && allChildPaths.length > 0 && allChildPaths.every((p) => selectedPaths.has(p));

  return (
    <div>
      <div className="flex items-center gap-1 rounded px-2 py-1 text-sm hover:bg-muted">
        {node.isDir ? (
          <button
            className="p-0.5"
            onClick={() => onToggleExpand(node.path)}
          >
            {isExpanded ? (
              <ChevronDown className="h-3 w-3" />
            ) : (
              <ChevronRight className="h-3 w-3" />
            )}
          </button>
        ) : (
          <span className="w-4" />
        )}
        <input
          type="checkbox"
          checked={node.isDir ? allChildrenSelected : isSelected}
          onChange={() => {
            if (node.isDir) {
              allChildPaths.forEach((p) => onToggle(p));
            } else {
              onToggle(node.path);
            }
          }}
          className="rounded border-input"
        />
        {node.isDir ? (
          <Folder className="h-4 w-4 text-muted-foreground" />
        ) : (
          <File className="h-4 w-4 text-muted-foreground" />
        )}
        <span className="font-mono text-xs">{node.name}</span>
      </div>
      {node.isDir && isExpanded && (
        <div className="ml-4">
          {node.children.map((child) => (
            <TreeItem
              key={child.path}
              node={child}
              selectedPaths={selectedPaths}
              onToggle={onToggle}
              expandedDirs={expandedDirs}
              onToggleExpand={onToggleExpand}
            />
          ))}
        </div>
      )}
    </div>
  );
}

export function RestoreDialog({ backupId, open, onOpenChange }: RestoreDialogProps) {
  const [files, setFiles] = useState<BackupFile[]>([]);
  const [selectedPaths, setSelectedPaths] = useState<Set<string>>(new Set());
  const [expandedDirs, setExpandedDirs] = useState<Set<string>>(new Set());
  const [dryRun, setDryRun] = useState(true);
  const [isRestoring, setIsRestoring] = useState(false);
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (open && backupId) {
      backupApi.getBackupFiles(backupId).then((res) => {
        const fileList = toArray<BackupFile>(res.data);
        setFiles(fileList);
        // Expand top-level dirs by default
        const topDirs = new Set<string>();
        for (const f of fileList) {
          const parts = f.file_path.split("/").filter(Boolean);
          if (parts.length > 0) topDirs.add("/" + parts[0]);
        }
        setExpandedDirs(topDirs);
      });
      setSelectedPaths(new Set());
      setResult(null);
      setError(null);
    }
  }, [open, backupId]);

  const tree = useMemo(() => buildTree(files), [files]);

  const togglePath = (path: string) => {
    setSelectedPaths((prev) => {
      const next = new Set(prev);
      if (next.has(path)) {
        next.delete(path);
      } else {
        next.add(path);
      }
      return next;
    });
  };

  const toggleExpand = (path: string) => {
    setExpandedDirs((prev) => {
      const next = new Set(prev);
      if (next.has(path)) {
        next.delete(path);
      } else {
        next.add(path);
      }
      return next;
    });
  };

  const handleRestore = async () => {
    if (selectedPaths.size === 0) return;
    setIsRestoring(true);
    setError(null);
    setResult(null);
    try {
      const res = await backupApi.restoreBackup(backupId, {
        file_paths: Array.from(selectedPaths),
        dry_run: dryRun,
      });
      setResult(res.data?.data || res.data);
    } catch {
      setError("Wiederherstellung fehlgeschlagen.");
    }
    setIsRestoring(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl max-h-[80vh] overflow-auto">
        <DialogHeader>
          <DialogTitle>Backup wiederherstellen</DialogTitle>
          <DialogDescription>
            Dateien aus dem Backup zur Wiederherstellung auswaehlen.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="rounded border p-3 max-h-[40vh] overflow-auto">
            {tree.length === 0 ? (
              <p className="text-sm text-muted-foreground">Keine Dateien gefunden.</p>
            ) : (
              tree.map((node) => (
                <TreeItem
                  key={node.path}
                  node={node}
                  selectedPaths={selectedPaths}
                  onToggle={togglePath}
                  expandedDirs={expandedDirs}
                  onToggleExpand={toggleExpand}
                />
              ))
            )}
          </div>

          <div className="flex items-center gap-3">
            <Label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={dryRun}
                onChange={(e) => setDryRun(e.target.checked)}
                className="rounded border-input"
              />
              Dry-Run (Vorschau ohne Aenderungen)
            </Label>
            <Badge variant={dryRun ? "outline" : "destructive"}>
              {dryRun ? "Simulation" : "Live"}
            </Badge>
          </div>

          <p className="text-xs text-muted-foreground">
            {selectedPaths.size} Datei(en) ausgewaehlt
          </p>

          {result && (
            <div className="rounded border border-green-500/30 bg-green-500/10 p-3">
              <p className="text-sm font-medium text-green-600">
                {dryRun ? "Dry-Run Ergebnis" : "Wiederherstellung erfolgreich"}
              </p>
              <pre className="mt-1 text-xs font-mono overflow-x-auto whitespace-pre-wrap">
                {JSON.stringify(result, null, 2)}
              </pre>
            </div>
          )}

          {error && (
            <div className="rounded border border-destructive/30 bg-destructive/10 p-3">
              <p className="text-sm text-destructive">{error}</p>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Schliessen
          </Button>
          <Button
            onClick={handleRestore}
            disabled={isRestoring || selectedPaths.size === 0}
            variant={dryRun ? "default" : "destructive"}
          >
            {isRestoring
              ? "Wird ausgefuehrt..."
              : dryRun
                ? "Vorschau anzeigen"
                : "Wiederherstellen"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
