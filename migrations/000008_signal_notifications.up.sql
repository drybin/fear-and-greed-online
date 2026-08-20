CREATE TABLE IF NOT EXISTS signal_notifications (
    id BIGSERIAL PRIMARY KEY,
    channel TEXT NOT NULL,
    signal_dedupe_key TEXT NOT NULL,
    signal_id BIGINT NULL REFERENCES signals(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS signal_notifications_channel_dedupe_key_idx
    ON signal_notifications(channel, signal_dedupe_key);
