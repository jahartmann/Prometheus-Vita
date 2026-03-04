# Prometheus - Architekturplan & Technologiekonzept

> Intelligentes Proxmox Infrastructure Management mit KI-Agent-System

---

## 1. Vision & Positionierung

Prometheus ist **kein Ersatz** für die Proxmox WebGUI, sondern eine **übergeordnete Orchestrierungs- und Intelligenzschicht**, die:

- Einen einheitlichen Überblick über **alle** Proxmox-Nodes gibt (auch außerhalb von Clustern)
- Konfigurationen sichert und granular wiederherstellt
- Durch einen **KI-Agenten** die Infrastruktur autonom überwacht, optimiert und steuert
- Disaster Recovery auf ein neues Level hebt
- Über Telegram & E-Mail steuerbar und benachrichtigungsfähig ist

---

## 2. Technologie-Stack (Empfehlung)

### 2.1 Backend: **Go (Golang)**

| Aspekt | Begründung |
|--------|-----------|
| **Performance** | Kompiliert, minimal RAM, ideal für langlebige Monitoring-Daemons |
| **Concurrency** | Goroutines für paralleles Monitoring hunderter VMs/Nodes |
| **Single Binary** | Ein Binary deployen – perfekt für Infrastruktur-Tools |
| **SSH Native** | `golang.org/x/crypto/ssh` – erstklassiger SSH-Support |
| **Ollama** | Ollama ist eine REST-API – Go kann das nativ ohne Python-Abhängigkeiten |

**Framework:** [Echo](https://echo.labstack.com/) – Enterprise-grade, WebSocket-Support, HTTP/2, strukturierte Middleware

**Schlüssel-Libraries:**

```
# Proxmox VE API
github.com/luthermonson/go-proxmox      # Umfassendster PVE-Client

# Proxmox Backup Server
github.com/elbandi/go-proxmox-backup-client

# SSH
golang.org/x/crypto/ssh                  # SSH-Client/Server
github.com/gliderlabs/ssh                # High-Level SSH Abstraktion

# KI/LLM
github.com/ollama/ollama/api             # Offizieller Ollama Go Client

# Telegram
github.com/go-telegram/bot               # Offiziell von Telegram unterstützt

# E-Mail
github.com/wneessen/go-mail              # Moderner SMTP Client

# Datenbank
github.com/jackc/pgx/v5                  # PostgreSQL Driver
github.com/redis/go-redis/v9             # Redis Client

# WebSocket
github.com/gorilla/websocket             # WebSocket für Echo
```

### 2.2 Frontend: **Next.js 15 + React 19 + TypeScript**

| Aspekt | Begründung |
|--------|-----------|
| **Ökosystem** | Größtes React-Ökosystem, riesige Community |
| **App Router** | Verschachtelte Layouts ideal für Dashboard-Views |
| **RSC** | React Server Components für effizientes Rendering |
| **Real-Time** | WebSocket-Integration für Live-Monitoring |

**UI-Stack:**

```
shadcn/ui          # Basis-Komponenten (Tailwind-native, volle Code-Kontrolle)
Tremor              # Dashboard-spezifische Komponenten (KPIs, Sparklines, Metriken)
Recharts            # Charts für Monitoring-Graphen
TailwindCSS 4       # Styling
Framer Motion       # Animationen
React Flow          # Netzwerk-Topologie-Visualisierung
```

### 2.3 Datenbank & Infrastruktur

```
PostgreSQL 17       # Primäre Datenbank (Nodes, Configs, Users, Audit-Log)
Redis 8             # Cache, Real-Time Pub/Sub, Session-Store, Job-Queue
MinIO / S3          # Konfigurationsbackup-Speicher (optional, alternativ lokaler Storage)
```

### 2.4 Deployment

```
Docker Compose      # Primäres Deployment (Backend + Frontend + DB + Redis)
Systemd             # Alternative: Native Binary als Service
```

---

## 3. Systemarchitektur

```
┌──────────────────────────────────────────────────────────────────┐
│                        PROMETHEUS                                │
│                                                                  │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────┐   │
│  │   Next.js    │◄──►│   Go API     │◄──►│   PostgreSQL     │   │
│  │   Frontend   │    │   Backend    │    │   + Redis        │   │
│  │              │    │              │    └──────────────────┘   │
│  │  Dashboard   │    │  REST API    │                            │
│  │  Chat UI     │    │  WebSocket   │    ┌──────────────────┐   │
│  │  Topology    │    │  SSH Manager │◄──►│   Ollama / LLM   │   │
│  │  Config Mgmt │    │  Agent Core  │    │   API Provider   │   │
│  └──────────────┘    │  Scheduler   │    └──────────────────┘   │
│                      └──────┬───────┘                            │
│                             │                                    │
│              ┌──────────────┼──────────────┐                     │
│              ▼              ▼              ▼                     │
│     ┌──────────────┐ ┌───────────┐ ┌──────────────┐            │
│     │   Telegram   │ │   E-Mail  │ │  Webhook     │            │
│     │   Bot        │ │   SMTP    │ │  (optional)  │            │
│     └──────────────┘ └───────────┘ └──────────────┘            │
└──────────────────────────────────────────────────────────────────┘
              │              │              │
              ▼              ▼              ▼
┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐
│ PVE     │ │ PVE     │ │ PVE     │ │ PBS     │ │ PVE     │
│ Node 1  │ │ Node 2  │ │ Node 3  │ │ Server  │ │ Standalone│
│(Cluster)│ │(Cluster)│ │(Cluster)│ │         │ │(kein CL) │
└─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘
```

---

## 4. Modulübersicht

### 4.1 Node & Cluster Management

```
Module: node-manager
├── Node Discovery & Onboarding
│   ├── Automatische Proxmox API-Anbindung (Token-Erstellung)
│   ├── SSH Fingerprint Auto-Accept & Verifizierung
│   ├── SSH Trust Setup zwischen allen Nodes
│   └── Node-Gesundheitscheck bei Erstverbindung
├── Cluster-Überblick
│   ├── Cluster-Status & Quorum
│   ├── Node-Status (Online/Offline/Maintenance)
│   ├── Ressourcenverteilung im Cluster
│   └── Standalone-Nodes separat dargestellt
└── PBS Integration
    ├── Backup-Jobs Übersicht
    ├── Datastore-Status
    ├── Backup-Verifizierungsstatus
    └── GC & Prune Status
```

### 4.2 Konfigurationsbackup & Restore

```
Module: config-backup
├── Backup-Targets
│   ├── /etc/pve/                    # Cluster-Konfiguration
│   ├── /etc/network/interfaces      # Netzwerk
│   ├── /etc/hostname                # Hostname
│   ├── /etc/hosts                   # Hosts
│   ├── /etc/resolv.conf             # DNS
│   ├── /etc/apt/                    # APT Quellen
│   ├── /etc/modprobe.d/             # Kernel Module
│   ├── /etc/sysctl.d/              # Kernel Parameter
│   ├── /etc/cron.d/ + crontab      # Cronjobs
│   ├── /etc/lvm/                    # LVM Konfiguration
│   ├── /etc/zfs/                    # ZFS Pools (wenn vorhanden)
│   ├── /etc/postfix/                # Mail-Konfiguration
│   ├── /etc/ssh/                    # SSH-Konfiguration
│   ├── /etc/fstab                   # Mount-Points
│   ├── /etc/default/grub            # GRUB Konfiguration
│   ├── /var/lib/pve-cluster/        # Cluster-Daten
│   └── Benutzerdefinierte Pfade     # Vom User konfigurierbar
├── Backup-Features
│   ├── Zeitgesteuerte Backups (Cron)
│   ├── Pre/Post-Change Snapshots (Diff-basiert)
│   ├── Versionierung (Git-ähnlich mit Commit-Messages)
│   ├── Kompression & Verschlüsselung
│   └── Retention Policy (X Tage/Versionen behalten)
├── Restore-Features
│   ├── Granulare Datei-Wiederherstellung (einzelne Dateien)
│   ├── Vollständige Node-Wiederherstellung
│   ├── Diff-Ansicht vor Restore (Was ändert sich?)
│   ├── Dry-Run Modus
│   ├── Download als Archiv (.tar.gz)
│   └── Rollback auf bestimmte Version
└── Speicher
    ├── Lokaler Storage (Standard)
    ├── S3-kompatibler Storage (MinIO, AWS)
    └── PBS-Integration (optional)
```

### 4.3 Disaster Recovery

```
Module: disaster-recovery
├── Recovery-Vorbereitung
│   ├── Automatische Node-Profile erstellen
│   │   ├── Hardware-Inventar (CPU, RAM, Disks, NICs)
│   │   ├── Netzwerk-Konfiguration
│   │   ├── Storage-Layout (ZFS Pools, LVM, Ceph)
│   │   ├── Installierte Pakete
│   │   └── Proxmox Version & Patches
│   ├── Runbook-Generator pro Node
│   └── Recovery-Readiness Score
├── Recovery-Workflows
│   ├── Node-Austausch (Hardware-Defekt)
│   │   ├── Schritt-für-Schritt Wizard
│   │   ├── Automatisches Re-Join in Cluster
│   │   ├── Konfiguration wiederherstellen
│   │   ├── VM-Migration zurück auf Node
│   │   └── Verifizierung nach Recovery
│   ├── Cluster-Recovery
│   │   ├── Quorum-Wiederherstellung
│   │   ├── Corosync Re-Konfiguration
│   │   └── HA-Fencing Recovery
│   └── Partial Recovery
│       ├── Einzelne Services wiederherstellen
│       ├── Netzwerk-Konfiguration wiederherstellen
│       └── Storage re-attach
└── Testing
    ├── DR-Test Modus (Simulation ohne echte Änderungen)
    ├── Recovery-Validierung
    └── Regelmäßige DR-Readiness Reports
```

### 4.4 Benachrichtigungs- & Steuerungssystem

```
Module: notification-engine
├── Telegram Bot
│   ├── Benachrichtigungen
│   │   ├── Node Online/Offline
│   │   ├── VM Status-Änderungen
│   │   ├── Backup-Ergebnisse
│   │   ├── Speicher-Warnungen
│   │   ├── Agent-Aktionen & Empfehlungen
│   │   └── Konfigurierbare Schwellwerte
│   ├── Steuerung (Befehle)
│   │   ├── /status – Cluster-Übersicht
│   │   ├── /nodes – Node-Liste
│   │   ├── /vms – VM-Übersicht
│   │   ├── /backup <node> – Backup auslösen
│   │   ├── /restart <vmid> – VM neustarten
│   │   ├── /ask <frage> – KI befragen
│   │   └── Bestätigungs-Dialoge für kritische Aktionen
│   └── Inline-Keyboards für schnelle Aktionen
├── E-Mail (SMTP)
│   ├── Tägliche/Wöchentliche Summary-Reports
│   ├── Kritische Alerts (sofort)
│   ├── DR-Readiness Reports
│   └── HTML-Templates (konfigurierbar)
└── Webhook-System
    ├── Benutzerdefinierte Webhooks
    ├── Slack/Discord Integration (optional)
    └── Custom HTTP Callbacks
```

### 4.5 KI-Agent System

```
Module: ai-agent
├── LLM Backend
│   ├── Ollama (lokal im Netzwerk) – Standard
│   ├── OpenAI API (GPT-4o, GPT-4.1)
│   ├── Anthropic API (Claude Sonnet/Opus)
│   ├── Konfigurierbar pro Aufgabentyp
│   └── Fallback-Chain (lokal → remote)
├── Agent-Fähigkeiten
│   ├── Infrastruktur-Analyse
│   │   ├── Anomalie-Erkennung (ungewöhnliche CPU/RAM/IO Muster)
│   │   ├── Kapazitätsplanung (Wann wird Speicher knapp?)
│   │   ├── Performance-Empfehlungen
│   │   └── Sicherheits-Audit (offene Ports, veraltete Pakete)
│   ├── Autonome Aktionen (mit Genehmigungsstufen)
│   │   ├── Level 0: Nur Beobachten & Berichten
│   │   ├── Level 1: Unkritische Aktionen (Cleanup, Log-Rotation)
│   │   ├── Level 2: Moderate Aktionen (VM-Restart bei Crash)
│   │   ├── Level 3: Kritische Aktionen (Migration, Failover)
│   │   └── Jede Stufe konfigurierbar pro User/Rolle
│   ├── Natürlichsprachliche Steuerung
│   │   ├── "Zeige mir alle VMs mit hoher CPU-Last"
│   │   ├── "Erstelle ein Backup von Node1"
│   │   ├── "Migriere VM 101 auf Node2"
│   │   ├── "Was ist gestern Nacht passiert?"
│   │   └── Tool-Calling für strukturierte Aktionen
│   └── Proaktive Intelligenz
│       ├── Predictive Maintenance (Disk-SMART, Trends)
│       ├── Auto-Balancing Empfehlungen
│       ├── Update-Empfehlungen mit Risikoanalyse
│       └── "Morning Briefing" – tägliche Zusammenfassung
├── Chat-Interface
│   ├── Web-UI Chat (im Dashboard integriert)
│   ├── Telegram-Chat (gleiche Fähigkeiten)
│   ├── Kontext-bewusst (weiß welche Node/VM gerade angeschaut wird)
│   ├── Aktions-Bestätigung vor Ausführung
│   └── Chat-Historie & Audit-Log
└── Tool-System
    ├── get_node_status(node_id)
    ├── get_vm_list(node_id, filters)
    ├── start_vm(node_id, vmid)
    ├── stop_vm(node_id, vmid)
    ├── migrate_vm(vmid, target_node)
    ├── create_backup(node_id, paths)
    ├── restore_config(node_id, backup_id, path)
    ├── get_metrics(node_id, timerange)
    ├── run_ssh_command(node_id, command)
    ├── get_storage_status(node_id)
    ├── get_network_config(node_id)
    └── ... erweiterbar
```

### 4.6 Monitoring & Server-Detailansicht

```
Module: monitoring
├── Dashboard (Übersicht)
│   ├── Cluster Health Score (0-100)
│   ├── Alle Nodes mit Status-Ampel
│   ├── Gesamt-Ressourcenauslastung
│   ├── Aktive Alerts
│   ├── Letzte Agent-Aktionen
│   └── Quick-Actions (Backup, Scan, Report)
├── Server-Detailansicht (Pro Node)
│   ├── Hardware-Info
│   │   ├── CPU (Modell, Kerne, Auslastung, Temperatur)
│   │   ├── RAM (Gesamt, Verwendet, Swap)
│   │   ├── Uptime & Load Average
│   │   └── Kernel & PVE Version
│   ├── Netzwerk
│   │   ├── Interfaces mit leserlichen Namen
│   │   │   ├── eno1 → "Management (1Gbit)"
│   │   │   ├── enp3s0f0 → "VM Bridge (10Gbit)"
│   │   │   ├── vmbr0 → "Bridge: VM Netzwerk"
│   │   │   └── Benutzerdefinierte Aliases
│   │   ├── Traffic-Graphen pro Interface
│   │   ├── Bond/VLAN Status
│   │   └── Firewall-Regeln Übersicht
│   ├── Speicher
│   │   ├── ZFS Pools (Status, Used/Free, Fragmentation)
│   │   ├── LVM Volumes
│   │   ├── Ceph (wenn vorhanden)
│   │   ├── SMART-Daten der Disks
│   │   ├── IOPS & Throughput Graphen
│   │   └── Speicher-Prognose (Wann voll?)
│   ├── VMs & Container
│   │   ├── Alle VMs/CTs mit Live-Status
│   │   ├── Ressourcen pro VM (CPU, RAM, Disk, Net)
│   │   ├── Uptime & Snapshots
│   │   ├── Backup-Status
│   │   └── Quick-Actions (Start/Stop/Restart/Migrate)
│   ├── Tags
│   │   ├── Tag-Übersicht pro Node
│   │   ├── Tag-Sync zwischen Nodes (bidirektional)
│   │   ├── Tag-basierte Filterung
│   │   └── Auto-Tagging Regeln
│   └── Konfigurationsbackup-Status
│       ├── Letztes Backup (Datum, Status)
│       ├── Anzahl gesicherter Dateien
│       ├── Diff seit letztem Backup
│       └── Restore-Button
└── Alerting
    ├── Konfigurierbare Schwellwerte
    ├── Alert-Eskalation (Info → Warning → Critical)
    ├── Alert-Stummschaltung (Maintenance Windows)
    └── Alert-Historie
```

### 4.7 Benutzerverwaltung & Rechte

```
Module: auth
├── Authentifizierung
│   ├── Lokale Benutzer (bcrypt-Passwörter)
│   ├── LDAP/Active Directory (optional)
│   ├── OIDC/OAuth2 (optional, z.B. Authentik, Keycloak)
│   ├── 2FA (TOTP)
│   └── API-Tokens für Automatisierung
├── Rollen & Berechtigungen (RBAC)
│   ├── Admin – Voller Zugriff
│   ├── Operator – Monitoring, Backup, VM-Management
│   ├── Viewer – Nur Lesen
│   ├── Custom Roles – Frei konfigurierbar
│   └── Node-spezifische Berechtigungen
├── Agent-Berechtigungen
│   ├── Autonomie-Level pro User konfigurierbar
│   ├── Welche Agent-Aktionen erlaubt sind
│   └── Genehmigungspflicht pro Aktionstyp
└── Audit-Log
    ├── Alle Aktionen protokolliert
    ├── Wer hat was wann gemacht
    ├── Exportierbar (CSV, JSON)
    └── Compliance-tauglich
```

### 4.8 SSH Trust Management

```
Module: ssh-trust
├── Automatisches Onboarding
│   ├── Bei Node-Anbindung: SSH Key Exchange
│   ├── Fingerprint-Verifizierung (Auto-Accept mit Warnung oder manuell)
│   ├── Prometheus-eigenes SSH Keypair pro Installation
│   └── Optional: Eigenen SSH Key hinterlegen
├── Inter-Node Trust
│   ├── SSH Trust zwischen ALLEN angebundenen Nodes
│   ├── Gleichzeitiges Ausrollen auf alle Nodes
│   ├── Key-Rotation (periodisch, automatisierbar)
│   └── Trust-Widerruf bei Node-Entfernung
└── SSH Session Management
    ├── Persistent Connection Pool
    ├── Auto-Reconnect bei Verbindungsverlust
    ├── Parallel Command Execution (Fan-out)
    └── Command-Audit-Log
```

---

## 5. Innovative Features (Next-Level)

Diese Features heben Prometheus deutlich von bestehenden Tools ab:

### 5.1 Infrastructure Drift Detection
```
- Vergleicht den IST-Zustand mit dem gesicherten SOLL-Zustand
- Erkennt unerwartete Änderungen an Konfigurationsdateien
- Alert: "Jemand hat /etc/network/interfaces auf Node2 geändert,
  aber kein Backup erstellt"
- Automatischer Diff + Vorschlag: Backup oder Rollback?
```

### 5.2 Predictive Maintenance Engine
```
- Analysiert SMART-Daten, Temperatur-Trends, Error-Logs
- ML-basierte Vorhersage: "Disk /dev/sdb auf Node3 wird
  voraussichtlich in 14 Tagen ausfallen"
- Automatische Empfehlung: VM-Migration + Disk-Ersatz Runbook
- Speicher-Prognose: "Bei aktuellem Wachstum ist Pool 'data'
  in 45 Tagen voll"
```

### 5.3 Intelligentes Runbook-System
```
- Automatisch generierte Runbooks pro Szenario
  (Node-Ausfall, Disk-Defekt, Netzwerk-Problem)
- KI-gestützte Ausführung: Agent führt Runbook Schritt
  für Schritt aus (mit Bestätigung)
- Community-Runbooks (Import/Export)
- Post-Incident Analyse: "Was ist passiert? Was wurde getan?"
```

### 5.4 Live Cluster Topology Map
```
- Interaktive Netzwerk-/Cluster-Karte (React Flow)
- Nodes, VMs, Storage, Netzwerk-Verbindungen visuell
- Echtzeit-Status (Farben: grün/gelb/rot)
- Drag & Drop VM-Migration (visuell)
- Cluster vs. Standalone klar getrennt
```

### 5.5 Konfigurationsvergleich (Config Diff)
```
- Vergleiche Konfiguration zwischen Nodes
  ("Warum hat Node1 andere Netzwerkeinstellungen als Node2?")
- Vergleiche mit einem bestimmten Zeitpunkt
  ("Was hat sich seit letztem Dienstag geändert?")
- Template-System: "So SOLL ein Standard-Node aussehen"
- Compliance-Check gegen Templates
```

### 5.6 Morning Briefing / Daily Digest
```
- KI-generierte tägliche Zusammenfassung (per Telegram/E-Mail)
- "Guten Morgen! Hier ist der Status:
  ✓ Alle 5 Nodes online
  ⚠ Storage 'local-zfs' auf Node2 bei 82%
  ✓ 3 Backups erfolgreich, 0 fehlgeschlagen
  💡 Empfehlung: VM 104 verbraucht seit 3 Tagen
     nur 2% CPU – kandidat für Downsizing?"
```

### 5.7 Update & Patch Intelligence
```
- Überwacht verfügbare Proxmox Updates auf allen Nodes
- Risikoanalyse pro Update (basierend auf Changelogs + Community)
- Rolling Update Strategie: Node für Node mit HA-Checks
- Rollback-Plan vor jedem Update erstellen
```

### 5.8 Resource Right-Sizing
```
- Analysiert tatsächliche Ressourcennutzung vs. zugewiesene
- "VM 101 hat 8GB RAM zugewiesen, nutzt durchschnittlich 1.2GB"
- Empfehlung zum Down-/Upsizing
- Auswirkungsanalyse: "Wenn du VM 101 auf 2GB reduzierst,
  gewinnst du 6GB für andere VMs"
```

### 5.9 Multi-Environment Dashboard
```
- Verwalte mehrere Standorte/Umgebungen
  (Produktion, Staging, Homelab)
- Umgebungs-Tags und -Grouping
- Cross-Environment Vergleiche
- Failover zwischen Standorten dokumentieren
```

### 5.10 API-Gateway & Unified CLI
```
- Einheitliche REST API über alle Proxmox-Nodes
- CLI Tool (prometheus-cli) für Automatisierung
  $ prometheus nodes list
  $ prometheus backup create node1
  $ prometheus vm migrate 101 --to node2
  $ prometheus agent ask "Was läuft auf Node3?"
```

---

## 6. Datenmodell (Kern-Entities)

```sql
-- Nodes (PVE & PBS)
nodes
├── id (UUID)
├── name (string)
├── type (enum: pve, pbs)
├── hostname (string)
├── ip_address (string)
├── port (int, default: 8006)
├── api_token_id (encrypted)
├── api_token_secret (encrypted)
├── ssh_fingerprint (string)
├── ssh_port (int, default: 22)
├── cluster_name (string, nullable)
├── is_online (bool)
├── last_seen (timestamp)
├── environment (string: production, staging, lab)
└── metadata (jsonb)

-- Konfigurationsbackups
config_backups
├── id (UUID)
├── node_id (FK → nodes)
├── version (int, auto-increment pro node)
├── backup_type (enum: scheduled, manual, pre-change)
├── file_manifest (jsonb: [{path, hash, size, permissions}])
├── storage_path (string)
├── size_bytes (bigint)
├── created_at (timestamp)
├── retention_until (timestamp)
└── notes (text)

-- Backup-Dateien (granulare Wiederherstellung)
config_backup_files
├── id (UUID)
├── backup_id (FK → config_backups)
├── file_path (string)
├── file_hash (string, SHA256)
├── file_size (bigint)
├── file_permissions (string)
├── file_owner (string)
├── content (bytea oder Verweis auf Storage)
└── diff_from_previous (text, nullable)

-- Agent-Aktionen
agent_actions
├── id (UUID)
├── action_type (string)
├── target_node_id (FK → nodes, nullable)
├── target_vmid (int, nullable)
├── description (text)
├── autonomy_level (int: 0-3)
├── status (enum: pending, approved, executing, completed, failed, rejected)
├── approved_by (FK → users, nullable)
├── result (jsonb)
├── llm_reasoning (text)
├── created_at (timestamp)
└── executed_at (timestamp)

-- Benutzer
users
├── id (UUID)
├── username (string, unique)
├── email (string)
├── password_hash (string)
├── role_id (FK → roles)
├── totp_secret (encrypted, nullable)
├── telegram_chat_id (bigint, nullable)
├── max_autonomy_level (int: 0-3)
├── is_active (bool)
└── last_login (timestamp)

-- Rollen
roles
├── id (UUID)
├── name (string)
├── permissions (jsonb: {nodes: [read,write], vms: [read,start,stop], ...})
└── is_system (bool)

-- Alerts
alerts
├── id (UUID)
├── node_id (FK → nodes)
├── severity (enum: info, warning, critical)
├── category (string: cpu, memory, disk, network, backup, agent)
├── message (text)
├── acknowledged (bool)
├── acknowledged_by (FK → users, nullable)
├── created_at (timestamp)
└── resolved_at (timestamp)

-- Netzwerk-Interface Aliases
network_aliases
├── id (UUID)
├── node_id (FK → nodes)
├── interface_name (string: eno1, vmbr0)
├── display_name (string: "Management 1Gbit")
├── description (text)
└── color (string, hex)

-- SSH Keys
ssh_keys
├── id (UUID)
├── public_key (text)
├── private_key (encrypted)
├── created_at (timestamp)
├── rotated_at (timestamp)
└── is_active (bool)

-- Tag Sync
tag_definitions
├── id (UUID)
├── name (string)
├── color (string)
├── category (string)
├── auto_apply_rules (jsonb)
└── sync_enabled (bool)
```

---

## 7. API-Design (Beispiele)

```
REST API (v1):

# Nodes
GET    /api/v1/nodes                     # Alle Nodes
POST   /api/v1/nodes                     # Node hinzufügen (Auto-Setup)
GET    /api/v1/nodes/:id                  # Node-Details
DELETE /api/v1/nodes/:id                  # Node entfernen
GET    /api/v1/nodes/:id/status           # Live-Status
GET    /api/v1/nodes/:id/metrics          # Metriken (Zeitbereich)
GET    /api/v1/nodes/:id/vms              # VMs auf Node
GET    /api/v1/nodes/:id/network          # Netzwerk-Interfaces
GET    /api/v1/nodes/:id/storage          # Storage-Status
GET    /api/v1/nodes/:id/tags             # Tags
POST   /api/v1/nodes/:id/tags/sync        # Tags synchronisieren

# Konfigurationsbackup
POST   /api/v1/nodes/:id/backup           # Backup erstellen
GET    /api/v1/nodes/:id/backups          # Backup-Liste
GET    /api/v1/backups/:id                # Backup-Details
GET    /api/v1/backups/:id/files          # Dateien im Backup
GET    /api/v1/backups/:id/files/:path    # Einzelne Datei
POST   /api/v1/backups/:id/restore        # Restore (mit Dateiauswahl)
GET    /api/v1/backups/:id/download        # Download als Archiv
GET    /api/v1/backups/:id/diff            # Diff zum Vorgänger

# Disaster Recovery
GET    /api/v1/nodes/:id/dr/profile       # DR-Profil
GET    /api/v1/nodes/:id/dr/readiness     # Readiness-Score
POST   /api/v1/nodes/:id/dr/runbook       # Runbook generieren
POST   /api/v1/dr/simulate                # DR-Simulation

# KI Agent
POST   /api/v1/agent/chat                 # Chat-Nachricht
GET    /api/v1/agent/actions               # Agent-Aktionshistorie
POST   /api/v1/agent/actions/:id/approve   # Aktion genehmigen
GET    /api/v1/agent/briefing              # Tagesbriefing

# WebSocket
WS     /api/v1/ws/metrics                 # Live-Metriken Stream
WS     /api/v1/ws/alerts                  # Live-Alerts
WS     /api/v1/ws/agent                   # Agent Chat Stream

# Auth
POST   /api/v1/auth/login                 # Login
POST   /api/v1/auth/logout                # Logout
GET    /api/v1/auth/me                    # Aktueller User
POST   /api/v1/auth/2fa/setup             # 2FA einrichten
```

---

## 8. Frontend-Seitenstruktur

```
/                              → Dashboard (Übersicht aller Nodes/Cluster)
/nodes                         → Node-Liste (Tabelle + Karten-Ansicht)
/nodes/:id                     → Server-Detailansicht
/nodes/:id/vms                 → VMs & Container
/nodes/:id/network             → Netzwerk (mit Aliases)
/nodes/:id/storage             → Speicher & Disks
/nodes/:id/backups             → Konfigurationsbackups
/nodes/:id/monitoring          → Detaillierte Metriken
/topology                      → Cluster Topologie Map
/backups                       → Globale Backup-Übersicht
/disaster-recovery             → DR Dashboard & Runbooks
/agent                         → KI Chat (Vollbild)
/agent/actions                 → Agent-Aktionshistorie
/settings                      → App-Einstellungen
/settings/nodes                → Node-Verwaltung
/settings/users                → Benutzerverwaltung
/settings/roles                → Rollen & Berechtigungen
/settings/notifications        → Telegram/E-Mail Konfiguration
/settings/agent                → Agent-Konfiguration (LLM, Autonomie)
/settings/backup-policies      → Backup-Richtlinien
```

---

## 9. Projektstruktur

```
prometheus/
├── backend/                        # Go Backend
│   ├── cmd/
│   │   └── server/
│   │       └── main.go             # Entrypoint
│   ├── internal/
│   │   ├── api/                    # HTTP Handler & Routes
│   │   │   ├── handler/
│   │   │   ├── middleware/
│   │   │   └── router.go
│   │   ├── config/                 # App-Konfiguration
│   │   ├── model/                  # Datenmodelle
│   │   ├── repository/             # Datenbank-Zugriff
│   │   ├── service/                # Business-Logik
│   │   │   ├── node/               # Node Management
│   │   │   ├── backup/             # Config Backup
│   │   │   ├── recovery/           # Disaster Recovery
│   │   │   ├── monitor/            # Monitoring
│   │   │   ├── agent/              # KI Agent
│   │   │   ├── ssh/                # SSH Management
│   │   │   ├── notification/       # Telegram/E-Mail
│   │   │   └── auth/               # Authentifizierung
│   │   ├── proxmox/                # Proxmox API Client Wrapper
│   │   ├── llm/                    # LLM Abstraction Layer
│   │   │   ├── ollama.go
│   │   │   ├── openai.go
│   │   │   ├── anthropic.go
│   │   │   └── provider.go         # Interface
│   │   └── scheduler/              # Background Jobs
│   ├── migrations/                 # SQL Migrations
│   ├── go.mod
│   └── go.sum
├── frontend/                       # Next.js Frontend
│   ├── src/
│   │   ├── app/                    # App Router Pages
│   │   ├── components/             # React Komponenten
│   │   │   ├── ui/                 # shadcn/ui Basis
│   │   │   ├── dashboard/
│   │   │   ├── nodes/
│   │   │   ├── monitoring/
│   │   │   ├── agent/
│   │   │   ├── topology/
│   │   │   └── backup/
│   │   ├── hooks/                  # Custom Hooks
│   │   ├── lib/                    # Utilities
│   │   ├── stores/                 # State Management (Zustand)
│   │   └── types/                  # TypeScript Types
│   ├── public/
│   ├── next.config.ts
│   ├── tailwind.config.ts
│   └── package.json
├── docker-compose.yml              # Deployment
├── Dockerfile.backend
├── Dockerfile.frontend
├── .env.example
├── ARCHITECTURE.md                 # Dieses Dokument
└── README.md
```

---

## 10. Entwicklungsphasen

### Phase 1: Foundation (MVP)
```
Ziel: Grundgerüst lauffähig
├── [ ] Go Backend Setup (Echo, PostgreSQL, Redis)
├── [ ] Next.js Frontend Setup (shadcn/ui, TailwindCSS)
├── [ ] Authentifizierung (Login, JWT, Rollen Basis)
├── [ ] Node-Anbindung (Proxmox API + SSH Auto-Setup)
├── [ ] Basis-Dashboard (Node-Liste, Status-Anzeige)
├── [ ] Server-Detailansicht (CPU, RAM, Uptime, VMs)
└── [ ] Docker Compose Deployment
```

### Phase 2: Config Backup & Monitoring
```
Ziel: Kernfunktionalität
├── [ ] Konfigurationsbackup-Engine (SSH-basiert)
├── [ ] Granulare Wiederherstellung + Download
├── [ ] Backup-Scheduling (Cron)
├── [ ] Erweitertes Monitoring (Metriken-Graphen, Live-Updates)
├── [ ] Netzwerk-Interfaces mit Aliases
├── [ ] Speicher-Überwachung (ZFS, LVM, SMART)
├── [ ] Tag-System mit Cross-Node Sync
└── [ ] PBS Integration
```

### Phase 3: Disaster Recovery
```
Ziel: Ausfallsicherheit
├── [ ] Node-Profile (automatisches Hardware-Inventar)
├── [ ] DR Readiness Score
├── [ ] Recovery Wizards
├── [ ] Runbook-Generator
├── [ ] Cluster-Recovery Workflows
└── [ ] DR-Test/Simulations Modus
```

### Phase 4: KI Agent
```
Ziel: Intelligenz
├── [ ] LLM Abstraction Layer (Ollama + APIs)
├── [ ] Chat-Interface (Web-UI)
├── [ ] Tool-Calling System (Agent → Proxmox Aktionen)
├── [ ] Autonomie-Level System
├── [ ] Anomalie-Erkennung
├── [ ] Predictive Maintenance (Basis)
└── [ ] Morning Briefing Generator
```

### Phase 5: Kommunikation & Steuerung
```
Ziel: Externe Steuerung
├── [ ] Telegram Bot (Benachrichtigungen + Befehle)
├── [ ] E-Mail Notifications (Templates)
├── [ ] Telegram-Chat mit KI-Agent
├── [ ] Webhook-System
└── [ ] Alert-Eskalation
```

### Phase 6: Advanced Features
```
Ziel: Next-Level
├── [ ] Infrastructure Drift Detection
├── [ ] Cluster Topology Map (React Flow)
├── [ ] Config Diff (Node vs. Node, Zeitvergleich)
├── [ ] Resource Right-Sizing
├── [ ] Update Intelligence
├── [ ] Multi-Environment Support
├── [ ] SSH Trust Management (Key Rotation)
├── [ ] RBAC Erweiterung (Custom Roles, LDAP)
├── [ ] Unified CLI (prometheus-cli)
└── [ ] API-Gateway
```

---

## 11. Sicherheitskonzept

```
├── Verschlüsselung
│   ├── API-Tokens & SSH-Keys: AES-256-GCM verschlüsselt in DB
│   ├── TLS für alle Verbindungen (Backend ↔ Frontend, API ↔ Nodes)
│   ├── Backup-Verschlüsselung (optional, per Backup Policy)
│   └── Redis: Passwort-geschützt + TLS
├── Authentifizierung
│   ├── bcrypt für Passwörter (Cost Factor 12)
│   ├── JWT mit kurzer Laufzeit + Refresh Tokens
│   ├── Rate Limiting (Login, API)
│   └── 2FA (TOTP) für Admin-Accounts
├── Autorisierung
│   ├── RBAC auf API-Ebene
│   ├── Node-spezifische Berechtigungen
│   ├── Agent-Aktionen nur mit passendem Autonomie-Level
│   └── Kritische Aktionen erfordern 2FA-Bestätigung
└── Audit
    ├── Alle Aktionen geloggt (Wer, Was, Wann, Wo)
    ├── Agent-Entscheidungen mit LLM-Reasoning geloggt
    ├── Konfigurationsänderungen versioniert
    └── Login-Versuche protokolliert
```

---

## 12. Technische Entscheidungsbegründungen

| Entscheidung | Begründung | Alternativen betrachtet |
|---|---|---|
| **Go** statt Python | Single Binary, Concurrency, Performance, SSH-Support | Python (FastAPI) – besser für ML, aber mehr Overhead |
| **Echo** statt Fiber/Gin | Enterprise-ready, bester WebSocket-Support, HTTP/2 | Fiber (zu jung), Gin (WebSocket weniger integriert) |
| **Next.js** statt SvelteKit | Größtes Ökosystem, App Router ideal für Dashboards | SvelteKit (kleiner Bundle, aber kleineres Ökosystem) |
| **PostgreSQL** statt SQLite | JSONB Support, Concurrent Access, Skalierbar | SQLite (einfacher, aber Single-Writer Limitation) |
| **Redis** statt NATS | Pub/Sub + Cache + Queue in einem, weit verbreitet | NATS (überdimensioniert für diesen Use-Case) |
| **shadcn/ui** statt MUI | Tailwind-native, volle Kontrolle, kein Runtime-Overhead | MUI (schwer, opiniated), Ant Design (ähnlich) |
| **Zustand** statt Redux | Minimal, kein Boilerplate, ideal für mittlere Apps | Redux (zu viel Overhead), Jotai (zu atomar) |
| **Recharts** statt Chart.js | React-native Composability, gute Docs | Chart.js (imperative API), uPlot (zu low-level) |

---

*Erstellt: 04.03.2026*
*Status: Planungsphase*
