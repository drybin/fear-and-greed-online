package postgres

import (
	"context"
	"database/sql"
	"fmt"
)

type SignalNotificationRepository struct {
	db *sql.DB
}

func NewSignalNotificationRepository(db *sql.DB) *SignalNotificationRepository {
	return &SignalNotificationRepository{db: db}
}

func (r *SignalNotificationRepository) TryClaim(ctx context.Context, channel, dedupeKey string, signalID *int64) (bool, error) {
	var claimID int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO signal_notifications (channel, signal_dedupe_key, signal_id, created_at, sent_at)
		VALUES ($1, $2, $3, NOW(), NULL)
		ON CONFLICT (channel, signal_dedupe_key) DO NOTHING
		RETURNING id
	`, channel, dedupeKey, signalID).Scan(&claimID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("claim signal notification: %w", err)
	}
	return claimID > 0, nil
}

func (r *SignalNotificationRepository) MarkSent(ctx context.Context, channel, dedupeKey string) error {
	if _, err := r.db.ExecContext(ctx, `
		UPDATE signal_notifications
		SET sent_at = NOW()
		WHERE channel = $1 AND signal_dedupe_key = $2
	`, channel, dedupeKey); err != nil {
		return fmt.Errorf("mark signal notification sent: %w", err)
	}
	return nil
}

func (r *SignalNotificationRepository) ReleaseClaim(ctx context.Context, channel, dedupeKey string) error {
	if _, err := r.db.ExecContext(ctx, `
		DELETE FROM signal_notifications
		WHERE channel = $1 AND signal_dedupe_key = $2 AND sent_at IS NULL
	`, channel, dedupeKey); err != nil {
		return fmt.Errorf("release signal notification claim: %w", err)
	}
	return nil
}
