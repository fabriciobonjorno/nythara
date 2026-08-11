.PHONY: setup migrate-up migrate-down test test-race lint vuln web-build web-dev run e2e-real sim sim-smoke sim-100k sim-varied-100k calibrate down backup restore backup-test

VEURUBRO_DEV_DATABASE_URL ?= postgres://veurubro:veurubro_dev@localhost:55432/veurubro?sslmode=disable
VEURUBRO_DEV_API_PORT ?= 18080

setup: ## Sobe PostgreSQL e Redis locais
	docker compose up -d --wait
	$(MAKE) migrate-up

migrate-up:
	cd backend && DATABASE_URL="$(VEURUBRO_DEV_DATABASE_URL)" go run ./cmd/migrate up

migrate-down:
	cd backend && DATABASE_URL="$(VEURUBRO_DEV_DATABASE_URL)" go run ./cmd/migrate down

down:
	docker compose down

test:
	cd backend && go test ./...

test-race:
	cd backend && go test -race -count=1 ./...

lint:
	cd backend && go vet ./...

web-build:
	cd web && npm run build

web-dev:
	cd web && npm run dev

run:
	cd backend && DATABASE_URL="$(VEURUBRO_DEV_DATABASE_URL)" PORT="$(VEURUBRO_DEV_API_PORT)" go run ./cmd/server

e2e-real: ## PvP + treino naturais: contas, decks, fila, WebSocket, ocultação e rating
	cd backend && go run ./cmd/e2e-duel -base-url "http://127.0.0.1:$(VEURUBRO_DEV_API_PORT)"

sim: sim-smoke

sim-smoke:
	cd backend && go run ./cmd/sim -games 1000 -json artifacts/smoke.json -markdown artifacts/smoke.md

sim-100k:
	cd backend && go run ./cmd/sim -games 100000 -json artifacts/balance-report.json -markdown artifacts/balance-report.md

sim-varied-100k: ## gate de saúde de expansão: decks sorteados do pool inteiro
	cd backend && go run ./cmd/sim -games 100000 -decks varied -json artifacts/balance-varied.json -markdown artifacts/balance-varied.md

calibrate: ## varre parâmetros de ritmo e imprime a grade (ver ADR-054)
	cd backend && go run ./cmd/calibrate -games 2500 -vitality 44,56,64 -guard-bonus 2,3 -pressure 58

# Regenera goldens após revisão manual de mudança de regra
golden:
	cd backend && go test ./internal/engine/ -run TestScriptedMatchGolden -update

vuln:
	cd backend && go run golang.org/x/vuln/cmd/govulncheck@latest ./...

backup: ## DATABASE_URL=... make backup
	ops/backup.sh

restore: ## DATABASE_URL=... make restore DUMP=backups/arquivo.dump
	ops/restore.sh $(DUMP)

backup-test: ## prova de backup/restore num Postgres efêmero
	ops/backup-restore-test.sh
