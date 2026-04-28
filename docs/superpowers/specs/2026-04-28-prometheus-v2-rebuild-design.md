# Prometheus V2 — Rebuild Design

Datum: 2026-04-28

## Ziel

Prometheus V1 (Vita) hat in vielen Domaenen Symptome von "halb-fertig": Metriken kommen unvollstaendig durch, Netzwerk- und Port-Scans funktionieren nicht zuverlaessig, VM-Migration und Config-Backups sind unuebersichtlich und fehleranfaellig, der Agent kann zwar chatten, aber nichts in der Anwendung steuern. Stack-Wechsel ist nicht das primaere Problem; struktureller Wildwuchs ueber 29 Services und ~50 Repositories ist es.

Prometheus V2 ist deshalb ein **kuratiertes Rebuild** mit klarem Scope:

- weniger Funktionen, aber jede end-to-end fertig
- robuste Architektur fuer Multi-User-Betrieb, viele Hosts und viele parallel laufende Tasks
- Domain-orientierte Code-Struktur, die ein Mensch im Kopf halten kann
- Frontend-Stack-Modernisierung dort, wo sie messbar etwas bringt; Backend-Sprache bleibt
- Vorbereitung fuer spaetere Erweiterung auf generische Linux-Hosts ueber Capability-Adapter
- Agent als echter IT-Admin-Beisteller, nicht nur Empfehlungsmaschine

V2 ersetzt Vita parallel und gestaffelt. Vita bleibt bis zur stabilen V2-Abdeckung funktionsfaehig.

## Identitaet

Prometheus ist und bleibt ein **Operations-Cockpit fuer Proxmox-Cluster**, nicht ein Ersatz fuer das Proxmox-WebGUI. Die Rolle aendert sich aber: V2 wird die **alleinige Oberflaeche fuer Nicht-Admin-Nutzer**, damit Operatoren und Viewer fuer den Tagesbetrieb nicht mehr ins Proxmox-WebGUI muessen. Proxmox-WebGUI bleibt damit konsequent in Admin-Hand fuer tiefe Konfiguration (ISOs, Storage, Netzwerk, Cluster, Hardware).

Bedien-Modell: **UI-first Cockpit mit Agent als gleichberechtigtem zweiten Bedienweg**. Routine geht via UI, Mehrschrittiges und Empfehlungen gehen via Agent.

## Leitprinzipien

1. **Eine Domaene = ein Paket = ein Mensch im Kopf.** Domain-Module enthalten Handler, Service, Repo, Types, Jobs und Events fuer ihre Funktion in einem Verzeichnis. Keine Verstreuung ueber `handler/`, `service/`, `repository/`.
2. **Cross-Domain nur ueber veroeffentlichte Interfaces.** `domain/X/api.go` exportiert genau das, was andere Domains nutzen duerfen. Direkte Repo-Zugriffe quer durchs Backend sind verboten.
3. **Generierung statt Handarbeit fuer Vertraege.** SQL → Go via sqlc, OpenAPI → TS via openapi-typescript. Schema-Drift wird zum Compile-Error.
4. **Fehler sind sichtbar oder werden geworfen.** Kein stilles `catch`, kein `_ = err` ohne Begruendung. Lint erzwingt das.
5. **Kein nice-to-have-Code im V2.** Drift, Anomaly, Predictions, Brain, Logs, Security-Engine, Recovery bleiben in V1. Wenn etwas zurueckkommt, dann sauber neu als Domain-Modul.
6. **Single-Binary-Deploy.** Frontend wird in das Go-Binary eingebettet. Eine ausfuehrbare Datei plus Postgres und Redis, sonst nichts.
7. **Multi-User ist Pflicht-Fundament, nicht Aufsatz.** RBAC, Approval, Audit, Optimistic Concurrency, Distributed Locks sind in V2 von Anfang an vorhanden.
8. **Capability-Adapter-Pattern ist gebaut, auch wenn nur Proxmox implementiert ist.** Linux-General-Server bleibt Phase 3, der Code-Pfad ist aber offen.

## Stack-Entscheidungen

| Bereich | Wahl | Begruendung |
|---|---|---|
| Backend-Sprache | Go 1.23+ | Goroutines/Channels fuer Scheduler und SSH-Pools, reife Proxmox-Client-Bibliotheken |
| Web-Framework | Echo v4 | bleibt, solide, kein Wechselgrund |
| DB-Layer | sqlc + pgx | typisierte Queries gegen Schema-Drift |
| Job-Queue | River (Postgres-backed) | transactional enqueue, periodic Jobs, Retries, mehrere Queues |
| Distributed Locks | redislock | Resource-Exklusivitaet ueber Goroutine-Grenzen hinweg |
| API-Stil | REST + OpenAPI 3 (Spec-First) | menschenlesbar, generierbar |
| OpenAPI-Backend-Codegen | oapi-codegen | Server-Stubs aus Spec |
| OpenAPI-Frontend-Codegen | openapi-typescript + openapi-fetch | Types aus Spec, kleiner typisierter Fetch |
| Realtime (Metriken) | WebSocket mit Subscription-Filter | hochfrequent, selektiv pro Client |
| Realtime (Tasks/Notifications) | Server-Sent Events | leicht, eingebauter Reconnect, multiplexbar ueber HTTP/2 |
| Auth | JWT (Access 15min) + HttpOnly-Refresh-Cookie + API-Keys (zwei Klassen) | bleibt nahe V1, mit klarer Service-Identity-Erweiterung |
| Frontend-Bundler | Vite + SWC | schneller Dev-Server, kleines Build-Setup |
| UI-Framework | React 19 + TypeScript | Oekosystem, vorhandenes shadcn-Investment |
| Routing | TanStack Router | typsichere Routes und Search-Params |
| Server-State | TanStack Query | Cache, Invalidation, Devtools, SSR-frei |
| Client-State | Zustand | bleibt schlank fuer UI-Modi |
| Forms + Validierung | React Hook Form + Zod | Standard, typsicher |
| UI-Komponenten | shadcn/Radix | Investment behalten |
| Charts | Recharts | bleibt fuer Admin-Mengen ausreichend |
| Deploy | go:embed in Binary | eine ausfuehrbare Datei |
| Logging | log/slog (JSON-Handler) | strukturiert, stdlib |

## Repo-Layout

```
Prometheus/                                # bestehender Git-Root
├── Prometheus-Vita/                       # bleibt waehrend Cutover
├── Prometheus-V2/                         # NEU
│   ├── backend/
│   │   ├── cmd/prometheus/main.go
│   │   ├── internal/
│   │   │   ├── domain/
│   │   │   │   ├── auth/
│   │   │   │   ├── host/
│   │   │   │   │   ├── core.go            # Host-Typ, Capability-Registry
│   │   │   │   │   └── adapter/proxmox/
│   │   │   │   ├── vm/                    # Lifecycle, Tags, Resize
│   │   │   │   ├── template/              # VM-Templates
│   │   │   │   ├── console/               # noVNC-Proxy + xterm.js-SSH-Bridge
│   │   │   │   ├── migration/
│   │   │   │   ├── backup/
│   │   │   │   ├── netscan/
│   │   │   │   ├── notification/
│   │   │   │   ├── agent/                 # LLM + Tool-Registry
│   │   │   │   ├── task/
│   │   │   │   ├── approval/
│   │   │   │   └── audit/
│   │   │   ├── platform/
│   │   │   │   ├── proxmox/               # Proxmox-API-Client
│   │   │   │   ├── ssh/                   # Per-Host-Pools
│   │   │   │   ├── crypto/                # AES-256-GCM
│   │   │   │   ├── jobs/                  # River-Wiring
│   │   │   │   ├── events/                # Redis-Pub/Sub-Bus
│   │   │   │   ├── locks/                 # redislock-Wrapper
│   │   │   │   └── llm/                   # Provider (Ollama, OpenAI, Anthropic)
│   │   │   ├── http/                      # Server, Middleware, OpenAPI-Wiring
│   │   │   └── web/                       # embed.FS fuer SPA-Asset
│   │   ├── db/
│   │   │   ├── migrations/                # SQL-Migrations frisch ab 001
│   │   │   └── queries/                   # sqlc-Quellen
│   │   ├── api/openapi.yaml               # Single Source of Truth
│   │   ├── sqlc.yaml
│   │   └── Makefile
│   └── frontend/
│       ├── src/
│       │   ├── routes/                    # TanStack Router file-based
│       │   ├── domains/                   # Spiegel der Backend-Domains
│       │   ├── components/                # shadcn + eigene
│       │   ├── lib/api/                   # generierter Client
│       │   └── stores/                    # Zustand
│       ├── index.html
│       ├── vite.config.ts
│       └── tsconfig.json
├── docker-compose.yml                     # Postgres + Redis + V1 + V2 (waehrend Cutover)
└── docs/superpowers/                      # Specs und Plaene
```

## Top-Level-Datenfluss

```
Browser (SPA)  ──REST/JWT──▶  Go Backend  ──pgx/sqlc──▶  Postgres (Schema prom_v2)
     ▲                            │
     │                            ├──▶ Redis (Cache, Pub/Sub, Locks, Rate-Limit)
     │                            ├──▶ Proxmox-API
     ├◀──WS (Metriken)            ├──▶ SSH-Pool (Per-Host)
     └◀──SSE (Tasks/Notif)        └──▶ LLM-Provider
```

## Domain-Modul-Struktur (Innenleben)

Jedes Domain-Modul folgt der gleichen Anatomie:

```
internal/domain/<name>/
├── api.go              # public Interface fuer andere Domains
├── types.go            # Domain-Types
├── service.go          # Business-Logik
├── http.go             # Echo-Handler, OpenAPI-Operations
├── jobs.go             # River-Job-Definitionen und -Worker
├── events.go           # Event-Publish (SSE)
├── repo/               # sqlc-generierter Code
└── service_test.go     # Tests
```

Faustregeln:

- HTTP-Handler sind duenn: parsen, RBAC, an Service delegieren, antworten. Keine Business-Logik.
- Service ist der einzige Ort fuer Domain-Wissen. Er ruft Repo, Adapter, Job-Queue, Locks, Audit, Events.
- Repo enthaelt nur Datenzugriff und ist sqlc-generiert.
- Cross-Domain-Aufrufe gehen ueber `api.go`. Beispiel: `task` injiziert `migration.Reader` und ruft `reader.GetMigration(ctx, id)`.
- Wenn ein Datei-Block ueber 300 Zeilen waechst, splitten. Wenn eine Domaene mehr als acht Files braucht, pruefen, ob sie zwei sein sollte.

### Beispiel: VM-Migration end-to-end

```
POST /api/v1/migrations  (Operator User A)
    │
    ▼
http.go: parsen, validieren
    │
    ▼
service.StartMigration(ctx, req):
  1. authz.Check(user, "migration:start")
  2. lock.Acquire("vm:" + vmID, ttl=30min)
  3. concurrency.Check(vm.Version)
  4. host.GetCapabilities(targetHostID)
  5. approval.RequestIfRequired(user, "migration.start", payload)
  6. repo.CreateMigration(...)
  7. river.Insert(RunMigrationJob{migrationID})
  8. audit.Record("migration.requested", ...)
  9. events.Publish("migration.queued")
  10. return 202 Accepted

[asynchron im River-Worker]
RunMigrationJob:
  - host-Adapter.Migrate(...) ueber Proxmox-API + ggf. SSH
  - regelmaessig events.Publish("migration.progress")
  - Status-Maschine: queued → running → done | failed | cancelled
  - audit + event bei jedem Statusuebergang
```

## API-Vertrag und Datenfluss

OpenAPI-Spec ist handgepflegt unter `backend/api/openapi.yaml` und ist Vertrag, nicht Doku. Aus ihr werden Backend-Server-Stubs (oapi-codegen) und Frontend-Types (openapi-typescript) generiert. Spec-Aenderung → `make generate` → Compile-Errors zeigen jeden betroffenen Aufruf.

### REST-Konventionen

```
GET    /api/v1/{resource}              Liste (cursored)
POST   /api/v1/{resource}              Erstellen
GET    /api/v1/{resource}/{id}         Detail
PATCH  /api/v1/{resource}/{id}         Teil-Update mit If-Match-Header
DELETE /api/v1/{resource}/{id}
POST   /api/v1/{resource}/{id}/{verb}  Aktion (z.B. /migrations/{id}/cancel)
```

### Antwort-Envelope

Erfolg:

```json
{ "data": { ... }, "meta": { "request_id": "...", "version": 7 } }
```

Fehler:

```json
{ "error": { "code": "MIGRATION_VM_LOCKED", "message": "...", "details": { ... }, "request_id": "..." } }
```

Error-Codes sind typisiert. Backend exportiert die Liste, Frontend bezieht sie als TS-Union, kann gezielt darauf reagieren ohne Message-Parsing.

### Pagination

Cursor-basiert. `?limit=50&cursor=...`. Cursor ist Base64 von `{id, sort_value}`. Stabil bei parallelen Inserts, performant bei vielen Tasks ueber Zeit.

### IDs

UUIDv7 ueberall. Sortierbar nach Erstellungszeit, gut fuer Cursor-Pagination und Index-Locality.

### Optimistic Concurrency

PATCH-Requests senden `If-Match: <version>`. Backend prueft, inkrementiert bei Match, antwortet `409 Conflict` bei Mismatch.

### Idempotency

Side-Effect-POSTs akzeptieren `Idempotency-Key`-Header. Backend speichert Key + Response 24h. Wiederholter Call mit gleichem Key + Body liefert die gespeicherte Response, kein Doppel-Side-Effect.

### Versionierung

`/api/v1/` ist Pflicht-Prefix. Breaking Changes erzeugen `/api/v2/` parallel. V2-Pfad bleibt waehrend des Rebuilds frei.

### Auth-Flow

- Login: `POST /api/v1/auth/login` → Access-Token (JWT, 15min) im Body, Refresh-Token im HttpOnly-Cookie.
- Authentifizierte Calls: `Authorization: Bearer <jwt>`.
- Refresh: `POST /api/v1/auth/refresh` (Cookie wird mitgeschickt) → neues Access-Token. Frontend laeuft das transparent vor 401.
- API-Keys: `X-API-Key`-Header, Middleware vor JWT, eigene RBAC-Subjekte.

## Realtime-Strategie

### WebSocket fuer Live-Metriken

Endpunkt: `GET /api/v1/ws/metrics?token=<jwt>`. Subscription-Filter vom Client:

```json
→ { "op": "subscribe",   "hosts": ["uuid1", "uuid2"] }
← { "type": "metrics",   "host": "uuid1", "ts": "...", "cpu": 0.42, ... }
→ { "op": "unsubscribe", "hosts": ["uuid2"] }
```

Backend: ein `MetricsHub` ueber Redis-Pub/Sub. River-Periodic-Jobs sammeln Metriken pro Host und publishen auf Channel `metrics:{hostID}`. WS-Connection-Handler subscribet selektiv. Backpressure: Send-Buffer pro Connection ist begrenzt, Ueberlauf droppt aelteste Messages und inkrementiert `metrics.dropped`.

### Server-Sent Events fuer Push

Endpunkt: `GET /api/v1/events?token=<jwt>&topics=tasks,notifications,approvals`. Topics:

- `task.*`, `migration.*`, `backup.*`, `scan.*`
- `notification.delivered`, `notification.failed`
- `approval.requested`, `approval.granted`, `approval.denied`
- `audit.event` (Admin-only)

Topic-Filterung serverseitig anhand RBAC. Resume nach Disconnect via `Last-Event-ID`-Header und Redis-Stream-Buffer (24h Retention).

### Multi-Session-Sync

Frontend mountet einen globalen SSE-Listener im AppShell. Events triggern `queryClient.invalidateQueries(...)` und optionale Toasts/UI-Updates. Kein per-Component-EventSource.

### Authentifizierung beider Kanaele

Token im Query-String, weil Browser keine Header bei WS und SSE-EventSource erlauben. Token-Lebensdauer 15min begrenzt den Schaden bei Leak. Server logged Tokens **niemals**, Reverse-Proxy-Konfig (falls genutzt) muss Query-Strings aus Logs filtern.

### Was nicht ueber Realtime laeuft

- Cluster-Events ohne Sekundenbruchteil-Anspruch: TanStack Query mit `refetchInterval` (30s).
- Logs-Streaming: nicht V2-Scope.
- Agent-Chat: eigener WS- oder Streaming-Endpoint im `agent/`-Domain (Token-Streaming der LLM-Antworten).

## Job-Queue, Scheduling und Concurrency

### River als alleinige Job-Plattform

Ersetzt heutigen `scheduler/`, ad-hoc Background-Goroutines und das halb-implementierte Task-Center. Jede Domaene registriert eigene Jobs in ihrem `jobs.go`. Zentrales `internal/platform/jobs/client.go` macht das Wiring.

### Job-Klassen

**Periodic Jobs:**

```go
{Cron: "*/30 * * * * *", Constructor: HostStatusPollJob{}},
{Cron: "0 */1 * * * *",  Constructor: MetricsCollectJob{}},
{Cron: "0 0 */6 * * *",  Constructor: BackupSweepJob{}},
```

**On-Demand Jobs:** `RunMigrationJob`, `RunBackupJob`, `RunNetworkScanJob`, `SendNotificationJob`, `ExecuteAgentToolJob`, `RunVMCreateFromTemplateJob`, `RunVMDeleteJob`. Werden aus Service-Code transactional eingereiht (DB-Insert + Job-Insert in einer Transaktion).

**Recurring mit Jitter:** Per-Host-Jobs werden um 0–`interval/2` Sekunden randomisiert verschoben, damit 100 Hosts nicht gleichzeitig pollen.

### Worker-Pools

Konfigurierbar pro Job-Kind:

```go
Queues: map[string]river.QueueConfig{
    "host_polls":    {MaxWorkers: 16},
    "migrations":    {MaxWorkers: 4},
    "backups":       {MaxWorkers: 2},
    "scans":         {MaxWorkers: 6},
    "notifications": {MaxWorkers: 8},
    "vm_lifecycle":  {MaxWorkers: 8},
}
```

Jede Queue ist isoliert. Worker-Anzahl ist Tuning-Knopf, kein Code-Change. Kritische Aktionen wie Migrationen haben `MaxAttempts=1`, leichte Reads `MaxAttempts=3`.

### Idempotenz

Jeder Job pruegt am Anfang den Status:

```go
if m.Status == "done" || m.Status == "failed" { return nil }
if m.Status == "running" && time.Since(m.StartedAt) < 30*time.Minute {
    return river.JobSnoozeError{Duration: 1 * time.Minute}
}
```

Status-Maschinen sind explizit, Uebergaenge atomar (`UPDATE ... WHERE status = 'queued'`).

### Distributed Locks

`redislock` schuetzt Ressourcen-Exklusivitaet:

```go
lock, err := locker.Obtain(ctx, "vm:"+vmID, 30*time.Minute, &redislock.Options{
    RetryStrategy: redislock.NoRetry(),
})
if errors.Is(err, redislock.ErrNotObtained) {
    return apierror.Conflict("VM_BUSY", "Diese VM ist gerade in einer anderen Operation.")
}
defer lock.Release(ctx)
```

Lock-Hierarchie zur Deadlock-Vermeidung: `host:` zuerst, dann `vm:`, dann `backup:`/`migration:`. In umgekehrter Reihenfolge freigeben.

### Bulk-Concurrency

`errgroup` mit gebundener Parallelitaet, kein neues Tooling:

```go
g, gctx := errgroup.WithContext(ctx)
g.SetLimit(8)
for _, host := range hosts {
    host := host
    g.Go(func() error { return checkHost(gctx, host) })
}
return g.Wait()
```

### Beobachtbarkeit

- **Task-Center** (im UI) ist die offizielle Sicht fuer alle Nutzer, gefiltert nach RBAC und Ownership: Viewer sehen eigene Tasks und oeffentliche Cluster-Operationen lesend, Operatoren koennen eigene cancellen/retriggern, Admin sieht alles.
- **River-Web-UI** unter `/admin/jobs` ist Admin-Debug-Sicht, nicht Teil des UX-Pfads.
- Per-Domain-Metriken im Prometheus-Format unter `/metrics`.

### Was nicht ueber River laeuft

- Synchrone HTTP-Handler-Logik bleibt synchron. River ist nur fuer Arbeit, die >2s dauert oder asynchron sein muss.
- Realtime-Pub/Sub geht ueber Redis direkt, nicht River.
- Innere Schritte einer Operation (z.B. einzelne Migrations-Phasen) sind Code im Worker, keine eigenen Jobs. Keine Job-Inflation.

## Multi-User: Auth, RBAC, Approval, Audit

### Rollen

Drei Rollen, fein-granulare Permissions, beliebig viele User pro Rolle.

| Rolle | Lesen | Operative Aktionen | Riskante Aktionen | Admin-Aktionen |
|---|---|---|---|---|
| Viewer | sichtbares lesend | – | – | – |
| Operator | alles | Migrationen, Backups, Scans, Notifications testen, eigene VMs verwalten | mit Approval | – |
| Admin | alles inkl. Audit | alles | mit oder ohne Approval (per Policy) | User-, RBAC-, System-Config |

Permissions sind Strings wie `migration:start`, `vm:delete`, `audit:read`, in einer `roles`-Tabelle gebuendelt. JWT enthaelt das Permission-Array, refresh alle 5min.

### Permission-Set (V2-Stand)

```
auth:*                        Login/Logout, Sessions, API-Keys
audit:read                    Audit-Log lesen
host:read | host:write
vm:read | vm:read:owned
vm:lifecycle | vm:lifecycle:any
vm:tag | vm:tag:any
vm:resize | vm:resize:any
vm:snapshot | vm:snapshot:any
vm:console | vm:console:any
vm:create:from_template
vm:create:custom              # Admin only
vm:delete | vm:delete:any
template:read | template:manage
migration:start | migration:cancel | migration:read
backup:create | backup:restore | backup:delete | backup:read
backup:schedule:manage
netscan:run | netscan:read | netscan:baseline:manage
notification:send | notification:read
notification:channel:manage | notification:rule:manage
agent:chat                    Chat-Session starten
                              Tool-Calls erben die Permissions des Callers (User oder Service-Identity)
approval:grant | approval:read
task:read | task:cancel | task:cancel:any
admin:user:manage | admin:rbac:manage | admin:system
```

`*:any`-Suffix erlaubt Aktion auf fremden Ressourcen. Standard-Operator hat ohne `:any`.

### Multi-Admin-Verhalten

- Beliebig viele User koennen Admin sein.
- Approval zwischen Admins ist erlaubt: jeder mit `approval:grant` kann Antraege anderer User genehmigen, eigene nicht (ausser Self-Approval-Policy ist explizit aktiviert).
- Self-Approval (Express-Approve) ist per System-Policy-Schalter steuerbar. Default in Multi-Admin-Setup: aus. Default fuer Solo-Admin (nur ein User mit `approval:grant`): an. Erkennung beim ersten Start.

### API-Keys: zwei Klassen

| | Personal API-Key | Service API-Key |
|---|---|---|
| Bindung | An User-Account | An Service-Identity |
| Permissions | Subset der User-Permissions, scope-narrowable | Eigener Permission-Satz, kann admin-level sein |
| Approval-Aktionen | Blockiert (kein Mensch dahinter) | Konfigurierbar: `bypass`, `auto_approve`, `block` |
| Audit | `actor_kind=api_key`, `via_user=...` | `actor_kind=service`, `service_name=...` |
| Erstellen | User selbst (fuer sich) oder Admin | Nur Admin |
| Widerruf | User oder Admin | Nur Admin |

Service-Identity-Modell:

```sql
CREATE TABLE service_identities (
    id              uuid PRIMARY KEY,
    name            text NOT NULL UNIQUE,
    description     text,
    permissions     text[] NOT NULL,
    approval_mode   text NOT NULL DEFAULT 'block',  -- bypass | auto_approve | block
    enabled         bool NOT NULL DEFAULT true,
    created_by      uuid REFERENCES users,
    created_at      timestamptz NOT NULL DEFAULT now()
);
```

Standard-Service-Identity beim ersten V2-Start: `agent-runtime` mit Permissions fuer alle Agent-Tool-Calls (siehe Agent-Sektion). `approval_mode=auto_approve` mit voller Audit-Spur.

### Approval-Domain

Zentrale Domaene fuer Vier-Augen-Workflows.

```sql
CREATE TABLE approvals (
    id              uuid PRIMARY KEY,
    requester_id    uuid NOT NULL REFERENCES users,
    action_type     text NOT NULL,           -- "migration.start", "vm.delete", "host.tool_install"
    payload         jsonb NOT NULL,
    status          text NOT NULL,           -- pending | granted | denied | expired | cancelled
    approver_id     uuid REFERENCES users,
    decision_note   text,
    expires_at      timestamptz NOT NULL,
    requested_at    timestamptz NOT NULL DEFAULT now(),
    decided_at      timestamptz,
    version         int NOT NULL DEFAULT 1
);
```

Policies werden in `domain/approval/policy.go` definiert pro `action_type`. Default-Verhalten siehe Domain-Sektionen unten.

### Audit-Domain

Pflicht-Sidecar jeder Schreibaktion. Eintrag wird in derselben DB-Transaktion wie die Hauptaktion geschrieben — keine separaten Audit-Pfade.

```sql
CREATE TABLE audit_events (
    id              uuid PRIMARY KEY,                -- UUIDv7
    actor_id        uuid REFERENCES users,
    actor_kind      text NOT NULL,                   -- user | api_key | service | system | agent
    action          text NOT NULL,
    resource_kind   text,
    resource_id     uuid,
    request_id      text,
    payload         jsonb,
    result          text NOT NULL,                   -- success | failure | denied
    error_code      text,
    ip_address      inet,
    user_agent      text,
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ON audit_events (resource_kind, resource_id, created_at);
CREATE INDEX ON audit_events (actor_id, created_at);
CREATE INDEX ON audit_events (action, created_at);
```

Append-only. Keine Update-/Delete-API. Retention: 1 Jahr Default, konfigurierbar; einzelne Action-Typen koennen laenger aufgehoben werden (z.B. `auth.login_failed`, `host.tool_install`: 5 Jahre).

### Sessions

- Refresh-Token in `sessions`-Tabelle, widerrufbar.
- Session-Liste pro User unter `/settings/sessions` (Browser, IP, Last-Seen). User widerruft eigene, Admin kann fremde.
- Forced Logout bei Rolle-Downgrade: alle aktiven Sessions invalidiert.

### Concurrency-Schutz auf drei Schichten

- Optimistic Concurrency (If-Match/Version) gegen "ich ueberschreibe deine gerade gespeicherte Aenderung".
- Distributed Locks (redislock) gegen "wir machen beide gleichzeitig dieselbe schwere Aktion".
- Approval-Queue gegen "ich mache eine riskante Aktion ohne dass jemand draufschaut".

## Host- und VM-Domains im Detail

### Host-Abstraktion mit Capabilities

Ein **Host** ist ein verwaltetes System mit Capabilities. V2 implementiert nur den Proxmox-Adapter, das Modell ist aber offen fuer Linux-General-Server in spaeteren Phasen.

```go
type Capability string

const (
    CapProxmoxCluster Capability = "proxmox-cluster"
    CapSSH            Capability = "ssh"
    CapSystemd        Capability = "systemd"
    CapDocker         Capability = "docker"
    CapAPT            Capability = "apt"
)

type HostAdapter interface {
    Capabilities() []Capability
    ListVMs(ctx context.Context) ([]VM, error)
    ...
}
```

Operationen sind capability-typed. Eine Operation deklariert ihre erforderlichen Capabilities; Hosts ohne sie sehen die Operation nicht in der UI.

### VM-Domain

`domain/vm/` deckt den Lebenszyklus, der Operatoren tagtaeglich braucht, damit sie nicht ins Proxmox-WebGUI muessen:

- Lifecycle: Start, Stop, Reboot, Shutdown (graceful), Reset
- Tags: anlegen, aendern, loeschen (Schluessel-Wert-Paare via Proxmox-Tags)
- VM-Erstellen aus Template (siehe `template/`-Domain)
- VM-Loeschen (immer mit Approval ausser Admin-Bypass)
- VM-Resize: CPU, RAM, Disk-Erweiterung (kein Disk-Schrumpfen, kein Disk-Tausch — bleibt Proxmox-WebGUI)
- VM-Snapshots: erstellen, wiederherstellen, loeschen
- VM-Detail: Hardware-Config (CPU/RAM/Disks read-only mit Resize-Aktion separat), Boot-Reihenfolge anzeigen, Status, Live-Metriken

VM-Ownership ueber Proxmox-Tags: `prom-owner=<user-id>`. Prometheus liest die Tags, nutzt sie fuer RBAC-Filter (`vm:read:owned`), Templates setzen sie automatisch beim Create. Keine eigene Mapping-Tabelle in V2-DB; Proxmox bleibt Source-of-Truth.

### Console-Domain

`domain/console/` stellt den Browser-Zugriff auf VMs bereit, ohne dass User in Proxmox einloggen muessen.

- **Primaer: noVNC-Proxy.** Prometheus-Backend authentifiziert den User (RBAC `vm:console`), holt ein Proxmox-VNC-Ticket, proxied den WebSocket-Stream zum Browser. Keine direkten Credentials oder Browser-Cookies fuer Proxmox.
- **Fallback: xterm.js-SSH-Bridge.** Fuer Linux-VMs mit erreichbarem SSH-Port; nuetzlich fuer Skripting, ungeeignet fuer Boot-Probleme oder Windows.
- Sessions haben TTL (max 30min idle), werden auditiert (`console.opened` mit User, VM, Dauer).
- Admin-Console (`vm:console:any`) erlaubt Zugriff auf jede VM, Operator nur eigene.

### Template-Domain

`domain/template/` verwaltet VM-Vorlagen, aus denen Operatoren VMs erstellen koennen, ohne tiefe Proxmox-Kenntnis.

```sql
CREATE TABLE vm_templates (
    id              uuid PRIMARY KEY,
    name            text NOT NULL,
    description     text,
    os_template_id  text NOT NULL,                       -- Proxmox-Template-Ref oder ISO + cloud-init
    default_cores   int NOT NULL,
    default_memory  int NOT NULL,                        -- MiB
    default_disk    int NOT NULL,                        -- GiB
    default_storage text NOT NULL,
    default_bridge  text NOT NULL,
    default_tags    jsonb,
    cloud_init_user_data text,
    requires_approval bool NOT NULL DEFAULT false,
    allowed_roles   text[] NOT NULL DEFAULT '{operator,admin}',
    enabled         bool NOT NULL DEFAULT true,
    created_by      uuid REFERENCES users,
    version         int NOT NULL DEFAULT 1,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);
```

Operator-Flow: `/vms/create` zeigt Templates gefiltert nach `allowed_roles` und `enabled`. User waehlt, fuellt Name/Hostname und ggf. Override-Werte (innerhalb erlaubtem Range), bestaetigt. Bei `requires_approval=true` Approval-Workflow, sonst direkt River-Job. Live-Fortschritt im Task-Center, fertige VM erscheint mit User als Owner-Tag.

Admin-Flow: `/settings/templates` mit Schema-Editor und Validierung gegen Proxmox-Storages und Bridges des Clusters.

### Approval-Defaults fuer VM-Aktionen

| Aktion | Default-Approval |
|---|---|
| Lifecycle (start/stop/reboot/shutdown) | nein |
| Tag aendern | nein |
| Snapshot erstellen | nein |
| Snapshot wiederherstellen | ja |
| VM-Erstellen aus Template | per Template konfigurierbar |
| VM-Resize (klein, +/-25%) | nein |
| VM-Resize (gross) | ja |
| VM-Loeschen | ja, immer |

## Migration-, Backup-, Netscan-, Notification-Domains

### Migration

VM-Migration ueber Proxmox-API plus SSH wo noetig. River-Job mit `MaxAttempts=1` (kein Auto-Retry bei kritischer Aktion). Lock auf `vm:`. Approval per Policy. Status-Maschine `queued → running → done | failed | cancelled`. Live-Fortschritt via SSE-Events.

### Backup

Zwei Klassen: VM-Backups (Proxmox-PBS oder vzdump) und Config-Backups (per SSH gezogene Konfigurationsdateien einer Host-Liste). Beide haben Schedules, Run-Historie, Restore-Aktion mit Approval-Pflicht. Storage-Pfade pro Backup-Profile konfigurierbar.

### Netscan

Quick-Scan und Full-Scan unterschieden. Quick nutzt `ss` ueber SSH. Full nutzt `nmap` ueber SSH. Tool-Preflight (V1-Funktion) wandert nach V2 als Teil von `domain/host/` und blockiert Full-Scan klar mit Begruendung, wenn `nmap` fehlt. Ergebnisse: Ports, Devices, Services, Anomalien, Baseline-Vergleich. Lock auf `host:scan:` damit pro Host nur ein Scan gleichzeitig laeuft.

### Notification

Channels (Telegram, SMTP) mit Test-Aktion und Status-Anzeige (verbunden, nicht verbunden, Fehler, letzter Test). Alert-Regeln auf Cluster-Events. Eskalations-Stufen mit Zeitabstaenden. Versand laeuft als River-Job, Retries bei transient errors, persistente Fehler landen im Task-Center und triggern UI-Banner.

Beim ersten V2-Start gibt es Default-Alert-Regeln fuer kritische Trigger (Node down, Migration failed, Backup failed), damit das System auch ohne manuelle Konfiguration sinnvoll meldet.

## Agent-Domain

### Architektur

`domain/agent/` enthaelt den LLM-Chat-Layer und die Tool-Registry. LLM-Provider sind `Ollama`, `OpenAI`, `Anthropic`, fest registriert in `internal/platform/llm/`. Provider-Wahl pro User in den Settings.

### Tool-Registry

Jeder Tool-Aufruf ist annotiert:

```go
ToolDef{
    Name:        "vm.create_from_template",
    Permission:  "vm:create:from_template",
    RiskLevel:   "medium",
    Description: "Erstellt eine VM aus einer Vorlage.",
    Schema:      ...,
}
```

Bei jedem Tool-Call:

1. Caller-Identity bestimmt Permissions:
   - Im Chat-Flow erbt der Agent die Permissions des aktiven Users (User-Proxy).
   - Im autonomen Flow nutzt der Agent seine Service-Identity.
2. Permission-Check. Fehlt sie, wird der Tool-Call abgebrochen mit Fehlermeldung an LLM ("permission denied"), das LLM kann reagieren.
3. Bei `RiskLevel=high` und Caller-Mode != `bypass`: Approval-Workflow.
4. Audit-Eintrag (`agent.tool_call`).

### Konkrete Agent-Tools (V2-Standard)

```
host.list, host.get, host.metrics
vm.list, vm.get, vm.metrics
vm.lifecycle (start/stop/reboot)
vm.tag
vm.snapshot.{create,restore,delete}
vm.resize
vm.create_from_template
vm.delete                                        (RiskLevel=high)
template.list
migration.start                                  (RiskLevel=high)
backup.create, backup.restore                    (restore=high)
netscan.run
notification.send_test
audit.search                                     (admin scope)
```

### Bedienwege

- **User-Proxy** (Default): Im Chat ruft der Agent Tools mit User-Permissions. Approvals folgen User-Policy. Audit: `actor_kind=agent, on_behalf_of_user=...`.
- **Service-Identity** (autonom): Agent handelt proaktiv (Schedule oder reactive auf Cluster-Events). Permissions = Service-Identity. Audit: `actor_kind=service, service_name=agent-runtime`. UI zeigt das mit Badge.

### Beispiele fuer realistische Agent-Workflows

- "Ich brauche ein kleines Ubuntu fuer einen Webserver." → Agent listet passende Templates oder schlaegt eines vor, fragt Namen, fragt Approval falls noetig, ruft `vm.create_from_template`, setzt Tags, meldet Bereitschaft.
- "Stoppe alle VMs mit Tag `env=staging`." → Agent listet, fragt Bestaetigung mit Liste, ruft `vm.lifecycle` der Reihe nach.
- "Was machen die VMs auf node-2?" → Agent listet, gibt CPU/RAM-Status, kommentiert Auffaelligkeiten.
- "Snapshot vor dem Update." → Agent erkennt VM-Kontext aus aktueller UI-Route (uebergeben durch Frontend), ruft `vm.snapshot.create`.
- "Loesch VM 102." → Agent erkennt `RiskLevel=high`, erstellt Approval-Antrag mit Begruendung, wartet auf Entscheidung, ruft `vm.delete` nur bei Granted.

## Datenmigration aus V1

V2-Schema heisst `prom_v2`, V1-Schema `public`. Datenmitnahme ueber ein V2-internes Subkommando.

### Migrations-Tool

```
prometheus migrate-from-v1 --domain=auth
prometheus migrate-from-v1 --domain=host
prometheus migrate-from-v1 --domain=notification
prometheus migrate-from-v1 --domain=backup
prometheus migrate-from-v1 --domain=migration
prometheus migrate-from-v1 --domain=netscan
prometheus migrate-from-v1 --domain=task
prometheus migrate-from-v1 --domain=audit
prometheus migrate-from-v1 --domain=agent          # Chat-Historie + Brain-Eintraege
prometheus migrate-from-v1 --all
```

Eigenschaften:

- **Idempotent.** Erkennt bereits migrierte Records ueber Source-ID-Mapping.
- **Resumable.** Fortschritt in `prom_v2.migration_state`.
- **Dry-Run-Modus.** `--dry-run` zeigt Counts ohne zu schreiben.
- **Validierung.** V1-Datensatz wird gegen V2-Schema validiert; Mismatches werden gelogged und uebersprungen, am Ende ein Bericht.

### ID-Strategie

V2 nutzt UUIDv7 als neue Primary-Keys. V1-IDs werden als `legacy_id`-Spalte mitgefuehrt:

```sql
CREATE TABLE prom_v2.hosts (
    id            uuid PRIMARY KEY,         -- V2 UUIDv7
    legacy_id     uuid UNIQUE,              -- V1 UUIDv4
    ...
);
```

`legacy_id` bleibt ueber V2-Stabilitaetsphase erhalten (Default 6 Monate), spaeter optional droppbar. Damit sind Re-Runs idempotent, Audit-Cross-References moeglich.

### Schema-Mapping pro Domaene

Mapping-Funktionen in `internal/migrate/<domain>.go` sind handgepflegt und testbar:

```go
func mapV1HostToV2(v1 v1schema.Host) v2schema.Host {
    return v2schema.Host{
        ID:           uuidv7.New(),
        LegacyID:     v1.ID,
        Name:         v1.Name,
        Capabilities: deriveCapabilities(v1.Kind),
        ProxmoxAuth:  decryptAndReencrypt(v1.EncryptedToken, v1Key, v2Key),
    }
}
```

V1-Felder, die in V2 wegfallen, werden bewusst nicht gemappt.

### Was migriert wird

| Domaene | Was rueber | Was bleibt in V1 |
|---|---|---|
| auth | Users, password-hashes, API-Keys (re-encrypted) | Sessions (User loggen neu ein) |
| host | Host-Connections, Proxmox-Tokens, abgeleitete Capabilities | – |
| vm (read-cache) | – | wird live aus Proxmox neu geholt |
| migration | Historie + laufende (laufende werden auf "interrupted" gesetzt mit Audit-Note) | – |
| backup | Schedules, Configs, Records, File-Pfade | – |
| netscan | Scan-Historie, Baselines, Port-Discoveries | Drift-Events (nice-to-have) |
| notification | Channels, Alert-Rules, Escalations, Versand-Historie (letzte 30 Tage) | Reflex-Rules (nice-to-have) |
| task | Letzte 30 Tage | – |
| audit | Vollstaendig | – |
| agent | Chat-Historie (letzte 30 Tage), Knowledge-Brain-Eintraege | – |
| Drift, Anomaly, Predictions, Recovery, Updates, Logs, Security-Engine | – | bleibt in V1 |

### Cutover-Phasen

| Phase | V2-Status | Datenstand |
|---|---|---|
| A | lesend, alle Must-Haves implementiert, kein Schreibverkehr | initiales `migrate-from-v1 --domain=...` pro Domain; Metriken werden ab da live gesammelt |
| B (pro Domain) | eine Domain aktiv | Catch-up-Migration vor Aktivierung; V1-Schreibsperre fuer diese Domain via Feature-Flag-Banner; V2 uebernimmt |
| C | V2 vollstaendig aktiv | alle Domain-Migrations durch; V1 nur lesend |
| D | V1 stillgelegt | V1-Schema bleibt liegen fuer Forensik, oder wird spaeter manuell gedropped |

### Rollback

Per Feature-Flag pro Domain: `domains.<name>=read_only` zurueck → V1 uebernimmt wieder. V2-Daten bleiben, V1-Daten waren nie weg. In der Zwischenzeit in V2 ausgefuehrte Aktionen sind in V2 sichtbar; Proxmox haelt die Wahrheit fuer Cluster-Zustand.

## Reliability und Observability

### Sichtbarkeit von Fehlern

Drei Schichten:

- **Inline-Card-Fehler** mit `Erneut versuchen`-Button, ausgeloest durch TanStack-Query-Error-State.
- **Toast-Fehler** bei nutzer-ausgeloesten Aktionen (Mutation `onError`). Code-uebersetzt-zu-deutsch, Verlinkung zum Audit-Eintrag bei kritischen Aktionen.
- **System-Banner** fuer fundamentale Ausfaelle (Postgres weg, Backend offline). Persistent bis behoben.

Stille Fehler sind verboten. `errcheck` in CI bricht den Build bei ungecheckten Errors.

### Retry-Strategien

| Wo | Pattern |
|---|---|
| TanStack Query (Reads) | Auto-Retry bei 5xx oder Netzwerkfehler, max 2x mit Backoff |
| TanStack Query (Mutations) | Kein Auto-Retry; User entscheidet (Idempotency-Key macht es sicher) |
| River-Jobs | Domain-spezifisch (kritisch: 1, idempotent-leicht: 3) |
| Proxmox-HTTP | Backoff bei 429/503, max 3x, expliziter Timeout |
| SSH | Pool-managed, broken connection → Rebuild, max 3x |
| WS/SSE | Browser Exponential Backoff (1s, 2s, 5s, 10s, max 30s) |

### Strukturiertes Logging

`slog` mit JSON-Handler. Feldset: `time`, `level`, `request_id`, `user_id`/`service_id`, `domain`, `action`, `error_code`, `error` (mit gewrappter Kette).

### Metriken

`/metrics` (Admin-only) im Prometheus-Format:

- `http_requests_total{method, path_pattern, status}`
- `http_request_duration_seconds`
- `db_query_duration_seconds{query}`
- `river_job_duration_seconds{kind, status}`
- `river_jobs_total{kind, status}`
- `ssh_connections_open{host_id}`
- `proxmox_api_calls_total{host_id, endpoint, status}`
- `notification_sent_total{channel_kind, status}`
- `agent_tool_calls_total{tool, status}`

### Health-Endpoints

- `GET /healthz` — Liveness, immer 200 solange Prozess lebt, kein Auth.
- `GET /readyz` — Readiness, prueft Postgres, Redis, mindestens einen erreichbaren Host. 503 wenn nicht ready, kein Auth.

## Tests und Verifikation

### Backend

- **Unit:** Service-Logik mit Mock-Repos, Mapping-Funktionen, Permission-Policies.
- **Integration:** Postgres mit Testcontainers, River-Roundtrip, HTTP-API gegen Echo-In-Memory-Server. OpenAPI-Spec wird gegen Server gefuzzt.
- `go fmt`, `golangci-lint` (mit `errcheck`, `govet`, `staticcheck`, `ineffassign`).
- `sqlc generate --check` als Drift-Detektor.
- `oapi-codegen --check` als Drift-Detektor.

### Frontend

- `tsc --noEmit`
- `eslint`
- `vitest` fuer Unit-Tests
- `playwright` fuer 3–5 kritische E2E-Flows: Login, Host-Hinzufuegen, VM-Erstellen-aus-Template, Migration starten + Approval, Backup ausloesen.

### CI

```
make verify
├── backend: fmt, lint, test, sqlc check, oapi-codegen check
├── frontend: tsc, eslint, vitest
└── frontend: playwright (smoke, optional in CI)
```

Bricht beim ersten Fehler. Kein Merge auf Main ohne gruenes `verify`.

## Sicherheit

- AES-256-GCM fuer in DB gespeicherte Secrets (Proxmox-Tokens, SMTP-Passwoerter, etc.).
- bcrypt cost 12 fuer User-Passwoerter.
- CSP-Header und Standard-Security-Header in Echo-Middleware.
- Token-Lebensdauer 15min limits Schaden bei JWT-Leak.
- Rate-Limiting pro IP und pro User/API-Key auf sensiblen Endpoints (Login, Token-Refresh, Approval-Decisions).
- noVNC- und SSH-Console-Sessions auditiert mit User, VM, Dauer, IP.
- API-Keys als Hash gespeichert, letzte 4 Zeichen sichtbar.

## Nicht-Ziele

Bewusst ausserhalb V2-Kern, bleiben in V1:

- Drift-Detection
- Anomaly-Detection / Prediction-Engine
- Recovery / DR-Profiles
- Updates-Check
- Logs-Streaming und -Analyse
- Security-Engine mit Befund-Workflow

In V2 vorbereitet, aber nicht ausgebaut:

- 2FA (Schema vorbereitet, kein UI-Flow)
- Plugin-System fuer LLM-Provider (drei feste Provider)
- Linux-General-Server-Adapter (Capability-Pattern gebaut, nur Proxmox implementiert)
- Horizontal-Scaling ueber mehrere V2-Binaries (Code unterstuetzt es, kein aktiver Deployment-Pfad)
- `masscan` und aggressive Scan-Engines

Eindeutig ausserhalb V2-Linie auch in Future-Phasen:

- Ersatz fuer Proxmox-WebGUI (V2 ist Cockpit, nicht WebGUI-Replacement)
- VM-Erstellen mit voller ISO/Storage/Netzwerk-Konfiguration ausserhalb von Templates
- Hardware-tiefe VM-Aenderungen (Disk-Tausch, PCIe-Passthrough, Boot-Reihenfolge-Edits)
- ISO-Verwaltung, Storage-Verwaltung, Netzwerk-Bridges, Cluster-Konfiguration
- Container/K8s-Cluster-Management
- Mandantenfaehigkeit, OAuth/SSO/SAML

## Akzeptanzkriterien

- V2-Binary startet selbststaendig, liefert SPA aus, exponiert REST-API unter `/api/v1/`.
- Login mit V1-User-Account funktioniert nach `migrate-from-v1 --domain=auth`.
- Hosts aus V1 sind nach `migrate-from-v1 --domain=host` lesbar und liefern Live-Metriken via WebSocket.
- VM-Liste, Lifecycle (Start/Stop/Reboot), Tags, Resize, Snapshots, Console funktionieren fuer Operator-Rolle ohne Proxmox-Login.
- VM-Erstellen aus Template fuehrt zu lauffaehiger VM mit korrekten Tags.
- VM-Loeschen erzeugt immer einen Approval-Antrag (ausser Admin-Bypass), der entscheidbar ist.
- VM-Migration laeuft als River-Job, ist im Task-Center fuer alle berechtigten User live sichtbar, Audit-Log enthaelt Initiator und Approver.
- Backup-Erstellen und -Restore laufen end-to-end, Restore mit Approval-Pflicht.
- Netzwerk-Quick-Scan und Full-Scan laufen; fehlendes `nmap` blockiert Full-Scan mit klarer Begruendung.
- Telegram- und SMTP-Channels haben Status-Anzeige und Test-Aktion mit sichtbarem Ergebnis.
- Default-Alert-Regeln triggern auf Node-down, Migration-failed, Backup-failed.
- Agent kann via Chat VM-Lifecycle, Erstellen-aus-Template, Tagging, Migration, Backup, Snapshot ausloesen — mit Permission-Check, Approval-Workflow und Audit.
- Approval-Inbox fuer Admins zeigt offene Antraege live via SSE, Decision-Latency wird auditiert.
- Audit-Log ist append-only, durchsuchbar nach Actor, Resource, Action, Zeit.
- Multi-Admin-Setup: zwei Admin-User koennen Antraege voneinander genehmigen, Self-Approval ist per System-Policy steuerbar.
- API-Keys: Personal-Key hat User-Permissions; Service-Key (`agent-runtime`) kann Approval-Aktionen mit `auto_approve` ausloesen, Audit zeigt das eindeutig.
- `/metrics`-Endpoint exportiert die definierten Prometheus-Metriken.
- `/healthz` und `/readyz` antworten korrekt.
- `make verify` ist gruen.
- Single-Binary-Deploy funktioniert: `prometheus`-Binary plus Postgres plus Redis, sonst nichts.
