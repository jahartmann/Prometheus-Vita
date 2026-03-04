# Prometheus

Intelligentes Proxmox Infrastructure Management mit KI-Agent-System.

## Quick Start

```bash
# 1. Environment konfigurieren
cp .env.example .env
# Mindestens POSTGRES_PASSWORD, REDIS_PASSWORD, JWT_SECRET, ENCRYPTION_KEY anpassen!
# Keys generieren: openssl rand -hex 32

# 2. Alles starten
docker compose up --build

# 3. Zugriff
# Frontend: http://localhost:3000
# Backend:  http://localhost:8080
# Health:   http://localhost:8080/health
```

Beim ersten Start wird ein Admin-User erstellt. Passwort aus `ADMIN_PASSWORD` in `.env` oder automatisch generiert (siehe Server-Logs).

## Architektur

```
Frontend (Next.js 15)  ──►  Backend (Go/Echo)  ──►  PostgreSQL 17
     :3000                      :8080                   :5432
                                   │
                                   ├──►  Redis 8 (:6379)
                                   └──►  Proxmox API (Remote)
```

## Stack

| Komponente | Technologie |
|-----------|-------------|
| Backend | Go 1.23, Echo v4, pgx/v5, go-redis/v9 |
| Frontend | Next.js 15, React 19, TailwindCSS 4, shadcn/ui |
| Auth | JWT (HS256) + HttpOnly Refresh Cookies |
| Datenbank | PostgreSQL 17, Redis 8 |
| Deployment | Docker Compose |

## Development

### Nur Datenbanken starten
```bash
docker compose up postgres redis
```

### Backend lokal
```bash
cd backend
go run ./cmd/server
```

### Frontend lokal
```bash
cd frontend
npm install
npm run dev
```

## API Endpoints

### Auth (kein JWT nötig)
| Method | Path | Beschreibung |
|--------|------|-------------|
| POST | `/api/v1/auth/login` | Login, gibt Access Token + Refresh Cookie |
| POST | `/api/v1/auth/logout` | Logout, revoked Refresh Token |
| POST | `/api/v1/auth/refresh` | Token erneuern via Refresh Cookie |

### Auth (JWT nötig)
| Method | Path | Beschreibung |
|--------|------|-------------|
| GET | `/api/v1/auth/me` | Aktueller User |

### Nodes (JWT nötig)
| Method | Path | Rollen | Beschreibung |
|--------|------|--------|-------------|
| GET | `/api/v1/nodes` | Alle | Node-Liste |
| GET | `/api/v1/nodes/:id` | Alle | Node-Details |
| GET | `/api/v1/nodes/:id/status` | Alle | Live-Status von Proxmox |
| GET | `/api/v1/nodes/:id/vms` | Alle | VM-Liste |
| GET | `/api/v1/nodes/:id/storage` | Alle | Storage-Info |
| POST | `/api/v1/nodes` | Admin, Operator | Node hinzufügen |
| PUT | `/api/v1/nodes/:id` | Admin, Operator | Node bearbeiten |
| POST | `/api/v1/nodes/test` | Admin, Operator | Verbindung testen |
| DELETE | `/api/v1/nodes/:id` | Admin | Node löschen |

### WebSocket
| Path | Beschreibung |
|------|-------------|
| `WS /api/v1/ws?token=<jwt>` | Live-Metriken Stream |

### System
| Method | Path | Beschreibung |
|--------|------|-------------|
| GET | `/health` | Health Check (DB + Redis) |

## Environment-Variablen

| Variable | Default | Beschreibung |
|----------|---------|-------------|
| `POSTGRES_HOST` | `postgres` | PostgreSQL Host |
| `POSTGRES_PORT` | `5432` | PostgreSQL Port |
| `POSTGRES_USER` | `prometheus` | PostgreSQL User |
| `POSTGRES_PASSWORD` | - | PostgreSQL Passwort |
| `POSTGRES_DB` | `prometheus` | Datenbank-Name |
| `REDIS_HOST` | `redis` | Redis Host |
| `REDIS_PORT` | `6379` | Redis Port |
| `REDIS_PASSWORD` | - | Redis Passwort |
| `SERVER_HOST` | `0.0.0.0` | Backend Bind-Adresse |
| `SERVER_PORT` | `8080` | Backend Port |
| `JWT_SECRET` | - | JWT Signing Key (min. 32 Zeichen) |
| `JWT_ACCESS_TOKEN_EXPIRY` | `15` | Access Token Lebensdauer (Minuten) |
| `JWT_REFRESH_TOKEN_EXPIRY` | `168` | Refresh Token Lebensdauer (Stunden) |
| `ENCRYPTION_KEY` | - | AES-256-GCM Key (64 Hex-Zeichen) |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000` | Erlaubte CORS Origins |
| `ADMIN_USERNAME` | `admin` | Admin-Username beim Seed |
| `ADMIN_PASSWORD` | (generiert) | Admin-Passwort beim Seed |

## Rollen

| Rolle | Beschreibung |
|-------|-------------|
| `admin` | Voller Zugriff, Node-Verwaltung |
| `operator` | Nodes hinzufügen/bearbeiten, Monitoring |
| `viewer` | Nur Leserechte |

## Sicherheit

- Passwörter: bcrypt (Cost 12)
- API-Tokens in DB: AES-256-GCM verschlüsselt
- JWT: HS256, 15min Access Token
- Refresh Tokens: HttpOnly, Secure, SameSite=Strict Cookies
- CORS: Konfigurierbar, Credentials erlaubt
