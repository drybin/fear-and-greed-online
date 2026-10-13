## 1. Spec and seed migration

- [x] 1.1 Draft OpenSpec change `expand-position-limits-shortlist` (proposal, design, delta spec)
- [x] 1.2 Add migration `000009_expand_position_limits_shortlist` inserting shortlist symbols and activating `HYPE`
- [x] 1.3 Add matching down migration (delete shortlist rows; restore `HYPE` inactive)

## 2. Invariants and docs

- [x] 2.1 Update acceptance universe constants and inactive list (remove `HYPE`)
- [x] 2.2 Update smoke and migrations integration expected counts (57 / 46)
- [x] 2.3 Update README universe numbers and inactive asset list
- [x] 2.4 Sync main `openspec/specs/market-data-platform/spec.md` with the delta

## 3. Verification

- [x] 3.1 Apply migrations and confirm active symbol list includes shortlist + `HYPE`
- [x] 3.2 Optionally scoped `sync-candles` for the eight newly active assets

## 4. Requested Spot symbols follow-up

- [x] 4.1 Add migration `000010_add_requested_spot_symbols` for `1INCH`, `ANKR`, `APT`, `ARB`, `DASH`, `EGLD`, `STRK`
- [x] 4.2 Update universe invariants and docs to 64 total / 53 active
