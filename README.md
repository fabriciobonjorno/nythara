# Projeto NYTHARA

Card game PvP digital de fantasia sombria — monorepo do MVP Web/PWA.

Pacote de design completo em [docs/design/](docs/design/) (GDD, 130 cartas, 10 Avatares,
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
make setup   # sobe PostgreSQL :55432 + Redis :56379 e aplica migrações
make test    # testes da engine (inclui simulação bot-vs-bot e replays)
make run     # API do jogo em :18080, isolada de outros projetos
make web-dev # cliente em :5173, com proxy REST/WebSocket para :18080
```

Use os três comandos de desenvolvimento em terminais separados (`make setup`
termina após preparar o banco). Overrides explícitos: `VEURUBRO_DEV_API_PORT`,
`VEURUBRO_DEV_DATABASE_URL`, `VEURUBRO_POSTGRES_PORT`,
`VEURUBRO_REDIS_PORT` e `VITE_API_TARGET`.

## Estado atual (alpha 0.10 — Selos Táticos)

O produto servido foi simplificado para um card game direto: um baralho ativo
de 30 cartas (mínimo 8 Assaltos, 8 Guardas e 4 Ritos), Vitalidade como vida e
recurso, turno Compra → Assalto → Guarda reativa → Rito e Avatares sem poder.
A camada tática inclui Selos que pulam a próxima janela de Assalto, Guarda ou
Rito; o texto profissional de cada carta é derivado do efeito executável.
A Arena mostra as cartas no centro, calcula Poder − Prevenção e estilhaça a
perdedora. O baralho fica protegido por 24 horas após salvar.

Treino com bot, casual/ranked por WebSocket, reconexão, histórico, progressão e
LiveOps usam o mesmo motor authoritative. Os gates do `alpha-0.10.2` foram
aprovados em **200 mil partidas**: precons com iniciativa 47,87% e p95 23;
decks variados com iniciativa 49,54% e p95 25. Não houve crash, loop, estado
inválido, comando rejeitado ou divergência de replay. Snapshots
0.9.1/0.10.0/0.10.1 continuam registrados para replay.

- **Fase 0** — monorepo, compose, CI, governança: pronto.
- **Fase 1** — engine determinística legado: pronto e preservado para replay. Essência, Posturas, Eclipse,
  Ressonância (trilha própria + linha do tempo global), Ward/Sangramento/Exposto/
  Véu/Maldição, Fadiga, janelas de Rito/Confronto/Guarda, decisões pendentes
  tipadas, mulligan, limite de mão, replay integral por `seed + decks + comandos`.
- **Fase 2** — DSL de cartas: pronto. Efeitos declarativos versionados em
  `backend/internal/engine/data/effects_alpha.json`, validador anti-loop no
  boot, intérprete + compilador (zero `switch` por carta).
- **Conteúdo versionado**: **130/130 cartas** e os kits legados dos 10
  personagens continuam executáveis em replays ≤0.8.3. No produto 0.10.2,
  **63 cartas** formam o pool competitivo e os 10 personagens são Avatares
  cosméticos; o relatório explicita por que cada uma das outras 68 cartas
  permanece no arquivo.
- **Fase 3** — persistência/API: contas e perfis, auth com refresh rotacionado,
  catálogo, coleção Alpha, decks com validação dupla, Campeões, ruleset,
  temporada, recompensas auditadas, RBAC e idempotência.
- **Fase 4** — battle server authoritative: fila/lobby 1v1, ready, salas actor,
  WebSocket com ticket de uso único, sequência idempotente, timer server-side,
  persistência de comandos/eventos/snapshots, reconexão e espectador read-only.
- **Fase 5** — Web/PWA: landing/login, home, coleção, Avatares, construtor,
  fila, batalha, resultado, replay da sessão, ranked/perfil, tutorial e ajustes.
  Mesa mobile-first com confronto central, Assalto em voo, Guarda reativa,
  impacto/estilhaço, pilhas de compra/descarte, histórico, zoom de carta,
  teclado/toque, preferências de acessibilidade e reconexão.
  As **130/130 cartas** têm ilustrações originais otimizadas, exibidas no catálogo,
  no montador de decks, na ampliação e na mão durante a batalha.
- **Fase 6** — bots e balanceamento: bots random e heuristic, simulador headless
  com matriz 10×10 de Avatares, baralhos preconstruídos legais, métricas por
  Avatar/carta/iniciativa/duração, validação de saúde e replay integral. Pull
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
- **Fase 9** — conteúdo completo: 130 cartas e 10 Campeões (núcleo + Set 2,
  engine), 10 precons oficiais como produto (`GET /v1/catalog/precons` +
  `POST /v1/decks/precon`), tutorial com os 8 fundamentos exigidos e rotação
  de coleção/decks entre versões (`rotate`, fechando o ADR-022).
- **Jogabilidade P0 (alpha-0.8.0)** — modo treino contra o bot heurístico
  (`POST /v1/practice`, mesmo pipeline authoritative, replays inclusos),
  ritmo competitivo (Vitalidade 27–30, Fadiga 6/12/18, Essência máx. 10 e
  Ruptura do Véu — p95 29 rodadas) e rodada
  de balanceamento guiada por 100 mil simulações por iteração
  (ADR-024; fundo da tabela 20–30%→24–42%; dívida Solara documentada).
- **Progressão P1** — rituais diários compatíveis com o Confronto (3/dia,
  sorteio determinístico por jogador), maestria cosmética por Avatar, ranked Elo por temporada e carteira de
  Fragmentos do Véu auditada. Gravação idempotente por partida, derivada só
  dos eventos authoritative (`GET /v1/progress`, `GET /v1/ranked/leaderboard`;
  ADR-025). Home mostra tudo; a fila ganhou treino em um clique.
- **Game feel P1½** — eventos authoritative alimentam voo das cartas para o
  centro, choque Poder × Prevenção, números de dano/cura, estilhaçamento,
  partículas, sons, banners de turno e pressão final, com áudio opcional e
  redução de movimento preservada (ADRs 026 e 044).
- **Fase 10** — Definition of Done: suíte race verde, 100 mil partidas locais
  bit a bit idênticas à baseline, zero TODOs críticos, backup/restore provado,
  scan de vulnerabilidades zerado. O painel web LiveOps está disponível;
  2FA permanece no backlog pós-alpha.

Decisões de regra onde o GDD era ambíguo estão registradas em [DECISIONS.md](DECISIONS.md).

## Nome

“Nythara” é o nome atual do projeto — busca marcária (INPI etc.) antes de lançamento.
