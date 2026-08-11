APP_NAME=fear-and-greed-online
BIN_DIR=bin

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
	@echo "  sync-candles       Sync candles for active universe"
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

.PHONY: tidy upvendors build unit-test integration-test test lint check
tidy:
	$(call _info,Adding missing module requirements...)
	@go mod tidy

upvendors:
	$(call _info,Updating module dependencies...)
	@go get -u ./...
	@go mod tidy

build:
	$(call _info,Building binaries into $(BIN_DIR)/...)
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/api ./cmd/api
	@go build -o $(BIN_DIR)/worker ./cmd/worker
	@go build -o $(BIN_DIR)/migrate ./cmd/migrate
	@go build -o $(BIN_DIR)/smoke ./cmd/smoke
	@go build -o $(BIN_DIR)/acceptance ./cmd/acceptance

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
	@docker compose up -d postgres

postgres-down:
	$(call _info,Stopping PostgreSQL...)
	@docker compose down

postgres-logs:
	@docker compose logs -f postgres

migrate-up:
	$(call _info,Applying migrations...)
	@go run ./cmd/migrate up

migrate-down:
	$(call _info,Rolling back latest migration...)
	@go run ./cmd/migrate down

migrate-reset:
	$(call _info,Resetting database schema...)
	@go run ./cmd/migrate reset

bootstrap: postgres-up migrate-up

################################################################################################################
# Application
################################################################################################################

.PHONY: api worker sync-candles list-active-symbols run-strategies dev
api:
	$(call _info,Starting API...)
	@go run ./cmd/api

worker:
	$(call _info,Starting worker...)
	@go run ./cmd/worker

sync-candles:
	$(call _info,Syncing candles...)
	@go run ./cmd/worker sync-candles

list-active-symbols:
	@go run ./cmd/worker list-active-symbols

run-strategies:
	$(call _info,Running strategies...)
	@go run ./cmd/worker run-strategies

dev: bootstrap api

################################################################################################################
# Verification
################################################################################################################

.PHONY: smoke acceptance verify
smoke:
	$(call _info,Running MVP smoke workflow...)
	@go run ./cmd/smoke

acceptance:
	$(call _info,Running MVP acceptance pass...)
	@go run ./cmd/acceptance

verify: smoke
