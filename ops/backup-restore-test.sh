#!/usr/bin/env bash
# Prova de backup/restore de ponta a ponta num PostgreSQL efêmero:
# migra → semeia marcador → backup → destrói dados → restaura → verifica.
# Requer docker. Sai com código ≠0 se a restauração não preservar os dados.
set -euo pipefail
cd "$(dirname "$0")/.."

PORT="${BACKUP_TEST_PORT:-}"
NAME="vr-backup-test-$$-$RANDOM"
TMP=""

cleanup() {
  docker stop "$NAME" >/dev/null 2>&1 || true
  if [ -n "$TMP" ] && [ -d "$TMP" ]; then
    rm -rf -- "$TMP"
  fi
}
trap cleanup EXIT

PORT_ARGS=(-p "127.0.0.1::5432")
if [ -n "$PORT" ]; then
  PORT_ARGS=(-p "127.0.0.1:$PORT:5432")
fi
docker run -d --rm --name "$NAME" -e POSTGRES_USER=veurubro \
  -e POSTGRES_PASSWORD=veurubro_dev -e POSTGRES_DB=veurubro \
  "${PORT_ARGS[@]}" postgres:16-alpine >/dev/null
if [ -z "$PORT" ]; then
  PUBLISHED="$(docker port "$NAME" 5432/tcp)"
  PORT="${PUBLISHED##*:}"
fi
URL="postgres://veurubro:veurubro_dev@127.0.0.1:$PORT/veurubro?sslmode=disable"

for _ in {1..30}; do
  if docker exec "$NAME" pg_isready -U veurubro >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
if ! docker exec "$NAME" pg_isready -U veurubro >/dev/null 2>&1; then
  echo "FALHA: PostgreSQL temporário não ficou pronto" >&2
  exit 1
fi

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
if [ "$MARKER" != "1" ]; then
  echo "FALHA: marcador não sobreviveu à restauração" >&2
  exit 1
fi
echo "OK: restauração preservou o marcador; $TABLES tabelas no schema"
