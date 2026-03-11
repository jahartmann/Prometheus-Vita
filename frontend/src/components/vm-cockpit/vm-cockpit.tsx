"use client";

import { useEffect } from "react";
import { Monitor, Container, FolderOpen, Bot, BarChart3 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ReactFlowProvider } from "@xyflow/react";
import { useVMCockpitStore } from "@/stores/vm-cockpit-store";
import { ShellTab } from "./shell-tab";
import { SystemTab } from "./system-tab";
import { FilesTab } from "./files-tab";
import { AITab } from "./ai-tab";
import { HealthCard } from "./health-card";
import { SnapshotPolicySection } from "./snapshot-policy-dialog";
import { ScheduledActionsSection } from "./scheduled-actions-section";
import { DependencyGraph } from "./dependency-graph";
import type { VM } from "@/types/api";

interface VMCockpitProps {
  vm: VM;
  nodeId: string;
}

export function VMCockpit({ vm, nodeId }: VMCockpitProps) {
  const { setVM } = useVMCockpitStore();

  useEffect(() => {
    setVM(vm, nodeId);
  }, [vm, nodeId, setVM]);

  const statusVariant = (status: string) => {
    switch (status) {
      case "running":
        return "success" as const;
      case "stopped":
        return "secondary" as const;
      case "paused":
        return "warning" as const;
      default:
        return "outline" as const;
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <div className="flex items-center gap-3">
          {vm.type === "qemu" ? (
            <Monitor className="h-6 w-6 text-muted-foreground" />
          ) : (
            <Container className="h-6 w-6 text-muted-foreground" />
          )}
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-xl font-bold">{vm.name}</h2>
              <Badge variant={statusVariant(vm.status)}>{vm.status}</Badge>
              <Badge variant="outline">{vm.type === "qemu" ? "QEMU" : "LXC"}</Badge>
              <span className="text-sm text-muted-foreground">VMID: {vm.vmid}</span>
            </div>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <Tabs defaultValue="shell">
        <TabsList>
          <TabsTrigger value="shell">Shell</TabsTrigger>
          <TabsTrigger value="system">System</TabsTrigger>
          <TabsTrigger value="files">Dateien</TabsTrigger>
          <TabsTrigger value="ai">KI-Assistent</TabsTrigger>
          <TabsTrigger value="monitoring">Monitoring</TabsTrigger>
          <TabsTrigger value="dependencies">Abhaengigkeiten</TabsTrigger>
        </TabsList>

        <TabsContent value="shell" className="mt-4">
          {vm.status === "running" ? (
            <ShellTab nodeId={nodeId} vmid={vm.vmid} vmType={vm.type} />
          ) : (
            <Card>
              <CardContent className="flex flex-col items-center justify-center py-12">
                <p className="text-muted-foreground">
                  Shell ist nur verfuegbar, wenn die VM laeuft.
                </p>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="system" className="mt-4">
          {vm.status === "running" ? (
            <SystemTab />
          ) : (
            <Card>
              <CardContent className="flex flex-col items-center justify-center py-12">
                <p className="text-muted-foreground">
                  Systeminformationen sind nur verfuegbar, wenn die VM laeuft.
                </p>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="files" className="mt-4">
          {vm.status === "running" ? (
            <FilesTab />
          ) : (
            <Card>
              <CardContent className="flex flex-col items-center justify-center py-12">
                <FolderOpen className="mb-4 h-12 w-12 text-muted-foreground" />
                <p className="text-muted-foreground">
                  Dateimanager ist nur verfuegbar, wenn die VM laeuft.
                </p>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="ai" className="mt-4">
          {vm.status === "running" ? (
            <AITab vm={vm} nodeId={nodeId} />
          ) : (
            <Card>
              <CardContent className="flex flex-col items-center justify-center py-12">
                <Bot className="mb-4 h-12 w-12 text-muted-foreground" />
                <p className="text-muted-foreground">
                  KI-Assistent ist nur verfuegbar, wenn die VM laeuft.
                </p>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="monitoring" className="mt-4">
          <div className="space-y-6">
            <HealthCard nodeId={nodeId} vmid={vm.vmid} />
            <SnapshotPolicySection nodeId={nodeId} vmid={vm.vmid} vmType={vm.type} />
            <ScheduledActionsSection nodeId={nodeId} vmid={vm.vmid} vmType={vm.type} />
          </div>
        </TabsContent>

        <TabsContent value="dependencies" className="mt-4">
          <ReactFlowProvider>
            <DependencyGraph nodeId={nodeId} vmid={vm.vmid} />
          </ReactFlowProvider>
        </TabsContent>
      </Tabs>
    </div>
  );
}
