## Why

Trading position-limit instruments with Max per lot ≥ $10,000 include several Binance Spot USDT pairs that are not yet in the active ingestion universe. Operators need candles for those pairs without expanding to the full exchange list.

## What Changes

- Keep the July 6, 2026 CoinMarketCap top-50 non-stable snapshot as the base seed.
- Add a position-limits shortlist of seven Binance Spot USDT symbols: `BONK`, `FLOKI`, `TRUMP`, `LDO`, `WIF`, `PUMP`, `VIRTUAL`.
- Activate `HYPE` for ingestion now that Binance Spot USDT mapping is available (`TRADING`).
- Leave unsupported shortlist candidates (`XMR`, `LIT`) and remaining sub-$10k instruments out of scope.
- Update acceptance/smoke/integration universe invariants (after follow-up requested symbols): 64 total symbols, 53 active, 11 inactive.
- Add requested Spot USDT symbols: `1INCH`, `ANKR`, `APT`, `ARB`, `DASH`, `EGLD`, `STRK`.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `market-data-platform`: Extend the configured symbol universe beyond the frozen CMC top-50 with an explicit position-limits shortlist and reactivate `HYPE`.

## Impact

- New SQL migration seeds seven symbols and flips `HYPE` to active.
- Candle sync and strategy workers automatically pick up new actives via `is_active = TRUE`.
- README and acceptance constants must match the new counts.
- First backfill for eight newly active series increases Binance API load; prefer scoped sync after migrate.
