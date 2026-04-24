"use client";

import { useEffect, useState } from "react";
import { Ban, Check, Copy, Key, MoreHorizontal, Pencil, Plus, RefreshCw, ShieldOff, Trash2, UserPlus, WifiOff } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ChangePasswordDialog } from "@/components/users/change-password-dialog";
import { CreateUserDialog } from "@/components/users/create-user-dialog";
import { DeleteUserDialog } from "@/components/users/delete-user-dialog";
import { EditUserDialog } from "@/components/users/edit-user-dialog";
import { userApi, toArray } from "@/lib/api";
import { useUserStore } from "@/stores/user-store";
import type { APIToken, CreateUserInvitationResponse, UserInvitation, UserResponse, UserSession } from "@/types/api";

const roleBadgeVariant: Record<string, "default" | "secondary" | "outline"> = {
  admin: "default",
  operator: "secondary",
  viewer: "outline",
};

function formatDate(dateStr?: string | null) {
  if (!dateStr) return null;
  return new Date(dateStr).toLocaleString("de-DE", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function formatRelativeTime(dateStr?: string | null): string | null {
  if (!dateStr) return null;
  const date = new Date(dateStr);
  const diffMs = Date.now() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return "Gerade eben";
  if (diffMins < 60) return `vor ${diffMins} Min.`;
  if (diffHours < 24) return `vor ${diffHours} Std.`;
  if (diffDays < 7) return `vor ${diffDays} Tagen`;
  return null;
}

export default function UsersSettingsPage() {
  const { users, isLoading, fetchUsers } = useUserStore();
  const [createOpen, setCreateOpen] = useState(false);
  const [inviteOpen, setInviteOpen] = useState(false);
  const [editUser, setEditUser] = useState<UserResponse | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<UserResponse | null>(null);
  const [deleteUser, setDeleteUser] = useState<UserResponse | null>(null);
  const [passwordUser, setPasswordUser] = useState<UserResponse | null>(null);
  const [invitations, setInvitations] = useState<UserInvitation[]>([]);
  const [accessUser, setAccessUser] = useState<UserResponse | null>(null);
  const [sessions, setSessions] = useState<UserSession[]>([]);
  const [apiTokens, setApiTokens] = useState<APIToken[]>([]);
  const [accessLoading, setAccessLoading] = useState(false);

  const fetchInvitations = async () => {
    try {
      const res = await userApi.listInvitations();
      setInvitations(toArray<UserInvitation>(res.data));
    } catch {
      setInvitations([]);
    }
  };

  useEffect(() => {
    fetchUsers();
    fetchInvitations();
  }, [fetchUsers]);

  const setUserActive = async (user: UserResponse, active: boolean) => {
    try {
      await userApi.update(user.id, { is_active: active });
      toast.success(active ? "Benutzer aktiviert" : "Benutzer deaktiviert und Zugriffe widerrufen");
      fetchUsers();
      if (accessUser?.id === user.id) fetchAccessData(user);
    } catch {
      toast.error("Benutzerstatus konnte nicht geaendert werden");
    }
  };

  const fetchAccessData = async (user: UserResponse) => {
    setAccessUser(user);
    setAccessLoading(true);
    try {
      const [sessionRes, tokenRes] = await Promise.all([
        userApi.listSessions(user.id),
        userApi.listApiTokens(user.id),
      ]);
      setSessions(toArray<UserSession>(sessionRes.data));
      setApiTokens(toArray<APIToken>(tokenRes.data));
    } catch {
      setSessions([]);
      setApiTokens([]);
      toast.error("Zugriffe konnten nicht geladen werden");
    } finally {
      setAccessLoading(false);
    }
  };

  const revokeSession = async (sessionId: string) => {
    if (!accessUser) return;
    try {
      await userApi.revokeSession(accessUser.id, sessionId);
      toast.success("Session widerrufen");
      fetchAccessData(accessUser);
    } catch {
      toast.error("Session konnte nicht widerrufen werden");
    }
  };

  const revokeAllAccess = async () => {
    if (!accessUser) return;
    try {
      await userApi.revokeAllAccess(accessUser.id);
      toast.success("Alle Sessions und API-Tokens widerrufen");
      fetchAccessData(accessUser);
    } catch {
      toast.error("Zugriffe konnten nicht widerrufen werden");
    }
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h2 className="text-lg font-semibold">Benutzerverwaltung</h2>
          <p className="text-sm text-muted-foreground">
            Benutzer, Einladungen, Sessions und Zugriffstokens verwalten.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button variant="outline" onClick={() => setInviteOpen(true)}>
            <UserPlus className="mr-2 h-4 w-4" />
            Einladung erstellen
          </Button>
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Benutzer erstellen
          </Button>
        </div>
      </div>

      <Card>
        <CardContent className="p-0">
          <div className="overflow-x-auto">
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
                    <TableCell colSpan={7} className="py-8 text-center text-muted-foreground">
                      Laden...
                    </TableCell>
                  </TableRow>
                ) : users.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7} className="py-8 text-center text-muted-foreground">
                      Keine Benutzer vorhanden.
                    </TableCell>
                  </TableRow>
                ) : (
                  users.map((user) => (
                    <TableRow key={user.id}>
                      <TableCell className="font-medium">{user.username}</TableCell>
                      <TableCell className="text-muted-foreground">{user.email || "-"}</TableCell>
                      <TableCell>
                        <Badge variant={roleBadgeVariant[user.role] || "outline"}>{user.role}</Badge>
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
                      <TableCell className="text-muted-foreground">{formatDate(user.created_at) || "-"}</TableCell>
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
                            <DropdownMenuItem onClick={() => fetchAccessData(user)}>
                              <RefreshCw className="mr-2 h-4 w-4" />
                              Sessions & Tokens
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => setUserActive(user, !user.is_active)}>
                              {user.is_active ? (
                                <ShieldOff className="mr-2 h-4 w-4" />
                              ) : (
                                <Check className="mr-2 h-4 w-4" />
                              )}
                              {user.is_active ? "Deaktivieren" : "Aktivieren"}
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => setDeleteTarget(user)} className="text-destructive">
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
          </div>
        </CardContent>
      </Card>

      <InvitationTable invitations={invitations} onRefresh={fetchInvitations} />

      <CreateUserDialog open={createOpen} onOpenChange={setCreateOpen} onSuccess={fetchUsers} />
      <EditUserDialog user={editUser} open={!!editUser} onOpenChange={(open) => !open && setEditUser(null)} onSuccess={fetchUsers} />
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
      <DeleteUserDialog user={deleteUser} open={!!deleteUser} onOpenChange={(open) => !open && setDeleteUser(null)} onSuccess={fetchUsers} />
      <ChangePasswordDialog user={passwordUser} open={!!passwordUser} onOpenChange={(open) => !open && setPasswordUser(null)} onSuccess={fetchUsers} />
      <CreateInvitationDialog open={inviteOpen} onOpenChange={setInviteOpen} onSuccess={fetchInvitations} />
      <AccessOverviewDialog
        user={accessUser}
        sessions={sessions}
        apiTokens={apiTokens}
        isLoading={accessLoading}
        onOpenChange={(open) => !open && setAccessUser(null)}
        onRefresh={() => accessUser && fetchAccessData(accessUser)}
        onRevokeSession={revokeSession}
        onRevokeAll={revokeAllAccess}
      />
    </div>
  );
}

function InvitationTable({ invitations, onRefresh }: { invitations: UserInvitation[]; onRefresh: () => void }) {
  const revokeInvitation = async (id: string) => {
    try {
      await userApi.deleteInvitation(id);
      toast.success("Einladung widerrufen");
      onRefresh();
    } catch {
      toast.error("Einladung konnte nicht widerrufen werden");
    }
  };

  return (
    <Card>
      <CardContent className="p-0">
        <div className="flex items-center justify-between p-4">
          <div>
            <h3 className="font-medium">Einladungen</h3>
            <p className="text-sm text-muted-foreground">Offene, abgelaufene und angenommene Einladungen.</p>
          </div>
          <Button variant="outline" size="sm" onClick={onRefresh}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Aktualisieren
          </Button>
        </div>
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Benutzer</TableHead>
                <TableHead>Rolle</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Ablauf</TableHead>
                <TableHead className="w-12"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {invitations.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="py-8 text-center text-muted-foreground">
                    Keine Einladungen vorhanden.
                  </TableCell>
                </TableRow>
              ) : (
                invitations.map((invitation) => {
                  const expired = new Date(invitation.expires_at).getTime() < Date.now();
                  return (
                    <TableRow key={invitation.id}>
                      <TableCell>
                        <div className="font-medium">{invitation.username}</div>
                        <div className="text-sm text-muted-foreground">{invitation.email || "-"}</div>
                      </TableCell>
                      <TableCell>
                        <Badge variant={roleBadgeVariant[invitation.role] || "outline"}>{invitation.role}</Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant={invitation.accepted_at ? "default" : expired ? "secondary" : "outline"}>
                          {invitation.accepted_at ? "Angenommen" : expired ? "Abgelaufen" : "Offen"}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">{formatDate(invitation.expires_at)}</TableCell>
                      <TableCell>
                        {!invitation.accepted_at && (
                          <Button variant="ghost" size="icon" onClick={() => revokeInvitation(invitation.id)}>
                            <Ban className="h-4 w-4" />
                          </Button>
                        )}
                      </TableCell>
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  );
}

function CreateInvitationDialog({ open, onOpenChange, onSuccess }: { open: boolean; onOpenChange: (open: boolean) => void; onSuccess: () => void }) {
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [role, setRole] = useState("viewer");
  const [token, setToken] = useState("");
  const [copied, setCopied] = useState(false);
  const [saving, setSaving] = useState(false);

  const close = () => {
    setUsername("");
    setEmail("");
    setRole("viewer");
    setToken("");
    setCopied(false);
    onOpenChange(false);
  };

  const submit = async () => {
    if (!username.trim()) return;
    setSaving(true);
    try {
      const res = await userApi.createInvitation({ username, email, role, expires_in_hours: 168 });
      const data = res.data as CreateUserInvitationResponse;
      setToken(data.token);
      onSuccess();
      toast.success("Einladung erstellt");
    } catch {
      toast.error("Einladung konnte nicht erstellt werden");
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(next) => (next ? onOpenChange(true) : close())}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{token ? "Einladung erstellt" : "Benutzer einladen"}</DialogTitle>
        </DialogHeader>
        {token ? (
          <div className="flex flex-col gap-4">
            <p className="text-sm text-muted-foreground">Der Token wird nur einmal angezeigt.</p>
            <div className="flex items-center gap-2">
              <code className="flex-1 break-all rounded-md bg-muted p-3 text-sm">{token}</code>
              <Button
                variant="outline"
                size="icon"
                onClick={() => {
                  navigator.clipboard.writeText(token);
                  setCopied(true);
                  toast.success("Token kopiert");
                }}
              >
                {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
              </Button>
            </div>
            <DialogFooter>
              <Button onClick={close}>Schliessen</Button>
            </DialogFooter>
          </div>
        ) : (
          <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <Label htmlFor="invite-username">Benutzername</Label>
              <Input id="invite-username" value={username} onChange={(e) => setUsername(e.target.value)} />
            </div>
            <div className="flex flex-col gap-2">
              <Label htmlFor="invite-email">E-Mail</Label>
              <Input id="invite-email" value={email} onChange={(e) => setEmail(e.target.value)} />
            </div>
            <div className="flex flex-col gap-2">
              <Label htmlFor="invite-role">Rolle</Label>
              <select id="invite-role" value={role} onChange={(e) => setRole(e.target.value)} className="h-10 rounded-md border bg-background px-3 text-sm">
                <option value="viewer">Viewer</option>
                <option value="operator">Operator</option>
                <option value="admin">Admin</option>
              </select>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={close}>Abbrechen</Button>
              <Button onClick={submit} disabled={saving || !username.trim()}>
                {saving ? "Erstelle..." : "Einladung erstellen"}
              </Button>
            </DialogFooter>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

function AccessOverviewDialog({
  user,
  sessions,
  apiTokens,
  isLoading,
  onOpenChange,
  onRefresh,
  onRevokeSession,
  onRevokeAll,
}: {
  user: UserResponse | null;
  sessions: UserSession[];
  apiTokens: APIToken[];
  isLoading: boolean;
  onOpenChange: (open: boolean) => void;
  onRefresh: () => void;
  onRevokeSession: (sessionId: string) => void;
  onRevokeAll: () => void;
}) {
  return (
    <Dialog open={!!user} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl">
        <DialogHeader>
          <DialogTitle>Sessions & Tokens{user ? `: ${user.username}` : ""}</DialogTitle>
        </DialogHeader>
        <div className="flex justify-end gap-2">
          <Button variant="outline" size="sm" onClick={onRefresh} disabled={isLoading}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Aktualisieren
          </Button>
          <Button variant="destructive" size="sm" onClick={onRevokeAll}>
            <WifiOff className="mr-2 h-4 w-4" />
            Alle widerrufen
          </Button>
        </div>
        <div className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardContent className="p-4">
              <h3 className="font-medium">Sessions</h3>
              <div className="mt-3 flex flex-col gap-2">
                {sessions.length === 0 ? (
                  <p className="text-sm text-muted-foreground">Keine Sessions vorhanden.</p>
                ) : (
                  sessions.map((session) => (
                    <div key={session.id} className="flex items-center justify-between gap-3 rounded-md border p-3">
                      <div>
                        <Badge variant={session.is_active ? "default" : "secondary"}>
                          {session.is_active ? "Aktiv" : session.revoked ? "Widerrufen" : "Abgelaufen"}
                        </Badge>
                        <p className="mt-1 text-xs text-muted-foreground">
                          Erstellt: {new Date(session.created_at).toLocaleString("de-DE")}
                        </p>
                      </div>
                      {session.is_active && (
                        <Button variant="ghost" size="icon" onClick={() => onRevokeSession(session.id)}>
                          <Ban className="h-4 w-4" />
                        </Button>
                      )}
                    </div>
                  ))
                )}
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <h3 className="font-medium">API-Tokens</h3>
              <div className="mt-3 flex flex-col gap-2">
                {apiTokens.length === 0 ? (
                  <p className="text-sm text-muted-foreground">Keine API-Tokens vorhanden.</p>
                ) : (
                  apiTokens.map((token) => (
                    <div key={token.id} className="rounded-md border p-3">
                      <div className="flex items-center justify-between gap-2">
                        <span className="font-medium">{token.name}</span>
                        <Badge variant={token.is_active ? "default" : "secondary"}>
                          {token.is_active ? "Aktiv" : "Widerrufen"}
                        </Badge>
                      </div>
                      <p className="mt-1 text-xs text-muted-foreground">
                        {token.token_prefix}... - zuletzt {token.last_used_at ? new Date(token.last_used_at).toLocaleString("de-DE") : "nie"}
                      </p>
                    </div>
                  ))
                )}
              </div>
            </CardContent>
          </Card>
        </div>
      </DialogContent>
    </Dialog>
  );
}
