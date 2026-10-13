-- Position-limits shortlist (Max per lot >= $10k) with Binance Spot USDT TRADING.
-- market_cap_rank 51-57 is synthetic ordering by descending lot size, not CMC rank.
INSERT INTO symbols (asset_code, name, market_cap_rank, quote_asset, binance_symbol, is_active)
VALUES
    ('BONK', 'BONK', 51, 'USDT', 'BONKUSDT', TRUE),
    ('FLOKI', 'FLOKI', 52, 'USDT', 'FLOKIUSDT', TRUE),
    ('TRUMP', 'TRUMP', 53, 'USDT', 'TRUMPUSDT', TRUE),
    ('LDO', 'LDO', 54, 'USDT', 'LDOUSDT', TRUE),
    ('WIF', 'WIF', 55, 'USDT', 'WIFUSDT', TRUE),
    ('PUMP', 'PUMP', 56, 'USDT', 'PUMPUSDT', TRUE),
    ('VIRTUAL', 'VIRTUAL', 57, 'USDT', 'VIRTUALUSDT', TRUE)
ON CONFLICT (asset_code) DO UPDATE
SET name = EXCLUDED.name,
    market_cap_rank = EXCLUDED.market_cap_rank,
    quote_asset = EXCLUDED.quote_asset,
    binance_symbol = EXCLUDED.binance_symbol,
    is_active = EXCLUDED.is_active,
    updated_at = NOW();

-- HYPE is in the CMC top-50 seed; Spot USDT is now tradable.
UPDATE symbols
SET is_active = TRUE,
    updated_at = NOW()
WHERE asset_code = 'HYPE';
