## Context

The online app seeds symbols from a frozen CMC top-50 snapshot and activates only Binance Spot-compatible pairs. A trading position-limits table identifies high-lot instruments; several are missing from the active set even though Spot USDT pairs trade on Binance.

## Goals / Non-Goals

**Goals:**
- Add seven shortlist symbols with `is_active = TRUE` and USDT Binance mappings.
- Activate existing `HYPE` for ingestion.
- Keep universe counts and inactive lists consistent across migrations, acceptance, smoke, and README.

**Non-Goals:**
- Adding all ~100 remaining position-limit instruments.
- Enabling `XMR` or `LIT` without Spot USDT `TRADING`.
- Changing candle sync, rate limiting, or strategy logic.
- Recomputing CMC ranks dynamically.

## Decisions

- **Threshold:** Max per lot ≥ $10,000 and live Binance Spot USDT `TRADING`.
- **Ranks:** Assign `market_cap_rank` 51–57 to new inserts in descending lot order (does not imply CMC rank).
- **Migration style:** Additive `000009` with `ON CONFLICT` upserts for the seven inserts and `UPDATE` for `HYPE`; down deletes the seven and restores `HYPE` inactive.
- **Sync:** No code changes; operators run scoped `sync-candles --asset ...` for new codes after migrate.

## Risks / Trade-offs

- First full-universe sync is heavier (46×3 timeframes). Mitigation: scoped sync scripts / per-asset worker flags.
- `market_cap_rank` for shortlist assets is synthetic. Acceptable for ordering metadata until a dedicated ranking source exists.

## Migration Plan

1. Apply `000009_expand_position_limits_shortlist`.
2. Update acceptance/smoke/integration expected counts.
3. Scoped candle sync for `BONK`, `FLOKI`, `TRUMP`, `HYPE`, `LDO`, `WIF`, `PUMP`, `VIRTUAL`.

## Open Questions

- None for this change.
