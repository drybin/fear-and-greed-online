## 1. Foundation and workspace bootstrap

- [x] 1.1 Create the repository runtime layout for `cmd/api`, `cmd/worker`, `internal`, `web`, `migrations`, and local development tooling
- [x] 1.2 Add environment configuration and dependency management for a local workflow where PostgreSQL runs in Docker and `api`/`worker` can run from the host
- [x] 1.3 Add Docker Compose configuration for PostgreSQL with persistent storage, stable credentials, and developer-friendly defaults
- [x] 1.4 Add migration tooling and a repeatable local bootstrap flow for creating and resetting the Docker-managed database
- [x] 1.5 Add basic health and startup checks so the API and worker fail clearly when configuration or database connectivity is invalid

## 2. Market data platform

- [x] 2.1 Create PostgreSQL migrations for symbols, timeframes, candles, and ingestion job tracking tables
- [x] 2.2 Seed reference data for the July 6, 2026 CoinMarketCap top-50 non-stable snapshot, Binance Spot symbol mappings, and supported timeframes `15m`, `1h`, and `4h`
- [x] 2.3 Implement the market data provider client and mapping layer for normalized candle records
- [x] 2.4 Implement candle repositories with idempotent upsert behavior and indexed range queries
- [x] 2.5 Implement initial one-month backfill and incremental sync logic per active symbol and timeframe, including handling of the latest still-open candle
- [x] 2.6 Implement ingestion job persistence for success and failure visibility
- [x] 2.7 Implement a CLI command that runs candle synchronization for the active Binance Spot-compatible universe

## 3. Independent strategy engine

- [x] 3.1 Create shared strategy domain types, execution input/output contracts, and strategy registry interfaces
- [x] 3.2 Implement migrations and repositories for strategy definitions, strategy runs, signals, and trades
- [x] 3.3 Seed initial strategy definitions for `trend-long-v1` and `breakout-retest-v1` with versioned identifiers and default parameters
- [x] 3.4 Implement shared indicator utilities and warmup/history handling needed by MVP strategies
- [x] 3.5 Implement `trend-long-v1` as the first independent online strategy
- [x] 3.6 Implement `breakout-retest-v1` as the second independent online strategy
- [x] 3.7 Implement strategy run orchestration, persistence, and error handling
- [x] 3.8 Implement deterministic signal deduplication and trade persistence behavior
- [x] 3.9 Implement a CLI command that runs strategy recalculation for one or both MVP strategies after data updates

## 4. API and dashboard

- [x] 4.1 Implement reference data endpoints for symbols, timeframes, and strategies
- [x] 4.2 Implement candle, signal, trade, and strategy run query endpoints with filter validation
- [x] 4.3 Implement a chart-oriented API response shape that returns chart-ready candles and overlays
- [x] 4.4 Create the SPA shell with routing or view-state structure for the dashboard workflow
- [x] 4.5 Implement the chart screen with symbol, timeframe, strategy, and date-range filters for the active seeded universe
- [x] 4.6 Integrate candlestick rendering and signal/trade markers using the selected chart library
- [x] 4.7 Implement detail panels for signal and trade inspection with UTC+7 timestamp presentation
- [x] 4.8 Implement run history and data freshness widgets for the active chart context

## 5. Validation and hardening

- [x] 5.1 Add unit tests for core indicators and strategy fixtures
- [x] 5.2 Add integration tests for repositories, candle sync behavior, and strategy run persistence
- [x] 5.3 Add end-to-end verification for the API plus dashboard chart workflow against seeded data
- [x] 5.4 Review indexes, query plans, and operational logging for the expected MVP symbol set
- [x] 5.5 Document local development, migration, seeding, and operational workflows in the repository README

## 6. MVP closeout

- [x] 6.1 Add a complete dashboard date-range workflow with both `from` and `to` filters, where the default range remains usable without manual input and the active filter state is reflected in chart and detail requests
- [x] 6.2 Add dashboard freshness widgets that show the last successful candle sync, the last successful strategy recalculation, and the latest available candle time for the active symbol/timeframe context
- [x] 6.3 Add a lightweight smoke workflow such as `make smoke` or an equivalent documented command that validates the local MVP path: database up, migrations applied, seed loaded, market sync completed, strategy recalculation completed, and API responding
- [x] 6.4 Finalize README operational guidance for the independent online app, including Docker PostgreSQL startup, migration/bootstrap commands, symbol universe inspection, candle sync, strategy recalculation, and dashboard launch steps
- [x] 6.5 Run an MVP acceptance pass against the frozen top-50 universe and verify that unsupported assets stay inactive, active symbols are queryable from the API/CLI, and both MVP strategies produce inspectable results on seeded data
