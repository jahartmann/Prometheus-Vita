import api from "@/lib/api";
import type { VMProcess, VMServiceInfo, VMPort, VMDisk, VMExecResult, VMPermission, VMFileEntry } from "@/types/api";
import { AxiosError } from "axios";

export interface VMCockpitError {
  errorCode: string;
  message: string;
  details?: string;
  hint?: string;
}

export function extractCockpitError(error: unknown): VMCockpitError {
  if (error instanceof AxiosError && error.response?.data) {
    const data = error.response.data;
    if (data.code) {
      return {
        errorCode: data.code,
        message: data.error || "Unbekannter Fehler",
        details: data.details,
        hint: data.hint,
      };
    }
    if (data.error) {
      return {
        errorCode: "UNKNOWN",
        message: data.error,
      };
    }
  }
  return {
    errorCode: "NETWORK_ERROR",
    message: "Verbindung zum Server fehlgeschlagen",
    details: "Der Server ist nicht erreichbar oder die Anfrage wurde abgebrochen.",
    hint: "Prüfen Sie Ihre Netzwerkverbindung und versuchen Sie es erneut.",
  };
}

export const vmCockpitApi = {
  exec: (nodeId: string, vmid: number, command: string[], type = "lxc") =>
    api.post<{ data: VMExecResult }>(`/nodes/${nodeId}/vms/${vmid}/cockpit/exec?type=${type}`, { command }),

  getProcesses: (nodeId: string, vmid: number, type = "lxc") =>
    api.get<{ data: VMProcess[] }>(`/nodes/${nodeId}/vms/${vmid}/cockpit/processes?type=${type}`),

  getServices: (nodeId: string, vmid: number, type = "lxc") =>
    api.get<{ data: VMServiceInfo[] }>(`/nodes/${nodeId}/vms/${vmid}/cockpit/services?type=${type}`),

  getPorts: (nodeId: string, vmid: number, type = "lxc") =>
    api.get<{ data: VMPort[] }>(`/nodes/${nodeId}/vms/${vmid}/cockpit/ports?type=${type}`),

  getDisk: (nodeId: string, vmid: number, type = "lxc") =>
    api.get<{ data: VMDisk[] }>(`/nodes/${nodeId}/vms/${vmid}/cockpit/disk?type=${type}`),

  serviceAction: (nodeId: string, vmid: number, service: string, action: string, type = "lxc") =>
    api.post(`/nodes/${nodeId}/vms/${vmid}/cockpit/services/action?type=${type}`, { service, action }),

  killProcess: (nodeId: string, vmid: number, pid: number, signal = "TERM", type = "lxc") =>
    api.post(`/nodes/${nodeId}/vms/${vmid}/cockpit/processes/kill?type=${type}`, { pid, signal }),

  // File operations
  listDir: (nodeId: string, vmid: number, path: string, type = "lxc") =>
    api.get<{ data: VMFileEntry[] }>(`/nodes/${nodeId}/vms/${vmid}/cockpit/files?type=${type}&path=${encodeURIComponent(path)}`),

  readFile: (nodeId: string, vmid: number, path: string, type = "lxc") =>
    api.get<{ data: { content: string } }>(`/nodes/${nodeId}/vms/${vmid}/cockpit/files/read?type=${type}&path=${encodeURIComponent(path)}`),

  writeFile: (nodeId: string, vmid: number, path: string, content: string, type = "lxc") =>
    api.post(`/nodes/${nodeId}/vms/${vmid}/cockpit/files/write?type=${type}`, { path, content }),

  uploadFile: (nodeId: string, vmid: number, path: string, content: string, type = "lxc") =>
    api.post(`/nodes/${nodeId}/vms/${vmid}/cockpit/files/upload?type=${type}`, { path, content }),

  deleteFile: (nodeId: string, vmid: number, path: string, type = "lxc") =>
    api.delete(`/nodes/${nodeId}/vms/${vmid}/cockpit/files?type=${type}&path=${encodeURIComponent(path)}`),

  mkdir: (nodeId: string, vmid: number, path: string, type = "lxc") =>
    api.post(`/nodes/${nodeId}/vms/${vmid}/cockpit/files/mkdir?type=${type}`, { path }),
};

export const vmPermissionApi = {
  list: (userId: string) => api.get<{ data: VMPermission[] }>(`/vm-permissions?user_id=${userId}`),
  create: (perm: Partial<VMPermission>) => api.post("/vm-permissions", perm),
  update: (id: string, perm: Partial<VMPermission>) => api.put(`/vm-permissions/${id}`, perm),
  delete: (id: string) => api.delete(`/vm-permissions/${id}`),
};
