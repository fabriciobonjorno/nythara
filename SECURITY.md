# SECURITY.md — Threat model e postura (Fase 8)

Reporte vulnerabilidades por issue privada ao mantenedor. Não abra issues
públicas com detalhes exploráveis.

## Ameaças do roteiro e mitigação atual

| Ameaça | Mitigação | Resíduo/observação |
| --- | --- | --- |
| WebSocket tampering | Servidor authoritative: cliente envia intenção; engine valida tudo antes de mutar (comando ilegal = zero efeitos, com assert). Tickets de uso único por conexão; `SetReadLimit` 64 KiB. | Payloads são dados, nunca resultado. |
| Replay attack (comandos) | `client_sequence` idempotente por jogador; reenvio devolve o resultado gravado; comando fora de turno/janela rejeitado. | Replays de partida são reconstruíveis por design (não é ataque). |
| Command spam | Rate limit por conexão WS (120/min, rajada 30): excedeu → conexão fechada por política (`ws_command_flood_closed_total`). Timer de ação server-side. | Reconexão legítima permitida. |
| Account takeover | PBKDF2-HMAC-SHA256 com salt aleatório e 600 mil iterações para senhas; tokens aleatórios de 256 bits com hash em repouso; refresh com rotação e histórico; recuperação por link de uso único (30 min), resposta não enumerável e revogação de todas as sessões; troca autenticada de senha; Google Authorization Code + PKCE/state e passe local de uso único; rate limit estrito nas rotas de autenticação. | Sem 2FA no alpha (backlog). Migração oportunista para Argon2id fica para depois do MVP. |
| Inventory fraud | Economia 100% server-side com constraints no banco; concessões só via admin com `Idempotency-Key`; `economy_transactions` auditável. | — |
| Reward duplication | Chave de idempotência com hash do corpo (reuso com corpo diferente = 409); transações atômicas. | — |
| Matchmaking abuse | Fila exige deck próprio, validado, da versão ativa e sem cartas banidas; um jogador não pode estar 2× na fila nem em 2 partidas; rate limit geral por IP. | Detecção de win-trading fica para telemetria ranked (pós-MVP). |
| Botting | Rate limits por IP/conexão; padrões de jogo ficam auditáveis via eventos persistidos. | Detecção comportamental é pós-MVP; ranked exige análise antes de recompensas de topo. |
| Admin compromise | RBAC dupla camada (middleware + serviço); **toda mutação admin grava auditoria na mesma transação** (sem auditoria, sem mudança); mutações idempotentes por construção; elevação a admin só fora da API pública. | Segregar credenciais de admin + 2FA/One-time elevation no hosting (runbook). |

## Controles de plataforma

- **Cabeçalhos**: API JSON com `CSP default-src 'none'`, nosniff, DENY frames,
  no-referrer, COOP/CORP, HSTS sob TLS. A PWA aplica via meta as diretivas de
  carregamento (`connect-src 'self' ws: wss:`; sem script externo) e entrega
  `frame-ancestors 'none'`/`X-Frame-Options: DENY` como cabeçalhos HTTP —
  `frame-ancestors` não tem efeito quando declarado em meta.
- **Rate limiting**: token bucket por chave com poda de memória
  (`internal/security/ratelimit.go`); faixas em `internal/httpapi/hardening.go`.
- **Scanning**: `govulncheck` no CI e local (`make vuln`) — base zerada em
  2026-08-10 (toolchain go1.25.12; grpc 1.82.1; x/text 0.39.0); `npm audit
  --audit-level=high` para a PWA. SAST estático: `go vet` no CI.
- **Backups**: `make backup`/`make restore` (pg_dump custom); prova executável
  `make backup-test` (migra → marca → destrói → restaura → verifica), também
  utilizável na CI. PITR/WAL no provedor de produção.
- **Observabilidade**: OTel traces (OTLP), contadores em `/internal/metrics`
  (requisições, 429, floods de WS, panics); alertas em `ops/observability.md`.
- **Segredos**: nenhum segredo no repositório; configuração via ambiente ou
  arquivo montado (`DATABASE_URL_FILE`). A stack da VPS usa arquivos `0600`
  ignorados pelo Git, nunca interpola a senha no Compose e falha no boot se o
  segredo estiver ausente ou ambíguo. Rotas `/internal/*` são bloqueadas no
  edge. O proxy é confiado por CIDR exato; cabeçalhos de origem enviados
  diretamente pelo cliente são ignorados.
- **OAuth**: client secret existe somente na API. Estado e verifier usam
  cookies HttpOnly/SameSite de cinco minutos; a aplicação persiste o `sub` do
  provedor, nunca usa e-mail como identificador federado e não carrega script
  externo na PWA. O passe de retorno expira em dois minutos e é uso único.

## Garantias da engine

- Determinismo total: `ruleset + seed + decks + command_log` reproduz a
  partida bit a bit (verificado em teste e em 100 mil partidas simuladas).
- RNG da partida no servidor — cliente não influencia sorteios.
- Versões de ruleset imutáveis; partidas antigas executam sob a versão
  original para sempre (ADR-022).
