#!/usr/bin/env bash
# Sync candles for every active symbol (the 38 Binance Spot pairs).
# Continues on per-asset errors so one failed pair does not abort the rest.

set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "==> listing active symbols"
list_out="$(go run ./cmd/worker list-active-symbols)"
echo "$list_out"

failed=0
count=0
failed_assets=()

while read -r rank asset _rest; do
  if [[ "$rank" == "RANK" || -z "${asset:-}" ]]; then
    continue
  fi
  count=$((count + 1))
  echo "==> sync ${asset} (${count})"
  if ! go run ./cmd/worker sync-candles --asset "$asset"; then
    echo "WARN: sync failed for ${asset}" >&2
    failed=$((failed + 1))
    failed_assets+=("$asset")
  fi
done <<< "$list_out"

echo "==> done: ${count} symbols, ${failed} failed"
if [[ "$failed" -gt 0 ]]; then
  echo "failed assets: ${failed_assets[*]}" >&2
  exit 1
fi
if [[ "$count" -eq 0 ]]; then
  echo "no active symbols found" >&2
  exit 1
fi
