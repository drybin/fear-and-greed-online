INSERT INTO strategies (
    code, version, slug, name, description, category, default_params, supported_timeframes, required_history_bars, is_active
)
VALUES
    (
        'prev-day-range-breakout',
        'v1',
        'prev-day-range-breakout-v1',
        'Prev-day Range Breakout v1',
        'Alert when candle close breaks above previous Moscow-day high or below previous Moscow-day low.',
        'breakout',
        '{}'::jsonb,
        '["15m", "1h", "4h"]'::jsonb,
        48,
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
