-- Remove requested symbols and dependent rows so FK constraints allow DELETE.
WITH requested AS (
    SELECT id
    FROM symbols
    WHERE asset_code IN ('1INCH', 'ANKR', 'APT', 'ARB', 'DASH', 'EGLD', 'STRK')
),
requested_signals AS (
    SELECT id
    FROM signals
    WHERE symbol_id IN (SELECT id FROM requested)
)
UPDATE signal_notifications
SET signal_id = NULL
WHERE signal_id IN (SELECT id FROM requested_signals);

DELETE FROM trades
WHERE symbol_id IN (
    SELECT id FROM symbols
    WHERE asset_code IN ('1INCH', 'ANKR', 'APT', 'ARB', 'DASH', 'EGLD', 'STRK')
);

DELETE FROM signals
WHERE symbol_id IN (
    SELECT id FROM symbols
    WHERE asset_code IN ('1INCH', 'ANKR', 'APT', 'ARB', 'DASH', 'EGLD', 'STRK')
);

DELETE FROM strategy_runs
WHERE symbol_id IN (
    SELECT id FROM symbols
    WHERE asset_code IN ('1INCH', 'ANKR', 'APT', 'ARB', 'DASH', 'EGLD', 'STRK')
);

DELETE FROM candles
WHERE symbol_id IN (
    SELECT id FROM symbols
    WHERE asset_code IN ('1INCH', 'ANKR', 'APT', 'ARB', 'DASH', 'EGLD', 'STRK')
);

DELETE FROM ingestion_jobs
WHERE symbol_id IN (
    SELECT id FROM symbols
    WHERE asset_code IN ('1INCH', 'ANKR', 'APT', 'ARB', 'DASH', 'EGLD', 'STRK')
);

DELETE FROM symbols
WHERE asset_code IN ('1INCH', 'ANKR', 'APT', 'ARB', 'DASH', 'EGLD', 'STRK');
