"use client";

import { useEffect, useState } from "react";
import { Plus, Pencil, Trash2, Key, MoreHorizontal } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
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
import { useUserStore } from "@/stores/user-store";
import { CreateUserDialog } from "@/components/users/create-user-dialog";
import { EditUserDialog } from "@/components/users/edit-user-dialog";
import { DeleteUserDialog } from "@/components/users/delete-user-dialog";
import { ChangePasswordDialog } from "@/components/users/change-password-dialog";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import type { UserResponse } from "@/types/api";

const roleBadgeVariant: Record<string, "default" | "secondary" | "outline"> = {
  admin: "default",
  operator: "secondary",
  viewer: "outline",
};

export default function UsersSettingsPage() {
  const { users, isLoading, fetchUsers } = useUserStore();
  const [createOpen, setCreateOpen] = useState(false);
  const [editUser, setEditUser] = useState<UserResponse | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<UserResponse | null>(null);
  const [deleteUser, setDeleteUser] = useState<UserResponse | null>(null);
  const [passwordUser, setPasswordUser] = useState<UserResponse | null>(null);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  const formatDate = (dateStr?: string | null) => {
    if (!dateStr) return null;
    return new Date(dateStr).toLocaleString("de-DE", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  const formatRelativeTime = (dateStr?: string | null): string | null => {
    if (!dateStr) return null;
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return "Gerade eben";
    if (diffMins < 60) return `vor ${diffMins} Min.`;
    if (diffHours < 24) return `vor ${diffHours} Std.`;
    if (diffDays < 7) return `vor ${diffDays} Tagen`;
    return null; // Fall back to absolute date
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">Benutzerverwaltung</h2>
          <p className="text-sm text-muted-foreground">
            Benutzer erstellen, bearbeiten und verwalten.
          </p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Benutzer erstellen
        </Button>
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Benutzername</TableHead>
                <TableHead>E-Mail</TableHead>
                <TableHead>Rolle</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Letzter Login</TableHead>
                <TableHead>Erstellt</TableHead>
                <TableHead className="w-12"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading && users.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-center py-8 text-muted-foreground">
                    Laden...
                  </TableCell>
                </TableRow>
              ) : users.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-center py-8 text-muted-foreground">
                    Keine Benutzer vorhanden.
                  </TableCell>
                </TableRow>
              ) : (
                users.map((user) => (
                  <TableRow key={user.id}>
                    <TableCell className="font-medium">{user.username}</TableCell>
                    <TableCell className="text-muted-foreground">
                      {user.email || "-"}
                    </TableCell>
                    <TableCell>
                      <Badge variant={roleBadgeVariant[user.role] || "outline"}>
                        {user.role}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <Badge variant={user.is_active ? "default" : "secondary"}>
                        {user.is_active ? "Aktiv" : "Inaktiv"}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {user.last_login ? (
                        <div title={formatDate(user.last_login) || ""}>
                          {formatRelativeTime(user.last_login) || formatDate(user.last_login)}
                        </div>
                      ) : (
                        <span className="text-muted-foreground/50 italic">Nie</span>
                      )}
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {formatDate(user.created_at) || "-"}
                    </TableCell>
                    <TableCell>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="icon">
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onClick={() => setEditUser(user)}>
                            <Pencil className="mr-2 h-4 w-4" />
                            Bearbeiten
                          </DropdownMenuItem>
                          <DropdownMenuItem onClick={() => setPasswordUser(user)}>
                            <Key className="mr-2 h-4 w-4" />
                            Passwort aendern
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            onClick={() => setDeleteTarget(user)}
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
        </CardContent>
      </Card>

      <CreateUserDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSuccess={fetchUsers}
      />
      <EditUserDialog
        user={editUser}
        open={!!editUser}
        onOpenChange={(open) => !open && setEditUser(null)}
        onSuccess={fetchUsers}
      />
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title="Benutzer loeschen?"
        description="Diese Aktion kann nicht rueckgaengig gemacht werden."
        confirmLabel="Loeschen"
        variant="destructive"
        onConfirm={() => {
          setDeleteUser(deleteTarget);
          setDeleteTarget(null);
        }}
      />
      <DeleteUserDialog
        user={deleteUser}
        open={!!deleteUser}
        onOpenChange={(open) => !open && setDeleteUser(null)}
        onSuccess={fetchUsers}
      />
      <ChangePasswordDialog
        user={passwordUser}
        open={!!passwordUser}
        onOpenChange={(open) => !open && setPasswordUser(null)}
        onSuccess={fetchUsers}
      />
    </div>
  );
}
