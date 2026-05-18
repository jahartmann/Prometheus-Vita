# Troubleshooting

First steps for any incident, in order:

```sh
docker compose ps                 # which containers exist, are they (healthy)?
docker compose logs --tail=200    # recent events across the whole stack
curl -fsS http://localhost:8080/ready  # backend self-report
```

Then jump to the specific symptom below.

## Backend won't start

### Error: `JWT_SECRET must be at least 32 characters long`

The `.env` is missing or still has the placeholder. Run `./setup.sh` once to
generate strong values, or fill them in by hand:

```sh
JWT_SECRET=$(openssl rand -hex 32)
ENCRYPTION_KEY=$(openssl rand -hex 32)
```

`ENCRYPTION_KEY` **must** be exactly 64 hex characters (32 bytes for AES-256)
— validated at startup.

### Error: `POSTGRES_PASSWORD is using the default placeholder value`

The backend refuses to boot with the `changeme_db_password` placeholder so
you don't accidentally ship a default-password instance to production.
Change it in `.env` and `docker compose up -d --force-recreate postgres
backend` so Postgres also picks up the new value.

### Error: `failed to connect to PostgreSQL` / `failed to connect to Redis`

Check the dependency is healthy:

```sh
docker compose ps postgres redis
docker compose logs --tail=100 postgres
```

If you just rotated `POSTGRES_PASSWORD`, the **Postgres data directory still
holds the old password**. Either:

- restore from a backup taken before the rotation, or
- `docker compose down -v` to drop the data volume and re-initialise
  (loses all data).

## Frontend shows blank page / "API momentan nicht erreichbar"

The frontend is fine — the backend isn't reachable.

```sh
curl -fsS http://localhost:8080/live   # process alive?
curl -fsS http://localhost:8080/ready  # deps reachable?
docker compose logs --tail=100 backend
```

If `/ready` returns 503 with `"postgres": "unhealthy"`, see the previous
section.

## "Sitzung abgelaufen" toast keeps appearing

The JWT refresh is failing repeatedly. Check:

1. `JWT_SECRET` is the same value the access tokens were signed with. If
   you rotated it, all existing sessions become invalid — log out and back
   in.
2. Browser clock is within ~5 minutes of the server clock. Wildly skewed
   clocks make JWTs look expired even when they aren't.
3. Refresh-token cookie is being sent — under *DevTools → Application →
   Cookies*, look for the HttpOnly refresh cookie on the API origin.

## WebSocket disconnects / no live updates

The WS hub drops messages when the broadcast buffer overflows. Symptoms:
metrics charts freeze, but other API calls work.

```sh
docker compose logs --tail=200 backend | grep "ws broadcast"
```

A few drops are fine. Continuous drops mean a metric is being published
faster than clients can consume it. Either:

- reduce the scrape interval of the offending scheduler job, or
- bump the broadcast channel buffer (currently 1024) if RAM allows.

## Database migrations stuck on boot

The migrator acquires a Postgres advisory lock so concurrent boots don't
race. If one instance crashes mid-migration the lock is released
automatically, but if you killed the container with `SIGKILL` while
holding it, manually clear it:

```sh
docker compose exec postgres psql -U prometheus -c "SELECT pg_advisory_unlock_all();"
docker compose restart backend
```

Each migration runs inside a transaction. A failed migration is rolled
back fully and logged with `migration rollback failed` (or success). Fix
the migration SQL and restart.

## Caddy: cert issuance keeps failing

Common causes:

- `PROMETHEUS_HOSTNAME` doesn't resolve publicly. ACME validates by HTTP
  challenge on port 80 → ensure DNS is correct and port 80 reaches the
  host.
- ACME rate-limited. Let's Encrypt has weekly limits per domain. Stay on
  `PROMETHEUS_TLS_MODE_DIRECTIVE=internal` while you iterate, switch to
  `auto` once.
- Firewall blocks port 443. `curl -v https://<hostname>` from another
  machine to confirm.

Watch issuance live: `docker compose logs -f caddy`.

## Resource exhaustion

```sh
docker stats
```

If `backend` or `postgres` is consistently at ≥ 90% CPU or memory, edit
`docker-compose.yml` `deploy.resources.limits` upward. The defaults target
small clusters (≤ 10 nodes); for 30+ nodes double them.

## Collecting a diagnostic bundle

When asking for help, run:

```sh
docker compose ps > diag.txt
docker compose logs --tail=2000 >> diag.txt
docker compose exec backend wget -qO- http://localhost:8080/ready >> diag.txt
```

`diag.txt` will not contain secrets — the backend redacts DSNs in
`SafeDSN()` and never logs `JWT_SECRET`/`ENCRYPTION_KEY`. Still skim it
once before sharing externally.
