INSERT INTO strategies (
    code, version, slug, name, description, category, default_params, supported_timeframes, required_history_bars, is_active
)
VALUES
    (
        'trend-long',
        'v1',
        'trend-long-v1',
        'Trend Long v1',
        'Long-only SMA trend strategy with entry on close crossing above SMA and exit on close below SMA.',
        'trend',
        '{"sma_period": 50}'::jsonb,
        '["15m", "1h", "4h"]'::jsonb,
        60,
        TRUE
    ),
    (
        'breakout-retest',
        'v1',
        'breakout-retest-v1',
        'Breakout Retest v1',
        'Long-only breakout and retest strategy using a recent swing-high breakout with retest confirmation.',
        'breakout',
        '{"lookback_bars": 20, "retest_bars": 6, "risk_reward": 2.0}'::jsonb,
        '["15m"]'::jsonb,
        40,
        TRUE
    )
ON CONFLICT (slug) DO UPDATE
SET code = EXCLUDED.code,
    version = EXCLUDED.version,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    default_params = EXCLUDED.default_params,
    supported_timeframes = EXCLUDED.supported_timeframes,
    required_history_bars = EXCLUDED.required_history_bars,
    is_active = EXCLUDED.is_active,
    updated_at = NOW();
