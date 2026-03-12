"use client";

import { useEffect, useState, useCallback } from "react";
import {
  Folder,
  File,
  Link2,
  ChevronRight,
  ArrowUp,
  Upload,
  FilePlus,
  FolderPlus,
  Trash2,
  Download,
  Loader2,
  Bookmark,
  RefreshCw,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useVMCockpitStore } from "@/stores/vm-cockpit-store";
import { FileEditor } from "./file-editor";
import { CockpitError } from "./cockpit-error";
import type { VMFileEntry } from "@/types/api";

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} K`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} M`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} G`;
}

export function FilesTab() {
  const {
    currentPath,
    files,
    isLoadingFiles,
    filesError,
    openFilePath,
    openFileContent,
    openFileOriginal,
    isLoadingFile,
    isSavingFile,
    bookmarks,
    navigateTo,
    openFile,
    closeFile,
    saveFile,
    deleteEntry,
    createDirectory,
    createFile,
    uploadFile,
  } = useVMCockpitStore();

  const [showNewFileDialog, setShowNewFileDialog] = useState(false);
  const [showNewFolderDialog, setShowNewFolderDialog] = useState(false);
  const [newName, setNewName] = useState("");
  const [selectedEntry, setSelectedEntry] = useState<VMFileEntry | null>(null);
  const [editContent, setEditContent] = useState("");

  useEffect(() => {
    navigateTo("/");
  }, [navigateTo]);

  useEffect(() => {
    if (openFileContent !== null) {
      setEditContent(openFileContent);
    }
  }, [openFileContent]);

  const pathSegments = currentPath.split("/").filter(Boolean);

  const handleNavigate = useCallback(
    (entry: VMFileEntry) => {
      if (entry.type === "directory") {
        const newPath =
          currentPath === "/"
            ? `/${entry.name}`
            : `${currentPath}/${entry.name}`;
        navigateTo(newPath);
      } else {
        const filePath =
          currentPath === "/"
            ? `/${entry.name}`
            : `${currentPath}/${entry.name}`;
        openFile(filePath);
      }
    },
    [currentPath, navigateTo, openFile]
  );

  const handleGoUp = useCallback(() => {
    if (currentPath === "/") return;
    const parts = currentPath.split("/").filter(Boolean);
    parts.pop();
    navigateTo("/" + parts.join("/") || "/");
  }, [currentPath, navigateTo]);

  const handleBreadcrumb = useCallback(
    (index: number) => {
      const path = "/" + pathSegments.slice(0, index + 1).join("/");
      navigateTo(path);
    },
    [pathSegments, navigateTo]
  );

  const handleCreateFile = useCallback(() => {
    if (!newName.trim()) return;
    const path =
      currentPath === "/"
        ? `/${newName.trim()}`
        : `${currentPath}/${newName.trim()}`;
    createFile(path);
    setNewName("");
    setShowNewFileDialog(false);
  }, [newName, currentPath, createFile]);

  const handleCreateFolder = useCallback(() => {
    if (!newName.trim()) return;
    const path =
      currentPath === "/"
        ? `/${newName.trim()}`
        : `${currentPath}/${newName.trim()}`;
    createDirectory(path);
    setNewName("");
    setShowNewFolderDialog(false);
  }, [newName, currentPath, createDirectory]);

  const handleDelete = useCallback(
    (entry: VMFileEntry) => {
      const path =
        currentPath === "/"
          ? `/${entry.name}`
          : `${currentPath}/${entry.name}`;
      if (confirm(`"${entry.name}" wirklich loeschen?`)) {
        deleteEntry(path);
      }
    },
    [currentPath, deleteEntry]
  );

  const handleUpload = useCallback(() => {
    const input = document.createElement("input");
    input.type = "file";
    input.onchange = (e) => {
      const file = (e.target as HTMLInputElement).files?.[0];
      if (!file) return;
      const reader = new FileReader();
      reader.onload = () => {
        const result = reader.result as string;
        // Extract base64 content after the data URL prefix
        const base64 = result.includes(",") ? result.split(",")[1] : result;
        const path =
          currentPath === "/"
            ? `/${file.name}`
            : `${currentPath}/${file.name}`;
        uploadFile(path, base64);
      };
      reader.readAsDataURL(file);
    };
    input.click();
  }, [currentPath, uploadFile]);

  const handleDownload = useCallback(
    async (entry: VMFileEntry) => {
      const path =
        currentPath === "/"
          ? `/${entry.name}`
          : `${currentPath}/${entry.name}`;
      // We open the file which triggers a read, then offer download
      openFile(path);
    },
    [currentPath, openFile]
  );

  const handleSave = useCallback(() => {
    if (openFilePath) {
      saveFile(openFilePath, editContent);
    }
  }, [openFilePath, editContent, saveFile]);

  const handleDiscard = useCallback(() => {
    if (openFileOriginal !== null) {
      setEditContent(openFileOriginal);
    }
  }, [openFileOriginal]);

  // If file editor is open, show it
  if (openFilePath) {
    return (
      <FileEditor
        path={openFilePath}
        content={editContent}
        original={openFileOriginal ?? ""}
        isLoading={isLoadingFile}
        isSaving={isSavingFile}
        onChange={setEditContent}
        onSave={handleSave}
        onDiscard={handleDiscard}
        onClose={closeFile}
      />
    );
  }

  if (filesError) {
    return <CockpitError {...filesError} onRetry={() => navigateTo(currentPath)} />;
  }

  return (
    <div className="space-y-4">
      {/* Toolbar */}
      <div className="flex flex-wrap items-center gap-2">
        <Button variant="outline" size="sm" onClick={handleUpload}>
          <Upload className="mr-1.5 h-3.5 w-3.5" />
          Upload
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            setShowNewFileDialog(true);
            setShowNewFolderDialog(false);
            setNewName("");
          }}
        >
          <FilePlus className="mr-1.5 h-3.5 w-3.5" />
          Neue Datei
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            setShowNewFolderDialog(true);
            setShowNewFileDialog(false);
            setNewName("");
          }}
        >
          <FolderPlus className="mr-1.5 h-3.5 w-3.5" />
          Neuer Ordner
        </Button>
        <div className="flex-1" />
        <Button
          variant="ghost"
          size="sm"
          onClick={() => navigateTo(currentPath)}
        >
          <RefreshCw className="h-3.5 w-3.5" />
        </Button>
      </div>

      {/* New file/folder input */}
      {(showNewFileDialog || showNewFolderDialog) && (
        <div className="flex items-center gap-2">
          <Input
            placeholder={
              showNewFileDialog ? "Dateiname..." : "Ordnername..."
            }
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                showNewFileDialog ? handleCreateFile() : handleCreateFolder();
              }
              if (e.key === "Escape") {
                setShowNewFileDialog(false);
                setShowNewFolderDialog(false);
              }
            }}
            autoFocus
            className="max-w-xs"
          />
          <Button
            size="sm"
            onClick={
              showNewFileDialog ? handleCreateFile : handleCreateFolder
            }
          >
            Erstellen
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => {
              setShowNewFileDialog(false);
              setShowNewFolderDialog(false);
            }}
          >
            Abbrechen
          </Button>
        </div>
      )}

      {/* Bookmarks */}
      <div className="flex items-center gap-1.5 flex-wrap">
        <Bookmark className="h-3.5 w-3.5 text-muted-foreground" />
        {bookmarks.map((bm) => (
          <Button
            key={bm}
            variant={currentPath === bm ? "secondary" : "ghost"}
            size="sm"
            className="h-7 text-xs"
            onClick={() => navigateTo(bm)}
          >
            {bm}
          </Button>
        ))}
      </div>

      {/* Breadcrumb */}
      <div className="flex items-center gap-1 text-sm">
        <Button
          variant="ghost"
          size="sm"
          className="h-7 px-2 text-xs font-mono"
          onClick={() => navigateTo("/")}
        >
          /
        </Button>
        {pathSegments.map((seg, i) => (
          <div key={i} className="flex items-center gap-1">
            <ChevronRight className="h-3 w-3 text-muted-foreground" />
            <Button
              variant="ghost"
              size="sm"
              className="h-7 px-2 text-xs font-mono"
              onClick={() => handleBreadcrumb(i)}
            >
              {seg}
            </Button>
          </div>
        ))}
      </div>

      {/* File listing */}
      <Card>
        <CardContent className="p-0">
          {isLoadingFiles ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[40%]">Name</TableHead>
                  <TableHead>Rechte</TableHead>
                  <TableHead>Besitzer</TableHead>
                  <TableHead className="text-right">Groesse</TableHead>
                  <TableHead>Geaendert</TableHead>
                  <TableHead className="w-[80px]" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {/* Go up row */}
                {currentPath !== "/" && (
                  <TableRow
                    className="cursor-pointer hover:bg-muted/50"
                    onClick={handleGoUp}
                  >
                    <TableCell colSpan={6} className="text-sm text-muted-foreground">
                      <div className="flex items-center gap-2">
                        <ArrowUp className="h-4 w-4" />
                        ..
                      </div>
                    </TableCell>
                  </TableRow>
                )}
                {files.length === 0 && !isLoadingFiles && (
                  <TableRow>
                    <TableCell
                      colSpan={6}
                      className="py-8 text-center text-muted-foreground"
                    >
                      Verzeichnis ist leer
                    </TableCell>
                  </TableRow>
                )}
                {files
                  .sort((a, b) => {
                    // Directories first, then alphabetically
                    if (a.type === "directory" && b.type !== "directory") return -1;
                    if (a.type !== "directory" && b.type === "directory") return 1;
                    return a.name.localeCompare(b.name);
                  })
                  .map((entry) => (
                    <TableRow
                      key={entry.name}
                      className="cursor-pointer hover:bg-muted/50"
                      onClick={() => handleNavigate(entry)}
                    >
                      <TableCell>
                        <div className="flex items-center gap-2">
                          {entry.type === "directory" ? (
                            <Folder className="h-4 w-4 text-blue-500" />
                          ) : entry.type === "symlink" ? (
                            <Link2 className="h-4 w-4 text-purple-500" />
                          ) : (
                            <File className="h-4 w-4 text-muted-foreground" />
                          )}
                          <span className="text-sm font-mono">{entry.name}</span>
                          {entry.link_target && (
                            <span className="text-xs text-muted-foreground">
                              → {entry.link_target}
                            </span>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        <span className="font-mono text-xs text-muted-foreground">
                          {entry.permissions}
                        </span>
                      </TableCell>
                      <TableCell>
                        <span className="text-xs text-muted-foreground">
                          {entry.owner}:{entry.group}
                        </span>
                      </TableCell>
                      <TableCell className="text-right">
                        <span className="text-xs text-muted-foreground tabular-nums">
                          {entry.type === "directory" ? "-" : formatFileSize(entry.size)}
                        </span>
                      </TableCell>
                      <TableCell>
                        <span className="text-xs text-muted-foreground">
                          {entry.modified}
                        </span>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
                          {entry.type === "file" && (
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-7 w-7"
                              title="Herunterladen"
                              onClick={() => handleDownload(entry)}
                            >
                              <Download className="h-3.5 w-3.5" />
                            </Button>
                          )}
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-7 w-7 text-destructive hover:text-destructive"
                            title="Loeschen"
                            onClick={() => handleDelete(entry)}
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
