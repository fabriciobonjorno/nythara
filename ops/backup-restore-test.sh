#!/usr/bin/env bash
# Prova de backup/restore de ponta a ponta num PostgreSQL efêmero:
# migra → semeia marcador → backup → destrói dados → restaura → verifica.
# Requer docker. Sai com código ≠0 se a restauração não preservar os dados.
set -euo pipefail
cd "$(dirname "$0")/.."

PORT="${BACKUP_TEST_PORT:-55433}"
NAME="vr-backup-test"
URL="postgres://veurubro:veurubro_dev@localhost:$PORT/veurubro?sslmode=disable"

cleanup() { docker stop "$NAME" >/dev/null 2>&1 || true; }
trap cleanup EXIT

docker run -d --rm --name "$NAME" -e POSTGRES_USER=veurubro \
  -e POSTGRES_PASSWORD=veurubro_dev -e POSTGRES_DB=veurubro \
  -p "$PORT:5432" postgres:16-alpine >/dev/null
until docker exec "$NAME" pg_isready -U veurubro >/dev/null 2>&1; do sleep 1; done

(cd backend && DATABASE_URL="$URL" go run ./cmd/migrate up)
docker exec "$NAME" psql -U veurubro -q -c \
  "INSERT INTO rulesets(version) VALUES('backup-marker-0.0.1');"

TMP="$(mktemp -d)"
docker exec "$NAME" pg_dump --format=custom --no-owner -U veurubro veurubro \
  > "$TMP/backup.dump"

docker exec "$NAME" psql -U veurubro -q -c \
  "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
docker exec -i "$NAME" pg_restore --clean --if-exists --no-owner \
  -U veurubro --dbname=veurubro < "$TMP/backup.dump"

MARKER="$(docker exec "$NAME" psql -U veurubro -tA -c \
  "SELECT count(*) FROM rulesets WHERE version='backup-marker-0.0.1';")"
TABLES="$(docker exec "$NAME" psql -U veurubro -tA -c \
  "SELECT count(*) FROM information_schema.tables WHERE table_schema='public';")"
rm -rf "$TMP"

if [ "$MARKER" != "1" ]; then
  echo "FALHA: marcador não sobreviveu à restauração" >&2
  exit 1
fi
echo "OK: restauração preservou o marcador; $TABLES tabelas no schema"
