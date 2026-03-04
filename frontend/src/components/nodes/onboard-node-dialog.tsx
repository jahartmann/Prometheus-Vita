"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { Loader2, CheckCircle2, XCircle, ChevronDown, ChevronRight } from "lucide-react";
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
import { useNodeStore } from "@/stores/node-store";
import { AddNodeDialog } from "./add-node-dialog";
import type { Node, NodeType, OnboardNodeRequest } from "@/types/api";

interface OnboardNodeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

type StepStatus = "pending" | "active" | "done" | "error";

interface ProgressStep {
  label: string;
  status: StepStatus;
}

const INITIAL_STEPS: ProgressStep[] = [
  { label: "Authentifiziere bei Proxmox...", status: "pending" },
  { label: "Erstelle API-Token...", status: "pending" },
  { label: "Konfiguriere SSH-Zugang...", status: "pending" },
  { label: "Pruefe Verbindung...", status: "pending" },
  { label: "Fertig!", status: "pending" },
];

export function OnboardNodeDialog({ open, onOpenChange }: OnboardNodeDialogProps) {
  const { addNode } = useNodeStore();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [showManualDialog, setShowManualDialog] = useState(false);
  const [steps, setSteps] = useState<ProgressStep[]>(INITIAL_STEPS);
  const [error, setError] = useState<string | null>(null);
  const [showProgress, setShowProgress] = useState(false);
  const stepTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const [form, setForm] = useState<OnboardNodeRequest>({
    name: "",
    type: "pve",
    hostname: "",
    password: "",
    port: 8006,
    ssh_port: 22,
    username: "root@pam",
  });

  const updateField = <K extends keyof OnboardNodeRequest>(
    field: K,
    value: OnboardNodeRequest[K]
  ) => {
    setForm((prev) => ({ ...prev, [field]: value }));
    setError(null);
  };

  const resetForm = useCallback(() => {
    setForm({
      name: "",
      type: "pve",
      hostname: "",
      password: "",
      port: 8006,
      ssh_port: 22,
      username: "root@pam",
    });
    setSteps(INITIAL_STEPS);
    setError(null);
    setShowProgress(false);
    setIsSubmitting(false);
    setShowAdvanced(false);
    if (stepTimerRef.current) {
      clearInterval(stepTimerRef.current);
      stepTimerRef.current = null;
    }
  }, []);

  useEffect(() => {
    return () => {
      if (stepTimerRef.current) {
        clearInterval(stepTimerRef.current);
      }
    };
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    setShowProgress(true);
    setError(null);

    // Start simulated progress steps
    let currentStep = 0;
    setSteps((prev) =>
      prev.map((s, i) => (i === 0 ? { ...s, status: "active" as StepStatus } : s))
    );

    stepTimerRef.current = setInterval(() => {
      currentStep++;
      if (currentStep < 4) {
        setSteps((prev) =>
          prev.map((s, i) => {
            if (i < currentStep) return { ...s, status: "done" as StepStatus };
            if (i === currentStep) return { ...s, status: "active" as StepStatus };
            return s;
          })
        );
      } else {
        // Stop at step 4 (index 3), wait for API response
        if (stepTimerRef.current) {
          clearInterval(stepTimerRef.current);
          stepTimerRef.current = null;
        }
      }
    }, 1500);

    try {
      const response = await api.post<Node>("/nodes/onboard", form);

      // Clear timer and jump to done
      if (stepTimerRef.current) {
        clearInterval(stepTimerRef.current);
        stepTimerRef.current = null;
      }

      setSteps((prev) =>
        prev.map((s) => ({ ...s, status: "done" as StepStatus }))
      );

      // Wait a moment so the user sees "Fertig!"
      setTimeout(() => {
        addNode(response.data);
        onOpenChange(false);
        resetForm();
      }, 1000);
    } catch (err: unknown) {
      if (stepTimerRef.current) {
        clearInterval(stepTimerRef.current);
        stepTimerRef.current = null;
      }

      const message =
        err instanceof Error
          ? err.message
          : "Onboarding fehlgeschlagen. Bitte pruefen Sie die Zugangsdaten.";

      // Try to extract error from axios response
      let apiMessage = message;
      if (
        typeof err === "object" &&
        err !== null &&
        "response" in err
      ) {
        const axiosErr = err as { response?: { data?: { error?: string; message?: string } } };
        apiMessage =
          axiosErr.response?.data?.message ||
          axiosErr.response?.data?.error ||
          message;
      }

      setSteps((prev) =>
        prev.map((s) => {
          if (s.status === "active") return { ...s, status: "error" as StepStatus };
          if (s.status === "pending") return s;
          return s;
        })
      );
      setError(apiMessage);
      setIsSubmitting(false);
    }
  };

  const isValid = form.name && form.hostname && form.password;

  return (
    <>
      <Dialog
        open={open}
        onOpenChange={(o) => {
          if (!isSubmitting) {
            onOpenChange(o);
            if (!o) resetForm();
          }
        }}
      >
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Node einrichten</DialogTitle>
            <DialogDescription>
              Verbinden Sie einen Proxmox Node automatisch. Token und SSH werden
              automatisch konfiguriert.
            </DialogDescription>
          </DialogHeader>

          {!showProgress ? (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="onboard-name">Name</Label>
                  <Input
                    id="onboard-name"
                    placeholder="pve-node-01"
                    value={form.name}
                    onChange={(e) => updateField("name", e.target.value)}
                    required
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="onboard-type">Typ</Label>
                  <select
                    id="onboard-type"
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

              <div className="space-y-2">
                <Label htmlFor="onboard-hostname">Hostname / IP</Label>
                <Input
                  id="onboard-hostname"
                  placeholder="192.168.1.100 oder pve01.local"
                  value={form.hostname}
                  onChange={(e) => updateField("hostname", e.target.value)}
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="onboard-password">Passwort</Label>
                <Input
                  id="onboard-password"
                  type="password"
                  placeholder="Root-Passwort des Proxmox Nodes"
                  value={form.password}
                  onChange={(e) => updateField("password", e.target.value)}
                  required
                />
              </div>

              {/* Advanced toggle */}
              <button
                type="button"
                className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors"
                onClick={() => setShowAdvanced(!showAdvanced)}
              >
                {showAdvanced ? (
                  <ChevronDown className="h-4 w-4" />
                ) : (
                  <ChevronRight className="h-4 w-4" />
                )}
                Erweitert
              </button>

              {showAdvanced && (
                <div className="space-y-4 border-l-2 border-muted pl-4">
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="onboard-port">API Port</Label>
                      <Input
                        id="onboard-port"
                        type="number"
                        value={form.port}
                        onChange={(e) =>
                          updateField("port", parseInt(e.target.value) || 8006)
                        }
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="onboard-ssh-port">SSH Port</Label>
                      <Input
                        id="onboard-ssh-port"
                        type="number"
                        value={form.ssh_port}
                        onChange={(e) =>
                          updateField("ssh_port", parseInt(e.target.value) || 22)
                        }
                      />
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="onboard-username">Benutzername</Label>
                    <Input
                      id="onboard-username"
                      value={form.username}
                      onChange={(e) => updateField("username", e.target.value)}
                    />
                  </div>
                </div>
              )}

              {error && (
                <div className="flex items-center gap-2 rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-700 dark:text-red-400">
                  <XCircle className="h-4 w-4 shrink-0" />
                  <span>{error}</span>
                </div>
              )}

              <DialogFooter className="flex-col gap-3 sm:flex-col">
                <div className="flex justify-end gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => {
                      onOpenChange(false);
                      resetForm();
                    }}
                  >
                    Abbrechen
                  </Button>
                  <Button type="submit" disabled={isSubmitting || !isValid}>
                    Node einrichten
                  </Button>
                </div>
                <div className="w-full text-center">
                  <button
                    type="button"
                    className="text-xs text-muted-foreground hover:text-foreground transition-colors underline"
                    onClick={() => {
                      onOpenChange(false);
                      resetForm();
                      setShowManualDialog(true);
                    }}
                  >
                    Erweitert: Manuell mit API-Token hinzufuegen &rarr;
                  </button>
                </div>
              </DialogFooter>
            </form>
          ) : (
            <div className="space-y-4 py-4">
              <div className="space-y-3">
                {steps.map((step, i) => (
                  <div key={i} className="flex items-center gap-3">
                    <div className="shrink-0">
                      {step.status === "pending" && (
                        <div className="h-5 w-5 rounded-full border-2 border-muted" />
                      )}
                      {step.status === "active" && (
                        <Loader2 className="h-5 w-5 animate-spin text-primary" />
                      )}
                      {step.status === "done" && (
                        <CheckCircle2 className="h-5 w-5 text-green-500" />
                      )}
                      {step.status === "error" && (
                        <XCircle className="h-5 w-5 text-red-500" />
                      )}
                    </div>
                    <span
                      className={`text-sm ${
                        step.status === "pending"
                          ? "text-muted-foreground"
                          : step.status === "active"
                          ? "text-foreground font-medium"
                          : step.status === "done"
                          ? "text-green-600 dark:text-green-400"
                          : "text-red-600 dark:text-red-400"
                      }`}
                    >
                      {step.label}
                    </span>
                  </div>
                ))}
              </div>

              {error && (
                <div className="flex items-center gap-2 rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-700 dark:text-red-400">
                  <XCircle className="h-4 w-4 shrink-0" />
                  <span>{error}</span>
                </div>
              )}

              {error && (
                <DialogFooter>
                  <Button
                    variant="outline"
                    onClick={() => {
                      setShowProgress(false);
                      setSteps(INITIAL_STEPS);
                      setError(null);
                    }}
                  >
                    Zurueck
                  </Button>
                </DialogFooter>
              )}
            </div>
          )}
        </DialogContent>
      </Dialog>

      <AddNodeDialog open={showManualDialog} onOpenChange={setShowManualDialog} />
    </>
  );
}
