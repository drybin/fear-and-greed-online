CREATE INDEX IF NOT EXISTS signals_strategy_symbol_timeframe_signal_time_idx
    ON signals(strategy_id, symbol_id, timeframe_id, signal_time);

CREATE INDEX IF NOT EXISTS trades_strategy_symbol_timeframe_entry_time_idx
    ON trades(strategy_id, symbol_id, timeframe_id, entry_time);

CREATE INDEX IF NOT EXISTS ingestion_jobs_success_finished_at_idx
    ON ingestion_jobs(symbol_id, timeframe_id, finished_at DESC)
    WHERE status = 'success' AND finished_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS strategy_runs_success_finished_at_idx
    ON strategy_runs(strategy_id, symbol_id, timeframe_id, finished_at DESC)
    WHERE status = 'success' AND finished_at IS NOT NULL;
