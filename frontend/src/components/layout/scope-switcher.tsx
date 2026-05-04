"use client";

import { useEffect } from "react";
import { usePathname, useRouter } from "next/navigation";
import { Globe2, Server } from "lucide-react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useAuthStore } from "@/stores/auth-store";
import { useNodeStore } from "@/stores/node-store";

function currentNodeId(pathname: string): string | null {
  const match = pathname.match(/^\/nodes\/([^/]+)/);
  return match?.[1] ?? null;
}

function globalTargetForNodePath(pathname: string): string {
  const parts = pathname.split("/").filter(Boolean);
  const nodeTab = parts[2] ?? "";

  switch (nodeTab) {
    case "monitoring":
      return "/monitoring";
    case "network":
    case "ports":
      return "/network";
    case "storage":
      return "/storage";
    case "backups":
      return "/backups";
    case "migrations":
      return "/migrations";
    case "dr":
      return "/disaster-recovery";
    case "iso-templates":
      return "/isos";
    case "logs":
      return "/logs";
    case "vms":
      return "/nodes";
    default:
      return "/nodes";
  }
}

function nodeTargetForGlobalPath(pathname: string, nodeId: string): string {
  if (pathname.startsWith("/monitoring")) return `/nodes/${nodeId}/monitoring`;
  if (pathname.startsWith("/network")) return `/nodes/${nodeId}/network`;
  if (pathname.startsWith("/storage")) return `/nodes/${nodeId}/storage`;
  if (pathname.startsWith("/backups")) return `/nodes/${nodeId}/backups`;
  if (pathname.startsWith("/migrations")) return `/nodes/${nodeId}/migrations`;
  if (pathname.startsWith("/disaster-recovery")) return `/nodes/${nodeId}/dr`;
  if (pathname.startsWith("/isos")) return `/nodes/${nodeId}/iso-templates`;
  if (pathname.startsWith("/logs")) return `/nodes/${nodeId}/logs`;
  return `/nodes/${nodeId}`;
}

export function ScopeSwitcher() {
  const pathname = usePathname();
  const router = useRouter();
  const { nodes, fetchNodes } = useNodeStore();
  const nodeId = currentNodeId(pathname);

  useEffect(() => {
    if (useAuthStore.getState().accessToken) {
      fetchNodes();
    }
  }, [fetchNodes]);

  const selectedValue = nodeId ? `node:${nodeId}` : "global";
  const selectedNode = nodeId ? nodes.find((node) => node.id === nodeId) : undefined;

  const handleChange = (value: string) => {
    if (value === "global") {
      router.push(nodeId ? globalTargetForNodePath(pathname) : "/");
      return;
    }

    const nextNodeId = value.replace("node:", "");
    router.push(nodeTargetForGlobalPath(pathname, nextNodeId));
  };

  return (
    <div className="hidden items-center gap-2 md:flex">
      <span className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
        Scope
      </span>
      <Select value={selectedValue} onValueChange={handleChange}>
        <SelectTrigger className="h-8 w-[180px] border-border/80 bg-muted/35 px-2.5 text-xs">
          <SelectValue
            placeholder={selectedNode ? selectedNode.name : "Global"}
          />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="global">
            <span className="flex items-center gap-2">
              <Globe2 className="h-3.5 w-3.5" />
              Global
            </span>
          </SelectItem>
          {nodeId && !selectedNode && (
            <SelectItem value={`node:${nodeId}`}>
              <span className="flex items-center gap-2">
                <Server className="h-3.5 w-3.5" />
                {nodeId}
              </span>
            </SelectItem>
          )}
          {nodes.map((node) => (
            <SelectItem key={node.id} value={`node:${node.id}`}>
              <span className="flex items-center gap-2">
                <Server className="h-3.5 w-3.5" />
                {node.name}
              </span>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
