## MODIFIED Requirements

### Requirement: System seeds the MVP asset universe from a dated CoinMarketCap snapshot
The system SHALL seed the base MVP asset universe from the CoinMarketCap top-50 non-stable market-cap ranking snapshot current on July 6, 2026, and SHALL activate only symbols that can be mapped to Binance Spot pairs.

The frozen MVP asset snapshot SHALL contain these 50 assets:
`BTC`, `ETH`, `BNB`, `XRP`, `SOL`, `TRX`, `HYPE`, `DOGE`, `LEO`, `ZEC`, `XLM`, `ADA`, `XMR`, `LINK`, `CC`, `GRAM`, `BCH`, `LTC`, `HBAR`, `SUI`, `AVAX`, `CRO`, `NEAR`, `SHIB`, `XAUt`, `DEXE`, `TAO`, `UNI`, `PAXG`, `WLFI`, `ASTER`, `OKB`, `ONDO`, `M`, `DOT`, `WLD`, `AAVE`, `MNT`, `SKY`, `PI`, `ICP`, `BGB`, `ETC`, `PEPE`, `MORPHO`, `KCS`, `KAS`, `RENDER`, `ATOM`, `QNT`.

#### Scenario: Load the frozen MVP symbol universe
- **WHEN** the database is seeded for the first time
- **THEN** the system creates symbol records based on the July 6, 2026 CoinMarketCap top-50 non-stable snapshot and marks Binance Spot-compatible pairs as active

#### Scenario: Exclude unsupported snapshot assets
- **WHEN** an asset appears in the frozen CoinMarketCap snapshot but has no supported Binance Spot mapping
- **THEN** the system does not activate that asset for ingestion until a valid mapping exists

#### Scenario: Activate HYPE when Binance Spot mapping is available
- **WHEN** `HYPE` has a supported Binance Spot USDT mapping in the trading state
- **THEN** the system marks `HYPE` as active for candle ingestion

## ADDED Requirements

### Requirement: System extends the active universe with a position-limits shortlist
The system SHALL extend the configured symbol universe with a fixed position-limits shortlist of Binance Spot USDT instruments whose Max per lot is at least $10,000, without replacing the frozen CMC top-50 base seed.

The position-limits shortlist SHALL contain these assets as active symbols:
`BONK`, `FLOKI`, `TRUMP`, `LDO`, `WIF`, `PUMP`, `VIRTUAL`.

The system SHALL further extend the active universe with these requested Binance Spot USDT symbols:
`1INCH`, `ANKR`, `APT`, `ARB`, `DASH`, `EGLD`, `STRK`.

After applying the shortlist extension, the requested-symbol extension, and the CMC base seed, the system SHALL expose 64 total symbol records and 53 active symbols for ingestion (11 inactive unsupported CMC snapshot assets remain inactive).

#### Scenario: Seed position-limits shortlist symbols
- **WHEN** the position-limits shortlist migration is applied
- **THEN** the system creates active symbol records for `BONK`, `FLOKI`, `TRUMP`, `LDO`, `WIF`, `PUMP`, and `VIRTUAL` with USDT Binance Spot mappings

#### Scenario: Seed requested Spot symbols
- **WHEN** the requested Spot symbols migration is applied
- **THEN** the system creates active symbol records for `1INCH`, `ANKR`, `APT`, `ARB`, `DASH`, `EGLD`, and `STRK` with USDT Binance Spot mappings

#### Scenario: Keep sub-threshold and unsupported instruments out of the shortlist
- **WHEN** an instrument from the broader position-limits table has Max per lot below $10,000 or lacks Binance Spot USDT trading
- **THEN** the system does not add that instrument via the shortlist migration
