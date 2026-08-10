# PROMPT MASTER — CLAUDE / CODEX
## Implementar Projeto VÉU RUBRO

Você é o arquiteto principal e engenheiro responsável por implementar um card game PvP digital chamado **Projeto VÉU RUBRO**.

Leia obrigatoriamente antes de codar:
- `GDD.md`
- `TECH_ARCHITECTURE.md`
- `cards_alpha.json`
- `champions_alpha.json`
- `BALANCE_TEST_PLAN.md`
- `RESEARCH_NOTES.md`

## Regra absoluta de propriedade intelectual

Este projeto é uma IP nova. Não copie nem reutilize nomes, cartas, textos, artes, personagens, código, assets ou UI de Heróis e Vampiros, Masters of Cards, Yu-Gi-Oh!, Magic, Hearthstone ou qualquer outro jogo. Use as referências apenas como contexto de gênero. O comportamento implementado deve seguir exclusivamente o GDD deste repositório.

## Meta

Entregar um MVP Web/PWA jogável:
- conta;
- coleção;
- Campeões;
- deck builder;
- tutorial;
- bot;
- lobby;
- matchmaking;
- PvP realtime;
- replay;
- ranked básico;
- painel admin de cartas;
- regras versionadas.

## Stack

Backend:
- Go
- PostgreSQL
- Redis
- WebSocket
- REST
- OpenTelemetry

Frontend:
- React
- TypeScript
- PixiJS
- PWA

Infra:
- Docker
- migrations versionadas
- CI/CD
- ambiente local reproduzível

## Arquivos de governança de IA

Criar e manter na raiz:
- `AGENTS.md`
- `CLAUDE.md`
- `CODEX.md`
- `AI_RULES.md`
- `ARCHITECTURE.md`
- `SECURITY.md`
- `CONTRIBUTING.md`
- `DECISIONS.md`

Esses arquivos devem impedir mudanças arquiteturais silenciosas e registrar decisões relevantes.

# FASE 0 — Bootstrap

1. Inicializar monorepo:
   - `/backend`
   - `/web`
   - `/shared`
   - `/docs`
   - `/ops`
2. Docker Compose local com PostgreSQL e Redis.
3. CI.
4. migrations.
5. lint/test/security.
6. health endpoints.
7. OpenTelemetry.
8. ADR inicial.

Gate:
- setup em um comando;
- CI verde;
- nenhuma regra de jogo ainda.

# FASE 1 — Engine determinística

Implementar a engine como package puro, sem HTTP/WebSocket.

Estruturas:
- GameState
- PlayerState
- CardInstance
- ChampionState
- EclipseState
- ResonanceTrack
- StatusEffect
- ActionWindow
- Command
- Event
- Ruleset

Comandos mínimos:
- Mulligan
- CommitStance
- PlayCard
- ChooseTarget
- PassWindow
- Discard
- Concede

A engine recebe estado + comando e retorna novo estado + eventos.

Requisitos:
- deterministic RNG;
- match seed;
- event ordering;
- no hidden global state;
- replay integral;
- snapshots;
- regras versionadas.

Implementar todos os termos do GDD:
- Essência
- Vitalidade
- Posturas
- Eclipse
- Ressonância
- Assalto
- Guarda
- Rito
- Relíquia
- Manifestação
- Ward
- Sangramento
- Exposto
- Véu
- Maldição
- Recuperar
- Exilar
- Fadiga

Gate:
- golden replay tests;
- property tests;
- fuzz;
- 100% das 80 cartas carregáveis e schema-valid;
- pelo menos 20 cartas totalmente funcionais end-to-end.

# FASE 2 — DSL de cartas

Não codificar cartas por switch gigantes.

Criar DSL/AST segura para efeitos:
- sequence
- conditional
- target
- damage
- heal
- draw
- discard
- ward
- shift_eclipse
- status
- recover
- exile
- cost_modifier
- copy_effect
- action_window
- sigil

Adicionar validator que rejeita:
- recursão sem limite;
- target impossível;
- custo negativo;
- loop de cópia;
- trigger circular.

Converter `cards_alpha.json` para definitions versionadas.

Gate:
- todas as cartas compilam;
- efeitos não suportados aparecem em relatório, nunca são silenciosamente ignorados.

# FASE 3 — Persistência/API

Implementar:
- user/profile;
- auth;
- catalog;
- collection;
- deck;
- validation;
- champions;
- rulesets;
- seasons;
- rewards.

Requisitos:
- RBAC admin/player;
- idempotência;
- migrations reversíveis quando possível;
- constraints no banco;
- nenhuma economia confiando no cliente.

Gate:
- testes de autorização;
- deck ilegal impossível de salvar.

# FASE 4 — Battle server realtime

WebSocket authoritative.

Implementar:
- lobby;
- room;
- matchmaking;
- ready;
- commands;
- server timer;
- reconnect;
- spectator read-only preparado;
- persist commands/events;
- snapshot + catch-up.

Nunca aceitar:
`damage=5`, `draw=2`, `winner=...` vindos do cliente.

Cliente envia apenas intenção:
`play_card(card_instance_id, target_id, client_sequence)`.

Gate:
- reconectar sem perder estado;
- comando duplicado idempotente;
- comando fora de turno rejeitado;
- cheat tests.

# FASE 5 — Web/PWA

Telas:
1. landing/login
2. home
3. coleção
4. Campeões
5. deck builder
6. fila
7. batalha
8. resultado
9. replay
10. ranked/profile
11. tutorial
12. configurações

Batalha:
- Eclipse extremamente visível;
- Ressonância legível;
- feedback de janela de Guarda;
- timeline/log;
- zoom de carta;
- acessibilidade;
- mobile-first responsivo.

Não produzir arte final copiando referências históricas. Use placeholders originais até pipeline de arte.

Gate:
- partida completa Chrome/Safari/mobile;
- teclado/touch;
- reconnect.

# FASE 6 — Bots e balanceamento

Implementar simulator headless.
Bots:
- random
- heuristic

Gerar relatório:
- win rate por Champion;
- duração;
- first-player advantage;
- cartas dominantes;
- loops;
- estados mortos.

Gate:
- 100k simulações sem crash;
- zero divergência determinística;
- relatório salvo como artifact de CI.

# FASE 7 — Admin/LiveOps

Criar admin para:
- card drafts;
- versioning;
- enable/disable ranked;
- ruleset;
- season;
- telemetry;
- emergency disable;
- rollback.

Toda alteração de carta cria nova versão. Partidas antigas continuam reproduzíveis com a versão histórica.

# FASE 8 — Segurança e produção

Threat model:
- WebSocket tampering;
- replay attack;
- command spam;
- account takeover;
- inventory fraud;
- reward duplication;
- matchmaking abuse;
- botting;
- admin compromise.

Implementar:
- rate limits;
- refresh rotation;
- audit;
- secure cookies/tokens conforme arquitetura;
- CSP;
- validation;
- dependency scanning;
- SAST;
- backups;
- monitoring;
- alerts.

# FASE 9 — Conteúdo completo do Alpha

Implementar todas as 80 cartas e 10 Campeões.
Criar preconstructed decks para cada Campeão.
Tutorial deve ensinar:
1. dano;
2. Guarda;
3. Rito;
4. Eclipse;
5. Ressonância;
6. Relíquias;
7. Manifestações;
8. deck building.

# FASE 10 — Definition of Done

Não declarar pronto enquanto:
- CI verde;
- testes da engine;
- 100k simulations;
- PvP real;
- reconnect;
- replays;
- 80 cartas;
- 10 Champions;
- admin;
- telemetry;
- security checklist;
- docs;
- migrations;
- backup/restore test;
- no TODO crítico.

Ao final de cada fase:
1. resumir mudanças;
2. listar arquivos;
3. executar testes;
4. informar métricas;
5. registrar riscos;
6. NÃO avançar se o gate falhar.
