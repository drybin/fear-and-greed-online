package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	domain "github.com/drybin/fear-and-greed-online/internal/domain/strategy"
)

type TradeRepository struct {
	db *sql.DB
}

func NewTradeRepository(db *sql.DB) *TradeRepository {
	return &TradeRepository{db: db}
}

func (r *TradeRepository) UpsertMany(ctx context.Context, runID, strategyID, symbolID, timeframeID int64, trades []domain.Trade) error {
	if len(trades) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin trade upsert: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO trades (
			strategy_run_id, strategy_id, symbol_id, timeframe_id, dedupe_key,
			entry_time, exit_time, side, entry_price, exit_price, pnl_abs, pnl_pct, status, meta, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW()
		)
		ON CONFLICT (dedupe_key)
		DO UPDATE SET
			strategy_run_id = EXCLUDED.strategy_run_id,
			exit_time = EXCLUDED.exit_time,
			exit_price = EXCLUDED.exit_price,
			pnl_abs = EXCLUDED.pnl_abs,
			pnl_pct = EXCLUDED.pnl_pct,
			status = EXCLUDED.status,
			meta = EXCLUDED.meta,
			updated_at = NOW()
	`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare trade upsert: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, trade := range trades {
		metaJSON, err := json.Marshal(trade.Meta)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("marshal trade meta: %w", err)
		}
		if _, err := stmt.ExecContext(
			ctx,
			runID, strategyID, symbolID, timeframeID, trade.DedupeKey,
			trade.EntryTime, trade.ExitTime, trade.Side, trade.EntryPrice, trade.ExitPrice, trade.PnLAbs, trade.PnLPct, trade.Status, metaJSON,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("upsert trade %s: %w", trade.DedupeKey, err)
		}
	}

	return tx.Commit()
}

func (r *TradeRepository) ReplaceRange(ctx context.Context, runID, strategyID, symbolID, timeframeID int64, from, to time.Time, trades []domain.Trade) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin trade replace: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM trades
		WHERE strategy_id = $1 AND symbol_id = $2 AND timeframe_id = $3
			AND entry_time >= $4 AND entry_time <= $5
	`, strategyID, symbolID, timeframeID, from, to); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("delete trades in range: %w", err)
	}

	if len(trades) == 0 {
		return tx.Commit()
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO trades (
			strategy_run_id, strategy_id, symbol_id, timeframe_id, dedupe_key,
			entry_time, exit_time, side, entry_price, exit_price, pnl_abs, pnl_pct, status, meta, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW()
		)
	`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare trade replace insert: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, trade := range trades {
		metaJSON, err := json.Marshal(trade.Meta)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("marshal trade meta: %w", err)
		}
		if _, err := stmt.ExecContext(
			ctx,
			runID, strategyID, symbolID, timeframeID, trade.DedupeKey,
			trade.EntryTime, trade.ExitTime, trade.Side, trade.EntryPrice, trade.ExitPrice, trade.PnLAbs, trade.PnLPct, trade.Status, metaJSON,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert trade %s: %w", trade.DedupeKey, err)
		}
	}

	return tx.Commit()
}

func (r *TradeRepository) ListRange(ctx context.Context, strategyID, symbolID, timeframeID int64, from, to time.Time) ([]TradeRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, strategy_id, symbol_id, timeframe_id, dedupe_key, entry_time, exit_time, side, entry_price, exit_price, pnl_abs, pnl_pct, status, meta
		FROM trades
		WHERE strategy_id = $1 AND symbol_id = $2 AND timeframe_id = $3
			AND entry_time >= $4 AND entry_time <= $5
		ORDER BY entry_time ASC
	`, strategyID, symbolID, timeframeID, from, to)
	if err != nil {
		return nil, fmt.Errorf("list trades by range: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []TradeRecord
	for rows.Next() {
		var item TradeRecord
		var metaJSON []byte
		if err := rows.Scan(
			&item.ID, &item.StrategyID, &item.SymbolID, &item.TimeframeID, &item.DedupeKey,
			&item.EntryTime, &item.ExitTime, &item.Side, &item.EntryPrice, &item.ExitPrice, &item.PnLAbs, &item.PnLPct, &item.Status, &metaJSON,
		); err != nil {
			return nil, fmt.Errorf("scan trade record: %w", err)
		}
		meta, err := decodeJSONMap(metaJSON)
		if err != nil {
			return nil, fmt.Errorf("decode trade meta: %w", err)
		}
		item.Meta = meta
		items = append(items, item)
	}

	return items, rows.Err()
}
