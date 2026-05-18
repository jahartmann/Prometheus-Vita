# Changelog

All notable changes to Prometheus are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project
follows [Semantic Versioning](https://semver.org/).

## [Unreleased] — 2026-05-18 release-hardening pass

### Added
- **Caddy reverse-proxy profile** (`docker compose --profile tls up`) with
  Let's Encrypt auto-issuance for public domains and internal CA for LAN/dev.
  Terminates TLS, sets HSTS + security headers, proxies REST + WebSocket
  under a single origin.
- **Postgres backup sidecar** (`docker compose --profile backup up`).
  `pg_dump --clean --if-exists` on a 24h interval with 14-day retention,
  plus an interactive `pg-restore.sh` helper.
- **`docs/DEPLOYMENT.md`** — production deployment checklist, TLS modes,
  resource sizing, update + revert procedure.
- **`docs/BACKUP.md`** — backup strategy, restore procedures (in-container
  and from host), DR runbook, verification.
- **`docs/TROUBLESHOOT.md`** — symptom-keyed troubleshooting (boot
  failures, cert issuance, WS drops, migration stuck, resource
  exhaustion).
- `/live` and `/ready` health endpoints alongside the legacy `/health`. `/live`
  is a cheap process-liveness probe; `/ready` verifies Postgres + Redis and
  reports `503` when either is down.
- `LOG_LEVEL` and `LOG_FORMAT` environment variables. Logging level is now
  runtime-configurable (`debug` | `info` | `warn` | `error`) and the format
  can be switched between `json` (default) and `text` for human reading.
- `Dropped` counter and `Shutdown(ctx)` method on the WebSocket hub. Exposes
  silent drops to ops dashboards and lets the hub stop cleanly on SIGTERM.
- Container `HEALTHCHECK` directives on both backend and frontend images.
- Postgres advisory lock around schema migrations so multiple replicas
  starting up don't race to migrate.
- Resource limits (cpu/memory) declared per service in `docker-compose.yml`.
- User-visible "Sitzung abgelaufen" toast when JWT refresh fails, instead of
  a silent logout.
- Toast feedback on backup download failures (previously swallowed silently).

### Changed
- Health endpoints no longer leak raw error strings to clients; details are
  logged server-side only.
- WebSocket hub uses `sync.Once` to guarantee each client's send channel is
  closed at most once, removing a latent double-close panic when a slow
  client was both dropped during broadcast and unregistered concurrently.
- Migration rollback failures are now logged (previously discarded with
  `_ = tx.Rollback(ctx)`).
- `docker-compose.yml` now uses `depends_on.condition: service_healthy`
  for the frontend → backend dependency, so the frontend doesn't start
  serving traffic against an unready backend.

### Hardening
- `ENCRYPTION_KEY` validated for length (64 hex chars) and character set at
  startup. Misconfigured keys now fail fast with a clear error.
- `POSTGRES_PASSWORD` and `REDIS_PASSWORD` validated against the
  `changeme_*` placeholders to prevent accidental production deploys with
  default credentials.
- Redis configured with `maxmemory 256mb` and an `allkeys-lru` policy so an
  unbounded cache cannot exhaust the container's memory budget.
