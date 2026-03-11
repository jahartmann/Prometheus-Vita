"use client";

import { useEffect, useState, useMemo, useCallback } from "react";
import { Save, Filter, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { toast } from "sonner";
import { useVMPermissionStore } from "@/stores/vm-permission-store";
import { useUserStore } from "@/stores/user-store";
import { useNodeStore } from "@/stores/node-store";
import { vmPermissionApi, nodeApi, toArray } from "@/lib/api";
import type { VMPermission, VM, VMGroup } from "@/types/api";
import { vmGroupApi } from "@/lib/api";

// Permission categories for grouping
const permissionCategories: Record<string, string[]> = {
  "Anzeige": ["vm.view"],
  "Shell": ["vm.shell"],
  "Dateien": ["vm.files.read", "vm.files.write"],
  "System": ["vm.system.view", "vm.system.service", "vm.system.kill", "vm.system.packages"],
  "Power": ["vm.power", "vm.snapshots"],
  "KI": ["vm.ai.proactive"],
};

const permShortLabels: Record<string, string> = {
  "vm.view": "Ansicht",
  "vm.shell": "Shell",
  "vm.files.read": "Lesen",
  "vm.files.write": "Schreiben",
  "vm.system.view": "Ansicht",
  "vm.system.service": "Services",
  "vm.system.kill": "Kill",
  "vm.system.packages": "Pakete",
  "vm.power": "Power",
  "vm.snapshots": "Snapshots",
  "vm.ai.proactive": "Proaktiv",
};

interface PermissionChange {
  userId: string;
  targetType: "vm" | "group";
  targetId: string;
  nodeId: string;
  permissions: string[];
}

type TargetEntry = {
  id: string;
  label: string;
  type: "vm" | "group";
  nodeId: string;
  vmid?: number;
};

export function VMPermissionMatrix() {
  const { permissions, fetchPermissions } = useVMPermissionStore();
  const { users, fetchUsers } = useUserStore();
  const { nodes, fetchNodes } = useNodeStore();

  const [vms, setVMs] = useState<Record<string, VM[]>>({});
  const [groups, setGroups] = useState<VMGroup[]>([]);
  const [changes, setChanges] = useState<Map<string, PermissionChange>>(new Map());
  const [filterUser, setFilterUser] = useState("");
  const [filterNode, setFilterNode] = useState("all");
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    fetchPermissions();
    fetchUsers();
    fetchNodes();
    vmGroupApi.list().then((r) => setGroups(toArray<VMGroup>(r.data))).catch(() => {});
  }, [fetchPermissions, fetchUsers, fetchNodes]);

  useEffect(() => {
    const loadVMs = async () => {
      const vmData: Record<string, VM[]> = {};
      for (const node of nodes) {
        try {
          const res = await nodeApi.getVMs(node.id);
          vmData[node.id] = toArray<VM>(res.data);
        } catch {
          vmData[node.id] = [];
        }
      }
      setVMs(vmData);
    };
    if (nodes.length > 0) loadVMs();
  }, [nodes]);

  // Build target entries (VMs + Groups)
  const targets: TargetEntry[] = useMemo(() => {
    const entries: TargetEntry[] = [];

    // Add VMs
    for (const node of nodes) {
      if (filterNode !== "all" && node.id !== filterNode) continue;
      const nodeVMs = vms[node.id] || [];
      for (const vm of nodeVMs) {
        entries.push({
          id: `${node.id}:${vm.vmid}`,
          label: `${vm.name || vm.vmid} (${node.name})`,
          type: "vm",
          nodeId: node.id,
          vmid: vm.vmid,
        });
      }
    }

    // Add Groups
    for (const group of groups) {
      entries.push({
        id: `group:${group.id}`,
        label: `[Gruppe] ${group.name}`,
        type: "group",
        nodeId: nodes[0]?.id || "", // Groups aren't node-scoped in display, but permissions are
      });
    }

    return entries;
  }, [nodes, vms, groups, filterNode]);

  // Filtered users
  const filteredUsers = useMemo(() => {
    if (!filterUser) return users.filter((u) => u.role !== "admin");
    const lower = filterUser.toLowerCase();
    return users.filter(
      (u) => u.role !== "admin" && (u.username.toLowerCase().includes(lower) || u.email?.toLowerCase().includes(lower))
    );
  }, [users, filterUser]);

  // Build permission lookup
  const permLookup = useMemo(() => {
    const map = new Map<string, VMPermission>();
    for (const p of permissions) {
      const key = `${p.user_id}:${p.target_type}:${p.target_id}:${p.node_id}`;
      map.set(key, p);
    }
    return map;
  }, [permissions]);

  const getPermKey = (userId: string, target: TargetEntry): string => {
    if (target.type === "group") {
      const groupId = target.id.replace("group:", "");
      // For group permissions, we need a node_id; use the first node
      return `${userId}:group:${groupId}:${nodes[0]?.id || ""}`;
    }
    return `${userId}:vm:${String(target.vmid)}:${target.nodeId}`;
  };

  const getCurrentPerms = useCallback((userId: string, target: TargetEntry): string[] => {
    const changeKey = `${userId}:${target.id}`;
    if (changes.has(changeKey)) {
      return changes.get(changeKey)!.permissions;
    }
    const key = getPermKey(userId, target);
    return permLookup.get(key)?.permissions || [];
  }, [changes, permLookup, nodes]);

  const isInherited = useCallback((userId: string, target: TargetEntry, perm: string): boolean => {
    if (target.type !== "vm") return false;
    // Check if the user has this permission via any group
    for (const group of groups) {
      const groupKey = `${userId}:group:${group.id}:${nodes[0]?.id || ""}`;
      const groupPerms = permLookup.get(groupKey)?.permissions || [];
      if (groupPerms.includes(perm)) return true;
    }
    return false;
  }, [groups, permLookup, nodes]);

  const togglePermission = (userId: string, target: TargetEntry, perm: string) => {
    const current = getCurrentPerms(userId, target);
    const changeKey = `${userId}:${target.id}`;
    let newPerms: string[];
    if (current.includes(perm)) {
      newPerms = current.filter((p) => p !== perm);
    } else {
      newPerms = [...current, perm];
    }

    const nodeId = target.type === "group"
      ? nodes[0]?.id || ""
      : target.nodeId;
    const targetId = target.type === "group"
      ? target.id.replace("group:", "")
      : String(target.vmid);

    const newChanges = new Map(changes);
    newChanges.set(changeKey, {
      userId,
      targetType: target.type,
      targetId,
      nodeId,
      permissions: newPerms,
    });
    setChanges(newChanges);
  };

  const saveChanges = async () => {
    setIsSaving(true);
    try {
      const promises = Array.from(changes.values()).map((change) =>
        vmPermissionApi.upsert({
          user_id: change.userId,
          target_type: change.targetType,
          target_id: change.targetId,
          node_id: change.nodeId,
          permissions: change.permissions,
        })
      );
      await Promise.all(promises);
      toast.success(`${changes.size} Berechtigungen gespeichert`);
      setChanges(new Map());
      await fetchPermissions();
    } catch {
      toast.error("Fehler beim Speichern der Berechtigungen");
    } finally {
      setIsSaving(false);
    }
  };

  const allPerms = Object.values(permissionCategories).flat();

  return (
    <div className="space-y-4">
      {/* Filters */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <Filter className="h-4 w-4" />
            Filter
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-3">
          <div className="w-60">
            <Input
              placeholder="Benutzer suchen..."
              value={filterUser}
              onChange={(e) => setFilterUser(e.target.value)}
            />
          </div>
          <div className="w-48">
            <Select value={filterNode} onValueChange={setFilterNode}>
              <SelectTrigger>
                <SelectValue placeholder="Alle Nodes" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Alle Nodes</SelectItem>
                {nodes.map((node) => (
                  <SelectItem key={node.id} value={node.id}>
                    {node.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          {changes.size > 0 && (
            <div className="flex items-center gap-2 ml-auto">
              <Badge variant="secondary">{changes.size} Aenderungen</Badge>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setChanges(new Map())}
              >
                <X className="h-3 w-3 mr-1" />
                Verwerfen
              </Button>
              <Button size="sm" onClick={saveChanges} disabled={isSaving}>
                <Save className="h-3 w-3 mr-1" />
                {isSaving ? "Speichern..." : "Speichern"}
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Matrix */}
      <Card>
        <CardContent className="p-0 overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="sticky left-0 z-10 bg-background min-w-[160px]">
                  Benutzer / Ziel
                </TableHead>
                {Object.entries(permissionCategories).map(([category, perms]) => (
                  <TableHead
                    key={category}
                    colSpan={perms.length}
                    className="text-center border-l text-xs"
                  >
                    {category}
                  </TableHead>
                ))}
              </TableRow>
              <TableRow>
                <TableHead className="sticky left-0 z-10 bg-background" />
                {allPerms.map((perm) => (
                  <TableHead key={perm} className="text-center text-xs px-1 min-w-[60px]">
                    {permShortLabels[perm] || perm.split(".").pop()}
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredUsers.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={allPerms.length + 1}
                    className="text-center py-8 text-muted-foreground"
                  >
                    Keine Benutzer gefunden.
                  </TableCell>
                </TableRow>
              ) : (
                filteredUsers.map((user) => (
                  <>
                    {/* User header row */}
                    <TableRow key={`user-${user.id}`} className="bg-muted/50">
                      <TableCell
                        colSpan={allPerms.length + 1}
                        className="sticky left-0 z-10 font-semibold text-sm py-1.5"
                      >
                        {user.username}
                        <Badge variant="outline" className="ml-2 text-xs">
                          {user.role}
                        </Badge>
                      </TableCell>
                    </TableRow>
                    {/* Target rows */}
                    {targets.map((target) => {
                      const currentPerms = getCurrentPerms(user.id, target);
                      const hasChange = changes.has(`${user.id}:${target.id}`);
                      return (
                        <TableRow
                          key={`${user.id}-${target.id}`}
                          className={hasChange ? "bg-yellow-50/50 dark:bg-yellow-900/10" : ""}
                        >
                          <TableCell className="sticky left-0 z-10 bg-background text-xs truncate max-w-[200px]">
                            {target.type === "group" ? (
                              <Badge variant="outline" className="text-xs">
                                {target.label}
                              </Badge>
                            ) : (
                              target.label
                            )}
                          </TableCell>
                          {allPerms.map((perm) => {
                            const hasDirect = currentPerms.includes(perm);
                            const hasInherited = isInherited(user.id, target, perm);
                            return (
                              <TableCell key={perm} className="text-center px-1">
                                <Checkbox
                                  checked={hasDirect || hasInherited}
                                  onCheckedChange={() => togglePermission(user.id, target, perm)}
                                  disabled={hasInherited && !hasDirect}
                                  className={
                                    hasInherited && !hasDirect
                                      ? "opacity-50 data-[state=checked]:bg-muted-foreground"
                                      : ""
                                  }
                                />
                              </TableCell>
                            );
                          })}
                        </TableRow>
                      );
                    })}
                  </>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Legend */}
      <div className="flex items-center gap-4 text-xs text-muted-foreground">
        <div className="flex items-center gap-1">
          <Checkbox checked disabled className="h-3 w-3" />
          <span>Direkte Berechtigung</span>
        </div>
        <div className="flex items-center gap-1">
          <Checkbox checked disabled className="h-3 w-3 opacity-50" />
          <span>Geerbt von Gruppe</span>
        </div>
        <div className="flex items-center gap-1">
          <div className="w-3 h-3 bg-yellow-50 dark:bg-yellow-900/10 border rounded" />
          <span>Ungespeicherte Aenderung</span>
        </div>
      </div>
    </div>
  );
}
