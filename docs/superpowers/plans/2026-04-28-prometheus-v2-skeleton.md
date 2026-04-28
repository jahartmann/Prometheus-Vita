# Prometheus V2 Skeleton & Plattform Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Lege das vollstaendige V2-Skeleton an: leeres Single-Binary, das Frontend (Vite+React+TanStack) eingebettet ausliefert, Postgres+River verbunden, OpenAPI/sqlc-Codegen-Pipeline laufend, Health-/Metrics-Endpoints, `make verify` gruen.

**Architecture:** Domain-Module-Layout vorbereiten (leer). Backend-Plattform (`internal/platform/`) ist die einzige cross-cutting-Schicht. Frontend laedt einen Test-Endpoint per generiertem TS-Client. Build-Pipeline embedded SPA via `go:embed`.

**Tech Stack:** Go 1.23, Echo v4, pgx v5, golang-migrate, sqlc, River v0.13+, Redis v9, oapi-codegen v2, Prometheus-Client, Vite 6, React 19, TypeScript 5, TanStack Router v1, TanStack Query v5, Tailwind v4, shadcn/ui, openapi-typescript, openapi-fetch.

---

## Scope Check

Dieser Plan baut **nur das Skeleton** — keine Domains, keine Auth, keine Business-Logik. Alle weiteren Plaene (Auth, Audit, Realtime-Bus, Approval, Host, VM, Template, Console, Migration, Backup, Netscan, Notification, Task-Center, Agent, Data-Migrator, Cutover) bauen auf diesem Skeleton auf. Wenn ein Worker Tasks aus diesem Plan nicht in einem fokussierten Pass durchziehen kann, splitten in Folge-Plan, **nicht** in andere Bereiche der Spec ausweiten.

## File Map

Repo-Root (committed in `Prometheus-Vita/` Git-Repo, neuer Geschwister-Ordner `Prometheus-V2/`):

**Backend:**

- Create: `Prometheus-V2/backend/go.mod`
- Create: `Prometheus-V2/backend/cmd/prometheus/main.go`
- Create: `Prometheus-V2/backend/internal/config/config.go`
- Create: `Prometheus-V2/backend/internal/config/config_test.go`
- Create: `Prometheus-V2/backend/internal/platform/log/log.go`
- Create: `Prometheus-V2/backend/internal/platform/db/db.go`
- Create: `Prometheus-V2/backend/internal/platform/db/migrate.go`
- Create: `Prometheus-V2/backend/internal/platform/db/db_test.go`
- Create: `Prometheus-V2/backend/internal/platform/jobs/client.go`
- Create: `Prometheus-V2/backend/internal/platform/jobs/client_test.go`
- Create: `Prometheus-V2/backend/internal/platform/redis/redis.go`
- Create: `Prometheus-V2/backend/internal/platform/metrics/metrics.go`
- Create: `Prometheus-V2/backend/internal/http/server.go`
- Create: `Prometheus-V2/backend/internal/http/health.go`
- Create: `Prometheus-V2/backend/internal/http/health_test.go`
- Create: `Prometheus-V2/backend/internal/web/embed.go`
- Create: `Prometheus-V2/backend/db/migrations/000001_init.up.sql`
- Create: `Prometheus-V2/backend/db/migrations/000001_init.down.sql`
- Create: `Prometheus-V2/backend/db/queries/meta.sql`
- Create: `Prometheus-V2/backend/sqlc.yaml`
- Create: `Prometheus-V2/backend/oapi-codegen.yaml`
- Create: `Prometheus-V2/backend/api/openapi.yaml`
- Create: `Prometheus-V2/backend/internal/api/openapi.gen.go` (sqlc/oapi-codegen-Output)
- Create: `Prometheus-V2/backend/internal/db/repo/meta.sql.go` (sqlc-Output)
- Create: `Prometheus-V2/backend/.golangci.yml`
- Create: `Prometheus-V2/backend/Makefile`

**Frontend:**

- Create: `Prometheus-V2/frontend/package.json`
- Create: `Prometheus-V2/frontend/tsconfig.json`
- Create: `Prometheus-V2/frontend/tsconfig.node.json`
- Create: `Prometheus-V2/frontend/vite.config.ts`
- Create: `Prometheus-V2/frontend/index.html`
- Create: `Prometheus-V2/frontend/postcss.config.cjs`
- Create: `Prometheus-V2/frontend/tailwind.config.ts`
- Create: `Prometheus-V2/frontend/components.json`
- Create: `Prometheus-V2/frontend/eslint.config.js`
- Create: `Prometheus-V2/frontend/vitest.config.ts`
- Create: `Prometheus-V2/frontend/src/main.tsx`
- Create: `Prometheus-V2/frontend/src/index.css`
- Create: `Prometheus-V2/frontend/src/routes/__root.tsx`
- Create: `Prometheus-V2/frontend/src/routes/index.tsx`
- Create: `Prometheus-V2/frontend/src/lib/api/client.ts`
- Create: `Prometheus-V2/frontend/src/lib/api/schema.d.ts` (openapi-typescript-Output)
- Create: `Prometheus-V2/frontend/src/lib/query.ts`
- Create: `Prometheus-V2/frontend/src/lib/utils.ts`
- Create: `Prometheus-V2/frontend/src/components/health-status.tsx`
- Create: `Prometheus-V2/frontend/src/components/health-status.test.tsx`
- Create: `Prometheus-V2/frontend/src/components/layout/app-shell.tsx`
- Create: `Prometheus-V2/frontend/src/components/layout/page-shell.tsx`
- Create: `Prometheus-V2/frontend/src/components/layout/sidebar.tsx`
- Create: `Prometheus-V2/frontend/src/components/layout/topbar.tsx`
- Create: `Prometheus-V2/frontend/src/components/ui/button.tsx`
- Create: `Prometheus-V2/frontend/src/components/ui/card.tsx`
- Create: `Prometheus-V2/frontend/src/components/ui/badge.tsx`
- Create: `Prometheus-V2/frontend/src/components/ui/status-badge.tsx`
- Create: `Prometheus-V2/frontend/src/components/ui/kpi-card.tsx`
- Create: `Prometheus-V2/frontend/src/components/ui/action-card.tsx`
- Create: `Prometheus-V2/frontend/src/components/ui/feature-status-card.tsx`
- Create: `Prometheus-V2/frontend/src/components/ui/empty-state.tsx`
- Create: `Prometheus-V2/frontend/src/components/ui/error-state.tsx`
- Create: `Prometheus-V2/frontend/src/components/theme-provider.tsx`
- Create: `Prometheus-V2/frontend/src/components/theme-toggle.tsx`
- Create: `Prometheus-V2/frontend/src/components/ui/status-badge.test.tsx`
- Create: `Prometheus-V2/frontend/.gitignore`

**Root V2:**

- Create: `Prometheus-V2/Makefile`
- Create: `Prometheus-V2/Dockerfile`
- Create: `Prometheus-V2/docker-compose.yml`
- Create: `Prometheus-V2/.gitignore`
- Create: `Prometheus-V2/README.md`

---

## Task 1: Repository-Skeleton anlegen

**Files:**
- Create: `Prometheus-V2/.gitignore`
- Create: `Prometheus-V2/README.md`
- Create: `Prometheus-V2/backend/go.mod`
- Create: `Prometheus-V2/frontend/.gitignore`
- Create: `Prometheus-V2/backend/.gitignore` (ggf.)

- [ ] **Step 1: V2-Verzeichnisbaum anlegen**

Erstelle die Ordnerstruktur (alle leer, nur Git-Tracking-Marker via `.gitkeep` wo noetig):

```bash
cd "D:/Dokumente & Inhalte/Programmierung/Prometheus/Prometheus-Vita"
mkdir -p Prometheus-V2/backend/cmd/prometheus
mkdir -p Prometheus-V2/backend/internal/config
mkdir -p Prometheus-V2/backend/internal/http
mkdir -p Prometheus-V2/backend/internal/web
mkdir -p Prometheus-V2/backend/internal/platform/log
mkdir -p Prometheus-V2/backend/internal/platform/db
mkdir -p Prometheus-V2/backend/internal/platform/jobs
mkdir -p Prometheus-V2/backend/internal/platform/redis
mkdir -p Prometheus-V2/backend/internal/platform/metrics
mkdir -p Prometheus-V2/backend/internal/api
mkdir -p Prometheus-V2/backend/internal/db/repo
mkdir -p Prometheus-V2/backend/db/migrations
mkdir -p Prometheus-V2/backend/db/queries
mkdir -p Prometheus-V2/backend/api
mkdir -p Prometheus-V2/frontend/src/routes
mkdir -p Prometheus-V2/frontend/src/lib/api
mkdir -p Prometheus-V2/frontend/src/components
```

Expected: keine Fehler, alle Verzeichnisse existieren.

- [ ] **Step 2: Root `.gitignore` schreiben**

Erstelle `Prometheus-V2/.gitignore`:

```gitignore
# Build outputs
backend/bin/
backend/internal/web/dist/
frontend/dist/
frontend/node_modules/
frontend/.vite/

# Generated code (regenerated via make generate)
backend/internal/api/openapi.gen.go
backend/internal/db/repo/*.sql.go
backend/internal/db/repo/db.go
backend/internal/db/repo/models.go
frontend/src/lib/api/schema.d.ts
frontend/src/routeTree.gen.ts

# IDE
.vscode/
.idea/
*.iml

# OS
.DS_Store
Thumbs.db

# Env
.env
.env.local

# Test artifacts
backend/coverage.out
frontend/coverage/

# Tooling
backend/.tools/
```

- [ ] **Step 3: Frontend-`.gitignore`**

Erstelle `Prometheus-V2/frontend/.gitignore`:

```gitignore
node_modules/
dist/
.vite/
coverage/
*.log
src/routeTree.gen.ts
src/lib/api/schema.d.ts
```

- [ ] **Step 4: README mit Bootstrap-Hinweisen**

Erstelle `Prometheus-V2/README.md`:

```markdown
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
```

- [ ] **Step 5: Go-Modul initialisieren**

```bash
cd Prometheus-V2/backend
go mod init github.com/antigravity/prometheus-v2
```

Expected: erstellt `Prometheus-V2/backend/go.mod` mit Modul-Pfad.

- [ ] **Step 6: Commit Skeleton**

```bash
cd "D:/Dokumente & Inhalte/Programmierung/Prometheus/Prometheus-Vita"
git add Prometheus-V2/
git commit -m "feat(v2): scaffold repository skeleton"
```

---

## Task 2: Backend-Bootstrap — Config, Logging, Health-Endpoints

**Files:**
- Create: `Prometheus-V2/backend/internal/config/config.go`
- Create: `Prometheus-V2/backend/internal/config/config_test.go`
- Create: `Prometheus-V2/backend/internal/platform/log/log.go`
- Create: `Prometheus-V2/backend/internal/http/server.go`
- Create: `Prometheus-V2/backend/internal/http/health.go`
- Create: `Prometheus-V2/backend/internal/http/health_test.go`
- Create: `Prometheus-V2/backend/cmd/prometheus/main.go`

- [ ] **Step 1: Echo + Test-Bibliotheken hinzufuegen**

```bash
cd Prometheus-V2/backend
go get github.com/labstack/echo/v4@v4.12.0
go get github.com/stretchr/testify@v1.9.0
```

Expected: `go.mod` und `go.sum` enthalten beide Module.

- [ ] **Step 2: Config-Test schreiben (failing)**

Erstelle `Prometheus-V2/backend/internal/config/config_test.go`:

```go
package config_test

import (
	"testing"

	"github.com/antigravity/prometheus-v2/internal/config"
	"github.com/stretchr/testify/require"
)

func TestLoad_DefaultsWhenEnvIsEmpty(t *testing.T) {
	t.Setenv("PROMETHEUS_HTTP_ADDR", "")
	t.Setenv("PROMETHEUS_LOG_LEVEL", "")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, ":8180", cfg.HTTPAddr)
	require.Equal(t, "info", cfg.LogLevel)
}

func TestLoad_OverridesFromEnv(t *testing.T) {
	t.Setenv("PROMETHEUS_HTTP_ADDR", ":9999")
	t.Setenv("PROMETHEUS_LOG_LEVEL", "debug")
	t.Setenv("PROMETHEUS_DATABASE_URL", "postgres://x:y@z/db")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, ":9999", cfg.HTTPAddr)
	require.Equal(t, "debug", cfg.LogLevel)
	require.Equal(t, "postgres://x:y@z/db", cfg.DatabaseURL)
}
```

- [ ] **Step 3: Test laufen lassen — muss fehlschlagen**

```bash
cd Prometheus-V2/backend
go test ./internal/config/...
```

Expected: FAIL mit "no Go files" oder "package config not found".

- [ ] **Step 4: Config-Implementation**

Erstelle `Prometheus-V2/backend/internal/config/config.go`:

```go
package config

import (
	"os"
)

type Config struct {
	HTTPAddr    string
	LogLevel    string
	DatabaseURL string
	RedisURL    string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:    getenv("PROMETHEUS_HTTP_ADDR", ":8180"),
		LogLevel:    getenv("PROMETHEUS_LOG_LEVEL", "info"),
		DatabaseURL: getenv("PROMETHEUS_DATABASE_URL", "postgres://prometheus:prometheus@localhost:5432/prometheus_v2?sslmode=disable&search_path=prom_v2"),
		RedisURL:    getenv("PROMETHEUS_REDIS_URL", "redis://localhost:6379/0"),
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
```

- [ ] **Step 5: Test laufen lassen — muss bestehen**

```bash
go test ./internal/config/...
```

Expected: PASS.

- [ ] **Step 6: Strukturiertes Logging**

Erstelle `Prometheus-V2/backend/internal/platform/log/log.go`:

```go
package log

import (
	"log/slog"
	"os"
	"strings"
)

func New(level string) *slog.Logger {
	var lv slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lv = slog.LevelDebug
	case "warn":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     lv,
		AddSource: false,
	})
	return slog.New(handler)
}
```

- [ ] **Step 7: Health-Handler-Test schreiben (failing)**

Erstelle `Prometheus-V2/backend/internal/http/health_test.go`:

```go
package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	httpserver "github.com/antigravity/prometheus-v2/internal/http"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestHealthz_Returns200(t *testing.T) {
	e := echo.New()
	httpserver.RegisterHealth(e, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"status":"ok"}`, rec.Body.String())
}

func TestReadyz_ReturnsServiceUnavailableWhenDBNil(t *testing.T) {
	e := echo.New()
	httpserver.RegisterHealth(e, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
}
```

- [ ] **Step 8: Test laufen lassen — muss fehlschlagen**

```bash
go test ./internal/http/...
```

Expected: FAIL — Package `http` enthaelt noch nichts.

- [ ] **Step 9: Health-Handler-Implementation**

Erstelle `Prometheus-V2/backend/internal/http/health.go`:

```go
package http

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
)

type DBPinger interface {
	Ping(ctx context.Context) error
}

type RedisPinger interface {
	Ping(ctx context.Context) error
}

func RegisterHealth(e *echo.Echo, db DBPinger, redis RedisPinger) {
	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	e.GET("/readyz", func(c echo.Context) error {
		ctx := c.Request().Context()
		if db == nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "db not initialized"})
		}
		if err := db.Ping(ctx); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "db unhealthy", "error": err.Error()})
		}
		if redis != nil {
			if err := redis.Ping(ctx); err != nil {
				return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "redis unhealthy", "error": err.Error()})
			}
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ready"})
	})
}
```

- [ ] **Step 10: Server-Setup-File**

Erstelle `Prometheus-V2/backend/internal/http/server.go`:

```go
package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Deps struct {
	Logger *slog.Logger
	DB     DBPinger
	Redis  RedisPinger
}

func NewServer(deps Deps) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.RequestID())
	e.Use(middleware.Recover())
	e.Use(middleware.Gzip())

	RegisterHealth(e, deps.DB, deps.Redis)
	return e
}

func ListenAndServe(ctx context.Context, e *echo.Echo, addr string, logger *slog.Logger) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           e,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("http server starting", slog.String("addr", addr))
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
```

- [ ] **Step 11: Health-Tests laufen lassen**

```bash
go test ./internal/http/...
```

Expected: PASS fuer beide Tests.

- [ ] **Step 12: main.go schreiben**

Erstelle `Prometheus-V2/backend/cmd/prometheus/main.go`:

```go
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/antigravity/prometheus-v2/internal/config"
	httpserver "github.com/antigravity/prometheus-v2/internal/http"
	"github.com/antigravity/prometheus-v2/internal/platform/log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", slog.Any("error", err))
		os.Exit(1)
	}

	logger := log.New(cfg.LogLevel)
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	server := httpserver.NewServer(httpserver.Deps{
		Logger: logger,
		DB:     nil,
		Redis:  nil,
	})

	if err := httpserver.ListenAndServe(ctx, server, cfg.HTTPAddr, logger); err != nil {
		logger.Error("server stopped with error", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("server stopped cleanly")
}
```

- [ ] **Step 13: Build und smoke-test**

```bash
go build ./cmd/prometheus
./prometheus &
sleep 1
curl -s http://localhost:8180/healthz
kill %1
```

Expected: Output `{"status":"ok"}`.

- [ ] **Step 14: Commit**

```bash
cd "D:/Dokumente & Inhalte/Programmierung/Prometheus/Prometheus-Vita"
git add Prometheus-V2/backend/
git commit -m "feat(v2): bootstrap echo server with health endpoints and config"
```

---

## Task 3: OpenAPI-Spec + oapi-codegen

**Files:**
- Create: `Prometheus-V2/backend/api/openapi.yaml`
- Create: `Prometheus-V2/backend/oapi-codegen.yaml`
- Create: `Prometheus-V2/backend/internal/api/openapi.gen.go` (generiert)
- Modify: `Prometheus-V2/backend/internal/http/server.go`

- [ ] **Step 1: oapi-codegen-Tool installieren**

```bash
cd Prometheus-V2/backend
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
```

Expected: Tool ist im `$GOPATH/bin` verfuegbar.

- [ ] **Step 2: oapi-codegen-Library hinzufuegen**

```bash
go get github.com/oapi-codegen/runtime@latest
go get github.com/oapi-codegen/echo-middleware@latest
```

- [ ] **Step 3: Initiale OpenAPI-Spec**

Erstelle `Prometheus-V2/backend/api/openapi.yaml`:

```yaml
openapi: 3.0.3
info:
  title: Prometheus V2 API
  version: 0.1.0
  description: Prometheus V2 Operations Cockpit API.
servers:
  - url: /api/v1
    description: Default V2 base path
paths:
  /system/health:
    get:
      summary: System health summary
      operationId: getSystemHealth
      tags: [system]
      responses:
        "200":
          description: System is healthy
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/SystemHealth"
        "503":
          description: System is degraded
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Error"
components:
  schemas:
    SystemHealth:
      type: object
      required: [status, version, request_id]
      properties:
        status:
          type: string
          enum: [ok, degraded, down]
        version:
          type: string
        request_id:
          type: string
    Error:
      type: object
      required: [code, message, request_id]
      properties:
        code:
          type: string
        message:
          type: string
        details:
          type: object
          additionalProperties: true
        request_id:
          type: string
```

- [ ] **Step 4: oapi-codegen-Konfiguration**

Erstelle `Prometheus-V2/backend/oapi-codegen.yaml`:

```yaml
package: api
generate:
  echo-server: true
  models: true
  embedded-spec: true
output: internal/api/openapi.gen.go
output-options:
  skip-prune: true
```

- [ ] **Step 5: Generierung laufen lassen**

```bash
cd Prometheus-V2/backend
oapi-codegen -config oapi-codegen.yaml api/openapi.yaml
```

Expected: erstellt `Prometheus-V2/backend/internal/api/openapi.gen.go` mit Generated-Models, Server-Interface und Embedded-Spec.

- [ ] **Step 6: Generierten Code commiten erlauben (lokal)**

Kontrolliere, dass `internal/api/openapi.gen.go` jetzt existiert. (Im `.gitignore` ist es ausgeschlossen, aber waehrend Build wird es regeneriert.)

- [ ] **Step 7: Echo-Server-Stub einhaengen**

Modifiziere `Prometheus-V2/backend/internal/http/server.go` — ergaenze nach den vorhandenen Imports und den Body von `NewServer`:

```go
// in den Imports ergaenzen:
//   "github.com/antigravity/prometheus-v2/internal/api"
//   oapimw "github.com/oapi-codegen/echo-middleware"

// ergaenze in NewServer nach RegisterHealth:
spec, err := api.GetSwagger()
if err != nil {
	deps.Logger.Error("openapi spec load failed", slog.Any("error", err))
	return e
}
spec.Servers = nil
v1 := e.Group("/api/v1")
v1.Use(oapimw.OapiRequestValidator(spec))
api.RegisterHandlersWithBaseURL(e, &apiServer{}, "/api/v1")
```

Definiere am File-Ende ein leeres apiServer-Stub fuer den `getSystemHealth`-Handler:

```go
type apiServer struct{}

func (apiServer) GetSystemHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, api.SystemHealth{
		Status:    "ok",
		Version:   "0.1.0",
		RequestId: c.Response().Header().Get(echo.HeaderXRequestID),
	})
}
```

Vollstaendiger Stand `server.go`:

```go
package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/antigravity/prometheus-v2/internal/api"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	oapimw "github.com/oapi-codegen/echo-middleware"
)

type Deps struct {
	Logger *slog.Logger
	DB     DBPinger
	Redis  RedisPinger
}

type apiServer struct{}

func (apiServer) GetSystemHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, api.SystemHealth{
		Status:    "ok",
		Version:   "0.1.0",
		RequestId: c.Response().Header().Get(echo.HeaderXRequestID),
	})
}

func NewServer(deps Deps) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.RequestID())
	e.Use(middleware.Recover())
	e.Use(middleware.Gzip())

	RegisterHealth(e, deps.DB, deps.Redis)

	spec, err := api.GetSwagger()
	if err != nil {
		deps.Logger.Error("openapi spec load failed", slog.Any("error", err))
		return e
	}
	spec.Servers = nil
	v1 := e.Group("/api/v1")
	v1.Use(oapimw.OapiRequestValidator(spec))
	api.RegisterHandlersWithBaseURL(e, &apiServer{}, "/api/v1")

	return e
}

func ListenAndServe(ctx context.Context, e *echo.Echo, addr string, logger *slog.Logger) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           e,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("http server starting", slog.String("addr", addr))
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
```

- [ ] **Step 8: Verifizieren via Build und HTTP-Call**

```bash
go build ./cmd/prometheus
./prometheus &
sleep 1
curl -s http://localhost:8180/api/v1/system/health | head
kill %1
```

Expected: Output enthaelt `"status":"ok"`, `"version":"0.1.0"`, plus `request_id`.

- [ ] **Step 9: Test fuer den generierten Endpoint**

Ergaenze `Prometheus-V2/backend/internal/http/health_test.go` um folgenden Test:

```go
func TestSystemHealth_ReturnsExpectedShape(t *testing.T) {
	deps := httpserver.Deps{Logger: slog.Default(), DB: nil, Redis: nil}
	server := httpserver.NewServer(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/health", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"status":"ok"`)
	require.Contains(t, rec.Body.String(), `"version":"0.1.0"`)
}
```

Ergaenze die fehlenden Imports oben im Test-File:

```go
import (
    "log/slog"
    // bestehende Imports
)
```

```bash
go test ./internal/http/...
```

Expected: PASS fuer alle drei Tests.

- [ ] **Step 10: Commit**

```bash
cd "D:/Dokumente & Inhalte/Programmierung/Prometheus/Prometheus-Vita"
git add Prometheus-V2/backend/api/ Prometheus-V2/backend/oapi-codegen.yaml Prometheus-V2/backend/internal/http/ Prometheus-V2/backend/go.mod Prometheus-V2/backend/go.sum
git commit -m "feat(v2): wire openapi spec with oapi-codegen and system/health endpoint"
```

---

## Task 4: Postgres-Anbindung, Migrations und sqlc

**Files:**
- Create: `Prometheus-V2/backend/internal/platform/db/db.go`
- Create: `Prometheus-V2/backend/internal/platform/db/migrate.go`
- Create: `Prometheus-V2/backend/internal/platform/db/db_test.go`
- Create: `Prometheus-V2/backend/db/migrations/000001_init.up.sql`
- Create: `Prometheus-V2/backend/db/migrations/000001_init.down.sql`
- Create: `Prometheus-V2/backend/db/queries/meta.sql`
- Create: `Prometheus-V2/backend/sqlc.yaml`
- Modify: `Prometheus-V2/backend/cmd/prometheus/main.go`

- [ ] **Step 1: pgx + golang-migrate hinzufuegen**

```bash
cd Prometheus-V2/backend
go get github.com/jackc/pgx/v5@v5.7.0
go get github.com/jackc/pgx/v5/pgxpool@v5.7.0
go get github.com/golang-migrate/migrate/v4@v4.18.0
go get github.com/golang-migrate/migrate/v4/database/pgx/v5@v4.18.0
go get github.com/golang-migrate/migrate/v4/source/file@v4.18.0
```

- [ ] **Step 2: Erste Migration — Schema und Meta-Tabelle**

Erstelle `Prometheus-V2/backend/db/migrations/000001_init.up.sql`:

```sql
CREATE SCHEMA IF NOT EXISTS prom_v2;
SET search_path TO prom_v2;

CREATE TABLE IF NOT EXISTS _v2_meta (
    key   text PRIMARY KEY,
    value text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO _v2_meta (key, value) VALUES
    ('schema_version', '1'),
    ('installed_at',   now()::text)
ON CONFLICT (key) DO NOTHING;
```

Erstelle `Prometheus-V2/backend/db/migrations/000001_init.down.sql`:

```sql
DROP TABLE IF EXISTS prom_v2._v2_meta;
DROP SCHEMA IF EXISTS prom_v2;
```

- [ ] **Step 3: sqlc-Konfiguration**

Erstelle `Prometheus-V2/backend/sqlc.yaml`:

```yaml
version: "2"
sql:
  - engine: postgresql
    queries: db/queries
    schema: db/migrations
    gen:
      go:
        package: repo
        out: internal/db/repo
        sql_package: pgx/v5
        emit_pointers_for_null_types: true
        emit_db_tags: true
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: true
```

- [ ] **Step 4: sqlc-Tool installieren**

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

Expected: `sqlc` CLI ist im `$GOPATH/bin` verfuegbar.

- [ ] **Step 5: Erste sqlc-Query**

Erstelle `Prometheus-V2/backend/db/queries/meta.sql`:

```sql
-- name: GetMetaValue :one
SELECT value FROM _v2_meta WHERE key = $1;

-- name: SetMetaValue :exec
INSERT INTO _v2_meta (key, value)
VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE
  SET value = EXCLUDED.value,
      updated_at = now();
```

- [ ] **Step 6: sqlc-Generierung laufen lassen**

```bash
cd Prometheus-V2/backend
sqlc generate
```

Expected: erstellt `internal/db/repo/db.go`, `internal/db/repo/models.go`, `internal/db/repo/meta.sql.go` mit typisierten Funktionen `GetMetaValue` und `SetMetaValue`.

- [ ] **Step 7: DB-Verbindungs-Wrapper**

Erstelle `Prometheus-V2/backend/internal/platform/db/db.go`:

```go
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool struct {
	*pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse pgx config: %w", err)
	}
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.MaxConnLifetime = 1 * time.Hour
	cfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("new pgx pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Pool{Pool: pool}, nil
}

func (p *Pool) Ping(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return p.Pool.Ping(pingCtx)
}
```

- [ ] **Step 8: Migrations-Runner**

Erstelle `Prometheus-V2/backend/internal/platform/db/migrate.go`:

```go
package db

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(dsn, sourceURL string) error {
	m, err := migrate.New(sourceURL, dsn)
	if err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}
```

- [ ] **Step 9: Integrations-Test fuer DB**

Erstelle `Prometheus-V2/backend/internal/platform/db/db_test.go`:

```go
package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/antigravity/prometheus-v2/internal/platform/db"
	"github.com/stretchr/testify/require"
)

func TestNew_PingsPostgres(t *testing.T) {
	dsn := os.Getenv("PROMETHEUS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("PROMETHEUS_TEST_DATABASE_URL not set; skipping integration test")
	}

	ctx := context.Background()
	pool, err := db.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	require.NoError(t, pool.Ping(ctx))
}
```

- [ ] **Step 10: Test ausfuehren**

```bash
cd Prometheus-V2/backend
PROMETHEUS_TEST_DATABASE_URL="" go test ./internal/platform/db/...
```

Expected: SKIP — kein Postgres-Setup.

- [ ] **Step 11: main.go um DB-Init erweitern**

Modifiziere `Prometheus-V2/backend/cmd/prometheus/main.go` komplett:

```go
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/antigravity/prometheus-v2/internal/config"
	httpserver "github.com/antigravity/prometheus-v2/internal/http"
	"github.com/antigravity/prometheus-v2/internal/platform/db"
	"github.com/antigravity/prometheus-v2/internal/platform/log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", slog.Any("error", err))
		os.Exit(1)
	}

	logger := log.New(cfg.LogLevel)
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := db.RunMigrations(cfg.DatabaseURL, "file://db/migrations"); err != nil {
		logger.Error("db migrations failed", slog.Any("error", err))
		os.Exit(1)
	}

	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("db init failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer pool.Close()

	server := httpserver.NewServer(httpserver.Deps{
		Logger: logger,
		DB:     pool,
		Redis:  nil,
	})

	if err := httpserver.ListenAndServe(ctx, server, cfg.HTTPAddr, logger); err != nil {
		logger.Error("server stopped with error", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("server stopped cleanly")
}
```

- [ ] **Step 12: Build verifizieren**

```bash
cd Prometheus-V2/backend
go build ./cmd/prometheus
```

Expected: keine Compile-Fehler.

- [ ] **Step 13: Commit**

```bash
cd "D:/Dokumente & Inhalte/Programmierung/Prometheus/Prometheus-Vita"
git add Prometheus-V2/backend/internal/platform/db/ Prometheus-V2/backend/db/ Prometheus-V2/backend/sqlc.yaml Prometheus-V2/backend/cmd/prometheus/main.go Prometheus-V2/backend/go.mod Prometheus-V2/backend/go.sum
git commit -m "feat(v2): wire postgres pool, migrations, and sqlc pipeline"
```

---

## Task 5: River-Job-Queue-Wiring (leer)

**Files:**
- Create: `Prometheus-V2/backend/internal/platform/jobs/client.go`
- Create: `Prometheus-V2/backend/internal/platform/jobs/client_test.go`
- Modify: `Prometheus-V2/backend/cmd/prometheus/main.go`

- [ ] **Step 1: River + Migrations-Tooling hinzufuegen**

```bash
cd Prometheus-V2/backend
go get github.com/riverqueue/river@v0.13.0
go get github.com/riverqueue/river/riverdriver/riverpgxv5@v0.13.0
go get github.com/riverqueue/river/rivermigrate@v0.13.0
```

- [ ] **Step 2: Test schreiben (failing): Client kann gebaut werden**

Erstelle `Prometheus-V2/backend/internal/platform/jobs/client_test.go`:

```go
package jobs_test

import (
	"context"
	"os"
	"testing"

	"github.com/antigravity/prometheus-v2/internal/platform/db"
	"github.com/antigravity/prometheus-v2/internal/platform/jobs"
	"github.com/stretchr/testify/require"
)

func TestNewClient_RegistersWorkers(t *testing.T) {
	dsn := os.Getenv("PROMETHEUS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("PROMETHEUS_TEST_DATABASE_URL not set; skipping integration test")
	}

	ctx := context.Background()
	pool, err := db.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	require.NoError(t, jobs.MigrateUp(ctx, pool))

	client, err := jobs.NewClient(ctx, pool, jobs.NewWorkers())
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NoError(t, client.Stop(ctx))
}
```

- [ ] **Step 3: Client-Implementation**

Erstelle `Prometheus-V2/backend/internal/platform/jobs/client.go`:

```go
package jobs

import (
	"context"
	"fmt"

	"github.com/antigravity/prometheus-v2/internal/platform/db"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

const QueueDefault = "default"

type Client = river.Client[*pgxv5Tx]

type pgxv5Tx = riverpgxv5.Tx

func NewWorkers() *river.Workers {
	return river.NewWorkers()
}

func NewClient(ctx context.Context, pool *db.Pool, workers *river.Workers) (*river.Client[*pgxv5Tx], error) {
	driver := riverpgxv5.New(pool.Pool)
	cli, err := river.NewClient(driver, &river.Config{
		Workers: workers,
		Queues: map[string]river.QueueConfig{
			QueueDefault: {MaxWorkers: 4},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("init river client: %w", err)
	}
	if err := cli.Start(ctx); err != nil {
		return nil, fmt.Errorf("start river client: %w", err)
	}
	return cli, nil
}

func MigrateUp(ctx context.Context, pool *db.Pool) error {
	migrator, err := rivermigrate.New(riverpgxv5.New(pool.Pool), nil)
	if err != nil {
		return fmt.Errorf("init river migrator: %w", err)
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return fmt.Errorf("river migrate up: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Test ausfuehren**

```bash
go test ./internal/platform/jobs/...
```

Expected: SKIP wenn `PROMETHEUS_TEST_DATABASE_URL` nicht gesetzt ist; ansonsten PASS.

- [ ] **Step 5: main.go um River-Wiring erweitern**

Modifiziere `Prometheus-V2/backend/cmd/prometheus/main.go` — ergaenze unmittelbar nach dem `pool`-Setup:

```go
if err := jobs.MigrateUp(ctx, pool); err != nil {
    logger.Error("river migrations failed", slog.Any("error", err))
    os.Exit(1)
}

riverClient, err := jobs.NewClient(ctx, pool, jobs.NewWorkers())
if err != nil {
    logger.Error("river client init failed", slog.Any("error", err))
    os.Exit(1)
}
defer func() {
    if err := riverClient.Stop(context.Background()); err != nil {
        logger.Error("river client stop failed", slog.Any("error", err))
    }
}()
```

Ergaenze den Import `"github.com/antigravity/prometheus-v2/internal/platform/jobs"` oben.

- [ ] **Step 6: Build verifizieren**

```bash
go build ./cmd/prometheus
```

Expected: keine Compile-Fehler.

- [ ] **Step 7: Commit**

```bash
cd "D:/Dokumente & Inhalte/Programmierung/Prometheus/Prometheus-Vita"
git add Prometheus-V2/backend/internal/platform/jobs/ Prometheus-V2/backend/cmd/prometheus/main.go Prometheus-V2/backend/go.mod Prometheus-V2/backend/go.sum
git commit -m "feat(v2): wire river job queue with default queue"
```

---

## Task 6: Redis-Client + Prometheus-Metriken

**Files:**
- Create: `Prometheus-V2/backend/internal/platform/redis/redis.go`
- Create: `Prometheus-V2/backend/internal/platform/metrics/metrics.go`
- Modify: `Prometheus-V2/backend/internal/http/server.go`
- Modify: `Prometheus-V2/backend/cmd/prometheus/main.go`

- [ ] **Step 1: Libraries hinzufuegen**

```bash
cd Prometheus-V2/backend
go get github.com/redis/go-redis/v9@v9.7.0
go get github.com/prometheus/client_golang/prometheus@v1.20.0
go get github.com/prometheus/client_golang/prometheus/promhttp@v1.20.0
```

- [ ] **Step 2: Redis-Wrapper**

Erstelle `Prometheus-V2/backend/internal/platform/redis/redis.go`:

```go
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	*redis.Client
}

func New(ctx context.Context, url string) (*Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	rdb := redis.NewClient(opts)

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return &Client{Client: rdb}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return c.Client.Ping(pingCtx).Err()
}
```

- [ ] **Step 3: Prometheus-Registry und Standard-Metriken**

Erstelle `Prometheus-V2/backend/internal/platform/metrics/metrics.go`:

```go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

type Registry struct {
	*prometheus.Registry
	HTTPRequestsTotal     *prometheus.CounterVec
	HTTPRequestDuration   *prometheus.HistogramVec
}

func New() *Registry {
	r := prometheus.NewRegistry()
	r.MustRegister(collectors.NewGoCollector())
	r.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	httpReqs := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "HTTP requests labeled by method, route and status code.",
	}, []string{"method", "route", "status"})
	r.MustRegister(httpReqs)

	httpDur := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Latency of HTTP requests labeled by method and route.",
		Buckets: prometheus.ExponentialBuckets(0.005, 2, 12),
	}, []string{"method", "route"})
	r.MustRegister(httpDur)

	return &Registry{
		Registry:            r,
		HTTPRequestsTotal:   httpReqs,
		HTTPRequestDuration: httpDur,
	}
}
```

- [ ] **Step 4: `/metrics`-Endpoint registrieren**

Modifiziere `Prometheus-V2/backend/internal/http/server.go` — ergaenze in `Deps` und `NewServer`:

```go
// Imports ergaenzen:
//   "github.com/antigravity/prometheus-v2/internal/platform/metrics"
//   "github.com/prometheus/client_golang/prometheus/promhttp"

type Deps struct {
	Logger  *slog.Logger
	DB      DBPinger
	Redis   RedisPinger
	Metrics *metrics.Registry
}

// in NewServer ergaenzen, nach RegisterHealth:
if deps.Metrics != nil {
	e.GET("/metrics", echo.WrapHandler(promhttp.HandlerFor(deps.Metrics.Registry, promhttp.HandlerOpts{})))

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			route := c.Path()
			if route == "" {
				route = "unknown"
			}
			status := c.Response().Status
			deps.Metrics.HTTPRequestsTotal.WithLabelValues(c.Request().Method, route, http.StatusText(status)).Inc()
			deps.Metrics.HTTPRequestDuration.WithLabelValues(c.Request().Method, route).Observe(time.Since(start).Seconds())
			return err
		}
	})
}
```

- [ ] **Step 5: main.go um Redis und Metriken erweitern**

Modifiziere `Prometheus-V2/backend/cmd/prometheus/main.go` — ergaenze nach River-Setup:

```go
redisClient, err := redis.New(ctx, cfg.RedisURL)
if err != nil {
    logger.Error("redis init failed", slog.Any("error", err))
    os.Exit(1)
}
defer redisClient.Close()

reg := metrics.New()
```

Aendere den `httpserver.NewServer`-Aufruf:

```go
server := httpserver.NewServer(httpserver.Deps{
    Logger:  logger,
    DB:      pool,
    Redis:   redisClient,
    Metrics: reg,
})
```

Ergaenze die Imports:

```go
"github.com/antigravity/prometheus-v2/internal/platform/metrics"
"github.com/antigravity/prometheus-v2/internal/platform/redis"
```

- [ ] **Step 6: Build pruefen**

```bash
go build ./cmd/prometheus
```

Expected: keine Compile-Fehler.

- [ ] **Step 7: Commit**

```bash
cd "D:/Dokumente & Inhalte/Programmierung/Prometheus/Prometheus-Vita"
git add Prometheus-V2/backend/
git commit -m "feat(v2): wire redis client and prometheus metrics endpoint"
```

---

## Task 7: Frontend-Bootstrap — Vite + React + Tailwind + shadcn-Baseline

**Files:**
- Create: `Prometheus-V2/frontend/package.json`
- Create: `Prometheus-V2/frontend/tsconfig.json`
- Create: `Prometheus-V2/frontend/tsconfig.node.json`
- Create: `Prometheus-V2/frontend/vite.config.ts`
- Create: `Prometheus-V2/frontend/index.html`
- Create: `Prometheus-V2/frontend/postcss.config.cjs`
- Create: `Prometheus-V2/frontend/tailwind.config.ts`
- Create: `Prometheus-V2/frontend/components.json`
- Create: `Prometheus-V2/frontend/eslint.config.js`
- Create: `Prometheus-V2/frontend/vitest.config.ts`
- Create: `Prometheus-V2/frontend/src/index.css`
- Create: `Prometheus-V2/frontend/src/main.tsx`
- Create: `Prometheus-V2/frontend/src/lib/utils.ts`

- [ ] **Step 1: package.json**

Erstelle `Prometheus-V2/frontend/package.json`:

```json
{
  "name": "prometheus-v2-frontend",
  "private": true,
  "version": "0.1.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc -b && vite build",
    "preview": "vite preview",
    "lint": "eslint .",
    "test": "vitest run",
    "test:watch": "vitest",
    "type-check": "tsc -b --noEmit",
    "generate:api": "openapi-typescript ../backend/api/openapi.yaml -o src/lib/api/schema.d.ts"
  },
  "dependencies": {
    "@tanstack/react-query": "^5.59.0",
    "@tanstack/react-query-devtools": "^5.59.0",
    "@tanstack/react-router": "^1.95.0",
    "class-variance-authority": "^0.7.1",
    "clsx": "^2.1.1",
    "lucide-react": "^0.460.0",
    "openapi-fetch": "^0.13.0",
    "react": "^19.0.0",
    "react-dom": "^19.0.0",
    "tailwind-merge": "^2.5.0"
  },
  "devDependencies": {
    "@tanstack/router-plugin": "^1.95.0",
    "@testing-library/jest-dom": "^6.6.0",
    "@testing-library/react": "^16.1.0",
    "@types/node": "^22.10.0",
    "@types/react": "^19.0.0",
    "@types/react-dom": "^19.0.0",
    "@vitejs/plugin-react-swc": "^3.7.0",
    "autoprefixer": "^10.4.20",
    "eslint": "^9.17.0",
    "eslint-plugin-react-hooks": "^5.1.0",
    "eslint-plugin-react-refresh": "^0.4.14",
    "globals": "^15.14.0",
    "happy-dom": "^15.11.0",
    "openapi-typescript": "^7.4.0",
    "postcss": "^8.4.49",
    "tailwindcss": "^4.0.0-beta.7",
    "typescript": "^5.7.0",
    "typescript-eslint": "^8.18.0",
    "vite": "^6.0.0",
    "vitest": "^2.1.8"
  }
}
```

- [ ] **Step 2: TypeScript-Konfigurationen**

Erstelle `Prometheus-V2/frontend/tsconfig.json`:

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "useDefineForClassFields": true,
    "lib": ["ES2022", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "moduleDetection": "force",
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "baseUrl": ".",
    "paths": { "@/*": ["./src/*"] }
  },
  "include": ["src"],
  "references": [{ "path": "./tsconfig.node.json" }]
}
```

Erstelle `Prometheus-V2/frontend/tsconfig.node.json`:

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "lib": ["ES2023"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowSyntheticDefaultImports": true,
    "strict": true,
    "noEmit": true
  },
  "include": ["vite.config.ts", "vitest.config.ts", "tailwind.config.ts"]
}
```

- [ ] **Step 3: Vite-Konfiguration**

Erstelle `Prometheus-V2/frontend/vite.config.ts`:

```ts
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react-swc";
import { TanStackRouterVite } from "@tanstack/router-plugin/vite";
import path from "node:path";

export default defineConfig({
  plugins: [TanStackRouterVite({ routesDirectory: "src/routes" }), react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
    sourcemap: true,
  },
  server: {
    port: 5173,
    proxy: {
      "/api": "http://localhost:8180",
    },
  },
});
```

- [ ] **Step 4: index.html**

Erstelle `Prometheus-V2/frontend/index.html`:

```html
<!doctype html>
<html lang="de">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Prometheus V2</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

- [ ] **Step 5: PostCSS- und Tailwind-Konfig**

Erstelle `Prometheus-V2/frontend/postcss.config.cjs`:

```js
module.exports = {
  plugins: {
    "@tailwindcss/postcss": {},
    autoprefixer: {},
  },
};
```

Erstelle `Prometheus-V2/frontend/tailwind.config.ts`:

```ts
import type { Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {},
  },
  plugins: [],
} satisfies Config;
```

- [ ] **Step 6: shadcn-Konfig**

Erstelle `Prometheus-V2/frontend/components.json`:

```json
{
  "$schema": "https://ui.shadcn.com/schema.json",
  "style": "new-york",
  "rsc": false,
  "tsx": true,
  "tailwind": {
    "config": "tailwind.config.ts",
    "css": "src/index.css",
    "baseColor": "neutral",
    "cssVariables": true
  },
  "aliases": {
    "components": "@/components",
    "utils": "@/lib/utils",
    "lib": "@/lib",
    "ui": "@/components/ui"
  }
}
```

- [ ] **Step 7: ESLint-Konfig**

Erstelle `Prometheus-V2/frontend/eslint.config.js`:

```js
import js from "@eslint/js";
import globals from "globals";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import tseslint from "typescript-eslint";

export default tseslint.config(
  { ignores: ["dist", "src/lib/api/schema.d.ts", "src/routeTree.gen.ts"] },
  {
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    files: ["**/*.{ts,tsx}"],
    languageOptions: {
      ecmaVersion: 2022,
      globals: globals.browser,
    },
    plugins: {
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      "react-refresh/only-export-components": ["warn", { allowConstantExport: true }],
      "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_" }],
    },
  }
);
```

- [ ] **Step 8: Vitest-Konfig**

Erstelle `Prometheus-V2/frontend/vitest.config.ts`:

```ts
import { defineConfig, mergeConfig } from "vitest/config";
import viteConfig from "./vite.config";

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      environment: "happy-dom",
      globals: true,
      setupFiles: ["./src/test-setup.ts"],
    },
  })
);
```

- [ ] **Step 9: index.css**

Erstelle `Prometheus-V2/frontend/src/index.css`:

```css
@import "tailwindcss";

:root {
  color-scheme: light dark;
}

html, body, #root {
  height: 100%;
}

body {
  margin: 0;
  font-family: ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, sans-serif;
  background: oklch(0.98 0.005 240);
  color: oklch(0.18 0.02 240);
}

@media (prefers-color-scheme: dark) {
  body {
    background: oklch(0.16 0.01 240);
    color: oklch(0.95 0.01 240);
  }
}
```

- [ ] **Step 10: Utility-Funktionen**

Erstelle `Prometheus-V2/frontend/src/lib/utils.ts`:

```ts
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
```

- [ ] **Step 11: Test-Setup-Datei**

Erstelle `Prometheus-V2/frontend/src/test-setup.ts`:

```ts
import "@testing-library/jest-dom/vitest";
```

- [ ] **Step 12: Dependencies installieren**

```bash
cd Prometheus-V2/frontend
npm install
```

Expected: `node_modules/` wird angelegt, `package-lock.json` wird erstellt, keine Fehler.

- [ ] **Step 13: Frontend-Build pruefen (Sanity)**

Erstelle vorlaeufig minimal `Prometheus-V2/frontend/src/main.tsx`:

```tsx
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";

const Root = () => <main className="p-8 text-2xl">Prometheus V2 — Skeleton</main>;

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <Root />
  </StrictMode>
);
```

```bash
npm run type-check
npm run build
```

Expected: tsc bestanden, `dist/` wird angelegt, `dist/index.html` und `dist/assets/...` existieren.

- [ ] **Step 14: Commit**

```bash
cd "D:/Dokumente & Inhalte/Programmierung/Prometheus/Prometheus-Vita"
git add Prometheus-V2/frontend/
git commit -m "feat(v2): bootstrap vite+react+tailwind frontend skeleton"
```

---

## Task 8: Frontend — TanStack Router, Query und API-Client

**Files:**
- Create: `Prometheus-V2/frontend/src/routes/__root.tsx`
- Create: `Prometheus-V2/frontend/src/routes/index.tsx`
- Create: `Prometheus-V2/frontend/src/lib/query.ts`
- Create: `Prometheus-V2/frontend/src/lib/api/client.ts`
- Create: `Prometheus-V2/frontend/src/lib/api/schema.d.ts` (generiert)
- Create: `Prometheus-V2/frontend/src/components/health-status.tsx`
- Create: `Prometheus-V2/frontend/src/components/health-status.test.tsx`
- Modify: `Prometheus-V2/frontend/src/main.tsx`

- [ ] **Step 1: API-Schema generieren**

```bash
cd Prometheus-V2/frontend
npm run generate:api
```

Expected: erstellt `src/lib/api/schema.d.ts` mit Types fuer `paths` und `components` aus `../backend/api/openapi.yaml`.

- [ ] **Step 2: API-Client**

Erstelle `Prometheus-V2/frontend/src/lib/api/client.ts`:

```ts
import createClient from "openapi-fetch";
import type { paths } from "./schema";

export const api = createClient<paths>({
  baseUrl: "/api/v1",
});

export type SystemHealth = NonNullable<
  paths["/system/health"]["get"]["responses"]["200"]["content"]["application/json"]
>;
```

- [ ] **Step 3: TanStack Query-Setup**

Erstelle `Prometheus-V2/frontend/src/lib/query.ts`:

```ts
import { QueryClient } from "@tanstack/react-query";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: (failureCount, error) => {
        if (error instanceof Response && error.status >= 400 && error.status < 500) return false;
        return failureCount < 2;
      },
      refetchOnWindowFocus: false,
    },
    mutations: {
      retry: false,
    },
  },
});
```

- [ ] **Step 4: TanStack Router root + Index-Route**

Erstelle `Prometheus-V2/frontend/src/routes/__root.tsx`:

```tsx
import { createRootRoute, Outlet } from "@tanstack/react-router";
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools";

export const Route = createRootRoute({
  component: () => (
    <div className="min-h-full">
      <Outlet />
      {import.meta.env.DEV && <TanStackRouterDevtools position="bottom-right" />}
    </div>
  ),
});
```

(`@tanstack/react-router-devtools` ist Teil des Plugins; bei Installation faellt es ggf. unter `@tanstack/router-devtools`. Falls Compile fehlschlaegt: `npm install --save-dev @tanstack/react-router-devtools` nachinstallieren.)

Erstelle `Prometheus-V2/frontend/src/routes/index.tsx`:

```tsx
import { createFileRoute } from "@tanstack/react-router";
import { HealthStatus } from "@/components/health-status";

export const Route = createFileRoute("/")({
  component: HomeRoute,
});

function HomeRoute() {
  return (
    <main className="mx-auto max-w-3xl p-8">
      <h1 className="text-3xl font-semibold tracking-tight">Prometheus V2</h1>
      <p className="mt-2 text-base text-muted-foreground">Skeleton — Backend antwortet:</p>
      <div className="mt-6">
        <HealthStatus />
      </div>
    </main>
  );
}
```

- [ ] **Step 5: HealthStatus-Component**

Erstelle `Prometheus-V2/frontend/src/components/health-status.tsx`:

```tsx
import { useQuery } from "@tanstack/react-query";
import { api, type SystemHealth } from "@/lib/api/client";

export function HealthStatus() {
  const { data, isLoading, error } = useQuery({
    queryKey: ["system-health"],
    queryFn: async (): Promise<SystemHealth> => {
      const { data, error } = await api.GET("/system/health");
      if (error || !data) {
        throw new Error("Backend health check failed");
      }
      return data;
    },
  });

  if (isLoading) {
    return <p data-testid="health-status">Lade...</p>;
  }
  if (error || !data) {
    return <p data-testid="health-status" className="text-red-600">Backend nicht erreichbar.</p>;
  }
  return (
    <p data-testid="health-status" className="font-mono text-sm">
      status={data.status} version={data.version}
    </p>
  );
}
```

- [ ] **Step 6: HealthStatus-Test schreiben (failing first)**

Erstelle `Prometheus-V2/frontend/src/components/health-status.test.tsx`:

```tsx
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { HealthStatus } from "./health-status";

vi.mock("@/lib/api/client", () => ({
  api: {
    GET: vi.fn(),
  },
}));

import { api } from "@/lib/api/client";

describe("HealthStatus", () => {
  let qc: QueryClient;
  beforeEach(() => {
    qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    vi.clearAllMocks();
  });

  function renderWith() {
    return render(
      <QueryClientProvider client={qc}>
        <HealthStatus />
      </QueryClientProvider>
    );
  }

  it("renders status and version on success", async () => {
    (api.GET as ReturnType<typeof vi.fn>).mockResolvedValue({
      data: { status: "ok", version: "0.1.0", request_id: "rid" },
      error: undefined,
    });

    renderWith();
    await waitFor(() => {
      expect(screen.getByTestId("health-status").textContent).toContain("status=ok");
      expect(screen.getByTestId("health-status").textContent).toContain("version=0.1.0");
    });
  });

  it("renders error state when backend fails", async () => {
    (api.GET as ReturnType<typeof vi.fn>).mockResolvedValue({
      data: undefined,
      error: { code: "X", message: "down", request_id: "rid" },
    });

    renderWith();
    await waitFor(() => {
      expect(screen.getByTestId("health-status").textContent).toContain("Backend nicht erreichbar");
    });
  });
});
```

- [ ] **Step 7: Tests laufen lassen — muessen bestehen**

```bash
cd Prometheus-V2/frontend
npm run test
```

Expected: beide Tests PASS.

- [ ] **Step 8: main.tsx mit Router + QueryClient**

Modifiziere `Prometheus-V2/frontend/src/main.tsx` komplett:

```tsx
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { RouterProvider, createRouter } from "@tanstack/react-router";
import { QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { routeTree } from "./routeTree.gen";
import { queryClient } from "./lib/query";
import "./index.css";

const router = createRouter({ routeTree });

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
      {import.meta.env.DEV && <ReactQueryDevtools initialIsOpen={false} />}
    </QueryClientProvider>
  </StrictMode>
);
```

- [ ] **Step 9: Build verifizieren**

```bash
npm run type-check
npm run build
```

Expected: Build erfolgreich. `routeTree.gen.ts` wird vom TanStack-Plugin automatisch erstellt; falls nicht, einmal `npm run dev` kurz starten und mit Strg+C abbrechen, dann erneut bauen.

- [ ] **Step 10: Commit**

```bash
cd "D:/Dokumente & Inhalte/Programmierung/Prometheus/Prometheus-Vita"
git add Prometheus-V2/frontend/src/ Prometheus-V2/frontend/package.json Prometheus-V2/frontend/package-lock.json
git commit -m "feat(v2): wire tanstack router, query and openapi-fetch client"
```

---

## Task 9: Frontend ins Go-Binary einbetten

**Files:**
- Create: `Prometheus-V2/backend/internal/web/embed.go`
- Modify: `Prometheus-V2/backend/internal/http/server.go`

- [ ] **Step 1: Embed-Wrapper erstellen**

Erstelle `Prometheus-V2/backend/internal/web/embed.go`:

```go
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

//go:embed all:dist
var distFS embed.FS

func RegisterStatic(e *echo.Echo) error {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return err
	}
	staticHandler := http.FileServer(http.FS(sub))

	e.GET("/*", func(c echo.Context) error {
		path := c.Request().URL.Path
		if strings.HasPrefix(path, "/api/") || path == "/healthz" || path == "/readyz" || path == "/metrics" {
			return echo.ErrNotFound
		}

		if !hasStaticFile(sub, strings.TrimPrefix(path, "/")) {
			c.Request().URL.Path = "/"
		}
		staticHandler.ServeHTTP(c.Response(), c.Request())
		return nil
	})
	return nil
}

func hasStaticFile(fsys fs.FS, name string) bool {
	if name == "" {
		return true
	}
	_, err := fs.Stat(fsys, name)
	return err == nil
}
```

- [ ] **Step 2: Placeholder-`dist`-Ordner**

Damit `go:embed all:dist` beim Compile auch ohne ausgefuehrten Frontend-Build greift, lege einen Platzhalter an:

```bash
mkdir -p Prometheus-V2/backend/internal/web/dist
echo '<!doctype html><html><body>Build the frontend with `make build-frontend` first.</body></html>' > Prometheus-V2/backend/internal/web/dist/index.html
```

- [ ] **Step 3: Server-Wiring**

Modifiziere `Prometheus-V2/backend/internal/http/server.go` — ergaenze am Ende von `NewServer` (vor dem `return e`):

```go
if err := web.RegisterStatic(e); err != nil {
    deps.Logger.Error("static asset registration failed", slog.Any("error", err))
}
```

Ergaenze den Import:

```go
"github.com/antigravity/prometheus-v2/internal/web"
```

- [ ] **Step 4: Build verifizieren**

```bash
cd Prometheus-V2/backend
go build ./cmd/prometheus
./prometheus &
sleep 1
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8180/
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8180/api/v1/system/health
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8180/healthz
kill %1
```

Expected: drei `200`-Antworten.

- [ ] **Step 5: Commit**

```bash
cd "D:/Dokumente & Inhalte/Programmierung/Prometheus/Prometheus-Vita"
git add Prometheus-V2/backend/internal/web/ Prometheus-V2/backend/internal/http/server.go
git commit -m "feat(v2): embed frontend dist into single binary"
```

---

## Task 10: Make-Pipeline (verify, generate, build, dev)

**Files:**
- Create: `Prometheus-V2/Makefile`
- Create: `Prometheus-V2/backend/Makefile`
- Create: `Prometheus-V2/backend/.golangci.yml`

- [ ] **Step 1: golangci-lint-Konfig**

Erstelle `Prometheus-V2/backend/.golangci.yml`:

```yaml
linters:
  enable:
    - errcheck
    - govet
    - staticcheck
    - ineffassign
    - unused
    - gofmt
    - goimports

issues:
  exclude-dirs:
    - internal/api
    - internal/db/repo
```

- [ ] **Step 2: Backend-Makefile**

Erstelle `Prometheus-V2/backend/Makefile`:

```makefile
.PHONY: tools fmt lint test test-integration generate sqlc oapi build verify clean

GO ?= go
SQLC ?= sqlc
OAPI ?= oapi-codegen

tools:
	$(GO) install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	$(GO) install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	$(GO) install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

fmt:
	$(GO) fmt ./...

lint:
	golangci-lint run ./...

test:
	$(GO) test -race -cover ./...

test-integration:
	$(GO) test -race -cover -tags=integration ./...

generate: sqlc oapi

sqlc:
	$(SQLC) generate

oapi:
	$(OAPI) -config oapi-codegen.yaml api/openapi.yaml

build:
	$(GO) build -trimpath -ldflags="-s -w" -o bin/prometheus ./cmd/prometheus

verify: fmt lint test

clean:
	rm -rf bin/ internal/api/openapi.gen.go internal/db/repo/*.sql.go internal/db/repo/db.go internal/db/repo/models.go
```

- [ ] **Step 3: Root-Makefile**

Erstelle `Prometheus-V2/Makefile`:

```makefile
.PHONY: tools generate verify build dev dev-up dev-down clean

tools:
	$(MAKE) -C backend tools
	cd frontend && npm install

generate:
	$(MAKE) -C backend generate
	cd frontend && npm run generate:api

verify:
	$(MAKE) -C backend verify
	cd frontend && npm run type-check && npm run lint && npm run test

build-frontend:
	cd frontend && npm run build
	rm -rf backend/internal/web/dist
	cp -r frontend/dist backend/internal/web/dist

build: build-frontend
	$(MAKE) -C backend build

dev-up:
	docker compose up -d postgres redis

dev-down:
	docker compose down

dev:
	@echo "Run two terminals:"
	@echo "  1) cd backend && go run ./cmd/prometheus"
	@echo "  2) cd frontend && npm run dev"

clean:
	$(MAKE) -C backend clean
	rm -rf frontend/dist frontend/node_modules backend/internal/web/dist
```

- [ ] **Step 4: Verify-Schritt manuell ausfuehren**

```bash
cd Prometheus-V2
make tools
make generate
make verify
```

Expected:

- `make tools` installiert die CLIs (sqlc, oapi-codegen, migrate, golangci-lint, npm-Pakete).
- `make generate` regeneriert OpenAPI/sqlc-Code und das TS-Schema.
- `make verify` laeuft `go fmt`, `golangci-lint`, `go test`, `tsc`, `eslint`, `vitest` — alle Schritte muessen gruen sein. Falls golangci-lint Warnungen auf generierten Files meldet, ist der `exclude-dirs`-Eintrag im `.golangci.yml` zu pruefen.

- [ ] **Step 5: Commit**

```bash
cd "D:/Dokumente & Inhalte/Programmierung/Prometheus/Prometheus-Vita"
git add Prometheus-V2/Makefile Prometheus-V2/backend/Makefile Prometheus-V2/backend/.golangci.yml
git commit -m "feat(v2): add make pipeline for tools, generate, verify and build"
```

---

## Task 11: docker-compose-Setup und Smoke-Test

**Files:**
- Create: `Prometheus-V2/docker-compose.yml`
- Create: `Prometheus-V2/Dockerfile`
- Create: `Prometheus-V2/.env.example`

- [ ] **Step 1: Dockerfile**

Erstelle `Prometheus-V2/Dockerfile`:

```dockerfile
# syntax=docker/dockerfile:1
FROM node:20-alpine AS frontend-build
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
COPY backend/api ../backend/api
RUN npm run generate:api
RUN npm run build

FROM golang:1.23-alpine AS backend-build
WORKDIR /app
RUN apk add --no-cache make
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend-build /app/dist ./internal/web/dist
RUN go build -trimpath -ldflags="-s -w" -o /out/prometheus ./cmd/prometheus

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata nmap openssh-client iproute2
RUN adduser -D -u 10001 prometheus
USER prometheus
WORKDIR /app
COPY --from=backend-build /out/prometheus /app/prometheus
COPY --from=backend-build /app/db/migrations /app/db/migrations
EXPOSE 8180
ENTRYPOINT ["/app/prometheus"]
```

- [ ] **Step 2: docker-compose.yml**

Erstelle `Prometheus-V2/docker-compose.yml`:

```yaml
name: prometheus-v2

services:
  postgres:
    image: postgres:17-alpine
    environment:
      POSTGRES_USER: prometheus
      POSTGRES_PASSWORD: prometheus
      POSTGRES_DB: prometheus_v2
    ports:
      - "5433:5432"
    volumes:
      - postgres-v2:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U prometheus -d prometheus_v2"]
      interval: 5s
      timeout: 3s
      retries: 10

  redis:
    image: redis:7-alpine
    ports:
      - "6380:6379"
    volumes:
      - redis-v2:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 10

  backend:
    build:
      context: .
      dockerfile: Dockerfile
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    environment:
      PROMETHEUS_HTTP_ADDR: ":8180"
      PROMETHEUS_LOG_LEVEL: "info"
      PROMETHEUS_DATABASE_URL: "postgres://prometheus:prometheus@postgres:5432/prometheus_v2?sslmode=disable&search_path=prom_v2"
      PROMETHEUS_REDIS_URL: "redis://redis:6379/0"
    ports:
      - "8180:8180"
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8180/healthz"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  postgres-v2:
  redis-v2:
```

- [ ] **Step 3: Beispiel-Env**

Erstelle `Prometheus-V2/.env.example`:

```
PROMETHEUS_HTTP_ADDR=:8180
PROMETHEUS_LOG_LEVEL=info
PROMETHEUS_DATABASE_URL=postgres://prometheus:prometheus@localhost:5433/prometheus_v2?sslmode=disable&search_path=prom_v2
PROMETHEUS_REDIS_URL=redis://localhost:6380/0
```

- [ ] **Step 4: Smoke-Test des kompletten Stacks**

```bash
cd Prometheus-V2
docker compose build
docker compose up -d
sleep 15
curl -s http://localhost:8180/healthz
curl -s http://localhost:8180/api/v1/system/health
curl -s http://localhost:8180/metrics | head -5
docker compose logs backend --tail=20
docker compose down
```

Expected:

- `docker compose build` baut das Image erfolgreich.
- `/healthz` antwortet `{"status":"ok"}`.
- `/api/v1/system/health` antwortet mit `status`, `version`, `request_id`.
- `/metrics` zeigt Prometheus-Metriken (`# HELP go_gc_duration_seconds ...`).
- Backend-Logs enthalten `http server starting` und keine Fehler.

- [ ] **Step 5: Commit**

```bash
cd "D:/Dokumente & Inhalte/Programmierung/Prometheus/Prometheus-Vita"
git add Prometheus-V2/docker-compose.yml Prometheus-V2/Dockerfile Prometheus-V2/.env.example
git commit -m "feat(v2): add dockerfile and compose stack with smoke test"
```

---

## Task 12: UI-Foundation — Design-Tokens, App-Shell, Shared Primitives

**Files:**
- Modify: `Prometheus-V2/frontend/src/index.css`
- Create: `Prometheus-V2/frontend/src/components/theme-provider.tsx`
- Create: `Prometheus-V2/frontend/src/components/theme-toggle.tsx`
- Create: `Prometheus-V2/frontend/src/components/ui/button.tsx`
- Create: `Prometheus-V2/frontend/src/components/ui/card.tsx`
- Create: `Prometheus-V2/frontend/src/components/ui/badge.tsx`
- Create: `Prometheus-V2/frontend/src/components/ui/status-badge.tsx`
- Create: `Prometheus-V2/frontend/src/components/ui/status-badge.test.tsx`
- Create: `Prometheus-V2/frontend/src/components/ui/kpi-card.tsx`
- Create: `Prometheus-V2/frontend/src/components/ui/action-card.tsx`
- Create: `Prometheus-V2/frontend/src/components/ui/feature-status-card.tsx`
- Create: `Prometheus-V2/frontend/src/components/ui/empty-state.tsx`
- Create: `Prometheus-V2/frontend/src/components/ui/error-state.tsx`
- Create: `Prometheus-V2/frontend/src/components/layout/app-shell.tsx`
- Create: `Prometheus-V2/frontend/src/components/layout/page-shell.tsx`
- Create: `Prometheus-V2/frontend/src/components/layout/sidebar.tsx`
- Create: `Prometheus-V2/frontend/src/components/layout/topbar.tsx`
- Modify: `Prometheus-V2/frontend/src/routes/__root.tsx`
- Modify: `Prometheus-V2/frontend/src/routes/index.tsx`

Diese Task etabliert die visuelle Identitaet ab Tag 1: ruhige, neutrale Basis mit gezielten Akzenten, klare Statussprache (ok/warning/critical/info/muted), App-Shell mit Sidebar+Topbar, Shared Primitives, die alle Folge-Domains nutzen werden. Keine Feature-Inhalte — nur das Geruest, das durchgehend gepflegt aussieht.

- [ ] **Step 1: Erweiterte Tailwind-Tokens und globale Stile**

Ueberschreibe `Prometheus-V2/frontend/src/index.css` komplett:

```css
@import "tailwindcss";

@theme {
  --color-background: oklch(0.98 0.005 240);
  --color-foreground: oklch(0.18 0.02 240);
  --color-card: oklch(1 0 0);
  --color-card-foreground: oklch(0.18 0.02 240);
  --color-muted: oklch(0.95 0.005 240);
  --color-muted-foreground: oklch(0.46 0.02 240);
  --color-border: oklch(0.9 0.01 240);
  --color-ring: oklch(0.6 0.05 240);
  --color-primary: oklch(0.45 0.16 250);
  --color-primary-foreground: oklch(0.98 0 0);
  --color-accent: oklch(0.92 0.01 240);
  --color-accent-foreground: oklch(0.18 0.02 240);

  --color-status-ok: oklch(0.65 0.16 150);
  --color-status-warning: oklch(0.75 0.16 70);
  --color-status-critical: oklch(0.6 0.22 25);
  --color-status-info: oklch(0.65 0.13 230);
  --color-status-muted: oklch(0.55 0.02 240);

  --radius-sm: 6px;
  --radius-md: 8px;
  --radius-lg: 12px;

  --shadow-elev1: 0 1px 2px oklch(0 0 0 / 0.04), 0 8px 20px oklch(0 0 0 / 0.04);
  --shadow-elev2: 0 4px 8px oklch(0 0 0 / 0.05), 0 18px 44px oklch(0 0 0 / 0.07);
}

@layer base {
  :root[data-theme="dark"] {
    --color-background: oklch(0.16 0.01 240);
    --color-foreground: oklch(0.95 0.01 240);
    --color-card: oklch(0.2 0.012 240);
    --color-card-foreground: oklch(0.95 0.01 240);
    --color-muted: oklch(0.24 0.012 240);
    --color-muted-foreground: oklch(0.7 0.02 240);
    --color-border: oklch(0.3 0.012 240);
    --color-ring: oklch(0.7 0.05 240);
    --color-primary: oklch(0.7 0.16 250);
    --color-primary-foreground: oklch(0.16 0.01 240);
    --color-accent: oklch(0.28 0.012 240);
    --color-accent-foreground: oklch(0.95 0.01 240);
  }

  html, body, #root { height: 100%; }
  body {
    margin: 0;
    background: var(--color-background);
    color: var(--color-foreground);
    font-family: ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, sans-serif;
    -webkit-font-smoothing: antialiased;
    text-rendering: optimizeLegibility;
  }
}

@layer utilities {
  .surface-panel {
    background: var(--color-card);
    color: var(--color-card-foreground);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-elev1);
  }
  .surface-panel-strong {
    background: var(--color-card);
    color: var(--color-card-foreground);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-elev2);
  }
  .status-accent-ok       { border-left: 4px solid var(--color-status-ok); }
  .status-accent-warning  { border-left: 4px solid var(--color-status-warning); }
  .status-accent-critical { border-left: 4px solid var(--color-status-critical); }
  .status-accent-info     { border-left: 4px solid var(--color-status-info); }
  .status-accent-muted    { border-left: 4px solid var(--color-status-muted); }
}
```

- [ ] **Step 2: Theme-Provider (Light/Dark/System)**

Erstelle `Prometheus-V2/frontend/src/components/theme-provider.tsx`:

```tsx
import { createContext, useContext, useEffect, useState, type ReactNode } from "react";

type Theme = "light" | "dark" | "system";

type ThemeContextValue = {
  theme: Theme;
  setTheme: (theme: Theme) => void;
  resolved: "light" | "dark";
};

const ThemeContext = createContext<ThemeContextValue | null>(null);
const STORAGE_KEY = "prometheus-v2-theme";

function resolveTheme(theme: Theme): "light" | "dark" {
  if (theme === "system") {
    return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  }
  return theme;
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<Theme>(() => {
    return (localStorage.getItem(STORAGE_KEY) as Theme | null) ?? "system";
  });
  const [resolved, setResolved] = useState<"light" | "dark">(() => resolveTheme(theme));

  useEffect(() => {
    const next = resolveTheme(theme);
    setResolved(next);
    document.documentElement.dataset.theme = next;
    localStorage.setItem(STORAGE_KEY, theme);
  }, [theme]);

  useEffect(() => {
    if (theme !== "system") return;
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => setResolved(media.matches ? "dark" : "light");
    media.addEventListener("change", onChange);
    return () => media.removeEventListener("change", onChange);
  }, [theme]);

  return (
    <ThemeContext.Provider value={{ theme, setTheme: setThemeState, resolved }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  const ctx = useContext(ThemeContext);
  if (!ctx) throw new Error("useTheme must be used within ThemeProvider");
  return ctx;
}
```

- [ ] **Step 3: Theme-Toggle**

Erstelle `Prometheus-V2/frontend/src/components/theme-toggle.tsx`:

```tsx
import { Monitor, Moon, Sun } from "lucide-react";
import { cn } from "@/lib/utils";
import { useTheme } from "./theme-provider";

export function ThemeToggle({ className }: { className?: string }) {
  const { theme, setTheme } = useTheme();
  const options: Array<{ value: "light" | "dark" | "system"; label: string; icon: typeof Sun }> = [
    { value: "light", label: "Hell", icon: Sun },
    { value: "system", label: "System", icon: Monitor },
    { value: "dark", label: "Dunkel", icon: Moon },
  ];

  return (
    <div className={cn("inline-flex items-center gap-1 rounded-full border border-border bg-card p-1", className)}>
      {options.map((opt) => {
        const Icon = opt.icon;
        const active = theme === opt.value;
        return (
          <button
            key={opt.value}
            type="button"
            onClick={() => setTheme(opt.value)}
            aria-label={opt.label}
            className={cn(
              "inline-flex h-7 w-7 items-center justify-center rounded-full transition",
              active
                ? "bg-primary text-primary-foreground"
                : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
            )}
          >
            <Icon className="h-3.5 w-3.5" />
          </button>
        );
      })}
    </div>
  );
}
```

- [ ] **Step 4: Button-Primitive**

Erstelle `Prometheus-V2/frontend/src/components/ui/button.tsx`:

```tsx
import { forwardRef, type ButtonHTMLAttributes } from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-50",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground hover:opacity-90",
        outline: "border border-border bg-card text-foreground hover:bg-accent",
        ghost: "text-foreground hover:bg-accent",
        destructive: "bg-[oklch(var(--color-status-critical))] text-white hover:opacity-90",
      },
      size: {
        sm: "h-8 px-3",
        md: "h-9 px-4",
        lg: "h-10 px-5",
        icon: "h-9 w-9",
      },
    },
    defaultVariants: { variant: "default", size: "md" },
  }
);

export interface ButtonProps
  extends ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, ...props }, ref) => (
    <button ref={ref} className={cn(buttonVariants({ variant, size }), className)} {...props} />
  )
);
Button.displayName = "Button";
```

- [ ] **Step 5: Card-Primitive**

Erstelle `Prometheus-V2/frontend/src/components/ui/card.tsx`:

```tsx
import { forwardRef, type HTMLAttributes } from "react";
import { cn } from "@/lib/utils";

export const Card = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement> & { hover?: boolean }>(
  ({ className, hover = false, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(
        "surface-panel transition",
        hover && "hover:-translate-y-0.5 hover:shadow-[var(--shadow-elev2)]",
        className
      )}
      {...props}
    />
  )
);
Card.displayName = "Card";

export const CardHeader = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn("flex flex-col gap-1.5 p-4", className)} {...props} />
  )
);
CardHeader.displayName = "CardHeader";

export const CardTitle = forwardRef<HTMLHeadingElement, HTMLAttributes<HTMLHeadingElement>>(
  ({ className, ...props }, ref) => (
    <h3 ref={ref} className={cn("text-base font-semibold leading-tight", className)} {...props} />
  )
);
CardTitle.displayName = "CardTitle";

export const CardContent = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn("flex flex-col gap-3 p-4 pt-0", className)} {...props} />
  )
);
CardContent.displayName = "CardContent";
```

- [ ] **Step 6: Badge-Primitive**

Erstelle `Prometheus-V2/frontend/src/components/ui/badge.tsx`:

```tsx
import { type HTMLAttributes } from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "inline-flex items-center gap-1 rounded-full border px-2.5 py-0.5 text-xs font-medium",
  {
    variants: {
      variant: {
        outline: "border-border bg-card text-foreground",
        muted: "border-border bg-muted text-muted-foreground",
        primary: "border-transparent bg-primary text-primary-foreground",
      },
    },
    defaultVariants: { variant: "outline" },
  }
);

export type BadgeProps = HTMLAttributes<HTMLDivElement> & VariantProps<typeof badgeVariants>;

export function Badge({ className, variant, ...props }: BadgeProps) {
  return <div className={cn(badgeVariants({ variant }), className)} {...props} />;
}
```

- [ ] **Step 7: StatusBadge mit Statussprache**

Erstelle `Prometheus-V2/frontend/src/components/ui/status-badge.tsx`:

```tsx
import { AlertTriangle, CheckCircle2, Clock, Info, XCircle, type LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";

export type StatusTone = "ok" | "warning" | "critical" | "info" | "muted";

const toneToken: Record<StatusTone, string> = {
  ok: "var(--color-status-ok)",
  warning: "var(--color-status-warning)",
  critical: "var(--color-status-critical)",
  info: "var(--color-status-info)",
  muted: "var(--color-status-muted)",
};

const toneIcon: Record<StatusTone, LucideIcon> = {
  ok: CheckCircle2,
  warning: AlertTriangle,
  critical: XCircle,
  info: Info,
  muted: Clock,
};

export interface StatusBadgeProps {
  tone: StatusTone;
  children: React.ReactNode;
  className?: string;
  withIcon?: boolean;
}

export function StatusBadge({ tone, children, className, withIcon = true }: StatusBadgeProps) {
  const Icon = toneIcon[tone];
  return (
    <span
      data-tone={tone}
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium",
        className
      )}
      style={{
        borderColor: `oklch(from ${toneToken[tone]} l c h / 0.4)`,
        backgroundColor: `oklch(from ${toneToken[tone]} l c h / 0.1)`,
        color: toneToken[tone],
      }}
    >
      {withIcon && <Icon className="h-3.5 w-3.5" />}
      {children}
    </span>
  );
}
```

- [ ] **Step 8: StatusBadge-Test**

Erstelle `Prometheus-V2/frontend/src/components/ui/status-badge.test.tsx`:

```tsx
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { StatusBadge } from "./status-badge";

describe("StatusBadge", () => {
  it("renders children with the chosen tone", () => {
    render(<StatusBadge tone="ok">Cluster operativ</StatusBadge>);
    const el = screen.getByText("Cluster operativ");
    expect(el).toBeInTheDocument();
    expect(el.getAttribute("data-tone")).toBe("ok");
  });

  it("renders without icon when withIcon=false", () => {
    render(
      <StatusBadge tone="warning" withIcon={false}>
        Achtung
      </StatusBadge>
    );
    const el = screen.getByText("Achtung");
    expect(el.querySelector("svg")).toBeNull();
  });
});
```

```bash
cd Prometheus-V2/frontend
npm run test
```

Expected: alle Frontend-Tests inkl. StatusBadge bestehen.

- [ ] **Step 9: KpiCard**

Erstelle `Prometheus-V2/frontend/src/components/ui/kpi-card.tsx`:

```tsx
import type { LucideIcon } from "lucide-react";
import { Card, CardContent } from "./card";
import { cn } from "@/lib/utils";

export type KpiTone = "neutral" | "primary" | "ok" | "warning" | "critical" | "info";

const toneIconBg: Record<KpiTone, string> = {
  neutral: "bg-muted text-muted-foreground",
  primary: "bg-primary/12 text-primary",
  ok: "bg-[oklch(from_var(--color-status-ok)_l_c_h_/_0.12)] text-[oklch(var(--color-status-ok))]",
  warning: "bg-[oklch(from_var(--color-status-warning)_l_c_h_/_0.12)] text-[oklch(var(--color-status-warning))]",
  critical: "bg-[oklch(from_var(--color-status-critical)_l_c_h_/_0.12)] text-[oklch(var(--color-status-critical))]",
  info: "bg-[oklch(from_var(--color-status-info)_l_c_h_/_0.12)] text-[oklch(var(--color-status-info))]",
};

export interface KpiCardProps {
  title: string;
  value: string | number;
  delta?: string;
  icon?: LucideIcon;
  tone?: KpiTone;
  className?: string;
}

export function KpiCard({ title, value, delta, icon: Icon, tone = "neutral", className }: KpiCardProps) {
  return (
    <Card className={cn("h-full", className)}>
      <CardContent className="flex items-start justify-between gap-3 p-4">
        <div className="min-w-0">
          <p className="truncate text-xs font-medium uppercase tracking-wide text-muted-foreground">{title}</p>
          <p className="mt-1 text-2xl font-semibold tabular-nums">{value}</p>
          {delta && <p className="mt-1 text-xs text-muted-foreground">{delta}</p>}
        </div>
        {Icon && (
          <div className={cn("flex h-10 w-10 shrink-0 items-center justify-center rounded-md", toneIconBg[tone])}>
            <Icon className="h-5 w-5" />
          </div>
        )}
      </CardContent>
    </Card>
  );
}
```

- [ ] **Step 10: ActionCard**

Erstelle `Prometheus-V2/frontend/src/components/ui/action-card.tsx`:

```tsx
import type { LucideIcon } from "lucide-react";
import { ArrowRight } from "lucide-react";
import { Link } from "@tanstack/react-router";
import { Card, CardContent } from "./card";
import { Button } from "./button";
import { StatusBadge, type StatusTone } from "./status-badge";
import { cn } from "@/lib/utils";

const accentClass: Record<StatusTone, string> = {
  ok: "status-accent-ok",
  warning: "status-accent-warning",
  critical: "status-accent-critical",
  info: "status-accent-info",
  muted: "status-accent-muted",
};

export interface ActionCardProps {
  tone: StatusTone;
  icon: LucideIcon;
  title: string;
  description: string;
  badge: string;
  href: string;
  actionLabel: string;
}

export function ActionCard({ tone, icon: Icon, title, description, badge, href, actionLabel }: ActionCardProps) {
  return (
    <Card hover className={cn("overflow-hidden", accentClass[tone])}>
      <CardContent className="flex h-full flex-col gap-4 p-4">
        <div className="flex items-start justify-between gap-3">
          <div className="flex min-w-0 items-start gap-3">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-muted">
              <Icon className="h-4 w-4" />
            </div>
            <div className="min-w-0">
              <h3 className="text-sm font-semibold">{title}</h3>
              <p className="mt-1 text-sm text-muted-foreground">{description}</p>
            </div>
          </div>
          <StatusBadge tone={tone}>{badge}</StatusBadge>
        </div>
        <Button asChild={false} variant="outline" size="sm" className="mt-auto w-fit">
          <Link to={href}>
            {actionLabel}
            <ArrowRight className="ml-2 h-4 w-4" />
          </Link>
        </Button>
      </CardContent>
    </Card>
  );
}
```

- [ ] **Step 11: FeatureStatusCard**

Erstelle `Prometheus-V2/frontend/src/components/ui/feature-status-card.tsx`:

```tsx
import type { LucideIcon } from "lucide-react";
import { RefreshCw } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "./card";
import { Button } from "./button";
import { StatusBadge, type StatusTone } from "./status-badge";

export interface FeatureStatusCardProps {
  title: string;
  description: string;
  icon: LucideIcon;
  tone: StatusTone;
  status: string;
  details?: React.ReactNode;
  actionLabel?: string;
  onAction?: () => void;
  pending?: boolean;
  error?: string | null;
}

export function FeatureStatusCard({
  title,
  description,
  icon: Icon,
  tone,
  status,
  details,
  actionLabel,
  onAction,
  pending,
  error,
}: FeatureStatusCardProps) {
  return (
    <Card>
      <CardHeader className="flex-row items-start justify-between gap-4 pb-3">
        <div className="flex items-start gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-md bg-muted">
            <Icon className="h-5 w-5" />
          </div>
          <div>
            <CardTitle>{title}</CardTitle>
            <p className="mt-1 text-sm text-muted-foreground">{description}</p>
          </div>
        </div>
        <StatusBadge tone={tone}>{status}</StatusBadge>
      </CardHeader>
      <CardContent>
        {details}
        {error && (
          <p className="rounded-md border border-[oklch(from_var(--color-status-critical)_l_c_h_/_0.4)] bg-[oklch(from_var(--color-status-critical)_l_c_h_/_0.08)] px-3 py-2 text-sm" style={{ color: "oklch(var(--color-status-critical))" }}>
            {error}
          </p>
        )}
        {actionLabel && onAction && (
          <Button variant="outline" size="sm" className="w-fit" onClick={onAction} disabled={pending}>
            {pending && <RefreshCw className="mr-2 h-4 w-4 animate-spin" />}
            {actionLabel}
          </Button>
        )}
      </CardContent>
    </Card>
  );
}
```

- [ ] **Step 12: EmptyState und ErrorState**

Erstelle `Prometheus-V2/frontend/src/components/ui/empty-state.tsx`:

```tsx
import type { LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";

export interface EmptyStateProps {
  icon: LucideIcon;
  title: string;
  description?: string;
  action?: React.ReactNode;
  className?: string;
}

export function EmptyState({ icon: Icon, title, description, action, className }: EmptyStateProps) {
  return (
    <div className={cn("flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-border bg-card p-8 text-center", className)}>
      <div className="flex h-10 w-10 items-center justify-center rounded-full bg-muted text-muted-foreground">
        <Icon className="h-5 w-5" />
      </div>
      <div>
        <p className="text-sm font-semibold">{title}</p>
        {description && <p className="mt-1 text-sm text-muted-foreground">{description}</p>}
      </div>
      {action}
    </div>
  );
}
```

Erstelle `Prometheus-V2/frontend/src/components/ui/error-state.tsx`:

```tsx
import { AlertTriangle, RefreshCw } from "lucide-react";
import { Button } from "./button";
import { cn } from "@/lib/utils";

export interface ErrorStateProps {
  title?: string;
  message: string;
  onRetry?: () => void;
  retryLabel?: string;
  className?: string;
}

export function ErrorState({
  title = "Etwas ist schiefgelaufen",
  message,
  onRetry,
  retryLabel = "Erneut versuchen",
  className,
}: ErrorStateProps) {
  return (
    <div
      className={cn(
        "flex flex-col items-start gap-3 rounded-lg border px-4 py-3 text-sm",
        className
      )}
      style={{
        borderColor: "oklch(from var(--color-status-critical) l c h / 0.4)",
        backgroundColor: "oklch(from var(--color-status-critical) l c h / 0.08)",
        color: "oklch(var(--color-status-critical))",
      }}
    >
      <div className="flex items-start gap-2">
        <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
        <div>
          <p className="font-semibold">{title}</p>
          <p className="mt-0.5 opacity-90">{message}</p>
        </div>
      </div>
      {onRetry && (
        <Button variant="outline" size="sm" onClick={onRetry}>
          <RefreshCw className="mr-2 h-3.5 w-3.5" />
          {retryLabel}
        </Button>
      )}
    </div>
  );
}
```

- [ ] **Step 13: Sidebar**

Erstelle `Prometheus-V2/frontend/src/components/layout/sidebar.tsx`:

```tsx
import { Activity, Boxes, Bell, Server, ShieldCheck, Wrench, ListChecks, Bot } from "lucide-react";
import { Link } from "@tanstack/react-router";
import { cn } from "@/lib/utils";

const navItems = [
  { to: "/", label: "Lagezentrum", icon: Activity },
  { to: "/hosts", label: "Hosts", icon: Server },
  { to: "/vms", label: "VMs", icon: Boxes },
  { to: "/migrations", label: "Migrationen", icon: Wrench },
  { to: "/backups", label: "Backups", icon: ShieldCheck },
  { to: "/notifications", label: "Notifications", icon: Bell },
  { to: "/agent", label: "Agent", icon: Bot },
  { to: "/tasks", label: "Task-Center", icon: ListChecks },
];

export function Sidebar({ className }: { className?: string }) {
  return (
    <aside className={cn("flex h-full w-60 shrink-0 flex-col gap-2 border-r border-border bg-card p-3", className)}>
      <div className="px-2 py-3">
        <p className="text-sm font-semibold tracking-tight">Prometheus V2</p>
        <p className="text-[10px] uppercase tracking-wide text-muted-foreground">Operations Cockpit</p>
      </div>
      <nav className="flex flex-col gap-0.5">
        {navItems.map((item) => {
          const Icon = item.icon;
          return (
            <Link
              key={item.to}
              to={item.to}
              activeProps={{ className: "bg-accent text-accent-foreground" }}
              inactiveProps={{ className: "text-muted-foreground hover:bg-muted hover:text-foreground" }}
              className="flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition"
            >
              <Icon className="h-4 w-4" />
              <span>{item.label}</span>
            </Link>
          );
        })}
      </nav>
    </aside>
  );
}
```

(Die hier referenzierten Routen — `/hosts`, `/vms`, etc. — werden in den Folge-Plaenen angelegt. In Plan 1 fuehren sie auf 404; das ist akzeptabel und macht die Sidebar visuell vollstaendig.)

- [ ] **Step 14: Topbar**

Erstelle `Prometheus-V2/frontend/src/components/layout/topbar.tsx`:

```tsx
import { ThemeToggle } from "@/components/theme-toggle";
import { StatusBadge } from "@/components/ui/status-badge";

export function Topbar() {
  return (
    <header className="flex h-14 items-center justify-between border-b border-border bg-card px-5">
      <div className="flex items-center gap-3">
        <StatusBadge tone="ok" withIcon>Live</StatusBadge>
        <p className="text-sm text-muted-foreground">Skeleton-Build</p>
      </div>
      <div className="flex items-center gap-3">
        <ThemeToggle />
      </div>
    </header>
  );
}
```

- [ ] **Step 15: AppShell**

Erstelle `Prometheus-V2/frontend/src/components/layout/app-shell.tsx`:

```tsx
import type { ReactNode } from "react";
import { Sidebar } from "./sidebar";
import { Topbar } from "./topbar";

export function AppShell({ children }: { children: ReactNode }) {
  return (
    <div className="flex h-full">
      <Sidebar />
      <div className="flex min-w-0 flex-1 flex-col">
        <Topbar />
        <main className="min-h-0 flex-1 overflow-auto bg-background p-6">{children}</main>
      </div>
    </div>
  );
}
```

- [ ] **Step 16: PageShell**

Erstelle `Prometheus-V2/frontend/src/components/layout/page-shell.tsx`:

```tsx
import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

export interface PageShellProps {
  title: string;
  description?: string;
  eyebrow?: string;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
}

export function PageShell({ title, description, eyebrow, actions, children, className }: PageShellProps) {
  return (
    <div className={cn("flex flex-col gap-6", className)}>
      <header className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          {eyebrow && (
            <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">{eyebrow}</p>
          )}
          <h1 className="text-2xl font-bold tracking-tight">{title}</h1>
          {description && (
            <p className="mt-1 max-w-3xl text-sm text-muted-foreground">{description}</p>
          )}
        </div>
        {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
      </header>
      {children}
    </div>
  );
}
```

- [ ] **Step 17: Root-Route mit ThemeProvider und AppShell**

Modifiziere `Prometheus-V2/frontend/src/routes/__root.tsx` komplett:

```tsx
import { createRootRoute, Outlet } from "@tanstack/react-router";
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools";
import { ThemeProvider } from "@/components/theme-provider";
import { AppShell } from "@/components/layout/app-shell";

export const Route = createRootRoute({
  component: () => (
    <ThemeProvider>
      <AppShell>
        <Outlet />
      </AppShell>
      {import.meta.env.DEV && <TanStackRouterDevtools position="bottom-right" />}
    </ThemeProvider>
  ),
});
```

- [ ] **Step 18: Index-Route nutzt PageShell und Primitives**

Modifiziere `Prometheus-V2/frontend/src/routes/index.tsx` komplett:

```tsx
import { createFileRoute } from "@tanstack/react-router";
import { Activity, Bell, Boxes, Server } from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import { KpiCard } from "@/components/ui/kpi-card";
import { FeatureStatusCard } from "@/components/ui/feature-status-card";
import { HealthStatus } from "@/components/health-status";

export const Route = createFileRoute("/")({
  component: HomeRoute,
});

function HomeRoute() {
  return (
    <PageShell
      title="Lagezentrum"
      description="Skeleton-Build von Prometheus V2. Domains werden in Folge-Plaenen angelegt."
      eyebrow="Operations"
    >
      <section className="surface-panel-strong p-5">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              Backend-Status
            </p>
            <h2 className="mt-1 text-xl font-semibold">Live-Verbindung</h2>
            <p className="mt-1 max-w-xl text-sm text-muted-foreground">
              Sobald Auth, Hosts und Realtime-Bus implementiert sind, fliessen hier echte Daten ein.
            </p>
          </div>
          <HealthStatus />
        </div>
      </section>

      <section className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <KpiCard title="Hosts" value="0" delta="kommt mit Plan 6" icon={Server} tone="neutral" />
        <KpiCard title="VMs" value="0" delta="kommt mit Plan 7" icon={Boxes} tone="neutral" />
        <KpiCard title="Aktive Tasks" value="0" delta="kommt mit Plan 14" icon={Activity} tone="neutral" />
        <KpiCard title="Notifications" value="0" delta="kommt mit Plan 13" icon={Bell} tone="neutral" />
      </section>

      <section className="grid gap-3 md:grid-cols-2">
        <FeatureStatusCard
          title="Skeleton bereit"
          description="Backend, Frontend und Build-Pipeline laufen."
          icon={Activity}
          tone="ok"
          status="Stabil"
          details={<p className="text-sm text-muted-foreground">Folge-Plaene fuellen die Domains.</p>}
        />
        <FeatureStatusCard
          title="Domains kommen"
          description="Auth, Audit, Realtime, Approval, Host, VM, ..."
          icon={Bell}
          tone="info"
          status="Geplant"
          details={<p className="text-sm text-muted-foreground">Siehe Plan-Reihenfolge im Skeleton-Plan.</p>}
        />
      </section>
    </PageShell>
  );
}
```

- [ ] **Step 19: Build und Tests verifizieren**

```bash
cd Prometheus-V2/frontend
npm run type-check
npm run lint
npm run test
npm run build
```

Expected: alle Schritte gruen.

- [ ] **Step 20: Smoke-Test im Browser**

```bash
cd Prometheus-V2/frontend
npm run dev
```

Im Browser `http://localhost:5173/` oeffnen. Erwartet:

- Sidebar links mit allen geplanten Domain-Links (klick fuehrt zu 404 — ok in Plan 1).
- Topbar mit Live-Badge und Theme-Toggle.
- Lagezentrum-Header, Status-Card, KPI-Reihe und zwei Feature-Status-Cards.
- Theme-Toggle wechselt zwischen Hell/System/Dunkel ohne Flicker.

Mit Strg+C beenden.

- [ ] **Step 21: Commit**

```bash
cd "D:/Dokumente & Inhalte/Programmierung/Prometheus/Prometheus-Vita"
git add Prometheus-V2/frontend/src/
git commit -m "feat(v2): establish ui foundation with design tokens, app-shell and primitives"
```

---

## Self-Review

Nach Abschluss aller Tasks die folgenden Pruefungen durchfuehren:

- [ ] **Spec-Coverage:**
  - Stack-Tabelle aus Spec ist abgedeckt: Vite, React, TanStack Router, TanStack Query, shadcn-Setup, Tailwind v4, Echo, sqlc, River, Redis, Prometheus-Metriken, Single-Binary-Embed, OpenAPI Spec-First mit oapi-codegen + openapi-typescript. Keine Domain-Module gebaut (per Plan-Scope korrekt).
  - UI-Foundation aus Spec-Sektion "Komponenten" abgedeckt: PageShell, StatusBadge, ActionCard, FeatureStatusCard, KpiCard, EmptyState, ErrorState, AppShell mit Sidebar und Topbar, ThemeProvider mit Light/Dark/System.
  - Keine Auth, kein Audit, kein Approval, kein Realtime-Bus — laut Plan-Scope absichtlich.
- [ ] **Build laeuft:** `cd Prometheus-V2 && make build` baut das Single-Binary.
- [ ] **Tests gruen:** `make verify` ist gruen.
- [ ] **Smoke gruen:** `docker compose up -d` plus `/healthz`, `/api/v1/system/health`, `/metrics` antworten 200.
- [ ] **Generated-Files unversioniert:** `internal/api/openapi.gen.go`, `internal/db/repo/*.sql.go`, `frontend/src/lib/api/schema.d.ts`, `frontend/src/routeTree.gen.ts` werden bei Bedarf regeneriert.
- [ ] **Verzeichnisstruktur** entspricht der Spec-Sektion "Repo-Layout".
- [ ] **Mehrfache Builds idempotent:** Wiederholtes `make generate && make build` produziert gleiche Outputs.

Wenn ein Punkt fehlschlaegt, das betroffene Step in den entsprechenden Tasks korrigieren, neu committen.

## Folge-Plaene

Nach Abschluss von Plan 1 (Skeleton) folgen in dieser Reihenfolge:

1. Auth-Domain
2. Audit-Domain
3. Realtime-Bus (SSE + WS-Hub)
4. Approval-Domain
5. Host-Domain (Proxmox-Adapter)
6. VM-Domain
7. Template-Domain
8. Console-Domain
9. Migration-Domain
10. Backup-Domain
11. Netscan-Domain
12. Notification-Domain
13. Task-Center
14. Agent-Domain
15. Data-Migrator (V1 → V2)
16. Cutover-Tooling

Jeder weitere Plan wird einzeln nach diesem Muster geschrieben und ausgefuehrt.
