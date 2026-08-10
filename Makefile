.PHONY: setup migrate-up migrate-down test test-race lint web-build web-dev run sim sim-smoke sim-100k down

setup: ## Sobe PostgreSQL e Redis locais
	docker compose up -d --wait
	$(MAKE) migrate-up

migrate-up:
	cd backend && go run ./cmd/migrate up

migrate-down:
	cd backend && go run ./cmd/migrate down

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
	cd backend && go run ./cmd/server

sim: sim-smoke

sim-smoke:
	cd backend && go run ./cmd/sim -games 1000 -json artifacts/smoke.json -markdown artifacts/smoke.md

sim-100k:
	cd backend && go run ./cmd/sim -games 100000 -json artifacts/balance-report.json -markdown artifacts/balance-report.md

# Regenera goldens após revisão manual de mudança de regra
golden:
	cd backend && go test ./internal/engine/ -run TestScriptedMatchGolden -update
