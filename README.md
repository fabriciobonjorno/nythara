# Projeto VÉU RUBRO

Card game PvP digital de fantasia sombria — monorepo do MVP Web/PWA.

Pacote de design completo em [docs/design/](docs/design/) (GDD, 80 cartas, 10 Campeões,
arquitetura, plano de balanceamento).

Contrato da persistência/API: [docs/API.md](docs/API.md).

## Estrutura

- `backend/` — Go. Engine determinística de regras (`internal/engine`) e servidor HTTP mínimo.
- `web/` — PWA React + TypeScript + Vite, mesa visual PixiJS e controles HTML acessíveis.
- `shared/` — JSON Schemas do catálogo e contratos de protocolo.
- `docs/` — design (`docs/design/`) e decisões (`DECISIONS.md` na raiz).
- `ops/` — infraestrutura local e de deploy.

## Começando

Requisitos: Go 1.25+, Node 20+, Docker.

```bash
make setup   # sobe PostgreSQL + Redis locais
make test    # testes da engine (inclui simulação bot-vs-bot e replays)
make run     # servidor de health/relatórios em :8080
make web-dev # cliente em :5173, com proxy REST/WebSocket para :8080
```

## Estado atual (alpha 0.5)

- **Fase 0** — monorepo, compose, CI, governança: pronto.
- **Fase 1** — engine determinística: pronto. Essência, Posturas, Eclipse,
  Ressonância (trilha própria + linha do tempo global), Ward/Sangramento/Exposto/
  Véu/Maldição, Fadiga, janelas de Rito/Confronto/Guarda, decisões pendentes
  tipadas, mulligan, limite de mão, replay integral por `seed + decks + comandos`.
- **Fase 2** — DSL de cartas: pronto. Efeitos declarativos versionados em
  `backend/internal/engine/data/effects_alpha.json`, validador anti-loop no
  boot, intérprete + compilador (zero `switch` por carta).
- **Conteúdo de regras completo** (antecipando a Fase 9): **80/80 cartas** e
  **10/10 Campeões** (passivas, Formas de Eclipse e ultimates) funcionais
  end-to-end — incluindo cópias, janela de reação a Ritos (counter), janelas
  extras de Assalto, habilidades ativadas (`activate`/`ultimate`) e cartas de
  informação oculta. Superfície de comandos estável para as Fases 3–4.
- **Fase 3** — persistência/API: contas e perfis, auth com refresh rotacionado,
  catálogo, coleção Alpha, decks com validação dupla, Campeões, ruleset,
  temporada, recompensas auditadas, RBAC e idempotência.
- **Fase 4** — battle server authoritative: fila/lobby 1v1, ready, salas actor,
  WebSocket com ticket de uso único, sequência idempotente, timer server-side,
  persistência de comandos/eventos/snapshots, reconexão e espectador read-only.
- **Fase 5** — Web/PWA: landing/login, home, coleção, Campeões, deck builder,
  fila, batalha, resultado, replay da sessão, ranked/perfil, tutorial e ajustes.
  Mesa mobile-first com Eclipse, Ressonância, janela de Guarda, histórico,
  zoom de carta, teclado/toque, preferências de acessibilidade e reconexão.
- **Fase 6** — bots e balanceamento: bots random e heuristic, simulador headless
  com matriz 10×10 de Campeões, decks preconstruídos legais, métricas por
  Campeão/carta/iniciativa/duração, validação de saúde e replay integral. Pull
  requests rodam smoke; CI noturna/manual salva o relatório de 100 mil partidas.
- **Fase 7** — Admin/LiveOps: rulesets versionados executáveis (engine compila
  e registra qualquer versão publicada; replays históricos preservados),
  drafts de carta com validar/simular/publicar, ativação com rollback,
  bans emergenciais de ranked, temporadas, telemetria de partidas e trilha de
  auditoria em toda mutação — via API admin com RBAC (página web do painel
  fica para o ciclo de UI).
- **Fase 8** — segurança e produção: threat model vivo em `SECURITY.md`,
  rate limiting (HTTP por IP, autenticação estrita, flood de comandos WS),
  cabeçalhos de segurança + CSP (API e PWA), `govulncheck` bloqueante na CI
  (base zerada), backups com prova executável de restauração
  (`make backup-test`), métricas em `/internal/metrics` e runbook de alertas
  em `ops/observability.md`.
- **Fase 9** — conteúdo completo: 80 cartas e 10 Campeões (desde o ciclo da
  engine), 10 precons oficiais como produto (`GET /v1/catalog/precons` +
  `POST /v1/decks/precon`), tutorial com os 8 fundamentos exigidos e rotação
  de coleção/decks entre versões (`rotate`, fechando o ADR-022).
- **Jogabilidade P0 (alpha-0.5.0)** — modo treino contra o bot heurístico
  (`POST /v1/practice`, mesmo pipeline authoritative, replays inclusos),
  ritmo mais rápido (Fadiga ×2, Essência máx. 10 — p95 75→59 rodadas) e 1ª
  rodada de balanceamento guiada por 100 mil simulações por iteração
  (ADR-024; fundo da tabela 20–30%→24–42%; dívida Solara documentada).
- **Progressão P1** — rituais diários (3/dia, sorteio determinístico por
  jogador), maestria por Campeão, ranked Elo por temporada e carteira de
  Fragmentos do Véu auditada. Gravação idempotente por partida, derivada só
  dos eventos authoritative (`GET /v1/progress`, `GET /v1/ranked/leaderboard`;
  ADR-025). Home mostra tudo; a fila ganhou treino em um clique.
- **Fase 10** — Definition of Done: suíte race verde, 100 mil partidas locais
  bit a bit idênticas à baseline, zero TODOs críticos, backup/restore provado,
  scan de vulnerabilidades zerado. MVP Alpha completo; pendências honestas:
  página web do painel admin e 2FA (backlog pós-alpha).

Decisões de regra onde o GDD era ambíguo estão registradas em [DECISIONS.md](DECISIONS.md).

## Nome

“Véu Rubro” é nome de trabalho — busca marcária (INPI etc.) antes de lançamento.
