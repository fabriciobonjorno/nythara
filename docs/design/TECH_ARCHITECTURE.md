# Arquitetura Técnica — Nythara

## Direção

Para o MVP, usar **monólito modular em Go** no backend e **React + TypeScript + PixiJS** no cliente web/PWA.

### Backend
- Go
- PostgreSQL
- Redis
- WebSocket
- REST/JSON para conta, coleção, deck e catálogo
- protocolo de eventos versionado para batalha
- worker assíncrono para matchmaking, recompensas, e-mails e jobs de manutenção

### Front
- React + TypeScript
- PixiJS para mesa, cartas e animações
- Zustand ou store equivalente para UI local
- TanStack Query para dados de API
- PWA como primeira entrega
- responsivo desktop/mobile
- cliente nunca decide resultado de carta

## Módulos do backend

- identity
- players
- champions
- card_catalog
- collections
- decks
- matchmaking
- battles
- rules_engine
- rewards
- seasons
- ranked
- replay
- telemetry
- moderation
- admin

## Engine determinística

Separar a engine do transporte de rede.

Entrada:
- RulesetVersion
- MatchSeed
- Deck A / Deck B
- Champions
- PlayerCommand

Saída:
- DomainEvents[]
- novo GameState

Exemplos de eventos:
- RoundStarted
- CardDrawn
- StanceCommitted
- StanceRevealed
- CardPlayed
- EssenceSpent
- VitalityChanged
- EclipseShifted
- EclipseTriggered
- ResonanceSigilAdded
- GuardWindowOpened
- DamagePrevented
- CardDiscarded
- CardExiled
- MatchEnded

Toda partida deve poder ser recriada apenas com:
`ruleset_version + seed + decks + command_log`.

## Segurança

- servidor authoritative;
- cliente envia intenção, nunca resultado;
- IDs públicos opacos;
- rate limit;
- autenticação curta + refresh rotacionado;
- WebSocket autenticado;
- validação de sequência/turno;
- anti-replay de comando;
- idempotency key;
- número incremental de comando por jogador;
- logs estruturados;
- trilha de auditoria para economia;
- assinatura/Hash do replay;
- nenhum segredo no cliente;
- regras e catálogo versionados.

## Banco — entidades principais

users
player_profiles
champions
card_definitions
card_versions
card_sets
player_cards
decks
deck_cards
rulesets
seasons
matches
match_players
match_commands
match_events
match_snapshots
ranked_ratings
rewards
transactions
cosmetics
player_cosmetics
balance_metrics

## Realtime

Fluxo:
1. matchmaking cria `match`;
2. battle service instancia engine;
3. clientes entram em sala WS;
4. comando recebe `client_sequence`;
5. servidor valida;
6. engine processa;
7. eventos são persistidos;
8. broadcast dos eventos;
9. snapshots periódicos;
10. reconexão recebe último snapshot + eventos posteriores.

## Escala

Primeiro: monólito modular + PostgreSQL + Redis.

Extrair apenas se telemetria justificar:
- matchmaking;
- battle workers;
- replay/analytics.

Evitar microserviços prematuros.

## Testes indispensáveis

- unitário por efeito de carta;
- property-based tests da engine;
- golden tests de partidas;
- fuzzing de comandos;
- simulação bot-vs-bot;
- teste de reconexão;
- concorrência;
- teste de versões antigas de regras;
- carga de salas WebSocket;
- economia/idempotência.

## Content pipeline

Carta não deve exigir `if card_id == ...` espalhado no código.

Usar DSL/AST de efeitos validada:
- DealDamage
- Heal
- Draw
- Discard
- GainWard
- ShiftEclipse
- AddStatus
- RemoveStatus
- Recover
- Exile
- ModifyCost
- CopyEffect
- OpenActionWindow
- AddSigil
- Conditional

Cartas complexas podem usar handlers especializados, mas devem ser exceção e registrados explicitamente.

## Admin/LiveOps

Painel:
- visão operacional de jogadores, atividade e partidas recentes;
- moderação de jogadores com revogação de sessão e auditoria;
- convite de admin por token opaco, emitido somente pelo owner;
- criar draft de carta;
- validar schema;
- simular;
- publicar nova versão;
- desativar em ranked;
- consultar win rate/pick rate;
- comparar antes/depois;
- rollback de ruleset;
- configurar temporada;
- ban temporário emergencial sem apagar histórico.

## Deploy

- Docker
- CI: lint + test + race + security scan + migrations check
- staging
- canary de ruleset
- observabilidade: OpenTelemetry + Prometheus/Grafana ou equivalente
- backups PITR do PostgreSQL
- Redis nunca é fonte de verdade da economia
