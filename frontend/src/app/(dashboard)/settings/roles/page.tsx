"use client";

import { useEffect, useMemo, useState } from "react";
import { CheckCircle2, Lock, RotateCcw, Save, ShieldAlert, ShieldCheck } from "lucide-react";
import { toast } from "sonner";
import { permissionApi } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

interface PermissionDefinition {
  key: string;
  label: string;
  description: string;
  category: string;
  risk: "low" | "medium" | "high" | string;
}

interface RolePermissionSummary {
  role: "admin" | "operator" | "viewer" | string;
  permissions: string[];
  updated_at?: string;
  updated_by?: string;
}

interface PermissionCatalog {
  permissions: PermissionDefinition[];
  roles: RolePermissionSummary[];
}

const roleLabels: Record<string, string> = {
  admin: "Admin",
  operator: "Operator",
  viewer: "Viewer",
};

const riskLabels: Record<string, string> = {
  low: "Niedrig",
  medium: "Mittel",
  high: "Hoch",
};

const riskVariant: Record<string, "success" | "warning" | "degraded" | "outline"> = {
  low: "success",
  medium: "warning",
  high: "degraded",
};

const editableRoles = new Set(["operator", "viewer"]);

export default function RoleSettingsPage() {
  const [catalog, setCatalog] = useState<PermissionCatalog | null>(null);
  const [draftRoles, setDraftRoles] = useState<Record<string, string[]>>({});
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);

  const loadCatalog = async () => {
    const data = (await permissionApi.getCatalog()) as PermissionCatalog;
    setCatalog(data);
    setDraftRoles(Object.fromEntries(data.roles.map((role) => [role.role, role.permissions])));
  };

  useEffect(() => {
    let mounted = true;
    permissionApi.getCatalog()
      .then((data) => {
        if (!mounted) return;
        const catalogData = data as PermissionCatalog;
        setCatalog(catalogData);
        setDraftRoles(Object.fromEntries(catalogData.roles.map((role) => [role.role, role.permissions])));
      })
      .catch(() => {
        if (mounted) setCatalog(null);
      })
      .finally(() => {
        if (mounted) setIsLoading(false);
      });
    return () => {
      mounted = false;
    };
  }, []);

  const roles = catalog?.roles ?? [];
  const originalRoles = useMemo(
    () => Object.fromEntries(roles.map((role) => [role.role, role.permissions])),
    [roles]
  );

  const groupedPermissions = useMemo(() => {
    const groups = new Map<string, PermissionDefinition[]>();
    for (const permission of catalog?.permissions ?? []) {
      const entries = groups.get(permission.category) ?? [];
      entries.push(permission);
      groups.set(permission.category, entries);
    }
    return Array.from(groups.entries());
  }, [catalog]);

  const roleAllows = (role: string, permission: string) => {
    const permissions = draftRoles[role] ?? [];
    return permissions.includes("*") || permissions.includes(permission);
  };

  const isRoleDirty = (role: string) => {
    const before = [...(originalRoles[role] ?? [])].sort().join("|");
    const after = [...(draftRoles[role] ?? [])].sort().join("|");
    return before !== after;
  };

  const dirtyRoles = roles.filter((role) => editableRoles.has(role.role) && isRoleDirty(role.role));

  const togglePermission = (role: string, permission: string) => {
    if (!editableRoles.has(role)) return;
    setDraftRoles((current) => {
      const permissions = current[role] ?? [];
      const next = permissions.includes(permission)
        ? permissions.filter((entry) => entry !== permission)
        : [...permissions, permission];
      return { ...current, [role]: next };
    });
  };

  const discardChanges = () => {
    setDraftRoles(Object.fromEntries(roles.map((role) => [role.role, role.permissions])));
  };

  const saveChanges = async () => {
    if (dirtyRoles.length === 0) return;
    setIsSaving(true);
    try {
      await Promise.all(
        dirtyRoles.map((role) => permissionApi.updateRole(role.role, draftRoles[role.role] ?? []))
      );
      await loadCatalog();
      toast.success("Rollenrechte gespeichert");
    } catch {
      toast.error("Rollenrechte konnten nicht gespeichert werden");
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="space-y-5">
      <div className="rounded-lg border bg-card p-4">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="flex items-start gap-3">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-muted">
              <ShieldCheck className="h-4 w-4" />
            </div>
            <div>
              <h2 className="text-xl font-semibold tracking-tight">Rollen & Rechte</h2>
              <p className="mt-1 max-w-3xl text-sm text-muted-foreground">
                Globale Rollenrechte fuer Prometheus. Admin bleibt als Sicherheitsrolle gesperrt,
                Operator und Viewer koennen fein abgestimmt werden.
              </p>
            </div>
          </div>
          {catalog && (
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant={dirtyRoles.length > 0 ? "warning" : "outline"}>
                {dirtyRoles.length} geaendert
              </Badge>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={discardChanges}
                disabled={dirtyRoles.length === 0 || isSaving}
              >
                <RotateCcw className="h-4 w-4" />
                Verwerfen
              </Button>
              <Button
                type="button"
                size="sm"
                onClick={saveChanges}
                disabled={dirtyRoles.length === 0 || isSaving}
              >
                <Save className="h-4 w-4" />
                Speichern
              </Button>
            </div>
          )}
        </div>
      </div>

      {isLoading ? (
        <Card>
          <CardContent className="p-6 text-sm text-muted-foreground">Lade Berechtigungen...</CardContent>
        </Card>
      ) : !catalog ? (
        <Card>
          <CardContent className="flex items-start gap-3 p-6">
            <ShieldAlert className="mt-0.5 h-4 w-4 text-orange-500" />
            <p className="text-sm text-muted-foreground">Berechtigungskatalog konnte nicht geladen werden.</p>
          </CardContent>
        </Card>
      ) : (
        groupedPermissions.map(([category, permissions]) => (
          <Card key={category}>
            <CardHeader>
              <CardTitle className="text-base">{category}</CardTitle>
            </CardHeader>
            <CardContent className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="min-w-72">Berechtigung</TableHead>
                    <TableHead>Risiko</TableHead>
                    {roles.map((role) => (
                      <TableHead key={role.role} className="min-w-28 text-center">
                        <div className="flex items-center justify-center gap-2">
                          {roleLabels[role.role] ?? role.role}
                          {!editableRoles.has(role.role) && <Lock className="h-3.5 w-3.5 text-muted-foreground" />}
                        </div>
                      </TableHead>
                    ))}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {permissions.map((permission) => (
                    <TableRow key={permission.key}>
                      <TableCell>
                        <div className="space-y-1">
                          <div className="font-medium">{permission.label}</div>
                          <div className="text-xs text-muted-foreground">{permission.description}</div>
                          <code className="text-xs text-muted-foreground">{permission.key}</code>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant={riskVariant[permission.risk] ?? "outline"}>
                          {riskLabels[permission.risk] ?? permission.risk}
                        </Badge>
                      </TableCell>
                      {roles.map((role) => (
                        <TableCell key={`${role.role}-${permission.key}`} className="text-center">
                          {editableRoles.has(role.role) ? (
                            <Checkbox
                              checked={roleAllows(role.role, permission.key)}
                              disabled={isSaving}
                              aria-label={`${roleLabels[role.role] ?? role.role}: ${permission.label}`}
                              onCheckedChange={() => togglePermission(role.role, permission.key)}
                            />
                          ) : roleAllows(role.role, permission.key) ? (
                            <CheckCircle2 className="mx-auto h-4 w-4 text-green-600" />
                          ) : (
                            <span className="text-muted-foreground">-</span>
                          )}
                        </TableCell>
                      ))}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        ))
      )}
    </div>
  );
}
