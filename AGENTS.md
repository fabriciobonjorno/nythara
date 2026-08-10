# AGENTS.md

Instruções para qualquer agente de IA neste repositório.

1. Leia `CLAUDE.md` (regras de arquitetura e comandos) e `AI_RULES.md` (limites).
2. Design vem de `docs/design/`; ambiguidade de regra → registrar ADR em
   `DECISIONS.md`, nunca decidir silenciosamente.
3. Engine determinística e pura (`backend/internal/engine`); testes verdes com
   `make test-race` antes de concluir qualquer tarefa.
4. Mudanças arquiteturais exigem ADR prévio em `DECISIONS.md`.
5. IP original — nada copiado de jogos existentes.
