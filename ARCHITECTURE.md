# ARCHITECTURE.md

Direção completa em `docs/design/TECH_ARCHITECTURE.md`. Estado atual:

- **Monólito modular em Go** (`backend/`). `internal/engine` é a engine
  determinística pura (estado + comando → estado + eventos), com catálogo
  embutido (80 cartas/10 Campeões validados no load), bots random/heuristic e
  replay integral. `internal/sim` executa a matriz headless 10×10 em paralelo,
  com seeds estáveis, validação de zonas e relatório reproduzível JSON/Markdown.
- `cmd/server`: health checks + relatório de implementação (stdlib apenas;
  OpenTelemetry entra quando houver API real, Fase 3).
- **PostgreSQL** é a fonte de verdade da Fase 3: usuários/perfis, sessões
  opacas rotacionadas, catálogo versionado, coleção, decks, temporadas,
  recompensas, idempotência e auditoria de economia. A legalidade de deck é
  validada pela aplicação e por constraint trigger diferida. Redis permanece
  reservado à Fase 4 e nunca será fonte de verdade de economia.
- `internal/app` contém casos de uso; `internal/httpapi` expõe REST/JSON com
  RBAC player/admin; `internal/storage` implementa os contratos no PostgreSQL.
- Requisições HTTP geram spans OpenTelemetry; exportação OTLP é ativada por
  `OTEL_EXPORTER_OTLP_ENDPOINT` e fica no-op no desenvolvimento sem coletor.
- `web/`: PWA React+TypeScript+Vite. React Router organiza URLs, TanStack Query
  mantém estado remoto, Zustand guarda sessão/preferências e PixiJS desenha
  somente a atmosfera da mesa; ações continuam em HTML semântico e o servidor
  segue como única autoridade de regras.
- `shared/schemas`: JSON Schemas do catálogo, contrato entre backend, admin e
  pipeline de conteúdo.

Fluxo de batalha implementado (Fase 4): matchmaking FIFO local ao monólito,
ticket WebSocket de uso único, uma goroutine/actor por sala, comando com
`client_sequence`, transação de comando+eventos+snapshot e restauração por
snapshot + catch-up. O transporte sempre produz visão redigida; somente a
engine pura vê o estado completo. O `sync` também devolve a última sequência
confirmada do jogador, permitindo reenvio idempotente da intenção pendente pelo
PWA após uma queda.

O gate de balanceamento da Fase 6 roda uma amostra curta em pull requests e
100 mil partidas no job noturno/manual. Cada partida é reproduzida integralmente
e comparada por snapshot + log; crashes, loops, estados mortos/inválidos,
comandos rejeitados ou divergências fazem o job falhar e o relatório é publicado
como artifact de CI.
