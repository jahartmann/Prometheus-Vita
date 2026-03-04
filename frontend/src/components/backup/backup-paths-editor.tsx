"use client";

import { useState } from "react";
import { Plus, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";

const DEFAULT_PATHS = [
  "/etc/pve/",
  "/etc/network/",
  "/etc/hosts",
  "/etc/resolv.conf",
  "/etc/crontab",
];

interface BackupPathsEditorProps {
  paths: string[];
  onPathsChange: (paths: string[]) => void;
}

export function BackupPathsEditor({ paths, onPathsChange }: BackupPathsEditorProps) {
  const [newPath, setNewPath] = useState("");

  const togglePath = (path: string) => {
    if (paths.includes(path)) {
      onPathsChange(paths.filter((p) => p !== path));
    } else {
      onPathsChange([...paths, path]);
    }
  };

  const addCustomPath = () => {
    const trimmed = newPath.trim();
    if (trimmed && !paths.includes(trimmed)) {
      onPathsChange([...paths, trimmed]);
      setNewPath("");
    }
  };

  const removePath = (path: string) => {
    onPathsChange(paths.filter((p) => p !== path));
  };

  const allPaths = Array.from(new Set([...DEFAULT_PATHS, ...paths]));

  return (
    <div className="space-y-3">
      <Label>Backup-Pfade</Label>
      <div className="space-y-1">
        {allPaths.map((path) => {
          const isSelected = paths.includes(path);
          const isDefault = DEFAULT_PATHS.includes(path);
          return (
            <div
              key={path}
              className="flex items-center justify-between rounded px-3 py-2 text-sm hover:bg-muted"
            >
              <label className="flex items-center gap-2 cursor-pointer flex-1">
                <input
                  type="checkbox"
                  checked={isSelected}
                  onChange={() => togglePath(path)}
                  className="rounded border-input"
                />
                <span className="font-mono text-xs">{path}</span>
              </label>
              {!isDefault && (
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6"
                  onClick={() => removePath(path)}
                >
                  <X className="h-3 w-3" />
                </Button>
              )}
            </div>
          );
        })}
      </div>
      <div className="flex gap-2">
        <input
          className="flex-1 rounded-md border bg-background px-3 py-2 text-sm font-mono"
          placeholder="/pfad/zur/datei"
          value={newPath}
          onChange={(e) => setNewPath(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              addCustomPath();
            }
          }}
        />
        <Button variant="outline" size="sm" onClick={addCustomPath} disabled={!newPath.trim()}>
          <Plus className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}

BackupPathsEditor.defaultPaths = DEFAULT_PATHS;
