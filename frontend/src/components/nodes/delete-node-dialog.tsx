"use client";

import { useState } from "react";
import { Loader2, AlertTriangle } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import api from "@/lib/api";
import { toast } from "sonner";
import { useNodeStore } from "@/stores/node-store";
import type { Node } from "@/types/api";

interface DeleteNodeDialogProps {
  node: Node | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function DeleteNodeDialog({
  node,
  open,
  onOpenChange,
}: DeleteNodeDialogProps) {
  const { removeNode } = useNodeStore();
  const [confirmName, setConfirmName] = useState("");
  const [isDeleting, setIsDeleting] = useState(false);

  const handleDelete = async () => {
    if (!node || confirmName !== node.name) return;

    setIsDeleting(true);
    try {
      await api.delete(`/nodes/${node.id}`);
      removeNode(node.id);
      onOpenChange(false);
      setConfirmName("");
      toast.success("Node entfernt");
    } catch {
      toast.error("Fehler beim Entfernen des Nodes");
    }
    setIsDeleting(false);
  };

  const handleClose = (o: boolean) => {
    onOpenChange(o);
    if (!o) setConfirmName("");
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <AlertTriangle className="h-5 w-5 text-destructive" />
            Node entfernen
          </DialogTitle>
          <DialogDescription>
            Diese Aktion kann nicht rückgängig gemacht werden. Der Node{" "}
            <strong>{node?.name}</strong> wird aus Prometheus entfernt.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-2">
          <Label htmlFor="confirm-name">
            Geben Sie <strong>{node?.name}</strong> ein, um zu bestätigen:
          </Label>
          <Input
            id="confirm-name"
            value={confirmName}
            onChange={(e) => setConfirmName(e.target.value)}
            placeholder={node?.name}
          />
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => handleClose(false)}>
            Abbrechen
          </Button>
          <Button
            variant="destructive"
            onClick={handleDelete}
            disabled={isDeleting || confirmName !== node?.name}
          >
            {isDeleting ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Wird entfernt...
              </>
            ) : (
              "Node entfernen"
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
