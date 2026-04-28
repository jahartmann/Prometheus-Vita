# Prometheus V2 Auth-Domain Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Baue die Auth-Domain end-to-end: Users, Sessions, Passwords (bcrypt), JWT (HS256, 15min Access + HttpOnly-Refresh-Cookie), Permissions in JWT-Claims, RBAC-Middleware, Bootstrap-Admin-Seeder und einen minimalen Frontend-Login-Flow mit Token-Refresh-Interceptor und Route-Guard.

**Architecture:** Domain-Modul `internal/domain/auth/` enthaelt Service, Handlers, Middleware, Types, Permissions-Konstanten. Datenmodell: `users` + `sessions` (API-Keys + Service-Identities sind out-of-scope; kommen in einem Folge-Plan). Frontend: Zustand-Store fuer Access-Token im Memory, Refresh-Cookie HttpOnly, openapi-fetch-Middleware-Hook fuer 401-Recovery.

**Tech Stack:** Go (`golang-jwt/jwt/v5`, `golang.org/x/crypto/bcrypt`), pgx + sqlc, Echo, OpenAPI/oapi-codegen, React + TanStack Router/Query, Zustand, openapi-fetch.

---

## Scope Check

Plan 2 baut **nur die Auth-Maschinerie** plus den minimalen Login-Flow, der V2 nutzbar macht. **Out of scope** (in spaeteren Plaenen):

- API-Key-Erstellung/Verwaltung (Personal + Service-Identity) — kommt mit erstem Konsumenten (Agent oder Admin-UI)
- User-Registrierung / Self-Service-Profile — kommt mit Admin-UI
- Password-Reset / Recovery — kommt mit Notification-Domain
- Sessions-Listing / Revoke-UI — kommt mit Admin-UI
- 2FA — Schema vorbereiten, kein Flow
- OAuth/SSO — bewusst ausgeschlossen

Wenn ein Worker Tasks 1–16 nicht in einem fokussierten Pass durchziehen kann, splitten in Folge-Plan, **nicht** ueber den Auth-Scope hinaus erweitern.

## File Map

**Backend:**

- Create: `Prometheus-V2/backend/db/migrations/000002_users.up.sql`, `.down.sql`
- Create: `Prometheus-V2/backend/db/migrations/000003_sessions.up.sql`, `.down.sql`
- Create: `Prometheus-V2/backend/db/queries/users.sql`
- Create: `Prometheus-V2/backend/db/queries/sessions.sql`
- Create: `Prometheus-V2/backend/internal/domain/auth/api.go`
- Create: `Prometheus-V2/backend/internal/domain/auth/types.go`
- Create: `Prometheus-V2/backend/internal/domain/auth/permissions.go`
- Create: `Prometheus-V2/backend/internal/domain/auth/password.go`
- Create: `Prometheus-V2/backend/internal/domain/auth/password_test.go`
- Create: `Prometheus-V2/backend/internal/domain/auth/jwt.go`
- Create: `Prometheus-V2/backend/internal/domain/auth/jwt_test.go`
- Create: `Prometheus-V2/backend/internal/domain/auth/service.go`
- Create: `Prometheus-V2/backend/internal/domain/auth/service_test.go`
- Create: `Prometheus-V2/backend/internal/domain/auth/http.go`
- Create: `Prometheus-V2/backend/internal/domain/auth/middleware.go`
- Create: `Prometheus-V2/backend/internal/domain/auth/seed.go`
- Modify: `Prometheus-V2/backend/internal/config/config.go` (add JWT secret + bootstrap-admin envs)
- Modify: `Prometheus-V2/backend/internal/config/config_test.go`
- Modify: `Prometheus-V2/backend/internal/http/server.go` (wire auth deps)
- Modify: `Prometheus-V2/backend/cmd/prometheus/main.go` (wire auth + seeder)
- Modify: `Prometheus-V2/backend/api/openapi.yaml` (add auth schemas + endpoints)
- Modify: `Prometheus-V2/backend/sqlc.yaml` (no change unless needed)

**Frontend:**

- Create: `Prometheus-V2/frontend/src/lib/auth/store.ts` (Zustand)
- Create: `Prometheus-V2/frontend/src/lib/auth/client.ts` (login/refresh/logout/me helpers)
- Create: `Prometheus-V2/frontend/src/lib/auth/guard.tsx` (RouteGuard)
- Create: `Prometheus-V2/frontend/src/components/auth/login-form.tsx`
- Create: `Prometheus-V2/frontend/src/components/auth/login-form.test.tsx`
- Create: `Prometheus-V2/frontend/src/components/auth/user-menu.tsx`
- Create: `Prometheus-V2/frontend/src/routes/login.tsx`
- Modify: `Prometheus-V2/frontend/src/lib/api/client.ts` (add auth-aware middleware)
- Modify: `Prometheus-V2/frontend/src/routes/__root.tsx` (wrap Outlet in RouteGuard, render login route without AppShell)
- Modify: `Prometheus-V2/frontend/src/components/layout/topbar.tsx` (mount UserMenu)

---

## Task 1: Permissions Constants

**Files:**
- Create: `Prometheus-V2/backend/internal/domain/auth/permissions.go`

- [ ] **Step 1: Define permission constants and role-permissions map**

Create `Prometheus-V2/backend/internal/domain/auth/permissions.go`:

```go
package auth

// Role names. Persisted as text in users.role; bundle-of-permissions are
// resolved at login time via PermissionsForRole.
const (
	RoleViewer   = "viewer"
	RoleOperator = "operator"
	RoleAdmin    = "admin"
)

// Permission keys. Domain modules use these strings as middleware arguments.
// Keep the list lean: only permissions V2 actively enforces today. Add new
// ones as their first consumer lands.
const (
	PermAuthMe = "auth:me" // every authenticated user has this implicitly

	PermHostRead  = "host:read"
	PermHostWrite = "host:write"

	PermVMRead         = "vm:read"
	PermVMReadOwned    = "vm:read:owned"
	PermVMLifecycle    = "vm:lifecycle"
	PermVMLifecycleAny = "vm:lifecycle:any"

	PermAuditRead = "audit:read"

	PermAdminUserManage = "admin:user:manage"
	PermAdminRBACManage = "admin:rbac:manage"
	PermAdminSystem     = "admin:system"
)

// PermissionsForRole returns the permission set that a role grants. The
// returned slice is a fresh copy so callers can mutate or extend it.
func PermissionsForRole(role string) []string {
	switch role {
	case RoleViewer:
		return []string{
			PermAuthMe,
			PermHostRead,
			PermVMRead,
			PermVMReadOwned,
		}
	case RoleOperator:
		return []string{
			PermAuthMe,
			PermHostRead,
			PermVMRead,
			PermVMReadOwned,
			PermVMLifecycle,
		}
	case RoleAdmin:
		return []string{
			PermAuthMe,
			PermHostRead, PermHostWrite,
			PermVMRead, PermVMReadOwned,
			PermVMLifecycle, PermVMLifecycleAny,
			PermAuditRead,
			PermAdminUserManage, PermAdminRBACManage, PermAdminSystem,
		}
	default:
		return nil
	}
}

// HasPermission returns true if the granted slice contains the required
// permission. The implicit PermAuthMe is treated as always granted to any
// caller that has any other permission (i.e. authenticated user).
func HasPermission(granted []string, required string) bool {
	if required == PermAuthMe && len(granted) > 0 {
		return true
	}
	for _, p := range granted {
		if p == required {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Verify it compiles**

```bash
export PATH="/c/Program Files/Go/bin:/c/Users/Janik/go/bin:$PATH"
cd Prometheus-V2/backend
go build ./internal/domain/auth/...
```

Expected: success, no output.

- [ ] **Step 3: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add Prometheus-V2/backend/internal/domain/auth/permissions.go
git commit -m "feat(v2): define auth roles and permission constants"
```

---

## Task 2: User and Session Schema + sqlc Queries

**Files:**
- Create: `Prometheus-V2/backend/db/migrations/000002_users.up.sql`
- Create: `Prometheus-V2/backend/db/migrations/000002_users.down.sql`
- Create: `Prometheus-V2/backend/db/migrations/000003_sessions.up.sql`
- Create: `Prometheus-V2/backend/db/migrations/000003_sessions.down.sql`
- Create: `Prometheus-V2/backend/db/queries/users.sql`
- Create: `Prometheus-V2/backend/db/queries/sessions.sql`

- [ ] **Step 1: Users migration up**

Create `Prometheus-V2/backend/db/migrations/000002_users.up.sql`:

```sql
SET search_path TO prom_v2;

CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS users (
    id            uuid PRIMARY KEY,
    email         citext UNIQUE NOT NULL,
    name          text NOT NULL,
    password_hash text NOT NULL,
    role          text NOT NULL,
    enabled       boolean NOT NULL DEFAULT true,
    version       int NOT NULL DEFAULT 1,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_role_check CHECK (role IN ('viewer', 'operator', 'admin'))
);

CREATE INDEX IF NOT EXISTS users_role_idx ON users (role);
```

- [ ] **Step 2: Users migration down**

Create `Prometheus-V2/backend/db/migrations/000002_users.down.sql`:

```sql
DROP TABLE IF EXISTS prom_v2.users;
```

- [ ] **Step 3: Sessions migration up**

Create `Prometheus-V2/backend/db/migrations/000003_sessions.up.sql`:

```sql
SET search_path TO prom_v2;

CREATE TABLE IF NOT EXISTS sessions (
    id                  uuid PRIMARY KEY,
    user_id             uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash  text NOT NULL UNIQUE,
    user_agent          text,
    ip_address          inet,
    expires_at          timestamptz NOT NULL,
    revoked_at          timestamptz,
    last_seen_at        timestamptz NOT NULL DEFAULT now(),
    created_at          timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS sessions_user_active_idx ON sessions (user_id) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS sessions_expires_idx ON sessions (expires_at);
```

- [ ] **Step 4: Sessions migration down**

Create `Prometheus-V2/backend/db/migrations/000003_sessions.down.sql`:

```sql
DROP TABLE IF EXISTS prom_v2.sessions;
```

- [ ] **Step 5: User queries**

Create `Prometheus-V2/backend/db/queries/users.sql`:

```sql
-- name: GetUserByID :one
SELECT id, email, name, password_hash, role, enabled, version, created_at, updated_at
FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, email, name, password_hash, role, enabled, version, created_at, updated_at
FROM users WHERE email = $1;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CreateUser :one
INSERT INTO users (id, email, name, password_hash, role, enabled)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, email, name, password_hash, role, enabled, version, created_at, updated_at;

-- name: UpdateUserPasswordHash :exec
UPDATE users
SET password_hash = $2,
    version = version + 1,
    updated_at = now()
WHERE id = $1;
```

- [ ] **Step 6: Session queries**

Create `Prometheus-V2/backend/db/queries/sessions.sql`:

```sql
-- name: CreateSession :one
INSERT INTO sessions (id, user_id, refresh_token_hash, user_agent, ip_address, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, revoked_at, last_seen_at, created_at;

-- name: GetSessionByRefreshHash :one
SELECT id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, revoked_at, last_seen_at, created_at
FROM sessions
WHERE refresh_token_hash = $1;

-- name: TouchSession :exec
UPDATE sessions
SET last_seen_at = now()
WHERE id = $1;

-- name: RevokeSession :exec
UPDATE sessions
SET revoked_at = now()
WHERE id = $1;

-- name: RevokeUserSessions :exec
UPDATE sessions
SET revoked_at = now()
WHERE user_id = $1 AND revoked_at IS NULL;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE expires_at < now() OR (revoked_at IS NOT NULL AND revoked_at < now() - interval '7 days');
```

- [ ] **Step 7: Regenerate sqlc**

```bash
export PATH="/c/Program Files/Go/bin:/c/Users/Janik/go/bin:$PATH"
cd Prometheus-V2/backend
sqlc generate
go build ./...
```

Expected: sqlc produces `internal/db/repo/users.sql.go` and `sessions.sql.go`; build succeeds.

- [ ] **Step 8: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add Prometheus-V2/backend/db/migrations/ Prometheus-V2/backend/db/queries/
git commit -m "feat(v2): add users and sessions schema with sqlc queries"
```

---

## Task 3: Password Hashing

**Files:**
- Create: `Prometheus-V2/backend/internal/domain/auth/password.go`
- Create: `Prometheus-V2/backend/internal/domain/auth/password_test.go`

- [ ] **Step 1: Add bcrypt dep**

```bash
cd Prometheus-V2/backend
go get golang.org/x/crypto/bcrypt
```

- [ ] **Step 2: Write failing test**

Create `Prometheus-V2/backend/internal/domain/auth/password_test.go`:

```go
package auth_test

import (
	"testing"

	"github.com/antigravity/prometheus-v2/internal/domain/auth"
	"github.com/stretchr/testify/require"
)

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := auth.HashPassword("correct-horse-battery-staple")
	require.NoError(t, err)
	require.NotEmpty(t, hash)
	require.NotEqual(t, "correct-horse-battery-staple", hash)

	require.NoError(t, auth.VerifyPassword(hash, "correct-horse-battery-staple"))
	require.Error(t, auth.VerifyPassword(hash, "wrong-password"))
}

func TestHashPassword_RejectsTooShort(t *testing.T) {
	_, err := auth.HashPassword("short")
	require.Error(t, err)
}
```

- [ ] **Step 3: Run failing test**

```bash
cd Prometheus-V2/backend
go test ./internal/domain/auth/... -run TestHashAndVerifyPassword
```

Expected: FAIL — undefined `HashPassword` / `VerifyPassword`.

- [ ] **Step 4: Implement**

Create `Prometheus-V2/backend/internal/domain/auth/password.go`:

```go
package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost   = 12
	minPasswordLen = 8
)

var ErrPasswordTooShort = errors.New("password must be at least 8 characters")

// HashPassword returns a bcrypt hash of the plaintext password. Returns
// ErrPasswordTooShort for passwords shorter than minPasswordLen.
func HashPassword(plain string) (string, error) {
	if len(plain) < minPasswordLen {
		return "", ErrPasswordTooShort
	}
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(h), nil
}

// VerifyPassword returns nil if plain matches hash, or an error otherwise.
// The error is intentionally opaque so callers do not leak whether the
// failure was a wrong password or an invalid hash.
func VerifyPassword(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
```

- [ ] **Step 5: Run tests pass**

```bash
go test ./internal/domain/auth/... -run TestHashAndVerifyPassword -v
go test ./internal/domain/auth/... -run TestHashPassword_RejectsTooShort -v
```

Expected: both PASS.

- [ ] **Step 6: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add Prometheus-V2/backend/internal/domain/auth/password.go Prometheus-V2/backend/internal/domain/auth/password_test.go Prometheus-V2/backend/go.mod Prometheus-V2/backend/go.sum
git commit -m "feat(v2): add bcrypt password hashing in auth domain"
```

---

## Task 4: JWT Sign/Verify

**Files:**
- Create: `Prometheus-V2/backend/internal/domain/auth/jwt.go`
- Create: `Prometheus-V2/backend/internal/domain/auth/jwt_test.go`

- [ ] **Step 1: Add jwt dep**

```bash
cd Prometheus-V2/backend
go get github.com/golang-jwt/jwt/v5
```

- [ ] **Step 2: Write failing test**

Create `Prometheus-V2/backend/internal/domain/auth/jwt_test.go`:

```go
package auth_test

import (
	"testing"
	"time"

	"github.com/antigravity/prometheus-v2/internal/domain/auth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSignAndVerifyAccessToken(t *testing.T) {
	signer := auth.NewJWTSigner([]byte("test-secret-please-be-32-bytes-or-more!"), "prometheus-v2")
	userID := uuid.New()

	tok, err := signer.SignAccessToken(userID, "admin", []string{"host:read", "vm:read"}, 15*time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, tok)

	claims, err := signer.VerifyAccessToken(tok)
	require.NoError(t, err)
	require.Equal(t, userID, claims.UserID)
	require.Equal(t, "admin", claims.Role)
	require.ElementsMatch(t, []string{"host:read", "vm:read"}, claims.Permissions)
}

func TestVerifyAccessToken_RejectsExpired(t *testing.T) {
	signer := auth.NewJWTSigner([]byte("test-secret-please-be-32-bytes-or-more!"), "prometheus-v2")
	tok, err := signer.SignAccessToken(uuid.New(), "viewer", nil, -1*time.Minute)
	require.NoError(t, err)

	_, err = signer.VerifyAccessToken(tok)
	require.Error(t, err)
}

func TestVerifyAccessToken_RejectsWrongSecret(t *testing.T) {
	signer := auth.NewJWTSigner([]byte("test-secret-please-be-32-bytes-or-more!"), "prometheus-v2")
	tok, err := signer.SignAccessToken(uuid.New(), "admin", nil, 5*time.Minute)
	require.NoError(t, err)

	other := auth.NewJWTSigner([]byte("a-different-secret-also-32-or-more-yes!"), "prometheus-v2")
	_, err = other.VerifyAccessToken(tok)
	require.Error(t, err)
}
```

- [ ] **Step 3: Run failing test**

```bash
cd Prometheus-V2/backend
go test ./internal/domain/auth/... -run JWT
```

Expected: FAIL — undefined `JWTSigner` etc.

- [ ] **Step 4: Implement**

Create `Prometheus-V2/backend/internal/domain/auth/jwt.go`:

```go
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const minJWTSecretLen = 32

var (
	ErrJWTSecretTooShort = errors.New("jwt secret must be at least 32 bytes")
	ErrInvalidToken      = errors.New("invalid token")
)

// AccessClaims is the payload of a Prometheus V2 access token.
type AccessClaims struct {
	UserID      uuid.UUID `json:"uid"`
	Role        string    `json:"role"`
	Permissions []string  `json:"perms,omitempty"`
	jwt.RegisteredClaims
}

// JWTSigner signs and verifies access tokens with HS256.
type JWTSigner struct {
	secret []byte
	issuer string
}

// NewJWTSigner constructs a signer. The secret must be at least 32 bytes;
// shorter secrets will panic at first sign call to surface misconfiguration
// in main.go before any request is served.
func NewJWTSigner(secret []byte, issuer string) *JWTSigner {
	return &JWTSigner{secret: secret, issuer: issuer}
}

func (s *JWTSigner) SignAccessToken(userID uuid.UUID, role string, perms []string, ttl time.Duration) (string, error) {
	if len(s.secret) < minJWTSecretLen {
		return "", ErrJWTSecretTooShort
	}
	now := time.Now().UTC()
	claims := AccessClaims{
		UserID:      userID,
		Role:        role,
		Permissions: perms,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

func (s *JWTSigner) VerifyAccessToken(raw string) (*AccessClaims, error) {
	claims := &AccessClaims{}
	parsed, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if !parsed.Valid {
		return nil, ErrInvalidToken
	}
	if claims.Issuer != s.issuer {
		return nil, fmt.Errorf("%w: wrong issuer", ErrInvalidToken)
	}
	return claims, nil
}
```

- [ ] **Step 5: Run JWT tests**

```bash
go test ./internal/domain/auth/... -run JWT -v
```

Expected: 3 PASS.

- [ ] **Step 6: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add Prometheus-V2/backend/internal/domain/auth/jwt.go Prometheus-V2/backend/internal/domain/auth/jwt_test.go Prometheus-V2/backend/go.mod Prometheus-V2/backend/go.sum
git commit -m "feat(v2): add hs256 jwt signer with access claims"
```

---

## Task 5: Auth Domain Types and Reader Interface

**Files:**
- Create: `Prometheus-V2/backend/internal/domain/auth/types.go`
- Create: `Prometheus-V2/backend/internal/domain/auth/api.go`

- [ ] **Step 1: Domain types**

Create `Prometheus-V2/backend/internal/domain/auth/types.go`:

```go
package auth

import (
	"time"

	"github.com/google/uuid"
)

// User is the domain representation of an authenticated principal.
type User struct {
	ID           uuid.UUID
	Email        string
	Name         string
	Role         string
	Enabled      bool
	Version      int32
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Session represents a long-lived refresh-token-bound session.
type Session struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	UserAgent  string
	IPAddress  string
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	LastSeenAt time.Time
	CreatedAt  time.Time
}

// AuthTokens is the bundle returned to a client after login or refresh.
type AuthTokens struct {
	AccessToken    string
	RefreshToken   string // raw, set in HttpOnly cookie by handler
	AccessExpiry   time.Time
	RefreshExpiry  time.Time
	User           User
}
```

- [ ] **Step 2: Public Reader interface**

Create `Prometheus-V2/backend/internal/domain/auth/api.go`:

```go
package auth

import (
	"context"

	"github.com/google/uuid"
)

// Reader is the read-only surface other domains can depend on. They never
// import the service or repo directly.
type Reader interface {
	GetUser(ctx context.Context, id uuid.UUID) (*User, error)
}
```

- [ ] **Step 3: Verify build**

```bash
cd Prometheus-V2/backend
go build ./internal/domain/auth/...
```

Expected: success.

- [ ] **Step 4: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add Prometheus-V2/backend/internal/domain/auth/types.go Prometheus-V2/backend/internal/domain/auth/api.go
git commit -m "feat(v2): define auth domain types and reader interface"
```

---

## Task 6: Auth Service — Login, Refresh, Logout

**Files:**
- Create: `Prometheus-V2/backend/internal/domain/auth/service.go`
- Create: `Prometheus-V2/backend/internal/domain/auth/service_test.go`

- [ ] **Step 1: Helper for refresh-token hashing**

Append to `Prometheus-V2/backend/internal/domain/auth/jwt.go`:

```go
// HashRefreshToken returns a deterministic hash of a raw refresh token,
// suitable for indexed lookup. Refresh tokens are random 32-byte strings,
// not JWTs, so we use SHA-256 (fast lookup, no secret needed; the secret
// lives in the cookie itself).
func HashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// NewRefreshToken returns a fresh 32-byte random token, base64url-encoded.
func NewRefreshToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("rand refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
```

Update imports in `jwt.go` to include:
```go
import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)
```

- [ ] **Step 2: Service skeleton + interfaces**

Create `Prometheus-V2/backend/internal/domain/auth/service.go`:

```go
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/antigravity/prometheus-v2/internal/db/repo"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 30 * 24 * time.Hour
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserDisabled       = errors.New("user disabled")
	ErrSessionNotFound    = errors.New("session not found or revoked")
	ErrSessionExpired     = errors.New("session expired")
)

// Querier is the subset of sqlc queries the auth service needs. Defined
// here so service_test can mock it without depending on a full Postgres.
type Querier interface {
	GetUserByEmail(ctx context.Context, email string) (repo.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (repo.User, error)
	CountUsers(ctx context.Context) (int64, error)
	CreateUser(ctx context.Context, arg repo.CreateUserParams) (repo.User, error)

	CreateSession(ctx context.Context, arg repo.CreateSessionParams) (repo.Session, error)
	GetSessionByRefreshHash(ctx context.Context, refreshTokenHash string) (repo.Session, error)
	TouchSession(ctx context.Context, id uuid.UUID) error
	RevokeSession(ctx context.Context, id uuid.UUID) error
	RevokeUserSessions(ctx context.Context, userID uuid.UUID) error
}

// Service wires the auth-domain logic. Construct via NewService.
type Service struct {
	queries Querier
	signer  *JWTSigner
	now     func() time.Time
}

func NewService(q Querier, signer *JWTSigner) *Service {
	return &Service{queries: q, signer: signer, now: time.Now}
}

// LoginRequest is what handlers pass in.
type LoginRequest struct {
	Email     string
	Password  string
	UserAgent string
	IPAddress string
}

// Login validates credentials, creates a session and returns tokens.
func (s *Service) Login(ctx context.Context, req LoginRequest) (*AuthTokens, error) {
	u, err := s.queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	if !u.Enabled {
		return nil, ErrUserDisabled
	}
	if err := VerifyPassword(u.PasswordHash, req.Password); err != nil {
		return nil, ErrInvalidCredentials
	}
	return s.issueTokens(ctx, u, req.UserAgent, req.IPAddress)
}

// Refresh validates the raw refresh token, rotates it, and returns fresh tokens.
func (s *Service) Refresh(ctx context.Context, rawRefresh, userAgent, ipAddress string) (*AuthTokens, error) {
	hash := HashRefreshToken(rawRefresh)
	sess, err := s.queries.GetSessionByRefreshHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	if sess.RevokedAt.Valid {
		return nil, ErrSessionNotFound
	}
	if s.now().After(sess.ExpiresAt.Time) {
		return nil, ErrSessionExpired
	}
	// Rotate: revoke the old session, mint a new one.
	if err := s.queries.RevokeSession(ctx, sess.ID); err != nil {
		return nil, fmt.Errorf("revoke session: %w", err)
	}
	u, err := s.queries.GetUserByID(ctx, sess.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user for refresh: %w", err)
	}
	if !u.Enabled {
		return nil, ErrUserDisabled
	}
	return s.issueTokens(ctx, u, userAgent, ipAddress)
}

// Logout revokes the session bound to the given raw refresh token. Idempotent.
func (s *Service) Logout(ctx context.Context, rawRefresh string) error {
	hash := HashRefreshToken(rawRefresh)
	sess, err := s.queries.GetSessionByRefreshHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("get session: %w", err)
	}
	if sess.RevokedAt.Valid {
		return nil
	}
	if err := s.queries.RevokeSession(ctx, sess.ID); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

// GetUser implements the Reader interface for cross-domain consumers.
func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	u, err := s.queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	domain := userFromRepo(u)
	return &domain, nil
}

func (s *Service) issueTokens(ctx context.Context, u repo.User, userAgent, ipAddress string) (*AuthTokens, error) {
	access, err := s.signer.SignAccessToken(u.ID, u.Role, PermissionsForRole(u.Role), AccessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("sign access: %w", err)
	}
	refresh, err := NewRefreshToken()
	if err != nil {
		return nil, err
	}
	accessExp := s.now().Add(AccessTokenTTL)
	refreshExp := s.now().Add(RefreshTokenTTL)

	var ip pgtype.Text
	if ipAddress != "" {
		ip = pgtype.Text{String: ipAddress, Valid: true}
	}
	var ua pgtype.Text
	if userAgent != "" {
		ua = pgtype.Text{String: userAgent, Valid: true}
	}

	_, err = s.queries.CreateSession(ctx, repo.CreateSessionParams{
		ID:               uuid.New(),
		UserID:           u.ID,
		RefreshTokenHash: HashRefreshToken(refresh),
		UserAgent:        ua,
		IpAddress:        ip,
		ExpiresAt:        pgtype.Timestamptz{Time: refreshExp, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &AuthTokens{
		AccessToken:   access,
		RefreshToken:  refresh,
		AccessExpiry:  accessExp,
		RefreshExpiry: refreshExp,
		User:          userFromRepo(u),
	}, nil
}

func userFromRepo(u repo.User) User {
	return User{
		ID:        u.ID,
		Email:     string(u.Email),
		Name:      u.Name,
		Role:      u.Role,
		Enabled:   u.Enabled,
		Version:   u.Version,
		CreatedAt: u.CreatedAt.Time,
		UpdatedAt: u.UpdatedAt.Time,
	}
}
```

NOTE: The exact field names of `repo.User`, `repo.CreateSessionParams`, etc. depend on sqlc-generated output. If your sqlc generates `Email pgtype.Text` instead of `Email string`, adjust `userFromRepo` accordingly. Run `sqlc generate` and inspect `internal/db/repo/models.go` after Task 2 to know exact names.

If `repo.User.IpAddress` is generated as `netip.Addr` or different, adapt the IP parsing. The plan assumes pgtype.Text-style outputs (sqlc default for nullable text/inet); your sqlc.yaml has `emit_pointers_for_null_types: true` — that means nullable text becomes `*string` and inet becomes `*netip.Addr`. Adjust the helper accordingly. Treat the snippet above as a template, not literal.

- [ ] **Step 3: Write service test with mock**

Create `Prometheus-V2/backend/internal/domain/auth/service_test.go`:

```go
package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/antigravity/prometheus-v2/internal/db/repo"
	"github.com/antigravity/prometheus-v2/internal/domain/auth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

type mockQuerier struct {
	user             *repo.User
	getByEmailErr    error
	getByIDErr       error
	createSessionErr error
	getSessionErr    error
	session          *repo.Session
	revokedIDs       []uuid.UUID
}

func (m *mockQuerier) GetUserByEmail(ctx context.Context, email string) (repo.User, error) {
	if m.getByEmailErr != nil {
		return repo.User{}, m.getByEmailErr
	}
	if m.user == nil {
		return repo.User{}, pgx.ErrNoRows
	}
	return *m.user, nil
}

func (m *mockQuerier) GetUserByID(ctx context.Context, id uuid.UUID) (repo.User, error) {
	if m.getByIDErr != nil {
		return repo.User{}, m.getByIDErr
	}
	if m.user == nil {
		return repo.User{}, pgx.ErrNoRows
	}
	return *m.user, nil
}

func (m *mockQuerier) CountUsers(ctx context.Context) (int64, error) {
	if m.user != nil {
		return 1, nil
	}
	return 0, nil
}

func (m *mockQuerier) CreateUser(ctx context.Context, arg repo.CreateUserParams) (repo.User, error) {
	u := repo.User{ID: arg.ID, Email: arg.Email, Name: arg.Name, PasswordHash: arg.PasswordHash, Role: arg.Role, Enabled: arg.Enabled, Version: 1}
	m.user = &u
	return u, nil
}

func (m *mockQuerier) CreateSession(ctx context.Context, arg repo.CreateSessionParams) (repo.Session, error) {
	if m.createSessionErr != nil {
		return repo.Session{}, m.createSessionErr
	}
	s := repo.Session{ID: arg.ID, UserID: arg.UserID, RefreshTokenHash: arg.RefreshTokenHash, ExpiresAt: arg.ExpiresAt}
	m.session = &s
	return s, nil
}

func (m *mockQuerier) GetSessionByRefreshHash(ctx context.Context, hash string) (repo.Session, error) {
	if m.getSessionErr != nil {
		return repo.Session{}, m.getSessionErr
	}
	if m.session == nil || m.session.RefreshTokenHash != hash {
		return repo.Session{}, pgx.ErrNoRows
	}
	return *m.session, nil
}

func (m *mockQuerier) TouchSession(ctx context.Context, id uuid.UUID) error { return nil }

func (m *mockQuerier) RevokeSession(ctx context.Context, id uuid.UUID) error {
	m.revokedIDs = append(m.revokedIDs, id)
	if m.session != nil && m.session.ID == id {
		now := time.Now()
		m.session.RevokedAt = pgtype.Timestamptz{Time: now, Valid: true}
	}
	return nil
}

func (m *mockQuerier) RevokeUserSessions(ctx context.Context, userID uuid.UUID) error { return nil }

func newTestService(t *testing.T) (*auth.Service, *mockQuerier) {
	t.Helper()
	hash, err := auth.HashPassword("a-good-password-1234")
	require.NoError(t, err)
	q := &mockQuerier{
		user: &repo.User{
			ID:           uuid.New(),
			Email:        "alice@example.com",
			Name:         "Alice",
			PasswordHash: hash,
			Role:         "admin",
			Enabled:      true,
			Version:      1,
			CreatedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
			UpdatedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		},
	}
	signer := auth.NewJWTSigner([]byte("test-secret-please-be-32-bytes-or-more!"), "prometheus-v2")
	return auth.NewService(q, signer), q
}

func TestLogin_AcceptsValidCredentials(t *testing.T) {
	svc, _ := newTestService(t)
	tokens, err := svc.Login(context.Background(), auth.LoginRequest{Email: "alice@example.com", Password: "a-good-password-1234"})
	require.NoError(t, err)
	require.NotEmpty(t, tokens.AccessToken)
	require.NotEmpty(t, tokens.RefreshToken)
	require.Equal(t, "Alice", tokens.User.Name)
}

func TestLogin_RejectsBadPassword(t *testing.T) {
	svc, _ := newTestService(t)
	_, err := svc.Login(context.Background(), auth.LoginRequest{Email: "alice@example.com", Password: "wrong-password"})
	require.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

func TestRefresh_RotatesAndReturnsNewTokens(t *testing.T) {
	svc, q := newTestService(t)
	tokens, err := svc.Login(context.Background(), auth.LoginRequest{Email: "alice@example.com", Password: "a-good-password-1234"})
	require.NoError(t, err)

	fresh, err := svc.Refresh(context.Background(), tokens.RefreshToken, "", "")
	require.NoError(t, err)
	require.NotEqual(t, tokens.RefreshToken, fresh.RefreshToken)
	require.Len(t, q.revokedIDs, 1)
}

func TestLogout_RevokesSessionIdempotent(t *testing.T) {
	svc, q := newTestService(t)
	tokens, err := svc.Login(context.Background(), auth.LoginRequest{Email: "alice@example.com", Password: "a-good-password-1234"})
	require.NoError(t, err)

	require.NoError(t, svc.Logout(context.Background(), tokens.RefreshToken))
	require.NoError(t, svc.Logout(context.Background(), tokens.RefreshToken)) // idempotent
	require.Len(t, q.revokedIDs, 1)
}
```

NOTE: `repo.User`, `repo.Session` field names depend on actual sqlc output. Adjust mock-construction sites if sqlc renames `Email` to a pgtype variant. Fix mocks to compile, do not bend the service-under-test.

- [ ] **Step 4: Run tests**

```bash
cd Prometheus-V2/backend
sqlc generate
go test ./internal/domain/auth/... -v
```

Expected: All tests in package pass. If sqlc-field names diverge from the plan, adjust `service.go`'s `userFromRepo` and the mock to match before re-running.

- [ ] **Step 5: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add Prometheus-V2/backend/internal/domain/auth/service.go Prometheus-V2/backend/internal/domain/auth/service_test.go Prometheus-V2/backend/internal/domain/auth/jwt.go
git commit -m "feat(v2): implement auth login refresh and logout service"
```

---

## Task 7: OpenAPI Auth Endpoints

**Files:**
- Modify: `Prometheus-V2/backend/api/openapi.yaml`

- [ ] **Step 1: Extend openapi.yaml with auth schemas**

Modify `Prometheus-V2/backend/api/openapi.yaml`. Add to `paths:`:

```yaml
  /auth/login:
    post:
      summary: Log in with email + password
      operationId: login
      tags: [auth]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/Credentials"
      responses:
        "200":
          description: Authenticated
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/AuthResponse"
        "401":
          description: Invalid credentials
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Error"

  /auth/refresh:
    post:
      summary: Rotate session via refresh cookie
      operationId: refresh
      tags: [auth]
      responses:
        "200":
          description: New tokens
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/AuthResponse"
        "401":
          description: No or expired session
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Error"

  /auth/logout:
    post:
      summary: Revoke current session
      operationId: logout
      tags: [auth]
      responses:
        "204":
          description: Logged out

  /auth/me:
    get:
      summary: Current user
      operationId: getMe
      tags: [auth]
      responses:
        "200":
          description: Current user
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
        "401":
          description: Not authenticated
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Error"
```

Add to `components.schemas:` (alongside existing `SystemHealth` / `Error`):

```yaml
    Credentials:
      type: object
      required: [email, password]
      properties:
        email:
          type: string
          format: email
        password:
          type: string
          minLength: 8

    AuthResponse:
      type: object
      required: [access_token, access_expires_at, user]
      properties:
        access_token:
          type: string
        access_expires_at:
          type: string
          format: date-time
        user:
          $ref: "#/components/schemas/User"

    User:
      type: object
      required: [id, email, name, role, enabled]
      properties:
        id:
          type: string
          format: uuid
        email:
          type: string
        name:
          type: string
        role:
          type: string
          enum: [viewer, operator, admin]
        enabled:
          type: boolean
```

- [ ] **Step 2: Regenerate**

```bash
cd Prometheus-V2/backend
oapi-codegen -config oapi-codegen.yaml api/openapi.yaml
go build ./...
```

Expected: build succeeds (existing apiServer needs new methods — they will be added in Task 8 below).

NOTE: After regen the `apiServer` in `internal/http/server.go` is now missing `Login`, `Refresh`, `Logout`, `GetMe` methods. Build will FAIL. That's expected; Task 8 implements them.

- [ ] **Step 3: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add Prometheus-V2/backend/api/openapi.yaml
git commit -m "feat(v2): extend openapi spec with auth endpoints"
```

---

## Task 8: Auth HTTP Handlers

**Files:**
- Create: `Prometheus-V2/backend/internal/domain/auth/http.go`
- Modify: `Prometheus-V2/backend/internal/http/server.go` (delegate auth ops to handlers)

- [ ] **Step 1: HTTP handlers**

Create `Prometheus-V2/backend/internal/domain/auth/http.go`:

```go
package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/antigravity/prometheus-v2/internal/api"
	"github.com/labstack/echo/v4"
)

const refreshCookieName = "prometheus_v2_refresh"

// HTTPHandler wires HTTP handlers for the auth domain. Construct via
// NewHTTPHandler. RegisterAPI wires the operations into the generated
// ServerInterface; main.go composes this with other domain handlers.
type HTTPHandler struct {
	service *Service
	cookieDomain string
	cookieSecure bool
}

func NewHTTPHandler(s *Service, cookieDomain string, cookieSecure bool) *HTTPHandler {
	return &HTTPHandler{service: s, cookieDomain: cookieDomain, cookieSecure: cookieSecure}
}

func (h *HTTPHandler) Login(c echo.Context) error {
	var req api.Credentials
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	tokens, err := h.service.Login(c.Request().Context(), LoginRequest{
		Email:     string(req.Email),
		Password:  req.Password,
		UserAgent: c.Request().UserAgent(),
		IPAddress: c.RealIP(),
	})
	if err != nil {
		return mapAuthError(err)
	}
	h.setRefreshCookie(c, tokens.RefreshToken, tokens.RefreshExpiry)
	return c.JSON(http.StatusOK, tokensToResponse(tokens))
}

func (h *HTTPHandler) Refresh(c echo.Context) error {
	cookie, err := c.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "no refresh cookie")
	}
	tokens, err := h.service.Refresh(c.Request().Context(), cookie.Value, c.Request().UserAgent(), c.RealIP())
	if err != nil {
		h.clearRefreshCookie(c)
		return mapAuthError(err)
	}
	h.setRefreshCookie(c, tokens.RefreshToken, tokens.RefreshExpiry)
	return c.JSON(http.StatusOK, tokensToResponse(tokens))
}

func (h *HTTPHandler) Logout(c echo.Context) error {
	cookie, err := c.Cookie(refreshCookieName)
	if err == nil && cookie.Value != "" {
		_ = h.service.Logout(c.Request().Context(), cookie.Value)
	}
	h.clearRefreshCookie(c)
	return c.NoContent(http.StatusNoContent)
}

func (h *HTTPHandler) GetMe(c echo.Context) error {
	claims, ok := ClaimsFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "no claims")
	}
	u, err := h.service.GetUser(c.Request().Context(), claims.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not found")
	}
	return c.JSON(http.StatusOK, userToAPI(*u))
}

func (h *HTTPHandler) setRefreshCookie(c echo.Context, token string, exp time.Time) {
	c.SetCookie(&http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     "/api/v1/auth",
		Expires:  exp,
		Domain:   h.cookieDomain,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *HTTPHandler) clearRefreshCookie(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/api/v1/auth",
		MaxAge:   -1,
		Domain:   h.cookieDomain,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}

func mapAuthError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrSessionNotFound), errors.Is(err, ErrSessionExpired):
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, ErrUserDisabled):
		return echo.NewHTTPError(http.StatusForbidden, "user disabled")
	default:
		return echo.NewHTTPError(http.StatusInternalServerError, "auth failure")
	}
}

func tokensToResponse(t *AuthTokens) api.AuthResponse {
	return api.AuthResponse{
		AccessToken:     t.AccessToken,
		AccessExpiresAt: t.AccessExpiry,
		User:            userToAPI(t.User),
	}
}

func userToAPI(u User) api.User {
	return api.User{
		Id:      u.ID,
		Email:   u.Email,
		Name:    u.Name,
		Role:    api.UserRole(u.Role),
		Enabled: u.Enabled,
	}
}
```

NOTE: `api.UserRole` and `api.User`/`api.Credentials`/`api.AuthResponse` come from the regenerated `openapi.gen.go`. Their exact field types depend on oapi-codegen output. If `api.Credentials.Email` is `openapi_types.Email` rather than a string, the cast `string(req.Email)` works because of the underlying string type. If oapi-codegen produces `time.Time` vs. `*time.Time` for AccessExpiresAt, adjust accordingly.

- [ ] **Step 2: Wire into server.go**

Modify `Prometheus-V2/backend/internal/http/server.go`. Update `Deps`:

```go
type Deps struct {
	Logger  *slog.Logger
	DB      DBPinger
	Redis   RedisPinger
	Metrics *metrics.Registry
	Auth    AuthHandler
}

// AuthHandler is the small surface server.go uses to delegate auth ops.
type AuthHandler interface {
	Login(c echo.Context) error
	Refresh(c echo.Context) error
	Logout(c echo.Context) error
	GetMe(c echo.Context) error
}
```

Update `apiServer` to delegate:

```go
type apiServer struct {
	auth AuthHandler
}

func (a apiServer) GetSystemHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, api.SystemHealth{
		Status:    api.Ok,
		Version:   "0.1.0",
		RequestId: c.Response().Header().Get(echo.HeaderXRequestID),
	})
}

func (a apiServer) Login(c echo.Context) error  { return a.auth.Login(c) }
func (a apiServer) Refresh(c echo.Context) error { return a.auth.Refresh(c) }
func (a apiServer) Logout(c echo.Context) error  { return a.auth.Logout(c) }
func (a apiServer) GetMe(c echo.Context) error   { return a.auth.GetMe(c) }
```

Update `RegisterHandlersWithBaseURL` call to pass auth:

```go
api.RegisterHandlersWithBaseURL(e, &apiServer{auth: deps.Auth}, "/api/v1")
```

- [ ] **Step 3: Build verification**

```bash
cd Prometheus-V2/backend
go build ./...
```

Expected: build succeeds. Test wiring once `main.go` provides Auth in Task 12.

- [ ] **Step 4: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add Prometheus-V2/backend/internal/domain/auth/http.go Prometheus-V2/backend/internal/http/server.go
git commit -m "feat(v2): wire auth http handlers"
```

---

## Task 9: RequireAuth Middleware

**Files:**
- Create: `Prometheus-V2/backend/internal/domain/auth/middleware.go`

- [ ] **Step 1: Middleware**

Create `Prometheus-V2/backend/internal/domain/auth/middleware.go`:

```go
package auth

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

const claimsContextKey = "auth.claims"

// ClaimsFromContext returns the claims set by RequireAuth, or false if the
// request was not authenticated.
func ClaimsFromContext(c echo.Context) (*AccessClaims, bool) {
	v := c.Get(claimsContextKey)
	if v == nil {
		return nil, false
	}
	claims, ok := v.(*AccessClaims)
	return claims, ok
}

// RequireAuth returns a middleware that rejects requests without a valid
// Authorization: Bearer <jwt> header. The decoded claims land in the Echo
// context under claimsContextKey.
func RequireAuth(signer *JWTSigner) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Request().Header.Get("Authorization")
			const prefix = "Bearer "
			if !strings.HasPrefix(h, prefix) {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing bearer token")
			}
			claims, err := signer.VerifyAccessToken(strings.TrimPrefix(h, prefix))
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}
			c.Set(claimsContextKey, claims)
			return next(c)
		}
	}
}

// RequirePermission returns a middleware that rejects requests whose claims
// do not contain the required permission. Must be chained AFTER RequireAuth.
func RequirePermission(perm string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims, ok := ClaimsFromContext(c)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
			}
			if !HasPermission(claims.Permissions, perm) {
				return echo.NewHTTPError(http.StatusForbidden, "permission denied")
			}
			return next(c)
		}
	}
}
```

- [ ] **Step 2: Verify build**

```bash
cd Prometheus-V2/backend
go build ./...
go vet ./...
```

Expected: clean.

- [ ] **Step 3: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add Prometheus-V2/backend/internal/domain/auth/middleware.go
git commit -m "feat(v2): add require-auth and require-permission middleware"
```

---

## Task 10: Apply Middleware to /auth/me and /system/health

**Files:**
- Modify: `Prometheus-V2/backend/internal/http/server.go`

- [ ] **Step 1: Apply RequireAuth selectively**

Modify `Prometheus-V2/backend/internal/http/server.go`. The challenge: oapi-codegen registers ALL operations on the same router. We need to wrap the auth-required operations after registration, NOT block /auth/login or /auth/refresh.

Replace the `RegisterHandlersWithBaseURL` call with a manual registration block. After:

```go
v1 := e.Group("/api/v1")
v1.Use(oapimw.OapiRequestValidator(spec))
api.RegisterHandlersWithBaseURL(e, &apiServer{auth: deps.Auth}, "/api/v1")
```

Add (after the Register call):

```go
// Apply auth middleware to operations that require it. The generated
// router registers the route on `e` directly, so we pin the middleware
// at handler-function level via a wrapper. For the bearer-required
// endpoints we register a higher-priority parallel route on the same
// path that calls auth middleware before calling the generated apiServer.
auth := deps.Auth
if signer := deps.AuthSigner; signer != nil {
	requireAuth := authmw.RequireAuth(signer)
	e.GET("/api/v1/auth/me", requireAuth(auth.GetMe))
	e.GET("/api/v1/system/health", requireAuth(func(c echo.Context) error {
		return (&apiServer{auth: auth}).GetSystemHealth(c)
	}))
}
```

NOTE: Echo's last-registered route does not override a previously-registered one with the same method+path. To make the auth-protected path actually win, the cleanest fix is to NOT use `RegisterHandlersWithBaseURL` for those operations. The pragmatic approach: split the API into "public" and "protected" groups.

Alternative cleaner approach: register the public endpoints (login/refresh) on the v1 group as-is, then handle me and system/health via custom registration that includes the middleware.

Replace the generated registration entirely with manual registration:

```go
// Imports to add:
//   authmw "github.com/antigravity/prometheus-v2/internal/domain/auth"

// Inside NewServer, replace the api.RegisterHandlersWithBaseURL block with:
v1 := e.Group("/api/v1")
v1.Use(oapimw.OapiRequestValidator(spec))

server := &apiServer{auth: deps.Auth}
v1.POST("/auth/login", server.Login)
v1.POST("/auth/refresh", server.Refresh)
v1.POST("/auth/logout", server.Logout)

if deps.AuthSigner != nil {
	protected := v1.Group("")
	protected.Use(authmw.RequireAuth(deps.AuthSigner))
	protected.GET("/auth/me", server.GetMe)
	protected.GET("/system/health", server.GetSystemHealth)
} else {
	v1.GET("/auth/me", server.GetMe)
	v1.GET("/system/health", server.GetSystemHealth)
}
```

Update `Deps`:

```go
type Deps struct {
	Logger     *slog.Logger
	DB         DBPinger
	Redis      RedisPinger
	Metrics    *metrics.Registry
	Auth       AuthHandler
	AuthSigner *authmw.JWTSigner
}
```

Add the import for `authmw`:

```go
authmw "github.com/antigravity/prometheus-v2/internal/domain/auth"
```

- [ ] **Step 2: Build**

```bash
cd Prometheus-V2/backend
go build ./...
```

Expected: success.

- [ ] **Step 3: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add Prometheus-V2/backend/internal/http/server.go
git commit -m "feat(v2): protect /auth/me and /system/health with require-auth"
```

---

## Task 11: Bootstrap Admin Seeder + Config

**Files:**
- Modify: `Prometheus-V2/backend/internal/config/config.go`
- Modify: `Prometheus-V2/backend/internal/config/config_test.go`
- Create: `Prometheus-V2/backend/internal/domain/auth/seed.go`

- [ ] **Step 1: Extend config**

Modify `Prometheus-V2/backend/internal/config/config.go`. Add fields and env-loading:

```go
type Config struct {
	HTTPAddr           string
	LogLevel           string
	DatabaseURL        string
	RedisURL           string
	JWTSecret          string
	JWTIssuer          string
	CookieDomain       string
	CookieSecure       bool
	BootstrapAdminEmail    string
	BootstrapAdminPassword string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:    getenv("PROMETHEUS_HTTP_ADDR", ":8180"),
		LogLevel:    getenv("PROMETHEUS_LOG_LEVEL", "info"),
		DatabaseURL: getenv("PROMETHEUS_DATABASE_URL", "postgres://prometheus:prometheus@localhost:5432/prometheus_v2?sslmode=disable&search_path=prom_v2"),
		RedisURL:    getenv("PROMETHEUS_REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:   getenv("PROMETHEUS_JWT_SECRET", ""),
		JWTIssuer:   getenv("PROMETHEUS_JWT_ISSUER", "prometheus-v2"),
		CookieDomain: getenv("PROMETHEUS_COOKIE_DOMAIN", ""),
		CookieSecure: getenv("PROMETHEUS_COOKIE_SECURE", "false") == "true",
		BootstrapAdminEmail:    getenv("PROMETHEUS_BOOTSTRAP_ADMIN_EMAIL", ""),
		BootstrapAdminPassword: getenv("PROMETHEUS_BOOTSTRAP_ADMIN_PASSWORD", ""),
	}
	return cfg, nil
}
```

- [ ] **Step 2: Update config test**

Modify `Prometheus-V2/backend/internal/config/config_test.go`. Append:

```go
func TestLoad_AuthDefaults(t *testing.T) {
	t.Setenv("PROMETHEUS_JWT_SECRET", "")
	t.Setenv("PROMETHEUS_JWT_ISSUER", "")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Empty(t, cfg.JWTSecret)
	require.Equal(t, "prometheus-v2", cfg.JWTIssuer)
	require.False(t, cfg.CookieSecure)
}

func TestLoad_AuthOverrides(t *testing.T) {
	t.Setenv("PROMETHEUS_JWT_SECRET", "abc-secret")
	t.Setenv("PROMETHEUS_COOKIE_SECURE", "true")
	t.Setenv("PROMETHEUS_BOOTSTRAP_ADMIN_EMAIL", "admin@example.com")
	t.Setenv("PROMETHEUS_BOOTSTRAP_ADMIN_PASSWORD", "init-password")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, "abc-secret", cfg.JWTSecret)
	require.True(t, cfg.CookieSecure)
	require.Equal(t, "admin@example.com", cfg.BootstrapAdminEmail)
	require.Equal(t, "init-password", cfg.BootstrapAdminPassword)
}
```

- [ ] **Step 3: Bootstrap seeder**

Create `Prometheus-V2/backend/internal/domain/auth/seed.go`:

```go
package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/antigravity/prometheus-v2/internal/db/repo"
	"github.com/google/uuid"
)

// SeedBootstrapAdmin creates an admin user if no users exist yet. It is a
// no-op when at least one user already exists. Returns an error if email/
// password are empty AND no user exists; the operator must provide them
// for the first start.
func SeedBootstrapAdmin(ctx context.Context, q Querier, email, password string, logger *slog.Logger) error {
	count, err := q.CountUsers(ctx)
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return nil
	}
	if email == "" || password == "" {
		return errors.New("no users in database; set PROMETHEUS_BOOTSTRAP_ADMIN_EMAIL and PROMETHEUS_BOOTSTRAP_ADMIN_PASSWORD on first start")
	}
	hash, err := HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash bootstrap password: %w", err)
	}
	id := uuid.New()
	if _, err := q.CreateUser(ctx, repo.CreateUserParams{
		ID:           id,
		Email:        email,
		Name:         "Bootstrap Admin",
		PasswordHash: hash,
		Role:         RoleAdmin,
		Enabled:      true,
	}); err != nil {
		return fmt.Errorf("create bootstrap admin: %w", err)
	}
	logger.Info("bootstrap admin created", slog.String("email", email), slog.String("id", id.String()))
	return nil
}
```

NOTE: `repo.CreateUserParams.Email` will likely be `pgtype.Text` or `string` depending on sqlc output. If it's `pgtype.Text`, wrap: `Email: pgtype.Text{String: email, Valid: true}`. Adjust at compile time.

- [ ] **Step 4: Run config tests**

```bash
cd Prometheus-V2/backend
go test ./internal/config/...
```

Expected: PASS for all 4 tests.

- [ ] **Step 5: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add Prometheus-V2/backend/internal/config/ Prometheus-V2/backend/internal/domain/auth/seed.go
git commit -m "feat(v2): add auth config and bootstrap admin seeder"
```

---

## Task 12: Wire Auth into main.go

**Files:**
- Modify: `Prometheus-V2/backend/cmd/prometheus/main.go`

- [ ] **Step 1: Compose auth in main**

Modify `Prometheus-V2/backend/cmd/prometheus/main.go`. After the `metrics.New()` line, add:

```go
if cfg.JWTSecret == "" {
    logger.Error("PROMETHEUS_JWT_SECRET is required")
    os.Exit(1)
}
if len(cfg.JWTSecret) < 32 {
    logger.Error("PROMETHEUS_JWT_SECRET must be at least 32 bytes")
    os.Exit(1)
}

queries := repo.New(pool)
signer := auth.NewJWTSigner([]byte(cfg.JWTSecret), cfg.JWTIssuer)
authSvc := auth.NewService(queries, signer)
authHandler := auth.NewHTTPHandler(authSvc, cfg.CookieDomain, cfg.CookieSecure)

if err := auth.SeedBootstrapAdmin(ctx, queries, cfg.BootstrapAdminEmail, cfg.BootstrapAdminPassword, logger); err != nil {
    logger.Error("bootstrap admin seed failed", slog.Any("error", err))
    os.Exit(1)
}
```

Update the `httpserver.NewServer` call:

```go
server := httpserver.NewServer(httpserver.Deps{
    Logger:     logger,
    DB:         pool,
    Redis:      redisClient,
    Metrics:    reg,
    Auth:       authHandler,
    AuthSigner: signer,
})
```

Add imports:

```go
"github.com/antigravity/prometheus-v2/internal/db/repo"
"github.com/antigravity/prometheus-v2/internal/domain/auth"
```

NOTE: `repo.New(pool)` is the sqlc-generated constructor. Verify the exact name in `internal/db/repo/db.go` after sqlc generation; it's typically `repo.New` taking a `DBTX` interface satisfied by the pgxpool wrapper. If sqlc emits `repo.New(pool.Pool)` instead, use that.

- [ ] **Step 2: Build verification**

```bash
cd Prometheus-V2/backend
PROMETHEUS_JWT_SECRET="" go build ./cmd/prometheus
```

Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add Prometheus-V2/backend/cmd/prometheus/main.go
git commit -m "feat(v2): wire auth service handler signer and seeder into main"
```

---

## Task 13: Frontend Auth Client + Schema Regen

**Files:**
- Modify: `Prometheus-V2/frontend/src/lib/api/client.ts` (regen schema; add auth helpers)
- Create: `Prometheus-V2/frontend/src/lib/auth/client.ts`

- [ ] **Step 1: Regen openapi-typescript**

```bash
cd Prometheus-V2/frontend
node ./node_modules/openapi-typescript/bin/cli.js ../backend/api/openapi.yaml -o src/lib/api/schema.d.ts
```

Expected: schema.d.ts now contains auth paths and User/Credentials/AuthResponse types.

- [ ] **Step 2: Auth client helpers**

Create `Prometheus-V2/frontend/src/lib/auth/client.ts`:

```ts
import { api, ApiError } from "@/lib/api/client";
import type { paths } from "@/lib/api/schema";

export type Credentials = paths["/auth/login"]["post"]["requestBody"]["content"]["application/json"];
export type AuthResponse = NonNullable<paths["/auth/login"]["post"]["responses"]["200"]["content"]["application/json"]>;
export type User = NonNullable<paths["/auth/me"]["get"]["responses"]["200"]["content"]["application/json"]>;

export async function loginRequest(credentials: Credentials): Promise<AuthResponse> {
  const { data, error, response } = await api.POST("/auth/login", { body: credentials });
  if (error || !data) {
    throw new ApiError(response?.status ?? 0, error, "Login failed");
  }
  return data;
}

export async function refreshRequest(): Promise<AuthResponse> {
  const { data, error, response } = await api.POST("/auth/refresh", {});
  if (error || !data) {
    throw new ApiError(response?.status ?? 0, error, "Session refresh failed");
  }
  return data;
}

export async function logoutRequest(): Promise<void> {
  await api.POST("/auth/logout", {});
}

export async function getMeRequest(): Promise<User> {
  const { data, error, response } = await api.GET("/auth/me");
  if (error || !data) {
    throw new ApiError(response?.status ?? 0, error, "Not authenticated");
  }
  return data;
}
```

- [ ] **Step 3: Verify**

```bash
cd Prometheus-V2/frontend
node ./node_modules/typescript/bin/tsc --noEmit
```

Expected: clean.

- [ ] **Step 4: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add Prometheus-V2/frontend/src/lib/auth/client.ts
git commit -m "feat(v2): add frontend auth api helpers"
```

---

## Task 14: Frontend Auth Store + API Interceptor

**Files:**
- Create: `Prometheus-V2/frontend/src/lib/auth/store.ts`
- Modify: `Prometheus-V2/frontend/src/lib/api/client.ts`

- [ ] **Step 1: Add Zustand to package.json**

```bash
cd Prometheus-V2/frontend
node ./node_modules/.bin/npm.cmd install zustand@^5
```

NOTE: If npm shim fails due to path-with-`&`, use:

```bash
& "C:\Program Files\nodejs\npm.cmd" install zustand@^5
```

Expected: zustand added to dependencies.

- [ ] **Step 2: Auth store**

Create `Prometheus-V2/frontend/src/lib/auth/store.ts`:

```ts
import { create } from "zustand";
import type { User } from "./client";

type AuthState = {
  accessToken: string | null;
  user: User | null;
  status: "anonymous" | "authenticated" | "refreshing";
  setSession: (token: string, user: User) => void;
  clearSession: () => void;
  setStatus: (status: AuthState["status"]) => void;
};

export const useAuthStore = create<AuthState>((set) => ({
  accessToken: null,
  user: null,
  status: "anonymous",
  setSession: (token, user) => set({ accessToken: token, user, status: "authenticated" }),
  clearSession: () => set({ accessToken: null, user: null, status: "anonymous" }),
  setStatus: (status) => set({ status }),
}));

// Read-only access for non-React code (api interceptor).
export function getAccessToken(): string | null {
  return useAuthStore.getState().accessToken;
}
```

- [ ] **Step 3: Wire bearer-token middleware in api client**

Modify `Prometheus-V2/frontend/src/lib/api/client.ts`. Add after `api` is created:

```ts
import { getAccessToken } from "@/lib/auth/store";

api.use({
  onRequest({ request }) {
    const token = getAccessToken();
    if (token) {
      request.headers.set("Authorization", `Bearer ${token}`);
    }
    return request;
  },
});
```

The full updated client.ts:

```ts
import createClient from "openapi-fetch";
import type { paths, components } from "./schema";
import { getAccessToken } from "@/lib/auth/store";

export const api = createClient<paths>({
  baseUrl: "/api/v1",
});

api.use({
  onRequest({ request }) {
    const token = getAccessToken();
    if (token) {
      request.headers.set("Authorization", `Bearer ${token}`);
    }
    return request;
  },
});

export type SystemHealth = NonNullable<
  paths["/system/health"]["get"]["responses"]["200"]["content"]["application/json"]
>;

export type ApiErrorEnvelope = components["schemas"]["Error"];

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly requestId: string;
  readonly details?: Record<string, unknown>;

  constructor(status: number, envelope: ApiErrorEnvelope | undefined, fallback: string) {
    super(envelope?.message ?? fallback);
    this.name = "ApiError";
    this.status = status;
    this.code = envelope?.code ?? "UNKNOWN";
    this.requestId = envelope?.request_id ?? "";
    this.details = envelope?.details;
  }
}
```

NOTE: `api.use` is openapi-fetch v0.13+ middleware API. Verify the function exists in installed version; if not, upgrade openapi-fetch to >=0.13 and re-run npm install.

- [ ] **Step 4: Verify**

```bash
cd Prometheus-V2/frontend
node ./node_modules/typescript/bin/tsc --noEmit
node ./node_modules/eslint/bin/eslint.js .
```

Expected: clean.

- [ ] **Step 5: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add Prometheus-V2/frontend/src/lib/auth/store.ts Prometheus-V2/frontend/src/lib/api/client.ts Prometheus-V2/frontend/package.json Prometheus-V2/frontend/package-lock.json
git commit -m "feat(v2): wire frontend auth store and bearer interceptor"
```

---

## Task 15: Login Form + Login Route + Route Guard

**Files:**
- Create: `Prometheus-V2/frontend/src/components/auth/login-form.tsx`
- Create: `Prometheus-V2/frontend/src/components/auth/login-form.test.tsx`
- Create: `Prometheus-V2/frontend/src/lib/auth/guard.tsx`
- Create: `Prometheus-V2/frontend/src/routes/login.tsx`
- Modify: `Prometheus-V2/frontend/src/routes/__root.tsx`

- [ ] **Step 1: Login form**

Create `Prometheus-V2/frontend/src/components/auth/login-form.tsx`:

```tsx
import { useState, type FormEvent } from "react";
import { Button } from "@/components/ui/button";
import { ErrorState } from "@/components/ui/error-state";
import { loginRequest } from "@/lib/auth/client";
import { useAuthStore } from "@/lib/auth/store";
import { ApiError } from "@/lib/api/client";

export function LoginForm({ onSuccess }: { onSuccess?: () => void }) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const setSession = useAuthStore((s) => s.setSession);

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);
    setPending(true);
    try {
      const res = await loginRequest({ email, password });
      setSession(res.access_token, res.user);
      onSuccess?.();
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.status === 401 ? "Email oder Passwort falsch." : err.message);
      } else {
        setError("Login fehlgeschlagen.");
      }
    } finally {
      setPending(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4 surface-panel-strong p-6 max-w-sm mx-auto">
      <div>
        <h1 className="text-xl font-semibold tracking-tight">Anmeldung</h1>
        <p className="mt-1 text-sm text-muted-foreground">Prometheus V2 Operations Cockpit</p>
      </div>
      <label className="flex flex-col gap-1 text-sm">
        Email
        <input
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
          autoComplete="username"
          className="h-9 rounded-md border border-border bg-card px-3 text-foreground"
        />
      </label>
      <label className="flex flex-col gap-1 text-sm">
        Passwort
        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
          autoComplete="current-password"
          className="h-9 rounded-md border border-border bg-card px-3 text-foreground"
        />
      </label>
      {error && <ErrorState message={error} />}
      <Button type="submit" disabled={pending}>{pending ? "Anmelden..." : "Anmelden"}</Button>
    </form>
  );
}
```

- [ ] **Step 2: Login-form test**

Create `Prometheus-V2/frontend/src/components/auth/login-form.test.tsx`:

```tsx
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { LoginForm } from "./login-form";

vi.mock("@/lib/auth/client", () => ({
  loginRequest: vi.fn(),
}));

import { loginRequest } from "@/lib/auth/client";
import { useAuthStore } from "@/lib/auth/store";
import { ApiError } from "@/lib/api/client";

describe("LoginForm", () => {
  beforeEach(() => {
    useAuthStore.getState().clearSession();
    vi.clearAllMocks();
  });

  it("calls onSuccess and stores session on success", async () => {
    (loginRequest as ReturnType<typeof vi.fn>).mockResolvedValue({
      access_token: "tok",
      access_expires_at: new Date().toISOString(),
      user: { id: "uid", email: "a@b.c", name: "Alice", role: "admin", enabled: true },
    });
    const onSuccess = vi.fn();
    render(<LoginForm onSuccess={onSuccess} />);

    fireEvent.change(screen.getByLabelText("Email"), { target: { value: "a@b.c" } });
    fireEvent.change(screen.getByLabelText("Passwort"), { target: { value: "secret-1234" } });
    fireEvent.submit(screen.getByRole("button", { name: /anmelden/i }).closest("form")!);

    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalledTimes(1);
      expect(useAuthStore.getState().accessToken).toBe("tok");
      expect(useAuthStore.getState().user?.email).toBe("a@b.c");
    });
  });

  it("renders 401 error message on bad credentials", async () => {
    (loginRequest as ReturnType<typeof vi.fn>).mockRejectedValue(new ApiError(401, undefined, "no"));
    render(<LoginForm />);

    fireEvent.change(screen.getByLabelText("Email"), { target: { value: "a@b.c" } });
    fireEvent.change(screen.getByLabelText("Passwort"), { target: { value: "secret-1234" } });
    fireEvent.submit(screen.getByRole("button", { name: /anmelden/i }).closest("form")!);

    await waitFor(() => {
      expect(screen.getByText(/Email oder Passwort falsch/i)).toBeInTheDocument();
    });
  });
});
```

- [ ] **Step 3: Login route**

Create `Prometheus-V2/frontend/src/routes/login.tsx`:

```tsx
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { LoginForm } from "@/components/auth/login-form";

export const Route = createFileRoute("/login")({
  component: LoginRoute,
});

function LoginRoute() {
  const navigate = useNavigate();
  return (
    <div className="min-h-full flex items-center justify-center p-4">
      <LoginForm onSuccess={() => navigate({ to: "/" })} />
    </div>
  );
}
```

- [ ] **Step 4: Auth-aware route guard**

Create `Prometheus-V2/frontend/src/lib/auth/guard.tsx`:

```tsx
import { useEffect, type ReactNode } from "react";
import { useNavigate, useRouterState } from "@tanstack/react-router";
import { useAuthStore } from "./store";
import { getMeRequest, refreshRequest } from "./client";
import { ApiError } from "@/lib/api/client";

// AuthGate ensures a session exists before rendering children. On mount it
// tries /auth/refresh (HttpOnly cookie hits the backend) and falls back to
// /login if no session can be established. While refreshing, it renders
// nothing to avoid flashing protected UI.
export function AuthGate({ children }: { children: ReactNode }) {
  const status = useAuthStore((s) => s.status);
  const setSession = useAuthStore((s) => s.setSession);
  const setStatus = useAuthStore((s) => s.setStatus);
  const clearSession = useAuthStore((s) => s.clearSession);
  const navigate = useNavigate();
  const routerLocation = useRouterState({ select: (s) => s.location.pathname });

  useEffect(() => {
    if (status !== "anonymous") return;
    if (routerLocation === "/login") return;
    setStatus("refreshing");
    refreshRequest()
      .then(async (r) => {
        const me = await getMeRequest();
        setSession(r.access_token, me);
      })
      .catch((err) => {
        clearSession();
        if (!(err instanceof ApiError) || err.status !== 401) {
          // network error vs unauth: swallow either; user lands on /login
        }
        navigate({ to: "/login" });
      });
  }, [status, routerLocation, setStatus, setSession, clearSession, navigate]);

  if (status === "refreshing") return null;
  if (status === "anonymous" && routerLocation !== "/login") return null;
  return <>{children}</>;
}
```

- [ ] **Step 5: Wire AuthGate + login route in __root.tsx**

Modify `Prometheus-V2/frontend/src/routes/__root.tsx`:

```tsx
import { createRootRoute, Outlet, useRouterState } from "@tanstack/react-router";
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools";
import { ThemeProvider } from "@/components/theme-provider";
import { AppShell } from "@/components/layout/app-shell";
import { AuthGate } from "@/lib/auth/guard";

export const Route = createRootRoute({
  component: RootRoute,
});

function RootRoute() {
  const isLogin = useRouterState({ select: (s) => s.location.pathname === "/login" });
  return (
    <ThemeProvider>
      <AuthGate>
        {isLogin ? (
          <Outlet />
        ) : (
          <AppShell>
            <Outlet />
          </AppShell>
        )}
      </AuthGate>
      {import.meta.env.DEV && <TanStackRouterDevtools position="bottom-right" />}
    </ThemeProvider>
  );
}
```

- [ ] **Step 6: Run frontend tests**

```bash
cd Prometheus-V2/frontend
node ./node_modules/vitest/vitest.mjs run
node ./node_modules/typescript/bin/tsc --noEmit
node ./node_modules/eslint/bin/eslint.js .
```

Expected: 6 tests pass (HealthStatus 2 + StatusBadge 2 + LoginForm 2). Type-check + lint clean.

- [ ] **Step 7: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add Prometheus-V2/frontend/src/components/auth/ Prometheus-V2/frontend/src/lib/auth/guard.tsx Prometheus-V2/frontend/src/routes/login.tsx Prometheus-V2/frontend/src/routes/__root.tsx
git commit -m "feat(v2): add login form route and auth gate"
```

---

## Task 16: Topbar User Menu + Logout

**Files:**
- Create: `Prometheus-V2/frontend/src/components/auth/user-menu.tsx`
- Modify: `Prometheus-V2/frontend/src/components/layout/topbar.tsx`

- [ ] **Step 1: User menu**

Create `Prometheus-V2/frontend/src/components/auth/user-menu.tsx`:

```tsx
import { LogOut } from "lucide-react";
import { useNavigate } from "@tanstack/react-router";
import { Button } from "@/components/ui/button";
import { useAuthStore } from "@/lib/auth/store";
import { logoutRequest } from "@/lib/auth/client";

export function UserMenu() {
  const user = useAuthStore((s) => s.user);
  const clearSession = useAuthStore((s) => s.clearSession);
  const navigate = useNavigate();

  async function handleLogout() {
    try {
      await logoutRequest();
    } finally {
      clearSession();
      navigate({ to: "/login" });
    }
  }

  if (!user) return null;
  return (
    <div className="flex items-center gap-3">
      <div className="text-right hidden sm:block">
        <p className="text-sm font-medium leading-tight">{user.name}</p>
        <p className="text-xs uppercase tracking-wide text-muted-foreground">{user.role}</p>
      </div>
      <Button variant="ghost" size="icon" onClick={handleLogout} aria-label="Abmelden">
        <LogOut className="h-4 w-4" />
      </Button>
    </div>
  );
}
```

- [ ] **Step 2: Topbar mounts UserMenu**

Modify `Prometheus-V2/frontend/src/components/layout/topbar.tsx`. Replace the existing right-side group:

```tsx
import { ThemeToggle } from "@/components/theme-toggle";
import { StatusBadge } from "@/components/ui/status-badge";
import { UserMenu } from "@/components/auth/user-menu";

export function Topbar() {
  return (
    <header className="flex h-14 items-center justify-between border-b border-border bg-card px-5">
      <div className="flex items-center gap-3">
        <StatusBadge tone="ok" withIcon>Live</StatusBadge>
        <p className="text-sm text-muted-foreground">Skeleton-Build</p>
      </div>
      <div className="flex items-center gap-3">
        <ThemeToggle />
        <UserMenu />
      </div>
    </header>
  );
}
```

- [ ] **Step 3: Verify**

```bash
cd Prometheus-V2/frontend
node ./node_modules/typescript/bin/tsc --noEmit
node ./node_modules/eslint/bin/eslint.js .
node ./node_modules/vitest/vitest.mjs run
node ./node_modules/vite/bin/vite.js build
```

Expected: type-check clean, eslint 0 errors, 6 tests pass, build succeeds.

- [ ] **Step 4: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add Prometheus-V2/frontend/src/components/auth/user-menu.tsx Prometheus-V2/frontend/src/components/layout/topbar.tsx
git commit -m "feat(v2): add user menu with logout in topbar"
```

---

## Self-Review

Run after all tasks:

- [ ] **Spec coverage:** Auth section of design spec (`docs/superpowers/specs/2026-04-28-prometheus-v2-rebuild-design.md`) is mostly covered:
  - ✓ Users + JWT + Refresh-Cookie
  - ✓ Three roles + permission constants
  - ✓ RequireAuth + RequirePermission middleware
  - ✓ Login/Logout/Refresh + /me endpoint
  - ✓ Bootstrap admin
  - ✗ API-Keys (Personal + Service-Identity) — out of scope (Plan 4 / Plan-Agent)
  - ✗ Sessions list/revoke UI — out of scope (Plan 4)
  - ✗ Audit-Log entries on auth events — Plan 3 (Audit-Domain) will hook into auth events via the events-bus when that lands
- [ ] **Build pipeline:** `make verify` (or equivalents via direct binaries) passes:
  - `go test ./...` includes auth-domain tests; integration tests still SKIP without env.
  - Frontend `vitest run` shows 6 tests, all PASS.
  - `go build ./cmd/prometheus` succeeds.
  - `vite build` succeeds.
- [ ] **Smoke test (live):** Once Postgres is available locally OR in Docker, run end-to-end:
  - Start `docker compose up -d postgres redis`.
  - `PROMETHEUS_JWT_SECRET="..."` `PROMETHEUS_BOOTSTRAP_ADMIN_EMAIL="admin@example.com"` `PROMETHEUS_BOOTSTRAP_ADMIN_PASSWORD="secretsecret"` `go run ./cmd/prometheus`.
  - In another terminal: `curl -X POST -H 'Content-Type: application/json' -d '{"email":"admin@example.com","password":"secretsecret"}' http://localhost:8180/api/v1/auth/login`.
  - Should return 200 + access_token + Set-Cookie header.
  - Use returned token in `Authorization: Bearer <token>` to hit `/api/v1/auth/me` → 200.
  - In browser: open `http://localhost:8180/`, redirected to `/login`, log in, see Lagezentrum with user-menu.
- [ ] **Rotation security:** Verify login → refresh → ensure old refresh token is now invalid (revoked).
- [ ] **No leaked secrets:** Search the diff for `secret`, `password`, `bootstrap` — make sure no test fixtures have committed real credentials.
- [ ] **Generated files NOT committed:** `internal/api/openapi.gen.go`, `internal/db/repo/*.go`, `frontend/src/lib/api/schema.d.ts`, `frontend/src/routeTree.gen.ts` — verify with `git ls-files | grep -E '(\\.gen\\.|schema\\.d\\.ts)'` (should be empty).

## Folge-Plaene

Nach Abschluss von Plan 2 folgen:

3. **Audit-Domain** — append-only Audit-Events-Tabelle, sidecar in jede Schreibaktion (incl. auth login/logout/refresh-fail), Audit-API + UI.
4. **Realtime-Bus** — Redis-Pub/Sub-Event-Bus, SSE-Endpoint mit Resume, WebSocket-Metrics-Hub.
5. **Approval-Domain** — Vier-Augen-Workflow, Approval-Inbox.
6. **Host-Domain** mit Proxmox-Adapter — danach folgen die anderen Domain-Plaene.
7. **API-Keys + Service-Identities** — landen wenn der erste Konsumer (Agent oder Admin-UI) sie braucht.
