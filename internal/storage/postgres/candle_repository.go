package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/drybin/fear-and-greed-online/internal/domain/marketdata"
)

type CandleRepository struct {
	db *sql.DB
}

func NewCandleRepository(db *sql.DB) *CandleRepository {
	return &CandleRepository{db: db}
}

func (r *CandleRepository) LatestOpenTime(ctx context.Context, symbolID, timeframeID int64) (*time.Time, error) {
	var latest time.Time
	err := r.db.QueryRowContext(ctx, `
		SELECT open_time
		FROM candles
		WHERE symbol_id = $1 AND timeframe_id = $2
		ORDER BY open_time DESC
		LIMIT 1
	`, symbolID, timeframeID).Scan(&latest)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("latest candle open time: %w", err)
	}

	return &latest, nil
}

func (r *CandleRepository) EarliestOpenTime(ctx context.Context, symbolID, timeframeID int64) (*time.Time, error) {
	var earliest time.Time
	err := r.db.QueryRowContext(ctx, `
		SELECT open_time
		FROM candles
		WHERE symbol_id = $1 AND timeframe_id = $2
		ORDER BY open_time ASC
		LIMIT 1
	`, symbolID, timeframeID).Scan(&earliest)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("earliest candle open time: %w", err)
	}

	return &earliest, nil
}

func (r *CandleRepository) UpsertMany(ctx context.Context, candles []marketdata.Candle) error {
	if len(candles) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin candle upsert: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO candles (
			symbol_id, timeframe_id, open_time, close_time, open, high, low, close,
			volume, quote_volume, trades, taker_buy_base_volume, taker_buy_quote_volume,
			is_closed, source, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13,
			$14, $15, NOW(), NOW()
		)
		ON CONFLICT (symbol_id, timeframe_id, open_time)
		DO UPDATE SET
			close_time = EXCLUDED.close_time,
			open = EXCLUDED.open,
			high = EXCLUDED.high,
			low = EXCLUDED.low,
			close = EXCLUDED.close,
			volume = EXCLUDED.volume,
			quote_volume = EXCLUDED.quote_volume,
			trades = EXCLUDED.trades,
			taker_buy_base_volume = EXCLUDED.taker_buy_base_volume,
			taker_buy_quote_volume = EXCLUDED.taker_buy_quote_volume,
			is_closed = EXCLUDED.is_closed,
			source = EXCLUDED.source,
			updated_at = NOW()
	`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare candle upsert: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, candle := range candles {
		if _, err := stmt.ExecContext(
			ctx,
			candle.SymbolID,
			candle.TimeframeID,
			candle.OpenTime,
			candle.CloseTime,
			candle.Open,
			candle.High,
			candle.Low,
			candle.Close,
			candle.Volume,
			candle.QuoteVolume,
			candle.Trades,
			candle.TakerBuyBaseVolume,
			candle.TakerBuyQuoteVolume,
			candle.IsClosed,
			candle.Source,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("upsert candle %s: %w", candle.OpenTime.Format(time.RFC3339), err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit candle upsert: %w", err)
	}

	return nil
}

func (r *CandleRepository) ListRange(ctx context.Context, symbolID, timeframeID int64, from, to time.Time) ([]marketdata.Candle, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT symbol_id, timeframe_id, open_time, close_time, open, high, low, close,
			volume, quote_volume, trades, taker_buy_base_volume, taker_buy_quote_volume, is_closed, source
		FROM candles
		WHERE symbol_id = $1 AND timeframe_id = $2 AND open_time >= $3 AND open_time <= $4
		ORDER BY open_time ASC
	`, symbolID, timeframeID, from, to)
	if err != nil {
		return nil, fmt.Errorf("list candles by range: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var candles []marketdata.Candle
	for rows.Next() {
		var candle marketdata.Candle
		if err := rows.Scan(
			&candle.SymbolID,
			&candle.TimeframeID,
			&candle.OpenTime,
			&candle.CloseTime,
			&candle.Open,
			&candle.High,
			&candle.Low,
			&candle.Close,
			&candle.Volume,
			&candle.QuoteVolume,
			&candle.Trades,
			&candle.TakerBuyBaseVolume,
			&candle.TakerBuyQuoteVolume,
			&candle.IsClosed,
			&candle.Source,
		); err != nil {
			return nil, fmt.Errorf("scan candle: %w", err)
		}
		candles = append(candles, candle)
	}

	return candles, rows.Err()
}
