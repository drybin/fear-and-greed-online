# market-data-platform Specification

## Purpose
TBD - created by archiving change bootstrap-online-market-dashboard. Update Purpose after archive.
## Requirements
### Requirement: System stores normalized candle history for configured symbols and timeframes
The system SHALL persist OHLCV candle history in PostgreSQL for each active symbol and supported timeframe and SHALL treat the database as the authoritative source of market history for the online application.

#### Scenario: Persist candles for an active symbol and timeframe
- **WHEN** the ingestion workflow loads candles for an active symbol and supported timeframe
- **THEN** the system stores each candle with normalized symbol, timeframe, time, price, and volume fields in PostgreSQL

#### Scenario: Reject duplicate candle identity
- **WHEN** ingestion attempts to write a candle whose symbol, timeframe, and open time already exist
- **THEN** the system preserves candle uniqueness and updates or ignores the duplicate record without creating a second candle row

#### Scenario: Persist the latest still-open candle
- **WHEN** the market-data source returns the most recent candle that has not fully closed yet
- **THEN** the system may store and expose that still-open candle as part of the active series

### Requirement: System supports incremental candle synchronization
The system SHALL synchronize candles incrementally for each active symbol and timeframe so routine refresh jobs do not require full historical reloads.

#### Scenario: Start from the latest stored candle
- **WHEN** a scheduled synchronization starts for a symbol and timeframe that already has historical candles
- **THEN** the system determines the latest stored candle boundary and requests only the missing window after that boundary

#### Scenario: Perform initial backfill for an empty series
- **WHEN** a scheduled synchronization starts for a symbol and timeframe with no stored candles
- **THEN** the system performs an initial historical backfill using the configured bootstrap range

### Requirement: System tracks ingestion job outcomes
The system SHALL record synchronization attempts, status, loaded ranges, and failures so operators and the UI can inspect data freshness and diagnose loading issues.

#### Scenario: Successful synchronization is recorded
- **WHEN** a candle synchronization job completes successfully
- **THEN** the system stores the job status, requested range, loaded range, and number of candles loaded

#### Scenario: Failed synchronization is recorded
- **WHEN** a candle synchronization job fails
- **THEN** the system stores the failed status together with the error context without deleting previously stored candles

#### Scenario: Expose latest successful sync for freshness checks
- **WHEN** the application needs to show whether chart data is current for an active symbol and timeframe
- **THEN** the system can determine the latest successful synchronization time and the latest stored candle boundary for that context

### Requirement: System exposes reference market data metadata
The system SHALL maintain configurable reference data for symbols and timeframes used by ingestion, strategy execution, and API queries.

#### Scenario: List active symbols
- **WHEN** the application queries active symbols for ingestion or UI use
- **THEN** the system returns only configured active symbols with their metadata

#### Scenario: Inspect active symbol universe from operations tooling
- **WHEN** an operator or developer runs the documented symbol inspection API or CLI for the independent online app
- **THEN** the system returns the currently active seeded symbols from PostgreSQL together with their provider mapping metadata

#### Scenario: List supported timeframes
- **WHEN** the application queries supported timeframes
- **THEN** the system returns the configured timeframe codes and duration metadata used by candles and strategies

### Requirement: System seeds the MVP asset universe from a dated CoinMarketCap snapshot
The system SHALL seed the MVP asset universe from the CoinMarketCap top-50 non-stable market-cap ranking snapshot current on July 6, 2026, and SHALL activate only symbols that can be mapped to Binance Spot pairs.

The frozen MVP asset snapshot SHALL contain these 50 assets:
`BTC`, `ETH`, `BNB`, `XRP`, `SOL`, `TRX`, `HYPE`, `DOGE`, `LEO`, `ZEC`, `XLM`, `ADA`, `XMR`, `LINK`, `CC`, `GRAM`, `BCH`, `LTC`, `HBAR`, `SUI`, `AVAX`, `CRO`, `NEAR`, `SHIB`, `XAUt`, `DEXE`, `TAO`, `UNI`, `PAXG`, `WLFI`, `ASTER`, `OKB`, `ONDO`, `M`, `DOT`, `WLD`, `AAVE`, `MNT`, `SKY`, `PI`, `ICP`, `BGB`, `ETC`, `PEPE`, `MORPHO`, `KCS`, `KAS`, `RENDER`, `ATOM`, `QNT`.

#### Scenario: Load the frozen MVP symbol universe
- **WHEN** the database is seeded for the first time
- **THEN** the system creates symbol records based on the July 6, 2026 CoinMarketCap top-50 non-stable snapshot and marks Binance Spot-compatible pairs as active

#### Scenario: Exclude unsupported snapshot assets
- **WHEN** an asset appears in the frozen CoinMarketCap snapshot but has no supported Binance Spot mapping
- **THEN** the system does not activate that asset for ingestion until a valid mapping exists

### Requirement: System limits MVP ingestion to defined spot timeframes and one-month backfill
The system SHALL ingest only Binance Spot candle data for the `15m`, `1h`, and `4h` timeframes in MVP and SHALL bootstrap each active series with approximately one month of history.

#### Scenario: Ingest only supported MVP timeframes
- **WHEN** a sync workflow expands work for an active symbol
- **THEN** it schedules ingestion only for `15m`, `1h`, and `4h` in MVP

#### Scenario: Bootstrap one month of history
- **WHEN** an active symbol and timeframe has no stored candles yet
- **THEN** the initial backfill targets approximately one month of historical spot candles

