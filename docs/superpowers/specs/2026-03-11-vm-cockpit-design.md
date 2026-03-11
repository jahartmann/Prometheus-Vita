# VM Cockpit — Design Spec

## Ziel

Jede VM/Container bekommt eine eigene Detail-Seite mit integriertem Terminal, Dateibrowser, System-Einblick, Monitoring und KI-Assistent. Feingranulare Berechtigungen pro VM/Gruppe. Ergaenzt durch Intelligenz, Automatisierung und Dependency-Mapping.

## Architektur-Ueberblick

Neue Route: `/nodes/{nodeId}/vms/{vmid}` mit 5 Tabs:

```
┌─────────────────────────────────────────────────┐
│  VM: web-proxy-01 (LXC 101)    ● Running        │
│  Node: pve-node-1   Tags: [web] [production]    │
├──────┬──────────┬────────┬───────────┬──────────┤
│ Shell│ Dateien  │ System │ Monitoring│ KI-Assist│
└──────┴──────────┴────────┴───────────┴──────────┘
```

Kein Agent/Daemon in den VMs noetig. Alle Interaktion ueber:
- LXC: `pct exec`, `pct pull/push`
- QEMU mit Guest Agent: `qm guest exec`, `qm guest file-read/write`
- QEMU ohne Guest Agent: SSH-Tunnel durch Backend

---

## Tab 1: Shell — Integriertes Terminal

### Features
- xterm.js + WebSocket fuer echtes Terminal im Browser
- Bis zu 4 Sessions gleichzeitig pro VM (Tab-Interface)
- Befehlshistorie persistent in DB (pro VM)
- Schnellbefehle-Leiste: Konfigurierbare Buttons (Logs, Disk, Services)
- Copy/Paste, Textsuche in Ausgabe
- Session-Recording (optional) fuer Audit-Trail
- Mobile-optimiert: Touch-Toolbar (Tab, Ctrl+C, Pfeiltasten)

### Verbindungsmethoden
| VM-Typ | Methode |
|--------|---------|
| LXC | `pct exec {vmid} -- /bin/bash` via WebSocket |
| QEMU + Guest Agent | `qm guest exec` |
| QEMU ohne Agent | SSH-Tunnel durch Backend (User hinterlegt SSH-Key) |

### Datenfluss
```
Browser (xterm.js) ↔ WebSocket ↔ Backend (Auth + Permissions) ↔ Proxmox API ↔ VM Shell
```

### Sicherheit
- JWT + VM-spezifische Berechtigung (`vm.shell`) pro WebSocket-Verbindung
- Session-Timeout nach Inaktivitaet (konfigurierbar, default 30 Min)
- Optionale Warnung bei gefaehrlichen Befehlen (rm -rf, shutdown)

---

## Tab 2: Dateien — Dateibrowser & Editor

### Features
- Verzeichnisbaum mit Navigation, Sortierung (Name/Groesse/Datum)
- Upload/Download (einzeln + Ordner als ZIP)
- Erstellen, Umbenennen, Loeschen, Verschieben
- Berechtigungen anzeigen (owner, group, chmod)
- Inline-Editor mit Syntax-Highlighting (nginx.conf, yaml, json, shell, etc.)
- Diff-Ansicht vor Speichern
- Automatisches Backup vor Aenderung (letzte 5 Versionen)
- Bookmarks fuer haeufige Pfade
- "Letzte Dateien" Liste
- Dateisuche (grep im Browser)

### Technische Umsetzung
| Operation | LXC | QEMU (Guest Agent) | QEMU (SSH) |
|-----------|-----|-------------------|------------|
| Verzeichnis listen | `pct exec -- ls -la` | `qm guest exec -- ls -la` | SFTP readdir |
| Datei lesen | `pct pull` | `qm guest file-read` | SFTP get |
| Datei schreiben | `pct push` | `qm guest file-write` | SFTP put |

### Sicherheit
- Pfad-Beschraenkung: Admin kann erlaubte Verzeichnisse pro VM definieren
- Schreibzugriff separat berechtigbar (`vm.files.read` vs `vm.files.write`)
- Aenderungen im Audit-Log (wer, welche Datei, wann)
- Max. Dateigroesse fuer Editor: 2 MB (groessere nur Download)

---

## Tab 3: System — Live-Systemeinblick

### Features
- **Prozesse:** Live-Liste (CPU/RAM pro Prozess), sortierbar, filterbar. Kill mit Bestaetigung.
- **Services:** Systemd-Units mit Status. Start/Stop/Restart/Enable/Disable. Journal-Logs inline.
- **Netzwerk:** Offene Ports + Prozess (`ss -tlnp`), aktive Verbindungen, Firewall-Regeln.
- **Disk:** Mountpoints mit Belegung (df), Treemap-Visualisierung der groessten Verzeichnisse (du).
- **Pakete:** Installierte Pakete, verfuegbare Updates, Ein-Klick-Update.

### Datenerhebung
Befehlsausfuehrung via pct exec / guest agent / SSH. Strukturierte Ausgabe parsen (ps aux, systemctl --output=json, ss -tlnp, df -h). Auto-Refresh konfigurierbar (2s/5s/10s/manuell).

### Sicherheit
- `vm.system.view` — Prozesse/Services/Ports anzeigen
- `vm.system.service` — Services steuern
- `vm.system.kill` — Prozesse beenden
- `vm.system.packages` — Pakete installieren (nur Admin)

---

## Tab 4: Monitoring

Bestehende RRD-Metriken (CPU, RAM, Disk, Netzwerk) eingebettet in VM-Kontext. Bereits implementiert via `RRDBandwidth`-Komponente — wird wiederverwendet und um VM-spezifische Metriken erweitert.

---

## Tab 5: KI-Assistent

### Assistenz-Modus (Standard)
- VM-kontextbezogener Chat — KI kennt automatisch VM-Zustand
- Beispiele:
  - "Warum ist die CPU so hoch?" → KI fuehrt top/ps aus, analysiert, erklaert
  - "Richte nginx als Reverse Proxy ein" → KI schlaegt Befehle vor, User bestaetigt
  - "Was hat sich seit gestern geaendert?" → Config-Diff
- Befehlsvorschlaege als Code-Block, User klickt "Ausfuehren" oder "Ablehnen"

### Proaktiv-Modus (Optional pro VM)
- Anomalie-Erkennung: Ungewoehnliches Disk-Wachstum, Memory-Leak-Muster, Service-Crashes
- Auto-Aktionen bei FullAuto: Log-Rotation, Service-Neustart (max 3x dann Alarm), Snapshot vor Aenderungen
- Benachrichtigungen als Badge + Reflex-Regel-Integration

### Neue AI-Agent-Tools
| Tool | Beschreibung | ReadOnly |
|------|-------------|----------|
| `vm_exec` | Befehl in VM ausfuehren | Nein |
| `vm_file_read` | Datei aus VM lesen | Ja |
| `vm_file_write` | Datei in VM schreiben | Nein |
| `vm_processes` | Prozessliste abrufen | Ja |
| `vm_services` | Service-Status abrufen | Ja |
| `vm_service_action` | Service starten/stoppen | Nein |
| `vm_disk_usage` | Disk-Analyse | Ja |
| `vm_network_info` | Ports + Verbindungen | Ja |

### Sicherheit
- Selbe VM-Berechtigungen wie manuelle Aktionen
- Proaktiv nur mit explizitem Opt-In pro VM
- Alle KI-Aktionen im Audit-Log (Quelle: agent)
- Autonomie-Level: ReadOnly / Confirm / FullAuto

---

## Berechtigungssystem

### Permission-Struktur
```
vm.view             — VM sehen + Monitoring
vm.shell            — Terminal nutzen
vm.files.read       — Dateien lesen
vm.files.write      — Dateien bearbeiten
vm.system.view      — Prozesse/Services anzeigen
vm.system.service   — Services steuern
vm.system.kill      — Prozesse beenden
vm.system.packages  — Pakete installieren
vm.power            — Start/Stop/Restart
vm.snapshots        — Snapshots verwalten
vm.ai.proactive     — KI-Proaktivmodus erlauben
```

### VM-Gruppen
- Tag-basierte Gruppierung (Tag "production" = Gruppe)
- Berechtigungen auf Gruppen-Ebene statt pro VM
- Wiederverwendung bestehender Tags

### Datenmodell
```sql
CREATE TABLE vm_permissions (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    target_type VARCHAR(10) NOT NULL, -- 'vm' | 'group'
    target_id VARCHAR(50) NOT NULL,   -- vmid oder group_id
    node_id UUID REFERENCES nodes(id),
    permissions TEXT[] NOT NULL,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

### UI
Settings → Benutzerverwaltung: Matrix-Ansicht (User x VM/Gruppe → Checkboxen pro Berechtigung).

---

## Ergaenzende Features (A/B/D)

### A) Intelligenz & Proaktivitaet
- **Right-Sizing:** Woechentliche RRD-Analyse → Empfehlungen ("VM 103 nutzt 12% RAM, reduzieren?")
- **Anomalie-Erkennung:** 7-Tage-Baseline, Abweichungen melden (Standardabweichung, keine ML-Pipeline)
- **Health-Score:** 0-100 pro VM (CPU, RAM, Disk, Uptime, Crash-Haeufigkeit). Dashboard sortiert nach "braucht Aufmerksamkeit"

### B) Workflow & Automatisierung
- **Snapshot-Rotation:** Pro VM konfigurierbar ("letzte 5 taegliche, 4 woechentliche"). Als Reflex-Regel.
- **Geplante Aktionen:** Wartungsfenster via Reflex-Cron ("Sonntag 03:00 Dev-VMs stoppen")
- **VM-Templates:** VM als Template markieren, Ein-Klick-Clone via Proxmox-Clone-API

### D) Ueberblick & Beziehungen
- **VM-Gruppen:** Tag-basiert (im Permissions-System enthalten)
- **Dependency-Map:** Manuelle Verknuepfung (nginx → app → postgres). Graph-Visualisierung. Ausfallkette anzeigen.
- **Change-Log:** VM-Config-Aenderungen tracken (Proxmox Task-Log + Audit-Log). Timeline pro VM.

---

## Implementierungsreihenfolge

1. **Phase 1:** VM-Cockpit mit Shell + System-Tab
2. **Phase 2:** Dateien-Tab + KI-Assistent
3. **Phase 3:** Berechtigungssystem
4. **Phase 4:** A/B/D Features (Intelligenz, Automation, Ueberblick)
