# CLAUDE.md — Projeto Nythara

Card game PvP digital. Fonte de verdade de design: `docs/design/GDD.md`.
Decisões de regra: `DECISIONS.md` (mudou regra → novo ADR + `make golden`).

## Comandos

```bash
make test        # suíte completa da engine
make test-race   # com race detector (obrigatório antes de finalizar)
make lint        # go vet
make golden      # regenera goldens (só após revisar a mudança de regra)
make setup       # Postgres + Redis locais
```

## Arquitetura (regras rígidas)

- `backend/internal/engine` é **pura**: sem HTTP/WebSocket/IO/relógio. Entrada:
  estado + comando; saída: novo estado + eventos. Nunca importe rede aqui.
- Determinismo absoluto: nada de `math/rand`, `time.Now()` ou iteração de map
  em lógica de jogo. Use o `RNG` da partida; eventos têm ordem estável.
- Cliente envia intenção; a engine decide resultado. Nunca aceite
  `damage=`/`winner=` de fora.
- Cartas são estruturas declarativas em `impls.go` (precursor da DSL da Fase 2).
  **Nunca** um `if card_id ==` espalhado. Carta sem implementação é rejeitada
  explicitamente e aparece em `ImplementationReport()` — jamais ignorada.
- Comando rejeitado não pode mutar estado nem emitir eventos (há assert disso).

## IP

Universo 100% original. Não copiar nomes, textos, arte ou código de Heróis e
Vampiros, Masters of Cards, Magic, Hearthstone, Yu-Gi-Oh! etc. Referências são
apenas contexto de gênero (`docs/design/RESEARCH_NOTES.md`).

## Gates por fase

Seguir `docs/design/PROMPT_MASTER_CODEX_CLAUDE.md`. Não avançar de fase com
gate falhando; ao mexer em regra, rodar também a simulação
(`TestRandomGamesEndAndReplayDeterministically`).
