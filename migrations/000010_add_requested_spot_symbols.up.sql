-- Additional requested Spot USDT symbols (operator list).
-- market_cap_rank 58-64 is synthetic, not CMC rank.
INSERT INTO symbols (asset_code, name, market_cap_rank, quote_asset, binance_symbol, is_active)
VALUES
    ('1INCH', '1INCH', 58, 'USDT', '1INCHUSDT', TRUE),
    ('ANKR', 'ANKR', 59, 'USDT', 'ANKRUSDT', TRUE),
    ('APT', 'APT', 60, 'USDT', 'APTUSDT', TRUE),
    ('ARB', 'ARB', 61, 'USDT', 'ARBUSDT', TRUE),
    ('DASH', 'DASH', 62, 'USDT', 'DASHUSDT', TRUE),
    ('EGLD', 'EGLD', 63, 'USDT', 'EGLDUSDT', TRUE),
    ('STRK', 'STRK', 64, 'USDT', 'STRKUSDT', TRUE)
ON CONFLICT (asset_code) DO UPDATE
SET name = EXCLUDED.name,
    market_cap_rank = EXCLUDED.market_cap_rank,
    quote_asset = EXCLUDED.quote_asset,
    binance_symbol = EXCLUDED.binance_symbol,
    is_active = EXCLUDED.is_active,
    updated_at = NOW();
