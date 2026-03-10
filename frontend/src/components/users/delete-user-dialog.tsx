"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { userApi } from "@/lib/api";
import { toast } from "sonner";
import type { UserResponse } from "@/types/api";

interface DeleteUserDialogProps {
  user: UserResponse | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function DeleteUserDialog({ user, open, onOpenChange, onSuccess }: DeleteUserDialogProps) {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleDelete = async () => {
    if (!user) return;
    setIsLoading(true);
    setError(null);

    try {
      await userApi.delete(user.id);
      onOpenChange(false);
      onSuccess();
      toast.success("Benutzer geloescht");
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { error?: string } } };
      const msg = axiosErr.response?.data?.error || "Benutzer konnte nicht geloescht werden";
      setError(msg);
      toast.error(`Fehler: ${msg}`);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Benutzer loeschen</DialogTitle>
          <DialogDescription>
            Sind Sie sicher, dass Sie den Benutzer &quot;{user?.username}&quot; loeschen moechten?
            Diese Aktion kann nicht rueckgaengig gemacht werden.
          </DialogDescription>
        </DialogHeader>

        {error && (
          <p className="text-sm text-destructive">{error}</p>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button variant="destructive" onClick={handleDelete} disabled={isLoading}>
            {isLoading ? "Loesche..." : "Loeschen"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
