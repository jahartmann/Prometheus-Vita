"use client";

import { useState } from "react";
import {
  Server,
  Monitor,
  Settings,
  Network,
  CheckCircle2,
  ChevronRight,
  ChevronLeft,
  Copy,
  Check,
  Play,
  Loader2,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { useNodeStore } from "@/stores/node-store";
import { useBackupStore } from "@/stores/backup-store";
import { backupApi } from "@/lib/api";
import type { Node, ConfigBackup } from "@/types/api";

// --- Types ---

type Scenario = "vm-ct-restore" | "node-restore" | "config-restore" | "cluster-recovery";

interface WizardState {
  step: number;
  scenario: Scenario | null;
  selectedNode: Node | null;
  selectedBackup: ConfigBackup | null;
  restoreInProgress: boolean;
  restoreResult: { success: boolean; message: string } | null;
  verificationChecks: Record<string, boolean>;
}

const SCENARIOS: { id: Scenario; label: string; description: string; icon: React.ComponentType<{ className?: string }> }[] = [
  {
    id: "vm-ct-restore",
    label: "VM/CT wiederherstellen",
    description: "Einzelne VM oder Container aus einem Backup wiederherstellen",
    icon: Monitor,
  },
  {
    id: "node-restore",
    label: "Node wiederherstellen",
    description: "Einen kompletten Proxmox-Node neu aufsetzen und konfigurieren",
    icon: Server,
  },
  {
    id: "config-restore",
    label: "Konfiguration wiederherstellen",
    description: "Proxmox-Konfigurationsdateien aus einem Config-Backup zurueckspielen",
    icon: Settings,
  },
  {
    id: "cluster-recovery",
    label: "Cluster Recovery",
    description: "Einen Proxmox-Cluster nach Ausfall neu aufbauen",
    icon: Network,
  },
];

const STEP_LABELS = [
  "Szenario waehlen",
  "Ressourcen auswaehlen",
  "Anleitung & Checkliste",
  "Ausfuehrung",
  "Verifizierung",
];

// --- Helper: Copyable command ---

function CopyableCommand({ command, label }: { command: string; label?: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(command);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="space-y-1">
      {label && <p className="text-xs font-medium text-muted-foreground">{label}</p>}
      <div className="group relative rounded-md bg-muted p-3 font-mono text-sm">
        <code>{command}</code>
        <button
          onClick={handleCopy}
          className="absolute right-2 top-2 rounded p-1 opacity-0 transition-opacity hover:bg-accent group-hover:opacity-100"
          title="Kopieren"
        >
          {copied ? <Check className="h-3.5 w-3.5 text-green-500" /> : <Copy className="h-3.5 w-3.5" />}
        </button>
      </div>
    </div>
  );
}

// --- Scenario-specific instructions ---

function getInstructions(scenario: Scenario, node: Node | null): { title: string; steps: { text: string; command?: string }[] }[] {
  const nodeName = node?.name || "<node>";

  switch (scenario) {
    case "vm-ct-restore":
      return [
        {
          title: "1. Backup-Datei lokalisieren",
          steps: [
            { text: "Pruefen Sie den Backup-Storage auf dem Ziel-Node." },
            { text: "Vzdump-Backups auflisten:", command: `ls -la /var/lib/vz/dump/` },
            { text: "Oder PBS-Backups pruefen:", command: `proxmox-backup-client list --repository <PBS-SERVER>:<DATASTORE>` },
          ],
        },
        {
          title: "2. VM wiederherstellen",
          steps: [
            { text: "VM aus Backup wiederherstellen:", command: `qmrestore /var/lib/vz/dump/vzdump-qemu-<VMID>-*.vma <VMID>` },
            { text: "Optional mit Live-Restore (VM startet waehrend Restore):", command: `qmrestore /var/lib/vz/dump/vzdump-qemu-<VMID>-*.vma <VMID> --live-restore` },
          ],
        },
        {
          title: "3. Container wiederherstellen",
          steps: [
            { text: "LXC-Container aus Backup wiederherstellen:", command: `pct restore <CTID> /var/lib/vz/dump/vzdump-lxc-<CTID>-*.tar.zst` },
            { text: "Mit spezifischem Storage:", command: `pct restore <CTID> /var/lib/vz/dump/vzdump-lxc-<CTID>-*.tar.zst --storage local-lvm` },
          ],
        },
        {
          title: "4. Nach dem Restore",
          steps: [
            { text: "VM/CT starten und Netzwerk pruefen." },
            { text: "VM starten:", command: `qm start <VMID>` },
            { text: "Container starten:", command: `pct start <CTID>` },
          ],
        },
      ];

    case "node-restore":
      return [
        {
          title: "1. Proxmox VE installieren",
          steps: [
            { text: `Installieren Sie Proxmox VE auf ${nodeName} mit dem ISO-Installer.` },
            { text: "Verwenden Sie dieselbe IP-Adresse und denselben Hostnamen wie zuvor." },
            { text: "Nach Installation: System aktualisieren:", command: `apt update && apt full-upgrade -y` },
          ],
        },
        {
          title: "2. Netzwerk konfigurieren",
          steps: [
            { text: "Netzwerk-Konfiguration wiederherstellen:" },
            { text: "Interfaces anpassen:", command: `nano /etc/network/interfaces` },
            { text: "Netzwerk neu laden:", command: `ifreload -a` },
          ],
        },
        {
          title: "3. Cluster-Konfiguration",
          steps: [
            { text: "Falls Node Teil eines Clusters war:" },
            { text: "Cluster beitreten:", command: `pvecm add <EXISTING-NODE-IP>` },
            { text: "Oder Cluster-Config manuell wiederherstellen:", command: `cp -a /backup/etc/pve/* /etc/pve/` },
          ],
        },
        {
          title: "4. Storage einrichten",
          steps: [
            { text: "Storage-Konfiguration wiederherstellen:" },
            { text: "ZFS-Pools importieren:", command: `zpool import -f <POOLNAME>` },
            { text: "Storage in PVE konfigurieren:", command: `pvesm set <STORAGE-ID> --path <PATH>` },
          ],
        },
        {
          title: "5. VMs/CTs wiederherstellen",
          steps: [
            { text: "Backups vom Backup-Storage wiederherstellen (siehe VM/CT-Restore Szenario)." },
            { text: "Alle VMs auflisten:", command: `qm list` },
            { text: "Alle Container auflisten:", command: `pct list` },
          ],
        },
      ];

    case "config-restore":
      return [
        {
          title: "1. Config-Backup identifizieren",
          steps: [
            { text: "Waehlen Sie ein Config-Backup aus der Liste unten aus." },
            { text: "Das Backup enthaelt Dateien aus /etc/pve/ und weiteren kritischen Pfaden." },
          ],
        },
        {
          title: "2. Kritische Pfade",
          steps: [
            { text: "/etc/pve/ — Cluster-Konfiguration, VM/CT-Configs, User-Management" },
            { text: "/etc/network/interfaces — Netzwerk-Konfiguration" },
            { text: "/etc/hosts — Hostname-Zuordnungen" },
            { text: "Aktuellen Stand sichern:", command: `cp -a /etc/pve /etc/pve.bak.$(date +%Y%m%d)` },
          ],
        },
        {
          title: "3. Restore durchfuehren",
          steps: [
            { text: "Sie koennen das Config-Backup ueber die Prometheus-API oder manuell zurueckspielen." },
            { text: "Nach dem Restore: Services neu starten:", command: `systemctl restart pvedaemon pveproxy pvestatd` },
          ],
        },
      ];

    case "cluster-recovery":
      return [
        {
          title: "1. Cluster-Status pruefen",
          steps: [
            { text: "Status des Clusters ermitteln:" },
            { text: "Cluster-Status anzeigen:", command: `pvecm status` },
            { text: "Erwartete Votes pruefen:", command: `pvecm expected 1` },
          ],
        },
        {
          title: "2. Quorum wiederherstellen",
          steps: [
            { text: "Falls nur ein Node verfuegbar ist, Quorum erzwingen:" },
            { text: "Erwartete Votes setzen:", command: `pvecm expected 1` },
            { text: "Corosync-Konfiguration anpassen:", command: `nano /etc/pve/corosync.conf` },
          ],
        },
        {
          title: "3. Nodes entfernen/hinzufuegen",
          steps: [
            { text: "Defekten Node aus Cluster entfernen:" },
            { text: "Node entfernen:", command: `pvecm delnode <NODENAME>` },
            { text: "Neuen Node hinzufuegen:", command: `pvecm add <EXISTING-NODE-IP>` },
          ],
        },
        {
          title: "4. PBS-Storage pruefen",
          steps: [
            { text: "Proxmox Backup Server Anbindung testen:" },
            { text: "PBS-Verzeichnisstruktur pruefen und Rechte setzen:", command: `chown backup:backup -R /mnt/datastore/<DATASTORE>` },
            { text: "Datastore verifizieren:", command: `proxmox-backup-manager datastore list` },
          ],
        },
        {
          title: "5. Cluster-Konfiguration verifizieren",
          steps: [
            { text: "Pruefen Sie, dass alle Nodes synchronisiert sind:" },
            { text: "Cluster-Status:", command: `pvecm status` },
            { text: "Node-Liste:", command: `pvecm nodes` },
          ],
        },
      ];
  }
}

function getVerificationItems(scenario: Scenario): { key: string; label: string }[] {
  switch (scenario) {
    case "vm-ct-restore":
      return [
        { key: "vm-started", label: "VM/CT erfolgreich gestartet" },
        { key: "network-ok", label: "Netzwerk erreichbar (Ping, SSH)" },
        { key: "services-running", label: "Services innerhalb der VM/CT laufen" },
        { key: "data-intact", label: "Daten und Konfiguration korrekt" },
      ];
    case "node-restore":
      return [
        { key: "node-online", label: "Node ist online und erreichbar" },
        { key: "network-ok", label: "Netzwerk korrekt konfiguriert" },
        { key: "storage-ok", label: "Storage eingebunden und verfuegbar" },
        { key: "cluster-ok", label: "Cluster-Mitgliedschaft aktiv" },
        { key: "vms-restored", label: "Alle VMs/CTs wiederhergestellt" },
        { key: "backups-ok", label: "Backup-Jobs laufen wieder" },
      ];
    case "config-restore":
      return [
        { key: "config-applied", label: "Konfiguration erfolgreich angewendet" },
        { key: "services-ok", label: "PVE-Services laufen (pvedaemon, pveproxy)" },
        { key: "web-ok", label: "Web-UI erreichbar" },
        { key: "vms-visible", label: "Alle VMs/CTs sichtbar" },
      ];
    case "cluster-recovery":
      return [
        { key: "quorum-ok", label: "Quorum hergestellt" },
        { key: "nodes-synced", label: "Alle Nodes synchronisiert" },
        { key: "ha-ok", label: "HA-Services aktiv" },
        { key: "storage-ok", label: "Shared Storage erreichbar" },
        { key: "migration-ok", label: "Live-Migration funktioniert" },
      ];
  }
}

// --- Main Wizard Component ---

export function DRWizard() {
  const { nodes } = useNodeStore();
  const { backups, fetchBackups } = useBackupStore();

  const [state, setState] = useState<WizardState>({
    step: 0,
    scenario: null,
    selectedNode: null,
    selectedBackup: null,
    restoreInProgress: false,
    restoreResult: null,
    verificationChecks: {},
  });

  const setStep = (step: number) => setState((s) => ({ ...s, step }));
  const canNext = () => {
    if (state.step === 0) return state.scenario !== null;
    if (state.step === 1) return state.selectedNode !== null;
    return true;
  };

  const handleNodeSelect = (node: Node) => {
    setState((s) => ({ ...s, selectedNode: node, selectedBackup: null }));
    fetchBackups(node.id);
  };

  const handleConfigRestore = async () => {
    if (!state.selectedBackup) return;
    setState((s) => ({ ...s, restoreInProgress: true, restoreResult: null }));
    try {
      await backupApi.restoreBackup(state.selectedBackup.id, { file_paths: [], dry_run: false });
      setState((s) => ({
        ...s,
        restoreInProgress: false,
        restoreResult: { success: true, message: "Config-Restore erfolgreich abgeschlossen." },
      }));
    } catch {
      setState((s) => ({
        ...s,
        restoreInProgress: false,
        restoreResult: { success: false, message: "Restore fehlgeschlagen. Pruefen Sie die Logs." },
      }));
    }
  };

  const toggleCheck = (key: string) => {
    setState((s) => ({
      ...s,
      verificationChecks: { ...s.verificationChecks, [key]: !s.verificationChecks[key] },
    }));
  };

  // --- Step renderers ---

  const renderStepIndicator = () => (
    <div className="flex items-center gap-2 overflow-x-auto pb-2">
      {STEP_LABELS.map((label, i) => (
        <div key={label} className="flex items-center gap-2">
          <button
            onClick={() => i <= state.step && setStep(i)}
            className={cn(
              "flex items-center gap-1.5 whitespace-nowrap rounded-full px-3 py-1 text-xs font-medium transition-colors",
              i === state.step
                ? "bg-primary text-primary-foreground"
                : i < state.step
                  ? "bg-primary/20 text-primary cursor-pointer"
                  : "bg-muted text-muted-foreground"
            )}
          >
            <span className="flex h-5 w-5 items-center justify-center rounded-full bg-background/20 text-[10px]">
              {i < state.step ? <CheckCircle2 className="h-3.5 w-3.5" /> : i + 1}
            </span>
            {label}
          </button>
          {i < STEP_LABELS.length - 1 && <ChevronRight className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />}
        </div>
      ))}
    </div>
  );

  const renderStep0 = () => (
    <div className="grid gap-4 md:grid-cols-2">
      {SCENARIOS.map((s) => {
        const Icon = s.icon;
        const selected = state.scenario === s.id;
        return (
          <Card
            key={s.id}
            className={cn(
              "cursor-pointer transition-all hover:border-primary/50",
              selected && "border-primary ring-2 ring-primary/20"
            )}
            onClick={() => setState((prev) => ({ ...prev, scenario: s.id }))}
          >
            <CardContent className="flex items-start gap-4 p-4">
              <div className={cn(
                "flex h-10 w-10 shrink-0 items-center justify-center rounded-lg",
                selected ? "bg-primary text-primary-foreground" : "bg-muted"
              )}>
                <Icon className="h-5 w-5" />
              </div>
              <div>
                <p className="font-medium">{s.label}</p>
                <p className="text-sm text-muted-foreground">{s.description}</p>
              </div>
            </CardContent>
          </Card>
        );
      })}
    </div>
  );

  const renderStep1 = () => (
    <div className="space-y-4">
      <div>
        <h3 className="text-sm font-medium mb-2">Ziel-Node auswaehlen</h3>
        <div className="grid gap-2 md:grid-cols-3">
          {nodes.map((node) => (
            <Card
              key={node.id}
              className={cn(
                "cursor-pointer transition-all hover:border-primary/50",
                state.selectedNode?.id === node.id && "border-primary ring-2 ring-primary/20"
              )}
              onClick={() => handleNodeSelect(node)}
            >
              <CardContent className="flex items-center gap-3 p-3">
                <span className={cn("h-2.5 w-2.5 rounded-full", node.is_online ? "bg-green-500" : "bg-red-500")} />
                <div>
                  <p className="text-sm font-medium">{node.name}</p>
                  <p className="text-xs text-muted-foreground">{node.is_online ? "Online" : "Offline"}</p>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>

      {state.scenario === "config-restore" && state.selectedNode && (
        <div>
          <h3 className="text-sm font-medium mb-2">Config-Backup auswaehlen</h3>
          {backups.length === 0 ? (
            <p className="text-sm text-muted-foreground">Keine Config-Backups fuer diesen Node verfuegbar.</p>
          ) : (
            <div className="space-y-2">
              {backups.map((backup) => (
                <Card
                  key={backup.id}
                  className={cn(
                    "cursor-pointer transition-all hover:border-primary/50",
                    state.selectedBackup?.id === backup.id && "border-primary ring-2 ring-primary/20"
                  )}
                  onClick={() => setState((s) => ({ ...s, selectedBackup: backup }))}
                >
                  <CardContent className="flex items-center justify-between p-3">
                    <div>
                      <p className="text-sm font-medium">
                        {new Date(backup.created_at).toLocaleString("de-DE")}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {backup.backup_type} {backup.notes && `— ${backup.notes}`}
                      </p>
                    </div>
                    <Badge variant="outline">{backup.file_count} Dateien</Badge>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );

  const renderStep2 = () => {
    if (!state.scenario) return null;
    const sections = getInstructions(state.scenario, state.selectedNode);

    return (
      <div className="space-y-6">
        {sections.map((section) => (
          <Card key={section.title}>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm">{section.title}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {section.steps.map((step, i) => (
                <div key={i}>
                  <p className="text-sm">{step.text}</p>
                  {step.command && <CopyableCommand command={step.command} />}
                </div>
              ))}
            </CardContent>
          </Card>
        ))}
      </div>
    );
  };

  const renderStep3 = () => (
    <div className="space-y-4">
      {state.scenario === "config-restore" && state.selectedBackup ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Config-Restore ausfuehren</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <p className="text-sm text-muted-foreground">
              Das Config-Backup vom{" "}
              {new Date(state.selectedBackup.created_at).toLocaleString("de-DE")}{" "}
              wird auf {state.selectedNode?.name} wiederhergestellt.
            </p>
            {state.restoreResult && (
              <div className={cn(
                "rounded-md p-3 text-sm",
                state.restoreResult.success ? "bg-green-500/10 text-green-700" : "bg-destructive/10 text-destructive"
              )}>
                {state.restoreResult.message}
              </div>
            )}
            <Button onClick={handleConfigRestore} disabled={state.restoreInProgress}>
              {state.restoreInProgress ? (
                <><Loader2 className="mr-2 h-4 w-4 animate-spin" />Restore laeuft...</>
              ) : (
                <><Play className="mr-2 h-4 w-4" />Restore starten</>
              )}
            </Button>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Manuelle Ausfuehrung</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <p className="text-sm text-muted-foreground">
              Dieses Szenario erfordert eine manuelle Ausfuehrung direkt auf dem Proxmox-Host.
              Folgen Sie der Anleitung im vorherigen Schritt und verwenden Sie die kopierbaren Befehle.
            </p>
            <div className="rounded-md bg-muted p-3">
              <p className="text-xs font-medium text-muted-foreground mb-2">Hilfreiche Befehle</p>
              <CopyableCommand command={`ssh root@${state.selectedNode?.name || "<node>"}`} label="SSH-Verbindung" />
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );

  const renderStep4 = () => {
    if (!state.scenario) return null;
    const items = getVerificationItems(state.scenario);
    const allChecked = items.every((item) => state.verificationChecks[item.key]);

    return (
      <div className="space-y-4">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Verifizierungs-Checkliste</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {items.map((item) => (
                <label
                  key={item.key}
                  className="flex cursor-pointer items-center gap-3 rounded-lg p-2 transition-colors hover:bg-accent"
                >
                  <input
                    type="checkbox"
                    checked={state.verificationChecks[item.key] || false}
                    onChange={() => toggleCheck(item.key)}
                    className="h-4 w-4 rounded border-muted-foreground"
                  />
                  <span className={cn(
                    "text-sm",
                    state.verificationChecks[item.key] && "text-muted-foreground line-through"
                  )}>
                    {item.label}
                  </span>
                </label>
              ))}
            </div>
          </CardContent>
        </Card>

        {allChecked && (
          <div className="rounded-md bg-green-500/10 p-4 text-center">
            <CheckCircle2 className="mx-auto mb-2 h-8 w-8 text-green-500" />
            <p className="font-medium text-green-700">Wiederherstellung erfolgreich verifiziert!</p>
            <p className="text-sm text-muted-foreground">Alle Pruefpunkte wurden abgehakt.</p>
          </div>
        )}
      </div>
    );
  };

  const stepRenderers = [renderStep0, renderStep1, renderStep2, renderStep3, renderStep4];

  return (
    <div className="space-y-6">
      {renderStepIndicator()}

      <div className="min-h-[300px]">
        {stepRenderers[state.step]()}
      </div>

      <div className="flex items-center justify-between border-t pt-4">
        <Button
          variant="outline"
          onClick={() => setStep(state.step - 1)}
          disabled={state.step === 0}
        >
          <ChevronLeft className="mr-1 h-4 w-4" /> Zurueck
        </Button>
        <Button
          onClick={() => setStep(state.step + 1)}
          disabled={state.step === STEP_LABELS.length - 1 || !canNext()}
        >
          Weiter <ChevronRight className="ml-1 h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}
