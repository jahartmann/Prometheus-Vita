"use client";

import { useEffect, useState } from "react";
import { Pencil, Trash2, MoreHorizontal } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useEnvironmentStore } from "@/stores/environment-store";
import { EnvironmentForm } from "./environment-form";
import type { Environment } from "@/types/api";

export function EnvironmentList() {
  const { environments, isLoading, fetchEnvironments, deleteEnvironment } = useEnvironmentStore();
  const [editEnv, setEditEnv] = useState<Environment | null>(null);
  const [createOpen, setCreateOpen] = useState(false);

  useEffect(() => {
    fetchEnvironments();
  }, [fetchEnvironments]);

  const handleDelete = async (id: string) => {
    try {
      await deleteEnvironment(id);
    } catch {
      // Fehler
    }
  };

  return (
    <>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Beschreibung</TableHead>
            <TableHead>Farbe</TableHead>
            <TableHead>Erstellt</TableHead>
            <TableHead className="w-12"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading && environments.length === 0 ? (
            <TableRow>
              <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                Laden...
              </TableCell>
            </TableRow>
          ) : environments.length === 0 ? (
            <TableRow>
              <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                Keine Umgebungen vorhanden.
              </TableCell>
            </TableRow>
          ) : (
            environments.map((env) => (
              <TableRow key={env.id}>
                <TableCell className="font-medium">
                  <div className="flex items-center gap-2">
                    <div
                      className="h-3 w-3 rounded-full"
                      style={{ backgroundColor: env.color || "#6b7280" }}
                    />
                    {env.name}
                  </div>
                </TableCell>
                <TableCell className="text-muted-foreground">{env.description || "-"}</TableCell>
                <TableCell>
                  <Badge variant="outline" style={{ borderColor: env.color }}>
                    {env.color}
                  </Badge>
                </TableCell>
                <TableCell className="text-muted-foreground">
                  {new Date(env.created_at).toLocaleString("de-DE")}
                </TableCell>
                <TableCell>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="ghost" size="icon">
                        <MoreHorizontal className="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem onClick={() => setEditEnv(env)}>
                        <Pencil className="mr-2 h-4 w-4" />
                        Bearbeiten
                      </DropdownMenuItem>
                      <DropdownMenuItem
                        onClick={() => handleDelete(env.id)}
                        className="text-destructive"
                      >
                        <Trash2 className="mr-2 h-4 w-4" />
                        Loeschen
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>

      <EnvironmentForm
        environment={editEnv}
        open={!!editEnv || createOpen}
        onOpenChange={(open) => {
          if (!open) {
            setEditEnv(null);
            setCreateOpen(false);
          }
        }}
        onSuccess={fetchEnvironments}
      />
    </>
  );
}
