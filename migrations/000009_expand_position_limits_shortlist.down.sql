-- Remove shortlist symbols and dependent rows so FK constraints allow DELETE.
WITH shortlist AS (
    SELECT id
    FROM symbols
    WHERE asset_code IN ('BONK', 'FLOKI', 'TRUMP', 'LDO', 'WIF', 'PUMP', 'VIRTUAL')
),
shortlist_signals AS (
    SELECT id
    FROM signals
    WHERE symbol_id IN (SELECT id FROM shortlist)
)
UPDATE signal_notifications
SET signal_id = NULL
WHERE signal_id IN (SELECT id FROM shortlist_signals);

DELETE FROM trades
WHERE symbol_id IN (
    SELECT id FROM symbols
    WHERE asset_code IN ('BONK', 'FLOKI', 'TRUMP', 'LDO', 'WIF', 'PUMP', 'VIRTUAL')
);

DELETE FROM signals
WHERE symbol_id IN (
    SELECT id FROM symbols
    WHERE asset_code IN ('BONK', 'FLOKI', 'TRUMP', 'LDO', 'WIF', 'PUMP', 'VIRTUAL')
);

DELETE FROM strategy_runs
WHERE symbol_id IN (
    SELECT id FROM symbols
    WHERE asset_code IN ('BONK', 'FLOKI', 'TRUMP', 'LDO', 'WIF', 'PUMP', 'VIRTUAL')
);

DELETE FROM candles
WHERE symbol_id IN (
    SELECT id FROM symbols
    WHERE asset_code IN ('BONK', 'FLOKI', 'TRUMP', 'LDO', 'WIF', 'PUMP', 'VIRTUAL')
);

DELETE FROM ingestion_jobs
WHERE symbol_id IN (
    SELECT id FROM symbols
    WHERE asset_code IN ('BONK', 'FLOKI', 'TRUMP', 'LDO', 'WIF', 'PUMP', 'VIRTUAL')
);

DELETE FROM symbols
WHERE asset_code IN ('BONK', 'FLOKI', 'TRUMP', 'LDO', 'WIF', 'PUMP', 'VIRTUAL');

UPDATE symbols
SET is_active = FALSE,
    updated_at = NOW()
WHERE asset_code = 'HYPE';
