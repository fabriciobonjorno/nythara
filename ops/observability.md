# Observabilidade e alertas — runbook

## Sinais

- **Traces**: OpenTelemetry OTLP (`OTEL_EXPORTER_OTLP_ENDPOINT`); spans HTTP
  via otelhttp. Sem endpoint configurado, provider no-op.
- **Contadores** (`GET /internal/metrics`, formato expvar — coletar via
  scraper/sidecar; não expor no edge):
  - `http_requests_total`, `http_rate_limited_total`, `http_panics_total`
  - `ws_command_flood_closed_total`
- **Banco**: `matches` (status/fim), `admin_audit` (ações administrativas),
  `economy_transactions` (economia).

## Alertas mínimos (produção)

| Alerta | Condição sugerida | Ação |
| --- | --- | --- |
| Erros 5xx | taxa > 1% por 5 min | investigar logs/traces; rollback de deploy |
| Panics | `http_panics_total` cresce | bug crítico; coletar stack no log |
| Flood WS | `ws_command_flood_closed_total` acelera | possível ataque; revisar IPs no edge |
| Rate limit | `http_rate_limited_total` >5% das requisições | abuso ou limite mal calibrado |
| Fila parada | partidas criadas = 0 com fila > 0 por 5 min | battle server travado; reiniciar sala |
| Sim noturna | job `sim-100k` falhou | regressão de regra/determinismo; bloquear release |
| Backup velho | último backup > 24 h | verificar cron/armazenamento |
| Auditoria admin | ação `ruleset:activate`/`ban:create` | notificação informativa no canal do time |

## Runbooks rápidos

- **Rollback de ruleset**: `POST /v1/admin/rulesets/{versão-anterior}/activate`
  (auditado; matchmaking reponta em tempo real; partidas em andamento seguem).
- **Carta quebrada em produção**: `POST /v1/admin/bans` com motivo → corrigir
  via draft → validar → simular → publicar → ativar → `POST
  /v1/admin/bans/{card}/lift`.
- **Restauração de banco**: `make restore DUMP=...` (prova de processo:
  `make backup-test`).
