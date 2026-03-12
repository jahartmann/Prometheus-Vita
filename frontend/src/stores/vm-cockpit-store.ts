import { create } from "zustand";
import { toast } from "sonner";
import { toArray } from "@/lib/api";
import { vmCockpitApi, extractCockpitError, type VMCockpitError } from "@/lib/vm-api";
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
  processesError: VMCockpitError | null;
  servicesError: VMCockpitError | null;
  portsError: VMCockpitError | null;
  diskError: VMCockpitError | null;
  filesError: VMCockpitError | null;
  fileError: VMCockpitError | null;
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
  processesError: null,
  servicesError: null,
  portsError: null,
  diskError: null,
  filesError: null,
  fileError: null,

  setVM: (vm: VM, nodeId: string) => {
    set({ vm, nodeId });
  },

  fetchProcesses: async () => {
    const { vm, nodeId } = get();
    if (!vm || !nodeId) return;
    set({ isLoadingProcesses: true, processesError: null });
    try {
      const res = await vmCockpitApi.getProcesses(nodeId, vm.vmid, vm.type);
      set({ processes: toArray<VMProcess>(res.data), isLoadingProcesses: false });
    } catch (error) {
      set({ isLoadingProcesses: false, processesError: extractCockpitError(error) });
    }
  },

  fetchServices: async () => {
    const { vm, nodeId } = get();
    if (!vm || !nodeId) return;
    set({ isLoadingServices: true, servicesError: null });
    try {
      const res = await vmCockpitApi.getServices(nodeId, vm.vmid, vm.type);
      set({ services: toArray<VMServiceInfo>(res.data), isLoadingServices: false });
    } catch (error) {
      set({ isLoadingServices: false, servicesError: extractCockpitError(error) });
    }
  },

  fetchPorts: async () => {
    const { vm, nodeId } = get();
    if (!vm || !nodeId) return;
    set({ isLoadingPorts: true, portsError: null });
    try {
      const res = await vmCockpitApi.getPorts(nodeId, vm.vmid, vm.type);
      set({ ports: toArray<VMPort>(res.data), isLoadingPorts: false });
    } catch (error) {
      set({ isLoadingPorts: false, portsError: extractCockpitError(error) });
    }
  },

  fetchDisk: async () => {
    const { vm, nodeId } = get();
    if (!vm || !nodeId) return;
    set({ isLoadingDisk: true, diskError: null });
    try {
      const res = await vmCockpitApi.getDisk(nodeId, vm.vmid, vm.type);
      set({ disks: toArray<VMDisk>(res.data), isLoadingDisk: false });
    } catch (error) {
      set({ isLoadingDisk: false, diskError: extractCockpitError(error) });
    }
  },

  killProcess: async (pid: number, signal = "TERM") => {
    const { vm, nodeId } = get();
    if (!vm || !nodeId) return;
    try {
      await vmCockpitApi.killProcess(nodeId, vm.vmid, pid, signal, vm.type);
      toast.success(`Prozess ${pid} beendet`);
      get().fetchProcesses();
    } catch (error) {
      const err = extractCockpitError(error);
      toast.error(err.message, { description: err.details });
    }
  },

  serviceAction: async (service: string, action: string) => {
    const { vm, nodeId } = get();
    if (!vm || !nodeId) return;
    try {
      await vmCockpitApi.serviceAction(nodeId, vm.vmid, service, action, vm.type);
      toast.success(`Service ${service}: ${action} erfolgreich`);
      get().fetchServices();
    } catch (error) {
      const err = extractCockpitError(error);
      toast.error(err.message, { description: err.details });
    }
  },

  navigateTo: async (path: string) => {
    const { vm, nodeId } = get();
    if (!vm || !nodeId) return;
    set({ isLoadingFiles: true, currentPath: path, filesError: null });
    try {
      const res = await vmCockpitApi.listDir(nodeId, vm.vmid, path, vm.type);
      set({ files: toArray<VMFileEntry>(res.data), isLoadingFiles: false });
    } catch (error) {
      set({ isLoadingFiles: false, filesError: extractCockpitError(error) });
    }
  },

  openFile: async (path: string) => {
    const { vm, nodeId } = get();
    if (!vm || !nodeId) return;
    set({ isLoadingFile: true, openFilePath: path, fileError: null });
    try {
      const res = await vmCockpitApi.readFile(nodeId, vm.vmid, path, vm.type);
      const content = (res.data as unknown as { content: string }).content ?? "";
      set({ openFileContent: content, openFileOriginal: content, isLoadingFile: false });
    } catch (error) {
      set({ isLoadingFile: false, openFilePath: null, fileError: extractCockpitError(error) });
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
    } catch (error) {
      const err = extractCockpitError(error);
      toast.error(err.message, { description: err.details });
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
    } catch (error) {
      const err = extractCockpitError(error);
      toast.error(err.message, { description: err.details });
    }
  },

  createDirectory: async (path: string) => {
    const { vm, nodeId } = get();
    if (!vm || !nodeId) return;
    try {
      await vmCockpitApi.mkdir(nodeId, vm.vmid, path, vm.type);
      toast.success("Verzeichnis erstellt");
      get().navigateTo(get().currentPath);
    } catch (error) {
      const err = extractCockpitError(error);
      toast.error(err.message, { description: err.details });
    }
  },

  createFile: async (path: string) => {
    const { vm, nodeId } = get();
    if (!vm || !nodeId) return;
    try {
      await vmCockpitApi.writeFile(nodeId, vm.vmid, path, "", vm.type);
      toast.success("Datei erstellt");
      get().navigateTo(get().currentPath);
    } catch (error) {
      const err = extractCockpitError(error);
      toast.error(err.message, { description: err.details });
    }
  },

  uploadFile: async (path: string, content: string) => {
    const { vm, nodeId } = get();
    if (!vm || !nodeId) return;
    try {
      await vmCockpitApi.uploadFile(nodeId, vm.vmid, path, content, vm.type);
      toast.success("Datei hochgeladen");
      get().navigateTo(get().currentPath);
    } catch (error) {
      const err = extractCockpitError(error);
      toast.error(err.message, { description: err.details });
    }
  },
}));
