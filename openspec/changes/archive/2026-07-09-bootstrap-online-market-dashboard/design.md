## Context

`fear-and-greed-online` is being created as a separate product application rather than an extension of the research-oriented `fear-and-greed` repository. The research codebase remains useful for exploratory backtests, ad-hoc experimentation, and rapid iteration, but it is not the right system boundary for a stable online application with a database, scheduled jobs, API contracts, and a chart-driven UI.

The new application must own five concerns end to end:

1. Market data ingestion for configured crypto instruments.
2. Durable persistence of normalized OHLCV candle history.
3. Strategy execution using implementations that live entirely inside this new repository.
4. Persistence and retrieval of strategy runs, signals, and trades in forms suitable for both debugging and UI use.
5. A dashboard application that lets users inspect candles and overlays without depending on generated HTML reports.

Constraints and assumptions:

- The initial product is single-tenant and internal-facing.
- The first version prioritizes correctness, auditability, and architectural clarity over maximum optimization.
- Data is refreshed by scheduled jobs rather than by full real-time streaming.
- The online strategy implementations do not depend on the research repository at runtime and are allowed to diverge from research implementations.
- The UI is expected to be interactive enough to benefit from SPA behavior, but the overall product surface should remain intentionally small.
- The initial asset universe is a frozen CoinMarketCap top-50 non-stable snapshot as of July 6, 2026, but only assets that can be mapped to Binance Spot symbols are eligible for ingestion.
- The supported MVP timeframes are `15m`, `1h`, and `4h`.
- Initial historical backfill targets one month of data per active symbol and timeframe.
- Time is stored canonically in UTC, while user-facing rendering defaults to UTC+7.
- The system may ingest and expose the most recent still-open candle for the active timeframe.

## Goals / Non-Goals

**Goals:**

- Establish a clean standalone repository architecture with `api`, `worker`, `storage`, `strategy`, and `web` concerns separated.
- Use PostgreSQL as the source of truth for symbols, timeframes, candles, strategies, runs, signals, trades, and sync metadata.
- Support incremental candle synchronization for configured symbols and timeframes.
- Support independent in-repo strategy implementations with explicit versioning and normalized outputs.
- Persist a complete history of strategy runs so the system can explain when and how signals were produced.
- Expose chart-ready API responses for candles, signals, trades, strategy metadata, and recent run history.
- Provide a lightweight SPA that supports selecting symbol, timeframe, strategy, and date range and then visualizing candles and markers.
- Limit the MVP strategy set to a small number of deliberately chosen strategies so the platform can stabilize before expansion.

**Non-Goals:**

- Achieving full feature parity with every strategy or report already present in the research repository.
- Sharing runtime code or packages with the research repository.
- Supporting multi-user access control, authentication, or tenant isolation in the initial release.
- Building a real-time streaming platform with websockets, push alerts, or sub-second updates.
- Building a visual strategy editor, portfolio analytics suite, or execution bot.
- Optimizing for arbitrary exchanges or data providers in the first implementation beyond establishing interfaces that can grow later.

## Decisions

### 1. Keep strategy implementations fully independent inside `fear-and-greed-online`

The online application will own its own strategy implementations and will not import strategy code from the research repository.

Rationale:
- Preserves a clean product boundary.
- Avoids tight coupling to exploratory or unstable internal models.
- Lets the online app shape strategy outputs for persistence and UI use instead of inheriting CLI-oriented results.
- Reduces the risk that changes in the research code accidentally destabilize the online product.

Alternatives considered:
- Reuse research code directly: rejected because it couples product stability to a research codebase and would force the new app to absorb non-product assumptions.
- Extract a shared engine first: rejected for MVP because it introduces an additional refactoring project before the product shape is proven.

### 2. Use PostgreSQL as the authoritative persistence layer

The application will persist all durable operational and product data in PostgreSQL.

Rationale:
- Supports relational modeling for symbols, timeframes, runs, signals, and trades.
- Supports indexed range queries by symbol, timeframe, strategy, and time windows.
- Supports JSONB for strategy-specific metadata without forcing schema churn for every new strategy nuance.
- Fits well with Go service development and future migrations, backfills, and auditing needs.

Alternatives considered:
- Flat files or generated JSON artifacts: rejected because they do not provide strong querying, scheduling history, or durable state coordination.
- Time-series-only specialized storage: rejected for MVP because the product also needs strategy, run, and UI-oriented relational data.

### 3. Split runtime responsibilities into `api` and `worker`

The application will expose at least two runnable processes:
- `api`: serves HTTP endpoints and optionally the web assets.
- `worker`: performs scheduled sync and strategy execution jobs.

Rationale:
- Prevents long-running jobs from interfering with request serving.
- Makes scheduling behavior explicit and testable.
- Creates a natural boundary for future horizontal scaling or deployment separation.

Alternatives considered:
- Single binary doing both jobs and HTTP: acceptable for very small prototypes, but rejected because background job concerns would bleed into request serving too early.

### 4. Model strategy outputs as normalized signals and trades with strategy-specific metadata in JSONB

Each strategy will return structured outputs that can be mapped into:
- `strategy_runs`
- `signals`
- `trades`

Signal and trade records will use normalized columns for frequently queried fields and JSONB for strategy-specific annotations.

Rationale:
- Gives the UI and API a stable cross-strategy contract.
- Preserves flexibility for strategy-specific values such as ATR, SMA, stop loss, retest zone, or risk-reward context.
- Avoids schema explosion when adding new strategies.

Alternatives considered:
- Store only raw strategy output blobs: rejected because the API and chart would need custom parsing per strategy and querying would become awkward.
- Fully normalize all strategy fields into columns: rejected because it overfits early and makes adding strategies expensive.

### 5. Version strategy definitions explicitly

Strategies will be represented by durable definitions that include `code`, `version`, and a stable external slug such as `trend-long-v1`.

Rationale:
- Prevents ambiguity when strategy behavior changes.
- Allows old runs and signals to remain attributable to the exact logic version that produced them.
- Gives the UI and API a durable identifier to filter on.

Alternatives considered:
- Only store a strategy name: rejected because it loses historical meaning once logic changes.

### 6. Start with a small strategy set and a chart-first product surface

The MVP will implement a limited number of strategies such as `trend-long-v1` and `breakout-retest-v1`, and the UI will focus on chart inspection rather than broader portfolio or automation workflows.

Rationale:
- Validates the storage model, run model, and chart UI with real strategy diversity but without overwhelming the first implementation.
- Ensures that the platform is shaped around a clear user job: inspect market history together with generated strategy signals.

Alternatives considered:
- Port many strategies immediately: rejected because it would delay validation of the platform shape and increase schema and UI complexity too early.

### 7. Use a lightweight SPA for the dashboard

The UI will be a small SPA, likely React-based, focused on one primary chart workflow rather than a large multi-surface web product.

Rationale:
- Symbol, timeframe, strategy, and date filters create enough client-side state to justify SPA behavior.
- Chart interactions and detail panels benefit from local state and smooth in-page updates.
- Keeps the front end product-oriented while remaining small enough to avoid unnecessary complexity.

Alternatives considered:
- Static HTML reports: rejected because the application now requires database-backed queries, dynamic filtering, and a reusable app shell.
- Heavy enterprise-style frontend architecture: rejected because the product surface is still intentionally narrow.

### 8. Run PostgreSQL in Docker for local development

Local development will use PostgreSQL as a Docker-managed service, started through `docker compose`, while `api` and `worker` may initially run directly on the host machine for faster iteration.

Rationale:
- Keeps the database isolated from the host system and easy to reset.
- Pins the PostgreSQL version for repeatable development and onboarding.
- Makes it straightforward to share the same database bootstrap flow across contributors and environments.
- Preserves a fast developer loop for Go services by avoiding full container rebuilds on every code change.

Alternatives considered:
- Install PostgreSQL directly on the host: rejected because it is harder to standardize and clean up across machines.
- Run the entire stack in Docker from day one: deferred because it is useful for parity and deployment, but slower for day-to-day Go iteration during MVP development.

### 9. Freeze the MVP universe from a dated CoinMarketCap snapshot and ingest only Binance Spot intersections

The MVP will start from the CoinMarketCap top-50 non-stable ranking snapshot current on July 6, 2026, but the effective tracked universe will be the subset of that snapshot that can be resolved to Binance Spot symbols.

Rationale:
- Preserves the user's intent to anchor the universe to a market-cap ranking rather than a handpicked watchlist.
- Keeps the universe stable for MVP comparisons instead of allowing it to drift as market caps change day to day.
- Avoids promising ingestion for assets that are absent from the chosen market-data source.

Alternatives considered:
- Recompute the current top-50 automatically on every run: rejected because the tracked universe would drift and make results harder to compare over time.
- Ignore Binance Spot availability and keep all top-50 names active: rejected because ingestion would fail for unsupported assets.
- Manually hand-pick only Binance-listed majors: rejected because it departs from the requested market-cap-based selection.

### 10. Trigger data sync and strategy recalculation from CLI commands in MVP

The MVP will expose explicit CLI-driven workflows for data synchronization and strategy execution instead of background cron scheduling or admin-triggered HTTP endpoints.

Rationale:
- Keeps operational control simple during early development.
- Reduces accidental background writes while the schema and strategy behavior are still stabilizing.
- Fits the current internal workflow where operators can intentionally refresh data and recalculate signals.

Alternatives considered:
- Always-on cron worker from day one: deferred because scheduling can be added later without changing the core domain model.
- Admin endpoints for manual trigger: rejected for MVP because they add surface area without being necessary yet.

## Risks / Trade-offs

- **[Strategy divergence from research]** Online strategy results may not match research backtests exactly.  
  Mitigation: treat online strategies as product-owned implementations, version them explicitly, and build regression fixtures for deterministic validation.

- **[Schema underfitting or overfitting]** The first schema could either miss important strategy concepts or become too abstract too early.  
  Mitigation: keep core query fields normalized and strategy-specific detail in JSONB, then evolve based on real usage.

- **[Data sync correctness]** Incremental candle synchronization can create gaps, duplicates, or partial windows if not handled carefully.  
  Mitigation: enforce unique keys on `(symbol_id, timeframe_id, open_time)`, track sync jobs, and use idempotent upserts.

- **[Operational complexity]** Introducing API, worker, PostgreSQL, migrations, and web assets increases setup complexity versus a simple CLI.  
  Mitigation: standardize local development with Docker Compose, seed data, and a small number of moving parts in MVP.

- **[Docker and host split]** Running PostgreSQL in Docker while `api` and `worker` run on the host creates two execution modes to understand.  
  Mitigation: document one recommended default workflow clearly and keep connection settings simple and environment-driven.

- **[Future strategy expansion]** Strategies with overlays such as zones, ranges, or FVG structures may outgrow a simple marker model.  
  Mitigation: reserve JSONB metadata and extend later with overlay-specific persistence only when a real requirement emerges.

- **[Performance at scale]** Querying large candle windows and repeated strategy runs can become expensive as symbol count grows.  
  Mitigation: start with bounded symbol sets, add indexes early, and only optimize execution windows once actual bottlenecks are measured.

- **[Universe/source mismatch]** Some assets in the CoinMarketCap snapshot may not exist on Binance Spot or may use different symbols.  
  Mitigation: create an explicit symbol seed and mapping layer, and persist only the snapshot assets that can be resolved to supported Binance Spot pairs.

## Migration Plan

This is a greenfield application, so migration primarily means staged bootstrap rather than production cutover.

1. Initialize the repository structure, local runtime, and migration tooling.
2. Add a Docker Compose setup that starts PostgreSQL with stable local defaults and persistent storage.
3. Create the first PostgreSQL schema for symbols, timeframes, candles, strategies, runs, signals, trades, and sync jobs.
4. Seed foundational reference data such as the frozen MVP symbol snapshot, supported timeframes, and initial strategy definitions.
5. Implement ingestion and verify one-month candle persistence for `15m`, `1h`, and `4h`.
6. Implement the first independent strategies, `trend-long-v1` and `breakout-retest-v1`, and persist runs and signals.
7. Expose read APIs and wire the dashboard to them.
8. Validate the end-to-end workflow locally against the Docker-managed database using the seeded Binance Spot-compatible universe.

Rollback approach:
- Schema changes are tracked through migrations and can be rolled back migration by migration during development.
- Strategy rollbacks are handled by deactivating a strategy definition version and keeping previously generated runs and signals intact for auditability.

## Open Questions

- Which assets from the July 6, 2026 CoinMarketCap top-50 non-stable snapshot are unavailable on Binance Spot and therefore excluded from the active seed list?
- Should the initial API serve the SPA directly, or should the web app be built and deployed as a separate artifact from the start?
- Should deduplication be enforced at the database level for signals in MVP, or handled first at the application layer with a stable business key?

## MVP Universe Snapshot

The MVP asset snapshot is frozen from the CoinMarketCap ranking viewed on July 6, 2026, after excluding dollar-pegged stablecoins from the top-ranking list. The selected 50 assets are:

1. BTC
2. ETH
3. BNB
4. XRP
5. SOL
6. TRX
7. HYPE
8. DOGE
9. LEO
10. ZEC
11. XLM
12. ADA
13. XMR
14. LINK
15. CC
16. GRAM
17. BCH
18. LTC
19. HBAR
20. SUI
21. AVAX
22. CRO
23. NEAR
24. SHIB
25. XAUt
26. DEXE
27. TAO
28. UNI
29. PAXG
30. WLFI
31. ASTER
32. OKB
33. ONDO
34. M
35. DOT
36. WLD
37. AAVE
38. MNT
39. SKY
40. PI
41. ICP
42. BGB
43. ETC
44. PEPE
45. MORPHO
46. KCS
47. KAS
48. RENDER
49. ATOM
50. QNT

Implementation note:
- This is the product universe snapshot.
- Exchange ingestion still depends on Binance Spot symbol support and symbol mapping.
- Assets from this snapshot that cannot be mapped to Binance Spot are seeded but remain inactive for ingestion until a valid mapping is defined.
