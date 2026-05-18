#!/bin/sh
# Restore a previously taken Postgres backup into the running database.
#
# Usage (inside the docker-compose stack):
#   docker compose --profile backup run --rm postgres-backup \
#     /scripts/pg-restore.sh /backup/prometheus-20260518T020000Z.sql.gz
#
# Or from the host with a one-shot postgres-client container:
#   docker run --rm -i --network prometheus-vita_prometheus-net \
#     -e PGHOST=postgres -e PGUSER=prometheus -e PGPASSWORD=$POSTGRES_PASSWORD \
#     postgres:17-alpine sh -c 'gunzip -c < /dev/stdin | psql prometheus' \
#     < ./backups/prometheus-20260518T020000Z.sql.gz

set -eu

: "${PGHOST:=postgres}"
: "${PGUSER:=prometheus}"
: "${PGDATABASE:=prometheus}"

dump="${1:-}"
if [ -z "$dump" ] || [ ! -f "$dump" ]; then
	echo "usage: $0 <path-to-dump.sql.gz>" >&2
	echo "example: $0 /backup/prometheus-20260518T020000Z.sql.gz" >&2
	exit 2
fi

echo "Restoring $dump into $PGUSER@$PGHOST/$PGDATABASE …"
echo "This will OVERWRITE the current database content (the dump uses DROP IF EXISTS)."
printf "Type YES to continue: "
read -r confirm
if [ "$confirm" != "YES" ]; then
	echo "Aborted."
	exit 1
fi

gunzip -c "$dump" | psql "$PGDATABASE"
echo "Restore complete."
