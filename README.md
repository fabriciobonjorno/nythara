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

## Estado atual (alpha 0.4)

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
- Próximo: Admin/LiveOps (Fase 7).

Decisões de regra onde o GDD era ambíguo estão registradas em [DECISIONS.md](DECISIONS.md).

## Nome

“Véu Rubro” é nome de trabalho — busca marcária (INPI etc.) antes de lançamento.
