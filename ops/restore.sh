#!/usr/bin/env bash
# Restauração de um backup lógico (destrutiva: limpa objetos antes).
# Uso: DATABASE_URL=postgres://... ops/restore.sh caminho/backup.dump
set -euo pipefail

DATABASE_URL="${DATABASE_URL:?defina DATABASE_URL}"
DUMP="${1:?informe o arquivo .dump}"

pg_restore --clean --if-exists --no-owner --dbname="$DATABASE_URL" "$DUMP"
echo "restauração concluída de $DUMP"
