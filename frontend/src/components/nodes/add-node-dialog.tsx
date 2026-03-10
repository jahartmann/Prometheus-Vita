"use client";

import { useState } from "react";
import { Loader2, CheckCircle2, XCircle } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import api from "@/lib/api";
import { toast } from "sonner";
import { useNodeStore } from "@/stores/node-store";
import type {
  CreateNodeRequest,
  Node,
  NodeType,
  TestConnectionResponse,
} from "@/types/api";

interface AddNodeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function AddNodeDialog({ open, onOpenChange }: AddNodeDialogProps) {
  const { addNode } = useNodeStore();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isTesting, setIsTesting] = useState(false);
  const [testResult, setTestResult] = useState<TestConnectionResponse | null>(
    null
  );

  const [form, setForm] = useState<CreateNodeRequest>({
    name: "",
    type: "pve" as NodeType,
    hostname: "",
    port: 8006,
    api_token_id: "",
    api_token_secret: "",
  });

  const updateField = <K extends keyof CreateNodeRequest>(
    field: K,
    value: CreateNodeRequest[K]
  ) => {
    setForm((prev) => ({ ...prev, [field]: value }));
    setTestResult(null);
  };

  const handleTest = async () => {
    setIsTesting(true);
    setTestResult(null);
    try {
      const response = await api.post<TestConnectionResponse>(
        "/nodes/test",
        {
          hostname: form.hostname,
          port: form.port,
          type: form.type,
          api_token_id: form.api_token_id,
          api_token_secret: form.api_token_secret,
        }
      );
      setTestResult(response.data);
    } catch {
      setTestResult({ success: false, error: "Verbindung fehlgeschlagen" });
    }
    setIsTesting(false);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      const response = await api.post<Node>("/nodes", form);
      addNode(response.data);
      onOpenChange(false);
      resetForm();
      toast.success("Node hinzugefuegt");
    } catch {
      toast.error("Fehler beim Hinzufuegen des Nodes");
    }
    setIsSubmitting(false);
  };

  const resetForm = () => {
    setForm({
      name: "",
      type: "pve",
      hostname: "",
      port: 8006,
      api_token_id: "",
      api_token_secret: "",
    });
    setTestResult(null);
  };

  const isValid =
    form.name && form.hostname && form.api_token_id && form.api_token_secret;

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        onOpenChange(o);
        if (!o) resetForm();
      }}
    >
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Node hinzufuegen</DialogTitle>
          <DialogDescription>
            Verbinden Sie einen neuen Proxmox Node mit Prometheus.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="name">Name</Label>
              <Input
                id="name"
                placeholder="pve-node-01"
                value={form.name}
                onChange={(e) => updateField("name", e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="type">Typ</Label>
              <select
                id="type"
                value={form.type}
                onChange={(e) =>
                  updateField("type", e.target.value as NodeType)
                }
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm"
              >
                <option value="pve">Proxmox VE</option>
                <option value="pbs">Proxmox Backup Server</option>
              </select>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="hostname">Hostname / IP</Label>
              <Input
                id="hostname"
                placeholder="pve01.local oder 192.168.1.100"
                value={form.hostname}
                onChange={(e) => updateField("hostname", e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="port">API Port</Label>
              <Input
                id="port"
                type="number"
                value={form.port}
                onChange={(e) => updateField("port", parseInt(e.target.value))}
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="token_id">API Token ID</Label>
            <Input
              id="token_id"
              placeholder="root@pam!prometheus"
              value={form.api_token_id}
              onChange={(e) => updateField("api_token_id", e.target.value)}
              required
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="token_secret">API Token Secret</Label>
            <Input
              id="token_secret"
              type="password"
              placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
              value={form.api_token_secret}
              onChange={(e) => updateField("api_token_secret", e.target.value)}
              required
            />
          </div>

          {testResult && (
            <div
              className={`flex items-center gap-2 rounded-lg border p-3 text-sm ${
                testResult.success
                  ? "border-green-500/30 bg-green-500/10 text-green-700 dark:text-green-400"
                  : "border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-400"
              }`}
            >
              {testResult.success ? (
                <CheckCircle2 className="h-4 w-4 shrink-0" />
              ) : (
                <XCircle className="h-4 w-4 shrink-0" />
              )}
              <span>
                {testResult.success
                  ? "Verbindung erfolgreich"
                  : testResult.error || "Verbindung fehlgeschlagen"}
              </span>
              {testResult.node && (
                <span className="ml-auto font-medium">
                  {testResult.node} ({testResult.version})
                </span>
              )}
            </div>
          )}

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={handleTest}
              disabled={
                isTesting ||
                !form.hostname ||
                !form.api_token_id ||
                !form.api_token_secret
              }
            >
              {isTesting ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Teste...
                </>
              ) : (
                "Verbindung testen"
              )}
            </Button>
            <Button type="submit" disabled={isSubmitting || !isValid}>
              {isSubmitting ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Wird hinzugefuegt...
                </>
              ) : (
                "Hinzufuegen"
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
