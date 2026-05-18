# Backup & Restore

The Prometheus meta-database in Postgres holds **everything that is not in
Proxmox itself**: users, RBAC, API tokens, alert rules, agent config, audit
trail, metrics history, backups metadata. Losing it is catastrophic — the
Proxmox cluster keeps running, but every Prometheus-managed configuration
is gone.

This document covers backups of the Prometheus database. For the Proxmox
config backups that Prometheus *takes of* the Proxmox nodes, see the
in-app *Backups* section.

## Strategy

- **Frequency:** daily (default `BACKUP_INTERVAL_S=86400`).
- **Retention:** 14 days on the host (default `BACKUP_RETENTION_DAYS=14`).
- **Format:** `pg_dump --clean --if-exists` gzipped; idempotent on restore.
- **Storage:** named volume `postgres_backups`. Mount it to an off-host
  location (SMB share, NFS, S3-FUSE) so a dying host doesn't take the
  backups down with it.

## Enable the backup sidecar

```sh
docker compose --profile backup up -d
docker compose logs -f postgres-backup
```

You should see `backup ok: <size>` within the first minute (the first dump
runs immediately, then once per interval).

## Off-host storage

The simplest pattern: replace the named volume with a bind mount pointing
at a mounted network share.

```yaml
# docker-compose.override.yml
services:
  postgres-backup:
    volumes:
      - /mnt/backup/prometheus:/backup
      - ./scripts/backup:/scripts:ro
```

For S3-compatible object storage, run a sidecar like `rclone sync` on a
cron or use `awscli` in a small wrapper.

## Manual backup

A one-shot dump without enabling the sidecar:

```sh
docker compose exec postgres pg_dump \
  --clean --if-exists --no-owner --no-privileges prometheus \
  | gzip -9 > prometheus-$(date -u +%Y%m%dT%H%M%SZ).sql.gz
```

## Restore

**Warning:** restore overwrites the live database (`DROP IF EXISTS`).
Schedule a maintenance window.

### From the backup container (preferred)

```sh
docker compose --profile backup run --rm postgres-backup \
  /scripts/pg-restore.sh /backup/prometheus-20260518T020000Z.sql.gz
```

The script prompts for `YES` before touching the DB.

### From the host (one-shot)

```sh
docker run --rm -i --network prometheus-vita_prometheus-net \
  -e PGHOST=postgres \
  -e PGUSER=prometheus \
  -e PGPASSWORD="$POSTGRES_PASSWORD" \
  postgres:17-alpine \
  sh -c 'gunzip -c < /dev/stdin | psql prometheus' \
  < ./backups/prometheus-20260518T020000Z.sql.gz
```

After restore, restart the backend so any cached state in memory is rebuilt:

```sh
docker compose restart backend
```

## Verification

A backup that never restores is not a backup. Schedule a quarterly
restore-rehearsal:

1. Spin up a throwaway Postgres in a separate compose project.
2. Restore the most recent dump into it.
3. Run `SELECT count(*) FROM users, nodes, audit_log;` — every count
   should be plausible.
4. Tear it down.

## Disaster recovery

Whole-host loss:

1. Provision a new host with Docker + Compose.
2. `git clone` this repo.
3. Restore the most recent off-host dump *before* the first `docker compose
   up`. The backend will run pending migrations on startup against the
   restored schema and continue from there.
4. Re-mount the off-host backup share so backups continue.

Don't try to restore by copying `/var/lib/postgresql/data` directly across
hosts — Postgres binary files are version-locked and OS-locked. Always
restore from a `pg_dump`.
