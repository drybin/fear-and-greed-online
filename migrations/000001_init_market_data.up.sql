CREATE TABLE IF NOT EXISTS symbols (
    id BIGSERIAL PRIMARY KEY,
    asset_code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    market_cap_rank INTEGER NOT NULL,
    quote_asset TEXT NOT NULL DEFAULT 'USDT',
    binance_symbol TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS timeframes (
    id BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    duration_sec INTEGER NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS candles (
    id BIGSERIAL PRIMARY KEY,
    symbol_id BIGINT NOT NULL REFERENCES symbols(id),
    timeframe_id BIGINT NOT NULL REFERENCES timeframes(id),
    open_time TIMESTAMPTZ NOT NULL,
    close_time TIMESTAMPTZ NOT NULL,
    open NUMERIC(20,8) NOT NULL,
    high NUMERIC(20,8) NOT NULL,
    low NUMERIC(20,8) NOT NULL,
    close NUMERIC(20,8) NOT NULL,
    volume NUMERIC(30,10) NOT NULL,
    quote_volume NUMERIC(30,10) NOT NULL,
    trades BIGINT NOT NULL,
    taker_buy_base_volume NUMERIC(30,10) NOT NULL,
    taker_buy_quote_volume NUMERIC(30,10) NOT NULL,
    is_closed BOOLEAN NOT NULL DEFAULT TRUE,
    source TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT candles_symbol_timeframe_open_time_key UNIQUE(symbol_id, timeframe_id, open_time)
);

CREATE INDEX IF NOT EXISTS candles_symbol_timeframe_open_time_idx
    ON candles(symbol_id, timeframe_id, open_time DESC);

CREATE TABLE IF NOT EXISTS ingestion_jobs (
    id BIGSERIAL PRIMARY KEY,
    symbol_id BIGINT NOT NULL REFERENCES symbols(id),
    timeframe_id BIGINT NOT NULL REFERENCES timeframes(id),
    status TEXT NOT NULL,
    requested_from TIMESTAMPTZ NOT NULL,
    requested_to TIMESTAMPTZ NOT NULL,
    loaded_from TIMESTAMPTZ NULL,
    loaded_to TIMESTAMPTZ NULL,
    candles_loaded INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS ingestion_jobs_symbol_timeframe_started_at_idx
    ON ingestion_jobs(symbol_id, timeframe_id, started_at DESC);
