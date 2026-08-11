## Why

The current research repository is optimized for CLI-driven backtests and exploratory work, but it does not provide a durable product boundary for storing market data, running scheduled strategy evaluations, and serving a user-facing chart application. We need a separate online system now so that market data, strategy logic, API contracts, and UI behavior can evolve as a stable product without inheriting the coupling and volatility of the research codebase.

## What Changes

- Create a new standalone online application architecture that owns its own market data ingestion, strategy execution, persistence, API, and web dashboard.
- Introduce PostgreSQL-backed storage for symbols, timeframes, OHLCV candles, strategy definitions, strategy runs, signals, trades, and synchronization history.
- Add a background ingestion capability that incrementally pulls candles for configured symbols and timeframes on a schedule and records sync progress.
- Add an independent strategy engine that re-implements supported strategies inside this repository instead of importing or depending on the research project.
- Add normalized persistence for strategy runs, signals, and trades so the system can track what was calculated, when, and with which parameters.
- Add chart-oriented API capabilities for retrieving candles, signals, trades, strategy metadata, and recent run history for the web application.
- Add a lightweight SPA dashboard focused on symbol selection, timeframe selection, strategy selection, date-range filtering, candlestick chart rendering, and signal inspection.
- Freeze the MVP asset universe from the CoinMarketCap top-50 non-stable ranking snapshot current on July 6, 2026, then map that snapshot to Binance Spot-supported symbols for actual ingestion.

## Capabilities

### New Capabilities
- `market-data-platform`: Scheduled ingestion, storage, and retrieval of normalized market candle data for configured instruments and timeframes.
- `independent-strategy-engine`: Standalone in-repo strategy implementations, strategy run tracking, and persistence of normalized signals and trades.
- `signal-chart-dashboard`: HTTP API and web dashboard for viewing candles, signals, trades, and strategy run history on an interactive chart.

### Modified Capabilities
- None.

## Impact

- Introduces a new product architecture in [fear-and-greed-online](</Users/d.rybin/projects/fear-and-greed-online>) with clear boundaries between `api`, `worker`, `storage`, `strategy`, and `web`.
- Introduces PostgreSQL as the source of truth for market data and strategy results.
- Requires migration tooling, seed data, scheduling, and data access conventions suitable for production-style application development.
- Establishes stable API contracts and UI-facing data shapes that are independent from the legacy CLI output and report formats.
