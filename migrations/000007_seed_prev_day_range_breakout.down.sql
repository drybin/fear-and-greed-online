DELETE FROM trades
WHERE strategy_id IN (
    SELECT id FROM strategies WHERE slug = 'prev-day-range-breakout-v1'
);

DELETE FROM signals
WHERE strategy_id IN (
    SELECT id FROM strategies WHERE slug = 'prev-day-range-breakout-v1'
);

DELETE FROM strategy_runs
WHERE strategy_id IN (
    SELECT id FROM strategies WHERE slug = 'prev-day-range-breakout-v1'
);

DELETE FROM strategies
WHERE slug = 'prev-day-range-breakout-v1';
