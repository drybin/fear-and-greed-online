# independent-strategy-engine Specification

## Purpose
TBD - created by archiving change bootstrap-online-market-dashboard. Update Purpose after archive.
## Requirements
### Requirement: System implements strategies inside the online repository
The system SHALL implement supported trading strategies inside `fear-and-greed-online` and SHALL NOT require runtime code imports from the research repository to calculate online signals.

#### Scenario: Execute an in-repo strategy
- **WHEN** the strategy runner requests execution of a supported strategy
- **THEN** the system uses a strategy implementation defined inside the online repository to calculate outputs

#### Scenario: Run without research repository dependency
- **WHEN** the online application starts or executes strategies
- **THEN** the system can complete strategy execution without loading code or packages from the research repository

### Requirement: System supports the MVP strategy set
The system SHALL include `trend-long-v1` and `breakout-retest-v1` as the initial supported strategy set for MVP.

#### Scenario: Execute trend-long-v1
- **WHEN** the operator triggers a run for `trend-long-v1`
- **THEN** the system can execute that strategy for supported symbols and timeframes and persist its run outputs

#### Scenario: Execute breakout-retest-v1
- **WHEN** the operator triggers a run for `breakout-retest-v1`
- **THEN** the system can execute that strategy for supported symbols and timeframes and persist its run outputs

### Requirement: System versions strategy definitions explicitly
The system SHALL persist strategy definitions with explicit code and version information so each run and signal can be traced to the exact strategy implementation version that produced it.

#### Scenario: Persist versioned strategy metadata
- **WHEN** a strategy definition is registered in the application
- **THEN** the system stores a stable external identifier, strategy code, version, and default parameters

#### Scenario: Attribute a run to a strategy version
- **WHEN** a strategy run completes
- **THEN** the system persists the exact strategy definition reference used for that run

### Requirement: System records strategy runs as first-class execution history
The system SHALL persist every strategy execution as a strategy run with parameters, execution window, status, and summary counts.

#### Scenario: Persist a successful strategy run
- **WHEN** the system completes a strategy execution successfully
- **THEN** it stores a strategy run with symbol, timeframe, parameters, input range, status, and output counts for signals and trades

#### Scenario: Persist a failed strategy run
- **WHEN** the system cannot complete a strategy execution
- **THEN** it stores a failed strategy run with error context and without pretending the run produced valid outputs

### Requirement: System persists normalized signals for chart and API use
The system SHALL persist strategy-generated signals in a normalized structure that supports cross-strategy querying and chart rendering while preserving strategy-specific details as metadata.

#### Scenario: Persist an entry signal
- **WHEN** a strategy emits an entry signal
- **THEN** the system stores the strategy, symbol, timeframe, signal time, side, price, type, and strategy-specific metadata for that signal

#### Scenario: Query signals by chart filters
- **WHEN** the application requests signals for a symbol, timeframe, strategy, and date range
- **THEN** the system can return matching signals without requiring strategy-specific parsing logic in the caller

### Requirement: System persists strategy trades separately from signals
The system SHALL persist trade outcomes separately from signals whenever a strategy produces entry and exit lifecycle information.

#### Scenario: Persist a closed trade
- **WHEN** a strategy run produces a completed trade
- **THEN** the system stores entry, exit, side, price, and profit fields linked back to the strategy run

#### Scenario: Persist an open trade
- **WHEN** a strategy run leaves a trade open at the end of its evaluation window
- **THEN** the system stores the trade as open without requiring a synthetic exit

### Requirement: System supports deterministic signal deduplication
The system SHALL support a stable deduplication identity for deterministic signals so repeated scheduled runs do not create duplicate chart markers for the same logical signal.

#### Scenario: Repeated run sees an existing deterministic signal
- **WHEN** a later strategy run emits a signal with the same deduplication identity as a previously stored signal
- **THEN** the system updates or reuses the existing logical signal record instead of inserting a duplicate marker

#### Scenario: Distinct signals remain distinct
- **WHEN** two signals differ by strategy version, symbol, timeframe, side, type, or signal time
- **THEN** the system treats them as separate signals

### Requirement: System exposes CLI-driven execution workflows in MVP
The system SHALL support operator-triggered CLI workflows for candle synchronization and strategy recalculation in MVP.

#### Scenario: Trigger market data sync from the CLI
- **WHEN** the operator runs the market-data synchronization CLI command
- **THEN** the system executes ingestion for the configured active symbol universe and supported timeframes

#### Scenario: Trigger strategy recalculation from the CLI
- **WHEN** the operator runs the strategy recalculation CLI command
- **THEN** the system executes the selected strategy or strategy set and persists new runs, signals, and trades

