# fear-and-greed-online

Standalone online application for market data ingestion, independent strategy execution, and chart-based signal inspection.

The MVP covers a frozen July 6, 2026 CoinMarketCap top-50 non-stable snapshot: **50 seeded symbols**, **38 active Binance Spot pairs**, two long-only strategies, candle sync from Binance, and a single-page dashboard for chart inspection.

## Prerequisites

- Go 1.22+
- Docker (for local PostgreSQL)
- Network access to Binance Spot API for candle sync

Copy environment defaults before the first run:

```bash
cp .env.example .env
```

## Quick start

### First-time bootstrap

```bash
make postgres-up
make migrate-up
go run ./cmd/worker sync-candles --asset BTC
go run ./cmd/worker run-strategies --strategy trend-long-v1 --asset BTC
make api
```

Open the dashboard at [http://localhost:8080/](http://localhost:8080/).

### Verify the full MVP path

After bootstrap, or on a clean machine, run the automated smoke workflow:

```bash
make smoke
```

`make smoke` starts PostgreSQL, applies migrations, verifies seed data, syncs `BTC`, recalculates `trend-long-v1`, starts a temporary API on port `18080`, and checks the core endpoints plus the dashboard shell.

### Daily update loop

When PostgreSQL is already running and migrations are applied:

```bash
make sync-candles          # or scope with --asset BTC
make run-strategies        # or scope with --strategy / --asset
make api
```

Re-run sync and strategies after code or parameter changes. The dashboard auto-refreshes chart data when filters change.

## Make targets

Run `make help` for the full target list.

### Dev hygiene

| Target | Purpose |
| --- | --- |
| `tidy` | Run `go mod tidy` |
| `build` | Build binaries into `bin/` |
| `unit-test` | Fast unit tests without PostgreSQL |
| `integration-test` | Full `go test ./...` suite (requires PostgreSQL on `5433`) |
| `test` | `unit-test` + `integration-test` |
| `lint` | Run `golangci-lint` |
| `check` | `tidy`, `build`, `unit-test`, and `lint` |

### Local infrastructure and app

| Target | Command | Purpose |
| --- | --- | --- |
| `bootstrap` | `postgres-up` + `migrate-up` | Start DB and apply migrations |
| `dev` | `bootstrap` + `api` | Bootstrap then start API |
| `postgres-up` | `docker compose up -d postgres` | Start local PostgreSQL on host port `5433` |
| `postgres-down` | `docker compose down` | Stop local PostgreSQL |
| `postgres-logs` | `docker compose logs -f postgres` | Follow PostgreSQL logs |
| `migrate-up` | `go run ./cmd/migrate up` | Apply SQL migrations and seed data |
| `migrate-down` | `go run ./cmd/migrate down` | Roll back the latest applied migration |
| `migrate-reset` | `go run ./cmd/migrate reset` | Drop application schema and re-apply migrations |
| `sync-candles` | `go run ./cmd/worker sync-candles` | Sync candles for the active universe |
| `run-strategies` | `go run ./cmd/worker run-strategies` | Recalculate all active strategies |
| `list-active-symbols` | `go run ./cmd/worker list-active-symbols` | Print active symbol universe |
| `api` | `go run ./cmd/api` | Start API and dashboard on `APP_PORT` |
| `worker` | `go run ./cmd/worker` | Run worker bootstrap / connectivity check |
| `smoke` | `go run ./cmd/smoke` | End-to-end local MVP verification |
| `acceptance` | `go run ./cmd/acceptance` | MVP acceptance pass for frozen top-50 universe |
| `verify` | alias for `smoke` | Quick MVP verification |

Worker commands also accept scoped flags:

```bash
go run ./cmd/worker sync-candles --asset BTC
go run ./cmd/worker run-strategies --strategy trend-long-v1 --asset BTC
go run ./cmd/worker run-strategies --strategy breakout-retest-v1 --asset ETH
```

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `APP_PORT` | `8080` | API and dashboard port |
| `POSTGRES_HOST` | `127.0.0.1` | PostgreSQL host |
| `POSTGRES_PORT` | `5433` | Host port for Docker PostgreSQL (avoids conflicts with a local Postgres on `5432`) |
| `POSTGRES_DB` | `fear_and_greed_online` | Database name |
| `POSTGRES_USER` | `fear_and_greed` | Database user |
| `POSTGRES_PASSWORD` | `fear_and_greed` | Database password |
| `BINANCE_BASE_URL` | `https://api.binance.com` | Binance Spot REST API |
| `SYNC_BACKFILL_DAYS` | `30` | Initial candle backfill window when no data exists yet |

The API, worker, and migrate commands auto-apply pending SQL migrations on startup. The API fails fast when PostgreSQL is unreachable or migrations cannot be applied.

## Database bootstrap

Start and stop local PostgreSQL:

```bash
make postgres-up
make postgres-down
make postgres-logs
```

Apply schema and seed data:

```bash
make migrate-up
```

Rollback the latest migration or reset the local application schema:

```bash
make migrate-down
make migrate-reset
```

`migrate reset` drops application tables and reapplies all migrations. It is the fastest way to return to the seeded MVP state without removing the Docker volume.

Migrations create the market-data and strategy-engine schema, then seed:

- timeframes: `15m`, `1h`, `4h`
- 50 top-market-cap symbols with Binance pair metadata
- 38 active symbols and 12 inactive symbols
- strategies `trend-long-v1` and `breakout-retest-v1`

To fully wipe the Docker volume instead, run:

```bash
make postgres-down
docker volume rm fear-and-greed-online_postgres_data
make postgres-up
make migrate-up
```

## Symbol universe

Inspect the active tradable universe:

```bash
make list-active-symbols
curl http://localhost:8080/symbols/active
curl http://localhost:8080/symbols/all
```

The seeded snapshot contains 50 assets. Twelve are marked inactive because they are not available as Binance Spot pairs in the MVP mapping:

`HYPE`, `LEO`, `XMR`, `CC`, `CRO`, `OKB`, `M`, `MNT`, `PI`, `BGB`, `KCS`, `KAS`

Inactive symbols remain in the database for auditability but are excluded from sync, strategy runs, and dashboard symbol filters.

## Candle sync

Sync one asset first for local iteration:

```bash
go run ./cmd/worker sync-candles --asset BTC
```

Sync the full active universe:

```bash
make sync-candles
```

Full-universe sync is slower and makes many Binance API requests across 38 symbols and 3 timeframes. Prefer scoped sync during development.

Behavior:

- On first run for a symbol/timeframe, backfills the last `SYNC_BACKFILL_DAYS` days.
- On later runs, fetches only new closed candles since the last stored bar.
- Writes ingestion job records used by freshness widgets and `/freshness`.

## Strategy recalculation

Run all active strategies across the active universe:

```bash
make run-strategies
```

Run one strategy and/or one asset:

```bash
go run ./cmd/worker run-strategies --strategy trend-long-v1 --asset BTC
go run ./cmd/worker run-strategies --strategy breakout-retest-v1 --asset ETH
```

Available strategies (seeded in migration `000005`):

| Slug | Timeframes | Description |
| --- | --- | --- |
| `trend-long-v1` | `15m`, `1h`, `4h` | SMA trend long with close cross entry/exit |
| `breakout-retest-v1` | `15m` | Swing-high breakout with retest confirmation |

Each run persists signals, trades, and a strategy-run audit row.

## API and dashboard

Start the API:

```bash
make api
```

Open the dashboard at [http://localhost:8080/](http://localhost:8080/).

The API serves JSON endpoints and static files from `web/`. The dashboard is a single `web/index.html` page using Lightweight Charts.

Health checks:

- `GET /health` — process is up
- `GET /ready` — process can reach PostgreSQL

Dashboard features:

- Symbol, timeframe, and strategy filters for the active universe
- `from` and `to` date-range filters (UTC+7 display via `Asia/Bangkok`)
- Candlestick chart with signal and trade markers
- Clickable signals/trades with an inspection panel
- Freshness grid: last candle sync, last strategy run, latest candle time
- Recent strategy run history for the current selection

### HTTP API

Common query parameters for chart endpoints:

- `symbol` — asset code, e.g. `BTC` (required)
- `timeframe` — e.g. `1h` (required)
- `strategy` — e.g. `trend-long-v1` (required for strategy-aware endpoints)
- `from`, `to` — RFC3339 timestamps; default to the last 30 days through now

Examples:

```bash
curl 'http://localhost:8080/chart-data?symbol=BTC&timeframe=1h&strategy=trend-long-v1'
curl 'http://localhost:8080/freshness?symbol=BTC&timeframe=1h&strategy=trend-long-v1'
curl 'http://localhost:8080/strategy-runs?symbol=BTC&timeframe=1h&strategy=trend-long-v1&limit=5'
```

Endpoints:

| Method | Path | Notes |
| --- | --- | --- |
| `GET` | `/health` | Liveness |
| `GET` | `/ready` | DB connectivity |
| `GET` | `/symbols/active` | Active tradable symbols |
| `GET` | `/symbols/all` | Full seeded universe |
| `GET` | `/timeframes` | Active timeframes |
| `GET` | `/strategies` | Active strategies; optional `?slug=` |
| `GET` | `/candles` | OHLCV range (`strategy` not required) |
| `GET` | `/signals` | Strategy signals in range |
| `GET` | `/trades` | Strategy trades in range |
| `GET` | `/strategy-runs` | Recent runs; optional `?limit=` |
| `GET` | `/freshness` | Last sync, last strategy run, latest candle |
| `GET` | `/chart-data` | Candles, signals, trades, recent runs, freshness |
| `GET` | `/` | Dashboard static assets |

## Verification

Build and unit tests (no database required for pure unit packages):

```bash
make build
make unit-test
make check
```

Full test suite including integration and e2e tests requires PostgreSQL on the host port from `.env` (default `5433`). Integration tests create ephemeral databases automatically:

```bash
make postgres-up
make integration-test
```

Notable test coverage:

- Strategy indicators and fixtures (`internal/strategy`)
- Repository, candle sync, and strategy persistence integration tests
- Query-plan checks for MVP indexes (`migrations/000006`)
- API + dashboard chart workflow e2e test

### MVP smoke workflow

```bash
make smoke
```

`make smoke` runs `go run ./cmd/smoke`, which:

1. Starts Docker PostgreSQL
2. Applies migrations and verifies seeded reference data
3. Syncs candles for one asset (default `BTC`) from Binance
4. Recalculates one strategy (default `trend-long-v1`) for that asset
5. Starts the API on a dedicated smoke port and checks `/health`, `/ready`, reference endpoints, `/chart-data`, and the dashboard shell

Optional overrides:

| Variable | Default | Purpose |
| --- | --- | --- |
| `SMOKE_ASSET` | `BTC` | Asset used for sync and strategy checks |
| `SMOKE_STRATEGY` | `trend-long-v1` | Strategy slug used for recalculation and chart checks |
| `SMOKE_PORT` | `18080` | Temporary API port for smoke (avoids clashing with a dev API on `8080`) |

Requires Docker, network access to Binance, and a `.env` file (or defaults from `.env.example`).

### MVP acceptance pass

```bash
make acceptance
```

`make acceptance` runs `go run ./cmd/acceptance` and verifies the frozen top-50 universe end to end:

1. Seeded universe invariants: 50 total symbols, 38 active, 12 inactive unsupported assets
2. Active symbols are queryable from the CLI and API
3. Inactive symbols stay out of `/symbols/active` but remain visible in `/symbols/all`
4. Candle sync and both MVP strategies run for the acceptance asset (default `BTC`)
5. `trend-long-v1` and `breakout-retest-v1` produce inspectable signals through the API

Optional overrides:

| Variable | Default | Purpose |
| --- | --- | --- |
| `ACCEPTANCE_ASSET` | `BTC` | Asset used for sync and strategy acceptance checks |
| `ACCEPTANCE_PORT` | `18081` | Temporary API port for acceptance |

The integration test `internal/acceptance/mvp_integration_test.go` covers the same universe and inspectability checks against an ephemeral test database.

## Troubleshooting

| Symptom | Likely cause | Fix |
| --- | --- | --- |
| `connection refused` on `5433` | PostgreSQL is not running | `make postgres-up` and wait a few seconds |
| API starts but `/ready` fails | Wrong DB settings or migrations not applied | Check `.env`, then `make migrate-up` |
| `make api` fails to bind port | Port `8080` already in use | Stop the other process or change `APP_PORT` |
| Candle sync fails immediately | No network access to Binance | Check connectivity and `BINANCE_BASE_URL` |
| Dashboard loads but chart is empty | No candles or strategy runs for the selected filters | Run scoped sync and `run-strategies`, then pick an active symbol/timeframe/strategy pair |
| `make smoke` passes but dev API does not | Smoke uses port `18080`, not `8080` | Use `make api` for normal development on `8080` |
| `make acceptance` fails on signals | Acceptance asset lacks synced candles or strategy output | Re-run `make acceptance`, or sync/run strategies manually for `ACCEPTANCE_ASSET` |

## Project layout

```
cmd/
  api/          HTTP API and dashboard file server
  worker/       sync-candles, run-strategies, list-active-symbols
  migrate/      apply SQL migrations
  smoke/        local MVP smoke workflow (`make smoke`)
  acceptance/   frozen universe MVP acceptance (`make acceptance`)
internal/
  acceptance/   shared MVP acceptance checks
  app/          API routes and worker wiring
  domain/       market data and strategy models
  infrastructure/binance/
  services/     candle sync and strategy runner
  storage/postgres/
  strategy/     trend-long-v1, breakout-retest-v1
migrations/     schema and seed SQL
web/            dashboard (index.html)
```

## Current scope

Implemented MVP capabilities:

- Go module with API, worker, migrate, and smoke entrypoints
- Local PostgreSQL via Docker Compose on host port `5433`
- SQL migrations for market data, strategy engine, seeds, and query indexes
- Binance Spot candle sync with backfill and incremental updates
- Two independent long-only strategies with persisted signals and trades
- JSON API and chart dashboard with date-range and freshness workflows
- Unit, integration, e2e, and smoke verification for core paths
