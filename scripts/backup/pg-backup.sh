#!/bin/sh
# Postgres backup loop. Designed to run inside the postgres:17-alpine image
# (or any image that ships pg_dump + gzip + tar). One file per run, gzipped,
# named with a UTC timestamp. Retention enforced by file mtime.
#
# Required env vars:
#   PGHOST            — postgres hostname (default: postgres)
#   PGUSER            — user with read access to the database
#   PGPASSWORD        — user password
#   PGDATABASE        — database to back up
#   BACKUP_INTERVAL_S — seconds between runs (default: 86400 = 24h)
#   BACKUP_RETENTION_DAYS — keep dumps newer than N days (default: 14)
#   BACKUP_DIR        — output directory (default: /backup)

set -eu

: "${PGHOST:=postgres}"
: "${PGUSER:=prometheus}"
: "${PGDATABASE:=prometheus}"
: "${BACKUP_INTERVAL_S:=86400}"
: "${BACKUP_RETENTION_DAYS:=14}"
: "${BACKUP_DIR:=/backup}"

mkdir -p "$BACKUP_DIR"

run_backup() {
	ts=$(date -u +%Y%m%dT%H%M%SZ)
	tmp="$BACKUP_DIR/.in-progress-$ts.sql.gz"
	final="$BACKUP_DIR/prometheus-$ts.sql.gz"
	echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] starting pg_dump → $final"
	# --clean adds DROP statements so the dump can restore over an existing
	# DB. --if-exists keeps it idempotent.
	if pg_dump --clean --if-exists --no-owner --no-privileges "$PGDATABASE" | gzip -9 > "$tmp"; then
		mv "$tmp" "$final"
		echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] backup ok: $(ls -lh "$final" | awk '{print $5}')"
	else
		echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] backup FAILED" >&2
		rm -f "$tmp"
		return 1
	fi
	# Retention: delete dumps older than BACKUP_RETENTION_DAYS days.
	find "$BACKUP_DIR" -name 'prometheus-*.sql.gz' -type f -mtime "+${BACKUP_RETENTION_DAYS}" -delete -print | while read -r f; do
		echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] retention: pruned $f"
	done
}

# Allow a single-shot run for testing: BACKUP_ONCE=true
if [ "${BACKUP_ONCE:-false}" = "true" ]; then
	run_backup
	exit 0
fi

while true; do
	run_backup || true
	echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] sleeping ${BACKUP_INTERVAL_S}s"
	sleep "$BACKUP_INTERVAL_S"
done
