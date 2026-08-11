package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type StrategyRunRepository struct {
	db *sql.DB
}

func NewStrategyRunRepository(db *sql.DB) *StrategyRunRepository {
	return &StrategyRunRepository{db: db}
}

func (r *StrategyRunRepository) Start(ctx context.Context, strategyID, symbolID, timeframeID int64, params map[string]any, inputFrom, inputTo *time.Time, candlesCount int) (int64, error) {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return 0, fmt.Errorf("marshal run params: %w", err)
	}

	var id int64
	err = r.db.QueryRowContext(ctx, `
		INSERT INTO strategy_runs (
			strategy_id, symbol_id, timeframe_id, status, started_at, params, input_from, input_to, candles_count, created_at
		) VALUES ($1, $2, $3, 'running', NOW(), $4, $5, $6, $7, NOW())
		RETURNING id
	`, strategyID, symbolID, timeframeID, paramsJSON, inputFrom, inputTo, candlesCount).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("start strategy run: %w", err)
	}

	return id, nil
}

func (r *StrategyRunRepository) Complete(ctx context.Context, runID int64, signalsCount, tradesCount int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE strategy_runs
		SET status = 'success',
			signals_count = $2,
			trades_count = $3,
			finished_at = NOW()
		WHERE id = $1
	`, runID, signalsCount, tradesCount)
	if err != nil {
		return fmt.Errorf("complete strategy run: %w", err)
	}
	return nil
}

func (r *StrategyRunRepository) Fail(ctx context.Context, runID int64, message string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE strategy_runs
		SET status = 'failed',
			error_message = $2,
			finished_at = NOW()
		WHERE id = $1
	`, runID, message)
	if err != nil {
		return fmt.Errorf("fail strategy run: %w", err)
	}
	return nil
}

func (r *StrategyRunRepository) ListRecent(ctx context.Context, strategyID, symbolID, timeframeID int64, limit int) ([]StrategyRunRecord, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, strategy_id, symbol_id, timeframe_id, status, started_at, finished_at, params, input_from, input_to, candles_count, signals_count, trades_count, error_message
		FROM strategy_runs
		WHERE strategy_id = $1 AND symbol_id = $2 AND timeframe_id = $3
		ORDER BY started_at DESC
		LIMIT $4
	`, strategyID, symbolID, timeframeID, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent strategy runs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []StrategyRunRecord
	for rows.Next() {
		var item StrategyRunRecord
		var paramsJSON []byte
		if err := rows.Scan(
			&item.ID, &item.StrategyID, &item.SymbolID, &item.TimeframeID, &item.Status, &item.StartedAt, &item.FinishedAt,
			&paramsJSON, &item.InputFrom, &item.InputTo, &item.CandlesCount, &item.SignalsCount, &item.TradesCount, &item.ErrorMessage,
		); err != nil {
			return nil, fmt.Errorf("scan strategy run record: %w", err)
		}
		params, err := decodeJSONMap(paramsJSON)
		if err != nil {
			return nil, fmt.Errorf("decode strategy run params: %w", err)
		}
		item.Params = params
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *StrategyRunRepository) LatestSuccessful(ctx context.Context, strategyID, symbolID, timeframeID int64) (*StrategyRunRecord, error) {
	var item StrategyRunRecord
	var paramsJSON []byte
	err := r.db.QueryRowContext(ctx, `
		SELECT id, strategy_id, symbol_id, timeframe_id, status, started_at, finished_at, params, input_from, input_to, candles_count, signals_count, trades_count, error_message
		FROM strategy_runs
		WHERE strategy_id = $1 AND symbol_id = $2 AND timeframe_id = $3 AND status = 'success' AND finished_at IS NOT NULL
		ORDER BY finished_at DESC
		LIMIT 1
	`, strategyID, symbolID, timeframeID).Scan(
		&item.ID, &item.StrategyID, &item.SymbolID, &item.TimeframeID, &item.Status, &item.StartedAt, &item.FinishedAt,
		&paramsJSON, &item.InputFrom, &item.InputTo, &item.CandlesCount, &item.SignalsCount, &item.TradesCount, &item.ErrorMessage,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("latest successful strategy run: %w", err)
	}

	params, err := decodeJSONMap(paramsJSON)
	if err != nil {
		return nil, fmt.Errorf("decode strategy run params: %w", err)
	}
	item.Params = params

	return &item, nil
}
