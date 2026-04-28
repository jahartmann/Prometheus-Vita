# Prometheus V2

Rebuild von Prometheus auf der V2-Architektur.

## Voraussetzungen

- Go 1.23+
- Node.js 20+
- Docker + Docker-Compose
- `sqlc` (`go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`)
- `oapi-codegen` v2 (`go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest`)
- `golang-migrate` (`go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`)

## Bootstrap

```bash
make tools          # installiert Codegen-CLIs
make generate       # generiert sqlc + oapi-codegen + openapi-typescript
make verify         # fmt, lint, test, type-check
make dev-up         # docker-compose hoch (postgres + redis)
make dev            # backend + frontend Live-Mode
```

## Architektur

Siehe `../Prometheus-Vita/docs/superpowers/specs/2026-04-28-prometheus-v2-rebuild-design.md`.
