# shared/

Contratos compartilhados entre backend, web e pipeline de conteúdo.

- `schemas/card.schema.json` — schema do catálogo de cartas (`docs/design/cards_alpha.json`).
- `schemas/champion.schema.json` — schema dos Campeões.

A engine Go replica estas validações no load (falha de schema = pânico no
boot, nunca carta meio-carregada). Quando a DSL de efeitos (Fase 2) existir,
seu schema também viverá aqui.
