#!/usr/bin/env bash
set -euo pipefail

env_file="${1:-.env.production}"
if [[ ! -f "$env_file" ]]; then
  echo "Arquivo $env_file não encontrado." >&2
  exit 1
fi
if ! command -v curl >/dev/null || ! command -v jq >/dev/null; then
  echo "curl e jq são obrigatórios." >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "$env_file"
set +a

: "${RESEND_API_KEY:?defina RESEND_API_KEY em $env_file}"
: "${PUBLIC_APP_URL:?defina PUBLIC_APP_URL em $env_file}"

endpoint="${PUBLIC_APP_URL%/}/v1/webhooks/resend"
response_file="$(mktemp)"
next_env="$(mktemp "${env_file}.next.XXXXXX")"
auth_file="$(mktemp)"
cleanup() {
  rm -f "$response_file" "$next_env" "$auth_file"
}
trap cleanup EXIT
chmod 0600 "$auth_file"
printf 'Authorization: Bearer %s\n' "$RESEND_API_KEY" >"$auth_file"

events='["email.sent","email.delivered","email.delivery_delayed","email.bounced","email.complained","email.failed","email.suppressed"]'

resend_request() {
  local method="$1"
  local url="$2"
  local payload="${3:-}"
  local args=(--silent --show-error --output "$response_file" --write-out '%{http_code}'
    --request "$method" --header "@${auth_file}")
  if [[ -n "$payload" ]]; then
    args+=(--header 'Content-Type: application/json' --data "$payload")
  fi
  curl "${args[@]}" "$url"
}

require_success() {
  local status="$1"
  if [[ "$status" -lt 200 || "$status" -ge 300 ]]; then
    local message
    message="$(jq -r '.message // .error // "erro sem mensagem"' "$response_file" 2>/dev/null || true)"
    echo "Resend recusou a operação do webhook (HTTP $status): $message" >&2
    exit 1
  fi
}

status="$(resend_request GET https://api.resend.com/webhooks)"
require_success "$status"
webhook_id="$(jq -r --arg endpoint "$endpoint" '.data[]? | select(.endpoint == $endpoint) | .id' "$response_file" | head -n 1)"

if [[ -n "$webhook_id" ]]; then
  payload="$(jq -nc --argjson events "$events" '{events:$events,status:"enabled"}')"
  status="$(resend_request PATCH "https://api.resend.com/webhooks/${webhook_id}" "$payload")"
  require_success "$status"
  status="$(resend_request GET "https://api.resend.com/webhooks/${webhook_id}")"
  require_success "$status"
else
  payload="$(jq -nc --arg endpoint "$endpoint" --argjson events "$events" '{endpoint:$endpoint,events:$events}')"
  status="$(resend_request POST https://api.resend.com/webhooks "$payload")"
  require_success "$status"
  webhook_id="$(jq -r '.id // empty' "$response_file")"
fi

signing_secret="$(jq -r '.signing_secret // empty' "$response_file")"
if [[ -z "$webhook_id" || "$signing_secret" != whsec_* ]]; then
  echo "Resposta do Resend não trouxe id/signing_secret válidos." >&2
  exit 1
fi

awk -v value="$signing_secret" '
  BEGIN { replaced=0 }
  /^RESEND_WEBHOOK_SECRET=/ { print "RESEND_WEBHOOK_SECRET=" value; replaced=1; next }
  { print }
  END { if (!replaced) print "RESEND_WEBHOOK_SECRET=" value }
' "$env_file" >"$next_env"
chmod 0600 "$next_env"
mv "$next_env" "$env_file"

echo "Webhook $webhook_id configurado em $endpoint."
echo "O signing secret foi salvo em $env_file sem ser exibido."
