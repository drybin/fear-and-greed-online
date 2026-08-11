CREATE TABLE IF NOT EXISTS strategies (
    id BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL,
    version TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT 'signal',
    default_params JSONB NOT NULL DEFAULT '{}'::jsonb,
    supported_timeframes JSONB NOT NULL DEFAULT '[]'::jsonb,
    required_history_bars INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS strategy_runs (
    id BIGSERIAL PRIMARY KEY,
    strategy_id BIGINT NOT NULL REFERENCES strategies(id),
    symbol_id BIGINT NOT NULL REFERENCES symbols(id),
    timeframe_id BIGINT NOT NULL REFERENCES timeframes(id),
    status TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NULL,
    params JSONB NOT NULL DEFAULT '{}'::jsonb,
    input_from TIMESTAMPTZ NULL,
    input_to TIMESTAMPTZ NULL,
    candles_count INTEGER NOT NULL DEFAULT 0,
    signals_count INTEGER NOT NULL DEFAULT 0,
    trades_count INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS strategy_runs_strategy_symbol_timeframe_started_at_idx
    ON strategy_runs(strategy_id, symbol_id, timeframe_id, started_at DESC);

CREATE TABLE IF NOT EXISTS signals (
    id BIGSERIAL PRIMARY KEY,
    strategy_run_id BIGINT NOT NULL REFERENCES strategy_runs(id),
    strategy_id BIGINT NOT NULL REFERENCES strategies(id),
    symbol_id BIGINT NOT NULL REFERENCES symbols(id),
    timeframe_id BIGINT NOT NULL REFERENCES timeframes(id),
    dedupe_key TEXT NOT NULL UNIQUE,
    signal_time TIMESTAMPTZ NOT NULL,
    signal_type TEXT NOT NULL,
    side TEXT NOT NULL,
    price NUMERIC(20,8) NOT NULL,
    confidence NUMERIC(5,4) NULL,
    status TEXT NOT NULL DEFAULT 'new',
    title TEXT NOT NULL DEFAULT '',
    details TEXT NOT NULL DEFAULT '',
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS signals_symbol_timeframe_signal_time_idx
    ON signals(symbol_id, timeframe_id, signal_time DESC);

CREATE TABLE IF NOT EXISTS trades (
    id BIGSERIAL PRIMARY KEY,
    strategy_run_id BIGINT NOT NULL REFERENCES strategy_runs(id),
    strategy_id BIGINT NOT NULL REFERENCES strategies(id),
    symbol_id BIGINT NOT NULL REFERENCES symbols(id),
    timeframe_id BIGINT NOT NULL REFERENCES timeframes(id),
    dedupe_key TEXT NOT NULL UNIQUE,
    entry_signal_id BIGINT NULL REFERENCES signals(id),
    exit_signal_id BIGINT NULL REFERENCES signals(id),
    entry_time TIMESTAMPTZ NOT NULL,
    exit_time TIMESTAMPTZ NULL,
    side TEXT NOT NULL,
    entry_price NUMERIC(20,8) NOT NULL,
    exit_price NUMERIC(20,8) NULL,
    pnl_abs NUMERIC(20,8) NULL,
    pnl_pct NUMERIC(10,4) NULL,
    status TEXT NOT NULL,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS trades_symbol_timeframe_entry_time_idx
    ON trades(symbol_id, timeframe_id, entry_time DESC);
