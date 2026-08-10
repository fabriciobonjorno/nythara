# CONTRIBUTING.md

## Setup

```bash
make setup       # Postgres + Redis
make test-race   # deve passar antes de qualquer PR
```

## Convenções

- Go padrão (`gofmt`, `go vet`); testes de comportamento por carta em
  `cards_test.go`; toda carta nova exige teste + entrada no relatório.
- Mudança de regra: ADR em `DECISIONS.md` + `make golden` com diff revisado.
- Commits pequenos e descritivos; PRs referenciam a fase do roteiro
  (`docs/design/PROMPT_MASTER_CODEX_CLAUDE.md`).
- Textos de jogo em pt-BR; código e identificadores em inglês, termos de
  domínio do GDD preservados (Assalto, Guarda, Rito...).
