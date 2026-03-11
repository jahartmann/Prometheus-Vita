"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import {
  Plus,
  Pencil,
  Trash2,
  MoreHorizontal,
  Users,
  Tag,
  Monitor,
  X,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { useAuthStore } from "@/stores/auth-store";
import { useVMGroupStore } from "@/stores/vm-group-store";
import { useNodeStore } from "@/stores/node-store";
import { VMGroupDialog } from "@/components/settings/vm-group-dialog";
import { nodeApi, toArray } from "@/lib/api";
import type { VMGroup, VM, VMGroupMember } from "@/types/api";

export default function VMGroupsPage() {
  const { user } = useAuthStore();
  const router = useRouter();
  const {
    groups,
    members,
    isLoading,
    fetchGroups,
    fetchMembers,
    deleteGroup,
    addMember,
    removeMember,
  } = useVMGroupStore();
  const { nodes, fetchNodes } = useNodeStore();

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editGroup, setEditGroup] = useState<VMGroup | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<VMGroup | null>(null);
  const [expandedGroup, setExpandedGroup] = useState<string | null>(null);
  const [vms, setVMs] = useState<Record<string, VM[]>>({});

  // Add member state
  const [addNodeId, setAddNodeId] = useState("");
  const [addVmid, setAddVmid] = useState("");

  useEffect(() => {
    if (user && user.role !== "admin") {
      router.push("/settings/nodes");
    }
  }, [user, router]);

  useEffect(() => {
    fetchGroups();
    fetchNodes();
  }, [fetchGroups, fetchNodes]);

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

  const toggleExpand = (groupId: string) => {
    if (expandedGroup === groupId) {
      setExpandedGroup(null);
    } else {
      setExpandedGroup(groupId);
      fetchMembers(groupId);
    }
  };

  const handleAddMember = async (groupId: string) => {
    if (!addNodeId || !addVmid) return;
    await addMember(groupId, addNodeId, parseInt(addVmid, 10));
    setAddNodeId("");
    setAddVmid("");
  };

  const formatDate = (dateStr?: string | null) => {
    if (!dateStr) return "-";
    return new Date(dateStr).toLocaleString("de-DE", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  const getVMName = (nodeId: string, vmid: number): string => {
    const nodeVMs = vms[nodeId] || [];
    const vm = nodeVMs.find((v) => v.vmid === vmid);
    return vm ? `${vm.name} (${vmid})` : String(vmid);
  };

  const getNodeName = (nodeId: string): string => {
    const node = nodes.find((n) => n.id === nodeId);
    return node?.name || nodeId.slice(0, 8);
  };

  if (!user || user.role !== "admin") {
    return null;
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">VM-Gruppen</h2>
          <p className="text-sm text-muted-foreground">
            VMs in Gruppen organisieren und Berechtigungen gruppenbasiert vergeben.
          </p>
        </div>
        <Button onClick={() => { setEditGroup(null); setDialogOpen(true); }}>
          <Plus className="mr-2 h-4 w-4" />
          Gruppe erstellen
        </Button>
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Beschreibung</TableHead>
                <TableHead>Tag-Filter</TableHead>
                <TableHead>Mitglieder</TableHead>
                <TableHead>Erstellt</TableHead>
                <TableHead className="w-12"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading && groups.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                    Laden...
                  </TableCell>
                </TableRow>
              ) : groups.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                    Keine VM-Gruppen vorhanden.
                  </TableCell>
                </TableRow>
              ) : (
                groups.map((group) => (
                  <>
                    <TableRow
                      key={group.id}
                      className="cursor-pointer hover:bg-accent/50"
                      onClick={() => toggleExpand(group.id)}
                    >
                      <TableCell className="font-medium">
                        <div className="flex items-center gap-2">
                          <Users className="h-4 w-4 text-muted-foreground" />
                          {group.name}
                        </div>
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {group.description || "-"}
                      </TableCell>
                      <TableCell>
                        {group.tag_filter ? (
                          <Badge variant="outline" className="text-xs">
                            <Tag className="h-3 w-3 mr-1" />
                            {group.tag_filter}
                          </Badge>
                        ) : (
                          <span className="text-muted-foreground">-</span>
                        )}
                      </TableCell>
                      <TableCell>
                        <Badge variant="secondary">{group.member_count || 0}</Badge>
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {formatDate(group.created_at)}
                      </TableCell>
                      <TableCell>
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild onClick={(e) => e.stopPropagation()}>
                            <Button variant="ghost" size="icon">
                              <MoreHorizontal className="h-4 w-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuItem
                              onClick={(e) => {
                                e.stopPropagation();
                                setEditGroup(group);
                                setDialogOpen(true);
                              }}
                            >
                              <Pencil className="mr-2 h-4 w-4" />
                              Bearbeiten
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              onClick={(e) => {
                                e.stopPropagation();
                                setDeleteTarget(group);
                              }}
                              className="text-destructive"
                            >
                              <Trash2 className="mr-2 h-4 w-4" />
                              Loeschen
                            </DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </TableCell>
                    </TableRow>

                    {/* Expanded member list */}
                    {expandedGroup === group.id && (
                      <TableRow key={`${group.id}-members`}>
                        <TableCell colSpan={6} className="bg-muted/30 p-4">
                          <div className="space-y-3">
                            <h4 className="text-sm font-medium">
                              Mitglieder ({(members[group.id] || []).length})
                            </h4>

                            {/* Member list */}
                            {(members[group.id] || []).length > 0 ? (
                              <div className="space-y-1">
                                {(members[group.id] || []).map((member: VMGroupMember) => (
                                  <div
                                    key={`${member.node_id}-${member.vmid}`}
                                    className="flex items-center justify-between py-1 px-2 rounded hover:bg-muted"
                                  >
                                    <div className="flex items-center gap-2 text-sm">
                                      <Monitor className="h-3.5 w-3.5 text-muted-foreground" />
                                      <span>{getVMName(member.node_id, member.vmid)}</span>
                                      <Badge variant="outline" className="text-xs">
                                        {getNodeName(member.node_id)}
                                      </Badge>
                                    </div>
                                    <Button
                                      variant="ghost"
                                      size="icon"
                                      className="h-6 w-6"
                                      onClick={() =>
                                        removeMember(group.id, member.node_id, member.vmid)
                                      }
                                    >
                                      <X className="h-3 w-3" />
                                    </Button>
                                  </div>
                                ))}
                              </div>
                            ) : (
                              <p className="text-sm text-muted-foreground">
                                Keine Mitglieder in dieser Gruppe.
                              </p>
                            )}

                            {/* Add member form */}
                            <div className="flex items-center gap-2 pt-2 border-t">
                              <Select value={addNodeId} onValueChange={(v) => { setAddNodeId(v); setAddVmid(""); }}>
                                <SelectTrigger className="w-40 h-8 text-xs">
                                  <SelectValue placeholder="Node waehlen" />
                                </SelectTrigger>
                                <SelectContent>
                                  {nodes.map((node) => (
                                    <SelectItem key={node.id} value={node.id}>
                                      {node.name}
                                    </SelectItem>
                                  ))}
                                </SelectContent>
                              </Select>

                              {addNodeId && (
                                <Select value={addVmid} onValueChange={setAddVmid}>
                                  <SelectTrigger className="w-48 h-8 text-xs">
                                    <SelectValue placeholder="VM waehlen" />
                                  </SelectTrigger>
                                  <SelectContent>
                                    {(vms[addNodeId] || []).map((vm) => (
                                      <SelectItem key={vm.vmid} value={String(vm.vmid)}>
                                        {vm.name || vm.vmid} ({vm.vmid})
                                      </SelectItem>
                                    ))}
                                  </SelectContent>
                                </Select>
                              )}

                              <Button
                                size="sm"
                                variant="outline"
                                className="h-8 text-xs"
                                disabled={!addNodeId || !addVmid}
                                onClick={() => handleAddMember(group.id)}
                              >
                                <Plus className="h-3 w-3 mr-1" />
                                Hinzufuegen
                              </Button>
                            </div>
                          </div>
                        </TableCell>
                      </TableRow>
                    )}
                  </>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <VMGroupDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        group={editGroup}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title="VM-Gruppe loeschen?"
        description={`Die Gruppe "${deleteTarget?.name}" wird unwiderruflich geloescht. Alle Mitgliedschaften und zugehoerige Berechtigungen werden entfernt.`}
        confirmLabel="Loeschen"
        variant="destructive"
        onConfirm={() => {
          if (deleteTarget) {
            deleteGroup(deleteTarget.id);
            setDeleteTarget(null);
          }
        }}
      />
    </div>
  );
}
