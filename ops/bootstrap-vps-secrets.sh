#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
SECRETS_DIR=${NYTHARA_SECRETS_DIR:-"$ROOT/secrets"}

if ! command -v openssl >/dev/null 2>&1; then
  echo "openssl é necessário para gerar as credenciais" >&2
  exit 1
fi

mkdir -p "$SECRETS_DIR"
chmod 700 "$SECRETS_DIR"
umask 077

PASSWORD_FILE="$SECRETS_DIR/postgres_password"
URL_FILE="$SECRETS_DIR/database_url"
if [ -e "$PASSWORD_FILE" ] || [ -e "$URL_FILE" ]; then
  echo "segredos já existem em $SECRETS_DIR; nada foi sobrescrito" >&2
  exit 1
fi

POSTGRES_PASSWORD=$(openssl rand -hex 32)
printf '%s\n' "$POSTGRES_PASSWORD" > "$PASSWORD_FILE"
printf 'postgres://nythara:%s@postgres:5432/nythara?sslmode=disable\n' "$POSTGRES_PASSWORD" > "$URL_FILE"
chmod 600 "$PASSWORD_FILE" "$URL_FILE"

if [ ! -e "$ROOT/.env.production" ]; then
  cp "$ROOT/.env.production.example" "$ROOT/.env.production"
  chmod 600 "$ROOT/.env.production"
fi

echo "Credenciais criadas em $SECRETS_DIR. Ajuste $ROOT/.env.production antes do deploy."
