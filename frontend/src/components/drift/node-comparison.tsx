"use client";

import { useState } from "react";
import { GitCompare, Loader2, CheckCircle2, XCircle, Search } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { Node, NodeDifference } from "@/types/api";
import { useDriftStore } from "@/stores/drift-store";

interface NodeComparisonProps {
  nodes: Node[];
}

const defaultFilePaths = [
  "/etc/pve/storage.cfg",
  "/etc/pve/corosync.conf",
  "/etc/pve/datacenter.cfg",
  "/etc/network/interfaces",
  "/etc/hostname",
  "/etc/hosts",
  "/etc/resolv.conf",
  "/etc/fstab",
  "/etc/apt/sources.list",
  "/etc/ssh/sshd_config",
];

export function NodeComparison({ nodes }: NodeComparisonProps) {
  const { comparisonResult, isComparing, compareNodes } = useDriftStore();

  const [selectedNodes, setSelectedNodes] = useState<string[]>([]);
  const [selectedPaths, setSelectedPaths] = useState<string[]>([
    "/etc/pve/storage.cfg",
    "/etc/network/interfaces",
    "/etc/hosts",
  ]);
  const [customPath, setCustomPath] = useState("");
  const [selectedDiff, setSelectedDiff] = useState<NodeDifference | null>(null);

  const toggleNode = (nodeId: string) => {
    setSelectedNodes((prev) =>
      prev.includes(nodeId) ? prev.filter((id) => id !== nodeId) : [...prev, nodeId]
    );
  };

  const togglePath = (path: string) => {
    setSelectedPaths((prev) =>
      prev.includes(path) ? prev.filter((p) => p !== path) : [...prev, path]
    );
  };

  const addCustomPath = () => {
    if (customPath && !selectedPaths.includes(customPath)) {
      setSelectedPaths((prev) => [...prev, customPath]);
      setCustomPath("");
    }
  };

  const handleCompare = () => {
    if (selectedNodes.length >= 2 && selectedPaths.length >= 1) {
      compareNodes(selectedPaths, selectedNodes);
    }
  };

  const onlineNodes = nodes.filter((n) => n.is_online);

  return (
    <div className="space-y-4">
      <div className="grid gap-4 md:grid-cols-2">
        {/* Node Selection */}
        <Card>
          <CardContent className="p-4">
            <h3 className="text-sm font-medium mb-3">Nodes auswaehlen (min. 2)</h3>
            <div className="space-y-2 max-h-48 overflow-y-auto">
              {onlineNodes.length === 0 ? (
                <p className="text-sm text-muted-foreground">Keine Online-Nodes verfuegbar.</p>
              ) : (
                onlineNodes.map((node) => (
                  <label
                    key={node.id}
                    className="flex items-center gap-2 cursor-pointer text-sm hover:bg-muted/50 rounded p-1"
                  >
                    <Checkbox
                      checked={selectedNodes.includes(node.id)}
                      onCheckedChange={() => toggleNode(node.id)}
                    />
                    <span>{node.name}</span>
                    <span className="text-xs text-muted-foreground">({node.hostname})</span>
                  </label>
                ))
              )}
            </div>
          </CardContent>
        </Card>

        {/* File Path Selection */}
        <Card>
          <CardContent className="p-4">
            <h3 className="text-sm font-medium mb-3">Dateipfade auswaehlen</h3>
            <div className="space-y-2 max-h-36 overflow-y-auto mb-3">
              {defaultFilePaths.map((path) => (
                <label
                  key={path}
                  className="flex items-center gap-2 cursor-pointer text-sm hover:bg-muted/50 rounded p-1"
                >
                  <Checkbox
                    checked={selectedPaths.includes(path)}
                    onCheckedChange={() => togglePath(path)}
                  />
                  <code className="text-xs">{path}</code>
                </label>
              ))}
              {/* Show custom paths that aren't in defaults */}
              {selectedPaths
                .filter((p) => !defaultFilePaths.includes(p))
                .map((path) => (
                  <label
                    key={path}
                    className="flex items-center gap-2 cursor-pointer text-sm hover:bg-muted/50 rounded p-1"
                  >
                    <Checkbox
                      checked
                      onCheckedChange={() => togglePath(path)}
                    />
                    <code className="text-xs">{path}</code>
                  </label>
                ))}
            </div>
            <div className="flex gap-2">
              <Input
                placeholder="Eigener Pfad, z.B. /etc/cron.d/backup"
                value={customPath}
                onChange={(e) => setCustomPath(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && addCustomPath()}
                className="text-sm h-8"
              />
              <Button variant="outline" size="sm" onClick={addCustomPath} className="h-8">
                +
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>

      <Button
        onClick={handleCompare}
        disabled={selectedNodes.length < 2 || selectedPaths.length < 1 || isComparing}
        className="w-full"
      >
        {isComparing ? (
          <>
            <Loader2 className="h-4 w-4 mr-2 animate-spin" />
            Vergleiche...
          </>
        ) : (
          <>
            <Search className="h-4 w-4 mr-2" />
            Nodes vergleichen ({selectedNodes.length} Nodes, {selectedPaths.length} Dateien)
          </>
        )}
      </Button>

      {/* Comparison Results */}
      {comparisonResult && (
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Datei</TableHead>
                  {comparisonResult.comparisons[0]?.node_files.map((nf) => (
                    <TableHead key={nf.node_id} className="text-center">
                      {nf.node_name}
                    </TableHead>
                  ))}
                  <TableHead className="text-center">Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {comparisonResult.comparisons.map((comp) => {
                  const allIdentical = comp.differences.every((d) => d.identical);
                  const hasErrors = comp.node_files.some((nf) => nf.error);

                  return (
                    <TableRow key={comp.file_path}>
                      <TableCell>
                        <code className="text-xs font-mono">{comp.file_path}</code>
                      </TableCell>
                      {comp.node_files.map((nf) => (
                        <TableCell key={nf.node_id} className="text-center">
                          {nf.error ? (
                            <Badge variant="outline" className="bg-red-500/10 text-red-500 border-red-500/20 text-xs">
                              Fehler
                            </Badge>
                          ) : (
                            <Badge variant="outline" className="bg-green-500/10 text-green-500 border-green-500/20 text-xs">
                              OK
                            </Badge>
                          )}
                        </TableCell>
                      ))}
                      <TableCell className="text-center">
                        {hasErrors ? (
                          <Badge variant="outline" className="bg-red-500/10 text-red-500 border-red-500/20">
                            <XCircle className="h-3 w-3 mr-1" />
                            Fehler
                          </Badge>
                        ) : allIdentical ? (
                          <Badge variant="outline" className="bg-green-500/10 text-green-500 border-green-500/20">
                            <CheckCircle2 className="h-3 w-3 mr-1" />
                            Identisch
                          </Badge>
                        ) : (
                          <div className="flex flex-wrap justify-center gap-1">
                            {comp.differences
                              .filter((d) => !d.identical)
                              .map((d, i) => (
                                <Button
                                  key={i}
                                  variant="outline"
                                  size="sm"
                                  className="h-6 text-xs bg-yellow-500/10 text-yellow-600 border-yellow-500/20 hover:bg-yellow-500/20"
                                  onClick={() => setSelectedDiff(d)}
                                >
                                  <GitCompare className="h-3 w-3 mr-1" />
                                  {d.node_a_name} vs {d.node_b_name}
                                </Button>
                              ))}
                          </div>
                        )}
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {/* Diff Dialog */}
      <Dialog open={!!selectedDiff} onOpenChange={(open) => !open && setSelectedDiff(null)}>
        <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>
              Unterschied: {selectedDiff?.node_a_name} vs {selectedDiff?.node_b_name}
            </DialogTitle>
          </DialogHeader>
          {selectedDiff && selectedDiff.diff && (
            <pre className="text-xs bg-muted p-3 rounded overflow-x-auto whitespace-pre-wrap font-mono">
              {selectedDiff.diff}
            </pre>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
