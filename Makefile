APP_NAME=fear-and-greed-online
BIN_DIR=bin
API_BIN=$(BIN_DIR)/$(APP_NAME)-api
WORKER_BIN=$(BIN_DIR)/$(APP_NAME)-worker
MIGRATE_BIN=$(BIN_DIR)/$(APP_NAME)-migrate
SMOKE_BIN=$(BIN_DIR)/$(APP_NAME)-smoke
ACCEPTANCE_BIN=$(BIN_DIR)/$(APP_NAME)-acceptance
DOCKER_COMPOSE := $(shell if docker compose version >/dev/null 2>&1; then echo "docker compose"; elif command -v docker-compose >/dev/null 2>&1; then echo "docker-compose"; else echo "docker compose"; fi)

define _info
	@echo "==> $(1)"
endef

.DEFAULT_GOAL := help

################################################################################################################
# Help
################################################################################################################

.PHONY: help
help:
	@echo "$(APP_NAME) make targets"
	@echo ""
	@echo "Dev hygiene:"
	@echo "  tidy               Run go mod tidy"
	@echo "  build              Build cmd binaries into $(BIN_DIR)/"
	@echo "  build-runtime      Build API and worker binaries for runtime commands"
	@echo "  unit-test          Run fast unit tests"
	@echo "  integration-test   Run full test suite (requires PostgreSQL)"
	@echo "  test               Run unit-test and integration-test"
	@echo "  lint               Run golangci-lint"
	@echo "  check              Run tidy, build, unit-test, and lint"
	@echo ""
	@echo "Local infrastructure:"
	@echo "  postgres-up        Start Docker PostgreSQL"
	@echo "  postgres-down      Stop Docker PostgreSQL"
	@echo "  postgres-logs      Follow PostgreSQL logs"
	@echo "  migrate-up         Apply SQL migrations"
	@echo "  migrate-down       Roll back the latest migration"
	@echo "  migrate-reset      Drop app schema and re-apply migrations"
	@echo "  bootstrap          Start PostgreSQL and apply migrations"
	@echo ""
	@echo "Application:"
	@echo "  api                Start API and dashboard"
	@echo "  worker             Run worker bootstrap check"
	@echo "  sync-candles       Sync candles for active universe (one worker run)"
	@echo "  sync-active        Sync each active symbol separately (resilient)"
	@echo "  list-active-symbols"
	@echo "  run-strategies     Recalculate active strategies"
	@echo ""
	@echo "Verification:"
	@echo "  smoke              Run local MVP smoke workflow"
	@echo "  acceptance         Run frozen universe acceptance pass"
	@echo "  verify             Alias for smoke"
	@echo ""
	@echo "Composites:"
	@echo "  dev                bootstrap + api"

################################################################################################################
# Dev hygiene
################################################################################################################

.PHONY: tidy upvendors build build-runtime unit-test integration-test test lint check
tidy:
	$(call _info,Adding missing module requirements...)
	@go mod tidy

upvendors:
	$(call _info,Updating module dependencies...)
	@go get -u ./...
	@go mod tidy

build: build-runtime
	$(call _info,Building binaries into $(BIN_DIR)/...)
	@mkdir -p $(BIN_DIR)
	@go build -p 1 -o $(MIGRATE_BIN) ./cmd/migrate
	@go build -p 1 -o $(SMOKE_BIN) ./cmd/smoke
	@go build -p 1 -o $(ACCEPTANCE_BIN) ./cmd/acceptance

build-runtime:
	$(call _info,Building API and worker binaries into $(BIN_DIR)/...)
	@mkdir -p $(BIN_DIR)
	@go build -p 1 -o $(API_BIN) ./cmd/api
	@go build -p 1 -o $(WORKER_BIN) ./cmd/worker

unit-test:
	$(call _info,Running unit tests...)
	@go test ./internal/strategy/... ./internal/domain/...

integration-test:
	$(call _info,Running integration tests...)
	@go test ./...

test: unit-test integration-test

lint:
	$(call _info,Running linters...)
	@golangci-lint run -v --timeout 3m0s ./...

check: tidy build unit-test lint

################################################################################################################
# Local infrastructure
################################################################################################################

.PHONY: postgres-up postgres-down postgres-logs migrate-up migrate-down migrate-reset bootstrap
postgres-up:
	$(call _info,Starting PostgreSQL...)
	@$(DOCKER_COMPOSE) up -d postgres

postgres-down:
	$(call _info,Stopping PostgreSQL...)
	@$(DOCKER_COMPOSE) down

postgres-logs:
	@$(DOCKER_COMPOSE) logs -f postgres

migrate-up: require-migrate-binary
	$(call _info,Applying migrations...)
	@$(MIGRATE_BIN) up

migrate-down: require-migrate-binary
	$(call _info,Rolling back latest migration...)
	@$(MIGRATE_BIN) down

migrate-reset: require-migrate-binary
	$(call _info,Resetting database schema...)
	@$(MIGRATE_BIN) reset

bootstrap: postgres-up migrate-up

################################################################################################################
# Application
################################################################################################################

.PHONY: api worker sync-candles sync-active list-active-symbols run-strategies dev require-api-binary require-worker-binary require-migrate-binary require-smoke-binary require-acceptance-binary
require-api-binary:
	@test -x "$(API_BIN)" || { echo "Missing API binary; run make build first" >&2; exit 1; }

require-worker-binary:
	@test -x "$(WORKER_BIN)" || { echo "Missing worker binary; run make build first" >&2; exit 1; }

require-migrate-binary:
	@test -x "$(MIGRATE_BIN)" || { echo "Missing migration binary; run make build first" >&2; exit 1; }

require-smoke-binary:
	@test -x "$(SMOKE_BIN)" || { echo "Missing smoke binary; run make build first" >&2; exit 1; }

require-acceptance-binary:
	@test -x "$(ACCEPTANCE_BIN)" || { echo "Missing acceptance binary; run make build first" >&2; exit 1; }

api: require-api-binary
	$(call _info,Starting API...)
	@$(API_BIN)

worker: require-worker-binary
	$(call _info,Starting worker...)
	@$(WORKER_BIN)

sync-candles: require-worker-binary
	$(call _info,Syncing candles...)
	@$(WORKER_BIN) sync-candles

sync-active: require-worker-binary
	$(call _info,Syncing candles per active symbol...)
	@./scripts/sync-active-candles.sh

list-active-symbols: require-worker-binary
	@$(WORKER_BIN) list-active-symbols

run-strategies: require-worker-binary
	$(call _info,Running strategies...)
	@$(WORKER_BIN) run-strategies

dev: bootstrap api

################################################################################################################
# Verification
################################################################################################################

.PHONY: smoke acceptance verify
smoke: require-smoke-binary
	$(call _info,Running MVP smoke workflow...)
	@$(SMOKE_BIN)

acceptance: require-acceptance-binary
	$(call _info,Running MVP acceptance pass...)
	@$(ACCEPTANCE_BIN)

verify: smoke
