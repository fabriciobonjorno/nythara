# SECURITY.md

Threat model completo: `docs/design/PROMPT_MASTER_CODEX_CLAUDE.md` (Fase 8).

Garantias já embutidas na engine (Fase 1):

- Servidor authoritative: comandos são **intenção**; validação total antes de
  qualquer mutação (comando ilegal = zero efeitos, com assert).
- Comando fora de turno/janela/fase rejeitado com código de erro tipado.
- Replay auditável: `ruleset + seed + decks + command_log` reproduz a partida
  bit a bit (verificado em teste a cada simulação).
- RNG próprio da partida — cliente não influencia sorteios.

Garantias adicionadas na Fase 3: tokens opacos armazenados apenas por hash,
refresh rotacionado, senhas PBKDF2 com salt, RBAC admin/player, idempotência em
escritas de decks/recompensas, coleção alterada apenas por transações do
servidor e auditoria de economia.

Garantias adicionadas na Fase 4: tickets WebSocket aleatórios e de uso único,
slot derivado da identidade (nunca do payload), schema fechado de intenções,
sequência monotônica/idempotente, espectador read-only, redação de informação
oculta e persistência atômica de comando/eventos/snapshot.

Na Fase 5, o service worker não cacheia APIs autenticadas, tokens ficam apenas
em `sessionStorage`, intenção pendente é reenviada com a mesma sequência após
reconexão e o CI faz build tipado + `npm audit`. A camada PixiJS é decorativa;
nenhuma ação, informação essencial ou validação de regra depende do canvas.

Pendências (Fase 8): rate limit, CSP, migração do refresh token para cookie
HttpOnly/SameSite, backups PITR e hardening operacional/multi-instância.

Reporte vulnerabilidades por issue privada ao mantenedor.
