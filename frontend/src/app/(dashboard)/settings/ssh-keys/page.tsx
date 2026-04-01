"use client";

import { useEffect, useState } from "react";
import { useNodeStore } from "@/stores/node-store";
import { Card, CardContent } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { KeyList } from "@/components/sshkeys/key-list";
import { GenerateKeyDialog } from "@/components/sshkeys/generate-key-dialog";

export default function SSHKeysPage() {
  const { nodes, fetchNodes } = useNodeStore();
  const [selectedNode, setSelectedNode] = useState("");
  const [generateOpen, setGenerateOpen] = useState(false);
  const [refreshKey, setRefreshKey] = useState(0);

  useEffect(() => {
    fetchNodes();
  }, [fetchNodes]);

  useEffect(() => {
    if (nodes.length > 0 && !selectedNode) {
      setSelectedNode(nodes[0].id);
    }
  }, [nodes, selectedNode]);

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-lg font-semibold">SSH-Schlüsselverwaltung</h2>
        <p className="text-sm text-muted-foreground">
          SSH-Schlüssel für Nodes generieren, deployen und verwalten.
        </p>
      </div>

      <div className="max-w-xs">
        <Select value={selectedNode} onValueChange={setSelectedNode}>
          <SelectTrigger>
            <SelectValue placeholder="Node wählen..." />
          </SelectTrigger>
          <SelectContent>
            {nodes.map((node) => (
              <SelectItem key={node.id} value={node.id}>
                {node.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {selectedNode ? (
        <Card>
          <CardContent className="p-4">
            <KeyList
              key={refreshKey}
              nodeId={selectedNode}
              onGenerate={() => setGenerateOpen(true)}
            />
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent className="p-8 text-center text-muted-foreground">
            Bitte wählen Sie einen Node aus.
          </CardContent>
        </Card>
      )}

      {selectedNode && (
        <GenerateKeyDialog
          nodeId={selectedNode}
          open={generateOpen}
          onOpenChange={setGenerateOpen}
          onSuccess={() => setRefreshKey((k) => k + 1)}
        />
      )}
    </div>
  );
}
