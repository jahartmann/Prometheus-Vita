import { create } from "zustand";
import { toast } from "sonner";
import { toArray } from "@/lib/api";
import { vmCockpitApi } from "@/lib/vm-api";
import type { VM, VMProcess, VMServiceInfo, VMPort, VMDisk, VMFileEntry } from "@/types/api";

interface VMCockpitState {
  vm: VM | null;
  nodeId: string | null;
  processes: VMProcess[];
  services: VMServiceInfo[];
  ports: VMPort[];
  disks: VMDisk[];
  isLoadingProcesses: boolean;
  isLoadingServices: boolean;
  isLoadingPorts: boolean;
  isLoadingDisk: boolean;
  // File browser state
  currentPath: string;
  files: VMFileEntry[];
  isLoadingFiles: boolean;
  openFilePath: string | null;
  openFileContent: string | null;
  openFileOriginal: string | null;
  isLoadingFile: boolean;
  isSavingFile: boolean;
  bookmarks: string[];
  setVM: (vm: VM, nodeId: string) => void;
  fetchProcesses: () => Promise<void>;
  fetchServices: () => Promise<void>;
  fetchPorts: () => Promise<void>;
  fetchDisk: () => Promise<void>;
  killProcess: (pid: number, signal?: string) => Promise<void>;
  serviceAction: (service: string, action: string) => Promise<void>;
  // File browser actions
  navigateTo: (path: string) => Promise<void>;
  openFile: (path: string) => Promise<void>;
  closeFile: () => void;
  saveFile: (path: string, content: string) => Promise<void>;
  deleteEntry: (path: string) => Promise<void>;
  createDirectory: (path: string) => Promise<void>;
  createFile: (path: string) => Promise<void>;
  uploadFile: (path: string, content: string) => Promise<void>;
}

export const useVMCockpitStore = create<VMCockpitState>()((set, get) => ({
  vm: null,
  nodeId: null,
  processes: [],
  services: [],
  ports: [],
  disks: [],
  isLoadingProcesses: false,
  isLoadingServices: false,
  isLoadingPorts: false,
  isLoadingDisk: false,
  // File browser
  currentPath: "/",
  files: [],
  isLoadingFiles: false,
  openFilePath: null,
  openFileContent: null,
  openFileOriginal: null,
  isLoadingFile: false,
  isSavingFile: false,
  bookmarks: ["/etc", "/var/log", "/home", "/root", "/tmp"],

  setVM: (vm: VM, nodeId: string) => {
    set({ vm, nodeId });
  },

  fetchProcesses: async () => {
    const { vm, nodeId } = get();
    if (!vm || !nodeId) return;
    set({ isLoadingProcesses: true });
    try {
      const res = await vmCockpitApi.getProcesses(nodeId, vm.vmid, vm.type);
      set({ processes: toArray<VMProcess>(res.data), isLoadingProcesses: false });
    } catch {
      toast.error("Prozesse konnten nicht geladen werden");
      set({ isLoadingProcesses: false });
    }
  },

  fetchServices: async () => {
    const { vm, nodeId } = get();
    if (!vm || !nodeId) return;
    set({ isLoadingServices: true });
    try {
      const res = await vmCockpitApi.getServices(nodeId, vm.vmid, vm.type);
      set({ services: toArray<VMServiceInfo>(res.data), isLoadingServices: false });
    } catch {
      toast.error("Services konnten nicht geladen werden");
      set({ isLoadingServices: false });
    }
  },

  fetchPorts: async () => {
    const { vm, nodeId } = get();
    if (!vm || !nodeId) return;
    set({ isLoadingPorts: true });
    try {
      const res = await vmCockpitApi.getPorts(nodeId, vm.vmid, vm.type);
      set({ ports: toArray<VMPort>(res.data), isLoadingPorts: false });
    } catch {
      toast.error("Ports konnten nicht geladen werden");
      set({ isLoadingPorts: false });
    }
  },

  fetchDisk: async () => {
    const { vm, nodeId } = get();
    if (!vm || !nodeId) return;
    set({ isLoadingDisk: true });
    try {
      const res = await vmCockpitApi.getDisk(nodeId, vm.vmid, vm.type);
      set({ disks: toArray<VMDisk>(res.data), isLoadingDisk: false });
    } catch {
      toast.error("Speicherinformationen konnten nicht geladen werden");
      set({ isLoadingDisk: false });
    }
  },

  killProcess: async (pid: number, signal = "TERM") => {
    const { vm, nodeId } = get();
    if (!vm || !nodeId) return;
    try {
      await vmCockpitApi.killProcess(nodeId, vm.vmid, pid, signal, vm.type);
      toast.success(`Prozess ${pid} beendet`);
      get().fetchProcesses();
    } catch {
      toast.error(`Prozess ${pid} konnte nicht beendet werden`);
    }
  },

  serviceAction: async (service: string, action: string) => {
    const { vm, nodeId } = get();
    if (!vm || !nodeId) return;
    try {
      await vmCockpitApi.serviceAction(nodeId, vm.vmid, service, action, vm.type);
      toast.success(`Service ${service}: ${action} erfolgreich`);
      get().fetchServices();
    } catch {
      toast.error(`Service-Aktion fehlgeschlagen: ${service} ${action}`);
    }
  },

  navigateTo: async (path: string) => {
    const { vm, nodeId } = get();
    if (!vm || !nodeId) return;
    set({ isLoadingFiles: true, currentPath: path });
    try {
      const res = await vmCockpitApi.listDir(nodeId, vm.vmid, path, vm.type);
      set({ files: toArray<VMFileEntry>(res.data), isLoadingFiles: false });
    } catch {
      toast.error("Verzeichnis konnte nicht geladen werden");
      set({ isLoadingFiles: false });
    }
  },

  openFile: async (path: string) => {
    const { vm, nodeId } = get();
    if (!vm || !nodeId) return;
    set({ isLoadingFile: true, openFilePath: path });
    try {
      const res = await vmCockpitApi.readFile(nodeId, vm.vmid, path, vm.type);
      const content = (res.data as unknown as { content: string }).content ?? "";
      set({ openFileContent: content, openFileOriginal: content, isLoadingFile: false });
    } catch {
      toast.error("Datei konnte nicht geladen werden");
      set({ isLoadingFile: false, openFilePath: null });
    }
  },

  closeFile: () => {
    set({ openFilePath: null, openFileContent: null, openFileOriginal: null });
  },

  saveFile: async (path: string, content: string) => {
    const { vm, nodeId } = get();
    if (!vm || !nodeId) return;
    set({ isSavingFile: true });
    try {
      await vmCockpitApi.writeFile(nodeId, vm.vmid, path, content, vm.type);
      set({ openFileOriginal: content, isSavingFile: false });
      toast.success("Datei gespeichert");
    } catch {
      toast.error("Datei konnte nicht gespeichert werden");
      set({ isSavingFile: false });
    }
  },

  deleteEntry: async (path: string) => {
    const { vm, nodeId } = get();
    if (!vm || !nodeId) return;
    try {
      await vmCockpitApi.deleteFile(nodeId, vm.vmid, path, vm.type);
      toast.success("Geloescht");
      get().navigateTo(get().currentPath);
    } catch {
      toast.error("Loeschen fehlgeschlagen");
    }
  },

  createDirectory: async (path: string) => {
    const { vm, nodeId } = get();
    if (!vm || !nodeId) return;
    try {
      await vmCockpitApi.mkdir(nodeId, vm.vmid, path, vm.type);
      toast.success("Verzeichnis erstellt");
      get().navigateTo(get().currentPath);
    } catch {
      toast.error("Verzeichnis konnte nicht erstellt werden");
    }
  },

  createFile: async (path: string) => {
    const { vm, nodeId } = get();
    if (!vm || !nodeId) return;
    try {
      await vmCockpitApi.writeFile(nodeId, vm.vmid, path, "", vm.type);
      toast.success("Datei erstellt");
      get().navigateTo(get().currentPath);
    } catch {
      toast.error("Datei konnte nicht erstellt werden");
    }
  },

  uploadFile: async (path: string, content: string) => {
    const { vm, nodeId } = get();
    if (!vm || !nodeId) return;
    try {
      await vmCockpitApi.uploadFile(nodeId, vm.vmid, path, content, vm.type);
      toast.success("Datei hochgeladen");
      get().navigateTo(get().currentPath);
    } catch {
      toast.error("Upload fehlgeschlagen");
    }
  },
}));
