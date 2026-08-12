#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
ENV_FILE=${NYTHARA_ENV_FILE:-"$ROOT/.env.production"}
BACKUP_DIR=${NYTHARA_BACKUP_DIR:-"$ROOT/backups"}
COMPOSE_FILE="$ROOT/compose.production.yml"

mkdir -p "$BACKUP_DIR"
chmod 700 "$BACKUP_DIR"
umask 077

STAMP=$(date -u +%Y%m%dT%H%M%SZ)
OUT="$BACKUP_DIR/nythara-$STAMP.dump"

docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" exec -T postgres \
  pg_dump --username=nythara --dbname=nythara --format=custom --compress=6 --no-owner > "$OUT"

test -s "$OUT"
echo "$OUT"
