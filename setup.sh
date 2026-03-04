#!/usr/bin/env bash
set -euo pipefail

# ---------------------------------------------------------------------------
# Prometheus – One-Command Setup
# ---------------------------------------------------------------------------

COMPOSE_FILE="docker-compose.yml"
ENV_FILE=".env"
ENV_EXAMPLE=".env.example"
SERVICES=(postgres redis backend frontend)
TIMEOUT=120  # seconds

# ── Helpers ────────────────────────────────────────────────────────────────

info()  { printf "\033[1;34m[INFO]\033[0m  %s\n" "$*"; }
ok()    { printf "\033[1;32m[OK]\033[0m    %s\n" "$*"; }
warn()  { printf "\033[1;33m[WARN]\033[0m  %s\n" "$*"; }
err()   { printf "\033[1;31m[ERROR]\033[0m %s\n" "$*"; exit 1; }

# ── 1. Prerequisite checks ────────────────────────────────────────────────

info "Pruefe Voraussetzungen ..."

command -v docker >/dev/null 2>&1 || err "docker ist nicht installiert. Bitte zuerst Docker installieren: https://docs.docker.com/get-docker/"

if docker compose version >/dev/null 2>&1; then
  COMPOSE_CMD="docker compose"
elif docker-compose version >/dev/null 2>&1; then
  COMPOSE_CMD="docker-compose"
else
  err "docker compose ist nicht verfuegbar. Bitte Docker Compose installieren."
fi

ok "docker & $COMPOSE_CMD gefunden"

# ── 2. .env erstellen ─────────────────────────────────────────────────────

if [ -f "$ENV_FILE" ]; then
  warn ".env existiert bereits – wird nicht ueberschrieben"
else
  [ -f "$ENV_EXAMPLE" ] || err "$ENV_EXAMPLE nicht gefunden"

  info "Erstelle .env aus $ENV_EXAMPLE mit sicheren Secrets ..."

  JWT_SECRET=$(openssl rand -hex 32)
  ENCRYPTION_KEY=$(openssl rand -hex 32)
  ADMIN_PASSWORD=$(openssl rand -base64 16)

  sed \
    -e "s|changeme_jwt_secret_at_least_32_characters_long|${JWT_SECRET}|" \
    -e "s|changeme_encryption_key_exactly_64_hex_characters_long_here|${ENCRYPTION_KEY}|" \
    -e "s|^ADMIN_PASSWORD=.*|ADMIN_PASSWORD=${ADMIN_PASSWORD}|" \
    -e "s|changeme_db_password|$(openssl rand -hex 16)|" \
    -e "s|changeme_redis_password|$(openssl rand -hex 16)|" \
    "$ENV_EXAMPLE" > "$ENV_FILE"

  ok ".env erstellt"
fi

# ── 3. Docker Compose starten ─────────────────────────────────────────────

info "Starte Services mit $COMPOSE_CMD ..."
$COMPOSE_CMD up --build -d

# ── 4. Auf Services warten ────────────────────────────────────────────────

info "Warte auf Services (Timeout: ${TIMEOUT}s) ..."

elapsed=0
while [ $elapsed -lt $TIMEOUT ]; do
  all_ready=true

  for svc in "${SERVICES[@]}"; do
    status=$($COMPOSE_CMD ps --format json "$svc" 2>/dev/null | head -1)

    # Check health status if available, otherwise check running state
    health=$(echo "$status" | grep -o '"Health":"[^"]*"' | cut -d'"' -f4 2>/dev/null || true)
    state=$(echo "$status" | grep -o '"State":"[^"]*"' | cut -d'"' -f4 2>/dev/null || true)

    if [ "$health" = "healthy" ] || { [ -z "$health" ] && [ "$state" = "running" ]; }; then
      continue
    else
      all_ready=false
    fi
  done

  if $all_ready; then
    break
  fi

  sleep 2
  elapsed=$((elapsed + 2))
  printf "."
done
echo ""

if [ $elapsed -ge $TIMEOUT ]; then
  warn "Timeout erreicht – einige Services sind moeglicherweise noch nicht bereit"
  $COMPOSE_CMD ps
  exit 1
fi

ok "Alle Services sind bereit!"

# ── 5. Zugangsdaten anzeigen ──────────────────────────────────────────────

# Read generated credentials from .env
SHOW_ADMIN_USER=$(grep '^ADMIN_USERNAME=' "$ENV_FILE" | cut -d'=' -f2)
SHOW_ADMIN_PASS=$(grep '^ADMIN_PASSWORD=' "$ENV_FILE" | cut -d'=' -f2)

echo ""
echo "==========================================="
echo "  Prometheus ist einsatzbereit!"
echo "==========================================="
echo ""
echo "  Frontend:  http://localhost:3000"
echo "  Backend:   http://localhost:8080"
echo ""
echo "  Admin-Login:"
echo "    Benutzer: ${SHOW_ADMIN_USER}"
echo "    Passwort: ${SHOW_ADMIN_PASS}"
echo ""
echo "  Die Zugangsdaten stehen in: .env"
echo "==========================================="
