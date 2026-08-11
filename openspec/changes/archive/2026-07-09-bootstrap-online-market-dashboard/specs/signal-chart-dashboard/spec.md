## ADDED Requirements

### Requirement: System provides chart-ready candle and signal APIs
The system SHALL expose HTTP APIs that allow the dashboard to retrieve candles, signals, trades, and run context for a selected symbol, timeframe, strategy, and date range.

#### Scenario: Request candles for chart rendering
- **WHEN** the dashboard requests candles for a symbol, timeframe, and date range
- **THEN** the system returns candle data in a format directly usable by the chart layer

#### Scenario: Request chart overlays for a strategy
- **WHEN** the dashboard requests signals and trades for a selected strategy and date range
- **THEN** the system returns normalized marker-ready data and related metadata for rendering overlays

### Requirement: Dashboard supports core chart filtering workflow
The system SHALL provide a dashboard workflow that lets the user choose a symbol, timeframe, strategy, and date range before viewing chart data.

#### Scenario: Change chart filters
- **WHEN** the user changes the selected symbol, timeframe, strategy, or date range
- **THEN** the dashboard refreshes its chart data to match the current filter state

#### Scenario: Load default chart state
- **WHEN** the dashboard opens for the first time
- **THEN** the system loads a valid default combination of active symbol, supported timeframe, and available strategy

#### Scenario: Apply bounded date range
- **WHEN** the user provides both `from` and `to` boundaries for the chart
- **THEN** the dashboard requests and renders only the candles, signals, and trades inside the selected bounded range

### Requirement: Dashboard renders candles together with strategy markers
The system SHALL render candlestick market data and visualize strategy outputs as markers or overlays on the same chart surface.

#### Scenario: Render entry and exit markers
- **WHEN** the dashboard has candles and corresponding signals or trades
- **THEN** the chart displays the appropriate marker positions aligned to the underlying candle times

#### Scenario: Render without markers
- **WHEN** the selected chart window contains candles but no matching signals or trades
- **THEN** the dashboard still renders the candles and clearly shows that no overlays exist in that range

### Requirement: Dashboard shows inspectable signal details
The system SHALL let the user inspect details for a selected signal or trade, including strategy-specific metadata returned by the API.

#### Scenario: Inspect a selected signal
- **WHEN** the user selects a signal marker or related list item
- **THEN** the dashboard displays the signal’s core fields and strategy-specific metadata in a detail panel

#### Scenario: Inspect recent trade context
- **WHEN** the user selects a trade or trade-linked signal
- **THEN** the dashboard displays associated trade lifecycle data such as entry, exit, status, and profit where available

### Requirement: Dashboard surfaces recent strategy run history and data freshness
The system SHALL expose and display enough operational context for the user to understand whether data and signals are current.

#### Scenario: Display recent runs
- **WHEN** the dashboard requests recent strategy run information for the active filters
- **THEN** the system returns recent run status, timestamps, and summary counts for display

#### Scenario: Display freshness information
- **WHEN** the dashboard requests operational context for the active chart
- **THEN** the system can show the last successful data sync or strategy run time relevant to the selected symbol and timeframe

#### Scenario: Display latest available market point
- **WHEN** the dashboard renders freshness information for the active symbol and timeframe
- **THEN** the system also provides the latest available candle timestamp so the user can distinguish delayed recalculation from missing market data

### Requirement: Dashboard renders timestamps in UTC+7 by default
The system SHALL present chart and detail timestamps in UTC+7 by default while preserving canonical UTC storage in backend data models.

#### Scenario: Display chart times in the user-facing timezone
- **WHEN** the dashboard renders candle and signal timestamps
- **THEN** the displayed times use UTC+7 semantics for user-facing presentation

#### Scenario: Preserve canonical backend timestamps
- **WHEN** the dashboard receives candle or signal timestamps from the API
- **THEN** the underlying API payloads remain unambiguous and suitable for canonical UTC interpretation

### Requirement: System supports MVP acceptance verification for the dashboard workflow
The system SHALL provide a locally repeatable way to verify that the seeded dashboard workflow is working end to end for the frozen MVP symbol universe.

#### Scenario: Run local smoke verification
- **WHEN** a developer runs the documented MVP smoke workflow
- **THEN** the workflow verifies that reference filters load, chart data can be requested for an active symbol, and at least one strategy result can be inspected through the API or dashboard
