#!/usr/bin/env bash
# Backup lógico do PostgreSQL (formato custom, comprimido).
# Uso: DATABASE_URL=postgres://... ops/backup.sh [diretório-destino]
# Produção: agendar via cron/K8s CronJob e enviar para armazenamento externo
# com retenção; PITR (WAL archiving) é configurado no provedor — ver
# ops/observability.md para o alerta de idade do último backup.
set -euo pipefail

DATABASE_URL="${DATABASE_URL:?defina DATABASE_URL}"
DEST="${1:-backups}"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p "$DEST"
OUT="$DEST/veurubro-$STAMP.dump"

pg_dump --format=custom --compress=6 --no-owner --file="$OUT" "$DATABASE_URL"
echo "backup gravado em $OUT ($(du -h "$OUT" | cut -f1))"
