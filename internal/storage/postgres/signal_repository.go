package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	domain "github.com/drybin/fear-and-greed-online/internal/domain/strategy"
)

type SignalRepository struct {
	db *sql.DB
}

func NewSignalRepository(db *sql.DB) *SignalRepository {
	return &SignalRepository{db: db}
}

func (r *SignalRepository) UpsertMany(ctx context.Context, runID, strategyID, symbolID, timeframeID int64, signals []domain.Signal) error {
	if len(signals) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin signal upsert: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO signals (
			strategy_run_id, strategy_id, symbol_id, timeframe_id, dedupe_key,
			signal_time, signal_type, side, price, confidence, status, title, details, meta, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW()
		)
		ON CONFLICT (dedupe_key)
		DO UPDATE SET
			strategy_run_id = EXCLUDED.strategy_run_id,
			price = EXCLUDED.price,
			confidence = EXCLUDED.confidence,
			status = EXCLUDED.status,
			title = EXCLUDED.title,
			details = EXCLUDED.details,
			meta = EXCLUDED.meta,
			updated_at = NOW()
	`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare signal upsert: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, signal := range signals {
		metaJSON, err := json.Marshal(signal.Meta)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("marshal signal meta: %w", err)
		}
		if _, err := stmt.ExecContext(
			ctx,
			runID, strategyID, symbolID, timeframeID, signal.DedupeKey,
			signal.Time, signal.Type, signal.Side, signal.Price, signal.Confidence, signal.Status, signal.Title, signal.Details, metaJSON,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("upsert signal %s: %w", signal.DedupeKey, err)
		}
	}

	return tx.Commit()
}

func (r *SignalRepository) ReplaceRange(ctx context.Context, runID, strategyID, symbolID, timeframeID int64, from, to time.Time, signals []domain.Signal) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin signal replace: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM signals
		WHERE strategy_id = $1 AND symbol_id = $2 AND timeframe_id = $3
			AND signal_time >= $4 AND signal_time <= $5
	`, strategyID, symbolID, timeframeID, from, to); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("delete signals in range: %w", err)
	}

	if len(signals) == 0 {
		return tx.Commit()
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO signals (
			strategy_run_id, strategy_id, symbol_id, timeframe_id, dedupe_key,
			signal_time, signal_type, side, price, confidence, status, title, details, meta, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW()
		)
	`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare signal replace insert: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, signal := range signals {
		metaJSON, err := json.Marshal(signal.Meta)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("marshal signal meta: %w", err)
		}
		if _, err := stmt.ExecContext(
			ctx,
			runID, strategyID, symbolID, timeframeID, signal.DedupeKey,
			signal.Time, signal.Type, signal.Side, signal.Price, signal.Confidence, signal.Status, signal.Title, signal.Details, metaJSON,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert signal %s: %w", signal.DedupeKey, err)
		}
	}

	return tx.Commit()
}

func (r *SignalRepository) ListRange(ctx context.Context, strategyID, symbolID, timeframeID int64, from, to time.Time) ([]SignalRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, strategy_id, symbol_id, timeframe_id, dedupe_key, signal_time, signal_type, side, price, confidence, status, title, details, meta
		FROM signals
		WHERE strategy_id = $1 AND symbol_id = $2 AND timeframe_id = $3
			AND signal_time >= $4 AND signal_time <= $5
		ORDER BY signal_time ASC
	`, strategyID, symbolID, timeframeID, from, to)
	if err != nil {
		return nil, fmt.Errorf("list signals by range: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []SignalRecord
	for rows.Next() {
		var item SignalRecord
		var metaJSON []byte
		if err := rows.Scan(
			&item.ID, &item.StrategyID, &item.SymbolID, &item.TimeframeID, &item.DedupeKey,
			&item.SignalTime, &item.SignalType, &item.Side, &item.Price, &item.Confidence, &item.Status, &item.Title, &item.Details, &metaJSON,
		); err != nil {
			return nil, fmt.Errorf("scan signal record: %w", err)
		}
		meta, err := decodeJSONMap(metaJSON)
		if err != nil {
			return nil, fmt.Errorf("decode signal meta: %w", err)
		}
		item.Meta = meta
		items = append(items, item)
	}

	return items, rows.Err()
}
